package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	defaultApprovalExpiry            = 10 * time.Minute
	approvalCreateRateLimitWindow    = 60 * time.Second
	approvalCreateRateLimitCount     = 5
	approvalCreateRateLimitHashCount = 3
	executingStaleTimeout            = 5 * time.Minute
	maxExecutionResultBytes          = 64 * 1024
	maxExecutionResultPreviewChars   = 280
)

var executionResultSensitiveFieldPattern = regexp.MustCompile(`(?i)(password|secret|token|apikey|api_key|authorization|credential)`)

// ApprovalRateLimitError is returned when approval creation exceeds the rate limit.
type ApprovalRateLimitError struct {
	Message string
}

func (e *ApprovalRateLimitError) Error() string { return e.Message }

type CreateApprovalInput struct {
	UserID           string
	ServerID         *string
	ToolName         string
	Args             map[string]any
	PolicyVersion    int
	UniformRequestID *string
	ExpiresIn        time.Duration
}

type ApprovalCheckResult struct {
	NeedsApproval bool
	Created       bool
	Request       *database.ApprovalRequest
	RequestHash   string
}

type ApprovalClaimResult struct {
	Claimed bool
	Request *database.ApprovalRequest
}

// ApprovalDecisionActor captures who made an approval decision and through which channel.
type ApprovalDecisionActor struct {
	ActorUserID *string
	ActorRole   *int
	Channel     string // "admin_api" or "socket"
}

type ApprovalService struct {
	db          *gorm.DB
	hasher      *ApprovalRequestHasher
	sweeperMu   sync.Mutex
	sweeperDone chan struct{}
}

var (
	approvalServiceOnce sync.Once
	approvalServiceInst *ApprovalService
)

func ApprovalServiceInstance() *ApprovalService {
	approvalServiceOnce.Do(func() {
		approvalServiceInst = &ApprovalService{db: database.DB, hasher: ApprovalRequestHasherInstance()}
	})
	return approvalServiceInst
}

func (s *ApprovalService) CheckOrCreateApproval(input CreateApprovalInput) (ApprovalCheckResult, error) {
	if input.Args == nil {
		input.Args = map[string]any{}
	}
	serverIDForHash := ""
	if input.ServerID != nil {
		serverIDForHash = *input.ServerID
	}
	requestHash := s.hasher.ComputeHash(input.UserID, serverIDForHash, input.ToolName, input.Args, input.PolicyVersion)

	canonicalArgs := datatypes.JSON([]byte(s.hasher.CanonicalizeArgs(input.ToolName, input.Args)))
	redacted, _ := json.Marshal(s.redactArgs(input.Args, 0))
	redactedArgs := datatypes.JSON(redacted)

	// Check for existing active request first (fast path)
	var existingActive database.ApprovalRequest
	existErr := s.getDB().Where(
		"request_hash = ? AND ((status IN (?, ?) AND expires_at > ?) OR status = ?)",
		requestHash,
		types.ApprovalStatusPending,
		types.ApprovalStatusApproved,
		time.Now(),
		types.ApprovalStatusExecuting,
	).Order("created_at DESC").First(&existingActive).Error
	if existErr == nil {
		if existingActive.Status == types.ApprovalStatusApproved {
			return ApprovalCheckResult{NeedsApproval: false, Created: false, Request: &existingActive, RequestHash: requestHash}, nil
		}
		return ApprovalCheckResult{NeedsApproval: true, Created: false, Request: &existingActive, RequestHash: requestHash}, nil
	}
	if existErr != nil && !errors.Is(existErr, gorm.ErrRecordNotFound) {
		return ApprovalCheckResult{}, existErr
	}

	recentByHashCount, byHashErr := s.CountRecentCreationsByRequestHash(input.UserID, requestHash, time.Now().Add(-approvalCreateRateLimitWindow))
	if byHashErr == nil && recentByHashCount >= approvalCreateRateLimitHashCount {
		return ApprovalCheckResult{}, &ApprovalRateLimitError{
			Message: "Too many retries for this exact approval request. Please retry after 60 seconds.",
		}
	}

	// Rate limit check before creating new request
	recentCount, countErr := s.CountRecentCreationsByTool(input.UserID, input.ServerID, input.ToolName, time.Now().Add(-approvalCreateRateLimitWindow))
	if countErr == nil && recentCount >= approvalCreateRateLimitCount {
		return ApprovalCheckResult{}, &ApprovalRateLimitError{
			Message: fmt.Sprintf("Too many approval requests for %s. Please retry after 60 seconds.", input.ToolName),
		}
	}

	expiresIn := input.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = defaultApprovalExpiry
	}
	now := time.Now()
	expiresAt := now.Add(expiresIn)

	created, request, err := s.createOrGetPending(createOrGetPendingParams{
		UserID:           input.UserID,
		ServerID:         input.ServerID,
		ToolName:         input.ToolName,
		CanonicalArgs:    canonicalArgs,
		RedactedArgs:     redactedArgs,
		PolicyVersion:    input.PolicyVersion,
		RequestHash:      requestHash,
		ExpiresAt:        expiresAt,
		UniformRequestID: input.UniformRequestID,
		Now:              now,
	}, 0)
	if err != nil {
		return ApprovalCheckResult{}, err
	}

	if !created && request != nil && request.Status == types.ApprovalStatusApproved {
		return ApprovalCheckResult{NeedsApproval: false, Created: false, Request: request, RequestHash: requestHash}, nil
	}
	return ApprovalCheckResult{NeedsApproval: true, Created: created, Request: request, RequestHash: requestHash}, nil
}

func (s *ApprovalService) ClaimForExecution(requestHash string) (ApprovalClaimResult, error) {
	rows := make([]database.ApprovalRequest, 0)
	err := s.getDB().Raw(`
		UPDATE approval_request
		SET status = ?, updated_at = NOW()
		WHERE request_hash = ?
		  AND status = ?
		  AND expires_at > NOW()
		RETURNING *
	`, types.ApprovalStatusExecuting, requestHash, types.ApprovalStatusApproved).Scan(&rows).Error
	if err != nil {
		return ApprovalClaimResult{}, err
	}
	if len(rows) == 0 {
		return ApprovalClaimResult{Claimed: false}, nil
	}
	return ApprovalClaimResult{Claimed: true, Request: &rows[0]}, nil
}

func (s *ApprovalService) ClaimForExecutionById(approvalRequestID string) (ApprovalClaimResult, error) {
	rows := make([]database.ApprovalRequest, 0)
	err := s.getDB().Raw(`
		UPDATE approval_request
		SET status = ?, updated_at = NOW()
		WHERE id = ?
		  AND status = ?
		  AND expires_at > NOW()
		RETURNING *
	`, types.ApprovalStatusExecuting, approvalRequestID, types.ApprovalStatusApproved).Scan(&rows).Error
	if err != nil {
		return ApprovalClaimResult{}, err
	}
	if len(rows) == 0 {
		return ApprovalClaimResult{Claimed: false}, nil
	}
	return ApprovalClaimResult{Claimed: true, Request: &rows[0]}, nil
}

func (s *ApprovalService) Decide(id string, decision string, actor ApprovalDecisionActor, reason *string) (*database.ApprovalRequest, error) {
	if decision != types.ApprovalStatusApproved && decision != types.ApprovalStatusRejected {
		return nil, fmt.Errorf("invalid decision: %s", decision)
	}
	now := time.Now()
	rows := make([]database.ApprovalRequest, 0)
	err := s.getDB().Raw(`
		UPDATE approval_request
		SET status = ?, decided_at = ?, decision_reason = ?,
		    decided_by_user_id = ?, decided_by_role = ?, decision_channel = ?,
		    updated_at = ?
		WHERE id = ?
		  AND status = ?
		  AND expires_at > ?
		  AND user_id != ?
		RETURNING *
	`, decision, now, reason,
		actor.ActorUserID, actor.ActorRole, actor.Channel,
		now, id, types.ApprovalStatusPending, now, actor.ActorUserID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *ApprovalService) MarkExecuted(id string, executionResult *mcp.CallToolResult) (*database.ApprovalRequest, error) {
	now := time.Now()
	safeExecutionResult := s.sanitizeExecutionResult(executionResult)
	rows := make([]database.ApprovalRequest, 0)
	err := s.getDB().Raw(`
		UPDATE approval_request
		SET status = ?, executed_at = ?, execution_result = ?::jsonb, updated_at = ?
		WHERE id = ?
		  AND status = ?
		RETURNING *
	`, types.ApprovalStatusExecuted, now, string(safeExecutionResult), now, id, types.ApprovalStatusExecuting).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *ApprovalService) MarkFailed(id string, executionError string, executionResult *mcp.CallToolResult) (*database.ApprovalRequest, error) {
	now := time.Now()
	rows := make([]database.ApprovalRequest, 0)
	var err error
	if executionResult != nil {
		safeExecutionResult := s.sanitizeExecutionResult(executionResult)
		err = s.getDB().Raw(`
			UPDATE approval_request
			SET status = ?, execution_error = ?, execution_result = ?::jsonb, updated_at = ?
			WHERE id = ?
			  AND status = ?
			RETURNING *
		`, types.ApprovalStatusFailed, executionError, string(safeExecutionResult), now, id, types.ApprovalStatusExecuting).Scan(&rows).Error
	} else {
		err = s.getDB().Raw(`
			UPDATE approval_request
			SET status = ?, execution_error = ?, updated_at = ?
			WHERE id = ?
			  AND status = ?
			RETURNING *
		`, types.ApprovalStatusFailed, executionError, now, id, types.ApprovalStatusExecuting).Scan(&rows).Error
	}
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *ApprovalService) TouchExecuting(id string) (*database.ApprovalRequest, error) {
	rows := make([]database.ApprovalRequest, 0)
	err := s.getDB().Raw(`
		UPDATE approval_request
		SET updated_at = NOW()
		WHERE id = ?
		  AND status = ?
		RETURNING *
	`, id, types.ApprovalStatusExecuting).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *ApprovalService) RecoverStaleExecuting(staleBefore time.Time) ([]database.ApprovalRequest, error) {
	rows := make([]database.ApprovalRequest, 0)
	err := s.getDB().Raw(`
		UPDATE approval_request
		SET status = ?,
		    execution_error = 'Stale EXECUTING approval recovered by sweeper timeout',
		    updated_at = NOW()
		WHERE status = ?
		  AND updated_at <= ?
		RETURNING *
	`, types.ApprovalStatusFailed, types.ApprovalStatusExecuting, staleBefore).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *ApprovalService) CountRecentCreationsByTool(userID string, serverID *string, toolName string, since time.Time) (int, error) {
	var count int64
	query := s.getDB().Model(&database.ApprovalRequest{}).
		Where("user_id = ? AND tool_name = ? AND created_at >= ?", userID, toolName, since)
	if serverID != nil {
		query = query.Where("server_id = ?", *serverID)
	} else {
		query = query.Where("server_id IS NULL")
	}
	err := query.Count(&count).Error
	return int(count), err
}

func (s *ApprovalService) CountRecentCreationsByRequestHash(userID string, requestHash string, since time.Time) (int, error) {
	var count int64
	err := s.getDB().Model(&database.ApprovalRequest{}).
		Where("user_id = ? AND request_hash = ? AND created_at >= ?", userID, requestHash, since).
		Count(&count).Error
	return int(count), err
}

// ApprovalListParams holds pagination and filter parameters for listing approval requests.
type ApprovalListParams struct {
	UserID   string
	Status   string
	Page     int
	PageSize int
	Filters  map[string]any
}

// ApprovalListResult holds paginated approval results.
type ApprovalListResult struct {
	Requests []database.ApprovalRequest
	Page     int
	PageSize int
	HasMore  bool
}

func (s *ApprovalService) ListApprovals(params ApprovalListParams) (ApprovalListResult, error) {
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := params.Page
	if page < 1 {
		page = 1
	}

	query := s.getDB().Model(&database.ApprovalRequest{})

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Status != "" {
		if params.Status == types.ApprovalStatusPending {
			query = query.Where("status = ? AND expires_at > ?", types.ApprovalStatusPending, time.Now())
		} else {
			query = query.Where("status = ?", params.Status)
		}
	}
	if params.Filters != nil {
		if serverID, ok := params.Filters["serverId"].(string); ok && serverID != "" {
			query = query.Where("server_id = ?", serverID)
		}
		if toolName, ok := params.Filters["toolName"].(string); ok && toolName != "" {
			query = query.Where("tool_name = ?", toolName)
		}
	}

	requests := make([]database.ApprovalRequest, 0)
	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize + 1).
		Find(&requests).Error
	if err != nil {
		return ApprovalListResult{}, err
	}

	hasMore := len(requests) > pageSize
	if hasMore {
		requests = requests[:pageSize]
	}
	return ApprovalListResult{
		Requests: requests,
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
	}, nil
}

func (s *ApprovalService) CountPending(userID string) (int64, error) {
	var count int64
	query := s.getDB().Model(&database.ApprovalRequest{}).
		Where("status = ? AND expires_at > ?", types.ApprovalStatusPending, time.Now())
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	err := query.Count(&count).Error
	return count, err
}

func (s *ApprovalService) GetByID(id string) (*database.ApprovalRequest, error) {
	var request database.ApprovalRequest
	err := s.getDB().Where("id = ?", id).First(&request).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &request, nil
}

func (s *ApprovalService) GetResultByID(id string) (*database.ApprovalRequest, error) {
	request, err := s.GetByID(id)
	if err != nil || request == nil {
		return request, err
	}
	if request.Status != types.ApprovalStatusExecuted && request.Status != types.ApprovalStatusFailed {
		return nil, nil
	}
	return request, nil
}

func (s *ApprovalService) ExpireStale() ([]database.ApprovalRequest, error) {
	rows := make([]database.ApprovalRequest, 0)
	err := s.getDB().Raw(`
		UPDATE approval_request
		SET status = ?, updated_at = NOW()
		WHERE status IN (?, ?)
		  AND expires_at <= NOW()
		RETURNING *
	`, types.ApprovalStatusExpired, types.ApprovalStatusPending, types.ApprovalStatusApproved).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *ApprovalService) StartExpirySweeper(notifier interface {
	NotifyApprovalExpired(userID string, approvalRequestID string, toolName string)
	NotifyApprovalFailed(userID string, approvalRequestID string, toolName string, executionError string, executionResultAvailable bool, executionResultPreview *string)
}) {
	if notifier == nil {
		return
	}

	s.sweeperMu.Lock()
	if s.sweeperDone != nil {
		s.sweeperMu.Unlock()
		return
	}
	done := make(chan struct{})
	s.sweeperDone = done
	s.sweeperMu.Unlock()

	go func(done <-chan struct{}) {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				expired, err := s.ExpireStale()
				if err == nil {
					for _, request := range expired {
						notifier.NotifyApprovalExpired(request.UserID, request.ID, request.ToolName)
					}
				}

				recovered, err := s.RecoverStaleExecuting(time.Now().Add(-executingStaleTimeout))
				if err == nil {
					for _, request := range recovered {
						execErr := request.ExecutionError
						if execErr == nil {
							defaultErr := "Stale EXECUTING approval recovered by sweeper timeout"
							execErr = &defaultErr
						}
						notifier.NotifyApprovalFailed(request.UserID, request.ID, request.ToolName, *execErr, false, nil)
					}
				}
			case <-done:
				return
			}
		}
	}(done)
}

func (s *ApprovalService) StopExpirySweeper() {
	s.sweeperMu.Lock()
	done := s.sweeperDone
	s.sweeperDone = nil
	s.sweeperMu.Unlock()

	if done != nil {
		close(done)
	}
}

type createOrGetPendingParams struct {
	UserID           string
	ServerID         *string
	ToolName         string
	CanonicalArgs    datatypes.JSON
	RedactedArgs     datatypes.JSON
	PolicyVersion    int
	RequestHash      string
	ExpiresAt        time.Time
	UniformRequestID *string
	Now              time.Time
}

func (s *ApprovalService) createOrGetPending(params createOrGetPendingParams, retryCount int) (bool, *database.ApprovalRequest, error) {
	const maxRetries = 3
	inserted := make([]database.ApprovalRequest, 0)
	err := s.getDB().Raw(`
		INSERT INTO approval_request (
			id, user_id, server_id, tool_name,
			canonical_args, redacted_args, policy_version,
			request_hash, status, expires_at,
			uniform_request_id,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?,
			?::jsonb, ?::jsonb, ?,
			?, ?, ?,
			?,
			?, ?
		)
		ON CONFLICT (request_hash)
		WHERE status IN ('PENDING', 'APPROVED', 'EXECUTING')
		DO NOTHING
		RETURNING *
	`, uuid.NewString(), params.UserID, params.ServerID, params.ToolName,
		string(params.CanonicalArgs), string(params.RedactedArgs), params.PolicyVersion,
		params.RequestHash, types.ApprovalStatusPending, params.ExpiresAt,
		params.UniformRequestID,
		params.Now, params.Now).Scan(&inserted).Error
	if err != nil {
		return false, nil, err
	}
	if len(inserted) > 0 {
		return true, &inserted[0], nil
	}

	var existing database.ApprovalRequest
	err = s.getDB().Where(
		"request_hash = ? AND ((status IN (?, ?) AND expires_at > ?) OR status = ?)",
		params.RequestHash,
		types.ApprovalStatusPending,
		types.ApprovalStatusApproved,
		params.Now,
		types.ApprovalStatusExecuting,
	).Order("created_at DESC").First(&existing).Error
	if err == nil {
		return false, &existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, err
	}

	if retryCount >= maxRetries {
		return false, nil, fmt.Errorf("failed to create or find approval request after %d retries: %s", maxRetries, params.RequestHash)
	}

	if err := s.getDB().Exec(`
		UPDATE approval_request
		SET status = ?, updated_at = NOW()
		WHERE request_hash = ?
		  AND status IN (?, ?)
		  AND expires_at <= NOW()
	`, types.ApprovalStatusExpired, params.RequestHash, types.ApprovalStatusPending, types.ApprovalStatusApproved).Error; err != nil {
		return false, nil, err
	}

	return s.createOrGetPending(params, retryCount+1)
}

func (s *ApprovalService) getDB() *gorm.DB {
	if s.db != nil {
		return s.db
	}
	return database.DB
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range []string{"password", "passwd", "secret", "token", "api_key", "apikey", "api-key", "authorization", "credential", "private_key", "privatekey", "private-key", "access_key", "accesskey", "refresh_token"} {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func (s *ApprovalService) redactArgs(args map[string]any, depth int) map[string]any {
	redacted := map[string]any{}
	const maxStringLength = 100
	const maxDepth = 2
	for key, value := range args {
		if isSensitiveKey(key) {
			redacted[key] = "[REDACTED]"
			continue
		}
		switch typed := value.(type) {
		case nil:
			redacted[key] = nil
		case string:
			if len(typed) > maxStringLength {
				redacted[key] = typed[:maxStringLength] + "..."
			} else {
				redacted[key] = typed
			}
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			redacted[key] = typed
		case []any:
			if depth < maxDepth {
				shallow := make([]any, len(typed))
				for i, item := range typed {
					switch iv := item.(type) {
					case nil:
						shallow[i] = nil
					case string:
						if len(iv) > maxStringLength {
							shallow[i] = iv[:maxStringLength] + "..."
						} else {
							shallow[i] = iv
						}
					case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
						shallow[i] = iv
					default:
						if _, isMap := iv.(map[string]any); isMap {
							shallow[i] = "[object]"
						} else {
							shallow[i] = fmt.Sprintf("%v", iv)
						}
					}
				}
				redacted[key] = shallow
			} else {
				redacted[key] = fmt.Sprintf("[Array(%d)]", len(typed))
			}
		case map[string]any:
			if depth < maxDepth {
				redacted[key] = s.redactArgs(typed, depth+1)
			} else {
				redacted[key] = "[object]"
			}
		default:
			redacted[key] = fmt.Sprintf("%v", typed)
		}
	}
	return redacted
}

func (s *ApprovalService) sanitizeExecutionResult(executionResult *mcp.CallToolResult) datatypes.JSON {
	if executionResult == nil {
		return s.fallbackExecutionResultJSON("Execution result is unavailable for replay.", true, map[string]any{"kind": "execution_result_unavailable", "truncated": false})
	}

	raw, err := json.Marshal(executionResult)
	if err != nil {
		return s.fallbackExecutionResultJSON("Execution result could not be serialized safely.", executionResult.IsError, map[string]any{"kind": "execution_result_unserializable", "truncated": false})
	}

	var asAny any
	if err := json.Unmarshal(raw, &asAny); err != nil {
		return s.fallbackExecutionResultJSON("Execution result could not be serialized safely.", executionResult.IsError, map[string]any{"kind": "execution_result_unserializable", "truncated": false})
	}

	redacted := s.redactSensitiveExecutionValue(asAny, "")
	redactedRaw, err := json.Marshal(redacted)
	if err != nil {
		return s.fallbackExecutionResultJSON("Execution result could not be serialized safely.", executionResult.IsError, map[string]any{"kind": "execution_result_unserializable", "truncated": false})
	}

	if len(redactedRaw) <= maxExecutionResultBytes {
		return datatypes.JSON(redactedRaw)
	}

	preview := string(redactedRaw)
	if len(preview) > maxExecutionResultPreviewChars {
		preview = preview[:maxExecutionResultPreviewChars]
	}
	return s.fallbackExecutionResultJSON(preview, executionResult.IsError, map[string]any{
		"kind":              "execution_result_truncated",
		"truncated":         true,
		"originalSizeBytes": len(redactedRaw),
	})
}

func (s *ApprovalService) fallbackExecutionResultJSON(text string, isError bool, extraMeta map[string]any) datatypes.JSON {
	meta := mcp.Meta{}
	for k, v := range extraMeta {
		meta[k] = v
	}
	fallback := &mcp.CallToolResult{
		IsError: isError,
		Meta:    meta,
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
	raw, err := json.Marshal(fallback)
	if err != nil {
		return datatypes.JSON([]byte(`{"content":[{"type":"text","text":"Execution result replay is unavailable."}],"isError":true}`))
	}
	return datatypes.JSON(raw)
}

func (s *ApprovalService) redactSensitiveExecutionValue(value any, keyHint string) any {
	if keyHint != "" && executionResultSensitiveFieldPattern.MatchString(keyHint) {
		return "[redacted]"
	}

	switch typed := value.(type) {
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, s.redactSensitiveExecutionValue(item, ""))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = s.redactSensitiveExecutionValue(v, k)
		}
		return out
	default:
		return value
	}
}

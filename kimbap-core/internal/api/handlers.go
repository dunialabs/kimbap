package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/auth"
	"github.com/dunialabs/kimbap-core/internal/policy"
	runtimepkg "github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/dunialabs/kimbap-core/internal/store"
	"github.com/go-chi/chi/v5"
)

type createTokenRequest struct {
	TenantID   string   `json:"tenant_id"`
	AgentName  string   `json:"agent_name"`
	Scopes     []string `json:"scopes"`
	TTLSeconds int64    `json:"ttl_seconds"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListActions(w http.ResponseWriter, r *http.Request) {
	defs := make([]actions.ActionDefinition, 0)
	if s.runtime != nil && s.runtime.ActionRegistry != nil {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := s.runtime.ActionRegistry.List(r.Context(), runtimepkg.ListOptions{
			Namespace: r.URL.Query().Get("namespace"),
			Resource:  r.URL.Query().Get("resource"),
			Verb:      r.URL.Query().Get("verb"),
			Limit:     limit,
		})
		if err != nil {
			writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return
		}
		defs = items
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": defs})
}

func (s *Server) handleDescribeAction(w http.ResponseWriter, r *http.Request) {
	if s.runtime == nil || s.runtime.ActionRegistry == nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrActionNotFound, "action registry unavailable", http.StatusNotFound, false, nil))
		return
	}
	name := chi.URLParam(r, "service") + "." + chi.URLParam(r, "action")
	def, err := s.runtime.ActionRegistry.Lookup(r.Context(), name)
	if err != nil || def == nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrActionNotFound, "action not found", http.StatusNotFound, false, map[string]any{"action": name}))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": def})
}

func (s *Server) handleExecuteAction(w http.ResponseWriter, r *http.Request) {
	if s.runtime == nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "runtime pipeline unavailable", http.StatusNotImplemented, false, nil))
		return
	}
	var body struct {
		Input map[string]any `json:"input"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	principal := principalFromContext(r.Context())
	requestID := requestIDFromContext(r.Context())
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = requestID
	}
	req := actions.ExecutionRequest{
		RequestID:      requestID,
		IdempotencyKey: idempotencyKey,
		TenantID:       tenantFromContext(r.Context()),
		Principal: actions.Principal{
			ID:        principal.ID,
			TenantID:  principal.TenantID,
			AgentName: principal.AgentName,
			Type:      string(principal.Type),
			Scopes:    append([]string(nil), principal.Scopes...),
		},
		Action: actions.ActionDefinition{Name: chi.URLParam(r, "service") + "." + chi.URLParam(r, "action")},
		Input:  body.Input,
		Mode:   actions.ModeCall,
	}
	result := s.runtime.Execute(r.Context(), req)
	if result.Error != nil {
		writeExecutionError(w, result.Error)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (s *Server) handleValidateAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Schema *actions.Schema `json:"schema"`
		Input  map[string]any  `json:"input"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	if verr := actions.ValidateInput(body.Schema, body.Input); verr != nil {
		writeExecutionError(w, verr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": true})
}

func (s *Server) handleListVaultKeys(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	if s.tokenService == nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "token service unavailable", http.StatusInternalServerError, false, nil))
		return
	}
	var req createTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	if strings.TrimSpace(req.AgentName) == "" {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, "agent_name is required", http.StatusBadRequest, false, nil))
		return
	}
	tenantID := tenantFromContext(r.Context())
	if tenantID == "" {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthorized, "tenant context required", http.StatusForbidden, false, nil))
		return
	}
	principal := principalFromContext(r.Context())
	if len(req.Scopes) > 0 && principal != nil {
		for _, requested := range req.Scopes {
			if !principalHasScope(principal, requested) {
				writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthorized,
					"cannot mint token with scope not held by caller: "+requested,
					http.StatusForbidden, false, nil))
				return
			}
		}
	}
	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	raw, issued, err := s.tokenService.Issue(r.Context(), tenantID, req.AgentName, req.Scopes, ttl)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidTTL) {
			writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
			return
		}
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": raw, "metadata": issued})
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromContext(r.Context())
	items, err := s.store.ListTokens(r.Context(), tenantID)
	if err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) handleInspectToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := tenantFromContext(r.Context())
	tok, err := s.store.GetToken(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeExecutionError(w, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
		return
	}
	if tok.TenantID != tenantID {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthorized, "token not accessible for tenant", http.StatusForbidden, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": tok})
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := tenantFromContext(r.Context())
	tok, err := s.store.GetToken(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeExecutionError(w, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
		return
	}
	if tok.TenantID != tenantID {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthorized, "token not accessible for tenant", http.StatusForbidden, false, nil))
		return
	}
	if err := s.store.RevokeToken(r.Context(), id); err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromContext(r.Context())
	doc, err := s.store.GetPolicy(r.Context(), tenantID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeExecutionError(w, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "document": string(doc)})
}

func (s *Server) handleSetPolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Document string `json:"document"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	tenantID := tenantFromContext(r.Context())
	if err := s.store.SetPolicy(r.Context(), tenantID, []byte(body.Document)); err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (s *Server) handleEvalPolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AgentName string         `json:"agent_name"`
		Service   string         `json:"service"`
		Action    string         `json:"action"`
		Risk      string         `json:"risk"`
		Mutating  bool           `json:"mutating"`
		Args      map[string]any `json:"args"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	tenantID := tenantFromContext(r.Context())
	docBytes, err := s.store.GetPolicy(r.Context(), tenantID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeExecutionError(w, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
		return
	}
	doc, err := policy.ParseDocument(docBytes)
	if err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	evaluator := policy.NewEvaluator(doc)
	result, err := evaluator.Evaluate(r.Context(), policy.EvalRequest{TenantID: tenantID, AgentName: body.AgentName, Service: body.Service, Action: body.Action, Risk: body.Risk, Mutating: body.Mutating, Args: body.Args})
	if err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromContext(r.Context())
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	items, err := s.store.ListApprovals(r.Context(), tenantID, status)
	if err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := tenantFromContext(r.Context())
	principal := principalFromContext(r.Context())

	existing, err := s.store.GetApproval(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeExecutionError(w, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
		return
	}
	if existing.TenantID != tenantID {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthorized, "approval not found", http.StatusNotFound, false, nil))
		return
	}
	if existing.Status != "pending" {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, "approval already resolved", http.StatusConflict, false, nil))
		return
	}
	if time.Now().After(existing.ExpiresAt) {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrApprovalTimeout, "approval expired", http.StatusGone, false, nil))
		return
	}

	if s.approvalManager != nil {
		err = s.approvalManager.Approve(r.Context(), id, principal.ID)
	} else {
		err = s.store.UpdateApprovalStatus(r.Context(), id, "approved", principal.ID, "approved via api")
	}
	if err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approved": true})
}

func (s *Server) handleDeny(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := tenantFromContext(r.Context())
	principal := principalFromContext(r.Context())

	existing, err := s.store.GetApproval(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeExecutionError(w, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
		return
	}
	if existing.TenantID != tenantID {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthorized, "approval not found", http.StatusNotFound, false, nil))
		return
	}
	if existing.Status != "pending" {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrValidationFailed, "approval already resolved", http.StatusConflict, false, nil))
		return
	}
	if time.Now().After(existing.ExpiresAt) {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrApprovalTimeout, "approval expired", http.StatusGone, false, nil))
		return
	}

	if s.approvalManager != nil {
		err = s.approvalManager.Deny(r.Context(), id, principal.ID, "denied via api")
	} else {
		err = s.store.UpdateApprovalStatus(r.Context(), id, "denied", principal.ID, "denied via api")
	}
	if err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"denied": true})
}

func (s *Server) handleQueryAudit(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	from, _ := parseRFC3339Ptr(r.URL.Query().Get("from"))
	to, _ := parseRFC3339Ptr(r.URL.Query().Get("to"))
	items, err := s.store.QueryAuditEvents(r.Context(), store.AuditFilter{
		TenantID:  tenantID,
		AgentName: r.URL.Query().Get("agent_name"),
		Service:   r.URL.Query().Get("service"),
		Action:    r.URL.Query().Get("action"),
		Status:    r.URL.Query().Get("status"),
		From:      from,
		To:        to,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) handleExportAudit(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromContext(r.Context())
	format := strings.TrimSpace(r.URL.Query().Get("format"))
	if strings.EqualFold(format, "csv") {
		w.Header().Set("Content-Type", "text/csv")
	}
	err := s.store.ExportAuditEvents(r.Context(), store.AuditFilter{TenantID: tenantID}, format, w)
	if err != nil {
		writeExecutionError(w, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeExecutionError(w http.ResponseWriter, execErr *actions.ExecutionError) {
	if execErr == nil {
		execErr = actions.NewExecutionError(actions.ErrDownstreamUnavailable, "unknown error", http.StatusInternalServerError, false, nil)
	}
	status := execErr.HTTPStatus
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	msg := execErr.Message
	var details map[string]any
	if status >= 500 {
		msg = "internal server error"
		details = nil
	} else {
		details = execErr.Details
	}
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":      execErr.Code,
			"message":   msg,
			"retryable": execErr.Retryable,
			"details":   details,
		},
	})
}

const maxAPIRequestBodyBytes int64 = 4 << 20

func decodeJSON(r *http.Request, out any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxAPIRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		switch {
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("invalid JSON at offset %d", syntaxErr.Offset)
		case errors.As(err, &typeErr):
			return fmt.Errorf("invalid type for field %q", typeErr.Field)
		case strings.Contains(err.Error(), "unknown field"):
			return err
		default:
			return errors.New("invalid request body")
		}
	}
	return nil
}

func parseRFC3339Ptr(v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &ts, nil
}

type storeTokenAdapter struct {
	st store.Store
}

type storeApprovalAdapter struct {
	st store.Store
}

func (a *storeApprovalAdapter) Create(ctx context.Context, req *approvals.ApprovalRequest) error {
	if req == nil {
		return errors.New("approval request is required")
	}
	inputJSON := "{}"
	if len(req.Input) > 0 {
		b, err := json.Marshal(req.Input)
		if err != nil {
			return err
		}
		inputJSON = string(b)
	}
	return a.st.CreateApproval(ctx, &store.ApprovalRecord{
		ID:         req.ID,
		TenantID:   req.TenantID,
		RequestID:  req.RequestID,
		AgentName:  req.AgentName,
		Service:    req.Service,
		Action:     req.Action,
		Status:     string(req.Status),
		InputJSON:  inputJSON,
		CreatedAt:  req.CreatedAt,
		ExpiresAt:  req.ExpiresAt,
		ResolvedAt: req.ResolvedAt,
		ResolvedBy: req.ResolvedBy,
		Reason:     req.DenyReason,
	})
}

func (a *storeApprovalAdapter) Get(ctx context.Context, id string) (*approvals.ApprovalRequest, error) {
	rec, err := a.st.GetApproval(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &approvals.ApprovalRequest{
		ID:         rec.ID,
		TenantID:   rec.TenantID,
		RequestID:  rec.RequestID,
		AgentName:  rec.AgentName,
		Service:    rec.Service,
		Action:     rec.Action,
		Status:     approvals.ApprovalStatus(rec.Status),
		CreatedAt:  rec.CreatedAt,
		ExpiresAt:  rec.ExpiresAt,
		ResolvedAt: rec.ResolvedAt,
		ResolvedBy: rec.ResolvedBy,
		DenyReason: rec.Reason,
	}
	if strings.TrimSpace(rec.InputJSON) != "" {
		var input map[string]any
		if err := json.Unmarshal([]byte(rec.InputJSON), &input); err == nil {
			out.Input = input
		}
	}
	return out, nil
}

func (a *storeApprovalAdapter) Update(ctx context.Context, req *approvals.ApprovalRequest) error {
	if req == nil {
		return errors.New("approval request is required")
	}
	reason := req.DenyReason
	if req.Status == approvals.StatusApproved && strings.TrimSpace(reason) == "" {
		reason = "approved via api"
	}
	return a.st.UpdateApprovalStatus(ctx, req.ID, string(req.Status), req.ResolvedBy, reason)
}

func (a *storeApprovalAdapter) ListPending(ctx context.Context, tenantID string) ([]approvals.ApprovalRequest, error) {
	items, err := a.st.ListApprovals(ctx, tenantID, string(approvals.StatusPending))
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, 0, len(items))
	for i := range items {
		it := items[i]
		out = append(out, approvals.ApprovalRequest{
			ID:         it.ID,
			TenantID:   it.TenantID,
			RequestID:  it.RequestID,
			AgentName:  it.AgentName,
			Service:    it.Service,
			Action:     it.Action,
			Status:     approvals.ApprovalStatus(it.Status),
			CreatedAt:  it.CreatedAt,
			ExpiresAt:  it.ExpiresAt,
			ResolvedAt: it.ResolvedAt,
			ResolvedBy: it.ResolvedBy,
			DenyReason: it.Reason,
		})
	}
	return out, nil
}

func (a *storeApprovalAdapter) ListAll(ctx context.Context, tenantID string, filter approvals.ApprovalFilter) ([]approvals.ApprovalRequest, error) {
	status := ""
	if filter.Status != nil {
		status = string(*filter.Status)
	}
	items, err := a.st.ListApprovals(ctx, tenantID, status)
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, 0, len(items))
	for i := range items {
		it := items[i]
		out = append(out, approvals.ApprovalRequest{
			ID:         it.ID,
			TenantID:   it.TenantID,
			RequestID:  it.RequestID,
			AgentName:  it.AgentName,
			Service:    it.Service,
			Action:     it.Action,
			Status:     approvals.ApprovalStatus(it.Status),
			CreatedAt:  it.CreatedAt,
			ExpiresAt:  it.ExpiresAt,
			ResolvedAt: it.ResolvedAt,
			ResolvedBy: it.ResolvedBy,
			DenyReason: it.Reason,
		})
	}
	return out, nil
}

func (a *storeApprovalAdapter) ExpireOld(_ context.Context) (int, error) {
	return 0, nil
}

func (a *storeTokenAdapter) Create(ctx context.Context, token *auth.ServiceToken) error {
	if token == nil {
		return errors.New("token is required")
	}
	return a.st.CreateToken(ctx, &store.TokenRecord{
		ID:          token.ID,
		TenantID:    token.TenantID,
		AgentName:   token.AgentName,
		TokenHash:   token.TokenHash,
		DisplayHint: token.DisplayHint,
		Scopes:      marshalScopes(token.Scopes),
		CreatedAt:   token.CreatedAt,
		ExpiresAt:   token.ExpiresAt,
		LastUsedAt:  token.LastUsedAt,
		RevokedAt:   token.RevokedAt,
		CreatedBy:   token.CreatedBy,
	})
}

func (a *storeTokenAdapter) ValidateAndResolve(ctx context.Context, rawToken string) (*auth.Principal, error) {
	hash := sha256.Sum256([]byte(rawToken))
	token, err := a.st.GetTokenByHash(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}
	now := time.Now().UTC()
	if token.RevokedAt != nil {
		return nil, auth.ErrRevokedToken
	}
	if now.After(token.ExpiresAt) {
		return nil, auth.ErrExpiredToken
	}
	return &auth.Principal{
		ID:        token.AgentName,
		Type:      auth.PrincipalTypeService,
		TenantID:  token.TenantID,
		AgentName: token.AgentName,
		Scopes:    parseScopes(token.Scopes),
		TokenID:   token.ID,
		IssuedAt:  token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
	}, nil
}

func (a *storeTokenAdapter) Revoke(ctx context.Context, tokenID string) error {
	if strings.TrimSpace(tokenID) == "" {
		return auth.ErrInvalidToken
	}
	err := a.st.RevokeToken(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
}

func (a *storeTokenAdapter) List(ctx context.Context, tenantID string) ([]auth.ServiceToken, error) {
	items, err := a.st.ListTokens(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]auth.ServiceToken, 0, len(items))
	for i := range items {
		it := items[i]
		out = append(out, auth.ServiceToken{
			ID:          it.ID,
			TenantID:    it.TenantID,
			AgentName:   it.AgentName,
			TokenHash:   it.TokenHash,
			DisplayHint: it.DisplayHint,
			Scopes:      parseScopes(it.Scopes),
			CreatedAt:   it.CreatedAt,
			ExpiresAt:   it.ExpiresAt,
			LastUsedAt:  it.LastUsedAt,
			RevokedAt:   it.RevokedAt,
			CreatedBy:   it.CreatedBy,
		})
	}
	return out, nil
}

func (a *storeTokenAdapter) Inspect(ctx context.Context, tokenID string) (*auth.ServiceToken, error) {
	it, err := a.st.GetToken(ctx, tokenID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}
	return &auth.ServiceToken{
		ID:          it.ID,
		TenantID:    it.TenantID,
		AgentName:   it.AgentName,
		TokenHash:   it.TokenHash,
		DisplayHint: it.DisplayHint,
		Scopes:      parseScopes(it.Scopes),
		CreatedAt:   it.CreatedAt,
		ExpiresAt:   it.ExpiresAt,
		LastUsedAt:  it.LastUsedAt,
		RevokedAt:   it.RevokedAt,
		CreatedBy:   it.CreatedBy,
	}, nil
}

func (a *storeTokenAdapter) MarkUsed(ctx context.Context, tokenID string) error {
	err := a.st.UpdateTokenLastUsed(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
}

func sanitizeErrMsg(err error, status int) string {
	if status == http.StatusInternalServerError {
		return "internal server error"
	}
	return err.Error()
}

func marshalScopes(scopes []string) string {
	if len(scopes) == 0 {
		return "[]"
	}
	b, err := json.Marshal(scopes)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func parseScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var scopes []string
	if err := json.Unmarshal([]byte(raw), &scopes); err != nil {
		return nil
	}
	return scopes
}

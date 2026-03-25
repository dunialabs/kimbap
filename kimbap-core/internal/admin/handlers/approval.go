package handlers

import (
	"encoding/json"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	"github.com/dunialabs/kimbap-core/internal/mcp/services"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ApprovalHandler struct {
	socketNotifier core.SocketNotifier
}

func NewApprovalHandler(socketNotifier core.SocketNotifier) *ApprovalHandler {
	if socketNotifier == nil {
		socketNotifier = core.NewNoopSocketNotifier()
	}
	return &ApprovalHandler{socketNotifier: socketNotifier}
}

func (h *ApprovalHandler) ListApprovalRequests(data map[string]any) (any, error) {
	var userID string
	if raw, ok := data["userId"]; ok && raw != nil {
		if s, ok2 := raw.(string); ok2 {
			userID = strings.TrimSpace(s)
		} else {
			return nil, &types.AdminError{Message: "Invalid field: userId must be a string", Code: types.AdminErrorCodeInvalidRequest}
		}
	}

	var statusFilter string
	if raw, ok := data["status"]; ok && raw != nil {
		if s, ok2 := raw.(string); ok2 {
			statusFilter = strings.TrimSpace(s)
		} else {
			return nil, &types.AdminError{Message: "Invalid field: status must be a string", Code: types.AdminErrorCodeInvalidRequest}
		}
	}

	if statusFilter != "" && !validApprovalStatus(statusFilter) {
		return nil, &types.AdminError{Message: "Invalid status filter: " + statusFilter, Code: types.AdminErrorCodeInvalidRequest}
	}

	filters := map[string]any{}
	if serverID, ok := data["serverId"].(string); ok && serverID != "" {
		filters["serverId"] = serverID
	}
	if toolName, ok := data["toolName"].(string); ok && toolName != "" {
		filters["toolName"] = toolName
	}

	page := toInt(data["page"], 1)
	pageSize := toInt(data["pageSize"], 20)

	result, err := services.ApprovalServiceInstance().ListApprovals(services.ApprovalListParams{
		UserID:   userID,
		Status:   statusFilter,
		Page:     page,
		PageSize: pageSize,
		Filters:  filters,
	})
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(result.Requests))
	for i := range result.Requests {
		items = append(items, approvalRequestResponseMap(&result.Requests[i]))
	}
	return map[string]any{
		"requests": items,
		"page":     result.Page,
		"pageSize": result.PageSize,
		"hasMore":  result.HasMore,
	}, nil
}

func (h *ApprovalHandler) GetApprovalRequest(data map[string]any) (any, error) {
	id := toString(data["id"])
	if id == "" {
		return nil, &types.AdminError{Message: "Missing required field: id", Code: types.AdminErrorCodeInvalidRequest}
	}
	request, err := services.ApprovalServiceInstance().GetByID(id)
	if err != nil {
		return nil, err
	}
	if request == nil {
		return nil, &types.AdminError{Message: "Approval request not found", Code: types.AdminErrorCodeInvalidRequest}
	}
	return approvalRequestResponseMap(request), nil
}

func (h *ApprovalHandler) DecideApprovalRequest(data map[string]any, authCtx *types.AuthContext) (any, error) {
	id := toString(data["id"])
	decision := toString(data["decision"])
	if id == "" {
		return nil, &types.AdminError{Message: "Missing required field: id", Code: types.AdminErrorCodeInvalidRequest}
	}
	if decision != types.ApprovalStatusApproved && decision != types.ApprovalStatusRejected {
		return nil, &types.AdminError{Message: "Invalid decision: must be APPROVED or REJECTED", Code: types.AdminErrorCodeInvalidRequest}
	}

	var reason *string
	if rawReason := toString(data["reason"]); rawReason != "" {
		reason = &rawReason
	}

	actor := services.ApprovalDecisionActor{Channel: "admin_api"}
	if authCtx != nil {
		actor.ActorUserID = &authCtx.UserID
		actor.ActorRole = &authCtx.Role
	}

	result, err := services.ApprovalServiceInstance().Decide(id, decision, actor, reason)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, &types.AdminError{Message: "Decision failed: request not found, not PENDING, or expired", Code: types.AdminErrorCodeInvalidRequest}
	}

	h.socketNotifier.NotifyApprovalDecided(result.UserID, result.ID, result.ToolName, result.Status, result.DecisionReason)
	return approvalRequestResponseMap(result), nil
}

func (h *ApprovalHandler) GetPendingApprovalsCount(data map[string]any) (any, error) {
	var userID string
	if rawUID, ok := data["userId"]; ok && rawUID != nil {
		if s, ok2 := rawUID.(string); ok2 {
			userID = strings.TrimSpace(s)
		} else {
			return nil, &types.AdminError{Message: "Invalid field: userId must be a string", Code: types.AdminErrorCodeInvalidRequest}
		}
	}
	count, err := services.ApprovalServiceInstance().CountPending(userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"count": count}, nil
}

func approvalRequestResponseMap(r *database.ApprovalRequest) map[string]any {
	if r == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":                       r.ID,
		"resumeToken":              r.ID,
		"userId":                   r.UserID,
		"serverId":                 r.ServerID,
		"toolName":                 r.ToolName,
		"canonicalArgs":            r.CanonicalArgs,
		"redactedArgs":             r.RedactedArgs,
		"policyVersion":            r.PolicyVersion,
		"requestHash":              r.RequestHash,
		"status":                   r.Status,
		"expiresAt":                r.ExpiresAt,
		"decidedAt":                r.DecidedAt,
		"decisionReason":           r.DecisionReason,
		"decidedByUserId":          r.DecidedByUserID,
		"decidedByRole":            r.DecidedByRole,
		"decisionChannel":          r.DecisionChannel,
		"executedAt":               r.ExecutedAt,
		"executionError":           r.ExecutionError,
		"executionResultAvailable": isExecutionResultReplayable(r.ExecutionResult),
		"uniformRequestId":         r.UniformRequestID,
		"createdAt":                r.CreatedAt,
		"updatedAt":                r.UpdatedAt,
	}
}

func isExecutionResultReplayable(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return false
	}

	var parsed mcp.CallToolResult
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false
	}
	return len(parsed.Content) > 0
}

var validApprovalStatuses = map[string]bool{
	types.ApprovalStatusPending:   true,
	types.ApprovalStatusApproved:  true,
	types.ApprovalStatusRejected:  true,
	types.ApprovalStatusExpired:   true,
	types.ApprovalStatusExecuting: true,
	types.ApprovalStatusExecuted:  true,
	types.ApprovalStatusFailed:    true,
}

func validApprovalStatus(s string) bool {
	return validApprovalStatuses[s]
}

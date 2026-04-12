package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
)

func (r *Runtime) requestApproval(ctx context.Context, req actions.ExecutionRequest) (*ApprovalResult, *actions.ExecutionError) {
	if r.ApprovalManager == nil {
		return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "approval manager unavailable", 500, false, nil)
	}
	approvalMeta := map[string]any{"mode": req.Mode}
	if approvalRef := strings.TrimSpace(req.Action.Adapter.ApprovalRef); approvalRef != "" {
		approvalMeta["approval_ref"] = approvalRef
	}

	approvalResult, err := r.ApprovalManager.CreateRequest(ctx, ApprovalRequest{
		RequestID: req.RequestID,
		TraceID:   req.TraceID,
		TenantID:  req.TenantID,
		Principal: req.Principal,
		Action:    req.Action,
		Input:     req.Input,
		Meta:      approvalMeta,
	})
	if err != nil {
		return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, err.Error(), 500, false, nil)
	}
	if approvalResult == nil {
		return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "approval response missing", 500, false, nil)
	}
	return approvalResult, nil
}

func (r *Runtime) cancelApprovalOnHoldFailure(ctx context.Context, approvalRequestID string, holdErr error) error {
	if r == nil || r.ApprovalManager == nil || strings.TrimSpace(approvalRequestID) == "" {
		return nil
	}
	reason := "auto-cancel after hold failure"
	if holdErr != nil {
		reason = fmt.Sprintf("auto-cancel after hold failure: %v", holdErr)
	}
	cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), auditWriteTimeout)
	defer cancel()
	return r.ApprovalManager.CancelRequest(cancelCtx, approvalRequestID, reason)
}

func requiresInlineScriptApproval(action actions.ActionDefinition) bool {
	if !strings.EqualFold(strings.TrimSpace(action.Adapter.Type), "applescript") {
		return false
	}
	if strings.TrimSpace(action.Adapter.ScriptSource) == "" {
		return false
	}
	return strings.TrimSpace(action.Adapter.ApprovalRef) != ""
}

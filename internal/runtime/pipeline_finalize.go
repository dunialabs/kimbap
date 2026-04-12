package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
)

func resolveErrorStatus(ctx context.Context, err *actions.ExecutionError) (actions.ExecutionStatus, int) {
	status := actions.StatusError
	if errors.Is(ctx.Err(), context.Canceled) {
		status = actions.StatusCancelled
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		status = actions.StatusTimeout
	}
	httpStatus := 500
	if err != nil && err.HTTPStatus > 0 {
		httpStatus = err.HTTPStatus
	}
	return status, httpStatus
}

func (r *Runtime) finalizeWithError(ctx context.Context, result *actions.ExecutionResult, req actions.ExecutionRequest, err *actions.ExecutionError, startedAt time.Time, policyDecision string, approvalRequestID string) actions.ExecutionResult {
	status, httpStatus := resolveErrorStatus(ctx, err)
	return r.finalizeWithStatus(ctx, result, req, status, err, nil, httpStatus, startedAt, policyDecision, approvalRequestID)
}

func (r *Runtime) finalizeWithErrorOutput(ctx context.Context, result *actions.ExecutionResult, req actions.ExecutionRequest, err *actions.ExecutionError, output map[string]any, startedAt time.Time, policyDecision string, approvalRequestID string) actions.ExecutionResult {
	status, httpStatus := resolveErrorStatus(ctx, err)
	return r.finalizeWithStatus(ctx, result, req, status, err, output, httpStatus, startedAt, policyDecision, approvalRequestID)
}

func transformErrorOutput(req actions.ExecutionRequest, adapterResult *adapters.AdapterResult) map[string]any {
	if adapterResult == nil || adapterResult.Output == nil {
		return nil
	}
	output := adapterResult.Output
	outputMode, _ := req.Input["_output_mode"].(string)
	if outputMode == "raw" || req.Action.TextFilterConfig == nil {
		return output
	}
	filtered, err := ApplyTextFilter(output, req.Action.TextFilterConfig)
	if err != nil {
		return output
	}
	return filtered
}

func (r *Runtime) finalizeWithStatus(ctx context.Context, result *actions.ExecutionResult, req actions.ExecutionRequest, status actions.ExecutionStatus, execErr *actions.ExecutionError, output map[string]any, httpStatus int, startedAt time.Time, policyDecision string, approvalRequestID string) actions.ExecutionResult {
	execErr = annotateExecutionError(execErr, policyDecision)
	result.Status = status
	result.Error = execErr
	result.HTTPStatus = httpStatus
	result.Output = output
	result.PolicyDecision = policyDecision
	result.DurationMS = r.now().Sub(startedAt).Milliseconds()
	result.Retryable = execErr != nil && execErr.Retryable
	if result.Meta == nil {
		result.Meta = map[string]any{}
	}
	if category := ExecutionErrorCategory(execErr); category != "" {
		result.Meta["error_category"] = category
	}
	if approvalRequestID != "" {
		result.Meta["approval_request_id"] = approvalRequestID
	}
	if err := ctx.Err(); err != nil {
		result.Meta["context_error"] = err.Error()
	}

	auditRef := strings.TrimSpace(req.Action.Adapter.AuditRef)
	if auditRef == "" {
		auditRef = req.RequestID
	}
	if auditRef == "" {
		auditRef = req.TraceID
	}
	result.AuditRef = auditRef
	result.Meta["audit_ref"] = auditRef

	auditCtx, cancelAudit := context.WithTimeout(context.WithoutCancel(ctx), auditWriteTimeout)
	defer cancelAudit()
	if auditErr := r.writeAudit(auditCtx, req, *result); auditErr != nil {
		if !r.AuditRequired {
			return *result
		}
		result.Meta["pre_audit_status"] = string(result.Status)
		result.Meta["pre_audit_http_status"] = result.HTTPStatus
		result.Meta["pre_audit_retryable"] = result.Retryable
		if result.Error != nil {
			result.Meta["original_error"] = result.Error.Message
			result.Meta["original_error_code"] = result.Error.Code
		}
		result.Status = actions.StatusError
		result.Error = annotateExecutionError(auditErr, policyDecision)
		result.HTTPStatus = auditErr.HTTPStatus
		result.Output = nil
		result.Retryable = auditErr.Retryable
		if category := ExecutionErrorCategory(result.Error); category != "" {
			result.Meta["error_category"] = category
		}
	}
	return *result
}

func (r *Runtime) writeAudit(ctx context.Context, req actions.ExecutionRequest, result actions.ExecutionResult) *actions.ExecutionError {
	if r.AuditWriter == nil {
		if r.AuditRequired {
			return actions.NewExecutionError(actions.ErrAuditRequired, "audit writer unavailable but audit is required", 500, false, nil)
		}
		return nil
	}
	event := AuditEvent{
		RequestID:      req.RequestID,
		TraceID:        req.TraceID,
		TenantID:       req.TenantID,
		PrincipalID:    req.Principal.ID,
		AgentName:      req.Principal.AgentName,
		ActionName:     req.Action.Name,
		Input:          cloneInputMap(req.Input),
		Mode:           req.Mode,
		Status:         result.Status,
		HTTPStatus:     result.HTTPStatus,
		PolicyDecision: result.PolicyDecision,
		DurationMS:     result.DurationMS,
		Timestamp:      r.now(),
		Meta:           cloneMetaMap(result.Meta),
	}
	if result.Error != nil {
		event.ErrorCode = result.Error.Code
		msg := result.Error.Message
		for len(msg) > 256 {
			_, size := utf8.DecodeLastRuneInString(msg)
			msg = msg[:len(msg)-size]
		}
		event.ErrorMessage = msg
	}
	if approvalID, ok := result.Meta["approval_request_id"].(string); ok {
		event.ApprovalRequest = approvalID
	}
	if err := r.AuditWriter.Write(ctx, event); err != nil {
		if r.AuditRequired {
			return actions.NewExecutionError(actions.ErrAuditRequired, fmt.Sprintf("audit write failed: %v", err), 500, false, nil)
		}
		_, _ = fmt.Fprintf(os.Stderr, "warning: audit write failed for request %s: %v\n", req.RequestID, err)
	}
	return nil
}

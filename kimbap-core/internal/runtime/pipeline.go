package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/adapters"
)

type PolicyRequest struct {
	TenantID       string
	Principal      actions.Principal
	Action         actions.ActionDefinition
	Input          map[string]any
	Mode           actions.ExecutionMode
	Session        *actions.SessionContext
	Classification *actions.ClassificationInfo
}

type PolicyDecision struct {
	Decision string
	Reason   string
	RuleID   string
	Meta     map[string]any
}

type CredentialResolver interface {
	Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error)
}

type PolicyEvaluator interface {
	Evaluate(ctx context.Context, req PolicyRequest) (*PolicyDecision, error)
}

type AuditEvent struct {
	RequestID       string
	TraceID         string
	TenantID        string
	PrincipalID     string
	ActionName      string
	Mode            actions.ExecutionMode
	Status          actions.ExecutionStatus
	HTTPStatus      int
	ErrorCode       string
	PolicyDecision  string
	ApprovalRequest string
	DurationMS      int64
	Timestamp       time.Time
	Meta            map[string]any
}

type AuditWriter interface {
	Write(ctx context.Context, event AuditEvent) error
}

type ListOptions struct {
	Namespace string
	Resource  string
	Verb      string
	Limit     int
}

type ActionRegistry interface {
	Lookup(ctx context.Context, name string) (*actions.ActionDefinition, error)
	List(ctx context.Context, opts ListOptions) ([]actions.ActionDefinition, error)
}

type ApprovalRequest struct {
	RequestID string
	TraceID   string
	TenantID  string
	Principal actions.Principal
	Action    actions.ActionDefinition
	Input     map[string]any
	Reason    string
	Deadline  *time.Time
	Meta      map[string]any
}

type ApprovalResult struct {
	Approved  bool
	RequestID string
	Timeout   bool
	Reason    string
	Meta      map[string]any
}

type ApprovalManager interface {
	CreateRequest(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error)
}

type Runtime struct {
	PolicyEvaluator    PolicyEvaluator
	CredentialResolver CredentialResolver
	AuditWriter        AuditWriter
	ActionRegistry     ActionRegistry
	ApprovalManager    ApprovalManager
	Adapters           map[string]adapters.Adapter
	Now                func() time.Time
}

type TraceStep struct {
	Step       string `json:"step"`
	Status     string `json:"status"`
	Detail     string `json:"detail,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type TraceCollector struct {
	Steps    []TraceStep
	start    time.Time
	lastStep time.Time
}

func NewTraceCollector() *TraceCollector {
	now := time.Now()
	return &TraceCollector{start: now, lastStep: now}
}

func (tc *TraceCollector) Record(step, status, detail string) {
	if tc == nil {
		return
	}
	now := time.Now()
	tc.Steps = append(tc.Steps, TraceStep{
		Step:       step,
		Status:     status,
		Detail:     detail,
		DurationMS: now.Sub(tc.lastStep).Milliseconds(),
	})
	tc.lastStep = now
}

func (r *Runtime) Execute(ctx context.Context, req actions.ExecutionRequest) actions.ExecutionResult {
	return r.execute(ctx, req, nil)
}

func (r *Runtime) ExecuteWithTrace(ctx context.Context, req actions.ExecutionRequest) (actions.ExecutionResult, []TraceStep) {
	tc := NewTraceCollector()
	result := r.execute(ctx, req, tc)
	return result, tc.Steps
}

func (r *Runtime) execute(ctx context.Context, req actions.ExecutionRequest, trace *TraceCollector) actions.ExecutionResult {
	startedAt := r.now()
	result := actions.ExecutionResult{
		RequestID:      req.RequestID,
		TraceID:        req.TraceID,
		Status:         actions.StatusError,
		HTTPStatus:     500,
		IdempotencyKey: req.IdempotencyKey,
		Meta:           map[string]any{},
	}
	trace.Record("start", "ok", "")

	ctx, cancel := withTimeout(ctx, req.Timeout)
	defer cancel()

	principalErr := r.authenticatePrincipal(req)
	if principalErr != nil {
		trace.Record("authenticate_principal", "error", principalErr.Error())
		return r.finalizeWithError(ctx, &result, req, principalErr, startedAt, "deny", "")
	}
	trace.Record("authenticate_principal", "ok", "")

	tenantID, tenantErr := r.resolveTenant(req)
	if tenantErr != nil {
		trace.Record("resolve_tenant", "error", tenantErr.Error())
		return r.finalizeWithError(ctx, &result, req, tenantErr, startedAt, "deny", "")
	}
	req.TenantID = tenantID
	trace.Record("resolve_tenant", "ok", tenantID)

	actionDef, resolveErr := r.resolveAction(ctx, req)
	if resolveErr != nil {
		trace.Record("resolve_action", "error", resolveErr.Error())
		return r.finalizeWithError(ctx, &result, req, resolveErr, startedAt, "deny", "")
	}
	req.Action = actionDef
	trace.Record("resolve_action", "ok", req.Action.Name)

	if validationErr := actions.ValidateInput(req.Action.InputSchema, req.Input); validationErr != nil {
		trace.Record("validate_input", "error", validationErr.Error())
		return r.finalizeWithError(ctx, &result, req, validationErr, startedAt, "deny", "")
	}
	trace.Record("validate_input", "ok", "")

	if !req.Action.Idempotent && strings.TrimSpace(req.IdempotencyKey) == "" {
		idempotencyErr := actions.NewExecutionError(actions.ErrIdempotencyRequired, "idempotency key required for non-idempotent action", 400, false, map[string]any{"action": req.Action.Name})
		trace.Record("check_idempotency", "error", idempotencyErr.Error())
		return r.finalizeWithError(ctx, &result, req, idempotencyErr, startedAt, "deny", "")
	}
	trace.Record("check_idempotency", "ok", "")

	policyDecision := "allow"
	approvalRequestID := ""
	if r.PolicyEvaluator != nil {
		decision, evalErr := r.PolicyEvaluator.Evaluate(ctx, PolicyRequest{
			TenantID:       req.TenantID,
			Principal:      req.Principal,
			Action:         req.Action,
			Input:          req.Input,
			Mode:           req.Mode,
			Session:        req.Session,
			Classification: req.Classification,
		})
		if evalErr != nil {
			errResult := actions.NewExecutionError(actions.ErrDownstreamUnavailable, evalErr.Error(), 500, true, nil)
			trace.Record("evaluate_policy", "error", errResult.Error())
			return r.finalizeWithError(ctx, &result, req, errResult, startedAt, "deny", "")
		}
		if decision != nil {
			policyDecision = normalizePolicyDecision(decision.Decision)
			trace.Record("evaluate_policy", "ok", policyDecision)
		} else {
			trace.Record("evaluate_policy", "ok", "allow")
		}
	} else {
		trace.Record("evaluate_policy", "skipped", "policy evaluator unavailable")
	}

	if policyDecision == "deny" {
		errResult := actions.NewExecutionError(actions.ErrUnauthorized, "policy denied action", 403, false, nil)
		trace.Record("check_policy_decision", "error", errResult.Error())
		return r.finalizeWithError(ctx, &result, req, errResult, startedAt, "deny", "")
	}
	trace.Record("check_policy_decision", "ok", policyDecision)

	approvalNeeded := policyDecision == "require_approval" || req.Action.ApprovalHint == actions.ApprovalRequired
	if approvalNeeded {
		trace.Record("request_approval", "ok", "requested")
		approvalRes, approvalErr := r.requestApproval(ctx, req)
		if approvalErr != nil {
			mapped := actions.NewExecutionError(actions.ErrApprovalRequired, approvalErr.Error(), 202, false, nil)
			trace.Record("request_approval", "error", mapped.Error())
			return r.finalizeWithStatus(ctx, &result, req, actions.StatusApprovalRequired, mapped, nil, 202, startedAt, "require_approval", approvalRequestID)
		}
		if approvalRes != nil {
			approvalRequestID = approvalRes.RequestID
			if approvalRes.Timeout {
				timeoutErr := actions.NewExecutionError(actions.ErrApprovalTimeout, "approval request timed out", 408, false, map[string]any{"approval_request_id": approvalRes.RequestID})
				trace.Record("request_approval", "error", timeoutErr.Error())
				return r.finalizeWithStatus(ctx, &result, req, actions.StatusTimeout, timeoutErr, nil, 408, startedAt, "require_approval", approvalRes.RequestID)
			}
			if !approvalRes.Approved {
				approvalErr := actions.NewExecutionError(actions.ErrApprovalRequired, "approval required", 202, false, map[string]any{"approval_request_id": approvalRes.RequestID})
				trace.Record("request_approval", "error", approvalErr.Error())
				return r.finalizeWithStatus(ctx, &result, req, actions.StatusApprovalRequired, approvalErr, nil, 202, startedAt, "require_approval", approvalRes.RequestID)
			}
			trace.Record("request_approval", "ok", approvalRes.RequestID)
		}
	} else {
		trace.Record("request_approval", "skipped", "not required")
	}

	authType := req.Action.Auth.Type
	if authType == "" {
		authType = actions.AuthTypeNone
	}
	if authType == actions.AuthTypeNone {
		trace.Record("resolve_credentials", "skipped", "auth none")
	} else if req.Credentials != nil {
		trace.Record("resolve_credentials", "ok", "provided")
	} else {
		if r.CredentialResolver == nil {
			missingErr := actions.NewExecutionError(actions.ErrCredentialMissing, "credential resolver unavailable", 500, false, nil)
			trace.Record("resolve_credentials", "error", missingErr.Error())
			return r.finalizeWithError(ctx, &result, req, missingErr, startedAt, policyDecision, approvalRequestID)
		}
		creds, credErr := r.CredentialResolver.Resolve(ctx, req.TenantID, req.Action.Auth)
		if credErr != nil {
			mapped := actions.NewExecutionError(actions.ErrDownstreamUnavailable, credErr.Error(), 502, true, nil)
			trace.Record("resolve_credentials", "error", mapped.Error())
			return r.finalizeWithError(ctx, &result, req, mapped, startedAt, policyDecision, approvalRequestID)
		}
		if creds == nil && !req.Action.Auth.Optional {
			mapped := actions.NewExecutionError(actions.ErrCredentialMissing, "credentials not found", 401, false, nil)
			trace.Record("resolve_credentials", "error", mapped.Error())
			return r.finalizeWithError(ctx, &result, req, mapped, startedAt, policyDecision, approvalRequestID)
		}
		req.Credentials = creds
		trace.Record("resolve_credentials", "ok", strings.TrimSpace(req.Action.Auth.CredentialRef))
	}

	adapter, adapterErr := r.getAdapter(req.Action.Adapter.Type)
	if adapterErr != nil {
		trace.Record("get_adapter", "error", adapterErr.Error())
		return r.finalizeWithError(ctx, &result, req, adapterErr, startedAt, policyDecision, approvalRequestID)
	}
	trace.Record("get_adapter", "ok", adapter.Type())
	if err := adapter.Validate(req.Action); err != nil {
		mapped := actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), 400, false, nil)
		trace.Record("validate_adapter", "error", mapped.Error())
		return r.finalizeWithError(ctx, &result, req, mapped, startedAt, policyDecision, approvalRequestID)
	}
	trace.Record("validate_adapter", "ok", "")

	adapterResult, executeErr := adapter.Execute(ctx, adapters.AdapterRequest{
		Action:      req.Action,
		Input:       req.Input,
		Credentials: req.Credentials,
		RequestID:   req.RequestID,
		Timeout:     req.Timeout,
	})
	if executeErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(executeErr, context.DeadlineExceeded) {
			timeoutErr := actions.NewExecutionError(actions.ErrDownstreamUnavailable, "execution timed out", 504, true, nil)
			trace.Record("execute_adapter", "error", timeoutErr.Error())
			return r.finalizeWithStatus(ctx, &result, req, actions.StatusTimeout, timeoutErr, nil, 504, startedAt, policyDecision, approvalRequestID)
		}
		mapped := actions.AsExecutionError(executeErr)
		trace.Record("execute_adapter", "error", mapped.Error())
		return r.finalizeWithError(ctx, &result, req, mapped, startedAt, policyDecision, approvalRequestID)
	}

	if adapterResult == nil {
		mapped := actions.NewExecutionError(actions.ErrDownstreamUnavailable, "adapter returned nil result", 502, true, nil)
		trace.Record("execute_adapter", "error", mapped.Error())
		return r.finalizeWithError(ctx, &result, req, mapped, startedAt, policyDecision, approvalRequestID)
	}
	trace.Record("execute_adapter", "ok", fmt.Sprintf("http_status=%d", adapterResult.HTTPStatus))

	if result.Meta == nil {
		result.Meta = map[string]any{}
	}
	result.Meta["adapter_type"] = adapter.Type()
	result.Meta["http_headers"] = redactHeaders(adapterResult.Headers)
	result.Meta["raw_response_bytes"] = len(adapterResult.RawBody)
	trace.Record("finalize", "ok", "success")

	return r.finalizeWithStatus(
		ctx,
		&result,
		req,
		actions.StatusSuccess,
		nil,
		adapterResult.Output,
		adapterResult.HTTPStatus,
		startedAt,
		policyDecision,
		approvalRequestID,
	)
}

func (r *Runtime) authenticatePrincipal(req actions.ExecutionRequest) *actions.ExecutionError {
	if strings.TrimSpace(req.Principal.ID) == "" {
		return actions.NewExecutionError(actions.ErrUnauthenticated, "principal identity required", 401, false, nil)
	}
	return nil
}

func (r *Runtime) resolveTenant(req actions.ExecutionRequest) (string, *actions.ExecutionError) {
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(req.Principal.TenantID)
	}
	if tenantID == "" {
		return "", actions.NewExecutionError(actions.ErrUnauthorized, "tenant context is required", 403, false, nil)
	}
	return tenantID, nil
}

func (r *Runtime) resolveAction(ctx context.Context, req actions.ExecutionRequest) (actions.ActionDefinition, *actions.ExecutionError) {
	if strings.TrimSpace(req.Action.Name) != "" {
		if r.ActionRegistry != nil {
			resolved, err := r.ActionRegistry.Lookup(ctx, req.Action.Name)
			if err != nil {
				return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrActionNotFound, err.Error(), 404, false, map[string]any{"action": req.Action.Name})
			}
			if resolved != nil {
				return *resolved, nil
			}
		}
		return req.Action, nil
	}

	if req.Classification == nil || strings.TrimSpace(req.Classification.ActionName) == "" {
		return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrClassificationFailed, "action resolution failed", 400, false, nil)
	}
	if r.ActionRegistry == nil {
		return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrActionNotFound, "action registry unavailable", 500, false, nil)
	}
	resolved, err := r.ActionRegistry.Lookup(ctx, req.Classification.ActionName)
	if err != nil || resolved == nil {
		message := "action not found"
		if err != nil {
			message = err.Error()
		}
		return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrActionNotFound, message, 404, false, map[string]any{"action": req.Classification.ActionName})
	}
	return *resolved, nil
}

func (r *Runtime) requestApproval(ctx context.Context, req actions.ExecutionRequest) (*ApprovalResult, *actions.ExecutionError) {
	if r.ApprovalManager == nil {
		return nil, actions.NewExecutionError(actions.ErrApprovalRequired, "approval manager unavailable", 202, false, nil)
	}
	approvalResult, err := r.ApprovalManager.CreateRequest(ctx, ApprovalRequest{
		RequestID: req.RequestID,
		TraceID:   req.TraceID,
		TenantID:  req.TenantID,
		Principal: req.Principal,
		Action:    req.Action,
		Input:     req.Input,
		Meta: map[string]any{
			"mode": req.Mode,
		},
	})
	if err != nil {
		return nil, actions.NewExecutionError(actions.ErrApprovalRequired, err.Error(), 202, false, nil)
	}
	if approvalResult == nil {
		return nil, actions.NewExecutionError(actions.ErrApprovalRequired, "approval response missing", 202, false, nil)
	}
	return approvalResult, nil
}

func (r *Runtime) getAdapter(adapterType string) (adapters.Adapter, *actions.ExecutionError) {
	kind := strings.TrimSpace(adapterType)
	if kind == "" {
		kind = "http"
	}
	if r.Adapters == nil {
		return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "adapter registry unavailable", 500, false, nil)
	}
	adapter, ok := r.Adapters[kind]
	if !ok || adapter == nil {
		return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, fmt.Sprintf("adapter %q not found", kind), 500, false, nil)
	}
	return adapter, nil
}

func (r *Runtime) finalizeWithError(
	ctx context.Context,
	result *actions.ExecutionResult,
	req actions.ExecutionRequest,
	err *actions.ExecutionError,
	startedAt time.Time,
	policyDecision string,
	approvalRequestID string,
) actions.ExecutionResult {
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
	return r.finalizeWithStatus(ctx, result, req, status, err, nil, httpStatus, startedAt, policyDecision, approvalRequestID)
}

func (r *Runtime) finalizeWithStatus(
	ctx context.Context,
	result *actions.ExecutionResult,
	req actions.ExecutionRequest,
	status actions.ExecutionStatus,
	execErr *actions.ExecutionError,
	output map[string]any,
	httpStatus int,
	startedAt time.Time,
	policyDecision string,
	approvalRequestID string,
) actions.ExecutionResult {
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
	if approvalRequestID != "" {
		result.Meta["approval_request_id"] = approvalRequestID
	}
	if err := ctx.Err(); err != nil {
		result.Meta["context_error"] = err.Error()
	}

	auditRef := req.RequestID
	if auditRef == "" {
		auditRef = req.TraceID
	}
	result.AuditRef = auditRef

	r.writeAudit(ctx, req, *result)
	return *result
}

func (r *Runtime) writeAudit(ctx context.Context, req actions.ExecutionRequest, result actions.ExecutionResult) {
	if r.AuditWriter == nil {
		return
	}
	event := AuditEvent{
		RequestID:      req.RequestID,
		TraceID:        req.TraceID,
		TenantID:       req.TenantID,
		PrincipalID:    req.Principal.ID,
		ActionName:     req.Action.Name,
		Mode:           req.Mode,
		Status:         result.Status,
		HTTPStatus:     result.HTTPStatus,
		PolicyDecision: result.PolicyDecision,
		DurationMS:     result.DurationMS,
		Timestamp:      r.now(),
		Meta:           result.Meta,
	}
	if result.Error != nil {
		event.ErrorCode = result.Error.Code
	}
	if approvalID, ok := result.Meta["approval_request_id"].(string); ok {
		event.ApprovalRequest = approvalID
	}
	if err := r.AuditWriter.Write(ctx, event); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: audit write failed for request %s: %v\n", req.RequestID, err)
	}
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func normalizePolicyDecision(decision string) string {
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "allow":
		return "allow"
	case "deny":
		return "deny"
	case "require_approval":
		return "require_approval"
	default:
		return "deny"
	}
}

func (r *Runtime) now() time.Time {
	if r.Now == nil {
		return time.Now()
	}
	return r.Now()
}

func redactHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}

	redacted := make(map[string]string, len(headers))
	for key, value := range headers {
		if isSensitiveHeader(key) {
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func isSensitiveHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "authorization", "proxy-authorization", "x-api-key", "x-auth-token", "x-access-token", "cookie", "set-cookie":
		return true
	default:
		return false
	}
}

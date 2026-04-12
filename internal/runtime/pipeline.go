package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
)

func (r *Runtime) Execute(ctx context.Context, req actions.ExecutionRequest) actions.ExecutionResult {
	return r.execute(ctx, req, nil)
}

func (r *Runtime) ExecuteWithTrace(ctx context.Context, req actions.ExecutionRequest) (actions.ExecutionResult, []TraceStep) {
	tc := NewTraceCollector(r.now)
	result := r.execute(ctx, req, tc)
	return result, tc.Steps
}

func (r *Runtime) ResumeApproved(ctx context.Context, approvalRequestID string) actions.ExecutionResult {
	if r.HeldExecutionStore == nil {
		return actions.ExecutionResult{
			Status:     actions.StatusError,
			HTTPStatus: 500,
			Error:      annotateExecutionError(actions.NewExecutionError(actions.ErrDownstreamUnavailable, "held execution store unavailable", 500, false, nil), "require_approval"),
		}
	}

	held, err := r.HeldExecutionStore.Resume(ctx, approvalRequestID)
	if err != nil {
		return actions.ExecutionResult{
			Status:     actions.StatusError,
			HTTPStatus: 500,
			Error:      annotateExecutionError(actions.NewExecutionError(actions.ErrDownstreamUnavailable, "held execution unavailable", 500, true, nil), "require_approval"),
		}
	}
	if held == nil {
		return actions.ExecutionResult{
			Status:     actions.StatusError,
			HTTPStatus: 404,
			Error:      annotateExecutionError(actions.NewExecutionError(actions.ErrActionNotFound, "held execution not found", 404, false, nil), "require_approval"),
		}
	}

	if held.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = withTimeout(ctx, held.Timeout)
		defer cancel()
	}

	startedAt := r.now()
	result := actions.ExecutionResult{
		RequestID:      held.RequestID,
		TraceID:        held.TraceID,
		Status:         actions.StatusError,
		HTTPStatus:     500,
		IdempotencyKey: held.IdempotencyKey,
		Meta:           map[string]any{},
	}

	reholdIfRetryable := func(res actions.ExecutionResult) actions.ExecutionResult {
		if res.Error != nil && res.Error.Retryable {
			holdCtx, holdCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer holdCancel()
			if holdErr := r.HeldExecutionStore.Hold(holdCtx, approvalRequestID, *held); holdErr != nil {
				if res.Meta == nil {
					res.Meta = map[string]any{}
				}
				res.Meta["resume_retry_hold_error"] = holdErr.Error()
			}
		}
		return res
	}

	if principalErr := r.authenticatePrincipal(ctx, *held); principalErr != nil {
		return reholdIfRetryable(r.finalizeWithError(ctx, &result, *held, principalErr, startedAt, "require_approval", approvalRequestID))
	}

	tenantID, tenantErr := r.resolveTenant(*held)
	if tenantErr != nil {
		return reholdIfRetryable(r.finalizeWithError(ctx, &result, *held, tenantErr, startedAt, "require_approval", approvalRequestID))
	}
	held.TenantID = tenantID

	if validationErr := actions.ValidateInput(held.Action.InputSchema, held.Input); validationErr != nil {
		return reholdIfRetryable(r.finalizeWithError(ctx, &result, *held, validationErr, startedAt, "require_approval", approvalRequestID))
	}

	if sanitizeErr := SanitizeInput(held.Input); sanitizeErr != nil {
		return reholdIfRetryable(r.finalizeWithError(ctx, &result, *held, actions.AsExecutionError(sanitizeErr), startedAt, "require_approval", approvalRequestID))
	}

	if r.PolicyEvaluator != nil {
		decision, evalErr := r.PolicyEvaluator.Evaluate(ctx, PolicyRequest{
			TenantID:       held.TenantID,
			Principal:      held.Principal,
			Action:         held.Action,
			Input:          held.Input,
			Mode:           held.Mode,
			Session:        held.Session,
			Classification: held.Classification,
		})
		if evalErr != nil {
			return reholdIfRetryable(r.finalizeWithError(ctx, &result, *held, actions.NewExecutionError(actions.ErrDownstreamUnavailable, evalErr.Error(), 500, true, nil), startedAt, "", approvalRequestID))
		}
		if decision != nil && normalizePolicyDecision(decision.Decision) == "deny" {
			return reholdIfRetryable(r.finalizeWithError(ctx, &result, *held, actions.NewExecutionError(actions.ErrUnauthorized, "policy denied action on resume: "+decision.Reason, 403, false, nil), startedAt, "deny", approvalRequestID))
		}
	}

	return reholdIfRetryable(r.executeFromCredentialsWithState(ctx, *held, nil, startedAt, "require_approval", approvalRequestID))
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

	principalErr := r.authenticatePrincipal(ctx, req)
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

	if req.Action.Defaults != nil {
		cloned := make(map[string]any, len(req.Input)+len(req.Action.Defaults))
		for k, v := range req.Input {
			cloned[k] = v
		}
		applied := 0
		for k, v := range req.Action.Defaults {
			if _, exists := cloned[k]; !exists {
				cloned[k] = deepCloneValue(v)
				applied++
			}
		}
		req.Input = cloned
		trace.Record("apply_defaults", "ok", fmt.Sprintf("%d defaults applied", applied))
	}

	if validationErr := actions.ValidateInput(req.Action.InputSchema, req.Input); validationErr != nil {
		trace.Record("validate_input", "error", validationErr.Error())
		return r.finalizeWithError(ctx, &result, req, validationErr, startedAt, "deny", "")
	}
	trace.Record("validate_input", "ok", "")

	if sanitizeErr := SanitizeInput(req.Input); sanitizeErr != nil {
		mapped := actions.AsExecutionError(sanitizeErr)
		trace.Record("sanitize_input", "error", mapped.Error())
		return r.finalizeWithError(ctx, &result, req, mapped, startedAt, "deny", "")
	}
	trace.Record("sanitize_input", "ok", "")

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
			return r.finalizeWithError(ctx, &result, req, errResult, startedAt, "", "")
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

	approvalNeeded := policyDecision == "require_approval" || req.Action.ApprovalHint == actions.ApprovalRequired || requiresInlineScriptApproval(req.Action)
	if approvalNeeded {
		approvalRes, approvalErr := r.requestApproval(ctx, req)
		if approvalErr != nil {
			trace.Record("request_approval", "error", approvalErr.Error())
			return r.finalizeWithError(ctx, &result, req, approvalErr, startedAt, "require_approval", approvalRequestID)
		}
		approvalRequestID = approvalRes.RequestID
		if approvalRes.Timeout {
			timeoutErr := actions.NewExecutionError(actions.ErrApprovalTimeout, "approval request timed out", 408, false, map[string]any{"approval_request_id": approvalRes.RequestID})
			trace.Record("request_approval", "error", timeoutErr.Error())
			return r.finalizeWithStatus(ctx, &result, req, actions.StatusTimeout, timeoutErr, nil, 408, startedAt, "require_approval", approvalRes.RequestID)
		}
		if !approvalRes.Approved {
			if r.HeldExecutionStore == nil {
				holdMissingErr := actions.NewExecutionError(actions.ErrDownstreamUnavailable, "approval resume unavailable: held execution store is not configured", 500, false, nil)
				trace.Record("hold_execution", "error", holdMissingErr.Error())
				return r.finalizeWithError(ctx, &result, req, holdMissingErr, startedAt, "require_approval", approvalRes.RequestID)
			}
			if holdErr := r.HeldExecutionStore.Hold(ctx, approvalRes.RequestID, req); holdErr != nil {
				holdFailErr := actions.NewExecutionError(actions.ErrDownstreamUnavailable, fmt.Sprintf("failed to hold execution for approval: %v", holdErr), 500, false, nil)
				if cancelErr := r.cancelApprovalOnHoldFailure(ctx, approvalRes.RequestID, holdErr); cancelErr != nil {
					if holdFailErr.Details == nil {
						holdFailErr.Details = map[string]any{}
					}
					holdFailErr.Details["approval_cancel_error"] = cancelErr.Error()
				}
				trace.Record("hold_execution", "error", holdFailErr.Error())
				return r.finalizeWithError(ctx, &result, req, holdFailErr, startedAt, "require_approval", approvalRes.RequestID)
			}
			approvalErr := actions.NewExecutionError(actions.ErrApprovalRequired, "approval required", 202, false, map[string]any{"approval_request_id": approvalRes.RequestID})
			trace.Record("request_approval", "error", approvalErr.Error())
			return r.finalizeWithStatus(ctx, &result, req, actions.StatusApprovalRequired, approvalErr, nil, 202, startedAt, "require_approval", approvalRes.RequestID)
		}
		trace.Record("request_approval", "ok", approvalRes.RequestID)
	} else {
		trace.Record("request_approval", "skipped", "not required")
	}

	return r.executeFromCredentialsWithState(ctx, req, trace, startedAt, policyDecision, approvalRequestID)
}

func (r *Runtime) executeFromCredentialsWithState(
	ctx context.Context,
	req actions.ExecutionRequest,
	trace *TraceCollector,
	startedAt time.Time,
	policyDecision string,
	approvalRequestID string,
) actions.ExecutionResult {
	result := actions.ExecutionResult{
		RequestID:      req.RequestID,
		TraceID:        req.TraceID,
		Status:         actions.StatusError,
		HTTPStatus:     500,
		IdempotencyKey: req.IdempotencyKey,
		Meta:           map[string]any{},
	}

	authType := req.Action.Auth.Type
	if authType == "" {
		authType = actions.AuthTypeNone
	}
	if authType == actions.AuthTypeNone {
		req.Credentials = nil
		trace.Record("resolve_credentials", "skipped", "auth none")
	} else if req.Credentials != nil {
		trace.Record("resolve_credentials", "ok", "provided")
	} else if r.CredentialResolver == nil {
		if req.Action.Auth.Optional {
			req.Credentials = nil
			trace.Record("resolve_credentials", "ok", "optional-no-resolver")
		} else {
			missingErr := actions.NewExecutionError(actions.ErrCredentialMissing, "credential resolver unavailable", 500, false, nil)
			trace.Record("resolve_credentials", "error", missingErr.Error())
			return r.finalizeWithError(ctx, &result, req, missingErr, startedAt, policyDecision, approvalRequestID)
		}
	} else {
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

	adapterAction := applyRuntimeActionOverrides(req.Action, req.Input)
	adapterResult, executeErr := adapter.Execute(ctx, adapters.AdapterRequest{
		Action:      adapterAction,
		Input:       stripRuntimeKeys(req.Input),
		Credentials: req.Credentials,
		RequestID:   req.RequestID,
		Timeout:     req.Timeout,
	})
	if executeErr != nil {
		errorOutput := transformErrorOutput(req, adapterResult)
		if adapterResult != nil {
			if result.Meta == nil {
				result.Meta = map[string]any{}
			}
			result.Meta["adapter_type"] = adapter.Type()
			result.Meta["http_headers"] = redactHeaders(adapterResult.Headers)
			result.Meta["raw_response_bytes"] = len(adapterResult.RawBody)
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(executeErr, context.DeadlineExceeded) {
			timeoutErr := actions.NewExecutionError(actions.ErrDownstreamUnavailable, "execution timed out", 504, true, nil)
			trace.Record("execute_adapter", "error", timeoutErr.Error())
			return r.finalizeWithStatus(ctx, &result, req, actions.StatusTimeout, timeoutErr, errorOutput, 504, startedAt, policyDecision, approvalRequestID)
		}
		mapped := actions.AsExecutionError(executeErr)
		trace.Record("execute_adapter", "error", mapped.Error())
		return r.finalizeWithErrorOutput(ctx, &result, req, mapped, errorOutput, startedAt, policyDecision, approvalRequestID)
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

	finalResult := r.finalizeWithStatus(ctx, &result, req, actions.StatusSuccess, nil, adapterResult.Output, adapterResult.HTTPStatus, startedAt, policyDecision, approvalRequestID)

	if finalResult.Status == actions.StatusSuccess {
		outputMode, _ := req.Input["_output_mode"].(string)
		rawMode := outputMode == "raw"

		if finalResult.Meta == nil {
			finalResult.Meta = map[string]any{}
		}

		if req.Action.FilterConfig != nil && !rawMode {
			filtered, filterMeta, filterErr := ApplyFilter(finalResult.Output, req.Action.FilterConfig)
			if filterErr != nil {
				finalResult.Meta["filter_error"] = filterErr.Error()
				finalResult.Meta["filter_applied"] = false
			} else {
				finalResult.Output = filtered
				finalResult.Meta["filter_applied"] = filterMeta.Applied
				finalResult.Meta["filter_original_bytes"] = filterMeta.OriginalBytes
				finalResult.Meta["filter_result_bytes"] = filterMeta.FilteredBytes
				if filterMeta.ItemsTruncatedFrom > 0 {
					finalResult.Meta["filter_items_truncated_from"] = filterMeta.ItemsTruncatedFrom
				}
				if len(filterMeta.PartialMiss) > 0 {
					finalResult.Meta["filter_partial_miss"] = filterMeta.PartialMiss
				}
				if filterMeta.Skipped != "" {
					finalResult.Meta["filter_skipped"] = filterMeta.Skipped
				}
			}
		}

		if req.Action.TextFilterConfig != nil && !rawMode {
			filtered, tfErr := ApplyTextFilter(finalResult.Output, req.Action.TextFilterConfig)
			if tfErr == nil {
				finalResult.Output = filtered
			}
		}

		if !rawMode {
			if budget := coerceBudgetInt(req.Input["_budget"]); budget > 0 {
				budgeted, budgetMeta := ApplyBudget(finalResult.Output, budget)
				finalResult.Output = budgeted
				if budgetMeta.Applied {
					finalResult.Meta["budget_applied"] = true
					finalResult.Meta["budget_limit"] = budgetMeta.Limit
					finalResult.Meta["budget_original_bytes"] = budgetMeta.OriginalBytes
					finalResult.Meta["budget_result_bytes"] = budgetMeta.ResultBytes
				}
			}
		}

		if req.Action.CompactTemplate != nil && !rawMode {
			compacted, compactErr := ApplyCompactTemplate(finalResult.Output, req.Action.CompactTemplate)
			if compactErr == nil {
				finalResult.Output = compacted
			} else {
				finalResult.Meta["compact_error"] = compactErr.Error()
			}
		}
	}
	return finalResult
}

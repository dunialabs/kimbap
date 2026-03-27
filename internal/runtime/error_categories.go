package runtime

import (
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
)

const (
	ErrorCategoryInput      = "input"
	ErrorCategoryConfig     = "config"
	ErrorCategoryAuth       = "auth"
	ErrorCategoryPolicy     = "policy"
	ErrorCategoryApproval   = "approval"
	ErrorCategoryStore      = "store"
	ErrorCategoryRuntime    = "runtime"
	ErrorCategoryDownstream = "downstream"
	ErrorCategoryInternal   = "internal"
)

func classifyExecutionError(err *actions.ExecutionError, policyDecision string) string {
	if err == nil {
		return ErrorCategoryInternal
	}

	if existing := ExecutionErrorCategory(err); existing != "" {
		return existing
	}

	message := strings.ToLower(strings.TrimSpace(err.Message))
	decision := strings.ToLower(strings.TrimSpace(policyDecision))

	if decision == "require_approval" {
		return ErrorCategoryApproval
	}
	if decision == "deny" && strings.Contains(message, "policy") {
		return ErrorCategoryPolicy
	}

	switch err.Code {
	case actions.ErrValidationFailed,
		actions.ErrServiceInvalid,
		actions.ErrIdempotencyRequired,
		actions.ErrActionNotFound,
		actions.ErrClassificationFailed:
		return ErrorCategoryInput
	case actions.ErrCredentialMissing,
		actions.ErrUnauthenticated,
		actions.ErrUnauthorized,
		actions.ErrTokenExpired,
		actions.ErrConnectorNotLoggedIn:
		if strings.Contains(message, "policy") {
			return ErrorCategoryPolicy
		}
		return ErrorCategoryAuth
	case actions.ErrApprovalRequired, actions.ErrApprovalTimeout:
		return ErrorCategoryApproval
	case actions.ErrRateLimited, actions.ErrDownstreamUnavailable:
		return ErrorCategoryDownstream
	case actions.ErrAuditRequired:
		return ErrorCategoryRuntime
	case actions.ErrUnsafeExistingCLI, actions.ErrUnsupportedProxyProto:
		return ErrorCategoryInternal
	default:
		if strings.Contains(message, "config") {
			return ErrorCategoryConfig
		}
		if strings.Contains(message, "database") || strings.Contains(message, "store") {
			return ErrorCategoryStore
		}
		if strings.Contains(message, "policy") {
			return ErrorCategoryPolicy
		}
		if strings.Contains(message, "approval") {
			return ErrorCategoryApproval
		}
		return ErrorCategoryInternal
	}
}

func annotateExecutionError(err *actions.ExecutionError, policyDecision string) *actions.ExecutionError {
	if err == nil {
		return nil
	}
	if err.Details == nil {
		err.Details = map[string]any{}
	}
	if existing := strings.TrimSpace(ExecutionErrorCategory(err)); existing != "" {
		return err
	}
	err.Details["category"] = classifyExecutionError(err, policyDecision)
	return err
}

func ExecutionErrorCategory(err *actions.ExecutionError) string {
	if err == nil || err.Details == nil {
		return ""
	}
	raw, ok := err.Details["category"]
	if !ok {
		return ""
	}
	category, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(category)
}

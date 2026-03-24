package main

import (
	"errors"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/actions"
)

const (
	ExitSuccess    = 0
	ExitAPIError   = 1
	ExitAuthError  = 2
	ExitValidation = 3
	ExitPolicy     = 4
	ExitInternal   = 5
)

func mapErrorToExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	var execErr *actions.ExecutionError
	if errors.As(err, &execErr) {
		return mapExecutionErrorCode(execErr.Code)
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unauthorized") || strings.Contains(msg, "unauthenticated") ||
		(strings.Contains(msg, "token") && (strings.Contains(msg, "expired") || strings.Contains(msg, "invalid") || strings.Contains(msg, "revoked"))):
		return ExitAuthError
	case strings.Contains(msg, "validation") || strings.Contains(msg, "required field") ||
		strings.Contains(msg, "missing required") || strings.Contains(msg, "invalid") && strings.Contains(msg, "field"):
		return ExitValidation
	case strings.Contains(msg, "policy denied") || strings.Contains(msg, "approval required"):
		return ExitPolicy
	default:
		return ExitInternal
	}
}

func mapExecutionErrorCode(code string) int {
	switch code {
	case actions.ErrUnauthenticated, actions.ErrUnauthorized,
		actions.ErrTokenExpired, actions.ErrConnectorNotLoggedIn:
		return ExitAuthError
	case actions.ErrValidationFailed, actions.ErrSkillInvalid,
		actions.ErrIdempotencyRequired, actions.ErrActionNotFound,
		actions.ErrClassificationFailed:
		return ExitValidation
	case actions.ErrApprovalRequired, actions.ErrApprovalTimeout:
		return ExitPolicy
	case actions.ErrDownstreamUnavailable, actions.ErrRateLimited:
		return ExitAPIError
	case actions.ErrCredentialMissing:
		return ExitAuthError
	case actions.ErrUnsafeExistingCLI, actions.ErrUnsupportedProxyProto:
		return ExitInternal
	default:
		return ExitInternal
	}
}

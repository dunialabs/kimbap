package main

import (
	"errors"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/kerrors"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
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

	var kErr *kerrors.KimbapError
	if errors.As(err, &kErr) && kErr.ExitCode >= 0 {
		return kErr.ExitCode
	}

	var execErr *actions.ExecutionError
	if errors.As(err, &execErr) {
		if category := runtimepkg.ExecutionErrorCategory(execErr); category != "" {
			switch category {
			case runtimepkg.ErrorCategoryInput:
				return ExitValidation
			case runtimepkg.ErrorCategoryAuth:
				return ExitAuthError
			case runtimepkg.ErrorCategoryPolicy, runtimepkg.ErrorCategoryApproval:
				return ExitPolicy
			case runtimepkg.ErrorCategoryDownstream:
				return ExitAPIError
			default:
				return ExitInternal
			}
		}
		if execErr.Code == actions.ErrUnauthorized {
			msg := strings.ToLower(execErr.Message)
			if strings.Contains(msg, "policy denied") || strings.Contains(msg, "policy") {
				return ExitPolicy
			}
		}
		return mapExecutionErrorCode(execErr.Code)
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unauthorized") || strings.Contains(msg, "unauthenticated") ||
		(strings.Contains(msg, "token") && (strings.Contains(msg, "expired") || strings.Contains(msg, "invalid") || strings.Contains(msg, "revoked"))):
		return ExitAuthError
	case strings.Contains(msg, "validation") || strings.Contains(msg, "required field") ||
		strings.Contains(msg, "missing required") || strings.Contains(msg, " is required") ||
		strings.Contains(msg, "invalid") && strings.Contains(msg, "field"):
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
	case actions.ErrValidationFailed, actions.ErrServiceInvalid,
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

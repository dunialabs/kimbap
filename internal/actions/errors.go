package actions

import (
	"errors"
	"fmt"
)

// ErrLookupNotFound is the sentinel error returned by ActionRegistry.Lookup
// when the requested action does not exist.
var ErrLookupNotFound = errors.New("action not found")

// Spec §7.4 stable error codes (13 canonical).
const (
	ErrUnauthenticated       = "ERR_UNAUTHENTICATED"
	ErrUnauthorized          = "ERR_UNAUTHORIZED"
	ErrApprovalRequired      = "ERR_APPROVAL_REQUIRED"
	ErrApprovalTimeout       = "ERR_APPROVAL_TIMEOUT"
	ErrActionNotFound        = "ERR_ACTION_NOT_FOUND"
	ErrResourceNotFound      = "ERR_RESOURCE_NOT_FOUND"
	ErrClassificationFailed  = "ERR_CLASSIFICATION_FAILED"
	ErrServiceInvalid        = "ERR_SERVICE_INVALID"
	ErrConnectorNotLoggedIn  = "ERR_CONNECTOR_NOT_LOGGED_IN"
	ErrTokenExpired          = "ERR_TOKEN_EXPIRED"
	ErrRateLimited           = "ERR_RATE_LIMITED"
	ErrDownstreamUnavailable = "ERR_DOWNSTREAM_UNAVAILABLE"
	ErrUnsafeExistingCLI     = "ERR_UNSAFE_EXISTING_CLI"
	ErrUnsupportedProxyProto = "ERR_UNSUPPORTED_PROXY_PROTOCOL"
)

// Runtime-derived extensions required by pipeline steps (validation, credential resolution, idempotency).
const (
	ErrValidationFailed    = "ERR_VALIDATION_FAILED"
	ErrCredentialMissing   = "ERR_CREDENTIAL_MISSING"
	ErrIdempotencyRequired = "ERR_IDEMPOTENCY_REQUIRED"
	ErrAuditRequired       = "ERR_AUDIT_REQUIRED"
)

type ExecutionError struct {
	Code       string
	Message    string
	Details    map[string]any
	Retryable  bool
	HTTPStatus int
}

func (e *ExecutionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	if e.Message == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewExecutionError(code, message string, httpStatus int, retryable bool, details map[string]any) *ExecutionError {
	if details == nil {
		details = map[string]any{}
	}
	return &ExecutionError{
		Code:       code,
		Message:    message,
		Details:    details,
		Retryable:  retryable,
		HTTPStatus: httpStatus,
	}
}

func AsExecutionError(err error) *ExecutionError {
	if err == nil {
		return nil
	}
	var execErr *ExecutionError
	if errors.As(err, &execErr) {
		return execErr
	}
	return NewExecutionError(
		ErrDownstreamUnavailable,
		err.Error(),
		502,
		true,
		nil,
	)
}

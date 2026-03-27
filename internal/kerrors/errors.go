package kerrors

import (
	"errors"
	"fmt"
)

const (
	ExitSuccess    = 0
	ExitAPIError   = 1
	ExitAuthError  = 2
	ExitValidation = 3
	ExitPolicy     = 4
	ExitInternal   = 5
)

type KimbapError struct {
	Code     string
	Message  string
	Hint     string
	DocsURL  string
	ExitCode int
	Cause    error
}

func (e *KimbapError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" && e.Cause != nil {
		return e.Cause.Error()
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *KimbapError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(exitCode int, code, message, hint string) *KimbapError {
	return &KimbapError{Code: code, Message: message, Hint: hint, ExitCode: exitCode}
}

func Wrap(cause error, exitCode int, code, message, hint string) *KimbapError {
	return &KimbapError{Code: code, Message: message, Hint: hint, ExitCode: exitCode, Cause: cause}
}

func WrapWithHint(cause error, hint string) *KimbapError {
	if cause == nil {
		return nil
	}
	return &KimbapError{Hint: hint, ExitCode: ExitInternal, Cause: cause}
}

func (e *KimbapError) WithDocs(url string) *KimbapError {
	copy := *e
	copy.DocsURL = url
	return &copy
}

func (e *KimbapError) WithExitCode(code int) *KimbapError {
	copy := *e
	copy.ExitCode = code
	return &copy
}

func GetHint(err error) string {
	var kErr *KimbapError
	if errors.As(err, &kErr) {
		return kErr.Hint
	}
	return ""
}

func GetDocsURL(err error) string {
	var kErr *KimbapError
	if errors.As(err, &kErr) {
		return kErr.DocsURL
	}
	return ""
}

func GetExitCode(err error) int {
	var kErr *KimbapError
	if errors.As(err, &kErr) {
		return kErr.ExitCode
	}
	return -1
}

var ErrConfigNotFound = New(ExitInternal, "E_CONFIG_NOT_FOUND",
	"config file not found",
	"Run 'kimbap init' to create a new configuration.")

var ErrVaultLocked = New(ExitAuthError, "E_VAULT_LOCKED",
	"vault is locked",
	"Set KIMBAP_MASTER_KEY_HEX or run with KIMBAP_DEV=true for development.")

var ErrServiceNotFound = New(ExitValidation, "E_SERVICE_NOT_FOUND",
	"service not found",
	"Run 'kimbap service list' to see installed services.")

var ErrActionNotFound = New(ExitValidation, "E_ACTION_NOT_FOUND",
	"action not found",
	"Run 'kimbap actions list' to see available actions.")

var ErrManifestInvalid = New(ExitValidation, "E_MANIFEST_INVALID",
	"service manifest is invalid",
	"Run 'kimbap service validate <file>' to see detailed validation errors.")

var ErrCredentialMissing = New(ExitAuthError, "E_CREDENTIAL_MISSING",
	"credential not found in vault",
	"Run 'kimbap vault set <credential-ref>' to store the credential.")

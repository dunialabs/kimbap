package kerrors

import (
	"errors"
	"testing"
)

func TestKimbapErrorFormattingAndUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := Wrap(cause, ExitValidation, "E_TEST", "top message", "hint")

	if got := err.Error(); got != "top message: root cause" {
		t.Fatalf("Error() = %q", got)
	}
	if !errors.Is(err, cause) {
		t.Fatal("wrapped cause should be discoverable via errors.Is")
	}
	if got := GetHint(err); got != "hint" {
		t.Fatalf("GetHint() = %q, want hint", got)
	}
	if got := GetExitCode(err); got != ExitValidation {
		t.Fatalf("GetExitCode() = %d, want %d", got, ExitValidation)
	}
}

func TestWithDocsAndWithExitCodeReturnCopiedError(t *testing.T) {
	base := New(ExitInternal, "E_BASE", "base", "base hint")
	withDocs := base.WithDocs("https://docs.example.com/error")
	withExit := base.WithExitCode(ExitAuthError)

	if base.DocsURL != "" {
		t.Fatalf("base error should remain unchanged, got docs=%q", base.DocsURL)
	}
	if withDocs.DocsURL != "https://docs.example.com/error" {
		t.Fatalf("withDocs URL = %q", withDocs.DocsURL)
	}
	if withExit.ExitCode != ExitAuthError {
		t.Fatalf("withExit code = %d, want %d", withExit.ExitCode, ExitAuthError)
	}
}

func TestWrapWithHintAndNilHandling(t *testing.T) {
	if WrapWithHint(nil, "hint") != nil {
		t.Fatal("WrapWithHint(nil, ...) must return nil")
	}

	cause := errors.New("boom")
	err := WrapWithHint(cause, "do this")
	if err == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if got := err.Error(); got != "boom: boom" {
		t.Fatalf("Error() = %q", got)
	}
	if got := GetHint(err); got != "do this" {
		t.Fatalf("GetHint() = %q", got)
	}
	if got := GetExitCode(err); got != ExitInternal {
		t.Fatalf("GetExitCode() = %d, want %d", got, ExitInternal)
	}
}

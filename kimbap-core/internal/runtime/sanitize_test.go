package runtime

import (
	"strings"
	"testing"

	"github.com/dunialabs/kimbap-core/internal/actions"
)

func TestCheckPathTraversalRejectsDangerousValues(t *testing.T) {
	tests := []string{
		"../../.ssh/id_rsa",
		"..\\..\\windows\\system32",
		"~/.ssh/config",
		"/etc/passwd",
	}

	for _, value := range tests {
		err := checkPathTraversal(value)
		assertValidationError(t, err)
	}
}

func TestCheckPathTraversalAllowsNormalValues(t *testing.T) {
	err := checkPathTraversal("/workspace/projects/kimbap/config.yaml")
	if err != nil {
		t.Fatalf("expected no path traversal error, got %v", err)
	}
}

func TestCheckControlCharsRejectsNullAndEscapeSequences(t *testing.T) {
	tests := []string{
		"safe\x00unsafe",
		"\x1b[31mred\x1b[0m",
		`literal-escape-\\x1b`,
	}

	for _, value := range tests {
		err := checkControlChars(value)
		assertValidationError(t, err)
	}
}

func TestCheckInputSizeRejectsLargeString(t *testing.T) {
	input := map[string]any{
		"payload": strings.Repeat("a", defaultMaxStringBytes+1),
	}

	err := checkInputSize(input)
	assertValidationError(t, err)
}

func TestCheckDangerousPatternsRejectsShellAndSQLInjectionBasics(t *testing.T) {
	tests := []string{
		"status; rm -rf /",
		"logs | cat /etc/passwd",
		"`whoami`",
		"' OR '1'='1",
	}

	for _, value := range tests {
		err := checkDangerousPatterns(value)
		assertValidationError(t, err)
	}
}

func TestSanitizeInputRejectsNestedMaliciousValue(t *testing.T) {
	input := map[string]any{
		"safe": "ok",
		"nested": map[string]any{
			"items": []any{
				"still-safe",
				map[string]any{"path": "../../etc/passwd"},
			},
		},
	}

	err := SanitizeInput(input)
	assertValidationError(t, err)

	execErr := err.(*actions.ExecutionError)
	if execErr.Details["path"] != "input.nested.items[1].path" {
		t.Fatalf("expected nested path in details, got %+v", execErr.Details)
	}
}

func TestSanitizeInputDangerousPatternsOffByDefault(t *testing.T) {
	input := map[string]any{"cmd": "echo hi; rm -rf /"}

	err := SanitizeInput(input)
	if err != nil {
		t.Fatalf("expected shell-like input to pass by default (dangerous patterns opt-in), got %v", err)
	}
}

func TestSanitizeInputDangerousPatternsOptIn(t *testing.T) {
	input := map[string]any{"cmd": "echo hi; rm -rf /"}

	err := SanitizeInputWithOptions(input, SanitizeOptions{EnableDangerousPatternCheck: true})
	if err == nil {
		t.Fatal("expected shell injection to fail when dangerous-pattern check enabled")
	}
}

func TestSanitizeInputWithOptionsRespectsCustomSizeLimits(t *testing.T) {
	input := map[string]any{"message": "123456"}

	err := SanitizeInputWithOptions(input, SanitizeOptions{MaxTotalInputBytes: 5})
	assertValidationError(t, err)
}

func TestSanitizeInputRejectsMaliciousMapKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"path traversal key", "../../etc/passwd"},
		{"control char key", "field\x00name"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]any{tc.key: "safe-value"}
			err := SanitizeInput(input)
			assertValidationError(t, err)
		})
	}
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error but got nil")
	}
	execErr, ok := err.(*actions.ExecutionError)
	if !ok {
		t.Fatalf("expected *actions.ExecutionError, got %T", err)
	}
	if execErr.Code != actions.ErrValidationFailed {
		t.Fatalf("expected %s, got %s", actions.ErrValidationFailed, execErr.Code)
	}
}

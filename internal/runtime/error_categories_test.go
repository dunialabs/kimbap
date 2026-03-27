package runtime

import (
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
)

func TestAnnotateExecutionErrorSetsExpectedCategoryFromCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		decision string
		want     string
	}{
		{name: "validation", code: actions.ErrValidationFailed, want: ErrorCategoryInput},
		{name: "auth", code: actions.ErrUnauthenticated, want: ErrorCategoryAuth},
		{name: "approval", code: actions.ErrApprovalRequired, want: ErrorCategoryApproval},
		{name: "downstream", code: actions.ErrDownstreamUnavailable, want: ErrorCategoryDownstream},
		{name: "audit", code: actions.ErrAuditRequired, want: ErrorCategoryRuntime},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := actions.NewExecutionError(tc.code, "boom", 500, false, nil)
			annotated := annotateExecutionError(err, tc.decision)
			if got := ExecutionErrorCategory(annotated); got != tc.want {
				t.Fatalf("expected category %q, got %q", tc.want, got)
			}
		})
	}
}

func TestAnnotateExecutionErrorUsesPolicyDecisionHints(t *testing.T) {
	err := actions.NewExecutionError(actions.ErrUnauthorized, "policy denied by rule", 403, false, nil)
	annotated := annotateExecutionError(err, "deny")
	if got := ExecutionErrorCategory(annotated); got != ErrorCategoryPolicy {
		t.Fatalf("expected category %q, got %q", ErrorCategoryPolicy, got)
	}

	err = actions.NewExecutionError(actions.ErrUnauthorized, "approval pending", 403, false, nil)
	annotated = annotateExecutionError(err, "require_approval")
	if got := ExecutionErrorCategory(annotated); got != ErrorCategoryApproval {
		t.Fatalf("expected category %q, got %q", ErrorCategoryApproval, got)
	}
}

func TestAnnotateExecutionErrorPreservesExistingCategory(t *testing.T) {
	err := actions.NewExecutionError(actions.ErrValidationFailed, "invalid input", 400, false, map[string]any{"category": ErrorCategoryConfig})
	annotated := annotateExecutionError(err, "")
	if got := ExecutionErrorCategory(annotated); got != ErrorCategoryConfig {
		t.Fatalf("expected existing category %q, got %q", ErrorCategoryConfig, got)
	}
}

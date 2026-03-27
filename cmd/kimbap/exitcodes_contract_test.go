package main

import (
	"errors"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/kerrors"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
)

func TestMapErrorToExitCode_CategoryContract(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "kerror override",
			err:  kerrors.Wrap(errors.New("nope"), ExitPolicy, "E_TEST", "policy blocked", ""),
			want: ExitPolicy,
		},
		{
			name: "input category",
			err:  actions.NewExecutionError(actions.ErrValidationFailed, "invalid", 400, false, map[string]any{"category": runtimepkg.ErrorCategoryInput}),
			want: ExitValidation,
		},
		{
			name: "auth category",
			err:  actions.NewExecutionError(actions.ErrUnauthorized, "token expired", 401, false, map[string]any{"category": runtimepkg.ErrorCategoryAuth}),
			want: ExitAuthError,
		},
		{
			name: "policy category",
			err:  actions.NewExecutionError(actions.ErrUnauthorized, "policy denied", 403, false, map[string]any{"category": runtimepkg.ErrorCategoryPolicy}),
			want: ExitPolicy,
		},
		{
			name: "downstream category",
			err:  actions.NewExecutionError(actions.ErrDownstreamUnavailable, "gateway timeout", 502, true, map[string]any{"category": runtimepkg.ErrorCategoryDownstream}),
			want: ExitAPIError,
		},
		{
			name: "internal category",
			err:  actions.NewExecutionError(actions.ErrAuditRequired, "audit broken", 500, false, map[string]any{"category": runtimepkg.ErrorCategoryRuntime}),
			want: ExitInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapErrorToExitCode(tc.err); got != tc.want {
				t.Fatalf("expected exit code %d, got %d", tc.want, got)
			}
		})
	}
}

func TestMapErrorToExitCode_FallbackContract(t *testing.T) {
	if got := mapErrorToExitCode(nil); got != ExitSuccess {
		t.Fatalf("expected success exit code %d, got %d", ExitSuccess, got)
	}

	err := actions.NewExecutionError(actions.ErrRateLimited, "rate limited", 429, true, nil)
	if got := mapErrorToExitCode(err); got != ExitAPIError {
		t.Fatalf("expected fallback execution mapping to API exit, got %d", got)
	}
}

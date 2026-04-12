package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
)

func (r *Runtime) authenticatePrincipal(ctx context.Context, req actions.ExecutionRequest) *actions.ExecutionError {
	if strings.TrimSpace(req.Principal.ID) == "" {
		return actions.NewExecutionError(actions.ErrUnauthenticated, "principal identity required", 401, false, nil)
	}
	if r.PrincipalVerifier != nil {
		if err := r.PrincipalVerifier.Verify(ctx, req.Principal); err != nil {
			return actions.NewExecutionError(actions.ErrUnauthenticated, err.Error(), 401, false, nil)
		}
	}
	return nil
}

func (r *Runtime) resolveTenant(req actions.ExecutionRequest) (string, *actions.ExecutionError) {
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(req.Principal.TenantID)
	}
	if tenantID == "" {
		return "", actions.NewExecutionError(actions.ErrUnauthorized, "tenant context is required", 403, false, nil)
	}
	return tenantID, nil
}

func (r *Runtime) resolveAction(ctx context.Context, req actions.ExecutionRequest) (actions.ActionDefinition, *actions.ExecutionError) {
	if strings.TrimSpace(req.Action.Name) != "" {
		if r.ActionRegistry != nil {
			resolved, err := r.ActionRegistry.Lookup(ctx, req.Action.Name)
			if err != nil {
				if errors.Is(err, actions.ErrLookupNotFound) {
					return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrActionNotFound, "action not found", 404, false, map[string]any{"action": req.Action.Name})
				}
				return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "action lookup failed", 500, true, map[string]any{"action": req.Action.Name})
			}
			if resolved != nil {
				return *resolved, nil
			}
		}
		return req.Action, nil
	}

	if req.Classification == nil || strings.TrimSpace(req.Classification.ActionName) == "" {
		return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrClassificationFailed, "action resolution failed", 400, false, nil)
	}
	if r.ActionRegistry == nil {
		return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrActionNotFound, "action registry unavailable", 500, false, nil)
	}
	resolved, err := r.ActionRegistry.Lookup(ctx, req.Classification.ActionName)
	if err != nil {
		if errors.Is(err, actions.ErrLookupNotFound) {
			return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrActionNotFound, "action not found", 404, false, map[string]any{"action": req.Classification.ActionName})
		}
		return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "action resolution failed", 500, true, map[string]any{"action": req.Classification.ActionName})
	}
	if resolved == nil {
		return actions.ActionDefinition{}, actions.NewExecutionError(actions.ErrActionNotFound, "action not found", 404, false, map[string]any{"action": req.Classification.ActionName})
	}
	return *resolved, nil
}

func (r *Runtime) getAdapter(adapterType string) (adapters.Adapter, *actions.ExecutionError) {
	kind := strings.TrimSpace(adapterType)
	if kind == "" {
		kind = "http"
	}
	if r.Adapters == nil {
		return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "adapter registry unavailable", 500, false, nil)
	}
	adapter, ok := r.Adapters[kind]
	if !ok || adapter == nil {
		return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, fmt.Sprintf("adapter %q not found", kind), 500, false, nil)
	}
	return adapter, nil
}

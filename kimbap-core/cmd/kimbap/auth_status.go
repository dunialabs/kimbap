package main

import (
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/connectors"
	"github.com/spf13/cobra"
)

func newAuthListCommand() *cobra.Command {
	var tenant string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List OAuth connection states",
		RunE: func(_ *cobra.Command, _ []string) error {
			activeTenant := connectorTenant(tenant)
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			store, storeErr := openConnectorStore(cfg)
			if storeErr != nil {
				_ = printOutput(map[string]any{
					"status":    "not_configured",
					"operation": "auth.list",
					"tenant_id": activeTenant,
					"message":   fmt.Sprintf("OAuth state store unavailable: %v", storeErr),
				})
				return fmt.Errorf("OAuth state store unavailable: %w", storeErr)
			}
			mgr := connectors.NewManager(store)
			states, listErr := mgr.List(contextBackground(), activeTenant)
			if listErr != nil {
				_ = printOutput(map[string]any{
					"status":    "not_configured",
					"operation": "auth.list",
					"tenant_id": activeTenant,
					"message":   fmt.Sprintf("OAuth state store unavailable: %v", listErr),
				})
				return fmt.Errorf("OAuth state store unavailable: %w", listErr)
			}

			items := make([]map[string]any, 0, len(states))
			for _, state := range states {
				cs := statusFromSanitizedState(&state)
				scopeLevel := string(state.ConnectionScope)
				if scopeLevel == "" {
					scopeLevel = string(connectors.ScopeUser)
				}
				items = append(items, map[string]any{
					"provider":       state.Provider,
					"connection_id":  state.Name,
					"scope_level":    scopeLevel,
					"status":         cs,
					"expires_at":     state.ExpiresAt,
					"refresh_health": refreshHealthFromConnectionStatus(cs),
					"last_used_at":   state.LastUsedAt,
				})
			}

			return printOutput(map[string]any{
				"status":      "ok",
				"operation":   "auth.list",
				"tenant_id":   activeTenant,
				"count":       len(items),
				"connections": items,
			})
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func newAuthStatusCommand() *cobra.Command {
	var tenant string
	cmd := &cobra.Command{
		Use:   "status [provider]",
		Short: "Show OAuth status details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			activeTenant := connectorTenant(tenant)
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			store, storeErr := openConnectorStore(cfg)
			if storeErr != nil {
				_ = printOutput(map[string]any{
					"status":    "not_configured",
					"operation": "auth.status",
					"tenant_id": activeTenant,
					"message":   fmt.Sprintf("OAuth state store unavailable: %v", storeErr),
				})
				return fmt.Errorf("OAuth state store unavailable: %w", storeErr)
			}
			mgr := connectors.NewManager(store)

			if len(args) == 0 {
				states, listErr := mgr.List(contextBackground(), activeTenant)
				if listErr != nil {
					_ = printOutput(map[string]any{
						"status":    "not_configured",
						"operation": "auth.status",
						"tenant_id": activeTenant,
						"message":   fmt.Sprintf("OAuth state store unavailable: %v", listErr),
					})
					return fmt.Errorf("OAuth state store unavailable: %w", listErr)
				}

				items := make([]map[string]any, 0, len(states))
				for _, state := range states {
					cs := statusFromSanitizedState(&state)
					scopeLevel := string(state.ConnectionScope)
					if scopeLevel == "" {
						scopeLevel = string(connectors.ScopeUser)
					}
					items = append(items, map[string]any{
						"provider":            state.Provider,
						"connection_scope":    scopeLevel,
						"connected_principal": state.ConnectedPrincipal,
						"granted_scopes":      state.Scopes,
						"expiry":              state.ExpiresAt,
						"refresh_state":       refreshStateFromConnectionStatus(cs),
						"last_refresh":        state.LastRefresh,
						"last_used_at":        state.LastUsedAt,
						"status":              cs,
						"remediation":         remediationFromStatus(cs),
					})
				}

				return printOutput(map[string]any{
					"status":      "ok",
					"operation":   "auth.status",
					"tenant_id":   activeTenant,
					"count":       len(items),
					"connections": items,
				})
			}

			providerID := strings.TrimSpace(args[0])
			if providerID == "" {
				return fmt.Errorf("provider is required")
			}

			state, statusErr := mgr.Status(contextBackground(), activeTenant, providerID)
			if statusErr != nil {
				_ = printOutput(map[string]any{
					"status":    "not_found",
					"operation": "auth.status",
					"tenant_id": activeTenant,
					"provider":  providerID,
					"message":   statusErr.Error(),
				})
				return fmt.Errorf("provider %q not found: %w", providerID, statusErr)
			}

			cs := statusFromSanitizedState(state)
			scopeLevel := string(state.ConnectionScope)
			if scopeLevel == "" {
				scopeLevel = string(connectors.ScopeUser)
			}
			return printOutput(map[string]any{
				"status":              "ok",
				"operation":           "auth.status",
				"tenant_id":           activeTenant,
				"provider":            state.Provider,
				"connection_scope":    scopeLevel,
				"connected_principal": state.ConnectedPrincipal,
				"granted_scopes":      state.Scopes,
				"expiry":              state.ExpiresAt,
				"refresh_state":       refreshStateFromConnectionStatus(cs),
				"last_refresh":        state.LastRefresh,
				"last_used_at":        state.LastUsedAt,
				"status_detail":       cs,
				"remediation":         remediationFromStatus(cs),
			})
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

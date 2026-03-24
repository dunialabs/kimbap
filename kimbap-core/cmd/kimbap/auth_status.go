package main

import (
	"fmt"
	"os"
	"strings"
	"time"

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

			if !outputAsJSON() {
				if len(items) == 0 {
					_, _ = fmt.Fprintln(os.Stdout, "No OAuth connections found.")
					return nil
				}
				_, _ = fmt.Fprintf(os.Stdout, "OAuth Connections (%d):\n\n", len(items))
				for _, item := range items {
					provider, _ := item["provider"].(string)
					status, _ := item["status"].(connectors.ConnectionStatus)
					scopeLevel, _ := item["scope_level"].(string)
					refreshHealth, _ := item["refresh_health"].(string)
					_, _ = fmt.Fprintf(os.Stdout, "  %-15s  status=%-20s  scope=%-10s  refresh=%s\n", provider, status, scopeLevel, refreshHealth)
				}
				return nil
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

				if !outputAsJSON() {
					if len(items) == 0 {
						_, _ = fmt.Fprintln(os.Stdout, "No OAuth connections found.")
						return nil
					}
					for _, item := range items {
						provider, _ := item["provider"].(string)
						status, _ := item["status"].(connectors.ConnectionStatus)
						scopeLevel, _ := item["connection_scope"].(string)
						principal, _ := item["connected_principal"].(string)
						_, _ = fmt.Fprintf(os.Stdout, "%-15s  status=%-20s  scope=%-10s", provider, status, scopeLevel)
						if principal != "" {
							_, _ = fmt.Fprintf(os.Stdout, "  user=%s", principal)
						}
						_, _ = fmt.Fprintln(os.Stdout)
					}
					return nil
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
			if p, pErr := providers.GetProvider(providerID); pErr == nil {
				providerID = p.ID
			}

			state, statusErr := mgr.Status(contextBackground(), activeTenant, providerID)
			if statusErr != nil {
				if strings.Contains(statusErr.Error(), "not found") {
					_ = printOutput(map[string]any{
						"status":    "not_connected",
						"operation": "auth.status",
						"tenant_id": activeTenant,
						"provider":  providerID,
						"message":   fmt.Sprintf("No connection found for %q. Run: kimbap auth connect %s", providerID, providerID),
					})
					return fmt.Errorf("no connection found for %q: %w", providerID, statusErr)
				}
				_ = printOutput(map[string]any{
					"status":    "error",
					"operation": "auth.status",
					"tenant_id": activeTenant,
					"provider":  providerID,
					"message":   fmt.Sprintf("failed to retrieve status: %v", statusErr),
				})
				return fmt.Errorf("failed to get provider %q status: %w", providerID, statusErr)
			}

			cs := statusFromSanitizedState(state)
			scopeLevel := string(state.ConnectionScope)
			if scopeLevel == "" {
				scopeLevel = string(connectors.ScopeUser)
			}
			if !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Provider:            %s\n", state.Provider)
				_, _ = fmt.Fprintf(os.Stdout, "Connection scope:    %s\n", scopeLevel)
				if state.ConnectedPrincipal != "" {
					_, _ = fmt.Fprintf(os.Stdout, "Connected as:        %s\n", state.ConnectedPrincipal)
				}
				_, _ = fmt.Fprintf(os.Stdout, "Status:              %s\n", cs)
				if len(state.Scopes) > 0 {
					_, _ = fmt.Fprintf(os.Stdout, "Granted scopes:      %s\n", strings.Join(state.Scopes, ", "))
				}
				_, _ = fmt.Fprintln(os.Stdout, "Access token:        managed by core")
				_, _ = fmt.Fprintf(os.Stdout, "Refresh status:      %s\n", refreshStateFromConnectionStatus(cs))
				if state.LastRefresh != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Last refresh:        %s\n", state.LastRefresh.Format(time.RFC3339))
				}
				if state.LastUsedAt != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Last used:           %s\n", state.LastUsedAt.Format(time.RFC3339))
				}
				if rem := remediationFromStatus(cs); rem != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Remediation:         %s\n", rem)
				}
				return nil
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

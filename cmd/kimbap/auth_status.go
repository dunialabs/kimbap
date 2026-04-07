package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/spf13/cobra"
)

func newAuthListCommand() *cobra.Command {
	var tenant string
	var workspace string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List OAuth connection states",
		RunE: func(_ *cobra.Command, _ []string) error {
			activeTenant := connectorTenant(tenant)
			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}

			store, storeErr := openConnectorStoreReadOnly(cfg)
			if storeErr != nil {
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "not_configured",
						"operation": "auth.list",
						"tenant_id": activeTenant,
						"message":   unavailableMessage(componentOAuthStateStore, storeErr),
					})
				}
				return unavailableError(componentOAuthStateStore, storeErr)
			}
			defer closeConnectorStoreIfPossible(store)
			mgr := connectors.NewManager(store)
			states, listErr := mgr.List(contextBackground(), activeTenant)
			if listErr != nil {
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "not_configured",
						"operation": "auth.list",
						"tenant_id": activeTenant,
						"message":   unavailableMessage(componentOAuthStateStore, listErr),
					})
				}
				return unavailableError(componentOAuthStateStore, listErr)
			}

			filtered := filterStatesByWorkspace(states, workspace)
			items := make([]map[string]any, 0, len(filtered))
			for _, state := range filtered {
				items = append(items, authListItem(state))
			}

			if !outputAsJSON() {
				if len(items) == 0 {
					_, _ = fmt.Fprintln(os.Stdout, "No OAuth connections found.")
					_, _ = fmt.Fprintln(os.Stdout, "Run 'kimbap auth connect <provider>' to connect an OAuth service.")
					return nil
				}
				_, _ = fmt.Fprintf(os.Stdout, "OAuth Connections (%d):\n\n", len(items))
				for _, item := range items {
					provider, _ := item["provider"].(string)
					connectionID, _ := item["connection_id"].(string)
					display := provider
					if connectionID != "" && connectionID != provider {
						display = fmt.Sprintf("%s (%s)", provider, connectionID)
					}
					status, _ := item["status_detail"].(connectors.ConnectionStatus)
					scopeLevel, _ := item["connection_scope"].(string)
					refreshHealth, _ := item["refresh_health"].(string)
					principal, _ := item["connected_principal"].(string)
					if principal == "" {
						principal = "-"
					}
					_, _ = fmt.Fprintf(os.Stdout, "  %-15s  status=%-20s  scope=%-10s  refresh=%-12s  principal=%s\n", display, status, scopeLevel, refreshHealth, principal)
				}
				_, _ = fmt.Fprintln(os.Stdout)
				_, _ = fmt.Fprintln(os.Stdout, "Run 'kimbap auth status <provider>' for connection details.")
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
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace id filter (applies to workspace-scoped connections)")
	return cmd
}

func newAuthStatusCommand() *cobra.Command {
	var tenant string
	var profile string
	var workspace string
	cmd := &cobra.Command{
		Use:   "status [provider]",
		Short: "Show OAuth status details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			activeTenant := connectorTenant(tenant)
			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}

			store, storeErr := openConnectorStoreReadOnly(cfg)
			if storeErr != nil {
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "not_configured",
						"operation": "auth.status",
						"tenant_id": activeTenant,
						"message":   unavailableMessage(componentOAuthStateStore, storeErr),
					})
				}
				return unavailableError(componentOAuthStateStore, storeErr)
			}
			defer closeConnectorStoreIfPossible(store)
			mgr := connectors.NewManager(store)

			if len(args) == 0 {
				states, listErr := mgr.List(contextBackground(), activeTenant)
				if listErr != nil {
					if outputAsJSON() {
						_ = printOutput(map[string]any{
							"status":    "not_configured",
							"operation": "auth.status",
							"tenant_id": activeTenant,
							"message":   unavailableMessage(componentOAuthStateStore, listErr),
						})
					}
					return unavailableError(componentOAuthStateStore, listErr)
				}

				filtered := filterStatesByWorkspace(states, workspace)
				items := make([]map[string]any, 0, len(filtered))
				for _, state := range filtered {
					items = append(items, authStatusItem(state))
				}

				if !outputAsJSON() {
					if len(items) == 0 {
						_, _ = fmt.Fprintln(os.Stdout, "No OAuth connections found.")
						_, _ = fmt.Fprintln(os.Stdout, "Run 'kimbap auth connect <provider>' to connect an OAuth service.")
						return nil
					}
					for _, item := range items {
						provider, _ := item["provider"].(string)
						connectionID, _ := item["connection_id"].(string)
						display := provider
						if connectionID != "" && connectionID != provider {
							display = fmt.Sprintf("%s (%s)", provider, connectionID)
						}
						status, _ := item["status_detail"].(connectors.ConnectionStatus)
						scopeLevel, _ := item["connection_scope"].(string)
						principal, _ := item["connected_principal"].(string)
						refreshHealth, _ := item["refresh_health"].(string)
						if principal == "" {
							principal = "-"
						}
						_, _ = fmt.Fprintf(os.Stdout, "%-15s  status=%-20s  scope=%-10s  refresh=%-12s  principal=%s\n", display, status, scopeLevel, refreshHealth, principal)
					}
					_, _ = fmt.Fprintln(os.Stdout)
					_, _ = fmt.Fprintln(os.Stdout, "Run 'kimbap auth reconnect <provider>' to refresh a connection.")
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
			storeName := connectorStoreName(providerID, profile)

			state, statusErr := mgr.Status(contextBackground(), activeTenant, storeName)
			if statusErr != nil {
				if errors.Is(statusErr, connectors.ErrConnectorNotFound) {
					msg := fmt.Sprintf("No connection found for %q. Run: kimbap auth connect %s", providerID, providerID)
					if !outputAsJSON() {
						_, _ = fmt.Fprintln(os.Stdout, msg)
						return fmt.Errorf("no connection found for %q: %w", providerID, statusErr)
					}
					_ = printOutput(map[string]any{
						"status":    "not_connected",
						"operation": "auth.status",
						"tenant_id": activeTenant,
						"provider":  providerID,
						"message":   msg,
					})
					return fmt.Errorf("no connection found for %q: %w", providerID, statusErr)
				}
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "error",
						"operation": "auth.status",
						"tenant_id": activeTenant,
						"provider":  providerID,
						"message":   fmt.Sprintf("failed to retrieve status: %v", statusErr),
					})
				}
				return fmt.Errorf("failed to get provider %q status: %w", providerID, statusErr)
			}

			cs := statusFromSanitizedState(state)
			if strings.TrimSpace(workspace) != "" {
				ws := strings.TrimSpace(workspace)
				if state.ConnectionScope != connectors.ScopeWorkspace || strings.TrimSpace(state.WorkspaceID) != ws {
					msg := fmt.Sprintf("No workspace-scoped connection found for %q in workspace %q.", providerID, workspace)
					if !outputAsJSON() {
						_, _ = fmt.Fprintln(os.Stdout, msg)
						return fmt.Errorf("workspace-scoped connection mismatch for %q", providerID)
					}
					_ = printOutput(map[string]any{
						"status":    "not_connected",
						"operation": "auth.status",
						"tenant_id": activeTenant,
						"provider":  providerID,
						"workspace": workspace,
						"message":   msg,
					})
					return fmt.Errorf("workspace-scoped connection mismatch for %q", providerID)
				}
			}
			scopeLevel := string(state.ConnectionScope)
			if scopeLevel == "" {
				scopeLevel = string(connectors.ScopeUser)
			}
			if !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Provider:            %s\n", state.Provider)
				if storeName != state.Provider {
					_, _ = fmt.Fprintf(os.Stdout, "Connection id:       %s\n", storeName)
				}
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
				if strings.TrimSpace(state.LastRefreshError) != "" {
					_, _ = fmt.Fprintf(os.Stdout, "Last refresh result: failed (%s)\n", state.LastRefreshError)
				} else if state.LastRefresh != nil {
					_, _ = fmt.Fprintln(os.Stdout, "Last refresh result: success")
				}
				if state.LastRefresh != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Last refresh:        %s\n", state.LastRefresh.Format("2006-01-02 15:04"))
				}
				if state.LastUsedAt != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Last used:           %s\n", state.LastUsedAt.Format("2006-01-02 15:04"))
				}
				if state.RevokedAt != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Revocation state:    revoked at %s\n", state.RevokedAt.Format("2006-01-02 15:04"))
				} else {
					_, _ = fmt.Fprintln(os.Stdout, "Revocation state:    active")
				}
				if rem := remediationFromStatus(cs); rem != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Remediation:         %s\n", rem)
				}
				return nil
			}
			return printOutput(authSingleStatusPayload(activeTenant, *state))
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	cmd.Flags().StringVar(&profile, "profile", "default", "connection profile name for multiple accounts per provider")
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace id filter (applies to workspace-scoped connections)")
	return cmd
}

func filterStatesByWorkspace(states []connectors.ConnectorState, workspace string) []connectors.ConnectorState {
	ws := strings.TrimSpace(workspace)
	if ws == "" {
		return states
	}
	filtered := make([]connectors.ConnectorState, 0, len(states))
	for _, state := range states {
		if state.ConnectionScope != connectors.ScopeWorkspace {
			continue
		}
		if strings.TrimSpace(state.WorkspaceID) == ws {
			filtered = append(filtered, state)
		}
	}
	return filtered
}

func authScopeLevel(state connectors.ConnectorState) string {
	scopeLevel := string(state.ConnectionScope)
	if scopeLevel == "" {
		return string(connectors.ScopeUser)
	}
	return scopeLevel
}

func authLastRefreshResult(state connectors.ConnectorState) string {
	if strings.TrimSpace(state.LastRefreshError) != "" {
		return "failed"
	}
	if state.LastRefresh != nil {
		return "success"
	}
	return "unknown"
}

func authRevocationState(state connectors.ConnectorState) string {
	if state.RevokedAt != nil {
		return "revoked"
	}
	return "active"
}

func authListItem(state connectors.ConnectorState) map[string]any {
	cs := statusFromSanitizedState(&state)
	scopeLevel := authScopeLevel(state)
	return map[string]any{
		"provider":            state.Provider,
		"connection_id":       state.Name,
		"connection_scope":    scopeLevel,
		"scope_level":         scopeLevel,
		"connected_principal": state.ConnectedPrincipal,
		"status_detail":       cs,
		"status":              cs,
		"expires_at":          state.ExpiresAt,
		"refresh_state":       refreshStateFromConnectionStatus(cs),
		"refresh_health":      refreshHealthFromConnectionStatus(cs),
		"last_refresh_result": authLastRefreshResult(state),
		"revocation_state":    authRevocationState(state),
		"last_used_at":        state.LastUsedAt,
	}
}

func authStatusItem(state connectors.ConnectorState) map[string]any {
	cs := statusFromSanitizedState(&state)
	scopeLevel := authScopeLevel(state)
	return map[string]any{
		"provider":            state.Provider,
		"connection_id":       state.Name,
		"connection_scope":    scopeLevel,
		"scope_level":         scopeLevel,
		"connected_principal": state.ConnectedPrincipal,
		"granted_scopes":      state.Scopes,
		"expires_at":          state.ExpiresAt,
		"refresh_state":       refreshStateFromConnectionStatus(cs),
		"refresh_health":      refreshHealthFromConnectionStatus(cs),
		"last_refresh_result": authLastRefreshResult(state),
		"last_refresh":        state.LastRefresh,
		"last_used_at":        state.LastUsedAt,
		"revocation_state":    authRevocationState(state),
		"status_detail":       cs,
		"status":              cs,
		"remediation":         remediationFromStatus(cs),
	}
}

func authSingleStatusPayload(tenantID string, state connectors.ConnectorState) map[string]any {
	item := authStatusItem(state)
	connectionStatus := item["status"]
	item["status"] = "ok"
	item["operation"] = "auth.status"
	item["tenant_id"] = tenantID
	item["connection_status"] = connectionStatus
	item["provider"] = state.Provider
	return item
}

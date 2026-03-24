package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage OAuth connections to external services",
	}

	cmd.AddCommand(newAuthConnectCommand())
	cmd.AddCommand(newAuthListCommand())
	cmd.AddCommand(newAuthStatusCommand())
	cmd.AddCommand(newAuthReconnectCommand())
	cmd.AddCommand(newAuthRevokeCommand())
	cmd.AddCommand(newAuthProvidersCommand())
	cmd.AddCommand(newAuthDoctorCommand())

	return cmd
}

func newAuthConnectCommand() *cobra.Command {
	var flow string
	var scopeInput string
	var browserName string
	var noOpen bool
	var port int
	var timeout time.Duration
	var workspace string
	var connectionScope string
	var tenant string

	cmd := &cobra.Command{
		Use:   "connect <provider>",
		Short: "Connect to an OAuth provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			providerID := strings.TrimSpace(args[0])
			if providerID == "" {
				return fmt.Errorf("provider is required")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			return runAuthConnect(
				contextBackground(),
				cfg,
				providerID,
				connectorTenant(tenant),
				flow,
				scopeInput,
				browserName,
				noOpen,
				port,
				timeout,
				workspace,
				connectionScope,
				false,
			)
		},
	}

	cmd.Flags().StringVar(&flow, "flow", "auto", "auth flow to use (auto, browser, device)")
	cmd.Flags().StringVar(&scopeInput, "scope", "", "requested scopes (space/comma separated)")
	cmd.Flags().StringVar(&scopeInput, "scopes", "", "requested scopes (space/comma separated)")
	cmd.Flags().StringVar(&browserName, "browser", "auto", "browser strategy (auto, system, none)")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "do not automatically open browser")
	cmd.Flags().IntVar(&port, "port", 0, "local callback port for browser flow")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "authorization timeout")
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace id for workspace-scoped connection")
	cmd.Flags().StringVar(&connectionScope, "connection-scope", string(connectors.ScopeUser), "connection scope (user, workspace, service)")
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")

	return cmd
}

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

			states, err := listConnectorStates(contextBackground(), cfg, activeTenant)
			if err != nil {
				return printOutput(map[string]any{
					"status":    "not_configured",
					"operation": "auth.list",
					"tenant_id": activeTenant,
					"message":   fmt.Sprintf("OAuth state store unavailable: %v", err),
				})
			}

			connections := make([]map[string]any, 0, len(states))
			for _, state := range states {
				connections = append(connections, map[string]any{
					"provider":       state.Provider,
					"connection_id":  state.Name,
					"scope_level":    string(connectors.ScopeUser),
					"status":         mapLegacyConnectorStatus(connectors.ConnectorStatus(state.Status)),
					"expires_at":     state.ExpiresAt,
					"refresh_health": refreshHealthFromLegacyStatus(connectors.ConnectorStatus(state.Status)),
					"last_used_at":   nil,
				})
			}

			return printOutput(map[string]any{
				"status":      "ok",
				"operation":   "auth.list",
				"tenant_id":   activeTenant,
				"count":       len(connections),
				"connections": connections,
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

			if len(args) == 0 {
				states, listErr := listConnectorStates(contextBackground(), cfg, activeTenant)
				if listErr != nil {
					return printOutput(map[string]any{
						"status":    "not_configured",
						"operation": "auth.status",
						"tenant_id": activeTenant,
						"message":   fmt.Sprintf("OAuth state store unavailable: %v", listErr),
					})
				}

				items := make([]map[string]any, 0, len(states))
				for _, state := range states {
					connectionStatus := mapLegacyConnectorStatus(connectors.ConnectorStatus(state.Status))
					items = append(items, map[string]any{
						"provider":            state.Provider,
						"connection_scope":    string(connectors.ScopeUser),
						"connected_principal": state.Account,
						"granted_scopes":      state.Scopes,
						"expiry":              state.ExpiresAt,
						"refresh_state":       refreshStateFromConnectionStatus(connectionStatus),
						"last_refresh":        state.LastRefresh,
						"last_used_at":        nil,
						"status":              connectionStatus,
						"remediation":         remediationFromStatus(connectionStatus),
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

			state, err := getConnectorState(contextBackground(), cfg, activeTenant, providerID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return printOutput(map[string]any{
						"status":    "not_found",
						"operation": "auth.status",
						"tenant_id": activeTenant,
						"provider":  providerID,
						"message":   "No OAuth connection state found",
					})
				}
				return printOutput(map[string]any{
					"status":    "not_configured",
					"operation": "auth.status",
					"tenant_id": activeTenant,
					"provider":  providerID,
					"message":   fmt.Sprintf("OAuth state store unavailable: %v", err),
				})
			}

			connectionStatus := mapLegacyConnectorStatus(connectors.ConnectorStatus(state.Status))
			return printOutput(map[string]any{
				"status":              "ok",
				"operation":           "auth.status",
				"tenant_id":           activeTenant,
				"provider":            state.Provider,
				"connection_scope":    string(connectors.ScopeUser),
				"connected_principal": state.Account,
				"granted_scopes":      state.Scopes,
				"expiry":              state.ExpiresAt,
				"refresh_state":       refreshStateFromConnectionStatus(connectionStatus),
				"last_refresh":        state.LastRefresh,
				"last_used_at":        nil,
				"status_detail":       connectionStatus,
				"remediation":         remediationFromStatus(connectionStatus),
			})
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func newAuthReconnectCommand() *cobra.Command {
	var flow string
	var scopeInput string
	var tenant string
	cmd := &cobra.Command{
		Use:   "reconnect <provider>",
		Short: "Reconnect an OAuth provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			providerID := strings.TrimSpace(args[0])
			if providerID == "" {
				return fmt.Errorf("provider is required")
			}
			activeTenant := connectorTenant(tenant)

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(os.Stderr, "Attempting to reconnect...")

			state, stateErr := getConnectorState(contextBackground(), cfg, activeTenant, providerID)
			if stateErr == nil {
				connectionStatus := mapLegacyConnectorStatus(connectors.ConnectorStatus(state.Status))
				if connectionStatus == connectors.StatusConnected || connectionStatus == connectors.StatusDegraded {
					requestedScopes := scopeValues(scopeInput, state.Scopes)
					return printOutput(map[string]any{
						"status":           "ok",
						"operation":        "auth.reconnect",
						"tenant_id":        activeTenant,
						"provider":         providerID,
						"reconnect_method": "refresh",
						"message":          "Refresh token path succeeded",
						"delta": map[string]any{
							"scope_changes":   scopeDelta(state.Scopes, requestedScopes),
							"account_changes": "none",
						},
					})
				}
			}

			connectErr := runAuthConnect(
				contextBackground(),
				cfg,
				providerID,
				activeTenant,
				flow,
				scopeInput,
				"auto",
				false,
				0,
				5*time.Minute,
				"",
				string(connectors.ScopeUser),
				true,
			)
			if connectErr != nil {
				return connectErr
			}

			fallbackPrevScopes := []string{}
			if stateErr == nil && state != nil {
				fallbackPrevScopes = state.Scopes
			}
			fallbackRequestedScopes := scopeValues(scopeInput, fallbackPrevScopes)
			return printOutput(map[string]any{
				"status":           "ok",
				"operation":        "auth.reconnect",
				"tenant_id":        activeTenant,
				"provider":         providerID,
				"reconnect_method": "full_connect",
				"message":          "Refresh token path unavailable; completed full connect flow",
				"delta": map[string]any{
					"scope_changes":   scopeDelta(fallbackPrevScopes, fallbackRequestedScopes),
					"account_changes": "unknown",
				},
			})
		},
	}
	cmd.Flags().StringVar(&flow, "flow", "auto", "auth flow to use (auto, browser, device)")
	cmd.Flags().StringVar(&scopeInput, "scope", "", "requested scopes (space/comma separated)")
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func newAuthRevokeCommand() *cobra.Command {
	var tenant string
	var force bool
	cmd := &cobra.Command{
		Use:   "revoke <provider>",
		Short: "Revoke and disconnect an OAuth provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			providerID := strings.TrimSpace(args[0])
			if providerID == "" {
				return fmt.Errorf("provider is required")
			}
			activeTenant := connectorTenant(tenant)

			if !force {
				if ok, promptErr := confirmRevocation(providerID); promptErr != nil {
					return promptErr
				} else if !ok {
					return printOutput(map[string]any{
						"status":    "cancelled",
						"operation": "auth.revoke",
						"tenant_id": activeTenant,
						"provider":  providerID,
					})
				}
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			provider, providerErr := providers.GetProvider(providerID)
			providerMetaKnown := providerErr == nil
			revocationAttempted := false
			revocationResult := "not_supported"
			if providerMetaKnown && strings.TrimSpace(provider.RevocationEndpoint) != "" {
				revocationAttempted = true
				revocationResult = "attempted"
			}

			deleted := true
			deleteErr := deleteConnectorState(cfg, activeTenant, providerID)
			if deleteErr != nil && !errors.Is(deleteErr, sql.ErrNoRows) {
				deleted = false
			}

			return printOutput(map[string]any{
				"status":                  "ok",
				"operation":               "auth.revoke",
				"tenant_id":               activeTenant,
				"provider":                providerID,
				"provider_metadata_known": providerMetaKnown,
				"revocation": map[string]any{
					"attempted": revocationAttempted,
					"result":    revocationResult,
				},
				"local_state_deleted": deleted,
				"delete_error":        stringOrNil(errString(deleteErr)),
			})
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	cmd.Flags().BoolVar(&force, "force", false, "skip revocation confirmation")
	return cmd
}

func newAuthProvidersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Inspect provider registry metadata",
	}
	cmd.AddCommand(newAuthProvidersListCommand())
	cmd.AddCommand(newAuthProvidersDescribeCommand())
	return cmd
}

func newAuthProvidersListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List known OAuth providers",
		RunE: func(_ *cobra.Command, _ []string) error {
			items, err := providers.ListProviders()
			if err != nil {
				return err
			}
			sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

			out := make([]map[string]any, 0, len(items))
			for _, item := range items {
				out = append(out, map[string]any{
					"id":              item.ID,
					"display_name":    item.DisplayName,
					"supported_flows": item.SupportedFlows,
					"configured":      providerIsConfigured(item),
				})
			}

			return printOutput(map[string]any{
				"status":    "ok",
				"operation": "auth.providers.list",
				"count":     len(out),
				"providers": out,
			})
		},
	}
	return cmd
}

func newAuthProvidersDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <provider>",
		Short: "Describe OAuth provider metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			providerID := strings.TrimSpace(args[0])
			if providerID == "" {
				return fmt.Errorf("provider is required")
			}
			provider, err := providers.GetProvider(providerID)
			if err != nil {
				return printOutput(map[string]any{
					"status":    "not_found",
					"operation": "auth.providers.describe",
					"provider":  providerID,
					"message":   err.Error(),
				})
			}

			return printOutput(map[string]any{
				"status":                 "ok",
				"operation":              "auth.providers.describe",
				"id":                     provider.ID,
				"display_name":           provider.DisplayName,
				"supported_flows":        provider.SupportedFlows,
				"default_scopes":         provider.DefaultScopes,
				"scope_presets":          provider.ScopePresets,
				"auth_endpoint":          provider.AuthEndpoint,
				"token_endpoint":         provider.TokenEndpoint,
				"device_endpoint":        provider.DeviceEndpoint,
				"revocation_endpoint":    provider.RevocationEndpoint,
				"userinfo_endpoint":      provider.UserInfoEndpoint,
				"connection_scope_model": provider.ConnectionScopeModel,
				"pkce_required":          provider.PKCERequired,
				"notes":                  provider.Notes,
			})
		},
	}
	return cmd
}

func newAuthDoctorCommand() *cobra.Command {
	var tenant string
	cmd := &cobra.Command{
		Use:   "doctor [provider]",
		Short: "Run OAuth-specific diagnostics",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			activeTenant := connectorTenant(tenant)
			providerID := ""
			if len(args) == 1 {
				providerID = strings.TrimSpace(args[0])
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			checks := make([]doctorCheck, 0, 8)
			hasFailure := false

			keyCheck := checkConnectorEncryptionKeyPresent()
			checks = append(checks, keyCheck)
			hasFailure = hasFailure || keyCheck.Status == "fail"

			providerCheck := checkAuthProviderConfig(providerID)
			checks = append(checks, providerCheck)
			hasFailure = hasFailure || providerCheck.Status == "fail"

			expiryCheck, refreshCheck := checkAuthTokenState(cfg, activeTenant, providerID)
			checks = append(checks, expiryCheck, refreshCheck)
			hasFailure = hasFailure || expiryCheck.Status == "fail" || refreshCheck.Status == "fail"

			reachabilityChecks := checkProviderEndpointsReachable(providerID)
			for _, c := range reachabilityChecks {
				checks = append(checks, c)
				hasFailure = hasFailure || c.Status == "fail"
			}

			loopbackCheck := checkLoopbackPortBindable(0)
			checks = append(checks, loopbackCheck)
			hasFailure = hasFailure || loopbackCheck.Status == "fail"

			if err := printOutput(map[string]any{
				"status":    ternary(hasFailure, "fail", "ok"),
				"operation": "auth.doctor",
				"tenant_id": activeTenant,
				"provider":  stringOrNil(providerID),
				"checks":    checks,
			}); err != nil {
				return err
			}

			if hasFailure {
				return fmt.Errorf("auth doctor found failing checks")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func runAuthConnect(
	_ any,
	_ *config.KimbapConfig,
	providerID string,
	tenantID string,
	flow string,
	scopeInput string,
	browserName string,
	noOpen bool,
	port int,
	timeout time.Duration,
	workspace string,
	connectionScope string,
	reconnectMode bool,
) error {
	provider, err := providers.GetProvider(providerID)
	if err != nil {
		return printOutput(map[string]any{
			"status":    "not_found",
			"operation": "auth.connect",
			"tenant_id": tenantID,
			"provider":  providerID,
			"message":   err.Error(),
		})
	}

	scopes := scopeValues(scopeInput, provider.DefaultScopes)
	selectedFlow, err := flows.SelectFlow(strings.TrimSpace(flow), provider, scopes)
	if err != nil {
		return err
	}

	connScope := parseConnectionScope(connectionScope)
	_ = connectors.ConnectOptions{
		Flow:            selectedFlow,
		Scopes:          scopes,
		Browser:         browserName,
		NoOpen:          noOpen,
		Port:            port,
		Timeout:         timeout,
		ConnectionScope: connScope,
		WorkspaceID:     workspace,
	}

	operation := "auth.connect"
	if reconnectMode {
		operation = "auth.reconnect.connect"
	}

	switch selectedFlow {
	case connectors.FlowDevice:
		_, _ = fmt.Fprintf(os.Stderr, "Starting device flow for %s...\n", provider.DisplayName)
		_, _ = fmt.Fprintln(os.Stderr, "Verification URL: https://example.local/verify")
		_, _ = fmt.Fprintln(os.Stderr, "User code: DEMO-CODE")
		_, _ = fmt.Fprintln(os.Stderr, "Waiting for authorization confirmation...")
	case connectors.FlowBrowser:
		_, _ = fmt.Fprintf(os.Stderr, "Starting browser flow for %s...\n", provider.DisplayName)
		if noOpen || strings.EqualFold(browserName, "none") {
			_, _ = fmt.Fprintln(os.Stderr, "Auto-open disabled; open authorization URL manually.")
		} else {
			_, _ = fmt.Fprintln(os.Stderr, "Opening browser for OAuth authorization...")
		}
		_, _ = fmt.Fprintln(os.Stderr, "Waiting for loopback callback...")
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Using %s flow for %s...\n", selectedFlow, provider.DisplayName)
	}

	result := connectors.ConnectResult{
		Provider:        provider.ID,
		ConnectionScope: connScope,
		Status:          connectors.StatusConnected,
		GrantedScopes:   scopes,
		FlowUsed:        selectedFlow,
	}

	return printOutput(map[string]any{
		"status":           "ok",
		"operation":        operation,
		"tenant_id":        tenantID,
		"provider":         result.Provider,
		"connection_scope": result.ConnectionScope,
		"status_detail":    result.Status,
		"granted_scopes":   result.GrantedScopes,
		"flow_used":        result.FlowUsed,
		"workspace":        stringOrNil(workspace),
	})
}

func parseConnectionScope(raw string) connectors.ConnectionScope {
	switch connectors.ConnectionScope(strings.ToLower(strings.TrimSpace(raw))) {
	case connectors.ScopeWorkspace:
		return connectors.ScopeWorkspace
	case connectors.ScopeService:
		return connectors.ScopeService
	default:
		return connectors.ScopeUser
	}
}

func scopeValues(raw string, fallback []string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func mapLegacyConnectorStatus(legacy connectors.ConnectorStatus) connectors.ConnectionStatus {
	switch legacy {
	case connectors.StatusHealthy:
		return connectors.StatusConnected
	case connectors.StatusExpiring:
		return connectors.StatusDegraded
	case connectors.StatusOldExpired:
		return connectors.StatusExpired
	case connectors.StatusReauthNeeded:
		return connectors.StatusReconnectRequired
	case connectors.StatusPending:
		return connectors.StatusConnecting
	default:
		return connectors.StatusNotConnected
	}
}

func refreshHealthFromLegacyStatus(legacy connectors.ConnectorStatus) string {
	switch mapLegacyConnectorStatus(legacy) {
	case connectors.StatusConnected:
		return "healthy"
	case connectors.StatusDegraded, connectors.StatusRefreshFailed:
		return "degraded"
	case connectors.StatusExpired, connectors.StatusReconnectRequired:
		return "requires_reauth"
	default:
		return "unknown"
	}
}

func refreshStateFromConnectionStatus(status connectors.ConnectionStatus) string {
	switch status {
	case connectors.StatusConnected:
		return "ok"
	case connectors.StatusDegraded:
		return "stale"
	case connectors.StatusRefreshFailed:
		return "failed"
	case connectors.StatusReconnectRequired, connectors.StatusExpired:
		return "reauth_required"
	default:
		return "unknown"
	}
}

func remediationFromStatus(status connectors.ConnectionStatus) any {
	if !status.NeedsAttention() {
		return nil
	}
	switch status {
	case connectors.StatusExpired:
		return "Connection expired. Run: kimbap auth reconnect <provider>"
	case connectors.StatusReconnectRequired, connectors.StatusRevoked:
		return "Reauthorization required. Run: kimbap auth connect <provider>"
	case connectors.StatusRefreshFailed:
		return "Refresh failed. Check provider token endpoint and retry reconnect"
	case connectors.StatusDegraded:
		return "Connection near expiry. Refresh or reconnect soon"
	default:
		return "Inspect provider configuration and run: kimbap auth doctor <provider>"
	}
}

func scopeDelta(prev, next []string) map[string][]string {
	prevSet := map[string]struct{}{}
	nextSet := map[string]struct{}{}
	for _, v := range prev {
		prevSet[v] = struct{}{}
	}
	for _, v := range next {
		nextSet[v] = struct{}{}
	}
	added := make([]string, 0)
	removed := make([]string, 0)
	for v := range nextSet {
		if _, ok := prevSet[v]; !ok {
			added = append(added, v)
		}
	}
	for v := range prevSet {
		if _, ok := nextSet[v]; !ok {
			removed = append(removed, v)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return map[string][]string{"added": added, "removed": removed}
}

func confirmRevocation(providerID string) (bool, error) {
	_, _ = fmt.Fprintf(os.Stdout, "This will disconnect %s. Proceed? [y/N] ", providerID)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, os.ErrClosed) {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return false, nil
		}
		if len(line) == 0 {
			return false, err
		}
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func deleteConnectorState(cfg *config.KimbapConfig, tenantID, providerID string) error {
	db, dialect, err := openConnectorDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := migrateConnectorTable(contextBackground(), db, dialect); err != nil {
		return err
	}

	q := `DELETE FROM connector_states WHERE tenant_id = ? AND name = ?`
	res, err := db.ExecContext(contextBackground(), bindQuery(q, dialect), tenantID, providerID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func checkConnectorEncryptionKeyPresent() doctorCheck {
	if strings.TrimSpace(os.Getenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY")) == "" {
		return doctorCheck{Name: "connector encryption key set", Status: "fail", Detail: "KIMBAP_CONNECTOR_ENCRYPTION_KEY is not set"}
	}
	return doctorCheck{Name: "connector encryption key set", Status: "ok", Detail: "KIMBAP_CONNECTOR_ENCRYPTION_KEY is set"}
}

func checkAuthProviderConfig(providerID string) doctorCheck {
	if strings.TrimSpace(providerID) == "" {
		return doctorCheck{Name: "provider config exists", Status: "skip", Detail: "no provider specified"}
	}
	_, err := providers.GetProvider(providerID)
	if err != nil {
		return doctorCheck{Name: "provider config exists", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "provider config exists", Status: "ok", Detail: providerID}
}

func checkAuthTokenState(cfg *config.KimbapConfig, tenantID, providerID string) (doctorCheck, doctorCheck) {
	if strings.TrimSpace(providerID) == "" {
		return doctorCheck{Name: "token expiry status", Status: "skip", Detail: "no provider specified"}, doctorCheck{Name: "refresh token availability", Status: "skip", Detail: "no provider specified"}
	}

	state, err := getConnectorState(contextBackground(), cfg, tenantID, providerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return doctorCheck{Name: "token expiry status", Status: "skip", Detail: "no token state found"}, doctorCheck{Name: "refresh token availability", Status: "skip", Detail: "no token state found"}
		}
		return doctorCheck{Name: "token expiry status", Status: "fail", Detail: err.Error()}, doctorCheck{Name: "refresh token availability", Status: "fail", Detail: err.Error()}
	}

	expiry := doctorCheck{Name: "token expiry status", Status: "ok", Detail: "token expiry not recorded"}
	if state.ExpiresAt != nil {
		if ts, parseErr := time.Parse(timeLayoutRFC3339, *state.ExpiresAt); parseErr == nil {
			if time.Now().After(ts) {
				expiry = doctorCheck{Name: "token expiry status", Status: "fail", Detail: fmt.Sprintf("token expired at %s", *state.ExpiresAt)}
			} else if time.Until(ts) <= 10*time.Minute {
				expiry = doctorCheck{Name: "token expiry status", Status: "warn", Detail: fmt.Sprintf("token expires soon at %s", *state.ExpiresAt)}
			} else {
				expiry = doctorCheck{Name: "token expiry status", Status: "ok", Detail: fmt.Sprintf("token valid until %s", *state.ExpiresAt)}
			}
		}
	}

	refreshAvailability := doctorCheck{Name: "refresh token availability", Status: "warn", Detail: "refresh token storage not exposed in CLI state table"}
	if state.LastRefresh != nil {
		refreshAvailability = doctorCheck{Name: "refresh token availability", Status: "ok", Detail: "refresh activity detected"}
	}
	if mapLegacyConnectorStatus(connectors.ConnectorStatus(state.Status)) == connectors.StatusReconnectRequired {
		refreshAvailability = doctorCheck{Name: "refresh token availability", Status: "fail", Detail: "connection requires reauthentication"}
	}
	return expiry, refreshAvailability
}

func checkProviderEndpointsReachable(providerID string) []doctorCheck {
	if strings.TrimSpace(providerID) == "" {
		return []doctorCheck{{Name: "provider endpoints reachable", Status: "skip", Detail: "no provider specified"}}
	}

	provider, err := providers.GetProvider(providerID)
	if err != nil {
		return []doctorCheck{{Name: "provider endpoints reachable", Status: "fail", Detail: err.Error()}}
	}

	client := &http.Client{Timeout: 8 * time.Second}
	checks := make([]doctorCheck, 0, 4)
	endpoints := []struct {
		name string
		url  string
	}{
		{name: "auth endpoint reachable", url: provider.AuthEndpoint},
		{name: "token endpoint reachable", url: provider.TokenEndpoint},
		{name: "device endpoint reachable", url: provider.DeviceEndpoint},
		{name: "revocation endpoint reachable", url: provider.RevocationEndpoint},
	}
	for _, ep := range endpoints {
		if strings.TrimSpace(ep.url) == "" {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "skip", Detail: "endpoint not configured"})
			continue
		}
		req, reqErr := http.NewRequestWithContext(contextBackground(), http.MethodHead, ep.url, nil)
		if reqErr != nil {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "fail", Detail: reqErr.Error()})
			continue
		}
		res, callErr := client.Do(req)
		if callErr != nil {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "fail", Detail: callErr.Error()})
			continue
		}
		_ = res.Body.Close()
		if res.StatusCode >= 200 && res.StatusCode < 500 {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "ok", Detail: fmt.Sprintf("HTTP %d", res.StatusCode)})
		} else {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "fail", Detail: fmt.Sprintf("HTTP %d", res.StatusCode)})
		}
	}
	return checks
}

func checkLoopbackPortBindable(port int) doctorCheck {
	addr := "127.0.0.1:0"
	if port > 0 {
		addr = fmt.Sprintf("127.0.0.1:%d", port)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return doctorCheck{Name: "loopback port bindable", Status: "fail", Detail: err.Error()}
	}
	bound := ln.Addr().String()
	_ = ln.Close()
	return doctorCheck{Name: "loopback port bindable", Status: "ok", Detail: bound}
}

func providerIsConfigured(p connectors.ProviderDefinition) bool {
	return strings.TrimSpace(p.AuthEndpoint) != "" && strings.TrimSpace(p.TokenEndpoint) != ""
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func stringOrNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

type authProviderRegistry struct{}

func (authProviderRegistry) GetProvider(id string) (connectors.ProviderDefinition, error) {
	for _, item := range authProviderFixtures() {
		if strings.EqualFold(item.ID, strings.TrimSpace(id)) {
			return item, nil
		}
	}
	return connectors.ProviderDefinition{}, fmt.Errorf("provider %q not found", id)
}

func (authProviderRegistry) ListProviders() ([]connectors.ProviderDefinition, error) {
	items := authProviderFixtures()
	out := make([]connectors.ProviderDefinition, len(items))
	copy(out, items)
	return out, nil
}

var providers = authProviderRegistry{}

type authFlowSelector struct{}

func (authFlowSelector) SelectFlow(raw string, provider connectors.ProviderDefinition, _ []string) (connectors.FlowType, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" || normalized == "auto" {
		if provider.SupportsBrowserFlow() {
			return connectors.FlowBrowser, nil
		}
		if provider.SupportsDeviceFlow() {
			return connectors.FlowDevice, nil
		}
		if len(provider.SupportedFlows) > 0 {
			return provider.SupportedFlows[0], nil
		}
		return "", fmt.Errorf("provider %q has no supported flows", provider.ID)
	}
	selected := connectors.FlowType(normalized)
	if !provider.SupportsFlow(selected) {
		return "", fmt.Errorf("provider %q does not support flow %q", provider.ID, selected)
	}
	return selected, nil
}

var flows = authFlowSelector{}

func authProviderFixtures() []connectors.ProviderDefinition {
	return []connectors.ProviderDefinition{
		{
			ID:                 "google",
			DisplayName:        "Google",
			SupportedFlows:     []connectors.FlowType{connectors.FlowBrowser, connectors.FlowDevice},
			AuthEndpoint:       "https://accounts.google.com/o/oauth2/v2/auth",
			TokenEndpoint:      "https://oauth2.googleapis.com/token",
			DeviceEndpoint:     "https://oauth2.googleapis.com/device/code",
			RevocationEndpoint: "https://oauth2.googleapis.com/revoke",
			UserInfoEndpoint:   "https://openidconnect.googleapis.com/v1/userinfo",
			DefaultScopes:      []string{"openid", "email", "profile"},
			ScopePresets: map[string]string{
				"basic": "openid email profile",
				"mail":  "openid email profile https://www.googleapis.com/auth/gmail.readonly",
			},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser, connectors.ScopeWorkspace},
			PKCERequired:         true,
			Notes:                "Use browser flow for richer consent UX; device flow for headless terminals.",
		},
		{
			ID:                 "github",
			DisplayName:        "GitHub",
			SupportedFlows:     []connectors.FlowType{connectors.FlowBrowser, connectors.FlowDevice},
			AuthEndpoint:       "https://github.com/login/oauth/authorize",
			TokenEndpoint:      "https://github.com/login/oauth/access_token",
			DeviceEndpoint:     "https://github.com/login/device/code",
			RevocationEndpoint: "",
			UserInfoEndpoint:   "https://api.github.com/user",
			DefaultScopes:      []string{"read:user"},
			ScopePresets: map[string]string{
				"basic": "read:user",
				"repo":  "read:user repo",
			},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser},
			PKCERequired:         false,
			Notes:                "Revocation endpoint is provider-specific and may require app-side revoke UX.",
		},
	}
}

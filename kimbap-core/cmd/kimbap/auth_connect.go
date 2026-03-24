package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	"github.com/dunialabs/kimbap-core/internal/connectors/flows/browser"
	"github.com/dunialabs/kimbap-core/internal/connectors/flows/clientcred"
	"github.com/dunialabs/kimbap-core/internal/connectors/flows/device"
	"github.com/spf13/cobra"
)

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

			store, storeErr := openConnectorStore(cfg)
			if storeErr == nil {
				mgr := connectors.NewManager(store)

				provider, provErr := providers.GetProvider(providerID)
				if provErr == nil {
					mgr.RegisterConfig(connectors.ConnectorConfig{
						Name:         providerID,
						Provider:     providerID,
						ClientID:     resolveClientID(cfg, providerID),
						ClientSecret: resolveClientSecret(cfg, providerID),
						TokenURL:     provider.TokenEndpoint,
						DeviceURL:    provider.DeviceEndpoint,
						Scopes:       scopeValues(scopeInput, provider.DefaultScopes),
					})
				}

				refreshErr := mgr.Refresh(contextBackground(), activeTenant, providerID)
				if refreshErr == nil {
					_, _ = fmt.Fprintln(os.Stderr, "Token refresh succeeded.")
					refreshedState, _ := mgr.Status(contextBackground(), activeTenant, providerID)
					prevScopes := []string{}
					if refreshedState != nil {
						prevScopes = refreshedState.Scopes
					}
					requestedScopes := scopeValues(scopeInput, prevScopes)
					return printOutput(map[string]any{
						"status":           "ok",
						"operation":        "auth.reconnect",
						"tenant_id":        activeTenant,
						"provider":         providerID,
						"reconnect_method": "refresh",
						"message":          "Refresh token path succeeded",
						"delta": map[string]any{
							"scope_changes":   scopeDelta(prevScopes, requestedScopes),
							"account_changes": "none",
						},
					})
				}
				_, _ = fmt.Fprintf(os.Stderr, "Refresh failed: %v\nFalling back to full connect flow...\n", refreshErr)
			}

			existingWorkspace := ""
			existingScope := string(connectors.ScopeUser)
			if storeErr == nil {
				prevMgr := connectors.NewManager(store)
				if prev, _ := prevMgr.Status(contextBackground(), activeTenant, providerID); prev != nil {
					if prev.WorkspaceID != "" {
						existingWorkspace = prev.WorkspaceID
					}
					if prev.ConnectionScope != "" {
						existingScope = string(prev.ConnectionScope)
					}
				}
			}

			return runAuthConnect(
				cfg,
				providerID,
				activeTenant,
				flow,
				scopeInput,
				"auto",
				false,
				0,
				5*time.Minute,
				existingWorkspace,
				existingScope,
				true,
			)
		},
	}
	cmd.Flags().StringVar(&flow, "flow", "auto", "auth flow to use (auto, browser, device)")
	cmd.Flags().StringVar(&scopeInput, "scope", "", "requested scopes (space/comma separated)")
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func runAuthConnect(
	cfg *config.KimbapConfig,
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

	if strings.Contains(provider.AuthEndpoint, "{") || strings.Contains(provider.TokenEndpoint, "{") {
		return fmt.Errorf("provider %q has endpoint placeholders that require substitution (not yet supported)", providerID)
	}

	scopes := scopeValues(scopeInput, provider.DefaultScopes)
	selectedFlow, err := flows.SelectFlow(strings.TrimSpace(flow), provider, scopes)
	if err != nil {
		return err
	}

	connScope := parseConnectionScope(connectionScope)
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	operation := "auth.connect"
	if reconnectMode {
		operation = "auth.reconnect.connect"
	}

	ctx := contextBackground()
	var accessToken, refreshToken, grantedScope string
	var expiresIn int

	switch selectedFlow {
	case connectors.FlowDevice:
		_, _ = fmt.Fprintf(os.Stderr, "Starting device authorization for %s...\n\n", provider.DisplayName)
		deviceResult, deviceErr := device.RunDeviceFlow(ctx, device.DeviceFlowConfig{
			DeviceEndpoint: provider.DeviceEndpoint,
			TokenEndpoint:  provider.TokenEndpoint,
			ClientID:       resolveClientID(cfg, providerID),
			ClientSecret:   resolveClientSecret(cfg, providerID),
			Scopes:         scopes,
			Timeout:        timeout,
		}, os.Stderr)
		if deviceErr != nil {
			return deviceErr
		}
		accessToken = deviceResult.AccessToken
		refreshToken = deviceResult.RefreshToken
		expiresIn = deviceResult.ExpiresIn
		grantedScope = deviceResult.Scope

	case connectors.FlowBrowser:
		_, _ = fmt.Fprintf(os.Stderr, "Preparing OAuth connection for %s...\n", provider.DisplayName)
		browserResult, browserErr := browser.RunBrowserFlow(ctx, browser.BrowserFlowConfig{
			AuthEndpoint:  provider.AuthEndpoint,
			TokenEndpoint: provider.TokenEndpoint,
			ClientID:      resolveClientID(cfg, providerID),
			ClientSecret:  resolveClientSecret(cfg, providerID),
			Scopes:        scopes,
			Port:          port,
			NoOpen:        noOpen || strings.EqualFold(browserName, "none"),
			Timeout:       timeout,
		}, os.Stderr)
		if browserErr != nil {
			return browserErr
		}
		accessToken = browserResult.AccessToken
		refreshToken = browserResult.RefreshToken
		expiresIn = browserResult.ExpiresIn
		grantedScope = browserResult.Scope

	case connectors.FlowClientCredentials:
		_, _ = fmt.Fprintf(os.Stderr, "Starting client credentials flow for %s...\n", provider.DisplayName)
		ccResult, ccErr := clientcred.RunClientCredentialsFlow(ctx, clientcred.ClientCredentialsConfig{
			TokenEndpoint: provider.TokenEndpoint,
			ClientID:      resolveClientID(cfg, providerID),
			ClientSecret:  resolveClientSecret(cfg, providerID),
			Scopes:        scopes,
		})
		if ccErr != nil {
			return ccErr
		}
		accessToken = ccResult.AccessToken
		expiresIn = ccResult.ExpiresIn
		grantedScope = ccResult.Scope

	default:
		return fmt.Errorf("unsupported flow: %s", selectedFlow)
	}

	store, storeErr := openConnectorStore(cfg)
	if storeErr != nil {
		return storeErr
	}
	mgr := connectors.NewManager(store)
	mgr.RegisterConfig(connectors.ConnectorConfig{
		Name:         providerID,
		Provider:     providerID,
		ClientID:     resolveClientID(cfg, providerID),
		ClientSecret: resolveClientSecret(cfg, providerID),
		TokenURL:     provider.TokenEndpoint,
		DeviceURL:    provider.DeviceEndpoint,
		Scopes:       scopes,
	})

	if err := mgr.StoreConnection(ctx, tenantID, providerID, provider.ID, accessToken, refreshToken, expiresIn, grantedScope, selectedFlow, connScope, workspace); err != nil {
		return fmt.Errorf("store connection: %w", err)
	}

	grantedScopes := scopes
	if grantedScope != "" {
		grantedScopes = strings.Fields(grantedScope)
	}

	_, _ = fmt.Fprintf(os.Stderr, "\nConnected to %s.\n", provider.DisplayName)
	_, _ = fmt.Fprintf(os.Stderr, "Connection scope: %s\n", connScope)
	if len(grantedScopes) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Granted scopes: %s\n", strings.Join(grantedScopes, ", "))
	}
	_, _ = fmt.Fprintln(os.Stderr, "Refresh: healthy")
	_, _ = fmt.Fprintln(os.Stderr, "Ready to use with Kimbap actions.")

	return printOutput(map[string]any{
		"status":           "ok",
		"operation":        operation,
		"tenant_id":        tenantID,
		"provider":         provider.ID,
		"connection_scope": connScope,
		"status_detail":    connectors.StatusConnected,
		"granted_scopes":   grantedScopes,
		"flow_used":        selectedFlow,
		"workspace":        stringOrNil(workspace),
	})
}

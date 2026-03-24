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

	cmd.Flags().StringVar(&flow, "flow", "auto", "auth flow to use (auto, browser, device, client-credentials)")
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
			if p, pErr := providers.GetProvider(providerID); pErr == nil {
				providerID = p.ID
			}
			activeTenant := connectorTenant(tenant)

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(os.Stderr, "Attempting to reconnect...")

			auditEmitter := initAuthAuditEmitter(cfg)
			defer closeAuditEmitter(auditEmitter)

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
					if auditEmitter != nil {
						auditEmitter.RefreshSucceeded(contextBackground(), providerID, activeTenant)
					}
					refreshedState, _ := mgr.Status(contextBackground(), activeTenant, providerID)
					prevScopes := []string{}
					if refreshedState != nil {
						prevScopes = refreshedState.Scopes
					}
					requestedScopes := scopeValues(scopeInput, prevScopes)
					if !outputAsJSON() {
						return nil
					}
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
				if auditEmitter != nil {
					auditEmitter.RefreshFailed(contextBackground(), providerID, activeTenant, refreshErr.Error())
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
	cmd.Flags().StringVar(&flow, "flow", "auto", "auth flow to use (auto, browser, device, client-credentials)")
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
		if outputAsJSON() {
			_ = printOutput(map[string]any{
				"status":    "not_found",
				"operation": "auth.connect",
				"tenant_id": tenantID,
				"provider":  providerID,
				"message":   err.Error(),
			})
		}
		return fmt.Errorf("provider %q not found: %w", providerID, err)
	}
	providerID = provider.ID

	if strings.Contains(provider.AuthEndpoint, "{") || strings.Contains(provider.TokenEndpoint, "{") {
		return fmt.Errorf("provider %q has endpoint placeholders that require substitution (not yet supported)", providerID)
	}

	scopes := scopeValues(scopeInput, provider.DefaultScopes)
	selectedFlow, err := flows.SelectFlow(strings.TrimSpace(flow), provider, scopes)
	if err != nil {
		return err
	}

	connScope := resolveConnectionScope(connectionScope, provider)
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	operation := "auth.connect"
	if reconnectMode {
		operation = "auth.reconnect.connect"
	}

	ctx := contextBackground()
	auditEmitter := initAuthAuditEmitter(cfg)
	defer closeAuditEmitter(auditEmitter)
	if auditEmitter != nil {
		auditEmitter.ConnectStarted(ctx, providerID, tenantID, selectedFlow)
	}

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
			if auditEmitter != nil {
				auditEmitter.ConnectFailed(ctx, providerID, tenantID, selectedFlow, deviceErr.Error())
			}
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
			if auditEmitter != nil {
				auditEmitter.ConnectFailed(ctx, providerID, tenantID, selectedFlow, browserErr.Error())
			}
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
			if auditEmitter != nil {
				auditEmitter.ConnectFailed(ctx, providerID, tenantID, selectedFlow, ccErr.Error())
			}
			return ccErr
		}
		accessToken = ccResult.AccessToken
		expiresIn = ccResult.ExpiresIn
		grantedScope = ccResult.Scope

	default:
		if auditEmitter != nil {
			auditEmitter.ConnectFailed(ctx, providerID, tenantID, selectedFlow, fmt.Sprintf("unsupported flow: %s", selectedFlow))
		}
		return fmt.Errorf("unsupported flow: %s", selectedFlow)
	}

	store, storeErr := openConnectorStore(cfg)
	if storeErr != nil {
		if auditEmitter != nil {
			auditEmitter.ConnectFailed(ctx, providerID, tenantID, selectedFlow, storeErr.Error())
		}
		return storeErr
	}
	mgr := connectors.NewManager(store)
	prevScopes := []string{}
	existingPrincipal := ""
	if reconnectMode {
		if prev, _ := store.Get(ctx, tenantID, providerID); prev != nil {
			prevScopes = append(prevScopes, prev.Scopes...)
			existingPrincipal = strings.TrimSpace(prev.ConnectedPrincipal)
		}
	}
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
		if auditEmitter != nil {
			auditEmitter.ConnectFailed(ctx, providerID, tenantID, selectedFlow, err.Error())
		}
		return fmt.Errorf("store connection: %w", err)
	}

	connectedPrincipal := ""
	if strings.TrimSpace(provider.UserInfoEndpoint) != "" && strings.TrimSpace(accessToken) != "" {
		if userInfo, uErr := connectors.FetchUserInfo(ctx, provider.UserInfoEndpoint, accessToken); uErr == nil && userInfo != nil {
			connectedPrincipal = userInfo.Principal()
			if connectedPrincipal != "" {
				if stored, _ := store.Get(ctx, tenantID, providerID); stored != nil {
					stored.ConnectedPrincipal = connectedPrincipal
					_ = store.Save(ctx, stored)
				}
			}
		}
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
	if expiresIn > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Expires in: %s\n", formatDuration(time.Duration(expiresIn)*time.Second))
	}
	_, _ = fmt.Fprintln(os.Stderr, "Refresh: healthy")
	_, _ = fmt.Fprintln(os.Stderr, "Ready to use with Kimbap actions.")

	accountChanges := "none"
	if reconnectMode {
		newPrincipal := strings.TrimSpace(connectedPrincipal)
		if newPrincipal == "" {
			if current, _ := store.Get(ctx, tenantID, providerID); current != nil {
				newPrincipal = strings.TrimSpace(current.ConnectedPrincipal)
			}
		}
		switch {
		case existingPrincipal == "" && newPrincipal != "":
			accountChanges = fmt.Sprintf("set to %s", newPrincipal)
		case existingPrincipal != "" && newPrincipal == "":
			accountChanges = fmt.Sprintf("cleared from %s", existingPrincipal)
		case existingPrincipal != "" && newPrincipal != "" && existingPrincipal != newPrincipal:
			accountChanges = fmt.Sprintf("changed from %s to %s", existingPrincipal, newPrincipal)
		}
	}

	result := map[string]any{
		"status":             "ok",
		"operation":          operation,
		"tenant_id":          tenantID,
		"provider":           provider.ID,
		"connection_scope":   connScope,
		"status_detail":      connectors.StatusConnected,
		"granted_scopes":     grantedScopes,
		"flow_used":          selectedFlow,
		"expires_in_seconds": expiresIn,
		"workspace":          stringOrNil(workspace),
	}
	if reconnectMode {
		result["delta"] = map[string]any{
			"scope_changes":   scopeDelta(prevScopes, grantedScopes),
			"account_changes": accountChanges,
		}
	}

	if auditEmitter != nil {
		if reconnectMode {
			auditEmitter.ReconnectCompleted(ctx, providerID, tenantID, selectedFlow)
		} else {
			auditEmitter.ConnectCompleted(ctx, providerID, tenantID, selectedFlow, grantedScopes)
		}
	}

	if !outputAsJSON() {
		return nil
	}
	return printOutput(result)
}

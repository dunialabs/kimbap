package main

import (
	"errors"
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
	var profile string
	var extras []string
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
				profile,
				false,
				parseExtras(extras),
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
	cmd.Flags().StringVar(&profile, "profile", "default", "connection profile name for multiple accounts per provider")
	cmd.Flags().StringArrayVar(&extras, "extra", nil, "provider-specific key=value pairs (e.g. --extra subdomain=acme)")
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")

	return cmd
}

func newAuthReconnectCommand() *cobra.Command {
	var flow string
	var scopeInput string
	var profile string
	var extras []string
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
			extraMap := parseExtras(extras)

			_, _ = fmt.Fprintln(os.Stderr, "Attempting to reconnect...")

			auditEmitter := initAuthAuditEmitter(cfg)
			defer closeAuditEmitter(auditEmitter)

			store, storeErr := openConnectorStore(cfg)
			if storeErr != nil {
				return storeErr
			}
			mgr := connectors.NewManager(store)
			storeName := connectorStoreName(providerID, profile)

			provider, provErr := providers.GetProvider(providerID)
			if provErr == nil {
				if err := validateProviderExtraValues(provider, extraMap); err != nil {
					return err
				}
				provider = substituteProviderEndpoints(provider, extraMap)
				if hasUnresolvedPlaceholders(provider) {
					missing := listUnresolvedPlaceholders(provider)
					if len(missing) == 0 {
						return fmt.Errorf("provider %q has unresolved endpoint placeholders. Use --extra key=value to provide them", providerID)
					}
					return fmt.Errorf("provider %q has unresolved endpoint placeholders: %s\nUse --extra key=value to provide them (e.g. --extra %s=<value>)", providerID, strings.Join(missing, ", "), missing[0])
				}
				if err := validateProviderEndpoints(provider); err != nil {
					return fmt.Errorf("provider %q endpoint validation failed: %w", providerID, err)
				}
				mgr.RegisterConfig(connectors.ConnectorConfig{
					Name:         storeName,
					Provider:     providerID,
					ClientID:     resolveClientID(cfg, providerID),
					ClientSecret: resolveClientSecret(cfg, providerID),
					TokenURL:     provider.TokenEndpoint,
					DeviceURL:    provider.DeviceEndpoint,
					Scopes:       scopeValues(scopeInput, provider.DefaultScopes),
				})
			}

			refreshErr := mgr.Refresh(contextBackground(), activeTenant, storeName)
			if refreshErr == nil {
				_, _ = fmt.Fprintln(os.Stderr, "Token refresh succeeded.")
				if auditEmitter != nil {
					auditEmitter.RefreshSucceeded(contextBackground(), providerID, activeTenant)
					flowUsed := connectors.FlowType("")
					if refreshedState, _ := mgr.Status(contextBackground(), activeTenant, storeName); refreshedState != nil {
						flowUsed = refreshedState.FlowUsed
					}
					auditEmitter.ReconnectCompleted(contextBackground(), providerID, activeTenant, flowUsed)
				}
				if !outputAsJSON() {
					_, _ = fmt.Fprintln(os.Stderr, "Scope changes: none (refresh does not change scopes)")
					_, _ = fmt.Fprintln(os.Stderr, "Account changes: none")
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
						"scope_changes":   map[string][]string{"added": {}, "removed": {}},
						"account_changes": "none",
					},
				})
			}
			if auditEmitter != nil {
				auditEmitter.RefreshFailed(contextBackground(), providerID, activeTenant, refreshErr.Error())
			}
			_, _ = fmt.Fprintf(os.Stderr, "Refresh failed: %v\nFalling back to full connect flow...\n", refreshErr)

			existingWorkspace := ""
			existingScope := string(connectors.ScopeUser)
			prevMgr := connectors.NewManager(store)
			if prev, _ := prevMgr.Status(contextBackground(), activeTenant, connectorStoreName(providerID, profile)); prev != nil {
				if prev.WorkspaceID != "" {
					existingWorkspace = prev.WorkspaceID
				}
				if prev.ConnectionScope != "" {
					existingScope = string(prev.ConnectionScope)
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
				profile,
				true,
				extraMap,
			)
		},
	}
	cmd.Flags().StringVar(&flow, "flow", "auto", "auth flow to use (auto, browser, device, client-credentials)")
	cmd.Flags().StringVar(&scopeInput, "scope", "", "requested scopes (space/comma separated)")
	cmd.Flags().StringVar(&profile, "profile", "default", "connection profile name for multiple accounts per provider")
	cmd.Flags().StringArrayVar(&extras, "extra", nil, "provider-specific key=value pairs (e.g. --extra subdomain=acme)")
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
	profile string,
	reconnectMode bool,
	extras map[string]string,
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
	if err := validateProviderExtraValues(provider, extras); err != nil {
		return err
	}

	provider = substituteProviderEndpoints(provider, extras)
	if hasUnresolvedPlaceholders(provider) {
		missing := listUnresolvedPlaceholders(provider)
		if len(missing) == 0 {
			return fmt.Errorf("provider %q has unresolved endpoint placeholders. Use --extra key=value to provide them", providerID)
		}
		return fmt.Errorf("provider %q has unresolved endpoint placeholders: %s\nUse --extra key=value to provide them (e.g. --extra %s=<value>)", providerID, strings.Join(missing, ", "), missing[0])
	}
	if err := validateProviderEndpoints(provider); err != nil {
		return fmt.Errorf("provider %q endpoint validation failed: %w", providerID, err)
	}

	storeName := connectorStoreName(providerID, profile)
	nextStatusCmd := fmt.Sprintf("kimbap auth status %s", provider.ID)
	if strings.TrimSpace(profile) != "" && strings.TrimSpace(profile) != "default" {
		nextStatusCmd = fmt.Sprintf("kimbap auth status %s --profile %s", provider.ID, strings.TrimSpace(profile))
	}

	scopes := scopeValues(scopeInput, provider.DefaultScopes)
	selectedFlow, err := flows.SelectFlow(strings.TrimSpace(flow), provider, []string{
		"browser=" + strings.ToLower(strings.TrimSpace(browserName)),
	})
	if err != nil {
		return err
	}

	connScope, err := resolveConnectionScope(connectionScope, provider)
	if err != nil {
		return err
	}
	workspace = strings.TrimSpace(workspace)
	if connScope == connectors.ScopeWorkspace {
		if workspace == "" {
			return fmt.Errorf("workspace connection scope requires --workspace")
		}
	} else {
		workspace = ""
	}
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
	if existing, _ := store.Get(ctx, tenantID, storeName); existing != nil {
		if connScope == connectors.ScopeWorkspace && existing.ConnectionScope == connectors.ScopeWorkspace {
			existingWorkspace := strings.TrimSpace(existing.WorkspaceID)
			if existingWorkspace != "" && workspace != "" && existingWorkspace != workspace {
				return fmt.Errorf("workspace-scoped connection %q already bound to workspace %q; use a different --profile or revoke first", storeName, existingWorkspace)
			}
		}
	}
	if reconnectMode {
		if prev, _ := store.Get(ctx, tenantID, storeName); prev != nil {
			prevScopes = append(prevScopes, prev.Scopes...)
			existingPrincipal = strings.TrimSpace(prev.ConnectedPrincipal)
		}
	}
	mgr.RegisterConfig(connectors.ConnectorConfig{
		Name:         storeName,
		Provider:     providerID,
		ClientID:     resolveClientID(cfg, providerID),
		ClientSecret: resolveClientSecret(cfg, providerID),
		TokenURL:     provider.TokenEndpoint,
		DeviceURL:    provider.DeviceEndpoint,
		Scopes:       scopes,
	})

	var accessToken, refreshToken, grantedScope string
	var expiresIn int

	switch selectedFlow {
	case connectors.FlowDevice:
		if auditEmitter != nil {
			auditEmitter.DeviceFlowStarted(ctx, providerID, tenantID)
		}
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
		if auditEmitter != nil {
			auditEmitter.DeviceFlowCompleted(ctx, providerID, tenantID)
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
			normalizedFlow := normalizeFlowInput(flow)
			isEnvFailure := isBrowserEnvFailure(browserErr)
			if isEnvFailure && (normalizedFlow == "" || normalizedFlow == "auto") && provider.SupportsDeviceFlow() {
				if auditEmitter != nil {
					auditEmitter.ConnectFailed(ctx, providerID, tenantID, connectors.FlowBrowser, browserErr.Error())
					auditEmitter.DeviceFlowStarted(ctx, providerID, tenantID)
				}
				_, _ = fmt.Fprintf(os.Stderr, "Browser flow failed (%v). Falling back to device authorization...\n", browserErr)
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
						auditEmitter.ConnectFailed(ctx, providerID, tenantID, connectors.FlowDevice, deviceErr.Error())
					}
					return deviceErr
				}
				if auditEmitter != nil {
					auditEmitter.DeviceFlowCompleted(ctx, providerID, tenantID)
				}
				selectedFlow = connectors.FlowDevice
				accessToken = deviceResult.AccessToken
				refreshToken = deviceResult.RefreshToken
				expiresIn = deviceResult.ExpiresIn
				grantedScope = deviceResult.Scope
				break
			}
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
		_, _ = fmt.Fprintf(os.Stderr, "Starting client credentials flow for %s (service-level connection)...\n", provider.DisplayName)
		_, _ = fmt.Fprintf(os.Stderr, "Client: %s\n", resolveClientID(cfg, providerID))
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

	if err := mgr.StoreConnection(ctx, tenantID, storeName, provider.ID, accessToken, refreshToken, expiresIn, grantedScope, selectedFlow, connScope, workspace); err != nil {
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
				if stored, _ := store.Get(ctx, tenantID, storeName); stored != nil {
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
	if connectedPrincipal != "" {
		_, _ = fmt.Fprintf(os.Stderr, "Connected as: %s\n", connectedPrincipal)
	}
	_, _ = fmt.Fprintf(os.Stderr, "Connection scope: %s\n", connScope)
	if len(grantedScopes) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Granted scopes: %s\n", strings.Join(grantedScopes, ", "))
	}
	if expiresIn > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Expires in: %s\n", formatDuration(time.Duration(expiresIn)*time.Second))
	}
	_, _ = fmt.Fprintln(os.Stderr, "Refresh: healthy")
	_, _ = fmt.Fprintln(os.Stderr, "Ready to use with Kimbap actions.")
	_, _ = fmt.Fprintf(os.Stderr, "Next: %s\n", nextStatusCmd)

	accountChanges := "none"
	scopeChanges := scopeDelta(prevScopes, grantedScopes)
	if reconnectMode {
		newPrincipal := strings.TrimSpace(connectedPrincipal)
		if newPrincipal == "" {
			if current, _ := store.Get(ctx, tenantID, storeName); current != nil {
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
		"next_action":        nextStatusCmd,
	}
	if reconnectMode {
		result["delta"] = map[string]any{
			"scope_changes":   scopeChanges,
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
		if reconnectMode {
			if len(scopeChanges["added"]) > 0 || len(scopeChanges["removed"]) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "Scope changes: added=%v removed=%v\n", scopeChanges["added"], scopeChanges["removed"])
			}
			_, _ = fmt.Fprintf(os.Stderr, "Account changes: %s\n", accountChanges)
		}
		return nil
	}
	return printOutput(result)
}

func isBrowserEnvFailure(err error) bool {
	return errors.Is(err, browser.ErrLoopbackListener)
}

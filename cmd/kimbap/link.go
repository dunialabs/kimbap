package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/spf13/cobra"
)

type linkServiceInfo struct {
	Service       string
	AuthType      actions.AuthType
	CredentialRef string
	OAuthProvider string
}

type linkListRow struct {
	Service       string `json:"service"`
	AuthType      string `json:"auth_type"`
	Status        string `json:"status"`
	CredentialRef string `json:"credential_ref,omitempty"`
	Connector     string `json:"connector,omitempty"`
}

func newLinkCommand() *cobra.Command {
	var flow string
	var scopeInput string
	var tenant string
	var profile string
	var statusOnly bool
	var fromStdin bool
	var fromFile string

	cmd := &cobra.Command{
		Use:   "link <service>",
		Short: "Connect a service for use with Kimbap",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			service := strings.TrimSpace(args[0])
			if service == "" {
				return fmt.Errorf("service is required")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			services, err := loadLinkServices(cfg)
			if err != nil {
				return err
			}

			lookup := strings.ToLower(service)
			info, ok := services[lookup]
			if !ok {
				names := make([]string, 0, len(services))
				for key := range services {
					names = append(names, key)
				}
				hint := "Run 'kimbap link list' to see available services."
				if suggestion := didYouMean(lookup, names); suggestion != "" {
					hint = fmt.Sprintf("Did you mean %q? Run 'kimbap link list' to see available services.", suggestion)
				}
				return fmt.Errorf("service %q not found in installed services. %s", service, hint)
			}

			if info.AuthType == actions.AuthTypeNone {
				if outputAsJSON() {
					return printOutput(map[string]any{
						"service":   info.Service,
						"auth_type": string(actions.AuthTypeNone),
						"status":    "connected",
						"message":   fmt.Sprintf("%s requires no authentication.", info.Service),
					})
				}
				return printOutput(fmt.Sprintf("%s requires no authentication.", info.Service))
			}

			tenantID := connectorTenant(tenant)
			oauthStates, oauthErr := listConnectorStates(contextBackground(), cfg, tenantID)
			if oauthErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "warning: %s\n", unavailableMessage(componentConnectorStore, oauthErr))
			}

			if err := linkRejectStdinFileForOAuth(info, oauthStates, fromStdin, fromFile); err != nil {
				return err
			}

			if statusOnly && (fromStdin || fromFile != "") {
				return fmt.Errorf("--status cannot be combined with --stdin or --file")
			}

			switch {
			case linkIsOAuthService(info, oauthStates):
				providerID := info.OAuthProvider
				if providerID == "" {
					providerID = info.Service
				}

				if statusOnly {
					if oauthErr != nil {
						return unavailableError(componentConnectorStore, oauthErr)
					}
					status := linkOAuthConnectionStatus(providerID, profile, oauthStates)
					connectorName := connectorStoreName(providerID, profile)
					if outputAsJSON() {
						return printOutput(map[string]any{
							"service":   info.Service,
							"auth_type": "oauth2",
							"status":    status,
							"connector": connectorName,
						})
					}
					switch status {
					case string(connectors.StatusConnected):
						return printOutput(fmt.Sprintf("✓ %s is connected via OAuth (%s)", info.Service, connectorName))
					case "not_connected":
						return printOutput(fmt.Sprintf("%s is not connected via OAuth (%s)", info.Service, connectorName))
					default:
						return printOutput(fmt.Sprintf("%s OAuth status: %s (%s)", info.Service, status, connectorName))
					}
				}

				return runAuthConnect(
					cfg,
					providerID,
					connectorTenant(tenant),
					flow,
					scopeInput,
					"auto",
					false,
					0,
					5*time.Minute,
					"",
					string(connectors.ScopeUser),
					profile,
					false,
					nil,
				)

			default:
				return linkHandleKeyBasedService(cfg, info, statusOnly, connectorTenant(tenant), fromStdin, fromFile)
			}
		},
	}

	cmd.Flags().StringVar(&flow, "flow", "auto", "auth flow to use (auto, browser, device, client-credentials)")
	cmd.Flags().StringVar(&scopeInput, "scope", "", "requested scopes (space/comma separated)")
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	cmd.Flags().StringVar(&profile, "profile", "default", "connection profile name for multiple accounts")
	cmd.Flags().BoolVar(&statusOnly, "status", false, "show connection status without connecting")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read credential value from stdin")
	cmd.Flags().StringVar(&fromFile, "file", "", "read credential value from file")

	cmd.AddCommand(newLinkListCommand())
	return cmd
}

func newLinkListCommand() *cobra.Command {
	var tenant string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List service connection status",
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			services, err := loadLinkServices(cfg)
			if err != nil {
				return err
			}

			tenantID := connectorTenant(tenant)
			oauthStates, oauthErr := listConnectorStates(contextBackground(), cfg, tenantID)
			if oauthErr != nil {
				return unavailableError(componentConnectorStore, oauthErr)
			}
			vs, vsErr := initVaultStore(cfg)
			if vsErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "warning: %s\n", unavailableMessage(componentVault, vsErr))
			}
			if vs != nil {
				defer closeVaultStoreIfPossible(vs)
			}

			keys := make([]string, 0, len(services))
			for key := range services {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			rows := make([]linkListRow, 0, len(keys))
			for _, key := range keys {
				info := services[key]
				row := linkListRow{
					Service:  info.Service,
					AuthType: string(info.AuthType),
				}

				switch {
				case linkIsOAuthService(info, oauthStates):
					providerID := info.OAuthProvider
					if providerID == "" {
						providerID = info.Service
					}
					row.AuthType = string(actions.AuthTypeOAuth2)
					row.Connector = connectorStoreName(providerID, "")
					row.Status = linkOAuthConnectionStatus(providerID, "", oauthStates)

				case info.AuthType == actions.AuthTypeNone:
					row.Status = "connected"

				default:
					row.CredentialRef = info.CredentialRef
					if vs != nil && strings.TrimSpace(info.CredentialRef) != "" {
						if exists, getErr := vs.Exists(contextBackground(), tenantID, info.CredentialRef); getErr == nil && exists {
							row.Status = "connected"
						} else if getErr == nil || errors.Is(getErr, vault.ErrSecretNotFound) {
							row.Status = "not_connected"
						} else {
							row.Status = "unknown"
						}
					} else if vsErr != nil {
						row.Status = "unknown"
					} else {
						row.Status = "not_connected"
					}
				}

				if row.Status == "" {
					row.Status = "not_connected"
				}
				rows = append(rows, row)
			}

			if outputAsJSON() {
				return printOutput(rows)
			}

			if len(rows) == 0 {
				return printOutput("No services found.")
			}

			_, _ = fmt.Fprintf(
				c.OutOrStdout(),
				"%-18s %-10s %-14s %-24s %s\n",
				"SERVICE",
				"AUTH",
				"STATUS",
				"CREDENTIAL_REF",
				"CONNECTOR",
			)
			for _, row := range rows {
				_, _ = fmt.Fprintf(
					c.OutOrStdout(),
					"%-18s %-10s %-14s %-24s %s\n",
					row.Service,
					row.AuthType,
					row.Status,
					linkDefaultDash(row.CredentialRef),
					linkDefaultDash(row.Connector),
				)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func loadLinkServices(cfg *config.KimbapConfig) (map[string]linkServiceInfo, error) {
	defs, err := loadInstalledActions(cfg)
	if err != nil {
		return nil, err
	}

	services := map[string]linkServiceInfo{}
	for _, def := range defs {
		namespace := strings.TrimSpace(def.Namespace)
		if namespace == "" {
			continue
		}
		key := strings.ToLower(namespace)

		current, ok := services[key]
		if !ok {
			services[key] = linkServiceInfo{
				Service:       namespace,
				AuthType:      def.Auth.Type,
				CredentialRef: strings.TrimSpace(def.Auth.CredentialRef),
				OAuthProvider: linkResolveOAuthProvider(namespace),
			}
			continue
		}

		current = linkMergeServiceAuth(current, def)
		if current.OAuthProvider == "" {
			current.OAuthProvider = linkResolveOAuthProvider(namespace)
		}
		services[key] = current
	}
	return services, nil
}

func linkMergeServiceAuth(current linkServiceInfo, def actions.ActionDefinition) linkServiceInfo {
	nextAuth := def.Auth.Type
	nextRef := strings.TrimSpace(def.Auth.CredentialRef)

	if current.AuthType != actions.AuthTypeOAuth2 && nextAuth == actions.AuthTypeOAuth2 {
		current.AuthType = nextAuth
		if nextRef != "" {
			current.CredentialRef = nextRef
		}
		return current
	}

	if current.AuthType == actions.AuthTypeNone && nextAuth != actions.AuthTypeNone {
		current.AuthType = nextAuth
	}

	if current.CredentialRef == "" && nextRef != "" {
		current.CredentialRef = nextRef
	}

	return current
}

func linkResolveOAuthProvider(service string) string {
	provider, err := providers.GetProvider(service)
	if err != nil {
		return ""
	}
	return provider.ID
}

func linkIsOAuthService(info linkServiceInfo, oauthStates []connectorStateRow) bool {
	if info.AuthType == actions.AuthTypeOAuth2 {
		return true
	}
	if info.OAuthProvider != "" && isConnectorResolvableRef(info.CredentialRef) {
		return true
	}
	serviceKey := strings.ToLower(strings.TrimSpace(info.Service))
	for _, state := range oauthStates {
		if strings.EqualFold(strings.TrimSpace(state.Provider), serviceKey) {
			return true
		}
	}
	return false
}

func isConnectorResolvableRef(ref string) bool {
	lower := strings.ToLower(strings.TrimSpace(ref))
	for _, suffix := range []string{".oauth_token", ".oidc_token", ".token", ".oauth"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func linkOAuthConnectionStatus(providerID, profile string, states []connectorStateRow) string {
	storeName := connectorStoreName(providerID, profile)
	for _, state := range states {
		if strings.EqualFold(state.Name, storeName) {
			return connectorComputedStatus(state)
		}
	}
	if p := strings.TrimSpace(profile); p != "" && !strings.EqualFold(p, "default") {
		return "not_connected"
	}
	for _, state := range states {
		if strings.EqualFold(state.Provider, providerID) && strings.EqualFold(state.Name, providerID) {
			return connectorComputedStatus(state)
		}
	}
	return "not_connected"
}

func linkHandleKeyBasedService(cfg *config.KimbapConfig, info linkServiceInfo, statusOnly bool, tenantID string, fromStdin bool, fromFile string) error {
	credentialRef := strings.TrimSpace(info.CredentialRef)
	if credentialRef == "" {
		if outputAsJSON() {
			return printOutput(map[string]any{
				"service":   info.Service,
				"auth_type": string(info.AuthType),
				"status":    "not_connected",
				"message":   "credential_ref is missing in service auth config",
			})
		}
		return fmt.Errorf("service %q requires %q auth but has no credential_ref", info.Service, info.AuthType)
	}

	vs, err := initVaultStore(cfg)
	if err != nil {
		return err
	}
	defer closeVaultStoreIfPossible(vs)

	if fromStdin || fromFile != "" {
		return linkStoreCredentialFromInput(vs, tenantID, credentialRef, info, fromStdin, fromFile)
	}

	exists, err := vs.Exists(contextBackground(), tenantID, credentialRef)
	if err == nil && exists {
		if outputAsJSON() {
			return printOutput(map[string]any{
				"service":        info.Service,
				"auth_type":      string(info.AuthType),
				"status":         "connected",
				"credential_ref": credentialRef,
			})
		}
		return printOutput(fmt.Sprintf("✓ %s is already connected (credential: %s)", info.Service, credentialRef))
	}

	if err != nil && !errors.Is(err, vault.ErrSecretNotFound) {
		return err
	}

	if statusOnly {
		if outputAsJSON() {
			return printOutput(map[string]any{
				"service":        info.Service,
				"auth_type":      string(info.AuthType),
				"status":         "not_connected",
				"credential_ref": credentialRef,
			})
		}
		return printOutput(fmt.Sprintf("%s is not connected (credential: %s)", info.Service, credentialRef))
	}

	authLabel := "an API key"
	switch info.AuthType {
	case actions.AuthTypeBearer:
		authLabel = "a bearer token"
	case actions.AuthTypeBasic:
		authLabel = "basic auth credentials"
	case actions.AuthTypeHeader:
		authLabel = "a custom header credential"
	case actions.AuthTypeQuery:
		authLabel = "a query parameter credential"
	case actions.AuthTypeAPIKey:
		authLabel = "an API key"
	default:
		authLabel = "a credential"
	}
	instructions := fmt.Sprintf(
		"%s requires %s.\n\nSet it with:\n  printf '%%s' \"YOUR_CREDENTIAL\" | kimbap link %s --stdin\n\nOr store from a file:\n  kimbap link %s --file /path/to/key.txt",
		info.Service,
		authLabel,
		info.Service,
		info.Service,
	)
	if outputAsJSON() {
		return printOutput(map[string]any{
			"service":        info.Service,
			"auth_type":      string(info.AuthType),
			"status":         "not_connected",
			"credential_ref": credentialRef,
			"message":        instructions,
		})
	}
	return printOutput(instructions)
}

func linkRejectStdinFileForOAuth(info linkServiceInfo, oauthStates []connectorStateRow, fromStdin bool, fromFile string) error {
	if !fromStdin && fromFile == "" {
		return nil
	}
	if !linkIsOAuthService(info, oauthStates) {
		return nil
	}
	return fmt.Errorf("service %q uses OAuth authentication. Use 'kimbap auth connect %s' instead of --stdin/--file", info.Service, info.Service)
}

func linkStoreCredentialFromInput(vs vault.Store, tenantID, credentialRef string, info linkServiceInfo, fromStdin bool, fromFile string) error {
	payload, err := readSecretInput(fromFile, fromStdin)
	if err != nil {
		return err
	}
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return fmt.Errorf("credential value is empty after trimming whitespace")
	}
	secretType := linkAuthTypeToSecretType(info.AuthType)
	if _, err := vs.Upsert(contextBackground(), tenantID, credentialRef, secretType, payload, nil, "cli"); err != nil {
		return err
	}
	if outputAsJSON() {
		return printOutput(map[string]any{
			"service":        info.Service,
			"auth_type":      string(info.AuthType),
			"status":         "connected",
			"credential_ref": credentialRef,
		})
	}
	return printOutput(fmt.Sprintf("✓ %s is connected (credential: %s)", info.Service, credentialRef))
}

func linkAuthTypeToSecretType(authType actions.AuthType) vault.SecretType {
	switch authType {
	case actions.AuthTypeBearer:
		return vault.SecretTypeBearerToken
	case actions.AuthTypeBasic:
		return vault.SecretTypePassword
	case actions.AuthTypeAPIKey, actions.AuthTypeHeader, actions.AuthTypeQuery:
		return vault.SecretTypeAPIKey
	default:
		return vault.SecretTypeAPIKey
	}
}

func linkDefaultDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/spf13/cobra"
)

func newAuthRevokeCommand() *cobra.Command {
	var tenant string
	var profile string
	var extras []string
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
			if p, pErr := providers.GetProvider(providerID); pErr == nil {
				providerID = p.ID
			}
			activeTenant := connectorTenant(tenant)
			storeName := connectorStoreName(providerID, profile)

			if !force {
				if ok, promptErr := confirmRevocation(providerID); promptErr != nil {
					return promptErr
				} else if !ok {
					if !outputAsJSON() {
						_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
						return nil
					}
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
			extraValues, parseErr := parseExtrasStrict(extras)
			if parseErr != nil {
				return parseErr
			}
			auditEmitter := initAuthAuditEmitter(cfg)
			defer closeAuditEmitter(auditEmitter)

			provider, providerMetaKnown, revocationResult, prepareErr := prepareRevocationProvider(providerID, extraValues)
			if prepareErr != nil {
				return emitRevokePrepareErrorAudit(auditEmitter, providerID, activeTenant, prepareErr)
			}
			revocationAttempted := false
			if providerMetaKnown && strings.TrimSpace(provider.RevocationEndpoint) != "" && !strings.HasPrefix(revocationResult, "failed: invalid endpoint configuration") && !strings.HasPrefix(revocationResult, "not_supported: unresolved") {
				var revokeToken string
				store, stErr := openConnectorStore(cfg)
				if stErr == nil {
					defer closeConnectorStoreIfPossible(store)
					if stored, _ := store.Get(contextBackground(), activeTenant, storeName); stored != nil {
						decrypted, decryptErr := decryptStoredToken(stored.RefreshToken)
						if decryptErr != nil {
							revocationResult = fmt.Sprintf("failed: %v", decryptErr)
						} else if decrypted == "" {
							decrypted, decryptErr = decryptStoredToken(stored.AccessToken)
							if decryptErr != nil {
								revocationResult = fmt.Sprintf("failed: %v", decryptErr)
							} else {
								revokeToken = decrypted
							}
						} else {
							revokeToken = decrypted
						}
					}
				}
				if !strings.HasPrefix(revocationResult, "failed:") {
					if strings.TrimSpace(revokeToken) == "" {
						revocationResult = "skipped: token unavailable for remote revocation"
					} else {
						revocationAttempted = true
						revokeCreds := resolveOAuthCreds(cfg, providerID)
						revokeErr := callRevocationEndpoint(provider.RevocationEndpoint, revokeCreds.ClientID, revokeCreds.ClientSecret, revokeToken, revokeCreds.AuthMethod)
						if revokeErr != nil {
							revocationResult = fmt.Sprintf("failed: %v", revokeErr)
						} else {
							revocationResult = "success"
						}
					}
				}
			}

			deleted := false
			deleteErr := deleteConnectorState(cfg, activeTenant, storeName)
			if deleteErr == nil {
				deleted = true
			} else if errors.Is(deleteErr, sql.ErrNoRows) {
				deleteErr = nil
			}

			if auditEmitter != nil {
				auditEmitter.RevokeCompleted(contextBackground(), providerID, activeTenant, revocationResult == "success")
			}

			if !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Disconnected %s.\n", providerID)
				_, _ = fmt.Fprintf(os.Stdout, "Remote revocation: %s\n", revocationResult)
				if deleted {
					_, _ = fmt.Fprintln(os.Stdout, "Local token material removed.")
				} else if deleteErr != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Local cleanup failed: %v\n", deleteErr)
				}
				return nil
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
	cmd.Flags().StringVar(&profile, "profile", "default", "connection profile name for multiple accounts per provider")
	cmd.Flags().StringArrayVar(&extras, "extra", nil, "provider-specific key=value pairs for placeholder endpoints")
	cmd.Flags().BoolVar(&force, "force", false, "skip revocation confirmation")
	return cmd
}

func emitRevokePrepareErrorAudit(emitter *connectors.AuditEmitter, providerID, tenantID string, prepareErr error) error {
	if prepareErr == nil {
		return nil
	}
	if emitter != nil {
		emitter.RevokeCompleted(contextBackground(), providerID, tenantID, false)
	}
	return prepareErr
}

func prepareRevocationProvider(providerID string, extras map[string]string) (connectors.ProviderDefinition, bool, string, error) {
	provider, providerErr := providers.GetProvider(providerID)
	if providerErr != nil {
		return connectors.ProviderDefinition{}, false, "not_supported", nil
	}
	if valErr := validateProviderExtraValues(provider, extras); valErr != nil {
		return connectors.ProviderDefinition{}, true, "failed: invalid placeholder value", fmt.Errorf("invalid --extra values: %w", valErr)
	}
	provider = substituteProviderEndpoints(provider, extras)
	if hasUnresolvedPlaceholders(provider) {
		return provider, true, "not_supported: unresolved endpoint placeholders", nil
	}
	if hasPlaceholderInEndpoint(provider.RevocationEndpoint) {
		return provider, true, "not_supported: unresolved endpoint placeholders", nil
	}
	if vErr := validateProviderEndpoints(provider); vErr != nil {
		return provider, true, fmt.Sprintf("failed: invalid endpoint configuration (%v)", vErr), nil
	}
	return provider, true, "not_supported", nil
}

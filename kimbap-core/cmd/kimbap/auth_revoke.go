package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

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
			if p, pErr := providers.GetProvider(providerID); pErr == nil {
				providerID = p.ID
			}
			activeTenant := connectorTenant(tenant)

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
			auditEmitter := initAuthAuditEmitter(cfg)
			defer closeAuditEmitter(auditEmitter)

			provider, providerErr := providers.GetProvider(providerID)
			providerMetaKnown := providerErr == nil
			revocationAttempted := false
			revocationResult := "not_supported"
			if providerMetaKnown && strings.TrimSpace(provider.RevocationEndpoint) != "" {
				var revokeToken string
				store, stErr := openConnectorStore(cfg)
				if stErr == nil {
					if stored, _ := store.Get(contextBackground(), activeTenant, providerID); stored != nil {
						decrypted := decryptStoredToken(stored.RefreshToken)
						if decrypted == "" {
							decrypted = decryptStoredToken(stored.AccessToken)
						}
						revokeToken = decrypted
					}
				}
				if strings.TrimSpace(revokeToken) == "" {
					revocationResult = "skipped: token unavailable for remote revocation"
				} else {
					revocationAttempted = true
					revokeErr := callRevocationEndpoint(provider.RevocationEndpoint, resolveClientID(cfg, providerID), resolveClientSecret(cfg, providerID), revokeToken)
					if revokeErr != nil {
						revocationResult = fmt.Sprintf("failed: %v", revokeErr)
					} else {
						revocationResult = "success"
					}
				}
			}

			deleted := true
			deleteErr := deleteConnectorState(cfg, activeTenant, providerID)
			if deleteErr != nil && !errors.Is(deleteErr, sql.ErrNoRows) {
				deleted = false
			}

			if auditEmitter != nil {
				auditEmitter.RevokeCompleted(contextBackground(), providerID, activeTenant, revocationResult == "success")
			}

			if !outputAsJSON() {
				fmt.Fprintf(os.Stdout, "Disconnected %s.\n", providerID)
				fmt.Fprintf(os.Stdout, "Remote revocation: %s\n", revocationResult)
				if deleted {
					fmt.Fprintln(os.Stdout, "Local token material removed.")
				} else if deleteErr != nil {
					fmt.Fprintf(os.Stdout, "Local cleanup failed: %v\n", deleteErr)
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
	cmd.Flags().BoolVar(&force, "force", false, "skip revocation confirmation")
	return cmd
}

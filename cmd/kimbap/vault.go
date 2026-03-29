package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/spf13/cobra"
)

func newVaultCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Manage encrypted secrets",
	}

	cmd.AddCommand(newVaultSetCommand())
	cmd.AddCommand(newVaultGetCommand())
	cmd.AddCommand(newVaultListCommand())
	cmd.AddCommand(newVaultRotateCommand())
	cmd.AddCommand(newVaultDeleteCommand())

	return cmd
}

func newVaultSetCommand() *cobra.Command {
	var (
		filePath   string
		readStdin  bool
		secretType string
		force      bool
	)
	cmd := &cobra.Command{
		Use:   "set <name> [--force]",
		Short: "Store encrypted secret from file/stdin",
		Example: `  # Store an API key from stdin
  printf '%s' "$API_KEY" | kimbap vault set github.api_key --stdin

  # Store from a file
  kimbap vault set stripe.api_key --file /path/to/key

  # Overwrite an existing secret
  printf '%s' "$NEW_KEY" | kimbap vault set github.api_key --stdin --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			store, err := initVaultStore(cfg)
			if err != nil {
				return err
			}
			defer closeVaultStoreIfPossible(store)
			payload, err := readSecretInput(filePath, readStdin)
			if err != nil {
				return err
			}
			kind, err := parseSecretType(secretType)
			if err != nil {
				return err
			}

			var rec *vault.SecretRecord
			if force {
				rec, err = store.Upsert(contextBackground(), defaultTenantID(), args[0], kind, payload, nil, "cli")
			} else {
				rec, err = store.Create(contextBackground(), defaultTenantID(), args[0], kind, payload, nil, "cli")
			}
			if err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(rec)
			}
			return printOutput(fmt.Sprintf(successCheck()+" %s stored (%s)", rec.Name, rec.Type))
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "read secret from file path")
	cmd.Flags().BoolVar(&readStdin, "stdin", false, "read secret from stdin")
	cmd.Flags().StringVar(&secretType, "type", string(vault.SecretTypeAPIKey), "secret type")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing secret if it already exists")
	return cmd
}

func newVaultGetCommand() *cobra.Command {
	var (
		reveal        bool
		confirmReveal bool
	)
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Show secret metadata or reveal value",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			store, err := initVaultStore(cfg)
			if err != nil {
				return err
			}
			defer closeVaultStoreIfPossible(store)

			if !reveal {
				rec, err := store.GetMeta(contextBackground(), defaultTenantID(), args[0])
				if err != nil {
					return err
				}
				if outputAsJSON() {
					return printOutput(rec)
				}
				fmt.Printf("Name:       %s\n", rec.Name)
				fmt.Printf("Type:       %s\n", rec.Type)
				fmt.Printf("Updated:    %s\n", rec.UpdatedAt.Format(time.RFC3339))
				lastUsed := "-"
				if rec.LastUsedAt != nil {
					lastUsed = rec.LastUsedAt.Format(time.RFC3339)
				}
				fmt.Printf("Last Used:  %s\n", lastUsed)
				fmt.Printf("Version:    %d\n", rec.CurrentVersion)
				return nil
			}

			if reveal && !confirmReveal {
				_, _ = fmt.Fprintln(os.Stderr, "⚠  vault get --reveal outputs the secret plaintext to stdout.")
				_, _ = fmt.Fprintln(os.Stderr, "   Add --confirm-reveal to proceed.")
				return fmt.Errorf("--confirm-reveal is required when using --reveal")
			}

			value, err := store.GetValue(contextBackground(), defaultTenantID(), args[0])
			if err != nil {
				return err
			}
			if err := store.MarkUsed(contextBackground(), defaultTenantID(), args[0]); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "warning: failed to record secret usage for %q: %v\n", args[0], err)
			}
			if outputAsJSON() {
				return printOutput(map[string]any{
					"name":       args[0],
					"revealed":   true,
					"value":      string(value),
					"value_size": len(value),
				})
			}
			_, err = os.Stdout.Write(value)
			return err
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "reveal the secret plaintext (audited operation)")
	cmd.Flags().BoolVar(&confirmReveal, "confirm-reveal", false, "confirm intent to reveal secret plaintext (required with --reveal)")
	return cmd
}

func newVaultListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored secret metadata",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			store, err := initVaultStore(cfg)
			if err != nil {
				return err
			}
			defer closeVaultStoreIfPossible(store)
			records, err := store.List(contextBackground(), defaultTenantID(), vault.ListOptions{})
			if err != nil {
				return err
			}

			if outputAsJSON() {
				return printOutput(records)
			}

			if len(records) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No secrets stored.")
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-28s %-16s %-24s %s\n", "NAME", "TYPE", "UPDATED", "LAST USED")
			for _, rec := range records {
				lastUsed := "-"
				if rec.LastUsedAt != nil {
					lastUsed = rec.LastUsedAt.Format(time.RFC3339)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-28s %-16s %-24s %s\n",
					rec.Name,
					string(rec.Type),
					rec.UpdatedAt.Format(time.RFC3339),
					lastUsed,
				)
			}
			return nil
		},
	}
	return cmd
}

func newVaultRotateCommand() *cobra.Command {
	var (
		filePath  string
		readStdin bool
	)
	cmd := &cobra.Command{
		Use:   "rotate <name>",
		Short: "Rotate an existing secret with a new value",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			store, err := initVaultStore(cfg)
			if err != nil {
				return err
			}
			defer closeVaultStoreIfPossible(store)
			payload, err := readSecretInput(filePath, readStdin)
			if err != nil {
				return err
			}
			rec, err := store.Rotate(contextBackground(), defaultTenantID(), args[0], payload, "cli")
			if err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(rec)
			}
			return printOutput(fmt.Sprintf(successCheck()+" %s rotated (version %d)", rec.Name, rec.CurrentVersion))
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "read new secret from file path")
	cmd.Flags().BoolVar(&readStdin, "stdin", false, "read new secret from stdin")
	return cmd
}

func newVaultDeleteCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <name> [--force]",
		Short: "Delete a secret and all its versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("secret name is required")
			}

			if !force {
				_, _ = fmt.Fprintf(os.Stderr, "⚠  This will permanently delete secret %q and all its versions.\n", name)
				_, _ = fmt.Fprintln(os.Stderr, "   Add --force to confirm deletion.")
				return fmt.Errorf("--force is required to delete a secret")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			store, err := initVaultStore(cfg)
			if err != nil {
				return err
			}
			defer closeVaultStoreIfPossible(store)

			if err := store.Delete(contextBackground(), defaultTenantID(), name); err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(map[string]any{"deleted": true, "name": name})
			}
			return printOutput(fmt.Sprintf(successCheck()+" %s deleted", name))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "confirm intent to permanently delete the secret")
	return cmd
}

func readSecretInput(filePath string, readStdin bool) ([]byte, error) {
	const maxSecretInputBytes int64 = 1 << 20

	fp := strings.TrimSpace(filePath)
	if (fp == "" && !readStdin) || (fp != "" && readStdin) {
		return nil, fmt.Errorf("exactly one input method is required: --file path or --stdin")
	}

	if readStdin {
		payload, err := io.ReadAll(io.LimitReader(os.Stdin, maxSecretInputBytes+1))
		if err != nil {
			return nil, err
		}
		if int64(len(payload)) > maxSecretInputBytes {
			return nil, fmt.Errorf("stdin payload exceeds %d bytes", maxSecretInputBytes)
		}
		if len(payload) == 0 {
			return nil, fmt.Errorf("empty stdin payload")
		}
		return payload, nil
	}

	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	payload, err := io.ReadAll(io.LimitReader(f, maxSecretInputBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maxSecretInputBytes {
		return nil, fmt.Errorf("file payload exceeds %d bytes", maxSecretInputBytes)
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty file payload")
	}
	return payload, nil
}

func parseSecretType(raw string) (vault.SecretType, error) {
	trimmed := vault.SecretType(strings.TrimSpace(raw))
	switch trimmed {
	case vault.SecretTypeAPIKey,
		vault.SecretTypeBearerToken,
		vault.SecretTypeOAuthClient,
		vault.SecretTypePassword,
		vault.SecretTypeRefreshToken,
		vault.SecretTypeCertificate:
		return trimmed, nil
	default:
		return "", fmt.Errorf("unsupported secret type %q. Valid types: api_key, bearer_token, oauth_client, password, refresh_token, certificate", raw)
	}
}

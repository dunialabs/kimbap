package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
)

func newServiceValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <path-to-yaml>",
		Short: "Validate a service manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			manifest, err := services.ParseManifestFile(args[0])
			if err != nil {
				if outputAsJSON() {
					_ = printOutput(map[string]any{"valid": false, "error": err.Error()})
				}
				if errors.Is(err, fs.ErrNotExist) {
					return fmt.Errorf("manifest file %q not found — provide a valid YAML file path", args[0])
				}
				return err
			}
			if outputAsJSON() {
				return printOutput(map[string]any{"valid": true, "name": manifest.Name, "version": manifest.Version})
			}
			if err := printOutput(fmt.Sprintf(successCheck()+" %s (%s) is valid", manifest.Name, manifest.Version)); err != nil {
				return err
			}
			if !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Install: run 'kimbap service install %s'.\n", args[0])
			}
			return nil
		},
	}
	return cmd
}

func newServiceVerifyCommand() *cobra.Command {
	var includeSignatures bool
	var pinnedKeyPath string

	cmd := &cobra.Command{
		Use:   "verify [name]",
		Short: "Verify installed service integrity against lockfile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			installer := installerFromConfig(cfg)

			var pinnedKey ed25519.PublicKey
			if strings.TrimSpace(pinnedKeyPath) != "" {
				keyData, readErr := os.ReadFile(pinnedKeyPath)
				if readErr != nil {
					return fmt.Errorf("read pinned key: %w", readErr)
				}
				keyBytes, decErr := hex.DecodeString(strings.TrimSpace(string(keyData)))
				if decErr != nil || len(keyBytes) != ed25519.PublicKeySize {
					return fmt.Errorf("pinned key must be %d hex-encoded bytes (ed25519 public key)", ed25519.PublicKeySize)
				}
				pinnedKey = ed25519.PublicKey(keyBytes)
			}

			verifyOne := func(name string) (*services.VerifyResult, error) {
				if pinnedKey != nil {
					return installer.VerifyWithKey(name, pinnedKey)
				}
				return installer.Verify(name)
			}

			if len(args) == 1 {
				result, verifyErr := verifyOne(args[0])
				if verifyErr != nil {
					return verifyErr
				}
				if !outputAsJSON() {
					printServiceVerifyResultText(*result, includeSignatures)
				} else if printErr := printOutput(result); printErr != nil {
					return printErr
				}
				if !result.Verified {
					return fmt.Errorf("service %q integrity check failed", args[0])
				}
				if !outputAsJSON() {
					_, _ = fmt.Fprintf(os.Stdout, "Run 'kimbap service update %s' to update to the latest version.\n", args[0])
				}
				return nil
			}

			installed, listErr := installer.List()
			if listErr != nil {
				return listErr
			}
			results := make([]services.VerifyResult, 0, len(installed))
			for _, s := range installed {
				result, verifyErr := verifyOne(s.Manifest.Name)
				if verifyErr != nil {
					return verifyErr
				}
				results = append(results, *result)
			}
			if !outputAsJSON() {
				for _, result := range results {
					printServiceVerifyResultText(result, includeSignatures)
				}
			} else if printErr := printOutput(results); printErr != nil {
				return printErr
			}
			allVerified := true
			for _, result := range results {
				if !result.Verified {
					allVerified = false
				}
			}
			if !allVerified {
				return fmt.Errorf("one or more services failed integrity check")
			}
			if !outputAsJSON() && len(results) > 0 {
				_, _ = fmt.Fprintln(os.Stdout, "Run 'kimbap service outdated' to check for available updates.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&includeSignatures, "signatures", false, "include lockfile signature status in text output")
	cmd.Flags().StringVar(&pinnedKeyPath, "key", "", "path to pinned ed25519 public key (hex) for signature verification (ignores embedded key)")
	return cmd
}

func newServiceSignCommand() *cobra.Command {
	var keyPath string

	cmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign lockfile entries with ed25519 key",
		Long:  "Signs all service digest entries in the lockfile for supply-chain integrity verification.",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			keyData, err := os.ReadFile(keyPath)
			if err != nil {
				return fmt.Errorf("read signing key: %w", err)
			}

			seedBytes, err := hex.DecodeString(strings.TrimSpace(string(keyData)))
			if err != nil || len(seedBytes) != ed25519.SeedSize {
				return fmt.Errorf("signing key must be %d hex-encoded bytes (ed25519 seed)", ed25519.SeedSize)
			}

			privateKey := ed25519.NewKeyFromSeed(seedBytes)
			installer := installerFromConfig(cfg)

			if err := installer.Sign(privateKey); err != nil {
				return err
			}

			if outputAsJSON() {
				return printOutput(map[string]any{"signed": true})
			}
			if err := printOutput(successCheck() + " lockfile signed"); err != nil {
				return err
			}
			if !outputAsJSON() {
				_, _ = fmt.Fprintln(os.Stdout, "Run 'kimbap service verify' to confirm integrity.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&keyPath, "key", "", "path to ed25519 private key (hex-encoded seed)")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}

func printServiceVerifyResultText(result services.VerifyResult, includeSignatures bool) {
	status := "NOT VERIFIED"
	if result.Verified {
		status = "VERIFIED"
	}
	if isColorStdout() {
		if result.Verified {
			status = "\x1b[32m" + status + "\x1b[0m"
		} else {
			status = "\x1b[31m" + status + "\x1b[0m"
		}
	}

	locked := "unlocked"
	if result.Locked {
		locked = "locked"
	}

	line := fmt.Sprintf("%s: %s (%s)", result.Name, status, locked)
	if includeSignatures && result.Signed {
		sigStatus := "invalid"
		if result.SignatureValid {
			sigStatus = "valid"
		}
		line += fmt.Sprintf(" [signature: %s]", sigStatus)
	}

	_, _ = fmt.Fprintln(os.Stdout, line)
}

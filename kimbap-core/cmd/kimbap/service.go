package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/services"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage service manifests",
	}

	cmd.AddCommand(newServiceInstallCommand())
	cmd.AddCommand(newServiceListCommand())
	cmd.AddCommand(newServiceRemoveCommand())
	cmd.AddCommand(newServiceVerifyCommand())
	cmd.AddCommand(newServiceSignCommand())
	cmd.AddCommand(newServiceValidateCommand())
	cmd.AddCommand(newServiceGenerateCommand())
	cmd.AddCommand(newServiceExportSkillMDCommand())

	return cmd
}

func newServiceInstallCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "install <path-to-yaml> [--force]",
		Short: "Install a local service manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			manifest, err := services.ParseManifestFile(args[0])
			if err != nil {
				return err
			}

			installed, err := installerFromConfig(cfg).InstallWithForce(manifest, args[0], force)
			if err != nil {
				return err
			}
			return printOutput(installed)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing service if already installed")
	return cmd
}

func newServiceListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed services",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			installed, err := installerFromConfig(cfg).List()
			if err != nil {
				return err
			}
			return printOutput(installed)
		},
	}
	return cmd
}

func newServiceRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove installed service by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			if err := installerFromConfig(cfg).Remove(args[0]); err != nil {
				return err
			}
			return printOutput(map[string]any{"removed": true, "name": args[0]})
		},
	}
	return cmd
}

func newServiceValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <path-to-yaml>",
		Short: "Validate a service manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			manifest, err := services.ParseManifestFile(args[0])
			if err != nil {
				return err
			}
			if errs := services.ValidateManifest(manifest); len(errs) > 0 {
				return fmt.Errorf("manifest invalid: %v", errs)
			}
			return printOutput(map[string]any{"valid": true, "name": manifest.Name, "version": manifest.Version})
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
				if includeSignatures && !outputAsJSON() {
					printServiceVerifyResultText(*result, true)
					return nil
				}
				return printOutput(result)
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
			if includeSignatures && !outputAsJSON() {
				for _, result := range results {
					printServiceVerifyResultText(result, true)
				}
				return nil
			}
			return printOutput(results)
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

			return printOutput(map[string]any{"signed": true})
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

func newServiceGenerateCommand() *cobra.Command {
	var openapiSource string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "generate --openapi <path-or-url> [--output file.yaml]",
		Short: "Generate a service manifest from OpenAPI 3.x",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				manifest *services.ServiceManifest
				err      error
			)

			if isServiceHTTPURL(openapiSource) {
				manifest, err = services.GenerateFromOpenAPIURL(openapiSource)
			} else {
				manifest, err = services.GenerateFromOpenAPIFile(openapiSource)
			}
			if err != nil {
				return err
			}

			encoded, err := yaml.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("marshal generated manifest as YAML: %w", err)
			}

			if strings.TrimSpace(outputPath) == "" {
				fmt.Print(string(encoded))
				return nil
			}

			if err := os.WriteFile(outputPath, encoded, 0o644); err != nil {
				return fmt.Errorf("write generated manifest file: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&openapiSource, "openapi", "", "OpenAPI 3.x spec path or URL")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output file path (defaults to stdout)")
	_ = cmd.MarkFlagRequired("openapi")

	return cmd
}

func newServiceExportSkillMDCommand() *cobra.Command {
	var outputPath string
	var outputDir string
	var exportPack bool

	cmd := &cobra.Command{
		Use:   "export-skillmd <name> [--output file] [--dir directory]",
		Short: "Export installed service as SKILL.md (Agent Skills open standard)",
		Long:  "Generate a SKILL.md file compatible with Claude Code, OpenAI Codex, GitHub Copilot, and other AI agents.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("dir") && cmd.Flags().Changed("output") {
				return fmt.Errorf("--dir and --output are mutually exclusive")
			}
			if exportPack && strings.TrimSpace(outputDir) == "" {
				return fmt.Errorf("--pack requires --dir")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			installed, err := installerFromConfig(cfg).Get(args[0])
			if err != nil {
				return fmt.Errorf("service %q not found: %w", args[0], err)
			}

			content, err := services.GenerateSkillMD(&installed.Manifest)
			if err != nil {
				return err
			}

			if strings.TrimSpace(outputDir) != "" {
				serviceDir := filepath.Join(outputDir, installed.Manifest.Name)
				if exportPack {
					pack, packErr := services.GenerateSkillPack(&installed.Manifest)
					if packErr != nil {
						return packErr
					}
					writtenFiles, writeErr := writeSkillPackDir(serviceDir, pack)
					if writeErr != nil {
						return writeErr
					}
					sort.Strings(writtenFiles)
					return printOutput(map[string]any{"exported": true, "pack": true, "files": writtenFiles})
				}
				if _, writeErr := writeSkillPackDir(serviceDir, map[string]string{"SKILL.md": content}); writeErr != nil {
					return writeErr
				}
				outPath := filepath.Join(serviceDir, "SKILL.md")
				return printOutput(map[string]any{"exported": true, "path": outPath})
			}

			if strings.TrimSpace(outputPath) != "" {
				if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("write SKILL.md: %w", err)
				}
				return printOutput(map[string]any{"exported": true, "path": outputPath})
			}

			fmt.Print(content)
			return nil
		},
	}

	cmd.Flags().StringVar(&outputPath, "output", "", "output file path")
	cmd.Flags().StringVar(&outputDir, "dir", "", "output directory (creates name/SKILL.md structure)")
	cmd.Flags().BoolVar(&exportPack, "pack", false, "export as folder-based skill pack")

	return cmd
}

func writeSkillPackDir(serviceDir string, pack map[string]string) ([]string, error) {
	names := make([]string, 0, len(pack))
	for filename := range pack {
		if err := validateExportPackFileName(filename); err != nil {
			return nil, err
		}
		names = append(names, filename)
	}
	sort.Strings(names)

	parentDir := filepath.Dir(serviceDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("create parent directory: %w", err)
	}
	tmpDir, err := os.MkdirTemp(parentDir, ".kimbap-export-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	staged := false
	defer func() {
		if !staged {
			os.RemoveAll(tmpDir)
		}
	}()

	writtenFiles := make([]string, 0, len(names))
	for _, filename := range names {
		if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(pack[filename]), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", filename, err)
		}
		writtenFiles = append(writtenFiles, filepath.Join(serviceDir, filename))
	}

	oldDir := serviceDir + ".old"
	hasOld := false
	if _, statErr := os.Stat(serviceDir); statErr == nil {
		if err := os.Rename(serviceDir, oldDir); err != nil {
			return nil, fmt.Errorf("backup existing export dir: %w", err)
		}
		hasOld = true
	}
	if err := os.Rename(tmpDir, serviceDir); err != nil {
		if hasOld {
			_ = os.Rename(oldDir, serviceDir)
		}
		return nil, fmt.Errorf("promote temp to target: %w", err)
	}
	if hasOld {
		os.RemoveAll(oldDir)
	}
	staged = true
	return writtenFiles, nil
}

func validateExportPackFileName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("pack filename must be non-empty")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("pack filename %q must be relative", name)
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("pack filename %q must not escape export directory", name)
	}
	if strings.Contains(clean, string(filepath.Separator)) {
		return fmt.Errorf("pack filename %q must not contain path separators", name)
	}
	return nil
}

func isServiceHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

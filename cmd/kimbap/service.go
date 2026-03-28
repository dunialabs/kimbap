package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/registry"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/skills"
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
	cmd.AddCommand(newServiceEnableCommand())
	cmd.AddCommand(newServiceDisableCommand())
	cmd.AddCommand(newServiceRemoveCommand())
	cmd.AddCommand(newServiceUpdateCommand())
	cmd.AddCommand(newServiceOutdatedCommand())
	cmd.AddCommand(newServiceVerifyCommand())
	cmd.AddCommand(newServiceSignCommand())
	cmd.AddCommand(newServiceValidateCommand())
	cmd.AddCommand(newServiceGenerateCommand())
	cmd.AddCommand(newServiceExportAgentSkillCommand())

	return cmd
}

func newServiceInstallCommand() *cobra.Command {
	var force bool
	var noActivate bool
	cmd := &cobra.Command{
		Use:   "install <name|path-to-yaml|url> [--force]",
		Short: "Install a service manifest",
		Example: `  # Install an official service by name
  kimbap service install github

  # Install from a local YAML file
  kimbap service install ./my-service.yaml

  # Install from a URL
  kimbap service install https://example.com/service.yaml

  # Reinstall (overwrite existing)
  kimbap service install github --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			manifest, source, err := resolveServiceInstallSource(args[0])
			if err != nil {
				return err
			}

			installed, err := installerFromConfig(cfg).InstallWithForceAndActivation(manifest, source, force, !noActivate)
			if err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(installed)
			}
			status := "enabled"
			if !installed.Enabled {
				status = "disabled"
			}
			return printOutput(fmt.Sprintf("✓ %s (%s) installed [%s]", installed.Manifest.Name, installed.Manifest.Version, status))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing service if already installed")
	cmd.Flags().BoolVar(&noActivate, "no-activate", false, "install service as disabled")
	return cmd
}

func newServiceListCommand() *cobra.Command {
	var available bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed services",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			installer := installerFromConfig(cfg)
			installed, err := installer.List()
			if err != nil {
				return err
			}

			if available {
				official, listErr := skills.List()
				if listErr != nil {
					return fmt.Errorf("list official services: %w", listErr)
				}

				installedByName := make(map[string]services.InstalledService, len(installed))
				for _, svc := range installed {
					installedByName[svc.Manifest.Name] = svc
				}

				rows := make([]map[string]any, 0, len(official))
				for _, name := range official {
					row := map[string]any{
						"name":      name,
						"official":  true,
						"installed": false,
						"enabled":   false,
						"status":    "not-installed",
					}
					if svc, ok := installedByName[name]; ok {
						row["installed"] = true
						row["enabled"] = svc.Enabled
						if svc.Enabled {
							row["status"] = "enabled"
						} else {
							row["status"] = "disabled"
						}
					}
					rows = append(rows, row)
				}
				return printOutput(rows)
			}

			return printOutput(installed)
		},
	}
	cmd.Flags().BoolVar(&available, "available", false, "list all official services with installed/enabled status")
	return cmd
}

func newServiceEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable an installed service",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			if err := installerFromConfig(cfg).Enable(args[0]); err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(map[string]any{"enabled": true, "name": args[0]})
			}
			return printOutput(fmt.Sprintf("✓ %s enabled", args[0]))
		},
	}
	return cmd
}

func newServiceDisableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <name>",
		Short: "Disable an installed service",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			if err := installerFromConfig(cfg).Disable(args[0]); err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(map[string]any{"enabled": false, "name": args[0]})
			}
			return printOutput(fmt.Sprintf("✓ %s disabled", args[0]))
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
			if outputAsJSON() {
				return printOutput(map[string]any{"removed": true, "name": args[0]})
			}
			return printOutput(fmt.Sprintf("✓ %s removed", args[0]))
		},
	}
	return cmd
}

func newServiceUpdateCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update an installed service to the latest version",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			installer := installerFromConfig(cfg)
			installed, err := installer.Get(name)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return fmt.Errorf("service %q not found: run 'kimbap service list' to see installed services", name)
				}
				return fmt.Errorf("get service %q: %w", name, err)
			}

			source := strings.TrimSpace(installed.Source)
			manifest, newSource, resolveErr := resolveServiceInstallSource(sourceToInstallArg(source))
			if resolveErr != nil {
				return fmt.Errorf("resolve update source for %q (%s): %w", name, source, resolveErr)
			}

			if strings.TrimSpace(manifest.Name) != name {
				return fmt.Errorf("update refused: fetched manifest has name %q but expected %q — source may have changed", manifest.Name, name)
			}

			if !force && strings.TrimSpace(manifest.Version) == strings.TrimSpace(installed.Manifest.Version) {
				if outputAsJSON() {
					return printOutput(map[string]any{
						"updated": false,
						"name":    installed.Manifest.Name,
						"version": installed.Manifest.Version,
						"source":  source,
						"message": "already up to date (use --force to reinstall)",
					})
				}
				return printOutput(fmt.Sprintf("✓ %s (%s) already up to date", installed.Manifest.Name, installed.Manifest.Version))
			}

			updated, installErr := installer.InstallWithForceAndActivation(manifest, newSource, true, installed.Enabled)
			if installErr != nil {
				return installErr
			}

			if outputAsJSON() {
				return printOutput(map[string]any{
					"updated": true,
					"name":    updated.Manifest.Name,
					"version": updated.Manifest.Version,
					"source":  updated.Source,
				})
			}
			return printOutput(fmt.Sprintf("✓ %s updated to %s", updated.Manifest.Name, updated.Manifest.Version))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "force update even if version is unchanged")
	return cmd
}

func newServiceOutdatedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "List outdated services from official catalog",
		Long:  "Lists installed services from the official catalog when installed version differs from the embedded official catalog version.",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			installer := installerFromConfig(cfg)
			installed, err := installer.List()
			if err != nil {
				return err
			}

			official := registry.NewEmbeddedRegistry()

			type outdatedEntry struct {
				Name             string `json:"name"`
				InstalledVersion string `json:"installed_version"`
				LatestVersion    string `json:"latest_version"`
				Source           string `json:"source"`
			}

			entries := make([]outdatedEntry, 0)
			for _, svc := range installed {
				source := strings.TrimSpace(svc.Source)
				if !strings.HasPrefix(source, "official:") {
					continue
				}

				officialName := strings.TrimSpace(strings.TrimPrefix(source, "official:"))
				if officialName == "" {
					officialName = svc.Manifest.Name
				}

				manifest, _, resolveErr := official.Resolve(contextBackground(), officialName)
				if resolveErr != nil {
					continue
				}

				if strings.TrimSpace(svc.Manifest.Version) == strings.TrimSpace(manifest.Version) {
					continue
				}

				entries = append(entries, outdatedEntry{
					Name:             svc.Manifest.Name,
					InstalledVersion: svc.Manifest.Version,
					LatestVersion:    manifest.Version,
					Source:           source,
				})
			}

			if outputAsJSON() {
				return printOutput(entries)
			}

			if len(entries) == 0 {
				fmt.Println("No outdated official services found.")
				return nil
			}

			fmt.Printf("%-30s %-12s %-12s %s\n", "SERVICE", "INSTALLED", "LATEST", "SOURCE")
			for _, e := range entries {
				fmt.Printf("%-30s %-12s %-12s %s\n", e.Name, e.InstalledVersion, e.LatestVersion, e.Source)
			}
			fmt.Printf("\nRun 'kimbap service update <name>' to update a service.\n")
			return nil
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
			if outputAsJSON() {
				return printOutput(map[string]any{"valid": true, "name": manifest.Name, "version": manifest.Version})
			}
			return printOutput(fmt.Sprintf("✓ %s (%s) is valid", manifest.Name, manifest.Version))
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
				} else {
					if printErr := printOutput(result); printErr != nil {
						return printErr
					}
				}
				if !result.Verified {
					return fmt.Errorf("service %q integrity check failed", args[0])
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
			} else {
				if printErr := printOutput(results); printErr != nil {
					return printErr
				}
			}
			for _, result := range results {
				if !result.Verified {
					return fmt.Errorf("one or more services failed integrity check")
				}
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

func newServiceExportAgentSkillCommand() *cobra.Command {
	var outputPath string
	var outputDir string
	var exportLegacy bool

	cmd := &cobra.Command{
		Use:   "export-agent-skill <name> [--output file] [--dir directory]",
		Short: "Export installed service as agent SKILL.md",
		Long:  "Generate a SKILL.md file compatible with Claude Code, OpenAI Codex, GitHub Copilot, and other AI agents.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("dir") && cmd.Flags().Changed("output") {
				return fmt.Errorf("--dir and --output are mutually exclusive")
			}
			if exportLegacy && strings.TrimSpace(outputDir) == "" {
				return fmt.Errorf("--legacy requires --dir")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			installed, err := installerFromConfig(cfg).Get(args[0])
			if err != nil {
				return fmt.Errorf("service %q not found: %w", args[0], err)
			}
			if !installed.Enabled {
				return fmt.Errorf("service %q is installed but disabled; enable it before exporting agent skill", args[0])
			}

			if strings.TrimSpace(outputDir) != "" {
				serviceDir := filepath.Join(outputDir, installed.Manifest.Name)
				if !exportLegacy {
					pack, packErr := services.GenerateAgentSkillPack(&installed.Manifest, services.WithSource(installed.Source))
					if packErr != nil {
						return packErr
					}
					writtenFiles, writeErr := writeAgentSkillPackDir(serviceDir, pack)
					if writeErr != nil {
						return writeErr
					}
					sort.Strings(writtenFiles)
					return printOutput(map[string]any{"exported": true, "pack": true, "files": writtenFiles})
				}
				content, legacyErr := services.GenerateAgentSkillMD(&installed.Manifest, services.WithSource(installed.Source))
				if legacyErr != nil {
					return legacyErr
				}
				if _, writeErr := writeAgentSkillPackDir(serviceDir, map[string]string{"SKILL.md": content}); writeErr != nil {
					return writeErr
				}
				outPath := filepath.Join(serviceDir, "SKILL.md")
				return printOutput(map[string]any{"exported": true, "path": outPath})
			}

			content, err := services.GenerateAgentSkillMD(&installed.Manifest, services.WithSource(installed.Source))
			if err != nil {
				return err
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
	cmd.Flags().StringVar(&outputDir, "dir", "", "output directory (creates name/ folder pack with SKILL.md, GOTCHAS.md, RECIPES.md)")
	cmd.Flags().BoolVar(&exportLegacy, "legacy", false, "export as single SKILL.md file (legacy mode) instead of folder pack")

	return cmd
}

func writeAgentSkillPackDir(serviceDir string, pack map[string]string) ([]string, error) {
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
		if err := os.RemoveAll(oldDir); err != nil {
			return nil, fmt.Errorf("remove stale backup export dir: %w", err)
		}
		if err := os.Rename(serviceDir, oldDir); err != nil {
			return nil, fmt.Errorf("backup existing export dir: %w", err)
		}
		hasOld = true
	}
	if err := os.Rename(tmpDir, serviceDir); err != nil {
		if hasOld {
			if restoreErr := os.Rename(oldDir, serviceDir); restoreErr != nil {
				return nil, fmt.Errorf("promote temp to target: %w (restore from backup %q failed: %v)", err, oldDir, restoreErr)
			}
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
	if strings.ContainsAny(clean, `/\`) {
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

func resolveServiceInstallSource(arg string) (*services.ServiceManifest, string, error) {
	trimmed := strings.TrimSpace(arg)
	if trimmed == "" {
		return nil, "", fmt.Errorf("service source is required")
	}

	if strings.HasPrefix(trimmed, "http://") {
		return nil, "", fmt.Errorf("insecure URL %q rejected: use https:// to install service manifests", trimmed)
	}
	if strings.HasPrefix(trimmed, "https://") {
		manifest, err := parseServiceManifestURL(trimmed)
		if err != nil {
			return nil, "", err
		}
		return manifest, "remote:" + trimmed, nil
	}

	if strings.HasPrefix(trimmed, "github:") {
		owner, repo, serviceName, subdir, parseErr := registry.ParseGitHubRef(trimmed)
		if parseErr != nil {
			return nil, "", parseErr
		}
		reg := registry.NewGitHubRegistry(owner, repo, "", subdir)
		ctx, cancel := context.WithTimeout(contextBackground(), 30*time.Second)
		defer cancel()
		manifest, source, resolveErr := reg.Resolve(ctx, serviceName)
		if resolveErr != nil {
			return nil, "", fmt.Errorf("fetch from GitHub %q: %w", trimmed, resolveErr)
		}
		if manifest.Name != serviceName {
			return nil, "", fmt.Errorf("manifest name %q does not match requested service %q", manifest.Name, serviceName)
		}
		return manifest, source, nil
	}

	if stat, err := os.Stat(trimmed); err == nil {
		if stat.IsDir() {
			return nil, "", fmt.Errorf("service source %q is a directory. Pass a YAML file path, URL, or official service name", trimmed)
		}
		absPath, absErr := filepath.Abs(trimmed)
		if absErr != nil {
			absPath = trimmed
		}
		manifest, parseErr := services.ParseManifestFile(absPath)
		if parseErr != nil {
			return nil, "", parseErr
		}
		return manifest, "local:" + absPath, nil
	} else if !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("stat service source %q: %w", trimmed, err)
	}

	if err := services.ValidateServiceName(trimmed); err != nil {
		return nil, "", err
	}

	data, err := skills.Get(trimmed)
	if err == nil {
		manifest, parseErr := services.ParseManifest(data)
		if parseErr != nil {
			return nil, "", fmt.Errorf("parse official service %q: %w", trimmed, parseErr)
		}
		return manifest, "official:" + trimmed, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, "", fmt.Errorf("load official service %q: %w", trimmed, err)
	}

	cfg, cfgErr := loadAppConfig()
	if cfgErr == nil {
		officialURL := strings.TrimSpace(cfg.Services.Official)
		if officialURL != "" && officialURL != "https://services.kimbap.ai" {
			remoteReg := registry.NewRemoteRegistry("official", officialURL)
			ctx, cancel := context.WithTimeout(contextBackground(), 30*time.Second)
			defer cancel()
			manifest, source, resolveErr := remoteReg.Resolve(ctx, trimmed)
			if resolveErr == nil {
				return manifest, source, nil
			}
			var notFound *registry.ErrNotFound
			if !errors.As(resolveErr, &notFound) {
				return nil, "", fmt.Errorf("load official service %q from %s: %w", trimmed, officialURL, resolveErr)
			}
		}
	}

	officialNames, _ := skills.List()
	hint := "Run 'kimbap service list --available' to see all official services."
	if suggestion := didYouMean(trimmed, officialNames); suggestion != "" {
		hint = fmt.Sprintf("Did you mean %q? Run 'kimbap service list --available' to see all official services.", suggestion)
	}
	return nil, "", fmt.Errorf("service %q not found in official catalog. %s", trimmed, hint)
}

func sourceToInstallArg(source string) string {
	trimmed := strings.TrimSpace(source)
	if strings.HasPrefix(trimmed, "official:") {
		return strings.TrimPrefix(trimmed, "official:")
	}
	if strings.HasPrefix(trimmed, "remote:") {
		return strings.TrimPrefix(trimmed, "remote:")
	}
	if strings.HasPrefix(trimmed, "local:") {
		return strings.TrimPrefix(trimmed, "local:")
	}
	if strings.HasPrefix(trimmed, "github:") {
		rest := strings.TrimPrefix(trimmed, "github:")
		base, serviceName, ok := strings.Cut(rest, ":")
		if !ok || strings.TrimSpace(serviceName) == "" {
			return trimmed
		}
		return "github:" + strings.Trim(base, "/") + "/" + strings.TrimSpace(serviceName)
	}
	return trimmed
}

func parseServiceManifestURL(serviceURL string) (*services.ServiceManifest, error) {
	ctx, cancel := context.WithTimeout(contextBackground(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serviceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %q: %w", serviceURL, err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if r.URL.Scheme != "https" {
				return fmt.Errorf("redirect to non-https URL %q rejected", r.URL)
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch service manifest from %q: %w", serviceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch service manifest from %q: got HTTP %d. Check the URL or use a local file path", serviceURL, resp.StatusCode)
	}

	const maxManifestBytes = 1 << 20 // 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read service manifest from %q: %w", serviceURL, err)
	}
	if int64(len(body)) > maxManifestBytes {
		return nil, fmt.Errorf("service manifest from %q exceeds %d bytes", serviceURL, maxManifestBytes)
	}

	manifest, err := services.ParseManifest(body)
	if err != nil {
		return nil, fmt.Errorf("parse service manifest from %q: %w", serviceURL, err)
	}

	return manifest, nil
}

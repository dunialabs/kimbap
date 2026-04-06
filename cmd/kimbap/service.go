package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/agents"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/dunialabs/kimbap/internal/registry"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/dunialabs/kimbap/services/catalog"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage service manifests",
	}

	cmd.AddCommand(newServiceInstallCommand())
	cmd.AddCommand(newServiceListCommand())
	cmd.AddCommand(newServiceSearchCommand())
	cmd.AddCommand(newServiceDescribeCommand())
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
	var noShortcuts bool
	cmd := &cobra.Command{
		Use:   "install <name|path-to-yaml|url> [--force]",
		Short: "Install a service manifest",
		Example: `  # Install a catalog service by name
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
			installer := installerFromConfig(cfg)

			manifest, source, err := resolveServiceInstallSource(commandContext(cmd), args[0])
			if err != nil {
				return err
			}

			installed, err := installer.InstallWithForceAndActivation(manifest, source, force, !noActivate)
			if err != nil {
				return err
			}
			if installed.Enabled && strings.HasPrefix(strings.TrimSpace(source), "registry:") {
				if credErr := ensureInstalledServiceCredentials(commandContext(cmd), cfg, &installed.Manifest); credErr != nil {
					return credErr
				}
			}

			autoAlias := ""
			autoAliasCreated := false
			actionAliasesCreated := make([]string, 0)

			setupShortcuts := !noShortcuts && installed.Enabled

			if setupShortcuts {
				shortcutResult := applyInstalledShortcuts(cfg, installer, &installed.Manifest, "auto")
				autoAlias = shortcutResult.AutoAlias
				autoAliasCreated = shortcutResult.AutoAliasCreated
				actionAliasesCreated = shortcutResult.ActionAliasesCreated
			}

			if outputAsJSON() {
				maybePrintAgentSyncHint(opts.format)
				return printOutput(map[string]any{
					"service":                installed,
					"auto_alias":             autoAlias,
					"auto_alias_created":     autoAliasCreated,
					"action_aliases_created": actionAliasesCreated,
				})
			}
			status := "enabled"
			if !installed.Enabled {
				status = "disabled"
			}
			msg := fmt.Sprintf(successCheck()+" %s (%s) installed [%s]", installed.Manifest.Name, installed.Manifest.Version, status)
			if autoAlias != "" {
				msg += fmt.Sprintf(" (alias: %s)", autoAlias)
			}
			if len(actionAliasesCreated) > 0 {
				msg += fmt.Sprintf(" (action aliases: %s)", formatActionAliasesSummary(actionAliasesCreated))
			}
			maybePrintAgentSyncHint(opts.format)
			return printOutput(msg)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing service if already installed")
	cmd.Flags().BoolVar(&noActivate, "no-activate", false, "install service as disabled")
	cmd.Flags().BoolVar(&noShortcuts, "no-shortcuts", false, "skip automatic shortcut alias setup during install")
	return cmd
}

func serviceManifestRequiresCredentials(manifest *services.ServiceManifest) bool {
	if manifest == nil {
		return false
	}
	if authTypeRequiresCredential(manifest.Auth.Type) {
		return true
	}
	for _, action := range manifest.Actions {
		if action.Auth != nil && authTypeRequiresCredential(action.Auth.Type) {
			return true
		}
	}
	return false
}

func authTypeRequiresCredential(raw string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	return trimmed != "" && trimmed != string(actions.AuthTypeNone)
}

func ensureInstalledServiceCredentials(ctx context.Context, cfg *config.KimbapConfig, manifest *services.ServiceManifest) error {
	if cfg == nil || manifest == nil || !serviceManifestRequiresCredentials(manifest) {
		return nil
	}
	if !canPromptInTTY() {
		return nil
	}

	serviceName := strings.TrimSpace(manifest.Name)
	if serviceName == "" {
		return nil
	}

	servicesByName, err := loadLinkServices(cfg)
	if err != nil {
		return fmt.Errorf("load installed services for credential setup: %w", err)
	}

	info, ok := servicesByName[strings.ToLower(serviceName)]
	if !ok {
		info = linkServiceInfo{
			Service:       serviceName,
			AuthType:      actions.AuthType(strings.ToLower(strings.TrimSpace(manifest.Auth.Type))),
			CredentialRef: strings.TrimSpace(manifest.Auth.CredentialRef),
			OAuthProvider: linkResolveOAuthProvider(serviceName),
		}
	}

	tenantID := defaultTenantID()
	oauthStates, oauthErr := listConnectorStates(ctx, cfg, tenantID)
	if oauthErr != nil {
		oauthStates = nil
	}

	if linkIsOAuthService(info, oauthStates) {
		providerID := strings.TrimSpace(info.OAuthProvider)
		if providerID == "" {
			providerID = serviceName
		}
		return runAuthConnect(
			cfg,
			providerID,
			tenantID,
			"auto",
			"",
			"auto",
			false,
			0,
			5*time.Minute,
			"",
			string(connectors.ScopeUser),
			"default",
			false,
			nil,
		)
	}

	credentialRef := strings.TrimSpace(info.CredentialRef)
	if credentialRef == "" {
		return fmt.Errorf("service %q requires authentication but has no credential_ref", serviceName)
	}

	vs, err := initVaultStore(cfg)
	if err != nil {
		return err
	}
	defer closeVaultStoreIfPossible(vs)

	exists, existsErr := vs.Exists(ctx, tenantID, credentialRef)
	if existsErr == nil && exists {
		return nil
	}
	if existsErr != nil && !errors.Is(existsErr, vault.ErrSecretNotFound) {
		return existsErr
	}
	authLabel := credentialPromptLabel(info.AuthType)
	_, _ = fmt.Fprintf(os.Stderr, "Enter %s for %s: ", authLabel, serviceName)
	payload, readErr := term.ReadPassword(int(os.Stdin.Fd()))
	if readErr != nil {
		return fmt.Errorf("read credential for %s: %w", serviceName, readErr)
	}
	_, _ = fmt.Fprintln(os.Stderr)
	payload = []byte(strings.TrimSpace(string(payload)))
	if len(payload) == 0 {
		return fmt.Errorf("credential value is empty")
	}

	secretType := linkAuthTypeToSecretType(info.AuthType)
	if _, upsertErr := vs.Upsert(ctx, tenantID, credentialRef, secretType, payload, nil, "cli"); upsertErr != nil {
		return upsertErr
	}
	return nil
}

func credentialPromptLabel(authType actions.AuthType) string {
	switch authType {
	case actions.AuthTypeBearer:
		return "a bearer token"
	case actions.AuthTypeBasic:
		return "basic auth credentials"
	case actions.AuthTypeHeader:
		return "a custom header credential"
	case actions.AuthTypeQuery:
		return "a query parameter credential"
	case actions.AuthTypeAPIKey:
		return "an API key"
	default:
		return "a credential"
	}
}

type serviceShortcutResult struct {
	AutoAlias            string
	AutoAliasCreated     bool
	ActionAliasesCreated []string
}

func runInstalledShortcutSetup(cfg *config.KimbapConfig, installer *services.LocalInstaller, manifest *services.ServiceManifest) (serviceShortcutResult, []string, error, error) {
	result := serviceShortcutResult{ActionAliasesCreated: make([]string, 0)}
	if cfg == nil || installer == nil || manifest == nil {
		return result, nil, nil, nil
	}

	alias, created, aliasErr := ensureInstalledServiceAlias(cfg, installer, manifest)
	if aliasErr == nil {
		result.AutoAlias = alias
		result.AutoAliasCreated = created
	}

	createdActionAliases, skippedActionAliases, actionAliasErr := ensureInstalledActionAliases(cfg, installer, manifest)
	if actionAliasErr == nil {
		result.ActionAliasesCreated = createdActionAliases
	}

	return result, skippedActionAliases, aliasErr, actionAliasErr
}

func applyInstalledShortcuts(cfg *config.KimbapConfig, installer *services.LocalInstaller, manifest *services.ServiceManifest, mode string) serviceShortcutResult {
	result := serviceShortcutResult{ActionAliasesCreated: make([]string, 0)}
	if cfg == nil || installer == nil || manifest == nil {
		return result
	}
	name := strings.TrimSpace(manifest.Name)
	if name == "" {
		return result
	}

	serviceWarn := "auto alias setup skipped"
	actionWarn := "action alias setup skipped"
	if strings.EqualFold(strings.TrimSpace(mode), "enabled") {
		serviceWarn = "enabled service alias setup skipped"
		actionWarn = "enabled action alias setup skipped"
	}

	shortcutResult, skipped, aliasErr, actionAliasErr := runInstalledShortcutSetup(cfg, installer, manifest)
	result = shortcutResult

	if aliasErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %s for %s: %v\n", serviceWarn, name, aliasErr)
	}

	if actionAliasErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %s for %s: %v\n", actionWarn, name, actionAliasErr)
	} else {
		if len(skipped) > 0 {
			_, _ = fmt.Fprintf(os.Stderr, "warning: skipped action aliases for %s: %s\n", name, strings.Join(skipped, ", "))
		}
	}

	return result
}

func newServiceListCommand() *cobra.Command {
	var available bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed services",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}

			if available {
				summaries, summaryErr := loadCatalogServiceSummaries(cfg)
				if summaryErr != nil {
					return summaryErr
				}
				rows := catalogListRows(summaries)
				if outputAsJSON() {
					return printOutput(rows)
				}
				useColor := isColorStdout()
				fmt.Printf("%-20s %-14s %s\n", "NAME", "STATUS", "SHORTCUTS")
				for _, row := range rows {
					shortcutDisplay := "-"
					if len(row.Shortcuts) > 0 {
						shortcutDisplay = strings.Join(row.Shortcuts, ",")
					}
					statusCol := fmt.Sprintf("%-14s", row.Status)
					if useColor {
						switch row.Status {
						case "enabled":
							statusCol = "\x1b[32m" + statusCol + "\x1b[0m"
						case "disabled":
							statusCol = "\x1b[2m" + statusCol + "\x1b[0m"
						}
					}
					fmt.Printf("%-20s %s %s\n", row.Name, statusCol, shortcutDisplay)
				}
				return nil
			}

			shortcutsByService := shortcutsByServiceName(cfg.CommandAliases)
			installer := installerFromConfig(cfg)
			installed, err := installer.List()
			if err != nil {
				return err
			}

			if outputAsJSON() {
				rows := make([]map[string]any, 0, len(installed))
				for _, svc := range installed {
					statusStr := "disabled"
					if svc.Enabled {
						statusStr = "enabled"
					}
					shortcuts := shortcutsByService[svc.Manifest.Name]
					if shortcuts == nil {
						shortcuts = []string{}
					}
					rows = append(rows, map[string]any{
						"name":      svc.Manifest.Name,
						"version":   svc.Manifest.Version,
						"actions":   len(svc.Manifest.Actions),
						"enabled":   svc.Enabled,
						"status":    statusStr,
						"shortcuts": shortcuts,
					})
				}
				return printOutput(rows)
			}
			if len(installed) == 0 {
				fmt.Println("No services installed.")
				fmt.Println("\nRun 'kimbap init --services select' to install services, or 'kimbap service install <name>' to install one.")
				return nil
			}
			useColor := isColorStdout()
			fmt.Printf("%-20s %-10s %-9s %-8s %s\n", "NAME", "VERSION", "ACTIONS", "STATUS", "SHORTCUTS")
			for _, svc := range installed {
				statusStr := "disabled"
				if svc.Enabled {
					statusStr = "enabled"
				}
				shortcutDisplay := "-"
				if shortcuts, exists := shortcutsByService[svc.Manifest.Name]; exists && len(shortcuts) > 0 {
					shortcutDisplay = strings.Join(shortcuts, ",")
				}
				statusCol := fmt.Sprintf("%-8s", statusStr)
				if useColor {
					if svc.Enabled {
						statusCol = "\x1b[32m" + statusCol + "\x1b[0m"
					} else {
						statusCol = "\x1b[2m" + statusCol + "\x1b[0m"
					}
				}
				fmt.Printf("%-20s %-10s %-9d %s %s\n", svc.Manifest.Name, svc.Manifest.Version, len(svc.Manifest.Actions), statusCol, shortcutDisplay)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&available, "available", false, "list all catalog services with installed/enabled status")
	return cmd
}

func newServiceEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable an installed service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			installer := installerFromConfig(cfg)
			name := strings.TrimSpace(args[0])
			if err := installer.Enable(name); err != nil {
				return augmentServiceNotFoundError(installer, name, err)
			}

			autoAlias := ""
			autoAliasCreated := false
			actionAliasesCreated := make([]string, 0)
			if enabledService, getErr := installer.Get(name); getErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "warning: enabled service alias setup skipped for %s: %v\n", name, getErr)
			} else {
				if strings.HasPrefix(strings.TrimSpace(enabledService.Source), "registry:") {
					if credErr := ensureInstalledServiceCredentials(commandContext(cmd), cfg, &enabledService.Manifest); credErr != nil {
						return credErr
					}
				}
				shortcutResult := applyInstalledShortcuts(cfg, installer, &enabledService.Manifest, "enabled")
				autoAlias = shortcutResult.AutoAlias
				autoAliasCreated = shortcutResult.AutoAliasCreated
				actionAliasesCreated = shortcutResult.ActionAliasesCreated
			}

			if outputAsJSON() {
				maybePrintAgentSyncHint(opts.format)
				return printOutput(map[string]any{
					"enabled":                true,
					"name":                   name,
					"auto_alias":             autoAlias,
					"auto_alias_created":     autoAliasCreated,
					"action_aliases_created": actionAliasesCreated,
				})
			}
			maybePrintAgentSyncHint(opts.format)
			msg := fmt.Sprintf(successCheck()+" %s enabled", name)
			if autoAlias != "" {
				msg += fmt.Sprintf(" (alias: %s)", autoAlias)
			}
			if len(actionAliasesCreated) > 0 {
				msg += fmt.Sprintf(" (action aliases: %s)", formatActionAliasesSummary(actionAliasesCreated))
			}
			return printOutput(msg)
		},
	}
	return cmd
}

func newServiceDisableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <name>",
		Short: "Disable an installed service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			installer := installerFromConfig(cfg)
			name := strings.TrimSpace(args[0])

			if _, getErr := installer.Get(name); getErr != nil {
				if errors.Is(getErr, fs.ErrNotExist) {
					return augmentServiceNotFoundError(installer, name, getErr)
				}
				return getErr
			}

			if err := withServiceAliasCleanupRollback(cfg, name, "disable", func() error {
				return installer.Disable(name)
			}); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return augmentServiceNotFoundError(installer, name, err)
				}
				return err
			}
			if outputAsJSON() {
				maybePrintAgentSyncHint(opts.format)
				return printOutput(map[string]any{"enabled": false, "name": name})
			}
			maybePrintAgentSyncHint(opts.format)
			return printOutput(fmt.Sprintf(successCheck()+" %s disabled", name))
		},
	}
	return cmd
}

func newServiceRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove installed service by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			installer := installerFromConfig(cfg)
			name := strings.TrimSpace(args[0])

			if _, getErr := installer.Get(name); getErr != nil {
				if errors.Is(getErr, fs.ErrNotExist) {
					return augmentServiceNotFoundError(installer, name, getErr)
				}
				return getErr
			}

			if err := withServiceAliasCleanupRollback(cfg, name, "remove", func() error {
				return installer.Remove(name)
			}); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return augmentServiceNotFoundError(installer, name, err)
				}
				return err
			}
			if outputAsJSON() {
				maybePrintAgentSyncHint(opts.format)
				return printOutput(map[string]any{"removed": true, "name": name})
			}
			maybePrintAgentSyncHint(opts.format)
			return printOutput(fmt.Sprintf(successCheck()+" %s removed", name))
		},
	}
	return cmd
}

func withServiceAliasCleanupRollback(cfg *config.KimbapConfig, name, operation string, run func() error) error {
	configPath, resolveErr := resolveConfigPath()
	if resolveErr != nil {
		return fmt.Errorf("service %q %s setup failed: resolve config path: %w", name, operation, resolveErr)
	}
	serviceAliasSnapshot := collectServiceAliasesForTarget(cfg.Aliases, name)
	commandAliasSnapshot := collectCommandAliasesForTarget(cfg.CommandAliases, name)
	if _, _, _, cleanupErr := cleanupAliasesForService(configPath, name, cfg.Aliases, cfg.CommandAliases); cleanupErr != nil {
		return fmt.Errorf("service %q %s failed during alias cleanup: %w", name, operation, cleanupErr)
	}

	if err := run(); err != nil {
		if rollbackErr := restoreServiceScopedAliases(configPath, cfg.Aliases, cfg.CommandAliases, serviceAliasSnapshot, commandAliasSnapshot); rollbackErr != nil {
			return fmt.Errorf("%s service %q: %w (and failed to restore aliases: %v)", operation, name, err, rollbackErr)
		}
		return err
	}

	return nil
}

func newServiceUpdateCommand() *cobra.Command {
	var force bool
	var noShortcuts bool
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update an installed service to the latest version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			manifest, newSource, resolveErr := resolveServiceInstallSource(commandContext(cmd), sourceToInstallArg(source))
			if resolveErr != nil {
				return fmt.Errorf("resolve update source for %q (%s): %w", name, source, resolveErr)
			}

			if strings.TrimSpace(manifest.Name) != name {
				return fmt.Errorf("update refused: fetched manifest has name %q but expected %q — source may have changed", manifest.Name, name)
			}

			if !force && strings.TrimSpace(manifest.Version) == strings.TrimSpace(installed.Manifest.Version) {
				if outputAsJSON() {
					maybePrintAgentSyncHint(opts.format)
					return printOutput(map[string]any{
						"updated": false,
						"name":    installed.Manifest.Name,
						"version": installed.Manifest.Version,
						"source":  source,
						"message": "already up to date (use --force to reinstall)",
					})
				}
				maybePrintAgentSyncHint(opts.format)
				return printOutput(fmt.Sprintf(successCheck()+" %s (%s) already up to date", installed.Manifest.Name, installed.Manifest.Version))
			}

			updated, installErr := installer.InstallWithForceAndActivation(manifest, newSource, true, installed.Enabled)
			if installErr != nil {
				return installErr
			}

			autoAlias := ""
			autoAliasCreated := false
			actionAliasesCreated := make([]string, 0)
			if !noShortcuts && updated.Enabled {
				shortcutResult := applyInstalledShortcuts(cfg, installer, &updated.Manifest, "auto")
				autoAlias = shortcutResult.AutoAlias
				autoAliasCreated = shortcutResult.AutoAliasCreated
				actionAliasesCreated = shortcutResult.ActionAliasesCreated
			}

			if outputAsJSON() {
				maybePrintAgentSyncHint(opts.format)
				return printOutput(map[string]any{
					"updated":                true,
					"name":                   updated.Manifest.Name,
					"version":                updated.Manifest.Version,
					"source":                 updated.Source,
					"auto_alias":             autoAlias,
					"auto_alias_created":     autoAliasCreated,
					"action_aliases_created": actionAliasesCreated,
				})
			}
			maybePrintAgentSyncHint(opts.format)
			msg := fmt.Sprintf(successCheck()+" %s updated to %s", updated.Manifest.Name, updated.Manifest.Version)
			if autoAlias != "" {
				msg += fmt.Sprintf(" (alias: %s)", autoAlias)
			}
			if len(actionAliasesCreated) > 0 {
				msg += fmt.Sprintf(" (action aliases: %s)", formatActionAliasesSummary(actionAliasesCreated))
			}
			return printOutput(msg)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "force update even if version is unchanged")
	cmd.Flags().BoolVar(&noShortcuts, "no-shortcuts", false, "skip automatic shortcut alias setup during update")
	return cmd
}

func newServiceOutdatedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "List outdated services from the service catalog",
		Long:  "Lists installed services from the registry when installed version differs from the service catalog version.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			installer := installerFromConfig(cfg)
			installed, err := installer.List()
			if err != nil {
				return err
			}

			type outdatedEntry struct {
				Name             string `json:"name"`
				InstalledVersion string `json:"installed_version"`
				LatestVersion    string `json:"latest_version"`
				Source           string `json:"source"`
			}

			entries := make([]outdatedEntry, 0)
			for _, svc := range installed {
				source := strings.TrimSpace(svc.Source)
				if !strings.HasPrefix(source, "registry:") {
					continue
				}

				installArg := sourceToInstallArg(source)
				if strings.TrimSpace(installArg) == "" {
					installArg = svc.Manifest.Name
				}

				manifest, _, resolveErr := resolveServiceInstallSource(commandContext(cmd), installArg)
				if resolveErr != nil {
					var notFound *registry.ErrNotFound
					if errors.As(resolveErr, &notFound) {
						continue
					}
					_, _ = fmt.Fprintf(os.Stderr, "warning: failed to resolve %q from catalog: %v\n", installArg, resolveErr)
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
				fmt.Println("No outdated catalog services found.")
				return nil
			}

			fmt.Printf("%-30s %-12s %-12s %s\n", "SERVICE", "INSTALLED", "LATEST", "SOURCE")
			useColor := isColorStdout()
			for _, e := range entries {
				instVer := fmt.Sprintf("%-12s", e.InstalledVersion)
				latestVer := fmt.Sprintf("%-12s", e.LatestVersion)
				if useColor {
					instVer = "\x1b[33m" + instVer + "\x1b[0m"
					latestVer = "\x1b[32m" + latestVer + "\x1b[0m"
				}
				fmt.Printf("%-30s %s %s %s\n", e.Name, instVer, latestVer, e.Source)
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
				if outputAsJSON() {
					_ = printOutput(map[string]any{"valid": false, "error": err.Error()})
				}
				return err
			}
			if outputAsJSON() {
				return printOutput(map[string]any{"valid": true, "name": manifest.Name, "version": manifest.Version})
			}
			return printOutput(fmt.Sprintf(successCheck()+" %s (%s) is valid", manifest.Name, manifest.Version))
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

			if outputAsJSON() {
				return printOutput(map[string]any{"signed": true})
			}
			return printOutput(successCheck() + " lockfile signed")
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

func shortcutsByServiceName(commandAliases map[string]string) map[string][]string {
	out := make(map[string][]string)
	for alias, target := range commandAliases {
		service, action := splitActionName(strings.TrimSpace(target))
		if service == "" || action == "" {
			continue
		}
		out[service] = append(out[service], alias)
	}
	for serviceName := range out {
		sort.Strings(out[serviceName])
	}
	return out
}

func resolveServiceInstallSource(ctx context.Context, arg string) (*services.ServiceManifest, string, error) {
	if ctx == nil {
		ctx = contextBackground()
	}
	trimmed := strings.TrimSpace(arg)
	if trimmed == "" {
		return nil, "", fmt.Errorf("service source is required")
	}

	if strings.HasPrefix(trimmed, "registry:") {
		registryName := strings.TrimSpace(strings.TrimPrefix(trimmed, "registry:"))
		return resolveRegistryServiceByName(ctx, registryName)
	}

	if scheme := serviceSourceURLScheme(trimmed); scheme == "http" {
		return nil, "", fmt.Errorf("insecure URL %q rejected: use https:// to install service manifests", trimmed)
	}
	if scheme := serviceSourceURLScheme(trimmed); scheme == "https" {
		manifest, err := parseServiceManifestURL(ctx, trimmed)
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
		resolveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		manifest, source, resolveErr := reg.Resolve(resolveCtx, serviceName)
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
			return nil, "", fmt.Errorf("service source %q is a directory. Pass a YAML file path, URL, or catalog service name", trimmed)
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

	return resolveRegistryServiceByName(ctx, trimmed)
}

func resolveRegistryServiceByName(ctx context.Context, name string) (*services.ServiceManifest, string, error) {
	if ctx == nil {
		ctx = contextBackground()
	}
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if err := services.ValidateServiceName(trimmed); err != nil {
		return nil, "", err
	}

	data, err := catalog.Get(trimmed)
	if err == nil {
		manifest, parseErr := services.ParseManifest(data)
		if parseErr != nil {
			return nil, "", fmt.Errorf("parse catalog service %q: %w", trimmed, parseErr)
		}
		return manifest, "registry:" + trimmed, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, "", fmt.Errorf("load catalog service %q: %w", trimmed, err)
	}

	cfg, cfgErr := loadAppConfig()
	if cfgErr == nil {
		registryURL := strings.TrimSpace(cfg.Services.RegistryURL)
		if registryURL != "" && registryURL != "https://services.kimbap.ai" {
			remoteReg := registry.NewRemoteRegistry("registry", registryURL)
			resolveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			manifest, _, resolveErr := remoteReg.Resolve(resolveCtx, trimmed)
			if resolveErr == nil {
				return manifest, "registry:" + trimmed, nil
			}
			var notFound *registry.ErrNotFound
			if !errors.As(resolveErr, &notFound) {
				return nil, "", fmt.Errorf("load registry service %q from %s: %w", trimmed, registryURL, resolveErr)
			}
		}
	}

	catalogNames, _ := catalog.List()
	hint := "Run 'kimbap service list --available' to see all catalog services."
	if suggestion := didYouMean(trimmed, catalogNames); suggestion != "" {
		hint = fmt.Sprintf("Did you mean %q? Run 'kimbap service list --available' to see all catalog services.", suggestion)
	}
	return nil, "", fmt.Errorf("%w. %s", &registry.ErrNotFound{Name: trimmed, Registry: "catalog"}, hint)
}

func serviceSourceURLScheme(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return ""
	}
	return scheme
}

func sourceToInstallArg(source string) string {
	trimmed := strings.TrimSpace(source)
	if strings.HasPrefix(trimmed, "registry:") {
		return trimmed
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

func augmentServiceNotFoundError(installer *services.LocalInstaller, name string, err error) error {
	if installer == nil || !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	installed, listErr := installer.List()
	if listErr != nil {
		return err
	}

	names := make([]string, len(installed))
	for i, svc := range installed {
		names[i] = svc.Manifest.Name
	}
	if suggestion := didYouMean(name, names); suggestion != "" {
		return fmt.Errorf("%w\n\nDid you mean %q?", err, suggestion)
	}
	return err
}

func collectServiceAliasesForTarget(serviceAliases map[string]string, serviceName string) map[string]string {
	target := strings.ToLower(strings.TrimSpace(serviceName))
	out := make(map[string]string)
	if target == "" {
		return out
	}
	for alias, mapped := range serviceAliases {
		if strings.ToLower(strings.TrimSpace(mapped)) != target {
			continue
		}
		out[alias] = mapped
	}
	return out
}

func collectCommandAliasesForTarget(commandAliases map[string]string, serviceName string) map[string]string {
	target := strings.ToLower(strings.TrimSpace(serviceName))
	out := make(map[string]string)
	if target == "" {
		return out
	}
	prefix := target + "."
	for alias, mapped := range commandAliases {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(mapped)), prefix) {
			continue
		}
		out[alias] = mapped
	}
	return out
}

func restoreServiceScopedAliases(configPath string, serviceAliases map[string]string, commandAliases map[string]string, serviceAliasSnapshot map[string]string, commandAliasSnapshot map[string]string) error {
	rollbackIssues := make([]string, 0)

	serviceKeys := make([]string, 0, len(serviceAliasSnapshot))
	for alias := range serviceAliasSnapshot {
		serviceKeys = append(serviceKeys, alias)
	}
	sort.Strings(serviceKeys)
	for _, alias := range serviceKeys {
		target := serviceAliasSnapshot[alias]
		if err := upsertConfigAlias(configPath, alias, target); err != nil {
			rollbackIssues = append(rollbackIssues, fmt.Sprintf("restore service alias %q: %v", alias, err))
			continue
		}
		serviceAliases[alias] = target
	}

	commandKeys := make([]string, 0, len(commandAliasSnapshot))
	for alias := range commandAliasSnapshot {
		commandKeys = append(commandKeys, alias)
	}
	sort.Strings(commandKeys)
	for _, alias := range commandKeys {
		target := commandAliasSnapshot[alias]
		if err := upsertConfigCommandAlias(configPath, alias, target); err != nil {
			rollbackIssues = append(rollbackIssues, fmt.Sprintf("restore command alias %q: %v", alias, err))
			continue
		}
		commandAliases[alias] = target
		if _, _, err := ensureExecutableActionAlias(alias); err != nil {
			rollbackIssues = append(rollbackIssues, fmt.Sprintf("restore executable alias %q: %v", alias, err))
		}
	}

	if len(rollbackIssues) > 0 {
		return fmt.Errorf("%s", strings.Join(rollbackIssues, "; "))
	}

	return nil
}

func cleanupAliasesForService(configPath string, serviceName string, serviceAliases map[string]string, commandAliases map[string]string) ([]string, []string, []string, error) {
	targetService := strings.ToLower(strings.TrimSpace(serviceName))
	if targetService == "" {
		return nil, nil, nil, nil
	}

	removedServiceAliases := make([]string, 0)
	removedCommandAliases := make([]string, 0)
	removedExecutables := make([]string, 0)
	serviceAliasTargets := make(map[string]string)
	commandAliasTargets := make(map[string]string)

	serviceAliasKeys := make([]string, 0)
	for alias, mapped := range serviceAliases {
		if strings.ToLower(strings.TrimSpace(mapped)) != targetService {
			continue
		}
		serviceAliasTargets[alias] = mapped
		serviceAliasKeys = append(serviceAliasKeys, alias)
	}
	sort.Strings(serviceAliasKeys)

	commandAliasKeys := make([]string, 0)
	prefix := targetService + "."
	for alias, mapped := range commandAliases {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(mapped)), prefix) {
			continue
		}
		commandAliasTargets[alias] = mapped
		commandAliasKeys = append(commandAliasKeys, alias)
	}
	sort.Strings(commandAliasKeys)

	rollback := func(cause error) error {
		rollbackIssues := make([]string, 0)

		for _, alias := range removedServiceAliases {
			target := serviceAliasTargets[alias]
			if err := upsertConfigAlias(configPath, alias, target); err != nil {
				rollbackIssues = append(rollbackIssues, fmt.Sprintf("restore service alias %q: %v", alias, err))
				continue
			}
			serviceAliases[alias] = target
		}

		for _, alias := range removedCommandAliases {
			target := commandAliasTargets[alias]
			if err := upsertConfigCommandAlias(configPath, alias, target); err != nil {
				rollbackIssues = append(rollbackIssues, fmt.Sprintf("restore command alias %q: %v", alias, err))
				continue
			}
			commandAliases[alias] = target
		}

		for _, alias := range removedExecutables {
			if _, _, err := ensureExecutableActionAlias(alias); err != nil {
				rollbackIssues = append(rollbackIssues, fmt.Sprintf("restore executable alias %q: %v", alias, err))
			}
		}

		if len(rollbackIssues) > 0 {
			return fmt.Errorf("%w (rollback issues: %s)", cause, strings.Join(rollbackIssues, "; "))
		}
		return cause
	}

	for _, alias := range serviceAliasKeys {
		removed, err := removeConfigAlias(configPath, alias)
		if err != nil {
			return removedServiceAliases, removedCommandAliases, removedExecutables, rollback(fmt.Errorf("remove service alias %q: %w", alias, err))
		}
		if !removed {
			continue
		}
		delete(serviceAliases, alias)
		removedServiceAliases = append(removedServiceAliases, alias)
	}

	for _, alias := range commandAliasKeys {
		removed, err := removeConfigCommandAlias(configPath, alias)
		if err != nil {
			return removedServiceAliases, removedCommandAliases, removedExecutables, rollback(fmt.Errorf("remove command alias %q: %w", alias, err))
		}
		if !removed {
			continue
		}

		delete(commandAliases, alias)
		removedCommandAliases = append(removedCommandAliases, alias)

		removedExecutable, removeErr := removeExecutableActionAlias(alias)
		if removeErr != nil {
			return removedServiceAliases, removedCommandAliases, removedExecutables, rollback(fmt.Errorf("remove command alias executable %q: %w", alias, removeErr))
		}
		if removedExecutable {
			removedExecutables = append(removedExecutables, alias)
		}
	}

	return removedServiceAliases, removedCommandAliases, removedExecutables, nil
}

func maybePrintAgentSyncHint(format string) {
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		return
	}

	projectDir, wdErr := os.Getwd()
	if wdErr != nil || strings.TrimSpace(projectDir) == "" {
		projectDir = "."
	}

	syncResult, syncErr := runAgentsSync(projectDir, "", "", false, false)
	if syncErr == nil && syncResult.AgentsFound > 0 {
		written := 0
		pruned := 0
		hasFailures := false
		for _, r := range syncResult.SyncResults {
			written += len(r.Written)
			pruned += len(r.Pruned)
			if len(r.Failed) > 0 || len(r.Errors) > 0 {
				hasFailures = true
			}
		}

		if hasFailures {
			_, _ = fmt.Fprintf(os.Stderr, "\nWarning: automatic agent sync completed with issues. Run 'kimbap agents sync --force' to inspect and repair.\n")
			return
		}

		if written > 0 || pruned > 0 || len(syncResult.MetaAgentSkillPaths) > 0 {
			_, _ = fmt.Fprintf(os.Stderr, "\nSynced AI agent skills automatically (agents=%d, written=%d, pruned=%d).\n", syncResult.AgentsFound, written, pruned)
		}
		return
	}

	results, err := agents.GlobalStatus()
	if err != nil || len(results) == 0 {
		return
	}
	hasAgent := false
	for _, r := range results {
		if r.Detected || r.AgentSkillPresent || r.InjectPresent {
			hasAgent = true
			break
		}
	}
	if !hasAgent {
		return
	}
	if syncErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\nwarning: automatic agent sync skipped: %v\n", syncErr)
	}
	_, _ = fmt.Fprintln(os.Stderr, "\nHint: Run 'kimbap agents sync' to update your AI agents with this change.")
}

func formatActionAliasesSummary(aliases []string) string {
	if len(aliases) <= 3 {
		return strings.Join(aliases, ", ")
	}
	return strings.Join(aliases[:3], ", ") + fmt.Sprintf(", ... and %d more", len(aliases)-3)
}

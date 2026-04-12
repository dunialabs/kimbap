package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/agents"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/registry"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
)

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
				if len(rows) == 0 {
					fmt.Println("No catalog services available.")
					return nil
				}
				useColor := isColorStdout()
				fmt.Printf("%-20s %-14s %s\n", "NAME", "STATUS", "SHORTCUTS")
				notInstalledNames := make([]string, 0)
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
					if row.Status == "not_installed" {
						notInstalledNames = append(notInstalledNames, row.Name)
					}
				}
				if len(notInstalledNames) > 0 {
					fmt.Printf("\n%s\n", serviceInstallFooter(notInstalledNames))
				} else if len(rows) > 0 {
					fmt.Println()
					fmt.Println("All catalog services are installed. Run 'kimbap service outdated' to check for updates.")
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
			disabledNames := make([]string, 0)
			for _, svc := range installed {
				statusStr := "disabled"
				if svc.Enabled {
					statusStr = "enabled"
				} else {
					disabledNames = append(disabledNames, svc.Manifest.Name)
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
			if len(disabledNames) > 0 {
				fmt.Printf("\n%s\n", serviceEnableFooter(disabledNames))
			} else if len(installed) > 0 {
				hasAnyShortcut := len(shortcutsByService) > 0
				fmt.Println()
				if hasAnyShortcut {
					fmt.Println("Run 'kimbap call <service>.<action>' to use your services.")
				} else {
					fmt.Println("Run 'kimbap call <service>.<action>' to use your services, or 'kimbap alias set <shortcut> <service>.<action>' to create shortcuts.")
				}
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
			msg += serviceShortcutHint(actionAliasesCreated)
			if err := printOutput(msg); err != nil {
				return err
			}
			if !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Run 'kimbap actions list --service %s' to see available actions.\n", name)
			}
			return nil
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
			if err := printOutput(fmt.Sprintf(successCheck()+" %s disabled", name)); err != nil {
				return err
			}
			if !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Re-enable: run 'kimbap service enable %s'.\n", name)
			}
			return nil
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
			if err := printOutput(fmt.Sprintf(successCheck()+" %s removed", name)); err != nil {
				return err
			}
			if !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Reinstall: run 'kimbap service install %s'.\n", name)
			}
			return nil
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
				if errors.Is(resolveErr, fs.ErrNotExist) {
					return fmt.Errorf("resolve update source for %q: source file not found (%s) — reinstall with 'kimbap service install %s'", name, source, name)
				}
				return fmt.Errorf("resolve update source for %q (%s): %w", name, source, resolveErr)
			}

			if strings.TrimSpace(manifest.Name) != name {
				return fmt.Errorf("update refused: fetched manifest has name %q but expected %q — source may have changed. Use 'kimbap service remove %s' and reinstall.", manifest.Name, name, name)
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
			msg += serviceShortcutHint(actionAliasesCreated)
			if err := printOutput(msg); err != nil {
				return err
			}
			if !outputAsJSON() {
				_, _ = fmt.Fprintln(os.Stdout, "Run 'kimbap service outdated' to check for other updates.")
			}
			return nil
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
				fmt.Println("All installed catalog services are up to date.")
				return nil
			}

			fmt.Printf("%-30s %-12s %-12s %s\n", "SERVICE", "INSTALLED", "LATEST", "SOURCE")
			useColor := isColorStdout()
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				instVer := fmt.Sprintf("%-12s", e.InstalledVersion)
				latestVer := fmt.Sprintf("%-12s", e.LatestVersion)
				if useColor {
					instVer = "\x1b[33m" + instVer + "\x1b[0m"
					latestVer = "\x1b[32m" + latestVer + "\x1b[0m"
				}
				fmt.Printf("%-30s %s %s %s\n", e.Name, instVer, latestVer, e.Source)
				names = append(names, e.Name)
			}
			fmt.Printf("\n%s\n", serviceUpdateFooter(names))
			return nil
		},
	}
	return cmd
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
		return fmt.Errorf("%w\n\nDid you mean %q? Run 'kimbap service list' to see installed services.", err, suggestion)
	}
	return fmt.Errorf("%w\n\nRun 'kimbap service list' to see installed services.", err)
}

func maybePrintAgentSyncHint(format string) {
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		return
	}

	results, err := agents.GlobalStatus()
	if err != nil || len(results) == 0 {
		return
	}
	hasAgent := false
	for _, r := range results {
		if r.AgentSkillPresent || r.InjectPresent {
			hasAgent = true
			break
		}
	}
	if !hasAgent {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, "\nHint: Run 'kimbap agents sync' to update your AI agents with this change.")
}

func formatActionAliasesSummary(aliases []string) string {
	if len(aliases) <= 3 {
		return strings.Join(aliases, ", ")
	}
	return strings.Join(aliases[:3], ", ") + fmt.Sprintf(", ... and %d more", len(aliases)-3)
}

func serviceInstallFooter(names []string) string {
	switch len(names) {
	case 0:
		return "Run 'kimbap service install <name>' to install a service."
	case 1:
		return fmt.Sprintf("Run 'kimbap service install %s' to install.", names[0])
	case 2:
		return fmt.Sprintf("Run 'kimbap service install %s' or 'kimbap service install %s' to install.", names[0], names[1])
	case 3:
		return fmt.Sprintf("Run 'kimbap service install %s', '%s', or '%s' to install.", names[0], names[1], names[2])
	default:
		return fmt.Sprintf("Run 'kimbap service install <name>' to install. (%d services available)", len(names))
	}
}

func serviceEnableFooter(names []string) string {
	switch len(names) {
	case 0:
		return "Run 'kimbap service enable <name>' to enable a disabled service."
	case 1:
		return fmt.Sprintf("Run 'kimbap service enable %s' to enable it.", names[0])
	case 2:
		return fmt.Sprintf("Run 'kimbap service enable %s' or 'kimbap service enable %s' to enable.", names[0], names[1])
	case 3:
		return fmt.Sprintf("Run 'kimbap service enable %s', '%s', or '%s' to enable.", names[0], names[1], names[2])
	default:
		return fmt.Sprintf("Run 'kimbap service enable <name>' to enable. (%d services disabled)", len(names))
	}
}

func serviceUpdateFooter(names []string) string {
	switch len(names) {
	case 0:
		return "Run 'kimbap service update <name>' to update a service."
	case 1:
		return fmt.Sprintf("Run 'kimbap service update %s' to update.", names[0])
	case 2:
		return fmt.Sprintf("Run 'kimbap service update %s' or 'kimbap service update %s' to update.", names[0], names[1])
	case 3:
		return fmt.Sprintf("Run 'kimbap service update %s', 'kimbap service update %s', or 'kimbap service update %s' to update.", names[0], names[1], names[2])
	default:
		return fmt.Sprintf("Run 'kimbap service update <name>' to update. (%d services outdated)", len(names))
	}
}

func serviceShortcutHint(actionAliasesCreated []string) string {
	if len(actionAliasesCreated) == 0 {
		return ""
	}
	return "\n  Shortcuts: " + formatActionAliasesSummary(actionAliasesCreated) +
		"\n  Try: " + actionAliasesCreated[0] + " --help"
}

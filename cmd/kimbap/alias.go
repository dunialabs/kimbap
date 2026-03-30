package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	actionAliasActionPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	aliasExecutablePath      = os.Executable
	aliasFileLstat           = os.Lstat
	aliasFileReadlink        = os.Readlink
	aliasFileSymlink         = os.Symlink
	aliasFileRemove          = os.Remove
)

func newAliasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Manage service name aliases",
		Long: `Manage short aliases for installed service names.

Aliases support two modes:

1) Service alias (for kimbap call)
   kimbap alias set gh github
   kimbap call gh.list-repos --owner octocat

2) Command alias (standalone executable)
   kimbap alias set geosearch open-meteo-geocoding.search
   geosearch --name "San Francisco"`,
	}
	cmd.AddCommand(newAliasSetCommand())
	cmd.AddCommand(newAliasListCommand())
	cmd.AddCommand(newAliasRemoveCommand())
	return cmd
}

func newAliasSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <alias> <service|service.action>",
		Short: "Create or update a service or command alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			alias := strings.ToLower(strings.TrimSpace(args[0]))
			target := strings.ToLower(strings.TrimSpace(args[1]))

			if alias == "" {
				return fmt.Errorf("alias must be non-empty")
			}
			if target == "" {
				return fmt.Errorf("target service must be non-empty")
			}
			if alias == target {
				return fmt.Errorf("alias %q must differ from target %q", alias, target)
			}
			if !services.IsValidServiceAlias(alias) {
				return fmt.Errorf("invalid alias name %q: must match [a-z][a-z0-9-]*, must not contain dots, and must not be reserved", alias)
			}
			if aliasConflictsWithBuiltinCommand(alias) {
				return fmt.Errorf("alias %q conflicts with built-in CLI command name", alias)
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			if cfg.CommandAliases == nil {
				cfg.CommandAliases = map[string]string{}
			}

			installer := installerFromConfig(cfg)
			installed, listErr := installer.List()
			if listErr != nil {
				return fmt.Errorf("list installed services: %w", listErr)
			}
			installedNames := make(map[string]bool, len(installed))
			for _, svc := range installed {
				installedNames[svc.Manifest.Name] = true
				if svc.Manifest.Name == alias {
					return fmt.Errorf("alias %q conflicts with installed service %q — choose a different alias name", alias, svc.Manifest.Name)
				}
			}

			configPath, resolveErr := resolveConfigPath()
			if resolveErr != nil {
				return fmt.Errorf("resolve config path: %w", resolveErr)
			}

			if strings.Contains(target, ".") {
				normalizedTarget, normalizeErr := normalizeActionAliasTarget(installer, target)
				if normalizeErr != nil {
					return normalizeErr
				}
				if existing, ok := cfg.Aliases[alias]; ok {
					return fmt.Errorf("alias %q already used as service alias -> %q", alias, existing)
				}
				executablePath, executableCreated, aliasErr := ensureExecutableActionAlias(alias)
				if aliasErr != nil {
					return aliasErr
				}
				if err := upsertConfigCommandAlias(configPath, alias, normalizedTarget); err != nil {
					if executableCreated {
						if _, cleanupErr := removeExecutableActionAlias(alias); cleanupErr != nil {
							return fmt.Errorf("set command alias %q -> %q: %w (rollback failed: %v)", alias, normalizedTarget, err, cleanupErr)
						}
					}
					return err
				}
				return printOutput(map[string]any{
					"alias":              alias,
					"target":             normalizedTarget,
					"type":               "action",
					"set":                true,
					"executable":         executablePath,
					"executable_created": executableCreated,
				})
			}

			if !installedNames[target] {
				if err := services.ValidateServiceName(target); err != nil {
					return fmt.Errorf("invalid alias target: %w", err)
				}
				return fmt.Errorf("target service %q is not installed — install it first with 'kimbap service install'", target)
			}

			if existing, isAlias := cfg.Aliases[target]; isAlias {
				return fmt.Errorf("target %q is itself an alias (-> %q) — aliases must point to real service names, not other aliases", target, existing)
			}
			if existing, isActionAlias := cfg.CommandAliases[alias]; isActionAlias {
				return fmt.Errorf("alias %q already used as action alias -> %q", alias, existing)
			}

			if err := upsertConfigAlias(configPath, alias, target); err != nil {
				return err
			}

			return printOutput(map[string]any{
				"alias":  alias,
				"target": target,
				"type":   "service",
				"set":    true,
			})
		},
	}
}

func newAliasListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured aliases",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			if len(cfg.Aliases) == 0 && len(cfg.CommandAliases) == 0 {
				if outputAsJSON() {
					return printOutput(map[string]any{
						"service_aliases": map[string]string{},
						"command_aliases": map[string]string{},
					})
				}
				return printOutput("No aliases configured. Use 'kimbap alias set <alias> <service>'")
			}

			if outputAsJSON() {
				return printOutput(map[string]any{
					"service_aliases": cfg.Aliases,
					"command_aliases": cfg.CommandAliases,
				})
			}

			if len(cfg.Aliases) > 0 {
				fmt.Println("Service aliases:")
				keys := make([]string, 0, len(cfg.Aliases))
				for k := range cfg.Aliases {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("%-20s → %s\n", k, cfg.Aliases[k])
				}
			}
			if len(cfg.CommandAliases) > 0 {
				if len(cfg.Aliases) > 0 {
					fmt.Println()
				}
				fmt.Println("Command aliases:")
				keys := make([]string, 0, len(cfg.CommandAliases))
				for k := range cfg.CommandAliases {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("%-20s → %s\n", k, cfg.CommandAliases[k])
				}
			}
			return nil
		},
	}
}

func newAliasRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <alias>",
		Short: "Remove a configured alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			alias := strings.ToLower(strings.TrimSpace(args[0]))
			if alias == "" {
				return fmt.Errorf("alias must be non-empty")
			}

			cfg, loadErr := loadAppConfig()
			if loadErr != nil {
				return loadErr
			}
			commandTarget, commandExists := cfg.CommandAliases[alias]

			configPath, err := resolveConfigPath()
			if err != nil {
				return fmt.Errorf("resolve config path: %w", err)
			}

			serviceRemoved, err := removeConfigAlias(configPath, alias)
			if err != nil {
				return err
			}
			commandRemoved, cmdErr := removeConfigCommandAlias(configPath, alias)
			if cmdErr != nil {
				return cmdErr
			}

			executableRemoved := false
			if commandRemoved {
				removed, removeErr := removeExecutableActionAlias(alias)
				if removeErr != nil {
					if commandExists {
						if rollbackErr := upsertConfigCommandAlias(configPath, alias, commandTarget); rollbackErr != nil {
							return fmt.Errorf("remove command alias executable for %q: %w (and failed to restore config entry: %v)", alias, removeErr, rollbackErr)
						}
					}
					return fmt.Errorf("remove command alias executable for %q: %w (command alias config entry restored)", alias, removeErr)
				}
				executableRemoved = removed
			}

			if !serviceRemoved && !commandRemoved {
				return fmt.Errorf("alias %q not found", alias)
			}

			return printOutput(map[string]any{
				"alias":              alias,
				"removed":            true,
				"service_removed":    serviceRemoved,
				"command_removed":    commandRemoved,
				"executable_removed": executableRemoved,
			})
		},
	}
}

func normalizeActionAliasTarget(installer *services.LocalInstaller, target string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(target))
	service, action := splitActionName(trimmed)
	if service == "" || action == "" {
		return "", fmt.Errorf("action alias target %q must be in <service.action> format", target)
	}
	if err := services.ValidateServiceName(service); err != nil {
		return "", fmt.Errorf("invalid action alias target service %q: %w", service, err)
	}
	if !actionAliasActionPattern.MatchString(action) {
		return "", fmt.Errorf("invalid action alias target action %q: must match [a-z][a-z0-9_-]*", action)
	}

	enabled, err := installer.ListEnabled()
	if err != nil {
		return "", fmt.Errorf("list enabled services: %w", err)
	}
	for _, svc := range enabled {
		if svc.Manifest.Name != service {
			continue
		}
		if _, ok := svc.Manifest.Actions[action]; ok {
			return service + "." + action, nil
		}
		actionsList := make([]string, 0, len(svc.Manifest.Actions))
		for actionName := range svc.Manifest.Actions {
			actionsList = append(actionsList, actionName)
		}
		sort.Strings(actionsList)
		if len(actionsList) == 0 {
			return "", fmt.Errorf("target service %q has no actions", service)
		}
		return "", fmt.Errorf("target action %q not found in enabled service %q (available: %s)", action, service, strings.Join(actionsList, ", "))
	}

	installed, listErr := installer.List()
	if listErr != nil {
		return "", fmt.Errorf("list installed services: %w", listErr)
	}
	for _, svc := range installed {
		if svc.Manifest.Name == service {
			return "", fmt.Errorf("target service %q is installed but disabled — enable it first with 'kimbap service enable %s'", service, service)
		}
	}

	return "", fmt.Errorf("target service %q is not installed — install it first with 'kimbap service install'", service)
}

func aliasConflictsWithBuiltinCommand(alias string) bool {
	if strings.EqualFold(strings.TrimSpace(alias), "kimbap") {
		return true
	}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == alias {
			return true
		}
		for _, item := range cmd.Aliases {
			if item == alias {
				return true
			}
		}
	}
	return false
}

func ensureExecutableActionAlias(alias string) (string, bool, error) {
	execPath, err := aliasExecutablePath()
	if err != nil {
		return "", false, fmt.Errorf("resolve kimbap executable path: %w", err)
	}
	execPath = filepath.Clean(execPath)

	aliasPath := filepath.Join(filepath.Dir(execPath), alias)
	if filepath.Clean(aliasPath) == execPath {
		return "", false, fmt.Errorf("alias %q conflicts with kimbap executable path", alias)
	}

	info, statErr := aliasFileLstat(aliasPath)
	if statErr == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return "", false, fmt.Errorf("cannot create command alias %q: path %s already exists and is not a symlink", alias, aliasPath)
		}
		currentTarget, readErr := aliasFileReadlink(aliasPath)
		if readErr != nil {
			return "", false, fmt.Errorf("inspect command alias symlink %s: %w", aliasPath, readErr)
		}
		if symlinkTargetMatchesExecutable(aliasPath, currentTarget, execPath) {
			return aliasPath, false, nil
		}
		return "", false, fmt.Errorf("cannot create command alias %q: symlink %s already points elsewhere", alias, aliasPath)
	}
	if !os.IsNotExist(statErr) {
		return "", false, fmt.Errorf("inspect alias path %s: %w", aliasPath, statErr)
	}

	if err := aliasFileSymlink(execPath, aliasPath); err != nil {
		return "", false, fmt.Errorf("create command alias symlink %s -> %s: %w", aliasPath, execPath, err)
	}
	return aliasPath, true, nil
}

func removeExecutableActionAlias(alias string) (bool, error) {
	execPath, err := aliasExecutablePath()
	if err != nil {
		return false, fmt.Errorf("resolve kimbap executable path: %w", err)
	}
	execPath = filepath.Clean(execPath)
	aliasPath := filepath.Join(filepath.Dir(execPath), alias)

	info, statErr := aliasFileLstat(aliasPath)
	if os.IsNotExist(statErr) {
		return false, nil
	}
	if statErr != nil {
		return false, fmt.Errorf("inspect alias path %s: %w", aliasPath, statErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, fmt.Errorf("alias executable path %s exists but is not a symlink", aliasPath)
	}
	target, readErr := aliasFileReadlink(aliasPath)
	if readErr != nil {
		return false, fmt.Errorf("inspect alias symlink %s: %w", aliasPath, readErr)
	}
	if !symlinkTargetMatchesExecutable(aliasPath, target, execPath) {
		return false, fmt.Errorf("alias executable path %s points to a different target", aliasPath)
	}
	if err := aliasFileRemove(aliasPath); err != nil {
		return false, fmt.Errorf("remove alias executable path %s: %w", aliasPath, err)
	}
	return true, nil
}

func symlinkTargetMatchesExecutable(symlinkPath, symlinkTarget, executablePath string) bool {
	resolvedTarget := symlinkTarget
	if !filepath.IsAbs(resolvedTarget) {
		resolvedTarget = filepath.Join(filepath.Dir(symlinkPath), resolvedTarget)
	}
	return filepath.Clean(resolvedTarget) == filepath.Clean(executablePath)
}

func upsertConfigAlias(configPath, alias, target string) error {
	return upsertConfigMapEntry(configPath, "aliases", alias, target)
}

func upsertConfigCommandAlias(configPath, alias, target string) error {
	return upsertConfigMapEntry(configPath, "command_aliases", alias, target)
}

func upsertConfigMapEntry(configPath, section, key, value string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	var raw map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if unmarshalErr := yaml.Unmarshal(data, &raw); unmarshalErr != nil {
			return fmt.Errorf("parse config file: %w", unmarshalErr)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read config file: %w", err)
	}
	if raw == nil {
		raw = map[string]any{}
	}

	entries := map[string]any{}
	if existing, ok := raw[section]; ok {
		switch typed := existing.(type) {
		case map[string]any:
			entries = typed
		case map[string]string:
			for k, v := range typed {
				entries[k] = v
			}
		}
	}

	entries[key] = value
	raw[section] = entries

	data, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(configPath), ".kimbap-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func removeConfigAlias(configPath, alias string) (bool, error) {
	return removeConfigMapEntry(configPath, "aliases", alias)
}

func removeConfigCommandAlias(configPath, alias string) (bool, error) {
	return removeConfigMapEntry(configPath, "command_aliases", alias)
}

func removeConfigMapEntry(configPath, section, key string) (bool, error) {
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read config file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false, fmt.Errorf("parse config file: %w", err)
	}
	if raw == nil {
		return false, nil
	}

	entries := map[string]any{}
	if existing, ok := raw[section]; ok {
		switch typed := existing.(type) {
		case map[string]any:
			entries = typed
		case map[string]string:
			for k, v := range typed {
				entries[k] = v
			}
		}
	}

	if _, exists := entries[key]; !exists {
		return false, nil
	}

	delete(entries, key)
	if len(entries) == 0 {
		delete(raw, section)
	} else {
		raw[section] = entries
	}

	updated, err := yaml.Marshal(raw)
	if err != nil {
		return false, fmt.Errorf("marshal config: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(configPath), ".kimbap-config-*.tmp")
	if err != nil {
		return false, fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(updated); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return false, fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return false, fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return false, fmt.Errorf("chmod temp config: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		_ = os.Remove(tmpPath)
		return false, fmt.Errorf("write config: %w", err)
	}

	return true, nil
}

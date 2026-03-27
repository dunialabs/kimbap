package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newAliasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Manage service name aliases",
		Long: `Manage short aliases for installed service names.

Aliases let you use short names in place of full service names:

  kimbap alias set gh github
  kimbap call gh.list-repos --owner octocat   # same as: kimbap call github.list-repos --owner octocat`,
	}
	cmd.AddCommand(newAliasSetCommand())
	cmd.AddCommand(newAliasListCommand())
	cmd.AddCommand(newAliasRemoveCommand())
	return cmd
}

func newAliasSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <alias> <service>",
		Short: "Create or update a service alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			alias := strings.TrimSpace(args[0])
			target := strings.TrimSpace(args[1])

			if alias == "" {
				return fmt.Errorf("alias must be non-empty")
			}
			if target == "" {
				return fmt.Errorf("target service must be non-empty")
			}
			if alias == target {
				return fmt.Errorf("alias %q must differ from target %q", alias, target)
			}
			if err := services.ValidateServiceName(alias); err != nil {
				return fmt.Errorf("invalid alias name: %w", err)
			}
			if err := services.ValidateServiceName(target); err != nil {
				return fmt.Errorf("invalid alias target: %w", err)
			}
			if strings.Contains(alias, ".") {
				return fmt.Errorf("alias %q must not contain dots — aliases resolve the service part only (before the first dot)", alias)
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
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
			if !installedNames[target] {
				return fmt.Errorf("target service %q is not installed — install it first with 'kimbap service install'", target)
			}

			if existing, isAlias := cfg.Aliases[target]; isAlias {
				return fmt.Errorf("target %q is itself an alias (-> %q) — aliases must point to real service names, not other aliases", target, existing)
			}

			configPath, resolveErr := resolveConfigPath()
			if resolveErr != nil {
				return fmt.Errorf("resolve config path: %w", resolveErr)
			}

			if err := upsertConfigAlias(configPath, alias, target); err != nil {
				return err
			}

			return printOutput(map[string]any{
				"alias":  alias,
				"target": target,
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

			if len(cfg.Aliases) == 0 {
				if outputAsJSON() {
					return printOutput(map[string]string{})
				}
				return printOutput("No aliases configured. Use 'kimbap alias set <alias> <service>'")
			}

			if outputAsJSON() {
				return printOutput(cfg.Aliases)
			}

			keys := make([]string, 0, len(cfg.Aliases))
			for k := range cfg.Aliases {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("%-20s → %s\n", k, cfg.Aliases[k])
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
			alias := strings.TrimSpace(args[0])
			if alias == "" {
				return fmt.Errorf("alias must be non-empty")
			}

			configPath, err := resolveConfigPath()
			if err != nil {
				return fmt.Errorf("resolve config path: %w", err)
			}

			removed, err := removeConfigAlias(configPath, alias)
			if err != nil {
				return err
			}
			if !removed {
				return fmt.Errorf("alias %q not found", alias)
			}

			return printOutput(map[string]any{
				"alias":   alias,
				"removed": true,
			})
		},
	}
}

func upsertConfigAlias(configPath, alias, target string) error {
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

	aliases := map[string]any{}
	if existing, ok := raw["aliases"]; ok {
		switch typed := existing.(type) {
		case map[string]any:
			aliases = typed
		case map[string]string:
			for k, v := range typed {
				aliases[k] = v
			}
		}
	}

	aliases[alias] = target
	raw["aliases"] = aliases

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

	aliases := map[string]any{}
	if existing, ok := raw["aliases"]; ok {
		switch typed := existing.(type) {
		case map[string]any:
			aliases = typed
		case map[string]string:
			for k, v := range typed {
				aliases[k] = v
			}
		}
	}

	if _, exists := aliases[alias]; !exists {
		return false, nil
	}

	delete(aliases, alias)
	if len(aliases) == 0 {
		delete(raw, "aliases")
	} else {
		raw["aliases"] = aliases
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

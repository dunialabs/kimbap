package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/services"
)

func shortestCommandAlias(actionName string, commandAliases map[string]string) string {
	shortest := ""
	for alias, target := range commandAliases {
		if target != actionName {
			continue
		}
		if shortest == "" || len(alias) < len(shortest) {
			shortest = alias
		}
	}
	return shortest
}

func preferredInvocation(actionName string, commandAliases map[string]string) string {
	if shortcut := shortestCommandAlias(actionName, commandAliases); shortcut != "" {
		return shortcut
	}
	return "kimbap call " + actionName
}

func resolveActionByName(cfg *config.KimbapConfig, name string) (*actions.ActionDefinition, error) {
	resolved := resolveAliasedActionName(cfg, name)
	defs, err := loadInstalledActions(cfg)
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		allInstalled, listErr := installerFromConfig(cfg).List()
		if listErr != nil {
			return nil, listErr
		}
		if len(allInstalled) > 0 {
			return nil, fmt.Errorf("no enabled services found — run 'kimbap service list' to see installed services, or 'kimbap service enable <name>' to enable one")
		}
		return nil, fmt.Errorf("no services installed — run 'kimbap init --services select' to choose what to install")
	}
	for i := range defs {
		if defs[i].Name == resolved {
			return &defs[i], nil
		}
	}

	if !strings.Contains(resolved, ".") {
		serviceName := strings.ToLower(resolved)
		var serviceActions []actions.ActionDefinition
		var canonicalNamespace string
		for _, d := range defs {
			if strings.EqualFold(d.Namespace, serviceName) {
				serviceActions = append(serviceActions, d)
				if canonicalNamespace == "" {
					canonicalNamespace = d.Namespace
				}
			}
		}
		if len(serviceActions) > 0 {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%q is a service name. Specify an action:\n", resolved))
			limit := 5
			if len(serviceActions) < limit {
				limit = len(serviceActions)
			}
			for i := 0; i < limit; i++ {
				desc := serviceActions[i].Description
				if desc == "" {
					desc = "-"
				}
				invocation := preferredInvocation(serviceActions[i].Name, cfg.CommandAliases)
				sb.WriteString(fmt.Sprintf("  %-44s %s\n", invocation, desc))
			}
			if len(serviceActions) > 5 {
				sb.WriteString(fmt.Sprintf("\nRun 'kimbap actions list --service %s' to see all %d actions.", canonicalNamespace, len(serviceActions)))
			}
			return nil, fmt.Errorf("%s", sb.String())
		}
	}

	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	hint := "Run 'kimbap actions list' to see available actions, or 'kimbap search <query>' to find by keyword."
	if suggestion := didYouMean(resolved, names); suggestion != "" {
		invocation := preferredInvocation(suggestion, cfg.CommandAliases)
		hint = fmt.Sprintf("Did you mean %q? Try: %s --help", suggestion, invocation)
	}
	return nil, fmt.Errorf("action %q not found in installed services. %s", resolved, hint)
}

func resolveAliasedActionName(cfg *config.KimbapConfig, actionName string) string {
	if cfg == nil || len(cfg.Aliases) == 0 {
		return actionName
	}
	service, action := splitActionName(actionName)
	if service == "" || action == "" {
		return actionName
	}
	if !services.IsValidServiceAlias(service) {
		return actionName
	}
	if target, ok := cfg.Aliases[service]; ok {
		target = strings.ToLower(strings.TrimSpace(target))
		if target != "" && services.ValidateServiceName(target) == nil {
			return target + "." + action
		}
	}
	return actionName
}

func splitActionName(actionName string) (service string, action string) {
	service, action, ok := strings.Cut(actionName, ".")
	if !ok {
		return "", actionName
	}
	return service, action
}

func rewriteArgsForExecutableAlias(argv []string) []string {
	if len(argv) == 0 {
		return argv
	}

	binary := strings.TrimSpace(filepath.Base(argv[0]))
	if binary == "" || strings.EqualFold(binary, "kimbap") {
		return argv
	}

	configPath, dataDir := parseConfigPathAndDataDirArgs(argv)

	var (
		cfg *config.KimbapConfig
		err error
	)
	if strings.TrimSpace(configPath) == "" && strings.TrimSpace(dataDir) == "" {
		cfg, err = config.LoadKimbapConfig()
	} else {
		resolvedPath, resolveErr := config.ResolveConfigPathWithDataDir(configPath, dataDir)
		if resolveErr != nil {
			return argv
		}
		if st, statErr := os.Stat(resolvedPath); statErr == nil {
			if st.IsDir() {
				return argv
			}
			cfg, err = config.LoadKimbapConfigWithoutDefault(resolvedPath)
		} else if os.IsNotExist(statErr) {
			cfg, err = config.LoadKimbapConfigWithoutDefault()
		} else {
			return argv
		}
	}
	if err != nil {
		return argv
	}

	return rewriteArgsForConfiguredExecutableAlias(argv, cfg.CommandAliases)
}

func parseConfigPathAndDataDirArgs(argv []string) (configPath string, dataDir string) {
	for i := 1; i < len(argv); i++ {
		tok := strings.TrimSpace(argv[i])
		if tok == "--" {
			break
		}
		if tok == "--config" && i+1 < len(argv) {
			next := strings.TrimSpace(argv[i+1])
			if next != "" && !strings.HasPrefix(next, "-") {
				configPath = next
				i++
				continue
			}
		}
		if tok == "--data-dir" && i+1 < len(argv) {
			next := strings.TrimSpace(argv[i+1])
			if next != "" && !strings.HasPrefix(next, "-") {
				dataDir = next
				i++
				continue
			}
		}
		if strings.HasPrefix(tok, "--config=") {
			configPath = strings.TrimSpace(strings.TrimPrefix(tok, "--config="))
			continue
		}
		if strings.HasPrefix(tok, "--data-dir=") {
			dataDir = strings.TrimSpace(strings.TrimPrefix(tok, "--data-dir="))
			continue
		}
	}
	return configPath, dataDir
}

func rewriteArgsForConfiguredExecutableAlias(argv []string, commandAliases map[string]string) []string {
	if len(argv) == 0 || len(commandAliases) == 0 {
		return argv
	}

	binary := strings.TrimSpace(filepath.Base(argv[0]))
	if binary == "" || strings.EqualFold(binary, "kimbap") {
		return argv
	}

	target := strings.TrimSpace(commandAliases[binary])
	if target == "" {
		target = strings.TrimSpace(commandAliases[strings.ToLower(binary)])
	}
	if target == "" || !strings.Contains(target, ".") {
		return argv
	}

	out := make([]string, 0, len(argv)+2)
	out = append(out, argv[0], "call", target)
	out = append(out, argv[1:]...)
	return out
}

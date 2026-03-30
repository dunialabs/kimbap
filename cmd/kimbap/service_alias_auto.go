package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/services"
)

func ensureInstalledServiceAlias(cfg *config.KimbapConfig, installer *services.LocalInstaller, manifest *services.ServiceManifest) (string, bool, error) {
	if cfg == nil || installer == nil || manifest == nil {
		return "", false, nil
	}

	alias, alreadyConfigured, err := selectServiceAliasCandidate(cfg, installer, manifest)
	if err != nil {
		return "", false, err
	}
	if alias == "" {
		return "", false, nil
	}
	if alreadyConfigured {
		return alias, false, nil
	}

	target := strings.ToLower(strings.TrimSpace(manifest.Name))
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]string{}
	}
	configPath, resolveErr := resolveConfigPath()
	if resolveErr != nil {
		return "", false, fmt.Errorf("resolve config path: %w", resolveErr)
	}
	if err := upsertConfigAlias(configPath, alias, target); err != nil {
		return "", false, fmt.Errorf("set auto alias %q -> %q: %w", alias, target, err)
	}
	cfg.Aliases[alias] = target
	return alias, true, nil
}

func selectServiceAliasCandidate(cfg *config.KimbapConfig, installer *services.LocalInstaller, manifest *services.ServiceManifest) (string, bool, error) {
	if cfg == nil || installer == nil || manifest == nil {
		return "", false, nil
	}
	target := strings.ToLower(strings.TrimSpace(manifest.Name))
	if target == "" {
		return "", false, nil
	}

	if existing := configuredServiceCallAlias(cfg, target); existing != "" {
		return existing, true, nil
	}

	installed, err := installer.List()
	if err != nil {
		return "", false, fmt.Errorf("list installed services for aliasing: %w", err)
	}
	takenNames := make(map[string]struct{}, len(installed))
	for _, svc := range installed {
		takenNames[strings.ToLower(strings.TrimSpace(svc.Manifest.Name))] = struct{}{}
	}

	for _, candidate := range services.SuggestedServiceAliases(target, manifest.Aliases) {
		cand := strings.ToLower(strings.TrimSpace(candidate))
		if cand == "" || cand == target {
			continue
		}
		if _, taken := takenNames[cand]; taken {
			continue
		}
		if cfg.Aliases != nil {
			if aliasTarget, exists := cfg.Aliases[cand]; exists {
				if strings.ToLower(strings.TrimSpace(aliasTarget)) == target {
					return cand, true, nil
				}
				continue
			}
		}
		return cand, false, nil
	}

	return "", false, nil
}

func ensureInstalledActionAliases(cfg *config.KimbapConfig, installer *services.LocalInstaller, manifest *services.ServiceManifest) ([]string, []string, error) {
	if cfg == nil || manifest == nil {
		return nil, nil, nil
	}
	if cfg.CommandAliases == nil {
		cfg.CommandAliases = map[string]string{}
	}

	configPath, resolveErr := resolveConfigPath()
	if resolveErr != nil {
		return nil, nil, fmt.Errorf("resolve config path: %w", resolveErr)
	}

	created := make([]string, 0)
	skipped := make([]string, 0)
	targetService := strings.ToLower(strings.TrimSpace(manifest.Name))
	reservedServiceNames := map[string]struct{}{}
	if targetService != "" {
		reservedServiceNames[targetService] = struct{}{}
	}
	if installer != nil {
		if installed, listErr := installer.List(); listErr == nil {
			for _, svc := range installed {
				name := strings.ToLower(strings.TrimSpace(svc.Manifest.Name))
				if name == "" {
					continue
				}
				reservedServiceNames[name] = struct{}{}
			}
		}
	}
	actionKeys := make([]string, 0, len(manifest.Actions))
	for actionKey := range manifest.Actions {
		actionKeys = append(actionKeys, actionKey)
	}
	sort.Strings(actionKeys)
	defaultActionKey := defaultShortcutActionKey(manifest)

	for _, actionKey := range actionKeys {
		action := manifest.Actions[actionKey]
		target := targetService + "." + actionKey
		generatedCandidates := len(action.Aliases) == 0 && actionKey == defaultActionKey
		candidateAliases := make([]string, 0)
		seenCandidates := map[string]struct{}{}
		appendCandidate := func(raw string) {
			alias := strings.ToLower(strings.TrimSpace(raw))
			if alias == "" {
				return
			}
			if _, exists := seenCandidates[alias]; exists {
				return
			}
			seenCandidates[alias] = struct{}{}
			candidateAliases = append(candidateAliases, alias)
		}
		for _, rawAlias := range action.Aliases {
			appendCandidate(rawAlias)
		}
		if generatedCandidates {
			for _, generated := range generatedDefaultActionAliases(manifest, actionKey) {
				appendCandidate(generated)
			}
		}

		aliasCreatedForAction := false
		for _, rawAlias := range candidateAliases {
			alias := strings.ToLower(strings.TrimSpace(rawAlias))
			if alias == "" {
				continue
			}
			if _, reserved := reservedServiceNames[alias]; reserved {
				skipped = append(skipped, alias+" (conflicts with service name)")
				continue
			}
			if aliasConflictsWithBuiltinCommand(alias) {
				skipped = append(skipped, alias+" (builtin command name)")
				continue
			}
			if svcTarget, exists := cfg.Aliases[alias]; exists {
				if strings.ToLower(strings.TrimSpace(svcTarget)) == targetService {
					skipped = append(skipped, alias+" (used as service alias)")
					continue
				}
				skipped = append(skipped, alias+" (conflicts with service alias)")
				continue
			}
			if existing, exists := cfg.CommandAliases[alias]; exists {
				if strings.ToLower(strings.TrimSpace(existing)) == target {
					aliasCreatedForAction = true
					if generatedCandidates {
						break
					}
					continue
				}
				skipped = append(skipped, alias+" (already mapped to "+existing+")")
				continue
			}

			executablePath, executableCreated, ensureErr := ensureExecutableActionAlias(alias)
			if ensureErr != nil {
				skipped = append(skipped, alias+" ("+ensureErr.Error()+")")
				continue
			}
			if err := upsertConfigCommandAlias(configPath, alias, target); err != nil {
				if executableCreated {
					if _, cleanupErr := removeExecutableActionAlias(alias); cleanupErr != nil {
						return created, skipped, fmt.Errorf("set action alias %q -> %q: %w (rollback failed for %s: %v)", alias, target, err, executablePath, cleanupErr)
					}
				}
				return created, skipped, fmt.Errorf("set action alias %q -> %q: %w", alias, target, err)
			}
			cfg.CommandAliases[alias] = target
			created = append(created, alias)
			aliasCreatedForAction = true
			if generatedCandidates {
				break
			}
		}
		if generatedCandidates && !aliasCreatedForAction && len(candidateAliases) > 0 {
			skipped = append(skipped, actionKey+" (no collision-free alias candidate)")
		}
	}

	return created, skipped, nil
}

func defaultShortcutActionKey(manifest *services.ServiceManifest) string {
	if manifest == nil || len(manifest.Actions) == 0 {
		return ""
	}
	keys := make([]string, 0, len(manifest.Actions))
	for key := range manifest.Actions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	bestKey := ""
	bestScore := int(^uint(0) >> 1)
	for _, key := range keys {
		action := manifest.Actions[key]
		score := shortcutActionPriorityScore(key, action)
		if score < bestScore {
			bestScore = score
			bestKey = key
		}
	}
	return bestKey
}

func shortcutActionPriorityScore(actionKey string, action services.ServiceAction) int {
	name := strings.ToLower(strings.TrimSpace(actionKey))
	verbScore := 20
	switch {
	case strings.Contains(name, "search"):
		verbScore = 0
	case strings.Contains(name, "list"):
		verbScore = 1
	case strings.Contains(name, "get"):
		verbScore = 2
	case strings.Contains(name, "fetch"):
		verbScore = 3
	case strings.Contains(name, "find"):
		verbScore = 4
	}
	required := 0
	for _, arg := range action.Args {
		if arg.Required {
			required++
		}
	}
	risk := 2
	switch strings.ToLower(strings.TrimSpace(action.Risk.Level)) {
	case "low":
		risk = 0
	case "medium":
		risk = 1
	case "high":
		risk = 2
	case "critical":
		risk = 3
	}
	return verbScore*100 + required*10 + risk
}

func generatedDefaultActionAliases(manifest *services.ServiceManifest, actionKey string) []string {
	if manifest == nil {
		return nil
	}
	serviceAliases := services.SuggestedServiceAliases(manifest.Name, manifest.Aliases)
	if len(serviceAliases) == 0 {
		serviceAliases = services.SuggestedServiceAliases(manifest.Name, nil)
	}

	stems := actionAliasStemCandidates(actionKey)
	if len(serviceAliases) == 0 || len(stems) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0)
	add := func(candidate string) {
		alias := strings.ToLower(strings.TrimSpace(candidate))
		if !services.IsValidServiceAlias(alias) {
			return
		}
		if _, exists := seen[alias]; exists {
			return
		}
		seen[alias] = struct{}{}
		out = append(out, alias)
	}

	for _, svc := range serviceAliases {
		svc = strings.ToLower(strings.TrimSpace(svc))
		if svc == "" {
			continue
		}
		for _, stem := range stems {
			add(svc + stem)
		}
	}

	for _, stem := range stems {
		add(stem)
	}

	return out
}

func actionAliasStemCandidates(actionKey string) []string {
	primary := actionAliasStem(actionKey)
	if primary == "" {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	add := func(value string) {
		v := strings.ToLower(strings.TrimSpace(value))
		if v == "" {
			return
		}
		if _, exists := seen[v]; exists {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	add(primary)
	if len(primary) > 4 {
		add(primary[:4])
	}
	if len(primary) > 3 {
		add(primary[:3])
	}

	return out
}

func actionAliasStem(actionKey string) string {
	trimmed := strings.ToLower(strings.TrimSpace(actionKey))
	if trimmed == "" {
		return ""
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) == 0 {
		parts = []string{trimmed}
	}
	preferred := parts[0]
	if len(parts) > 1 {
		switch preferred {
		case "get", "list", "fetch", "find", "create", "update", "delete", "set":
			preferred = parts[1]
		}
	}
	var out strings.Builder
	for _, r := range preferred {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		}
	}
	return out.String()
}

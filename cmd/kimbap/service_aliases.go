package main

import (
	"fmt"
	"sort"
	"strings"
)

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

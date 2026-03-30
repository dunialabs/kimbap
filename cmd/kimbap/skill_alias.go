package main

import (
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/services"
)

func configuredServiceCallAlias(cfg *config.KimbapConfig, serviceName string) string {
	if cfg == nil || len(cfg.Aliases) == 0 {
		return ""
	}
	target := strings.ToLower(strings.TrimSpace(serviceName))
	if target == "" {
		return ""
	}

	candidates := make([]string, 0)
	for alias, mapped := range cfg.Aliases {
		if strings.ToLower(strings.TrimSpace(mapped)) != target {
			continue
		}
		trimmedAlias := strings.ToLower(strings.TrimSpace(alias))
		if !services.IsValidServiceAlias(trimmedAlias) {
			continue
		}
		candidates = append(candidates, trimmedAlias)
	}
	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		if len(candidates[i]) == len(candidates[j]) {
			return candidates[i] < candidates[j]
		}
		return len(candidates[i]) < len(candidates[j])
	})

	return candidates[0]
}

package services

import "strings"

func normalizedAdapterType(adapter string) string {
	normalized := strings.ToLower(strings.TrimSpace(adapter))
	if normalized == "" {
		return "http"
	}
	return normalized
}

func normalizedAuthType(authType string) string {
	return strings.ToLower(strings.TrimSpace(authType))
}

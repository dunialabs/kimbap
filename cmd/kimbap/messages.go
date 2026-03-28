package main

import (
	"fmt"
	"strings"
)

const (
	componentApprovalStore   = "approval store"
	componentConnectorStore  = "connector store"
	componentOAuthStateStore = "oauth state store"
	componentRuntime         = "runtime"
	componentVault           = "vault"
)

func unavailableMessage(component string, err error) string {
	label := strings.TrimSpace(component)
	if label == "" {
		label = "resource"
	}
	return fmt.Sprintf("%s unavailable: %v", label, err)
}

func unavailableError(component string, err error) error {
	label := strings.TrimSpace(component)
	if label == "" {
		label = "resource"
	}
	return fmt.Errorf("%s unavailable: %w", label, err)
}

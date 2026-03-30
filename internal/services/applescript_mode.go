package services

import (
	"fmt"
	"strings"
	"sync/atomic"
)

type AppleScriptRegistryMode string

const (
	AppleScriptRegistryModeLegacy   AppleScriptRegistryMode = "legacy"
	AppleScriptRegistryModeDual     AppleScriptRegistryMode = "dual"
	AppleScriptRegistryModeManifest AppleScriptRegistryMode = "manifest"
)

var appleScriptRegistryMode atomic.Value

func init() {
	appleScriptRegistryMode.Store(string(AppleScriptRegistryModeDual))
}

func CurrentAppleScriptRegistryMode() AppleScriptRegistryMode {
	raw, _ := appleScriptRegistryMode.Load().(string)
	if mode, err := NormalizeAppleScriptRegistryMode(raw); err == nil {
		return mode
	}
	return AppleScriptRegistryModeDual
}

func SetAppleScriptRegistryMode(raw string) error {
	mode, err := NormalizeAppleScriptRegistryMode(raw)
	if err != nil {
		return err
	}
	appleScriptRegistryMode.Store(string(mode))
	return nil
}

func NormalizeAppleScriptRegistryMode(raw string) (AppleScriptRegistryMode, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "", string(AppleScriptRegistryModeDual):
		return AppleScriptRegistryModeDual, nil
	case string(AppleScriptRegistryModeLegacy):
		return AppleScriptRegistryModeLegacy, nil
	case string(AppleScriptRegistryModeManifest):
		return AppleScriptRegistryModeManifest, nil
	default:
		return "", fmt.Errorf("invalid services.applescript_registry_mode value %q: must be one of: legacy, dual, manifest", raw)
	}
}

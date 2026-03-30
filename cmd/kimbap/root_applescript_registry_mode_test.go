package main

import (
	"testing"

	"github.com/dunialabs/kimbap/internal/services"
)

func TestLoadAppConfigReadOnlyAppliesAppleScriptRegistryMode(t *testing.T) {
	prevMode := services.CurrentAppleScriptRegistryMode()
	t.Cleanup(func() {
		_ = services.SetAppleScriptRegistryMode(string(prevMode))
	})

	prevOpts := opts
	t.Cleanup(func() {
		opts = prevOpts
	})
	opts = cliOptions{}

	t.Setenv("HOME", t.TempDir())
	t.Setenv("KIMBAP_SERVICES_APPLESCRIPT_REGISTRY_MODE", "manifest")

	cfg, err := loadAppConfigReadOnly()
	if err != nil {
		t.Fatalf("loadAppConfigReadOnly() error: %v", err)
	}
	if cfg.Services.AppleScriptRegistryMode != "manifest" {
		t.Fatalf("cfg.Services.AppleScriptRegistryMode = %q, want manifest", cfg.Services.AppleScriptRegistryMode)
	}
	if got := services.CurrentAppleScriptRegistryMode(); got != services.AppleScriptRegistryModeManifest {
		t.Fatalf("services.CurrentAppleScriptRegistryMode() = %q, want manifest", got)
	}
}

package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/config"
)

func TestStatusCommandReturnsLoadErrorForMissingExplicitConfig(t *testing.T) {
	prev := opts
	t.Cleanup(func() { opts = prev })

	opts = cliOptions{configPath: filepath.Join(t.TempDir(), "missing-config.yaml")}
	err := newStatusCommand().RunE(nil, nil)
	if err == nil {
		t.Fatal("expected status command to return config load error")
	}
}

func TestStatusCommandIsVisibleByDefault(t *testing.T) {
	// Verify that the status command is registered as visible in rootCmd (not hidden),
	// not just that the constructor doesn't set Hidden.
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "status" {
			if cmd.Hidden {
				t.Fatal("expected registered status command to be visible (not hidden)")
			}
			return
		}
	}
	t.Fatal("status command not found in rootCmd")
}

func TestVaultKeyAvailableReadOnlyRejectsInvalidKimbapDev(t *testing.T) {
	t.Setenv("KIMBAP_MASTER_KEY_HEX", "")
	t.Setenv("KIMBAP_DEV", "not-a-bool")

	_, err := vaultKeyAvailableReadOnly(&config.KimbapConfig{Mode: "embedded"})
	if err == nil {
		t.Fatal("expected invalid KIMBAP_DEV parse error")
	}
}

func TestVaultKeyAvailableReadOnlyRejectsMalformedMasterKeyHex(t *testing.T) {
	t.Setenv("KIMBAP_MASTER_KEY_HEX", "xyz")
	_, err := vaultKeyAvailableReadOnly(&config.KimbapConfig{Mode: "embedded"})
	if err == nil {
		t.Fatal("expected malformed KIMBAP_MASTER_KEY_HEX error")
	}
}

func TestVaultStatusFromRawMalformedMasterKeyHexReturnsError(t *testing.T) {
	t.Setenv("KIMBAP_MASTER_KEY_HEX", "xyz")
	if got := vaultStatusFromRaw("embedded"); got != "error" {
		t.Fatalf("expected error, got %q", got)
	}
}

func TestVaultStatusStringMalformedMasterKeyHexReturnsErrorWhenVaultMissing(t *testing.T) {
	t.Setenv("KIMBAP_MASTER_KEY_HEX", "xyz")
	cfg := &config.KimbapConfig{}
	cfg.Vault.Path = filepath.Join(t.TempDir(), "missing-vault.db")
	if got := vaultStatusString(cfg); got != "error" {
		t.Fatalf("expected error, got %q", got)
	}
}

func TestClassifyVaultInitError(t *testing.T) {
	if got := classifyVaultInitError(errors.New("vault master key is required")); got != "vault_locked" {
		t.Fatalf("expected vault_locked, got %q", got)
	}
	if got := classifyVaultInitError(errors.New("decode KIMBAP_MASTER_KEY_HEX: encoding/hex: invalid byte")); got != "vault_error" {
		t.Fatalf("expected vault_error for malformed master key, got %q", got)
	}
	if got := classifyVaultInitError(errors.New("open sqlite: permission denied")); got != "vault_error" {
		t.Fatalf("expected vault_error, got %q", got)
	}
}

func TestRenderStatusSummaryNoServicesHint(t *testing.T) {
	summary := statusSummary{Services: 0, Credentials: 0, Agents: 0}
	out := renderStatusSummary(summary, nil)
	if !strings.Contains(out, "kimbap init") {
		t.Fatalf("expected init hint when no services, got:\n%s", out)
	}
}

func TestRenderStatusSummaryAgentHintWhenNoCredentialServices(t *testing.T) {
	summary := statusSummary{Services: 1, Credentials: 0, Agents: 0}
	out := renderStatusSummary(summary, nil)
	if !strings.Contains(out, "kimbap agents setup") {
		t.Fatalf("expected agents hint when no key-based services, got:\n%s", out)
	}
}

func TestRenderStatusSummaryNoHintWhenAllConnected(t *testing.T) {
	summary := statusSummary{Services: 3, Credentials: 2, Agents: 1}
	out := renderStatusSummary(summary, nil)
	if strings.Contains(out, "Run '") {
		t.Fatalf("expected no hint when all connected, got:\n%s", out)
	}
}

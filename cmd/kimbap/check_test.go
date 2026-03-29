package main

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	corecrypto "github.com/dunialabs/kimbap/internal/crypto"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/internal/vault"
)

func TestRunActionPreflightPolicyMutatingUsesIdempotent(t *testing.T) {
	servicesDir := t.TempDir()
	policyPath := filepath.Join(t.TempDir(), "policy.yaml")
	vaultPath := filepath.Join(t.TempDir(), "vault.db")

	idempotentFalse := false
	manifest := &services.ServiceManifest{
		Name:    "demo",
		Version: "1.0.0",
		Adapter: "http",
		BaseURL: "https://example.com",
		Auth: services.ServiceAuth{
			Type: string(actions.AuthTypeNone),
		},
		Actions: map[string]services.ServiceAction{
			"noop": {
				Method:      "GET",
				Path:        "/noop",
				Description: "noop",
				Idempotent:  &idempotentFalse,
				Risk: services.RiskSpec{
					Level: "low",
				},
				Response: services.ResponseSpec{Type: "object"},
			},
		},
	}
	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install service: %v", err)
	}

	policyDoc := `
version: "1.0.0"
rules:
  - id: deny-mutating
    priority: 1
    match:
      actions: ["demo.noop"]
    decision: deny
    conditions:
      - field: risk.mutating
        operator: eq
        value: true
  - id: allow-demo-noop
    priority: 100
    match:
      actions: ["demo.noop"]
    decision: allow
`
	if err := os.WriteFile(policyPath, []byte(policyDoc), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	cfg := &config.KimbapConfig{Mode: "embedded", DataDir: t.TempDir()}
	cfg.Services.Dir = servicesDir
	cfg.Policy.Path = policyPath
	cfg.Vault.Path = vaultPath

	def, err := resolveActionByName(cfg, "demo.noop")
	if err != nil {
		t.Fatalf("resolve action: %v", err)
	}
	if def.Idempotent {
		t.Fatal("expected explicit idempotent=false manifest field to be preserved")
	}

	report, err := runActionPreflight(cfg, "demo.noop")
	if err != nil {
		t.Fatalf("run preflight: %v", err)
	}
	if report.Verdict != "not_ready" {
		t.Fatalf("expected not_ready when idempotent=false implies mutating=true, got %q", report.Verdict)
	}
	foundDeny := false
	for _, blocker := range report.Blockers {
		if blocker == "policy_denied" {
			foundDeny = true
			break
		}
	}
	if !foundDeny {
		t.Fatalf("expected policy_denied blocker, got %#v", report.Blockers)
	}
}

func TestProbeCredentialReadOnlyLockedWithoutMasterKey(t *testing.T) {
	t.Setenv("KIMBAP_MASTER_KEY_HEX", "")
	t.Setenv("KIMBAP_DEV", "false")

	cfg := &config.KimbapConfig{Mode: "embedded", DataDir: t.TempDir()}
	cfg.Vault.Path = filepath.Join(t.TempDir(), "missing-vault.db")

	check := probeCredentialReadOnly(cfg, "demo.token")
	if check.Status != "fail" {
		t.Fatalf("expected fail, got %q", check.Status)
	}
	if !strings.Contains(check.Detail, "vault is locked") {
		t.Fatalf("expected vault locked detail, got %q", check.Detail)
	}
}

func TestProbeCredentialReadOnlyFailsWhenDecryptUnavailable(t *testing.T) {
	goodKey := bytes.Repeat([]byte{0x11}, 32)
	badKey := bytes.Repeat([]byte{0x22}, 32)

	vaultPath := filepath.Join(t.TempDir(), "vault.db")
	envelope, err := corecrypto.NewEnvelopeService(goodKey)
	if err != nil {
		t.Fatalf("new envelope: %v", err)
	}
	st, err := vault.OpenSQLiteStore(vaultPath, envelope)
	if err != nil {
		t.Fatalf("open vault store: %v", err)
	}
	if _, err := st.Upsert(contextBackground(), defaultTenantID(), "demo.token", vault.SecretTypeAPIKey, []byte("secret-value"), nil, "test"); err != nil {
		_ = st.Close()
		t.Fatalf("upsert secret: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close vault store: %v", err)
	}

	t.Setenv("KIMBAP_DEV", "false")
	t.Setenv("KIMBAP_MASTER_KEY_HEX", hex.EncodeToString(badKey))

	cfg := &config.KimbapConfig{Mode: "embedded", DataDir: t.TempDir()}
	cfg.Vault.Path = vaultPath

	check := probeCredentialReadOnly(cfg, "demo.token")
	if check.Status != "fail" {
		t.Fatalf("expected fail, got %q", check.Status)
	}
	if !strings.Contains(check.Detail, "vault is unavailable") {
		t.Fatalf("expected vault unavailable detail for undecryptable secret, got %q", check.Detail)
	}
}

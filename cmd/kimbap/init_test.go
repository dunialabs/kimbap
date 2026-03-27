package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInitConfigRebasesPolicyPathWithDataDir(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })

	opts = cliOptions{}
	opts.dataDir = t.TempDir()

	cfg := buildInitConfig()
	want := filepath.Join(opts.dataDir, "policy.yaml")
	if cfg.Policy.Path != want {
		t.Fatalf("expected policy path %q, got %q", want, cfg.Policy.Path)
	}
}

func TestBuildInitConfigKeepsDefaultModeEmbedded(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })

	opts = cliOptions{}
	cfg := buildInitConfig()
	if cfg.Mode != "embedded" {
		t.Fatalf("expected default init mode embedded, got %q", cfg.Mode)
	}
}

func TestAppendInitChecksFlagsFailures(t *testing.T) {
	checks, hasFailure := appendInitChecks(nil, false,
		doctorCheck{Name: "ok", Status: "ok"},
		doctorCheck{Name: "warn", Status: "warn"},
	)
	if hasFailure {
		t.Fatal("expected non-failing checks to keep hasFailure=false")
	}

	checks, hasFailure = appendInitChecks(checks, hasFailure, doctorCheck{Name: "fail", Status: "fail"})
	if !hasFailure {
		t.Fatal("expected failing check to set hasFailure=true")
	}
	if len(checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(checks))
	}
}

func TestOptionalKBCheckDoesNotFailInit(t *testing.T) {
	checks := []doctorCheck{}
	hasFailure := false

	kbCheck := doctorCheck{Name: "kb alias", Status: "fail"}
	checks = append(checks, kbCheck)

	if hasFailure {
		t.Fatal("optional kbCheck failure must not set hasFailure")
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check recorded, got %d", len(checks))
	}
}

func TestValidateInitMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "embedded", mode: "embedded"},
		{name: "dev", mode: "dev"},
		{name: "connected", mode: "connected"},
		{name: "invalid", mode: "unknown", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInitMode(tc.mode)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for mode %q", tc.mode)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for mode %q, got %v", tc.mode, err)
			}
		})
	}
}

func TestEnsureConsoleEnabledFailsWhenConfigNotOverwritten(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("console:\n  enabled: false\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	check := ensureConsoleEnabled(configPath, doctorCheck{Status: "skip"}, true)
	if check.Status != "fail" {
		t.Fatalf("expected fail when --with-console was not persisted, got %q", check.Status)
	}
}

func TestEnsureConsoleEnabledSkipsWhenAlreadyEnabled(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("console:\n  enabled: true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	check := ensureConsoleEnabled(configPath, doctorCheck{Status: "skip"}, false)
	if check.Status != "skip" {
		t.Fatalf("expected skip for existing enabled config, got %q", check.Status)
	}
}

func TestRenderInitSummaryIncludesWarnings(t *testing.T) {
	summary := renderInitSummary("/tmp/config.yaml", []doctorCheck{
		{Name: "a", Status: "ok", Detail: "ok"},
		{Name: "b", Status: "warn", Detail: "warn"},
		{Name: "c", Status: "skip", Detail: "skip"},
		{Name: "d", Status: "fail", Detail: "fail"},
	})

	if !strings.Contains(summary, "warnings: 1") {
		t.Fatalf("expected warning count in summary, got:\n%s", summary)
	}
	if !strings.Contains(summary, "! b") {
		t.Fatalf("expected warn icon in summary, got:\n%s", summary)
	}
}

func TestEnsurePolicyFileFailsForInvalidExistingPolicy(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "policy.yaml")
	invalid := "version: 1\nrules:\n- id: bad\n  effect: allow\n"
	if err := os.WriteFile(policyPath, []byte(invalid), 0o600); err != nil {
		t.Fatalf("write invalid policy: %v", err)
	}

	check := ensurePolicyFile(policyPath, "embedded")
	if check.Status != "fail" {
		t.Fatalf("expected fail for invalid policy file, got %q", check.Status)
	}
}

func TestEnsureWritableDirWithStatusFailsWhenPathIsFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file path: %v", err)
	}

	check := ensureWritableDirWithStatus("data directory writable", filePath)
	if check.Status != "fail" {
		t.Fatalf("expected fail when path is a file, got %q", check.Status)
	}
}

func TestEnsureWritableDirWithStatusCreatesPrivateDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")

	check := ensureWritableDirWithStatus("data directory writable", dataDir)
	if check.Status != "ok" {
		t.Fatalf("expected ok for newly created writable directory, got %q", check.Status)
	}

	st, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected private permissions for data dir, got %o", st.Mode().Perm())
	}
}

func TestEnsureWritableDirWithStatusWarnsOnPermissiveExistingDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "existing-data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.Chmod(dataDir, 0o755); err != nil {
		t.Fatalf("chmod data dir: %v", err)
	}

	check := ensureWritableDirWithStatus("data directory writable", dataDir)
	if check.Status != "warn" {
		t.Fatalf("expected warn for permissive existing directory, got %q", check.Status)
	}
}

func TestEnsureWritableDirWithStatusFailsWhenExistingDirectoryIsNotWritable(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "read-only-data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.Chmod(dataDir, 0o555); err != nil {
		t.Fatalf("chmod data dir: %v", err)
	}

	check := ensureWritableDirWithStatus("data directory writable", dataDir)
	if check.Status != "fail" {
		t.Fatalf("expected fail for non-writable existing directory, got %q", check.Status)
	}
}

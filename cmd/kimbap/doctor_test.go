package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/config"
)

func TestCheckConfigFileUsesExplicitConfigWithoutDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("mkdir default config dir: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte("mode: [\n"), 0o644); err != nil {
		t.Fatalf("write broken default config: %v", err)
	}

	explicitPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(explicitPath, []byte("mode: connected\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	prev := opts.configPath
	opts.configPath = explicitPath
	t.Cleanup(func() {
		opts.configPath = prev
	})

	check := checkConfigFile()
	if check.Status != "ok" {
		t.Fatalf("expected ok status, got %s (%s)", check.Status, check.Detail)
	}
	if check.Detail != explicitPath {
		t.Fatalf("expected explicit path %q, got %q", explicitPath, check.Detail)
	}
}

func TestCheckVaultAccessibleDoesNotCreateMissingVault(t *testing.T) {
	vaultPath := filepath.Join(t.TempDir(), "vault.db")

	check := checkVaultAccessible(&config.KimbapConfig{
		Vault: config.VaultConfig{Path: vaultPath},
	})
	if check.Status != "fail" {
		t.Fatalf("expected fail status, got %s (%s)", check.Status, check.Detail)
	}
	if _, err := os.Stat(vaultPath); !os.IsNotExist(err) {
		t.Fatalf("expected missing vault file to remain absent, stat err=%v", err)
	}
}

func TestDoctorCommandUsesExplicitConfigWithoutBrokenDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("mkdir default config dir: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte("mode: [\n"), 0o644); err != nil {
		t.Fatalf("write broken default config: %v", err)
	}

	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "services"), 0o755); err != nil {
		t.Fatalf("mkdir services dir: %v", err)
	}
	vaultPath := filepath.Join(dataDir, "vault.db")
	createTestVaultDB(t, vaultPath)

	explicitPath := filepath.Join(t.TempDir(), "config.yaml")
	configData := "mode: embedded\n" +
		"data_dir: " + dataDir + "\n" +
		"vault:\n  path: " + vaultPath + "\n" +
		"services:\n  dir: " + filepath.Join(dataDir, "services") + "\n"
	if err := os.WriteFile(explicitPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	prevOpts := opts
	opts = cliOptions{configPath: explicitPath, format: "json"}
	t.Cleanup(func() {
		opts = prevOpts
	})

	err := newDoctorCommand().RunE(nil, nil)
	if err != nil {
		t.Fatalf("doctor command with explicit config failed: %v", err)
	}
}

func TestDoctorCommandDoesNotCreateMissingDataDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("mkdir default config dir: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte("mode: [\n"), 0o644); err != nil {
		t.Fatalf("write broken default config: %v", err)
	}

	dataDir := filepath.Join(t.TempDir(), "missing-data")
	explicitPath := filepath.Join(t.TempDir(), "config.yaml")
	configData := "mode: embedded\n" +
		"data_dir: " + dataDir + "\n"
	if err := os.WriteFile(explicitPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	prevOpts := opts
	opts = cliOptions{configPath: explicitPath, format: "json"}
	t.Cleanup(func() {
		opts = prevOpts
	})

	err := newDoctorCommand().RunE(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "doctor found failing checks") {
		t.Fatalf("expected doctor failing checks error, got %v", err)
	}
	if _, statErr := os.Stat(dataDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected missing data dir to remain absent, stat err=%v", statErr)
	}
}

func TestRenderDoctorSummary(t *testing.T) {
	checks := []doctorCheck{
		{Name: "config file", Status: "ok", Detail: "/home/.kimbap/config.yaml"},
		{Name: "data directory writable", Status: "ok", Detail: "/home/.kimbap"},
		{Name: "vault accessible", Status: "fail", Detail: "vault path is empty"},
		{Name: "services directory exists", Status: "skip", Detail: "no such file or directory"},
		{Name: "policy file valid", Status: "warn", Detail: "permissions are 755"},
	}

	got := renderDoctorSummary(checks)

	if !strings.Contains(got, "Kimbap runtime diagnostics") {
		t.Errorf("expected header, got:\n%s", got)
	}
	if !strings.Contains(got, "passed: 2") {
		t.Errorf("expected passed: 2, got:\n%s", got)
	}
	if !strings.Contains(got, "failed: 1") {
		t.Errorf("expected failed: 1, got:\n%s", got)
	}
	if !strings.Contains(got, "skipped: 1") {
		t.Errorf("expected skipped: 1, got:\n%s", got)
	}
	if !strings.Contains(got, "warnings: 1") {
		t.Errorf("expected warnings: 1, got:\n%s", got)
	}
	if !strings.Contains(got, "✓ config file") {
		t.Errorf("expected ok icon for config file, got:\n%s", got)
	}
	if !strings.Contains(got, "✗ vault accessible") {
		t.Errorf("expected fail icon for vault, got:\n%s", got)
	}
	if !strings.Contains(got, "- services directory exists") {
		t.Errorf("expected skip icon for services, got:\n%s", got)
	}
	if !strings.Contains(got, "! policy file valid") {
		t.Errorf("expected warn icon for policy, got:\n%s", got)
	}
}

func TestDoctorCommandIncludesNewChecks(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		t.Fatalf("mkdir services dir: %v", err)
	}

	vaultPath := filepath.Join(dataDir, "vault.db")
	createTestVaultDB(t, vaultPath)

	explicitPath := filepath.Join(t.TempDir(), "config.yaml")
	configData := "mode: embedded\n" +
		"data_dir: " + dataDir + "\n" +
		"vault:\n  path: " + vaultPath + "\n" +
		"services:\n  dir: " + servicesDir + "\n"
	if err := os.WriteFile(explicitPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	prevOpts := opts
	opts = cliOptions{configPath: explicitPath, format: "json"}
	t.Cleanup(func() {
		opts = prevOpts
	})

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})

	runErr := newDoctorCommand().RunE(nil, nil)
	_ = w.Close()
	if runErr != nil {
		t.Fatalf("doctor command failed: %v", runErr)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read doctor output: %v", err)
	}

	var checks []map[string]any
	if err := json.Unmarshal(out, &checks); err != nil {
		t.Fatalf("unmarshal doctor json output: %v\noutput=%s", err, string(out))
	}
	if len(checks) < 8 {
		t.Fatalf("expected at least 8 checks, got %d", len(checks))
	}
}

func createTestVaultDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE secrets (id INTEGER PRIMARY KEY);`); err != nil {
		t.Fatalf("create secrets table: %v", err)
	}
}

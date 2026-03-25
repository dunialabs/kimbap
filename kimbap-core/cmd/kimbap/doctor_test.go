package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap-core/internal/config"
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
	if err := os.MkdirAll(filepath.Join(dataDir, "skills"), 0o755); err != nil {
		t.Fatalf("mkdir skills dir: %v", err)
	}
	vaultPath := filepath.Join(dataDir, "vault.db")
	createTestVaultDB(t, vaultPath)

	explicitPath := filepath.Join(t.TempDir(), "config.yaml")
	configData := "mode: embedded\n" +
		"data_dir: " + dataDir + "\n" +
		"vault:\n  path: " + vaultPath + "\n" +
		"skills:\n  dir: " + filepath.Join(dataDir, "skills") + "\n"
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

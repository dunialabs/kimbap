package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppConfigReadOnlyUsesDataDirConfigWhenProvided(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("mkdir default config dir: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte("mode: [\n"), 0o644); err != nil {
		t.Fatalf("write broken default config: %v", err)
	}

	dataDir := filepath.Join(t.TempDir(), "isolated")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	dataDirConfig := filepath.Join(dataDir, "config.yaml")
	if err := os.WriteFile(dataDirConfig, []byte("mode: connected\n"), 0o644); err != nil {
		t.Fatalf("write data-dir config: %v", err)
	}

	prevOpts := opts
	opts = cliOptions{dataDir: dataDir}
	t.Cleanup(func() {
		opts = prevOpts
	})

	cfg, err := loadAppConfigReadOnly()
	if err != nil {
		t.Fatalf("loadAppConfigReadOnly() error: %v", err)
	}
	if cfg.DataDir != dataDir {
		t.Fatalf("expected data dir %q, got %q", dataDir, cfg.DataDir)
	}
	if cfg.Mode != "connected" {
		t.Fatalf("expected mode loaded from data-dir config, got %q", cfg.Mode)
	}
}

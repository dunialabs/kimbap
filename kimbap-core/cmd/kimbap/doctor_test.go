package main

import (
	"os"
	"path/filepath"
	"testing"
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

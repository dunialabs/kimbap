package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigHasCoreDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Mode != "embedded" {
		t.Fatalf("unexpected mode default: %q", cfg.Mode)
	}
	if cfg.Auth.TokenTTL != "720h" {
		t.Fatalf("unexpected auth token ttl default: %q", cfg.Auth.TokenTTL)
	}
	if cfg.Auth.SessionTTL != "15m" {
		t.Fatalf("unexpected session ttl default: %q", cfg.Auth.SessionTTL)
	}
}

func TestLoadKimbapConfigPrecedenceDefaultEnvExplicit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultConfigDir := filepath.Join(home, ".kimbap")
	if err := os.MkdirAll(defaultConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir default config dir: %v", err)
	}

	defaultPath := filepath.Join(defaultConfigDir, "config.yaml")
	if err := os.WriteFile(defaultPath, []byte("mode: connected\nauth:\n  token_ttl: 48h\nlog_level: warn\n"), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	t.Setenv("KIMBAP_MODE", "embedded")
	t.Setenv("KIMBAP_LOG_LEVEL", "error")

	explicitPath := filepath.Join(t.TempDir(), "override.yaml")
	if err := os.WriteFile(explicitPath, []byte("mode: connected\nlog_level: debug\nauth:\n  token_ttl: 24h\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfig(explicitPath)
	if err != nil {
		t.Fatalf("load kimbap config: %v", err)
	}

	if cfg.Mode != "connected" {
		t.Fatalf("explicit file should win for mode, got %q", cfg.Mode)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("explicit file should win for log level, got %q", cfg.LogLevel)
	}
	if cfg.Auth.TokenTTL != "24h" {
		t.Fatalf("explicit file should win for auth token ttl, got %q", cfg.Auth.TokenTTL)
	}
}

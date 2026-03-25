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

func TestDefaultKimbapConfigPathPrefersExistingXDGPath(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	xdgPath := filepath.Join(xdg, "kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(xdgPath), 0o755); err != nil {
		t.Fatalf("mkdir xdg config dir: %v", err)
	}
	if err := os.WriteFile(xdgPath, []byte("mode: embedded\n"), 0o644); err != nil {
		t.Fatalf("write xdg config file: %v", err)
	}

	path, err := defaultKimbapConfigPath()
	if err != nil {
		t.Fatalf("defaultKimbapConfigPath: %v", err)
	}
	if path != xdgPath {
		t.Fatalf("expected xdg path %q, got %q", xdgPath, path)
	}
}

func TestDefaultKimbapConfigPathFallsBackToLegacyWhenXDGMissing(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	legacyPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("mkdir legacy config dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("mode: embedded\n"), 0o644); err != nil {
		t.Fatalf("write legacy config file: %v", err)
	}

	path, err := defaultKimbapConfigPath()
	if err != nil {
		t.Fatalf("defaultKimbapConfigPath: %v", err)
	}
	if path != legacyPath {
		t.Fatalf("expected legacy path %q, got %q", legacyPath, path)
	}
}

func TestDefaultKimbapConfigPathIgnoresDirectoryEntries(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	xdgPath := filepath.Join(xdg, "kimbap", "config.yaml")
	if err := os.MkdirAll(xdgPath, 0o755); err != nil {
		t.Fatalf("mkdir xdg config directory entry: %v", err)
	}

	legacyPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("mkdir legacy config dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("mode: embedded\n"), 0o644); err != nil {
		t.Fatalf("write legacy config file: %v", err)
	}

	path, err := defaultKimbapConfigPath()
	if err != nil {
		t.Fatalf("defaultKimbapConfigPath: %v", err)
	}
	if path != legacyPath {
		t.Fatalf("expected legacy file path %q, got %q", legacyPath, path)
	}
}

func TestDefaultKimbapConfigPathErrorsWhenXDGEntryIsDirectoryWithoutLegacy(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	xdgPath := filepath.Join(xdg, "kimbap", "config.yaml")
	if err := os.MkdirAll(xdgPath, 0o755); err != nil {
		t.Fatalf("mkdir xdg config directory entry: %v", err)
	}

	_, err := defaultKimbapConfigPath()
	if err == nil {
		t.Fatal("expected error when xdg config path is a directory and no legacy file exists")
	}
}

func TestDefaultKimbapConfigPathReturnsXDGPathWhenXDGMissingAndLegacyMissing(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	path, err := defaultKimbapConfigPath()
	if err != nil {
		t.Fatalf("defaultKimbapConfigPath: %v", err)
	}
	expected := filepath.Join(xdg, "kimbap", "config.yaml")
	if path != expected {
		t.Fatalf("expected xdg path %q when both files are missing, got %q", expected, path)
	}
}

func TestLoadKimbapConfigWithoutDefaultIgnoresBrokenDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("mkdir default config dir: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte("mode: [\n"), 0o644); err != nil {
		t.Fatalf("write broken default config: %v", err)
	}

	explicitPath := filepath.Join(t.TempDir(), "explicit.yaml")
	if err := os.WriteFile(explicitPath, []byte("mode: connected\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(explicitPath)
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}
	if cfg.Mode != "connected" {
		t.Fatalf("expected explicit config mode, got %q", cfg.Mode)
	}
}

func TestLoadKimbapConfigRebasesDerivedPathsWhenExplicitDataDirChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dataDir := filepath.Join(t.TempDir(), "custom-data")
	explicitPath := filepath.Join(t.TempDir(), "explicit.yaml")
	if err := os.WriteFile(explicitPath, []byte("data_dir: "+dataDir+"\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(explicitPath)
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}

	if cfg.Vault.Path != filepath.Join(dataDir, "vault.db") {
		t.Fatalf("expected rebased vault path, got %q", cfg.Vault.Path)
	}
	if cfg.Audit.Path != filepath.Join(dataDir, "audit.jsonl") {
		t.Fatalf("expected rebased audit path, got %q", cfg.Audit.Path)
	}
	if cfg.Policy.Path != filepath.Join(dataDir, "policy.yaml") {
		t.Fatalf("expected rebased policy path, got %q", cfg.Policy.Path)
	}
	if cfg.Skills.Dir != filepath.Join(dataDir, "skills") {
		t.Fatalf("expected rebased skills dir, got %q", cfg.Skills.Dir)
	}
	if cfg.Database.DSN != filepath.Join(dataDir, "kimbap.db") {
		t.Fatalf("expected rebased database dsn, got %q", cfg.Database.DSN)
	}
}

func TestLoadKimbapConfigRebasesDerivedPathsWhenEnvDataDirChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dataDir := filepath.Join(t.TempDir(), "env-data")
	t.Setenv("KIMBAP_DATA_DIR", dataDir)

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}

	if cfg.Vault.Path != filepath.Join(dataDir, "vault.db") {
		t.Fatalf("expected rebased vault path, got %q", cfg.Vault.Path)
	}
	if cfg.Skills.Dir != filepath.Join(dataDir, "skills") {
		t.Fatalf("expected rebased skills dir, got %q", cfg.Skills.Dir)
	}
}

func TestLoadKimbapConfigPreservesExplicitPathOverridesWhenDataDirChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dataDir := filepath.Join(t.TempDir(), "custom-data")
	customVaultPath := filepath.Join(t.TempDir(), "custom-vault.db")
	explicitPath := filepath.Join(t.TempDir(), "explicit.yaml")
	content := "data_dir: " + dataDir + "\n" +
		"vault:\n" +
		"  path: " + customVaultPath + "\n"
	if err := os.WriteFile(explicitPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(explicitPath)
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}

	if cfg.Vault.Path != customVaultPath {
		t.Fatalf("expected custom vault path preserved, got %q", cfg.Vault.Path)
	}
	if cfg.Policy.Path != filepath.Join(dataDir, "policy.yaml") {
		t.Fatalf("expected policy path rebased, got %q", cfg.Policy.Path)
	}
}

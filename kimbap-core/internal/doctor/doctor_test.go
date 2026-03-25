package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunAllReturnsResultsForAllChecks(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "skills"), 0o755); err != nil {
		t.Fatalf("create skills dir: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(validConfigYAML(dataDir)), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	d := NewDoctor(dataDir, configPath)
	results := d.RunAll(context.Background())
	if len(results) != 7 {
		t.Fatalf("expected 7 checks, got %d", len(results))
	}
}

func TestRunAllMissingDataDirReturnsFail(t *testing.T) {
	baseDir := t.TempDir()
	missingDataDir := filepath.Join(baseDir, "does-not-exist")
	configPath := filepath.Join(baseDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(validConfigYAML(missingDataDir)), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	d := NewDoctor("", configPath)
	results := d.RunAll(context.Background())
	dataCheck := findCheck(results, "data directory writable")
	if dataCheck == nil {
		t.Fatal("missing data directory check")
	}
	if dataCheck.Status != "fail" {
		t.Fatalf("expected fail status, got %s", dataCheck.Status)
	}
}

func TestRunAllValidConfigReturnsOK(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "skills"), 0o755); err != nil {
		t.Fatalf("create skills dir: %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(validConfigYAML(dataDir)), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	d := NewDoctor("", configPath)
	results := d.RunAll(context.Background())
	configCheck := findCheck(results, "config file")
	if configCheck == nil {
		t.Fatal("missing config file check")
	}
	if configCheck.Status != "ok" {
		t.Fatalf("expected ok status, got %s (%s)", configCheck.Status, configCheck.Message)
	}
}

func findCheck(results []CheckResult, name string) *CheckResult {
	for i := range results {
		if results[i].Name == name {
			return &results[i]
		}
	}
	return nil
}

func validConfigYAML(dataDir string) string {
	return "mode: embedded\n" +
		"data_dir: " + dataDir + "\n" +
		"vault:\n" +
		"  path: " + filepath.Join(dataDir, "vault.db") + "\n" +
		"skills:\n" +
		"  dir: " + filepath.Join(dataDir, "skills") + "\n" +
		"auth:\n" +
		"  server_url: https://example.com\n"
}

func TestResolveConfigPathPrefersExistingXDGPath(t *testing.T) {
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

	d := NewDoctor("", "")
	path, err := d.resolveConfigPath()
	if err != nil {
		t.Fatalf("resolveConfigPath: %v", err)
	}
	if path != xdgPath {
		t.Fatalf("expected xdg path %q, got %q", xdgPath, path)
	}
}

func TestResolveConfigPathFallsBackToLegacyWhenXDGMissing(t *testing.T) {
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

	d := NewDoctor("", "")
	path, err := d.resolveConfigPath()
	if err != nil {
		t.Fatalf("resolveConfigPath: %v", err)
	}
	if path != legacyPath {
		t.Fatalf("expected legacy path %q, got %q", legacyPath, path)
	}
}

func TestResolveConfigPathIgnoresDirectoryEntries(t *testing.T) {
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

	d := NewDoctor("", "")
	path, err := d.resolveConfigPath()
	if err != nil {
		t.Fatalf("resolveConfigPath: %v", err)
	}
	if path != legacyPath {
		t.Fatalf("expected legacy file path %q, got %q", legacyPath, path)
	}
}

func TestResolveConfigPathErrorsWhenXDGEntryIsDirectoryWithoutLegacy(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	xdgPath := filepath.Join(xdg, "kimbap", "config.yaml")
	if err := os.MkdirAll(xdgPath, 0o755); err != nil {
		t.Fatalf("mkdir xdg config directory entry: %v", err)
	}

	d := NewDoctor("", "")
	_, err := d.resolveConfigPath()
	if err == nil {
		t.Fatal("expected error when xdg config path is a directory and no legacy file exists")
	}
}

func TestResolveConfigPathReturnsXDGPathWhenXDGMissingAndLegacyMissing(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	d := NewDoctor("", "")
	path, err := d.resolveConfigPath()
	if err != nil {
		t.Fatalf("resolveConfigPath: %v", err)
	}
	expected := filepath.Join(xdg, "kimbap", "config.yaml")
	if path != expected {
		t.Fatalf("expected xdg path %q when both files are missing, got %q", expected, path)
	}
}

func TestRunAllWithExplicitConfigIgnoresBrokenDefaultConfig(t *testing.T) {
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
		t.Fatalf("create skills dir: %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(validConfigYAML(dataDir)), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	d := NewDoctor("", configPath)
	results := d.RunAll(context.Background())
	configCheck := findCheck(results, "config file")
	if configCheck == nil {
		t.Fatal("missing config file check")
	}
	if configCheck.Status != "ok" {
		t.Fatalf("expected ok status, got %s (%s)", configCheck.Status, configCheck.Message)
	}
}

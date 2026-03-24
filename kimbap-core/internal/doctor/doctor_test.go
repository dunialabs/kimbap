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

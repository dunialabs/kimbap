package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/dunialabs/kimbap/internal/services"
	_ "modernc.org/sqlite"
)

func writeMinimalConfig(t *testing.T, configPath, missingDataDir, servicesDir string) {
	t.Helper()
	cfgRaw := "mode: embedded\n" +
		"data_dir: " + missingDataDir + "\n" +
		"vault:\n" +
		"  path: " + filepath.Join(missingDataDir, "vault.db") + "\n" +
		"services:\n" +
		"  dir: " + servicesDir + "\n" +
		"policy:\n" +
		"  path: " + filepath.Join(missingDataDir, "policy.yaml") + "\n"
	if err := os.WriteFile(configPath, []byte(cfgRaw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func installMinimalService(t *testing.T, servicesDir string) {
	t.Helper()
	manifest := &services.ServiceManifest{
		Name:    "testservice",
		Version: "1.0.0",
		Adapter: "http",
		BaseURL: "https://example.com",
		Auth:    services.ServiceAuth{Type: "none"},
		Actions: map[string]services.ServiceAction{
			"ping": {
				Method:      "GET",
				Path:        "/ping",
				Description: "ping",
				Risk:        services.RiskSpec{Level: "low"},
				Response:    services.ResponseSpec{Type: "object"},
			},
		},
	}
	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install test service: %v", err)
	}
}

func TestSearchDoesNotMaterializeDataDir(t *testing.T) {
	servicesDir := t.TempDir()
	missingDataDir := filepath.Join(t.TempDir(), "missing-data-dir")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	installMinimalService(t, servicesDir)
	writeMinimalConfig(t, configPath, missingDataDir, servicesDir)

	prev := opts
	opts = cliOptions{configPath: configPath}
	t.Cleanup(func() { opts = prev })

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"ping"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("search command failed: %v", err)
	}

	if _, err := os.Stat(missingDataDir); !os.IsNotExist(err) {
		t.Fatalf("search must not create data_dir, stat err=%v", err)
	}
}

func TestPolicyGetDoesNotMaterializeDataDir(t *testing.T) {
	servicesDir := t.TempDir()
	missingDataDir := filepath.Join(t.TempDir(), "missing-data-dir")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	writeMinimalConfig(t, configPath, missingDataDir, servicesDir)

	prev := opts
	opts = cliOptions{configPath: configPath}
	t.Cleanup(func() { opts = prev })

	cmd := newPolicyCommand()
	cmd.SetArgs([]string{"get"})
	_ = cmd.Execute()

	if _, err := os.Stat(missingDataDir); !os.IsNotExist(err) {
		t.Fatalf("policy get must not create data_dir, stat err=%v", err)
	}
}

func TestPolicyEvalDoesNotMaterializeDataDir(t *testing.T) {
	servicesDir := t.TempDir()
	missingDataDir := filepath.Join(t.TempDir(), "missing-data-dir")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	writeMinimalConfig(t, configPath, missingDataDir, servicesDir)

	prev := opts
	opts = cliOptions{configPath: configPath}
	t.Cleanup(func() { opts = prev })

	cmd := newPolicyCommand()
	cmd.SetArgs([]string{"eval", "--agent", "test-agent", "--action", "testservice.ping"})
	_ = cmd.Execute()

	if _, err := os.Stat(missingDataDir); !os.IsNotExist(err) {
		t.Fatalf("policy eval must not create data_dir, stat err=%v", err)
	}
}

func TestAuthListDoesNotMaterializeDataDir(t *testing.T) {
	servicesDir := t.TempDir()
	missingDataDir := filepath.Join(t.TempDir(), "missing-data-dir")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	writeMinimalConfig(t, configPath, missingDataDir, servicesDir)

	prev := opts
	opts = cliOptions{configPath: configPath}
	t.Cleanup(func() { opts = prev })

	cmd := newAuthCommand()
	cmd.SetArgs([]string{"list"})
	_ = cmd.Execute()

	if _, err := os.Stat(missingDataDir); !os.IsNotExist(err) {
		t.Fatalf("auth list must not create data_dir, stat err=%v", err)
	}
}

func TestAuthStatusDoesNotMaterializeDataDir(t *testing.T) {
	servicesDir := t.TempDir()
	missingDataDir := filepath.Join(t.TempDir(), "missing-data-dir")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	writeMinimalConfig(t, configPath, missingDataDir, servicesDir)

	prev := opts
	opts = cliOptions{configPath: configPath}
	t.Cleanup(func() { opts = prev })

	cmd := newAuthCommand()
	cmd.SetArgs([]string{"status"})
	_ = cmd.Execute()

	if _, err := os.Stat(missingDataDir); !os.IsNotExist(err) {
		t.Fatalf("auth status must not create data_dir, stat err=%v", err)
	}
}

func TestAuthListDoesNotMutateExistingConnectorDB(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "kimbap.db")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	if _, err := db.ExecContext(contextBackground(), "CREATE TABLE probe (id INTEGER PRIMARY KEY)"); err != nil {
		_ = db.Close()
		t.Fatalf("create probe table: %v", err)
	}
	_ = db.Close()

	cfgRaw := "mode: embedded\n" +
		"data_dir: " + dataDir + "\n" +
		"database:\n" +
		"  driver: sqlite\n" +
		"  dsn: " + dbPath + "\n" +
		"services:\n" +
		"  dir: " + filepath.Join(dataDir, "services") + "\n"
	if err := os.WriteFile(configPath, []byte(cfgRaw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	statBefore, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db before: %v", err)
	}

	prev := opts
	opts = cliOptions{configPath: configPath}
	t.Cleanup(func() { opts = prev })

	cmd := newAuthCommand()
	cmd.SetArgs([]string{"list"})
	_ = cmd.Execute()

	statAfter, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db after: %v", err)
	}

	if statAfter.ModTime().After(statBefore.ModTime()) {
		t.Fatal("auth list must not modify the connector DB")
	}

	dbCheck, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer dbCheck.Close()
	rows, err := dbCheck.QueryContext(contextBackground(), "SELECT name FROM sqlite_master WHERE type='table' AND name='connector_states'")
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("auth list must not create connector_states table in read-only probe")
	}
}

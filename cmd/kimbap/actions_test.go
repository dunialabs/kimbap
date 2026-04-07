package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/services"
)

func TestBriefOutputFormat(t *testing.T) {
	defs := []actions.ActionDefinition{
		{Name: "github.issues.create", Description: "Create an issue", Risk: actions.RiskWrite},
		{Name: "brave-search.web-search", Description: "Search the web", Risk: actions.RiskRead},
	}

	for _, def := range defs {
		brief := map[string]string{
			"name":        def.Name,
			"description": def.Description,
			"risk":        string(def.Risk),
		}
		if brief["name"] == "" {
			t.Fatal("brief output missing name")
		}
		if brief["description"] == "" {
			t.Fatal("brief output missing description")
		}
		if brief["risk"] == "" {
			t.Fatal("brief output missing risk")
		}
	}
}

func TestActionsListShowsShortcutColumn(t *testing.T) {
	servicesDir := t.TempDir()
	dataDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	manifest := &services.ServiceManifest{
		Name:    "geo",
		Version: "1.0.0",
		Adapter: "http",
		BaseURL: "https://api.example.com",
		Auth:    services.ServiceAuth{Type: "none"},
		Actions: map[string]services.ServiceAction{
			"search": {
				Method:      "GET",
				Path:        "/search",
				Description: "Search for a location",
				Risk:        services.RiskSpec{Level: "low"},
				Response:    services.ResponseSpec{Type: "object"},
			},
			"forecast": {
				Method:      "GET",
				Path:        "/forecast",
				Description: "Get weather forecast",
				Risk:        services.RiskSpec{Level: "low"},
				Response:    services.ResponseSpec{Type: "object"},
			},
		},
	}
	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install service: %v", err)
	}

	cfgRaw := "mode: dev\n" +
		"data_dir: " + dataDir + "\n" +
		"services:\n" +
		"  dir: " + servicesDir + "\n" +
		"command_aliases:\n" +
		"  geosearch: geo.search\n"
	if err := os.WriteFile(configPath, []byte(cfgRaw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	prev := opts
	opts = cliOptions{configPath: configPath, noSplash: true}
	t.Cleanup(func() { opts = prev })

	cmd := newActionsCommand()
	cmd.SetArgs([]string{"list"})
	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("actions list failed: %v", err)
	}

	if !strings.Contains(output, "SHORTCUT") {
		t.Fatalf("expected SHORTCUT column header, got:\n%s", output)
	}
	if !strings.Contains(output, "geosearch") {
		t.Fatalf("expected shortcut 'geosearch' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "geo.forecast") {
		t.Fatalf("expected action 'geo.forecast' in output, got:\n%s", output)
	}
}

func TestActionsListNoShortcutColumnWhenNoneConfigured(t *testing.T) {
	servicesDir := t.TempDir()
	dataDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	manifest := &services.ServiceManifest{
		Name:    "geo",
		Version: "1.0.0",
		Adapter: "http",
		BaseURL: "https://api.example.com",
		Auth:    services.ServiceAuth{Type: "none"},
		Actions: map[string]services.ServiceAction{
			"search": {
				Method:      "GET",
				Path:        "/search",
				Description: "Search for a location",
				Risk:        services.RiskSpec{Level: "low"},
				Response:    services.ResponseSpec{Type: "object"},
			},
		},
	}
	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install service: %v", err)
	}

	cfgRaw := "mode: dev\n" +
		"data_dir: " + dataDir + "\n" +
		"services:\n" +
		"  dir: " + servicesDir + "\n"
	if err := os.WriteFile(configPath, []byte(cfgRaw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	prev := opts
	opts = cliOptions{configPath: configPath, noSplash: true}
	t.Cleanup(func() { opts = prev })

	cmd := newActionsCommand()
	cmd.SetArgs([]string{"list"})
	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("actions list failed: %v", err)
	}

	if strings.Contains(output, "SHORTCUT") {
		t.Fatalf("expected no SHORTCUT column when none configured, got:\n%s", output)
	}
}

func TestActionsDescribeDoesNotMaterializeDataDirForReadOnlyCredentialProbe(t *testing.T) {
	servicesDir := t.TempDir()
	missingDataDir := filepath.Join(t.TempDir(), "missing-data-dir")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	manifest := &services.ServiceManifest{
		Name:    "demo",
		Version: "1.0.0",
		Adapter: "http",
		BaseURL: "https://example.com",
		Auth: services.ServiceAuth{
			Type:          string(actions.AuthTypeBearer),
			CredentialRef: "demo.token",
		},
		Actions: map[string]services.ServiceAction{
			"noop": {
				Method:      "GET",
				Path:        "/noop",
				Description: "noop",
				Risk:        services.RiskSpec{Level: "low"},
				Response:    services.ResponseSpec{Type: "object"},
			},
		},
	}
	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install service: %v", err)
	}

	cfgRaw := "mode: dev\n" +
		"data_dir: " + missingDataDir + "\n" +
		"vault:\n" +
		"  path: " + filepath.Join(missingDataDir, "vault.db") + "\n" +
		"services:\n" +
		"  dir: " + servicesDir + "\n"
	if err := os.WriteFile(configPath, []byte(cfgRaw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	prev := opts
	opts = cliOptions{configPath: configPath}
	t.Cleanup(func() {
		opts = prev
	})

	cmd := newActionsCommand()
	cmd.SetArgs([]string{"describe", "demo.noop"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("actions describe failed: %v", err)
	}

	if _, err := os.Stat(missingDataDir); !os.IsNotExist(err) {
		t.Fatalf("expected read-only describe not to create data_dir, stat err=%v", err)
	}
}

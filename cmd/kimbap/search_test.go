package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/services"
)

func writeSearchTestConfig(t *testing.T, servicesDir, dataDir string, commandAliases map[string]string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	raw := "mode: dev\n" +
		"data_dir: " + dataDir + "\n" +
		"services:\n" +
		"  dir: " + servicesDir + "\n"
	if len(commandAliases) > 0 {
		raw += "command_aliases:\n"
		for alias, target := range commandAliases {
			raw += "  " + alias + ": " + target + "\n"
		}
	}
	if err := os.WriteFile(configPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func installSearchTestService(t *testing.T, servicesDir string) {
	t.Helper()
	manifest := &services.ServiceManifest{
		Name:    "open-meteo",
		Version: "1.0.0",
		Adapter: "http",
		BaseURL: "https://api.open-meteo.com",
		Auth:    services.ServiceAuth{Type: "none"},
		Actions: map[string]services.ServiceAction{
			"forecast": {
				Method:      "GET",
				Path:        "/forecast",
				Description: "Get weather forecast",
				Risk:        services.RiskSpec{Level: "low"},
				Response:    services.ResponseSpec{Type: "object"},
			},
			"archive": {
				Method:      "GET",
				Path:        "/archive",
				Description: "Get historical weather data",
				Risk:        services.RiskSpec{Level: "low"},
				Response:    services.ResponseSpec{Type: "object"},
			},
		},
	}
	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install service: %v", err)
	}
}

func TestSearchShowsShortcutColumnWhenResultHasShortcut(t *testing.T) {
	servicesDir := t.TempDir()
	dataDir := t.TempDir()
	installSearchTestService(t, servicesDir)

	configPath := writeSearchTestConfig(t, servicesDir, dataDir, map[string]string{
		"weather": "open-meteo.forecast",
	})

	prev := opts
	opts = cliOptions{configPath: configPath, noSplash: true}
	t.Cleanup(func() { opts = prev })

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"weather"})
	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if !strings.Contains(output, "SHORTCUT") {
		t.Fatalf("expected SHORTCUT column header, got:\n%s", output)
	}
	if !strings.Contains(output, "weather") {
		t.Fatalf("expected shortcut 'weather' in output, got:\n%s", output)
	}
}

func TestSearchNoShortcutColumnWhenNoResultHasShortcut(t *testing.T) {
	servicesDir := t.TempDir()
	dataDir := t.TempDir()
	installSearchTestService(t, servicesDir)

	configPath := writeSearchTestConfig(t, servicesDir, dataDir, map[string]string{
		"unrelated": "open-meteo.archive",
	})

	prev := opts
	opts = cliOptions{configPath: configPath, noSplash: true}
	t.Cleanup(func() { opts = prev })

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"forecast"})
	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if strings.Contains(output, "SHORTCUT") {
		t.Fatalf("expected no SHORTCUT column when no visible result has a shortcut, got:\n%s", output)
	}
	if strings.Contains(output, "Run '<shortcut>") {
		t.Fatalf("expected no shortcut footer when no visible result has a shortcut, got:\n%s", output)
	}
}

func TestSearchNoShortcutColumnWhenNoAliasesConfigured(t *testing.T) {
	servicesDir := t.TempDir()
	dataDir := t.TempDir()
	installSearchTestService(t, servicesDir)

	configPath := writeSearchTestConfig(t, servicesDir, dataDir, nil)

	prev := opts
	opts = cliOptions{configPath: configPath, noSplash: true}
	t.Cleanup(func() { opts = prev })

	cmd := newSearchCommand()
	cmd.SetArgs([]string{"weather"})
	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if strings.Contains(output, "SHORTCUT") {
		t.Fatalf("expected no SHORTCUT column when no aliases configured, got:\n%s", output)
	}
}

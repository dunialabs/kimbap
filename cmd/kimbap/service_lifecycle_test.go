package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/services/catalog"
)

func writeServiceCLIConfig(t *testing.T, dataDir, servicesDir string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	raw := "mode: embedded\n" +
		"data_dir: " + dataDir + "\n" +
		"services:\n" +
		"  dir: " + servicesDir + "\n"
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func writeServiceCLIConfigWithRegistryURL(t *testing.T, dataDir, servicesDir, registryURL string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	raw := "mode: embedded\n" +
		"data_dir: " + dataDir + "\n" +
		"services:\n" +
		"  dir: " + servicesDir + "\n"
	if strings.TrimSpace(registryURL) != "" {
		raw += "  registry_url: " + strings.TrimSpace(registryURL) + "\n"
	}
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func withServiceCLIOpts(t *testing.T, configPath string, fn func()) {
	t.Helper()
	prev := opts
	opts = cliOptions{configPath: configPath, format: "json", noSplash: true}
	t.Cleanup(func() { opts = prev })
	fn()
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w

	runErr := fn()
	_ = w.Close()
	os.Stdout = old

	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}
	return string(out), runErr
}

func pickCatalogServiceName(t *testing.T) string {
	t.Helper()
	names, err := catalog.List()
	if err != nil {
		t.Fatalf("catalog.List() error: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no catalog services available")
	}
	return names[0]
}

func decodeJSONObject(t *testing.T, raw string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode JSON object output failed: %v\noutput=%s", err, raw)
	}
	return out
}

func decodeJSONArrayOfObjects(t *testing.T, raw string) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode JSON array output failed: %v\noutput=%s", err, raw)
	}
	return out
}

func anySliceContainsString(items []any, needle string) bool {
	for _, item := range items {
		value, ok := item.(string)
		if !ok {
			continue
		}
		if value == needle {
			return true
		}
	}
	return false
}

func writeLocalManifest(t *testing.T, path, name, version string) {
	t.Helper()
	raw := "name: " + name + "\n" +
		"version: " + version + "\n" +
		"description: local test service\n" +
		"base_url: https://api.example.com\n" +
		"auth:\n" +
		"  type: header\n" +
		"  header_name: Authorization\n" +
		"  credential_ref: local.token\n" +
		"actions:\n" +
		"  ping:\n" +
		"    method: GET\n" +
		"    path: /ping\n" +
		"    description: ping endpoint\n" +
		"    risk:\n" +
		"      level: low\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write manifest %q: %v", path, err)
	}
}

func writeLocalManifestWithActionAlias(t *testing.T, path, name, version, actionAlias string) {
	t.Helper()
	raw := "name: " + name + "\n" +
		"version: " + version + "\n" +
		"description: local test service\n" +
		"base_url: https://api.example.com\n" +
		"auth:\n" +
		"  type: none\n" +
		"actions:\n" +
		"  search:\n" +
		"    method: GET\n" +
		"    path: /search\n" +
		"    description: search endpoint\n" +
		"    aliases:\n" +
		"      - " + actionAlias + "\n" +
		"    risk:\n" +
		"      level: low\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write manifest %q: %v", path, err)
	}
}

func TestServiceCLIInstallEnableDisableLifecycle(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)
	serviceName := pickCatalogServiceName(t)

	withServiceCLIOpts(t, configPath, func() {
		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{serviceName, "--no-activate"})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install --no-activate failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		installer := installerFromConfig(cfg)

		installed, err := installer.Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get() error: %v", err)
		}
		if installed.Enabled {
			t.Fatalf("expected %q to be disabled after --no-activate install", serviceName)
		}

		enableCmd := newServiceEnableCommand()
		enableCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, enableCmd.Execute); err != nil {
			t.Fatalf("service enable failed: %v", err)
		}
		enabledState, err := installer.Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get() after enable error: %v", err)
		}
		if !enabledState.Enabled {
			t.Fatalf("expected %q to be enabled", serviceName)
		}

		disableCmd := newServiceDisableCommand()
		disableCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, disableCmd.Execute); err != nil {
			t.Fatalf("service disable failed: %v", err)
		}
		disabledState, err := installer.Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get() after disable error: %v", err)
		}
		if disabledState.Enabled {
			t.Fatalf("expected %q to be disabled", serviceName)
		}
	})
}

func TestServiceCLIEnableBackfillsActionAliasesAfterNoActivateInstall(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "open-meteo-geocoding.yaml")
		const serviceName = "open-meteo-geocoding"
		writeLocalManifestWithActionAlias(t, manifestPath, serviceName, "1.0.0", "geosearch")

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
		})
		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		aliasFileSymlink = func(oldname, newname string) error { return nil }

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath, "--no-activate"})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install --no-activate failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		if _, exists := cfg.CommandAliases["geosearch"]; exists {
			t.Fatalf("did not expect geosearch alias before enabling service, got %+v", cfg.CommandAliases)
		}

		enableCmd := newServiceEnableCommand()
		enableCmd.SetArgs([]string{serviceName})
		enableOutput, err := captureStdout(t, enableCmd.Execute)
		if err != nil {
			t.Fatalf("service enable failed: %v", err)
		}

		payload := decodeJSONObject(t, enableOutput)
		created, ok := payload["action_aliases_created"].([]any)
		if !ok || len(created) == 0 {
			t.Fatalf("expected action_aliases_created in enable output payload, got %+v", payload)
		}

		cfg, err = loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() after enable error: %v", err)
		}
		if got := cfg.CommandAliases["geosearch"]; got != "open-meteo-geocoding.search" {
			t.Fatalf("expected command alias geosearch -> open-meteo-geocoding.search after enable, got %+v", cfg.CommandAliases)
		}
	})
}

func TestServiceCLIInstallAutoAliasConfiguredAndPersisted(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "open-meteo-geocoding.yaml")
		const serviceName = "open-meteo-geocoding"
		writeLocalManifest(t, manifestPath, serviceName, "1.0.0")

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		autoAlias := services.PreferredServiceAlias(serviceName, nil)
		if autoAlias == "" {
			t.Fatal("expected non-empty auto alias")
		}
		if target, ok := cfg.Aliases[autoAlias]; !ok || target != serviceName {
			t.Fatalf("expected alias %q -> %q in config, got aliases=%+v", autoAlias, serviceName, cfg.Aliases)
		}

		installer := installerFromConfig(cfg)
		installed, err := installer.Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get() error: %v", err)
		}
		if len(installed.Manifest.Aliases) != 0 {
			t.Fatalf("expected installed manifest aliases to remain unchanged when alias is config-level only, got %+v", installed.Manifest.Aliases)
		}
	})
}

func TestServiceCLIInstallAppliesManifestActionAliases(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "open-meteo-geocoding.yaml")
		const serviceName = "open-meteo-geocoding"
		writeLocalManifestWithActionAlias(t, manifestPath, serviceName, "1.0.0", "geosearch")

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
		})
		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		aliasFileSymlink = func(oldname, newname string) error { return nil }

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath})
		output, err := captureStdout(t, installCmd.Execute)
		if err != nil {
			t.Fatalf("service install failed: %v", err)
		}
		payload := decodeJSONObject(t, output)
		created, ok := payload["action_aliases_created"].([]any)
		if !ok || len(created) == 0 {
			t.Fatalf("expected action_aliases_created in output payload, got %+v", payload)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		if got := cfg.CommandAliases["geosearch"]; got != "open-meteo-geocoding.search" {
			t.Fatalf("expected command alias geosearch -> open-meteo-geocoding.search, got %+v", cfg.CommandAliases)
		}
	})
}

func TestServiceCLIListAvailableReflectsInstalledAndEnabledStatus(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)
	serviceName := pickCatalogServiceName(t)

	withServiceCLIOpts(t, configPath, func() {
		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{serviceName, "--no-activate"})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install --no-activate failed: %v", err)
		}

		listCmd := newServiceListCommand()
		listCmd.SetArgs([]string{"--available"})
		output, err := captureStdout(t, listCmd.Execute)
		if err != nil {
			t.Fatalf("service list --available failed: %v", err)
		}

		var rows []map[string]any
		if unmarshalErr := json.Unmarshal([]byte(output), &rows); unmarshalErr != nil {
			t.Fatalf("unmarshal list output: %v\noutput=%s", unmarshalErr, output)
		}

		var matched map[string]any
		for _, row := range rows {
			if row["name"] == serviceName {
				matched = row
				break
			}
		}
		if matched == nil {
			t.Fatalf("service %q not present in --available output", serviceName)
		}
		if installed, _ := matched["installed"].(bool); !installed {
			t.Fatalf("expected installed=true for %q row: %+v", serviceName, matched)
		}
		if enabled, _ := matched["enabled"].(bool); enabled {
			t.Fatalf("expected enabled=false for no-activate install row: %+v", matched)
		}
		if status, _ := matched["status"].(string); status != "disabled" {
			t.Fatalf("expected status=disabled, got %q (row=%+v)", status, matched)
		}
	})
}

func TestServiceCLIListUsesEmptyShortcutArrayWhenUnset(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "local-empty-shortcuts.yaml")
		const serviceName = "local-empty-shortcuts"
		writeLocalManifest(t, manifestPath, serviceName, "1.0.0")

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath, "--no-shortcuts"})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install --no-shortcuts failed: %v", err)
		}

		listCmd := newServiceListCommand()
		listCmd.SetArgs(nil)
		output, err := captureStdout(t, listCmd.Execute)
		if err != nil {
			t.Fatalf("service list failed: %v", err)
		}

		rows := decodeJSONArrayOfObjects(t, output)
		var matched map[string]any
		for _, row := range rows {
			if row["name"] == serviceName {
				matched = row
				break
			}
		}
		if matched == nil {
			t.Fatalf("service %q not present in list output", serviceName)
		}

		shortcuts, ok := matched["shortcuts"].([]any)
		if !ok {
			t.Fatalf("expected shortcuts array in list output row, got %+v", matched)
		}
		if len(shortcuts) != 0 {
			t.Fatalf("expected empty shortcuts array when unset, got %+v", shortcuts)
		}
	})
}

func TestServiceCLIListIncludesShortcutsForInstalledService(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "open-meteo-geocoding.yaml")
		const serviceName = "open-meteo-geocoding"
		writeLocalManifestWithActionAlias(t, manifestPath, serviceName, "1.0.0", "geosearch")

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
		})
		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		aliasFileSymlink = func(oldname, newname string) error { return nil }

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install failed: %v", err)
		}

		listCmd := newServiceListCommand()
		listCmd.SetArgs(nil)
		output, err := captureStdout(t, listCmd.Execute)
		if err != nil {
			t.Fatalf("service list failed: %v", err)
		}

		rows := decodeJSONArrayOfObjects(t, output)
		var matched map[string]any
		for _, row := range rows {
			if row["name"] == serviceName {
				matched = row
				break
			}
		}
		if matched == nil {
			t.Fatalf("service %q not present in list output", serviceName)
		}

		shortcuts, ok := matched["shortcuts"].([]any)
		if !ok {
			t.Fatalf("expected shortcuts array in list output row, got %+v", matched)
		}
		if !anySliceContainsString(shortcuts, "geosearch") {
			t.Fatalf("expected shortcuts to include geosearch, got %+v", shortcuts)
		}
	})
}

func TestServiceCLIListAvailableIncludesShortcutsForInstalledRow(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
		})
		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		aliasFileSymlink = func(oldname, newname string) error { return nil }

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{"open-meteo-geocoding"})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install failed: %v", err)
		}

		listCmd := newServiceListCommand()
		listCmd.SetArgs([]string{"--available"})
		output, err := captureStdout(t, listCmd.Execute)
		if err != nil {
			t.Fatalf("service list --available failed: %v", err)
		}

		rows := decodeJSONArrayOfObjects(t, output)
		var matched map[string]any
		for _, row := range rows {
			if row["name"] == "open-meteo-geocoding" {
				matched = row
				break
			}
		}
		if matched == nil {
			t.Fatal("open-meteo-geocoding not present in --available output")
		}

		shortcuts, ok := matched["shortcuts"].([]any)
		if !ok {
			t.Fatalf("expected shortcuts array in --available row, got %+v", matched)
		}
		if !anySliceContainsString(shortcuts, "geosearch") {
			t.Fatalf("expected --available shortcuts to include geosearch, got %+v", shortcuts)
		}
	})
}

func TestServiceCLIUpdateFromLocalSourceAndNoOpWhenUpToDate(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "local-update.yaml")
		const serviceName = "local-update"
		writeLocalManifest(t, manifestPath, serviceName, "1.0.0")

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install local manifest failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		installer := installerFromConfig(cfg)

		installed, err := installer.Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get() error: %v", err)
		}
		if installed.Manifest.Version != "1.0.0" {
			t.Fatalf("installed version = %q, want 1.0.0", installed.Manifest.Version)
		}

		updateCmd := newServiceUpdateCommand()
		updateCmd.SetArgs([]string{serviceName})
		noChangeOutput, err := captureStdout(t, updateCmd.Execute)
		if err != nil {
			t.Fatalf("service update no-change failed: %v", err)
		}
		noChangePayload := decodeJSONObject(t, noChangeOutput)
		if updated, _ := noChangePayload["updated"].(bool); updated {
			t.Fatalf("expected updated=false output, got payload: %+v", noChangePayload)
		}

		writeLocalManifest(t, manifestPath, serviceName, "1.0.1")
		updateOutput, err := captureStdout(t, updateCmd.Execute)
		if err != nil {
			t.Fatalf("service update from local source failed: %v", err)
		}
		updatePayload := decodeJSONObject(t, updateOutput)
		if updated, _ := updatePayload["updated"].(bool); !updated {
			t.Fatalf("expected updated=true output, got payload: %+v", updatePayload)
		}

		updated, err := installer.Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get() after update error: %v", err)
		}
		if updated.Manifest.Version != "1.0.1" {
			t.Fatalf("updated version = %q, want 1.0.1", updated.Manifest.Version)
		}
		if !strings.HasPrefix(updated.Source, "local:") {
			t.Fatalf("updated source = %q, want local:*", updated.Source)
		}
	})
}

func TestServiceCLIRemoveDeletesInstalledManifest(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "local-remove.yaml")
		const serviceName = "local-remove"
		writeLocalManifest(t, manifestPath, serviceName, "1.0.0")

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install failed: %v", err)
		}

		removeCmd := newServiceRemoveCommand()
		removeCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, removeCmd.Execute); err != nil {
			t.Fatalf("service remove failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		_, err = installerFromConfig(cfg).Get(serviceName)
		if err == nil {
			t.Fatal("expected installer.Get() to fail after remove")
		}

		if _, err := os.Stat(filepath.Join(servicesDir, serviceName+".yaml")); !os.IsNotExist(err) {
			t.Fatalf("expected manifest file to be removed, stat err=%v", err)
		}
	})
}

func TestServiceCLIDisableCleansUpServiceAndCommandAliases(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "open-meteo-geocoding.yaml")
		const serviceName = "open-meteo-geocoding"
		writeLocalManifestWithActionAlias(t, manifestPath, serviceName, "1.0.0", "geosearch")

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		aliasPath := filepath.Join(execDir, "geosearch")
		symlinkExists := false
		executableRemoved := false

		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		origReadlink := aliasFileReadlink
		origRemove := aliasFileRemove
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
			aliasFileReadlink = origReadlink
			aliasFileRemove = origRemove
		})

		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) {
			if path == aliasPath && symlinkExists {
				return symlinkFileInfo{}, nil
			}
			return nil, os.ErrNotExist
		}
		aliasFileSymlink = func(oldname, newname string) error {
			if oldname == execPath && newname == aliasPath {
				symlinkExists = true
			}
			return nil
		}
		aliasFileReadlink = func(path string) (string, error) {
			if path == aliasPath && symlinkExists {
				return execPath, nil
			}
			return "", os.ErrNotExist
		}
		aliasFileRemove = func(path string) error {
			if path == aliasPath && symlinkExists {
				symlinkExists = false
				executableRemoved = true
				return nil
			}
			return nil
		}

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install failed: %v", err)
		}

		disableCmd := newServiceDisableCommand()
		disableCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, disableCmd.Execute); err != nil {
			t.Fatalf("service disable failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() after disable error: %v", err)
		}
		if _, exists := cfg.CommandAliases["geosearch"]; exists {
			t.Fatalf("expected command alias geosearch removed on disable, got %+v", cfg.CommandAliases)
		}
		for alias, target := range cfg.Aliases {
			if strings.EqualFold(strings.TrimSpace(target), serviceName) {
				t.Fatalf("expected service aliases for %q removed on disable, still have %q -> %q", serviceName, alias, target)
			}
		}
		if !executableRemoved {
			t.Fatal("expected executable shortcut cleanup during disable")
		}
	})
}

func TestServiceCLIRemoveCleansUpServiceAndCommandAliases(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "open-meteo-geocoding-remove-cleanup.yaml")
		const serviceName = "open-meteo-geocoding"
		writeLocalManifestWithActionAlias(t, manifestPath, serviceName, "1.0.0", "geosearch")

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		aliasPath := filepath.Join(execDir, "geosearch")
		symlinkExists := false
		executableRemoved := false

		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		origReadlink := aliasFileReadlink
		origRemove := aliasFileRemove
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
			aliasFileReadlink = origReadlink
			aliasFileRemove = origRemove
		})

		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) {
			if path == aliasPath && symlinkExists {
				return symlinkFileInfo{}, nil
			}
			return nil, os.ErrNotExist
		}
		aliasFileSymlink = func(oldname, newname string) error {
			if oldname == execPath && newname == aliasPath {
				symlinkExists = true
			}
			return nil
		}
		aliasFileReadlink = func(path string) (string, error) {
			if path == aliasPath && symlinkExists {
				return execPath, nil
			}
			return "", os.ErrNotExist
		}
		aliasFileRemove = func(path string) error {
			if path == aliasPath && symlinkExists {
				symlinkExists = false
				executableRemoved = true
				return nil
			}
			return nil
		}

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install failed: %v", err)
		}

		removeCmd := newServiceRemoveCommand()
		removeCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, removeCmd.Execute); err != nil {
			t.Fatalf("service remove failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() after remove error: %v", err)
		}
		if _, exists := cfg.CommandAliases["geosearch"]; exists {
			t.Fatalf("expected command alias geosearch removed on remove, got %+v", cfg.CommandAliases)
		}
		for alias, target := range cfg.Aliases {
			if strings.EqualFold(strings.TrimSpace(target), serviceName) {
				t.Fatalf("expected service aliases for %q removed on remove, still have %q -> %q", serviceName, alias, target)
			}
		}
		if !executableRemoved {
			t.Fatal("expected executable shortcut cleanup during remove")
		}
	})
}

func TestCleanupAliasesForServiceRollsBackEarlierRemovalsOnFailure(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		resolvedConfigPath, resolveErr := resolveConfigPath()
		if resolveErr != nil {
			t.Fatalf("resolveConfigPath() error: %v", resolveErr)
		}
		if err := upsertConfigAlias(resolvedConfigPath, "geo", "svc"); err != nil {
			t.Fatalf("set service alias: %v", err)
		}
		if err := upsertConfigCommandAlias(resolvedConfigPath, "aone", "svc.search"); err != nil {
			t.Fatalf("set command alias aone: %v", err)
		}
		if err := upsertConfigCommandAlias(resolvedConfigPath, "ztwo", "svc.search"); err != nil {
			t.Fatalf("set command alias ztwo: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		symlinkState := map[string]bool{
			filepath.Join(execDir, "aone"): true,
			filepath.Join(execDir, "ztwo"): true,
		}
		restoredAoneExecutable := false

		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		origReadlink := aliasFileReadlink
		origRemove := aliasFileRemove
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
			aliasFileReadlink = origReadlink
			aliasFileRemove = origRemove
		})

		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) {
			if symlinkState[path] {
				return symlinkFileInfo{}, nil
			}
			return nil, os.ErrNotExist
		}
		aliasFileReadlink = func(path string) (string, error) {
			if symlinkState[path] {
				return execPath, nil
			}
			return "", os.ErrNotExist
		}
		aliasFileRemove = func(path string) error {
			if !symlinkState[path] {
				return os.ErrNotExist
			}
			if filepath.Base(path) == "ztwo" {
				return os.ErrPermission
			}
			symlinkState[path] = false
			return nil
		}
		aliasFileSymlink = func(oldname, newname string) error {
			if oldname != execPath {
				return os.ErrInvalid
			}
			symlinkState[newname] = true
			if filepath.Base(newname) == "aone" {
				restoredAoneExecutable = true
			}
			return nil
		}

		_, _, _, cleanupErr := cleanupAliasesForService(resolvedConfigPath, "svc", cfg.Aliases, cfg.CommandAliases)
		if cleanupErr == nil {
			t.Fatal("expected cleanupAliasesForService to fail")
		}

		reloaded, loadErr := loadAppConfig()
		if loadErr != nil {
			t.Fatalf("reload config: %v", loadErr)
		}
		if got := reloaded.Aliases["geo"]; got != "svc" {
			t.Fatalf("expected service alias geo restored, got aliases=%+v", reloaded.Aliases)
		}
		if got := reloaded.CommandAliases["aone"]; got != "svc.search" {
			t.Fatalf("expected command alias aone restored, got %+v", reloaded.CommandAliases)
		}
		if got := reloaded.CommandAliases["ztwo"]; got != "svc.search" {
			t.Fatalf("expected command alias ztwo restored, got %+v", reloaded.CommandAliases)
		}
		if !restoredAoneExecutable {
			t.Fatal("expected aone executable alias to be restored during rollback")
		}
	})
}

func TestServiceCLIUpdateRegistryNoOp(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)
	serviceName := pickCatalogServiceName(t)

	withServiceCLIOpts(t, configPath, func() {
		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install catalog failed: %v", err)
		}

		updateCmd := newServiceUpdateCommand()
		updateCmd.SetArgs([]string{serviceName})
		output, err := captureStdout(t, updateCmd.Execute)
		if err != nil {
			t.Fatalf("service update registry no-op failed: %v", err)
		}
		payload := decodeJSONObject(t, output)
		if updated, _ := payload["updated"].(bool); updated {
			t.Fatalf("expected registry no-op update to return updated=false, got payload: %+v", payload)
		}
	})
}

func TestServiceCLIInstallFromConfiguredRegistryPersistsRegistrySource(t *testing.T) {
	const serviceName = "remote-registry-test"

	registryServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+serviceName+".yaml" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, "name: "+serviceName+"\nversion: 1.0.0\ndescription: remote registry test service\nbase_url: https://api.example.com\nauth:\n  type: none\nactions:\n  ping:\n    method: GET\n    path: /ping\n    description: ping endpoint\n    risk:\n      level: low\n")
	}))
	defer registryServer.Close()

	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfigWithRegistryURL(t, dataDir, servicesDir, registryServer.URL)

	prevTransport := http.DefaultTransport
	http.DefaultTransport = registryServer.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = prevTransport })

	withServiceCLIOpts(t, configPath, func() {
		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install from configured registry failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}

		installed, err := installerFromConfig(cfg).Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get(%q) error: %v", serviceName, err)
		}
		if installed.Source != "registry:"+serviceName {
			t.Fatalf("installed source = %q, want %q", installed.Source, "registry:"+serviceName)
		}
	})
}

func TestServiceCLIUpdateFromConfiguredRegistryIgnoresLocalNameCollision(t *testing.T) {
	const serviceName = "remote-registry-update-collision-test"

	currentVersion := "1.0.0"
	var versionMu sync.RWMutex

	registryServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+serviceName+".yaml" {
			http.NotFound(w, r)
			return
		}

		versionMu.RLock()
		version := currentVersion
		versionMu.RUnlock()

		_, _ = io.WriteString(w,
			"name: "+serviceName+"\n"+
				"version: "+version+"\n"+
				"description: remote registry update collision test service\n"+
				"base_url: https://api.example.com\n"+
				"auth:\n"+
				"  type: none\n"+
				"actions:\n"+
				"  ping:\n"+
				"    method: GET\n"+
				"    path: /ping\n"+
				"    description: ping endpoint\n"+
				"    risk:\n"+
				"      level: low\n")
	}))
	defer registryServer.Close()

	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfigWithRegistryURL(t, dataDir, servicesDir, registryServer.URL)

	prevTransport := http.DefaultTransport
	http.DefaultTransport = registryServer.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = prevTransport })

	withServiceCLIOpts(t, configPath, func() {
		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install from configured registry failed: %v", err)
		}

		collisionDir := t.TempDir()
		collisionPath := filepath.Join(collisionDir, serviceName)
		writeLocalManifest(t, collisionPath, serviceName, "9.9.9")

		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("os.Getwd() error: %v", err)
		}
		if err := os.Chdir(collisionDir); err != nil {
			t.Fatalf("os.Chdir(%q) error: %v", collisionDir, err)
		}
		t.Cleanup(func() {
			_ = os.Chdir(oldWD)
		})

		versionMu.Lock()
		currentVersion = "1.1.0"
		versionMu.Unlock()

		updateCmd := newServiceUpdateCommand()
		updateCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, updateCmd.Execute); err != nil {
			t.Fatalf("service update from configured registry failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		installed, err := installerFromConfig(cfg).Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get(%q) error: %v", serviceName, err)
		}
		if installed.Manifest.Version != "1.1.0" {
			t.Fatalf("installed version = %q, want 1.1.0", installed.Manifest.Version)
		}
		if installed.Source != "registry:"+serviceName {
			t.Fatalf("installed source = %q, want %q", installed.Source, "registry:"+serviceName)
		}
	})
}

func TestServiceCLIOutdatedReportsVersionDriftFromConfiguredRegistry(t *testing.T) {
	const serviceName = "remote-registry-outdated-test"

	currentVersion := "1.0.0"
	var versionMu sync.RWMutex

	registryServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+serviceName+".yaml" {
			http.NotFound(w, r)
			return
		}

		versionMu.RLock()
		version := currentVersion
		versionMu.RUnlock()

		_, _ = io.WriteString(w,
			"name: "+serviceName+"\n"+
				"version: "+version+"\n"+
				"description: remote registry outdated test service\n"+
				"base_url: https://api.example.com\n"+
				"auth:\n"+
				"  type: none\n"+
				"actions:\n"+
				"  ping:\n"+
				"    method: GET\n"+
				"    path: /ping\n"+
				"    description: ping endpoint\n"+
				"    risk:\n"+
				"      level: low\n")
	}))
	defer registryServer.Close()

	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfigWithRegistryURL(t, dataDir, servicesDir, registryServer.URL)

	prevTransport := http.DefaultTransport
	http.DefaultTransport = registryServer.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = prevTransport })

	withServiceCLIOpts(t, configPath, func() {
		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install from configured registry failed: %v", err)
		}

		collisionDir := t.TempDir()
		collisionPath := filepath.Join(collisionDir, serviceName)
		writeLocalManifest(t, collisionPath, serviceName, "9.9.9")

		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("os.Getwd() error: %v", err)
		}
		if err := os.Chdir(collisionDir); err != nil {
			t.Fatalf("os.Chdir(%q) error: %v", collisionDir, err)
		}
		t.Cleanup(func() {
			_ = os.Chdir(oldWD)
		})

		versionMu.Lock()
		currentVersion = "1.1.0"
		versionMu.Unlock()

		outdatedCmd := newServiceOutdatedCommand()
		output, err := captureStdout(t, outdatedCmd.Execute)
		if err != nil {
			t.Fatalf("service outdated failed: %v", err)
		}

		var rows []map[string]any
		if unmarshalErr := json.Unmarshal([]byte(output), &rows); unmarshalErr != nil {
			t.Fatalf("unmarshal outdated output: %v\noutput=%s", unmarshalErr, output)
		}

		var matched map[string]any
		for _, row := range rows {
			if row["name"] == serviceName {
				matched = row
				break
			}
		}
		if matched == nil {
			t.Fatalf("service %q not present in outdated output: %+v", serviceName, rows)
		}
		if gotInstalled, _ := matched["installed_version"].(string); gotInstalled != "1.0.0" {
			t.Fatalf("installed_version = %q, want 1.0.0", gotInstalled)
		}
		if gotLatest, _ := matched["latest_version"].(string); gotLatest != "1.1.0" {
			t.Fatalf("latest_version = %q, want 1.1.0", gotLatest)
		}
		if gotSource, _ := matched["source"].(string); gotSource != "registry:"+serviceName {
			t.Fatalf("source = %q, want %q", gotSource, "registry:"+serviceName)
		}
	})
}

func TestServiceCLIOutdatedSkipsMissingCatalogRegistrySourcesWithoutWarning(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	manifestPath := filepath.Join(t.TempDir(), "missing-catalog.yaml")
	const serviceName = "missing-catalog-registry-source"
	writeLocalManifest(t, manifestPath, serviceName, "1.0.0")

	manifest, err := services.ParseManifestFile(manifestPath)
	if err != nil {
		t.Fatalf("ParseManifestFile(%q) error: %v", manifestPath, err)
	}

	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.InstallWithForceAndActivation(manifest, "registry:definitely-not-a-real-service", false, true); err != nil {
		t.Fatalf("InstallWithForceAndActivation() error: %v", err)
	}

	prev := opts
	opts = cliOptions{configPath: configPath, format: "text", noSplash: true}
	t.Cleanup(func() { opts = prev })

	outdatedCmd := newServiceOutdatedCommand()
	stderr, err := captureStderr(t, outdatedCmd.Execute)
	if err != nil {
		t.Fatalf("service outdated failed: %v", err)
	}
	if strings.Contains(stderr, "failed to resolve") {
		t.Fatalf("expected missing catalog entries to be skipped without warnings, got stderr=%q", stderr)
	}
}

func TestServiceCLIUpdatePreservesActivationState(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "local-state.yaml")
		const serviceName = "local-state"
		writeLocalManifest(t, manifestPath, serviceName, "1.0.0")

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath, "--no-activate"})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install --no-activate failed: %v", err)
		}

		writeLocalManifest(t, manifestPath, serviceName, "1.0.2")
		updateCmd := newServiceUpdateCommand()
		updateCmd.SetArgs([]string{serviceName})
		if _, err := captureStdout(t, updateCmd.Execute); err != nil {
			t.Fatalf("service update failed: %v", err)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		installed, err := installerFromConfig(cfg).Get(serviceName)
		if err != nil {
			t.Fatalf("installer.Get() error: %v", err)
		}
		if installed.Enabled {
			t.Fatalf("expected disabled state to be preserved after update for %q", serviceName)
		}
	})
}

func TestServiceCLIEnableMissingServiceFails(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		enableCmd := newServiceEnableCommand()
		enableCmd.SetArgs([]string{"missing-service"})
		_, err := captureStdout(t, enableCmd.Execute)
		if err == nil {
			t.Fatal("expected enable on missing service to fail")
		}
		if !strings.Contains(err.Error(), "is not installed") {
			t.Fatalf("unexpected enable error: %v", err)
		}
	})
}

func TestServiceCLIInstallRejectsInsecureHTTPURL(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{"http://example.com/manifest.yaml"})
		_, err := captureStdout(t, installCmd.Execute)
		if err == nil {
			t.Fatal("expected insecure http source to be rejected")
		}
		if !strings.Contains(err.Error(), "insecure URL") {
			t.Fatalf("unexpected install error: %v", err)
		}
	})
}

func TestServiceInstallPrintsSyncHint(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	agentSkillPath := filepath.Join(homeDir, ".claude", "skills", "kimbap", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(agentSkillPath), 0o755); err != nil {
		t.Fatalf("create agent skills dir: %v", err)
	}
	if err := os.WriteFile(agentSkillPath, []byte("# kimbap\n"), 0o644); err != nil {
		t.Fatalf("write agent skill file: %v", err)
	}

	manifestPath := filepath.Join(t.TempDir(), "local-hint.yaml")
	writeLocalManifest(t, manifestPath, "local-hint", "1.0.0")

	prev := opts
	opts = cliOptions{configPath: configPath, format: "text", noSplash: true}
	t.Cleanup(func() { opts = prev })

	installCmd := newServiceInstallCommand()
	installCmd.SetArgs([]string{manifestPath})
	stderr, err := captureStderr(t, installCmd.Execute)
	if err != nil {
		t.Fatalf("service install failed: %v", err)
	}
	if !strings.Contains(stderr, "Hint: Run 'kimbap agents sync' to update your AI agents with this change.") {
		t.Fatalf("expected sync hint in stderr, got %q", stderr)
	}
}

func TestServiceCLIInstalledFileParsesAsValidManifest(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		manifestPath := filepath.Join(t.TempDir(), "local-verify.yaml")
		const serviceName = "local-verify"
		writeLocalManifest(t, manifestPath, serviceName, "1.0.0")

		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install failed: %v", err)
		}

		installedPath := filepath.Join(servicesDir, serviceName+".yaml")
		manifest, err := services.ParseManifestFile(installedPath)
		if err != nil {
			t.Fatalf("ParseManifestFile(installed) error: %v", err)
		}
		if manifest.Name != serviceName {
			t.Fatalf("installed manifest name = %q, want %q", manifest.Name, serviceName)
		}
	})
}

func TestServiceGenerateRejectsInsecureHTTPURL(t *testing.T) {
	cmd := newServiceGenerateCommand()
	cmd.SetArgs([]string{"--openapi", "http://example.com/openapi.yaml"})
	_, err := captureStdout(t, cmd.Execute)
	if err == nil {
		t.Fatal("expected insecure OpenAPI URL to be rejected")
	}
	if !strings.Contains(err.Error(), "insecure URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

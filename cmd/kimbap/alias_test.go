package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/services"
)

type symlinkFileInfo struct{}

func (symlinkFileInfo) Name() string       { return "geosearch" }
func (symlinkFileInfo) Size() int64        { return 0 }
func (symlinkFileInfo) Mode() os.FileMode  { return os.ModeSymlink }
func (symlinkFileInfo) ModTime() time.Time { return time.Time{} }
func (symlinkFileInfo) IsDir() bool        { return false }
func (symlinkFileInfo) Sys() any           { return nil }

func TestAliasSetActionAliasCreatesExecutableAndConfigEntry(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		installer := installerFromConfig(cfg)
		manifest := &services.ServiceManifest{
			Name:    "open-meteo-geocoding",
			Version: "1.0.0",
			Adapter: "http",
			BaseURL: "https://example.com",
			Auth:    services.ServiceAuth{Type: "none"},
			Actions: map[string]services.ServiceAction{
				"search": {
					Method:      "GET",
					Path:        "/search",
					Description: "search",
					Risk:        services.RiskSpec{Level: "low"},
					Response:    services.ResponseSpec{Type: "object"},
				},
			},
		}
		if _, err := installer.Install(manifest, "local"); err != nil {
			t.Fatalf("install service: %v", err)
		}

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		var symlinkOld string
		var symlinkNew string

		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		origReadlink := aliasFileReadlink
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
			aliasFileReadlink = origReadlink
		})

		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }
		aliasFileSymlink = func(oldname, newname string) error {
			symlinkOld = oldname
			symlinkNew = newname
			return nil
		}
		aliasFileReadlink = os.Readlink

		setCmd := newAliasSetCommand()
		setCmd.SetArgs([]string{"geosearch", "open-meteo-geocoding.search"})
		output, err := captureStdout(t, setCmd.Execute)
		if err != nil {
			t.Fatalf("alias set action command failed: %v", err)
		}
		payload := decodeJSONObject(t, output)
		if payload["type"] != "action" {
			t.Fatalf("expected action alias type in output, got payload=%+v", payload)
		}
		if payload["alias"] != "geosearch" || payload["target"] != "open-meteo-geocoding.search" {
			t.Fatalf("unexpected alias payload: %+v", payload)
		}

		if symlinkOld != execPath {
			t.Fatalf("symlink source = %q, want %q", symlinkOld, execPath)
		}
		if symlinkNew != filepath.Join(execDir, "geosearch") {
			t.Fatalf("symlink destination = %q, want %q", symlinkNew, filepath.Join(execDir, "geosearch"))
		}

		updatedCfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("reload config: %v", err)
		}
		if got := updatedCfg.CommandAliases["geosearch"]; got != "open-meteo-geocoding.search" {
			t.Fatalf("command alias persisted as %q, want %q", got, "open-meteo-geocoding.search")
		}
	})
}

func TestAliasRemoveActionAliasRemovesExecutableAndConfigEntry(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	if err := os.WriteFile(configPath, []byte("mode: embedded\n"+
		"data_dir: "+dataDir+"\n"+
		"services:\n  dir: "+servicesDir+"\n"+
		"command_aliases:\n  geosearch: open-meteo-geocoding.search\n"), 0o644); err != nil {
		t.Fatalf("write config with command alias: %v", err)
	}

	withServiceCLIOpts(t, configPath, func() {
		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		aliasPath := filepath.Join(execDir, "geosearch")
		removedPath := ""

		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origReadlink := aliasFileReadlink
		origRemove := aliasFileRemove
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileReadlink = origReadlink
			aliasFileRemove = origRemove
		})

		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) {
			if path == aliasPath {
				return symlinkFileInfo{}, nil
			}
			return nil, os.ErrNotExist
		}
		aliasFileReadlink = func(path string) (string, error) {
			if path != aliasPath {
				return "", os.ErrNotExist
			}
			return execPath, nil
		}
		aliasFileRemove = func(path string) error {
			removedPath = path
			return nil
		}

		removeCmd := newAliasRemoveCommand()
		removeCmd.SetArgs([]string{"geosearch"})
		output, err := captureStdout(t, removeCmd.Execute)
		if err != nil {
			t.Fatalf("alias remove failed: %v", err)
		}
		payload := decodeJSONObject(t, output)
		if removed, _ := payload["command_removed"].(bool); !removed {
			t.Fatalf("expected command_removed=true, got payload=%+v", payload)
		}
		if removedPath != aliasPath {
			t.Fatalf("expected executable alias removal path %q, got %q", aliasPath, removedPath)
		}

		updatedCfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("reload config: %v", err)
		}
		if _, exists := updatedCfg.CommandAliases["geosearch"]; exists {
			t.Fatalf("expected command alias geosearch removed from config, got %+v", updatedCfg.CommandAliases)
		}
	})
}

func TestAliasRemoveActionAliasKeepsConfigWhenExecutableCleanupFails(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	if err := os.WriteFile(configPath, []byte("mode: embedded\n"+
		"data_dir: "+dataDir+"\n"+
		"services:\n  dir: "+servicesDir+"\n"+
		"command_aliases:\n  geosearch: open-meteo-geocoding.search\n"), 0o644); err != nil {
		t.Fatalf("write config with command alias: %v", err)
	}

	withServiceCLIOpts(t, configPath, func() {
		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		aliasPath := filepath.Join(execDir, "geosearch")

		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origReadlink := aliasFileReadlink
		origRemove := aliasFileRemove
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileReadlink = origReadlink
			aliasFileRemove = origRemove
		})

		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) {
			if path == aliasPath {
				return symlinkFileInfo{}, nil
			}
			return nil, os.ErrNotExist
		}
		aliasFileReadlink = func(path string) (string, error) {
			if path != aliasPath {
				return "", os.ErrNotExist
			}
			return execPath, nil
		}
		aliasFileRemove = func(path string) error {
			if path == aliasPath {
				return os.ErrPermission
			}
			return nil
		}

		removeCmd := newAliasRemoveCommand()
		removeCmd.SetArgs([]string{"geosearch"})
		_, err := captureStdout(t, removeCmd.Execute)
		if err == nil {
			t.Fatal("expected alias remove to fail when executable cleanup fails")
		}
		if !strings.Contains(err.Error(), "remove command alias executable") {
			t.Fatalf("expected executable cleanup error, got %v", err)
		}

		updatedCfg, loadErr := loadAppConfig()
		if loadErr != nil {
			t.Fatalf("reload config: %v", loadErr)
		}
		if got := updatedCfg.CommandAliases["geosearch"]; got != "open-meteo-geocoding.search" {
			t.Fatalf("expected command alias config preserved on cleanup failure, got %+v", updatedCfg.CommandAliases)
		}
	})
}

func TestAliasSetActionAliasRollsBackExecutableWhenConfigWriteFails(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		installer := installerFromConfig(cfg)
		manifest := &services.ServiceManifest{
			Name:    "open-meteo-geocoding",
			Version: "1.0.0",
			Adapter: "http",
			BaseURL: "https://example.com",
			Auth:    services.ServiceAuth{Type: "none"},
			Actions: map[string]services.ServiceAction{
				"search": {
					Method:      "GET",
					Path:        "/search",
					Description: "search",
					Risk:        services.RiskSpec{Level: "low"},
					Response:    services.ResponseSpec{Type: "object"},
				},
			},
		}
		if _, err := installer.Install(manifest, "local"); err != nil {
			t.Fatalf("install service: %v", err)
		}

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		aliasPath := filepath.Join(execDir, "geosearch")
		symlinkExists := false
		rollbackRemoved := false

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
			if oldname != execPath || newname != aliasPath {
				t.Fatalf("unexpected symlink args old=%q new=%q", oldname, newname)
			}
			symlinkExists = true
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
				rollbackRemoved = true
				return nil
			}
			return os.ErrNotExist
		}

		configDir := filepath.Dir(configPath)
		if err := os.Chmod(configDir, 0o500); err != nil {
			t.Fatalf("chmod config dir read-only: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(configDir, 0o700)
		})

		setCmd := newAliasSetCommand()
		setCmd.SetArgs([]string{"geosearch", "open-meteo-geocoding.search"})
		_, err = captureStdout(t, setCmd.Execute)
		if err == nil {
			t.Fatal("expected alias set to fail when config write fails")
		}
		if !rollbackRemoved {
			t.Fatal("expected executable alias rollback to run after config write failure")
		}

		if err := os.Chmod(configDir, 0o700); err != nil {
			t.Fatalf("restore config dir permissions: %v", err)
		}
		updatedCfg, loadErr := loadAppConfig()
		if loadErr != nil {
			t.Fatalf("reload config: %v", loadErr)
		}
		if _, exists := updatedCfg.CommandAliases["geosearch"]; exists {
			t.Fatalf("expected command alias not to be persisted on failed write, got %+v", updatedCfg.CommandAliases)
		}
	})
}

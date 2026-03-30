package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/services"
)

func TestGeneratedDefaultActionAliasesIncludeShortFallbacks(t *testing.T) {
	manifest := &services.ServiceManifest{
		Name:    "open-meteo-geocoding",
		Version: "1.0.0",
		Aliases: []string{"geo"},
		Actions: map[string]services.ServiceAction{
			"search": {
				Method:   "GET",
				Path:     "/v1/search",
				Response: services.ResponseSpec{Type: "object"},
				Risk:     services.RiskSpec{Level: "low"},
			},
		},
	}

	candidates := generatedDefaultActionAliases(manifest, "search")
	if len(candidates) < 2 {
		t.Fatalf("expected multiple alias candidates, got %+v", candidates)
	}
	if candidates[0] != "geosearch" {
		t.Fatalf("expected first candidate to be geosearch, got %+v", candidates)
	}
	if candidates[1] != "geosear" {
		t.Fatalf("expected second candidate to be geosear fallback, got %+v", candidates)
	}
}

func TestEnsureInstalledActionAliasesFallsBackWhenPrimaryCollidesWithServiceName(t *testing.T) {
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

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		installer := installerFromConfig(cfg)

		collidingService := &services.ServiceManifest{
			Name:    "geosearch",
			Version: "1.0.0",
			Adapter: "http",
			BaseURL: "https://example.com",
			Auth:    services.ServiceAuth{Type: string(actions.AuthTypeNone)},
			Actions: map[string]services.ServiceAction{
				"ping": {
					Method:   "GET",
					Path:     "/ping",
					Response: services.ResponseSpec{Type: "object"},
					Risk:     services.RiskSpec{Level: "low"},
				},
			},
		}
		if _, err := installer.Install(collidingService, "local"); err != nil {
			t.Fatalf("install colliding service: %v", err)
		}

		targetManifest := &services.ServiceManifest{
			Name:    "open-meteo-geocoding",
			Version: "1.0.0",
			Adapter: "http",
			BaseURL: "https://example.com",
			Aliases: []string{"geo"},
			Auth:    services.ServiceAuth{Type: string(actions.AuthTypeNone)},
			Actions: map[string]services.ServiceAction{
				"search": {
					Method:   "GET",
					Path:     "/search",
					Response: services.ResponseSpec{Type: "object"},
					Risk:     services.RiskSpec{Level: "low"},
				},
			},
		}

		created, skipped, err := ensureInstalledActionAliases(cfg, installer, targetManifest)
		if err != nil {
			t.Fatalf("ensureInstalledActionAliases() error: %v", err)
		}
		if len(created) == 0 {
			t.Fatalf("expected fallback alias to be created, skipped=%+v", skipped)
		}
		if created[0] == "geosearch" {
			t.Fatalf("expected collision-aware fallback alias, got primary colliding alias: %+v", created)
		}

		target := cfg.CommandAliases[created[0]]
		if target != "open-meteo-geocoding.search" {
			t.Fatalf("unexpected command alias target: %q", target)
		}
	})
}

func TestEnsureInstalledActionAliasesSkipsCandidateThatMatchesServiceAlias(t *testing.T) {
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

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		cfg.Aliases = map[string]string{"geo": "open-meteo-geocoding"}

		targetManifest := &services.ServiceManifest{
			Name:    "open-meteo-geocoding",
			Version: "1.0.0",
			Adapter: "http",
			BaseURL: "https://example.com",
			Auth:    services.ServiceAuth{Type: string(actions.AuthTypeNone)},
			Actions: map[string]services.ServiceAction{
				"search": {
					Method:   "GET",
					Path:     "/search",
					Aliases:  []string{"geo", "geosearch", "geolookup"},
					Response: services.ResponseSpec{Type: "object"},
					Risk:     services.RiskSpec{Level: "low"},
				},
			},
		}

		installer := installerFromConfig(cfg)
		created, _, err := ensureInstalledActionAliases(cfg, installer, targetManifest)
		if err != nil {
			t.Fatalf("ensureInstalledActionAliases() error: %v", err)
		}
		if len(created) != 2 {
			t.Fatalf("expected all non-conflicting explicit aliases to be created, got %+v", created)
		}
		createdSet := map[string]struct{}{}
		for _, alias := range created {
			createdSet[alias] = struct{}{}
		}
		if _, ok := createdSet["geosearch"]; !ok {
			t.Fatalf("expected geosearch alias to be created, got %+v", created)
		}
		if _, ok := createdSet["geolookup"]; !ok {
			t.Fatalf("expected geolookup alias to be created, got %+v", created)
		}
	})
}

func TestSelectServiceAliasCandidateUsesDeterministicConfiguredAlias(t *testing.T) {
	cfg := &config.KimbapConfig{
		Aliases: map[string]string{
			"bb": "svc",
			"aa": "svc",
			"1":  "svc",
		},
	}
	installer := services.NewLocalInstaller(t.TempDir())
	manifest := &services.ServiceManifest{Name: "svc", Version: "1.0.0"}

	alias, alreadyConfigured, err := selectServiceAliasCandidate(cfg, installer, manifest)
	if err != nil {
		t.Fatalf("selectServiceAliasCandidate() error: %v", err)
	}
	if !alreadyConfigured {
		t.Fatal("expected configured alias to be recognized")
	}
	if alias != "aa" {
		t.Fatalf("expected deterministic shortest+lexicographic alias 'aa', got %q", alias)
	}
}

func TestSelectServiceAliasCandidateIgnoresInvalidConfiguredAlias(t *testing.T) {
	cfg := &config.KimbapConfig{
		Aliases: map[string]string{
			"1": "open-meteo-geocoding",
		},
	}
	installer := services.NewLocalInstaller(t.TempDir())
	manifest := &services.ServiceManifest{Name: "open-meteo-geocoding", Version: "1.0.0"}

	alias, alreadyConfigured, err := selectServiceAliasCandidate(cfg, installer, manifest)
	if err != nil {
		t.Fatalf("selectServiceAliasCandidate() error: %v", err)
	}
	if alreadyConfigured {
		t.Fatalf("expected invalid configured alias to be ignored, got alias=%q alreadyConfigured=%v", alias, alreadyConfigured)
	}
	if alias == "1" {
		t.Fatalf("expected invalid configured alias to be ignored, got %q", alias)
	}
}

func TestEnsureInstalledActionAliasesGeneratedExistingMappingDoesNotEmitNoCandidateWarning(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		cfg.CommandAliases = map[string]string{"geosearch": "open-meteo-geocoding.search"}

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		aliasPath := filepath.Join(execDir, "geosearch")
		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origReadlink := aliasFileReadlink
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileReadlink = origReadlink
		})
		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) {
			if path == aliasPath {
				return symlinkFileInfo{}, nil
			}
			return nil, os.ErrNotExist
		}
		aliasFileReadlink = func(path string) (string, error) {
			if path == aliasPath {
				return execPath, nil
			}
			return "", os.ErrNotExist
		}

		manifest := &services.ServiceManifest{
			Name:    "open-meteo-geocoding",
			Version: "1.0.0",
			Adapter: "http",
			BaseURL: "https://example.com",
			Auth:    services.ServiceAuth{Type: string(actions.AuthTypeNone)},
			Actions: map[string]services.ServiceAction{
				"search": {
					Method:   "GET",
					Path:     "/search",
					Response: services.ResponseSpec{Type: "object"},
					Risk:     services.RiskSpec{Level: "low"},
				},
			},
		}

		created, skipped, err := ensureInstalledActionAliases(cfg, services.NewLocalInstaller(servicesDir), manifest)
		if err != nil {
			t.Fatalf("ensureInstalledActionAliases() error: %v", err)
		}
		if len(created) != 0 {
			t.Fatalf("expected no new aliases when generated alias already maps correctly, got %+v", created)
		}
		for _, item := range skipped {
			if strings.Contains(item, "no collision-free alias candidate") {
				t.Fatalf("expected no no-candidate warning when existing mapping satisfies generated alias, skipped=%+v", skipped)
			}
		}
	})
}

func TestEnsureInstalledActionAliasesRecreatesMissingExecutableForExistingMapping(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		cfg.CommandAliases = map[string]string{"weather": "open-meteo.get-forecast"}

		execDir := t.TempDir()
		execPath := filepath.Join(execDir, "kimbap")
		aliasPath := filepath.Join(execDir, "weather")
		symlinkCreated := false

		origExecutablePath := aliasExecutablePath
		origLstat := aliasFileLstat
		origSymlink := aliasFileSymlink
		t.Cleanup(func() {
			aliasExecutablePath = origExecutablePath
			aliasFileLstat = origLstat
			aliasFileSymlink = origSymlink
		})

		aliasExecutablePath = func() (string, error) { return execPath, nil }
		aliasFileLstat = func(path string) (os.FileInfo, error) {
			if path == aliasPath {
				return nil, os.ErrNotExist
			}
			return nil, os.ErrNotExist
		}
		aliasFileSymlink = func(oldname, newname string) error {
			if oldname != execPath || newname != aliasPath {
				t.Fatalf("unexpected symlink args old=%q new=%q", oldname, newname)
			}
			symlinkCreated = true
			return nil
		}

		manifest := &services.ServiceManifest{
			Name:    "open-meteo",
			Version: "1.0.0",
			Adapter: "http",
			BaseURL: "https://example.com",
			Auth:    services.ServiceAuth{Type: string(actions.AuthTypeNone)},
			Actions: map[string]services.ServiceAction{
				"get-forecast": {
					Method:   "GET",
					Path:     "/forecast",
					Aliases:  []string{"weather"},
					Response: services.ResponseSpec{Type: "object"},
					Risk:     services.RiskSpec{Level: "low"},
				},
			},
		}

		created, skipped, runErr := ensureInstalledActionAliases(cfg, services.NewLocalInstaller(servicesDir), manifest)
		if runErr != nil {
			t.Fatalf("ensureInstalledActionAliases() error: %v", runErr)
		}
		if !symlinkCreated {
			t.Fatal("expected missing executable alias to be recreated")
		}
		if len(created) == 0 || created[0] != "weather" {
			t.Fatalf("expected recreated alias to be reported in created list, got %+v", created)
		}
		if len(skipped) > 0 {
			t.Fatalf("expected no skips while recreating existing alias mapping, got %+v", skipped)
		}
	})
}

func TestEnsureInstalledActionAliasesRollsBackExecutableWhenConfigWriteFails(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
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

		manifest := &services.ServiceManifest{
			Name:    "open-meteo-geocoding",
			Version: "1.0.0",
			Adapter: "http",
			BaseURL: "https://example.com",
			Auth:    services.ServiceAuth{Type: string(actions.AuthTypeNone)},
			Actions: map[string]services.ServiceAction{
				"search": {
					Method:   "GET",
					Path:     "/search",
					Aliases:  []string{"geosearch"},
					Response: services.ResponseSpec{Type: "object"},
					Risk:     services.RiskSpec{Level: "low"},
				},
			},
		}

		configDir := filepath.Dir(configPath)
		if err := os.Chmod(configDir, 0o500); err != nil {
			t.Fatalf("chmod config dir read-only: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(configDir, 0o700)
		})

		created, _, runErr := ensureInstalledActionAliases(cfg, services.NewLocalInstaller(servicesDir), manifest)
		if runErr == nil {
			t.Fatal("expected ensureInstalledActionAliases to fail when config write fails")
		}
		if len(created) != 0 {
			t.Fatalf("expected no aliases created on failed config write, got %+v", created)
		}
		if !rollbackRemoved {
			t.Fatal("expected executable alias rollback to run after config write failure")
		}

		if err := os.Chmod(configDir, 0o700); err != nil {
			t.Fatalf("restore config dir permissions: %v", err)
		}
		reloaded, loadErr := loadAppConfig()
		if loadErr != nil {
			t.Fatalf("reload config: %v", loadErr)
		}
		if _, exists := reloaded.CommandAliases["geosearch"]; exists {
			t.Fatalf("expected geosearch not to be persisted on failed write, got %+v", reloaded.CommandAliases)
		}
	})
}

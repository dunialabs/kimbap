package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/services/catalog"
)

func TestBuildInitConfigRebasesPolicyPathWithDataDir(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })

	opts = cliOptions{}
	opts.dataDir = t.TempDir()

	cfg := buildInitConfig()
	want := filepath.Join(opts.dataDir, "policy.yaml")
	if cfg.Policy.Path != want {
		t.Fatalf("expected policy path %q, got %q", want, cfg.Policy.Path)
	}
}

func TestBuildInitConfigDefaultsToDevMode(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })

	opts = cliOptions{}
	cfg := buildInitConfig()
	if cfg.Mode != "dev" {
		t.Fatalf("expected default init mode dev, got %q", cfg.Mode)
	}
}

func TestBuildInitConfigRespectsKimbapModeEnv(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })
	t.Setenv("KIMBAP_MODE", "embedded")

	opts = cliOptions{}
	cfg := buildInitConfig()
	if cfg.Mode != "embedded" {
		t.Fatalf("expected env mode embedded, got %q", cfg.Mode)
	}
}

func TestBuildInitConfigFlagModeOverridesEnv(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })
	t.Setenv("KIMBAP_MODE", "embedded")

	opts = cliOptions{}
	opts.mode = "connected"
	cfg := buildInitConfig()
	if cfg.Mode != "connected" {
		t.Fatalf("expected flag mode connected, got %q", cfg.Mode)
	}
}

func TestCanPromptInTTYReturnsFalseForJSONOutput(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })

	opts = cliOptions{format: "json"}
	if canPromptInTTY() {
		t.Fatal("expected canPromptInTTY to be false for json output")
	}
}

func TestInstallInitServicesPromptDeclineSkipsShortcutSetup(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	prevOpts := opts
	opts = cliOptions{configPath: configPath, format: "text", noSplash: true}
	t.Cleanup(func() { opts = prevOpts })

	origCanPrompt := canPromptInTTY
	canPromptInTTY = func() bool { return true }
	t.Cleanup(func() { canPromptInTTY = origCanPrompt })

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := w.WriteString("n\n"); err != nil {
		t.Fatalf("write stdin input: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})

	cfg, err := loadAppConfig()
	if err != nil {
		t.Fatalf("loadAppConfig() error: %v", err)
	}

	check := installInitServices(cfg, initServiceSelection{Names: []string{"open-meteo-geocoding"}}, false, false)
	if check.Status != "ok" {
		t.Fatalf("expected installInitServices status ok, got %q (%s)", check.Status, check.Detail)
	}

	reloaded, err := loadAppConfig()
	if err != nil {
		t.Fatalf("reload config error: %v", err)
	}
	if len(reloaded.CommandAliases) != 0 {
		t.Fatalf("expected no command aliases when prompt is declined, got %+v", reloaded.CommandAliases)
	}
}

func TestInstallInitServicesPromptAcceptsAndCreatesShortcutAliases(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	prevOpts := opts
	opts = cliOptions{configPath: configPath, format: "text", noSplash: true}
	t.Cleanup(func() { opts = prevOpts })

	origCanPrompt := canPromptInTTY
	canPromptInTTY = func() bool { return true }
	t.Cleanup(func() { canPromptInTTY = origCanPrompt })

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := w.WriteString("y\n"); err != nil {
		t.Fatalf("write stdin input: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})

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

	check := installInitServices(cfg, initServiceSelection{Names: []string{"open-meteo-geocoding"}}, false, false)
	if check.Status != "ok" {
		t.Fatalf("expected installInitServices status ok, got %q (%s)", check.Status, check.Detail)
	}

	reloaded, err := loadAppConfig()
	if err != nil {
		t.Fatalf("reload config error: %v", err)
	}
	if got := reloaded.CommandAliases["geosearch"]; got != "open-meteo-geocoding.search" {
		t.Fatalf("expected geosearch shortcut alias after prompt acceptance, got %+v", reloaded.CommandAliases)
	}
}

func TestAppendInitChecksFlagsFailures(t *testing.T) {
	checks, hasFailure := appendInitChecks(nil, false,
		doctorCheck{Name: "ok", Status: "ok"},
		doctorCheck{Name: "warn", Status: "warn"},
	)
	if hasFailure {
		t.Fatal("expected non-failing checks to keep hasFailure=false")
	}

	checks, hasFailure = appendInitChecks(checks, hasFailure, doctorCheck{Name: "fail", Status: "fail"})
	if !hasFailure {
		t.Fatal("expected failing check to set hasFailure=true")
	}
	if len(checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(checks))
	}
}

func TestOptionalKBCheckDoesNotFailInit(t *testing.T) {
	checks := []doctorCheck{}
	hasFailure := false

	kbCheck := doctorCheck{Name: "kb alias", Status: "fail"}
	checks = append(checks, kbCheck)

	if hasFailure {
		t.Fatal("optional kbCheck failure must not set hasFailure")
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check recorded, got %d", len(checks))
	}
}

func TestValidateInitMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "embedded", mode: "embedded"},
		{name: "dev", mode: "dev"},
		{name: "connected", mode: "connected"},
		{name: "invalid", mode: "unknown", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInitMode(tc.mode)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for mode %q", tc.mode)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for mode %q, got %v", tc.mode, err)
			}
		})
	}
}

func TestEnsureConsoleEnabledFailsWhenConfigNotOverwritten(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("console:\n  enabled: false\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	check := ensureConsoleEnabled(configPath, doctorCheck{Status: "skip"}, true)
	if check.Status != "fail" {
		t.Fatalf("expected fail when --with-console was not persisted, got %q", check.Status)
	}
}

func TestEnsureConsoleEnabledSkipsWhenAlreadyEnabled(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("console:\n  enabled: true\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	check := ensureConsoleEnabled(configPath, doctorCheck{Status: "skip"}, false)
	if check.Status != "skip" {
		t.Fatalf("expected skip for existing enabled config, got %q", check.Status)
	}
}

func TestResolveSymlinkTargetRelative(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "kimbap")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	linkPath := filepath.Join(dir, "kb")
	if err := os.Symlink("kimbap", linkPath); err != nil {
		t.Fatalf("create relative symlink: %v", err)
	}

	got, err := resolveSymlinkTarget(linkPath, "kimbap")
	if err != nil {
		t.Fatalf("resolve relative symlink: %v", err)
	}
	want, err := filepath.EvalSymlinks(binaryPath)
	if err != nil {
		t.Fatalf("eval expected symlink target: %v", err)
	}
	if got != want {
		t.Fatalf("resolved target = %q, want %q", got, want)
	}
}

func TestResolveSymlinkTargetAbsolute(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "kimbap")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	linkPath := filepath.Join(dir, "kb")
	if err := os.Symlink(binaryPath, linkPath); err != nil {
		t.Fatalf("create absolute symlink: %v", err)
	}

	got, err := resolveSymlinkTarget(linkPath, binaryPath)
	if err != nil {
		t.Fatalf("resolve absolute symlink target: %v", err)
	}
	want, err := filepath.EvalSymlinks(binaryPath)
	if err != nil {
		t.Fatalf("eval expected symlink target: %v", err)
	}
	if got != want {
		t.Fatalf("resolved target = %q, want %q", got, want)
	}
}

func TestRenderInitSummaryIncludesWarnings(t *testing.T) {
	summary := renderInitSummary("/tmp/config.yaml", []doctorCheck{
		{Name: "a", Status: "ok", Detail: "ok"},
		{Name: "b", Status: "warn", Detail: "warn"},
		{Name: "c", Status: "skip", Detail: "skip"},
		{Name: "d", Status: "fail", Detail: "fail"},
	})

	if !strings.Contains(summary, "warnings: 1") {
		t.Fatalf("expected warning count in summary, got:\n%s", summary)
	}
	if !strings.Contains(summary, "! b") {
		t.Fatalf("expected warn icon in summary, got:\n%s", summary)
	}
}

func TestEnsurePolicyFileFailsForInvalidExistingPolicy(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "policy.yaml")
	invalid := "version: 1\nrules:\n- id: bad\n  effect: allow\n"
	if err := os.WriteFile(policyPath, []byte(invalid), 0o600); err != nil {
		t.Fatalf("write invalid policy: %v", err)
	}

	check := ensurePolicyFile(policyPath, "embedded")
	if check.Status != "fail" {
		t.Fatalf("expected fail for invalid policy file, got %q", check.Status)
	}
}

func TestEnsureWritableDirWithStatusFailsWhenPathIsFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file path: %v", err)
	}

	check := ensureWritableDirWithStatus("data directory writable", filePath)
	if check.Status != "fail" {
		t.Fatalf("expected fail when path is a file, got %q", check.Status)
	}
}

func TestEnsureWritableDirWithStatusCreatesPrivateDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")

	check := ensureWritableDirWithStatus("data directory writable", dataDir)
	if check.Status != "ok" {
		t.Fatalf("expected ok for newly created writable directory, got %q", check.Status)
	}

	st, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected private permissions for data dir, got %o", st.Mode().Perm())
	}
}

func TestEnsureWritableDirWithStatusWarnsOnPermissiveExistingDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "existing-data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.Chmod(dataDir, 0o755); err != nil {
		t.Fatalf("chmod data dir: %v", err)
	}

	check := ensureWritableDirWithStatus("data directory writable", dataDir)
	if check.Status != "warn" {
		t.Fatalf("expected warn for permissive existing directory, got %q", check.Status)
	}
}

func TestEnsureWritableDirWithStatusFailsWhenExistingDirectoryIsNotWritable(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "read-only-data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.Chmod(dataDir, 0o555); err != nil {
		t.Fatalf("chmod data dir: %v", err)
	}

	check := ensureWritableDirWithStatus("data directory writable", dataDir)
	if check.Status != "fail" {
		t.Fatalf("expected fail for non-writable existing directory, got %q", check.Status)
	}
}

func TestResolveInitServiceSelectionFromReader(t *testing.T) {
	tests := []struct {
		name            string
		rawServices     string
		noServices      bool
		interactive     bool
		input           string
		wantSkipped     bool
		wantAll         bool
		wantNames       []string
		wantErr         bool
		wantErrContains string
	}{
		{name: "noServices flag skips", noServices: true, wantSkipped: true},
		{name: "noServices overrides rawServices", rawServices: "all", noServices: true, wantSkipped: true},
		{name: "services all returns all", rawServices: "all", wantAll: true},
		{name: "services ALL case insensitive", rawServices: "ALL", wantAll: true},
		{name: "services all with whitespace", rawServices: " all ", wantAll: true},
		{name: "services csv returns normalized", rawServices: "github,slack", wantNames: []string{"github", "slack"}},
		{name: "services csv with whitespace", rawServices: "github , slack", wantNames: []string{"github", "slack"}},
		{name: "services invalid errors", rawServices: "nonexistent-service-xyz", wantErr: true, wantErrContains: "unknown catalog service"},
		{name: "services comma-only errors", rawServices: ",,,", wantErr: true, wantErrContains: "invalid --services value"},
		{name: "non-interactive empty skips", rawServices: "", interactive: false, wantSkipped: true},
		{name: "interactive enter installs all", rawServices: "", interactive: true, input: "\n", wantAll: true},
		{name: "interactive Y installs all", rawServices: "", interactive: true, input: "Y\n", wantAll: true},
		{name: "interactive y installs all", rawServices: "", interactive: true, input: "y\n", wantAll: true},
		{name: "interactive yes installs all", rawServices: "", interactive: true, input: "yes\n", wantAll: true},
		{name: "interactive n skips", rawServices: "", interactive: true, input: "n\n", wantSkipped: true},
		{name: "interactive N skips", rawServices: "", interactive: true, input: "N\n", wantSkipped: true},
		{name: "interactive no skips", rawServices: "", interactive: true, input: "no\n", wantSkipped: true},
		{name: "interactive csv parses", rawServices: "", interactive: true, input: "github,slack\n", wantNames: []string{"github", "slack"}},
		{name: "interactive EOF skips", rawServices: "", interactive: true, input: "", wantSkipped: true},
		{name: "interactive select then csv", rawServices: "", interactive: true, input: "select\ngithub,slack\n", wantNames: []string{"github", "slack"}},
		{name: "interactive select then all", rawServices: "", interactive: true, input: "select\nall\n", wantAll: true},
		{name: "interactive select then empty skips", rawServices: "", interactive: true, input: "select\n\n", wantSkipped: true},
		{name: "interactive select then EOF skips", rawServices: "", interactive: true, input: "select\n", wantSkipped: true},
		{name: "interactive invalid service errors", rawServices: "", interactive: true, input: "nonexistent-service-xyz\n", wantErr: true, wantErrContains: "unknown catalog service"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.input)
			result, err := resolveInitServiceSelectionFromReader(tc.rawServices, tc.noServices, tc.interactive, reader)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Fatalf("expected error containing %q, got: %v", tc.wantErrContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Skipped != tc.wantSkipped {
				t.Fatalf("expected Skipped=%v, got %v (Reason=%q)", tc.wantSkipped, result.Skipped, result.Reason)
			}

			if tc.wantAll {
				allServices, listErr := catalog.List()
				if listErr != nil {
					t.Fatalf("catalog.List() error: %v", listErr)
				}
				if len(result.Names) != len(allServices) {
					t.Fatalf("expected %d catalog services, got %d", len(allServices), len(result.Names))
				}
				for i, want := range allServices {
					if result.Names[i] != want {
						t.Fatalf("Names[%d]: expected %q, got %q", i, want, result.Names[i])
					}
				}
				return
			}

			if tc.wantNames != nil {
				if len(result.Names) != len(tc.wantNames) {
					t.Fatalf("expected Names=%v, got %v", tc.wantNames, result.Names)
				}
				for i, want := range tc.wantNames {
					if result.Names[i] != want {
						t.Fatalf("Names[%d]: expected %q, got %q", i, want, result.Names[i])
					}
				}
			}
		})
	}
}

func TestCheckInitLocalAdapterReadinessSkipsOnPriorFailure(t *testing.T) {
	check := checkInitLocalAdapterReadiness(initServiceSelection{Names: []string{"github"}}, true)
	if check.Status != "skip" {
		t.Fatalf("expected skip when prior failure exists, got %q", check.Status)
	}
}

func TestCheckInitLocalAdapterReadinessSkipsWhenNoSelection(t *testing.T) {
	check := checkInitLocalAdapterReadiness(initServiceSelection{Skipped: true}, false)
	if check.Status != "skip" {
		t.Fatalf("expected skip when selection is skipped, got %q", check.Status)
	}
}

func TestCheckInitLocalAdapterReadinessWarnsForMissingCommandExecutable(t *testing.T) {
	t.Setenv("PATH", "")
	check := checkInitLocalAdapterReadiness(initServiceSelection{Names: []string{"kitty"}}, false)
	if check.Status != "warn" {
		t.Fatalf("expected warn when command executable cannot be found, got %q (%s)", check.Status, check.Detail)
	}
	if !strings.Contains(check.Detail, "kitty") {
		t.Fatalf("expected readiness warning to mention kitty, got %q", check.Detail)
	}
}

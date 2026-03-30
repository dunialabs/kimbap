package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentsSetupNoAgentsDetected(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", homeDir)

	prev := opts
	opts = cliOptions{format: "text", noSplash: true}
	t.Cleanup(func() { opts = prev })

	cmd := newAgentsSetupCommand()
	cmd.SetArgs([]string{"--no-sync"})

	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("agents setup failed: %v", err)
	}

	if !strings.Contains(output, "No AI agents detected") {
		t.Fatalf("expected no-agents guidance in output, got: %s", output)
	}
}

func TestAgentsSetupNoAgentsDetected_JSON(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", homeDir)

	prev := opts
	opts = cliOptions{format: "json", noSplash: true}
	t.Cleanup(func() { opts = prev })

	cmd := newAgentsSetupCommand()
	cmd.SetArgs([]string{"--no-sync"})

	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("agents setup --format json failed: %v", err)
	}

	if strings.TrimSpace(output) != "[]" {
		t.Fatalf("expected empty JSON array for zero agents, got: %q", output)
	}
}

func TestAgentsSetupSyncJSONIncludesSyncMetadata(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", homeDir)

	prev := opts
	opts = cliOptions{format: "json", noSplash: true}
	t.Cleanup(func() { opts = prev })

	projectDir := t.TempDir()
	cmd := newAgentsSetupCommand()
	cmd.SetArgs([]string{"--sync", "--dir", projectDir})

	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("agents setup --sync --format json failed: %v", err)
	}

	var payload map[string]any
	if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
		t.Fatalf("expected JSON object output, got %q (err=%v)", output, unmarshalErr)
	}
	if enabled, ok := payload["sync_enabled"].(bool); !ok || !enabled {
		t.Fatalf("expected sync_enabled=true in payload, got %+v", payload)
	}
	if _, ok := payload["sync_result"].(map[string]any); !ok {
		t.Fatalf("expected sync_result object in payload, got %+v", payload)
	}
}

func TestAgentsSetupSyncTextReportsNoAgentsDetectedForSync(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", homeDir)

	prev := opts
	opts = cliOptions{format: "text", noSplash: true}
	t.Cleanup(func() { opts = prev })

	projectDir := t.TempDir()
	cmd := newAgentsSetupCommand()
	cmd.SetArgs([]string{"--sync", "--dir", projectDir})

	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("agents setup --sync failed: %v", err)
	}
	if !strings.Contains(output, "No AI agents detected") {
		t.Fatalf("expected no-agents setup message, got: %q", output)
	}
	if !strings.Contains(output, "No agent environments detected for sync") {
		t.Fatalf("expected explicit no-agent sync message, got: %q", output)
	}
}

func TestResolveAgentsSyncProjectDirRejectsRoot(t *testing.T) {
	_, err := resolveAgentsSyncProjectDir("/")
	if err == nil {
		t.Fatal("expected root directory to be rejected")
	}
	if !strings.Contains(err.Error(), "refusing to sync to root directory") {
		t.Fatalf("expected root rejection error, got %v", err)
	}
}

func TestResolveAgentsSyncProjectDirResolvesAbsoluteDirectory(t *testing.T) {
	dir := t.TempDir()
	resolved, err := resolveAgentsSyncProjectDir(dir)
	if err != nil {
		t.Fatalf("resolveAgentsSyncProjectDir() error: %v", err)
	}
	want, wantErr := filepath.Abs(dir)
	if wantErr != nil {
		t.Fatalf("filepath.Abs(%q) error: %v", dir, wantErr)
	}
	if resolved != want {
		t.Fatalf("resolved dir = %q, want %q", resolved, want)
	}
}

func TestAgentsSetupHelpExcludesGenericAgentKind(t *testing.T) {
	cmd := newAgentsSetupCommand()
	cmd.SetArgs([]string{"--help"})

	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("agents setup --help failed: %v", err)
	}
	if strings.Contains(output, "generic") {
		t.Fatalf("expected setup help to exclude generic agent kind, got: %q", output)
	}
}

func TestAgentsSyncHelpIncludesGenericAgentKind(t *testing.T) {
	cmd := newAgentsSyncCommand()
	cmd.SetArgs([]string{"--help"})

	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("agents sync --help failed: %v", err)
	}
	if !strings.Contains(output, "generic") {
		t.Fatalf("expected sync help to include generic agent kind, got: %q", output)
	}
}

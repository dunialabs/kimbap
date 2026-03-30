package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/agents"
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

func TestParseAgentKindsNormalizesCaseAndWhitespace(t *testing.T) {
	kinds := parseAgentKinds(" CODEX,  Claude-Code ,opencode ")
	if len(kinds) != 3 {
		t.Fatalf("expected 3 kinds, got %d", len(kinds))
	}
	if kinds[0] != "codex" {
		t.Fatalf("kinds[0] = %q, want codex", kinds[0])
	}
	if kinds[1] != "claude-code" {
		t.Fatalf("kinds[1] = %q, want claude-code", kinds[1])
	}
	if kinds[2] != "opencode" {
		t.Fatalf("kinds[2] = %q, want opencode", kinds[2])
	}
}

func TestRunAgentsSyncSkipsUnmanagedSkillDirsWithoutError(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)
	manifestPath := filepath.Join(t.TempDir(), "notes-service.yaml")
	writeLocalManifest(t, manifestPath, "notes-service", "1.0.0")
	projectDir := t.TempDir()
	unmanagedDir := filepath.Join(projectDir, ".claude", "skills", "notes-service")
	if err := os.MkdirAll(unmanagedDir, 0o755); err != nil {
		t.Fatalf("create unmanaged dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(unmanagedDir, "SKILL.md"), []byte("# custom\n"), 0o644); err != nil {
		t.Fatalf("write unmanaged skill: %v", err)
	}

	withServiceCLIOpts(t, configPath, func() {
		installCmd := newServiceInstallCommand()
		installCmd.SetArgs([]string{manifestPath, "--no-shortcuts"})
		if _, err := captureStdout(t, installCmd.Execute); err != nil {
			t.Fatalf("service install failed: %v", err)
		}

		result, err := runAgentsSync(projectDir, "claude-code", "", false, false)
		if err != nil {
			t.Fatalf("runAgentsSync returned unexpected error: %v", err)
		}
		if result.AgentsFound != 1 {
			t.Fatalf("expected 1 agent found, got %d", result.AgentsFound)
		}
		if len(result.SyncResults) != 1 {
			t.Fatalf("expected 1 sync result, got %d", len(result.SyncResults))
		}
		syncResult := result.SyncResults[0]
		if len(syncResult.Failed) != 0 {
			t.Fatalf("expected no failed entries, got %+v", syncResult.Failed)
		}
		if len(syncResult.Errors) != 0 {
			t.Fatalf("expected no errors, got %+v", syncResult.Errors)
		}
		if len(syncResult.Skipped) != 1 || syncResult.Skipped[0] != "notes-service" {
			t.Fatalf("expected unmanaged directory to be skipped, got %+v", syncResult.Skipped)
		}
		if len(syncResult.Protected) != 1 || syncResult.Protected[0] != "notes-service" {
			t.Fatalf("expected unmanaged directory to be marked protected, got %+v", syncResult.Protected)
		}
	})
}

func TestDescribeAgentsSetupSyncOutcomeDistinguishesProtectedSkips(t *testing.T) {
	msg := describeAgentsSetupSyncOutcome(&agentSetupResult{
		AgentsFound: 1,
		SyncResults: []agents.SyncResult{{Agent: agents.AgentClaudeCode, Protected: []string{"notes-service"}, Skipped: []string{"notes-service"}}},
	}, "/tmp/project", false)
	if !strings.Contains(msg, "existing unmanaged skill directories") {
		t.Fatalf("expected protected skip message, got %q", msg)
	}

	msg = describeAgentsSetupSyncOutcome(&agentSetupResult{
		AgentsFound: 1,
		SyncResults: []agents.SyncResult{{Agent: agents.AgentClaudeCode, Skipped: []string{"notes-service"}}},
	}, "/tmp/project", false)
	if !strings.Contains(msg, "already in sync") {
		t.Fatalf("expected already-in-sync message for ordinary skips, got %q", msg)
	}
}

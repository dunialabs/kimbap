package main

import (
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

	trimmed := strings.TrimSpace(output)
	if trimmed != "[]" && trimmed != "null" {
		t.Fatalf("expected empty JSON array for zero agents, got: %q", output)
	}
}

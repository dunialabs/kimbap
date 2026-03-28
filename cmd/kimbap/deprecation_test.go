package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestConnectorCommandRemoved(t *testing.T) {
	cmd := rootCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"connector", "login", "github"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected connector command to return removed error")
	}

	output := strings.ToLower(out.String() + "\n" + err.Error())
	if !strings.Contains(output, "removed") {
		t.Fatalf("expected removed language, got: %s", out.String())
	}
	if !strings.Contains(output, "auth connect") {
		t.Fatalf("expected migration hint to auth connect, got: %s", out.String())
	}
}

func TestProfileCommandRemoved(t *testing.T) {
	cmd := rootCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"profile", "install", "claude-code"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected profile command to return removed error")
	}

	output := strings.ToLower(out.String() + "\n" + err.Error())
	if !strings.Contains(output, "removed") {
		t.Fatalf("expected removed language, got: %s", out.String())
	}
	if !strings.Contains(output, "agents setup") {
		t.Fatalf("expected migration hint to agents setup, got: %s", out.String())
	}
}

func TestHelpSurfaceCommandCount(t *testing.T) {
	cmd := rootCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("--help failed: %v", err)
	}

	visibleCommands := countVisibleCommands(out.String())
	if visibleCommands > 15 {
		t.Fatalf("expected help surface to show at most 15 commands, got %d", visibleCommands)
	}
}

func countVisibleCommands(helpOutput string) int {
	lines := strings.Split(helpOutput, "\n")
	inAvailableCommands := false
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "Available Commands:":
			inAvailableCommands = true
			continue
		case inAvailableCommands && trimmed == "":
			inAvailableCommands = false
			continue
		}

		if !inAvailableCommands {
			continue
		}

		if strings.HasPrefix(line, "  ") {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] != "help" {
				count++
			}
		}
	}

	return count
}

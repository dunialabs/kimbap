package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandIncludesCoreFamilies(t *testing.T) {
	expected := []string{
		"call", "actions", "search", "vault", "token", "policy",
		"doctor", "init", "service", "connector", "auth", "link",
		"approve", "audit", "run", "daemon", "profile", "agents",
		"alias", "completion",
	}

	for _, name := range expected {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected command %q to be registered", name)
		}
	}
}

func TestDoctorHelpOutputContract(t *testing.T) {
	cmd := newDoctorCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor --help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Run runtime diagnostics") {
		t.Fatalf("expected doctor help contract string, got: %s", output)
	}
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected help usage block, got: %s", output)
	}
}

func TestApproveDenyValidationContract(t *testing.T) {
	cmd := newApproveDenyCommand()
	cmd.SetArgs([]string{"req-123"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing --reason")
	}
	if got := mapErrorToExitCode(err); got != ExitValidation {
		t.Fatalf("expected validation exit code %d, got %d", ExitValidation, got)
	}
}

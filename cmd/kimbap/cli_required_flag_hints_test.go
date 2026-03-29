package main

import (
	"strings"
	"testing"
)

func TestPolicySetRequiresFileWithHint(t *testing.T) {
	cmd := newPolicySetCommand()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected policy set to require --file")
	}
	if !strings.Contains(err.Error(), "--file is required") {
		t.Fatalf("expected missing file error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Run: kimbap policy set --file <path>") {
		t.Fatalf("expected actionable hint, got %v", err)
	}
}

func TestPolicyEvalRequiresAgentWithHint(t *testing.T) {
	cmd := newPolicyEvalCommand()
	cmd.SetArgs([]string{"--action", "github.issues.create"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected policy eval to require --agent")
	}
	if !strings.Contains(err.Error(), "--agent is required") {
		t.Fatalf("expected missing agent error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Run: kimbap policy eval --agent <name> --action <service.action>") {
		t.Fatalf("expected actionable hint, got %v", err)
	}
}

func TestPolicyEvalRequiresActionWithHint(t *testing.T) {
	cmd := newPolicyEvalCommand()
	cmd.SetArgs([]string{"--agent", "console"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected policy eval to require --action")
	}
	if !strings.Contains(err.Error(), "--action is required") {
		t.Fatalf("expected missing action error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Run: kimbap policy eval --agent console --action <service.action>") {
		t.Fatalf("expected actionable hint, got %v", err)
	}
}

func TestTokenCreateRequiresAgentWithHint(t *testing.T) {
	cmd := newTokenCreateCommand()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected token create to require --agent")
	}
	if !strings.Contains(err.Error(), "--agent is required") {
		t.Fatalf("expected missing agent error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Run: kimbap token create --agent <name>") {
		t.Fatalf("expected actionable hint, got %v", err)
	}
}

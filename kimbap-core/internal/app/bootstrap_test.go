package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/runtime"
)

func TestBuildRuntimeNilConfig(t *testing.T) {
	rt, err := BuildRuntime(RuntimeDeps{})
	if err == nil {
		t.Fatalf("expected error for nil config, got runtime=%v", rt)
	}
}

func TestBuildRuntimeMinimalConfig(t *testing.T) {
	cfg := &config.KimbapConfig{
		Skills: config.SkillsConfig{Dir: t.TempDir()},
		Policy: config.PolicyConfig{Path: ""},
	}

	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	if rt == nil {
		t.Fatal("expected runtime to be non-nil")
	}
}

func TestBuildRuntimeWithSkills(t *testing.T) {
	skillsDir := t.TempDir()
	const manifest = `name: test-skill
version: 1.0.0
description: test skill
base_url: https://api.example.com
auth:
  type: header
  header_name: Authorization
  credential_ref: test.token
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
      mutating: false
`
	if err := os.WriteFile(filepath.Join(skillsDir, "test-skill.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write skill manifest: %v", err)
	}

	cfg := &config.KimbapConfig{Skills: config.SkillsConfig{Dir: skillsDir}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	actions, err := rt.ActionRegistry.List(context.Background(), runtime.ListOptions{})
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action from loaded skill, got %d", len(actions))
	}
	if actions[0].Name != "test-skill.ping" {
		t.Fatalf("unexpected action name: %s", actions[0].Name)
	}
}

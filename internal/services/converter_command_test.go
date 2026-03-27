package services

import (
	"testing"
	"time"
)

func TestToActionDefinitions_CommandAdapterConfig(t *testing.T) {
	manifest := validCommandManifest()
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error = %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("len(defs) = %d, want 1", len(defs))
	}

	def := defs[0]
	if def.Adapter.Type != "command" {
		t.Fatalf("adapter.type = %q, want command", def.Adapter.Type)
	}
	if def.Adapter.ExecutablePath != "cli-anything-mermaid" {
		t.Fatalf("adapter.executable_path = %q, want cli-anything-mermaid", def.Adapter.ExecutablePath)
	}
	if def.Adapter.Command != "diagram create" {
		t.Fatalf("adapter.command = %q, want diagram create", def.Adapter.Command)
	}
	if def.Adapter.JSONFlag != "--json" {
		t.Fatalf("adapter.json_flag = %q, want --json", def.Adapter.JSONFlag)
	}
	if def.Adapter.Timeout != 30*time.Second {
		t.Fatalf("adapter.timeout = %s, want 30s", def.Adapter.Timeout)
	}
	if got := def.Adapter.EnvInject["MERMAID_ENV"]; got != "dev" {
		t.Fatalf("adapter.env_inject[MERMAID_ENV] = %q, want dev", got)
	}
}

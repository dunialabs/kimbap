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
	if def.Adapter.ExecutablePath != "mermaid" {
		t.Fatalf("adapter.executable_path = %q, want mermaid", def.Adapter.ExecutablePath)
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
	if len(def.Adapter.SuccessCodes) != 0 {
		t.Fatalf("adapter.success_codes = %v, want empty", def.Adapter.SuccessCodes)
	}
}

func TestToActionDefinitions_CommandAdapterActionOverrides(t *testing.T) {
	manifest := validCommandManifest()
	manifest.CommandSpec.SuccessCodes = []int{0, 2}
	action := manifest.Actions["create_diagram"]
	action.JSONFlag = "--output json"
	action.SuccessCodes = []int{0, 1}
	manifest.Actions["create_diagram"] = action

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error = %v", err)
	}
	def := defs[0]
	if def.Adapter.JSONFlag != "--output json" {
		t.Fatalf("adapter.json_flag = %q, want --output json", def.Adapter.JSONFlag)
	}
	if len(def.Adapter.SuccessCodes) != 2 || def.Adapter.SuccessCodes[0] != 0 || def.Adapter.SuccessCodes[1] != 1 {
		t.Fatalf("adapter.success_codes = %v, want [0 1]", def.Adapter.SuccessCodes)
	}
}

func TestToActionDefinitions_HTTPInputSchemaStrictWithPaginationControl(t *testing.T) {
	manifest := validHTTPManifest()
	action := manifest.Actions["get_item"]
	action.Pagination = &PageSpec{Type: "cursor", NextPath: "next_cursor"}
	manifest.Actions["get_item"] = action

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error = %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("len(defs) = %d, want 1", len(defs))
	}
	schema := defs[0].InputSchema
	if schema == nil {
		t.Fatal("expected input schema")
	}
	if schema.AdditionalProperties {
		t.Fatal("expected strict schema with AdditionalProperties=false")
	}
	if _, ok := schema.Properties["item_id"]; !ok {
		t.Fatal("expected item_id to be in input schema properties")
	}
	if _, ok := schema.Properties["_max_pages"]; !ok {
		t.Fatal("expected _max_pages pagination control in input schema properties")
	}
}

func TestToActionDefinitions_CommandFilterConfig(t *testing.T) {
	manifest := validCommandManifest()
	action := manifest.Actions["create_diagram"]
	action.Response.Filter = &FilterSpec{
		Select:   map[string]string{"id": "id", "name": "name"},
		MaxItems: 10,
	}
	manifest.Actions["create_diagram"] = action

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error = %v", err)
	}
	def := defs[0]

	if def.FilterConfig == nil {
		t.Fatalf("FilterConfig should not be nil when FilterSpec is set")
	}
	if def.FilterConfig.MaxItems != 10 {
		t.Errorf("FilterConfig.MaxItems = %d, want 10", def.FilterConfig.MaxItems)
	}
	if def.InputSchema == nil || def.InputSchema.Properties["_output_mode"] == nil {
		t.Error("_output_mode should be injected in InputSchema when FilterConfig is set")
	}
}

func TestToActionDefinitions_CommandNoFilterConfig(t *testing.T) {
	manifest := validCommandManifest()
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error = %v", err)
	}
	def := defs[0]
	if def.FilterConfig != nil {
		t.Error("FilterConfig should be nil when no FilterSpec set")
	}
	if def.InputSchema != nil && def.InputSchema.Properties["_output_mode"] != nil {
		t.Error("_output_mode should not be present when no FilterConfig")
	}
}

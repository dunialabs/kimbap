package services

import "testing"

func TestToHTTPDefinitions_FilterConfig(t *testing.T) {
	manifest := validHTTPManifest()
	action := manifest.Actions["get_item"]
	action.Response.Filter = &FilterSpec{
		Select:    map[string]string{"id": "id", "title": "title"},
		MaxItems:  25,
		DropNulls: true,
	}
	manifest.Actions["get_item"] = action

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error = %v", err)
	}
	def := defs[0]

	if def.FilterConfig == nil {
		t.Fatal("FilterConfig should not be nil when FilterSpec is set")
	}
	if def.FilterConfig.MaxItems != 25 {
		t.Errorf("FilterConfig.MaxItems = %d, want 25", def.FilterConfig.MaxItems)
	}
	if def.FilterConfig.DropNulls != true {
		t.Error("FilterConfig.DropNulls should be true")
	}
	if def.InputSchema == nil || def.InputSchema.Properties["_output_mode"] == nil {
		t.Error("_output_mode should be injected in InputSchema when FilterConfig is set")
	}
	mode := def.InputSchema.Properties["_output_mode"]
	if mode.Type != "string" {
		t.Errorf("_output_mode type = %q, want string", mode.Type)
	}
	if len(mode.Enum) != 2 {
		t.Errorf("_output_mode Enum len = %d, want 2", len(mode.Enum))
	}
}

func TestToHTTPDefinitions_NoFilterConfig(t *testing.T) {
	manifest := validHTTPManifest()
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error = %v", err)
	}
	def := defs[0]
	if def.FilterConfig != nil {
		t.Error("FilterConfig should be nil when no FilterSpec set")
	}
}

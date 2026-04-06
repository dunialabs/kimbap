package services

import "testing"

func TestToActionDefinitionsAppleScriptDualPrefersInlineScript(t *testing.T) {
	withAppleScriptRegistryMode(t, "dual")
	manifest := validAppleScriptManifest()
	a := manifest.Actions["list_notes"]
	a.Command = "list-notes"
	a.InlineScript = &InlineScript{
		ID:          "notes.list_notes.inline",
		Language:    "jxa",
		Timeout:     "12s",
		Source:      `ObjC.import('stdlib'); JSON.stringify({source:"inline"});`,
		ApprovalRef: "approval.default",
		AuditRef:    "audit.default",
	}
	manifest.Actions["list_notes"] = a

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 action definition, got %d", len(defs))
	}
	def := defs[0]
	if def.Adapter.Command != "notes.list_notes.inline" {
		t.Fatalf("adapter.command = %q, want notes.list_notes.inline", def.Adapter.Command)
	}
	if def.Adapter.ScriptSource == "" {
		t.Fatal("expected adapter.script_source to be populated from inline_script")
	}
	if def.Adapter.ScriptLanguage != "jxa" {
		t.Fatalf("adapter.script_language = %q, want jxa", def.Adapter.ScriptLanguage)
	}
	if def.Adapter.Timeout.String() != "12s" {
		t.Fatalf("adapter.timeout = %s, want 12s", def.Adapter.Timeout)
	}
	if def.Adapter.ApprovalRef != "approval.default" {
		t.Fatalf("adapter.approval_ref = %q, want approval.default", def.Adapter.ApprovalRef)
	}
	if def.Adapter.AuditRef != "audit.default" {
		t.Fatalf("adapter.audit_ref = %q, want audit.default", def.Adapter.AuditRef)
	}
	if def.Adapter.RegistryMode != "dual" {
		t.Fatalf("adapter.registry_mode = %q, want dual", def.Adapter.RegistryMode)
	}
}

func TestToAppleScriptDefinitions_FilterConfig(t *testing.T) {
	withAppleScriptRegistryMode(t, "legacy")
	manifest := validAppleScriptManifest()
	action := manifest.Actions["list_notes"]
	action.Response.Filter = &FilterSpec{
		Select:   map[string]string{"id": "id", "name": "name"},
		MaxItems: 20,
	}
	manifest.Actions["list_notes"] = action

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error = %v", err)
	}
	def := defs[0]

	if def.FilterConfig == nil {
		t.Fatal("FilterConfig should not be nil when FilterSpec is set")
	}
	if def.FilterConfig.MaxItems != 20 {
		t.Errorf("FilterConfig.MaxItems = %d, want 20", def.FilterConfig.MaxItems)
	}
	if def.InputSchema == nil || def.InputSchema.Properties["_output_mode"] == nil {
		t.Error("_output_mode should be injected when FilterConfig is set")
	}
}

func TestToActionDefinitionsAppleScriptLegacyUsesCommand(t *testing.T) {
	withAppleScriptRegistryMode(t, "legacy")
	manifest := validAppleScriptManifest()
	a := manifest.Actions["list_notes"]
	a.InlineScript = &InlineScript{
		ID:          "notes.list_notes.inline",
		Language:    "jxa",
		Source:      `ObjC.import('stdlib'); JSON.stringify({source:"inline"});`,
		ApprovalRef: "approval.default",
		AuditRef:    "audit.default",
	}
	manifest.Actions["list_notes"] = a

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 action definition, got %d", len(defs))
	}
	def := defs[0]
	if def.Adapter.Command != "list-notes" {
		t.Fatalf("adapter.command = %q, want list-notes", def.Adapter.Command)
	}
	if def.Adapter.ScriptSource != "" {
		t.Fatalf("adapter.script_source = %q, want empty in legacy mode", def.Adapter.ScriptSource)
	}
}

func TestToActionDefinitionsAppleScriptManifestModeRequiresInlineScript(t *testing.T) {
	withAppleScriptRegistryMode(t, "manifest")
	manifest := validAppleScriptManifest()

	_, err := ToActionDefinitions(manifest)
	if err == nil {
		t.Fatal("expected manifest mode to reject applescript action without inline_script")
	}
}

func TestToActionDefinitionsAppleScriptDualRejectsMalformedInlineScript(t *testing.T) {
	withAppleScriptRegistryMode(t, "dual")
	manifest := validAppleScriptManifest()
	a := manifest.Actions["list_notes"]
	a.InlineScript = &InlineScript{
		ID:          "notes.list_notes.inline",
		Language:    "jxa",
		Source:      "",
		ApprovalRef: "approval.default",
		AuditRef:    "audit.default",
	}
	manifest.Actions["list_notes"] = a

	_, err := ToActionDefinitions(manifest)
	if err == nil {
		t.Fatal("expected malformed inline_script to fail conversion in dual mode")
	}
}

func TestToActionDefinitionsAppleScriptManifestModeUsesInlineEvenWhenLegacyCommandPresent(t *testing.T) {
	withAppleScriptRegistryMode(t, "manifest")
	manifest := validAppleScriptManifest()
	a := manifest.Actions["list_notes"]
	a.Command = "list-notes"
	a.InlineScript = &InlineScript{
		ID:          "notes.list_notes.inline",
		Language:    "jxa",
		Source:      `ObjC.import('stdlib'); JSON.stringify({source:"inline"});`,
		ApprovalRef: "approval.default",
		AuditRef:    "audit.default",
	}
	manifest.Actions["list_notes"] = a

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions() error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 action definition, got %d", len(defs))
	}
	def := defs[0]
	if def.Adapter.Command != "notes.list_notes.inline" {
		t.Fatalf("adapter.command = %q, want inline id in manifest mode", def.Adapter.Command)
	}
	if def.Adapter.ScriptSource == "" {
		t.Fatal("expected adapter.script_source to be populated in manifest mode")
	}
	if def.Adapter.ApprovalRef != "approval.default" {
		t.Fatalf("adapter.approval_ref = %q, want approval.default", def.Adapter.ApprovalRef)
	}
	if def.Adapter.AuditRef != "audit.default" {
		t.Fatalf("adapter.audit_ref = %q, want audit.default", def.Adapter.AuditRef)
	}
	if def.Adapter.RegistryMode != "manifest" {
		t.Fatalf("adapter.registry_mode = %q, want manifest", def.Adapter.RegistryMode)
	}
}

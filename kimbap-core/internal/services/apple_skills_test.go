package services

import "testing"

func TestInstallAppleNotesService(t *testing.T) {
	manifest, err := ParseManifestFile("../../skills/official/apple-notes.yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(defs) != 5 {
		t.Errorf("got %d, want 5", len(defs))
	}
	for _, d := range defs {
		if d.Adapter.TargetApp != "Notes" {
			t.Errorf("action %s: target_app = %q, want Notes", d.Name, d.Adapter.TargetApp)
		}
		if d.Adapter.Command == "" {
			t.Errorf("action %s: command is empty", d.Name)
		}
	}
}

func TestInstallAppleCalendarService(t *testing.T) {
	manifest, err := ParseManifestFile("../../skills/official/apple-calendar.yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(defs) != 5 {
		t.Errorf("got %d, want 5", len(defs))
	}
}

func TestInstallAppleRemindersService(t *testing.T) {
	manifest, err := ParseManifestFile("../../skills/official/apple-reminders.yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(defs) != 5 {
		t.Errorf("got %d, want 5", len(defs))
	}
}

func TestInstallAppleMailService(t *testing.T) {
	manifest, err := ParseManifestFile("../../skills/official/apple-mail.yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(defs) != 5 {
		t.Errorf("got %d, want 5", len(defs))
	}
	for _, d := range defs {
		if d.Adapter.Command == "send-message" && d.Risk != "admin" && d.Risk != "destructive" {
			t.Errorf("send-message risk = %q, want admin or destructive", d.Risk)
		}
	}
}

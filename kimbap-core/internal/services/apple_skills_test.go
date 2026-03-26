package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllOfficialServiceYAMLsParseAndConvert(t *testing.T) {
	dir := "../../skills/official"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read official skills dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no official service YAMLs found — expected at least one")
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		t.Run(name, func(t *testing.T) {
			manifest, err := ParseManifestFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatalf("parse %s: %v", entry.Name(), err)
			}
			if len(manifest.Actions) == 0 {
				t.Errorf("%s: must define at least one action", entry.Name())
			}
			defs, err := ToActionDefinitions(manifest)
			if err != nil {
				t.Fatalf("convert %s: %v", entry.Name(), err)
			}
			if len(defs) != len(manifest.Actions) {
				t.Errorf("%s: got %d defs, want %d", entry.Name(), len(defs), len(manifest.Actions))
			}
		})
	}
}

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

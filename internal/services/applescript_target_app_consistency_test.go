package services

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	adaptercommands "github.com/dunialabs/kimbap/internal/adapters/commands"
)

func TestOfficialAppleScriptCommandsMatchManifestTargetApp(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	catalogDir := filepath.Join(repoRoot, "services", "catalog")

	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		t.Fatalf("read services/catalog: %v", err)
	}

	commands := map[string]adaptercommands.Command{}
	for name, cmd := range adaptercommands.NotesCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.CalendarCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.RemindersCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.MailCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.FinderCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.SafariCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.MessagesCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.ContactsCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.MSOfficeCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.IWorkCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.SpotifyCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}
	for name, cmd := range adaptercommands.ShortcutsCommands() {
		commands[strings.TrimSpace(name)] = cmd
	}

	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}
		path := filepath.Join(catalogDir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", entry.Name(), readErr)
		}
		manifest, parseErr := ParseManifest(data)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", entry.Name(), parseErr)
		}
		if normalizedAdapterType(manifest.Adapter) != "applescript" {
			continue
		}

		expectedApp := strings.TrimSpace(manifest.TargetApp)
		for actionKey, action := range manifest.Actions {
			cmdName := strings.TrimSpace(action.Command)
			if cmdName == "" {
				if action.InlineScript == nil || strings.TrimSpace(action.InlineScript.Source) == "" {
					t.Fatalf("%s action %q must define either command or inline_script.source", entry.Name(), actionKey)
				}
				checked++
				continue
			}
			cmd, exists := commands[cmdName]
			if !exists {
				t.Fatalf("%s action %q references unknown applescript command %q", entry.Name(), actionKey, action.Command)
			}
			if strings.TrimSpace(cmd.TargetApp) != expectedApp {
				t.Fatalf("%s action %q command %q target app mismatch: manifest=%q command=%q", entry.Name(), actionKey, action.Command, expectedApp, strings.TrimSpace(cmd.TargetApp))
			}
			checked++
		}
	}

	if checked == 0 {
		t.Fatal("expected to check at least one applescript action")
	}
}

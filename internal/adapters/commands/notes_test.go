package commands

import (
	"strings"
	"testing"
)

func TestNotesCommandsCount(t *testing.T) {
	cmds := NotesCommands()
	if len(cmds) != 5 {
		t.Errorf("got %d commands, want 5", len(cmds))
	}
}

func TestNotesCommandNames(t *testing.T) {
	cmds := NotesCommands()
	expected := []string{"list-folders", "list-notes", "get-note", "search-notes", "create-note"}
	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestNotesScriptsReadFromStdin(t *testing.T) {
	cmds := NotesCommands()
	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, "NSFileHandle.fileHandleWithStandardInput") {
			t.Errorf("%s: script does not read from stdin", name)
		}
	}
}

func TestNotesNoForbiddenPatterns(t *testing.T) {
	forbidden := []string{
		"do shell script",
		"$.NSTask",
		"ObjC.import('AppKit')",
		"Application('System Events')",
		"$.NSURLSession",
	}
	cmds := NotesCommands()
	for name, cmd := range cmds {
		for _, pattern := range forbidden {
			if strings.Contains(cmd.Script, pattern) {
				t.Errorf("%s: contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

func TestNotesScriptsOutputJSON(t *testing.T) {
	cmds := NotesCommands()
	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, "JSON.stringify") {
			t.Errorf("%s: script does not output JSON", name)
		}
	}
}

func TestNotesTargetApp(t *testing.T) {
	cmds := NotesCommands()
	for name, cmd := range cmds {
		if cmd.TargetApp != "Notes" {
			t.Errorf("%s: TargetApp = %q, want Notes", name, cmd.TargetApp)
		}
	}
}

func TestNotesNotFoundCommandsEmitSentinel(t *testing.T) {
	notFoundCmds := []string{"get-note"}
	cmds := NotesCommands()
	for _, name := range notFoundCmds {
		cmd, ok := cmds[name]
		if !ok {
			t.Fatalf("command %q not found", name)
		}
		if !strings.Contains(cmd.Script, "[NOT_FOUND]") {
			t.Errorf("%s: script does not emit [NOT_FOUND] sentinel for not-found case", name)
		}
	}
}

func TestNotesLimitPatterns(t *testing.T) {
	cmds := NotesCommands()

	listNotes := cmds["list-notes"]
	if !strings.Contains(listNotes.Script, "parseInt(input.limit, 10)") {
		t.Error("list-notes: script does not parse input.limit")
	}
	if !strings.Contains(listNotes.Script, "? 5 : parsedLimit") {
		t.Error("list-notes: script should default limit to 5")
	}
	if !strings.Contains(listNotes.Script, "app.notes.length") {
		t.Error("list-notes: script does not use app.notes.length for index-based access")
	}
	if !strings.Contains(listNotes.Script, "app.notes[i]") {
		t.Error("list-notes: script does not use app.notes[i] for index-based access")
	}
	if strings.Contains(listNotes.Script, "notes = app.notes();") {
		t.Error("list-notes: script still materializes all notes with app.notes()")
	}
	if strings.Contains(listNotes.Script, "folders[0].notes()") {
		t.Error("list-notes: script still materializes folder notes with folders[0].notes()")
	}
	if strings.Contains(listNotes.Script, ".notes().slice") {
		t.Error("list-notes: script still slices a materialized folder notes array")
	}
	if strings.Contains(listNotes.Script, "snippet:") {
		t.Error("list-notes: script should not fetch plaintext snippet in list view")
	}

	searchNotes := cmds["search-notes"]
	if !strings.Contains(searchNotes.Script, "parseInt(input.limit, 10)") {
		t.Error("search-notes: script does not parse input.limit")
	}
	if !strings.Contains(searchNotes.Script, "? 5 : parsedLimit") {
		t.Error("search-notes: script should default limit to 5")
	}
	if !strings.Contains(searchNotes.Script, "TIMEOUT_MS") || !strings.Contains(searchNotes.Script, "[TIMEOUT] search-notes timed out scanning Notes") {
		t.Error("search-notes: script should fail with a bounded timeout message instead of being killed")
	}
	if !strings.Contains(searchNotes.Script, "app.notes.length") {
		t.Error("search-notes: script does not use app.notes.length for index-based access")
	}
	if !strings.Contains(searchNotes.Script, "app.notes[i]") {
		t.Error("search-notes: script does not use app.notes[i] for index-based access")
	}
	if strings.Contains(searchNotes.Script, "app.notes()") {
		t.Error("search-notes: script still materializes all notes with app.notes()")
	}
	if !strings.Contains(searchNotes.Script, "if (noteName.toLowerCase().indexOf(query) >= 0)") {
		t.Error("search-notes: script should short-circuit title matches before fetching plaintext")
	}
	if !strings.Contains(listNotes.Script, "Application(\"com.apple.Notes\")") {
		t.Error("list-notes: script should target Notes by bundle identifier")
	}
	if !strings.Contains(searchNotes.Script, "Application(\"com.apple.Notes\")") {
		t.Error("search-notes: script should target Notes by bundle identifier")
	}
}

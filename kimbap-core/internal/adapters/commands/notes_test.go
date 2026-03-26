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

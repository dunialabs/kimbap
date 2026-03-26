package commands

import (
	"strings"
	"testing"
)

func TestContactsCommandsCount(t *testing.T) {
	cmds := ContactsCommands()
	if len(cmds) != 4 {
		t.Errorf("got %d commands, want 4", len(cmds))
	}
}

func TestContactsCommandNames(t *testing.T) {
	cmds := ContactsCommands()
	expected := []string{"contacts-list", "contacts-search", "contacts-get", "contacts-create"}
	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestContactsScriptsReadFromStdin(t *testing.T) {
	cmds := ContactsCommands()
	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Errorf("%s: script does not include stdinReader", name)
		}
	}
}

func TestContactsNoForbiddenPatterns(t *testing.T) {
	forbidden := []string{"do shell script", "$.NSTask", "${"}
	cmds := ContactsCommands()
	for name, cmd := range cmds {
		for _, pattern := range forbidden {
			if strings.Contains(cmd.Script, pattern) {
				t.Errorf("%s: contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

func TestContactsTargetApp(t *testing.T) {
	cmds := ContactsCommands()
	for name, cmd := range cmds {
		if cmd.TargetApp != "Contacts" {
			t.Errorf("%s: TargetApp = %q, want Contacts", name, cmd.TargetApp)
		}
	}
}

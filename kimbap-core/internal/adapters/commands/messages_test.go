package commands

import (
	"strings"
	"testing"
)

func TestMessagesCommandsCount(t *testing.T) {
	cmds := MessagesCommands()
	if len(cmds) != 2 {
		t.Errorf("got %d commands, want 2", len(cmds))
	}
}

func TestMessagesCommandNames(t *testing.T) {
	cmds := MessagesCommands()
	expected := []string{"messages-send", "messages-list-chats"}
	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestMessagesScriptsReadFromStdin(t *testing.T) {
	cmds := MessagesCommands()
	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Errorf("%s: script does not include stdinReader", name)
		}
	}
}

func TestMessagesNoForbiddenPatterns(t *testing.T) {
	forbidden := []string{"do shell script", "$.NSTask", "${"}
	cmds := MessagesCommands()
	for name, cmd := range cmds {
		for _, pattern := range forbidden {
			if strings.Contains(cmd.Script, pattern) {
				t.Errorf("%s: contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

func TestMessagesTargetApp(t *testing.T) {
	cmds := MessagesCommands()
	for name, cmd := range cmds {
		if cmd.TargetApp != "Messages" {
			t.Errorf("%s: TargetApp = %q, want Messages", name, cmd.TargetApp)
		}
	}
}

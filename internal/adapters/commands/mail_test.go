package commands

import (
	"slices"
	"strings"
	"testing"
)

func TestMailCommandsCount(t *testing.T) {
	commands := MailCommands()
	if len(commands) != 5 {
		t.Fatalf("expected 5 commands, got %d", len(commands))
	}
}

func TestMailCommandNames(t *testing.T) {
	commands := MailCommands()
	expected := []string{
		"list-mailboxes",
		"list-messages",
		"get-message",
		"send-message",
		"search-messages",
	}

	for _, name := range expected {
		cmd, ok := commands[name]
		if !ok {
			t.Fatalf("missing command %q", name)
		}
		if cmd.Name != name {
			t.Fatalf("command %q has mismatched Name %q", name, cmd.Name)
		}
	}

	for name := range commands {
		if !slices.Contains(expected, name) {
			t.Fatalf("unexpected command %q", name)
		}
	}
}

func TestMailScriptsReadFromStdin(t *testing.T) {
	commands := MailCommands()
	for name, cmd := range commands {
		if !strings.HasPrefix(cmd.Script, stdinReader) {
			t.Fatalf("command %q script does not start with stdinReader", name)
		}
		if !strings.Contains(cmd.Script, "JSON.stringify") {
			t.Fatalf("command %q script does not JSON.stringify output", name)
		}
	}
}

func TestMailNoForbiddenPatterns(t *testing.T) {
	commands := MailCommands()
	forbidden := []string{"do shell script", "$.NSTask", "${"}

	for name, cmd := range commands {
		for _, pattern := range forbidden {
			if strings.Contains(cmd.Script, pattern) {
				t.Fatalf("command %q contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

func TestMailTargetApp(t *testing.T) {
	commands := MailCommands()
	for name, cmd := range commands {
		if cmd.TargetApp != "Mail" {
			t.Fatalf("command %q has TargetApp %q", name, cmd.TargetApp)
		}
	}
}

func TestGetMessageIncludesMailboxField(t *testing.T) {
	cmd, ok := MailCommands()["get-message"]
	if !ok {
		t.Fatal("get-message command not found")
	}
	if !strings.Contains(cmd.Script, "foundMailbox") {
		t.Fatal("get-message script does not capture foundMailbox for the result")
	}
	if !strings.Contains(cmd.Script, "mailbox: foundMailbox") {
		t.Fatal("get-message result object does not include mailbox field")
	}
}

func TestMailNotFoundCommandsEmitSentinel(t *testing.T) {
	notFoundCmds := []string{"get-message"}
	cmds := MailCommands()
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

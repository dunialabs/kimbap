package commands

import (
	"strings"
	"testing"
)

func TestRemindersCommandsCount(t *testing.T) {
	cmds := RemindersCommands()
	if len(cmds) != 5 {
		t.Fatalf("expected 5 commands, got %d", len(cmds))
	}
}

func TestRemindersCommandNames(t *testing.T) {
	cmds := RemindersCommands()
	expected := []string{
		"list-lists",
		"list-reminders",
		"get-reminder",
		"create-reminder",
		"complete-reminder",
	}

	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Fatalf("missing command %q", name)
		}
		if cmds[name].Name != name {
			t.Fatalf("command %q has mismatched Name field %q", name, cmds[name].Name)
		}
	}
}

func TestRemindersScriptsReadFromStdin(t *testing.T) {
	for name, cmd := range RemindersCommands() {
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Fatalf("command %q does not include stdinReader", name)
		}
	}
}

func TestRemindersNoForbiddenPatterns(t *testing.T) {
	forbidden := []string{
		"do shell script",
		"$.NSTask",
		"${",
		"includeStandardAdditions = true",
	}

	for name, cmd := range RemindersCommands() {
		for _, pattern := range forbidden {
			if strings.Contains(cmd.Script, pattern) {
				t.Fatalf("command %q contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

func TestRemindersTargetApp(t *testing.T) {
	for name, cmd := range RemindersCommands() {
		if cmd.TargetApp != "Reminders" {
			t.Fatalf("command %q has target app %q, want %q", name, cmd.TargetApp, "Reminders")
		}
	}
}

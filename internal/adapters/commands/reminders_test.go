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
		if !strings.Contains(cmd.Script, "Application(\"com.apple.reminders\")") {
			t.Fatalf("command %q should target Reminders by bundle identifier", name)
		}
	}
}

func TestRemindersListRemindersFailsClosedWhileUnsupported(t *testing.T) {
	cmd, ok := RemindersCommands()["list-reminders"]
	if !ok {
		t.Fatal("list-reminders not found")
	}
	if !strings.Contains(cmd.Script, "[NOT_SUPPORTED] list-reminders is temporarily disabled because Reminders enumeration can hang in JXA") {
		t.Fatal("list-reminders should fail closed with an explicit NOT_SUPPORTED error")
	}
}

func TestRemindersNotFoundCommandsEmitSentinel(t *testing.T) {
	notFoundCmds := []string{"get-reminder", "complete-reminder"}
	cmds := RemindersCommands()
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

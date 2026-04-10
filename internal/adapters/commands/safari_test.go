package commands

import (
	"strings"
	"testing"
)

func TestSafariCommandsCount(t *testing.T) {
	cmds := SafariCommands()
	if len(cmds) != 5 {
		t.Errorf("got %d commands, want 5", len(cmds))
	}
}

func TestSafariCommandNames(t *testing.T) {
	cmds := SafariCommands()
	expected := []string{
		"safari-get-url", "safari-open-url", "safari-list-tabs",
		"safari-close-tab", "safari-get-source",
	}
	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestSafariScriptsReadFromStdin(t *testing.T) {
	cmds := SafariCommands()
	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Errorf("%s: script does not include stdinReader", name)
		}
	}
}

func TestSafariNoForbiddenPatterns(t *testing.T) {
	forbidden := []string{"do shell script", "$.NSTask", "${"}
	cmds := SafariCommands()
	for name, cmd := range cmds {
		for _, pattern := range forbidden {
			if strings.Contains(cmd.Script, pattern) {
				t.Errorf("%s: contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

func TestSafariTargetApp(t *testing.T) {
	cmds := SafariCommands()
	for name, cmd := range cmds {
		if cmd.TargetApp != "Safari" {
			t.Errorf("%s: TargetApp = %q, want Safari", name, cmd.TargetApp)
		}
		if !strings.Contains(cmd.Script, "Application(\"com.apple.Safari\")") {
			t.Errorf("%s: script should target Safari by bundle identifier", name)
		}
	}
}

package commands

import (
	"strings"
	"testing"
)

func TestShortcutsCommandsBundleIdentifierTargeting(t *testing.T) {
	cmds := ShortcutsCommands()
	if len(cmds) != 3 {
		t.Fatalf("got %d commands, want 3", len(cmds))
	}
	for name, cmd := range cmds {
		if cmd.TargetApp != "Shortcuts" {
			t.Fatalf("%s: TargetApp = %q, want Shortcuts", name, cmd.TargetApp)
		}
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Fatalf("%s: script does not include stdinReader", name)
		}
		if !strings.Contains(cmd.Script, "Application(\"com.apple.shortcuts\")") {
			t.Fatalf("%s: script should target Shortcuts by bundle identifier", name)
		}
	}
}

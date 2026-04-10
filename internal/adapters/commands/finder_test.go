package commands

import (
	"strings"
	"testing"
)

func TestFinderCommandsCount(t *testing.T) {
	cmds := FinderCommands()
	if len(cmds) != 7 {
		t.Errorf("got %d commands, want 7", len(cmds))
	}
}

func TestFinderCommandNames(t *testing.T) {
	cmds := FinderCommands()
	expected := []string{
		"finder-list-items", "finder-get-info", "finder-create-folder",
		"finder-move-item", "finder-copy-item", "finder-delete-item", "finder-open-item",
	}
	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestFinderScriptsReadFromStdin(t *testing.T) {
	cmds := FinderCommands()
	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Errorf("%s: script does not include stdinReader", name)
		}
	}
}

func TestFinderNoForbiddenPatterns(t *testing.T) {
	forbidden := []string{"$.NSTask", "${"}
	cmds := FinderCommands()
	for name, cmd := range cmds {
		for _, pattern := range forbidden {
			if strings.Contains(cmd.Script, pattern) {
				t.Errorf("%s: contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

func TestFinderTargetApp(t *testing.T) {
	cmds := FinderCommands()
	for name, cmd := range cmds {
		if cmd.TargetApp != "Finder" {
			t.Errorf("%s: TargetApp = %q, want Finder", name, cmd.TargetApp)
		}
		if !strings.Contains(cmd.Script, "Application(\"com.apple.finder\")") {
			t.Errorf("%s: script should target Finder by bundle identifier", name)
		}
	}
}

func TestFinderNotFoundMessagesDoNotEchoInputValues(t *testing.T) {
	cmds := FinderCommands()
	for name, cmd := range cmds {
		if strings.Contains(cmd.Script, "[NOT_FOUND] folder not found: ") ||
			strings.Contains(cmd.Script, "[NOT_FOUND] item not found: ") ||
			strings.Contains(cmd.Script, "[NOT_FOUND] source item not found: ") ||
			strings.Contains(cmd.Script, "[NOT_FOUND] destination folder not found: ") {
			t.Errorf("%s: should not include dynamic path values in not-found errors", name)
		}
	}
}

func TestFinderFolderLookupUsesVolumeAwareTraversal(t *testing.T) {
	cmds := FinderCommands()
	lookupScripts := 0

	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, "function pathLocator(path)") {
			continue
		}
		lookupScripts++

		if strings.Contains(cmd.Script, "app.folders.whose({url: {_equals: prefixes[i]}})") {
			t.Errorf("%s: folder lookup should not use top-level URL matching", name)
		}
		if !strings.Contains(cmd.Script, `parts[0] === "Volumes"`) {
			t.Errorf("%s: folder lookup should special-case /Volumes paths", name)
		}
		if !strings.Contains(cmd.Script, "app.disks.byName(parts[1])") {
			t.Errorf("%s: folder lookup should resolve mounted volumes by disk name", name)
		}
		if !strings.Contains(cmd.Script, "app.startupDisk") {
			t.Errorf("%s: folder lookup should still support startup disk paths", name)
		}
	}

	if lookupScripts != len(cmds) {
		t.Errorf("got %d scripts with pathLocator, want %d", lookupScripts, len(cmds))
	}
}

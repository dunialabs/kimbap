package commands

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestCalendarCommandsCount(t *testing.T) {
	cmds := CalendarCommands()
	if len(cmds) != 5 {
		t.Fatalf("expected 5 calendar commands, got %d", len(cmds))
	}
}

func TestCalendarCommandNames(t *testing.T) {
	cmds := CalendarCommands()
	got := make([]string, 0, len(cmds))
	for name := range cmds {
		got = append(got, name)
	}
	sort.Strings(got)

	want := []string{
		"create-event",
		"get-event",
		"list-calendars",
		"list-events",
		"search-events",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected calendar command names: got %v want %v", got, want)
	}
}

func TestCalendarScriptsReadFromStdin(t *testing.T) {
	cmds := CalendarCommands()
	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Fatalf("command %q does not include stdinReader preamble", name)
		}
	}
}

func TestCalendarNoForbiddenPatterns(t *testing.T) {
	cmds := CalendarCommands()
	forbidden := []string{
		"do shell script",
		"$.NSTask",
		"${",
	}

	for name, cmd := range cmds {
		for _, pattern := range forbidden {
			if strings.Contains(cmd.Script, pattern) {
				t.Fatalf("command %q contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

func TestCalendarTargetApp(t *testing.T) {
	cmds := CalendarCommands()
	for name, cmd := range cmds {
		if cmd.TargetApp != "Calendar" {
			t.Fatalf("command %q target app = %q, want %q", name, cmd.TargetApp, "Calendar")
		}
	}
}

func TestCalendarNotFoundCommandsEmitSentinel(t *testing.T) {
	notFoundCmds := []string{"get-event", "create-event"}
	cmds := CalendarCommands()
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

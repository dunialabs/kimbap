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
		if !strings.Contains(cmd.Script, "Application(\"com.apple.iCal\")") {
			t.Fatalf("command %q should target Calendar by bundle identifier", name)
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

func TestCalendarListEventsHasDefaultLimit(t *testing.T) {
	cmd := CalendarCommands()["list-events"]
	if !strings.Contains(cmd.Script, "var limit = (isNaN(parsedLimit) || parsedLimit <= 0) ? 100 : parsedLimit;") {
		t.Fatal("list-events should cap returned events by default")
	}
	if !strings.Contains(cmd.Script, "if (result.length >= limit)") {
		t.Fatal("list-events should stop once the limit is reached")
	}
	if !strings.Contains(cmd.Script, "[NOT_SUPPORTED] list-events across multiple calendars is too slow; specify --calendar") {
		t.Fatal("list-events should fail fast when multiple calendars would trigger a slow global scan")
	}
}

func TestCalendarGetEventHasTimeoutGuard(t *testing.T) {
	cmd := CalendarCommands()["get-event"]
	if !strings.Contains(cmd.Script, "[TIMEOUT]") {
		t.Fatal("get-event should emit [TIMEOUT] sentinel when scanning takes too long")
	}
	if !strings.Contains(cmd.Script, "TIMEOUT_MS") {
		t.Fatal("get-event should define a TIMEOUT_MS threshold")
	}
	if !strings.Contains(cmd.Script, "specify --calendar") {
		t.Fatal("get-event timeout message should hint --calendar")
	}
}

func TestCalendarSearchEventsHasTimeoutGuard(t *testing.T) {
	cmd := CalendarCommands()["search-events"]
	if !strings.Contains(cmd.Script, "[TIMEOUT]") {
		t.Fatal("search-events should emit [TIMEOUT] sentinel when scanning takes too long")
	}
	if !strings.Contains(cmd.Script, "TIMEOUT_MS") {
		t.Fatal("search-events should define a TIMEOUT_MS threshold")
	}
	if !strings.Contains(cmd.Script, "specify --calendar") {
		t.Fatal("search-events timeout message should hint --calendar")
	}
}

func TestCalendarGetEventSupportsCalendarScoping(t *testing.T) {
	cmd := CalendarCommands()["get-event"]
	if !strings.Contains(cmd.Script, "input.calendar") {
		t.Fatal("get-event should support optional --calendar parameter to scope search")
	}
	if !strings.Contains(cmd.Script, `app.calendars.whose({name: input.calendar})`) {
		t.Fatal("get-event should filter calendars by name when --calendar is provided")
	}
}

func TestCalendarSearchEventsSupportsCalendarScoping(t *testing.T) {
	cmd := CalendarCommands()["search-events"]
	if !strings.Contains(cmd.Script, "input.calendar") {
		t.Fatal("search-events should support optional --calendar parameter to scope search")
	}
	if !strings.Contains(cmd.Script, `app.calendars.whose({name: input.calendar})`) {
		t.Fatal("search-events should filter calendars by name when --calendar is provided")
	}
}

func TestCalendarCreateEventUsesEventClass(t *testing.T) {
	cmd := CalendarCommands()["create-event"]
	if strings.Contains(cmd.Script, "app.CalendarEvent(") {
		t.Fatal("create-event must not use app.CalendarEvent")
	}
	if !strings.Contains(cmd.Script, "var event = app.Event(") {
		t.Fatal("create-event should construct events with app.Event")
	}
}

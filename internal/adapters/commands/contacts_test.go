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
		if !strings.Contains(cmd.Script, "Application(\"com.apple.AddressBook\")") {
			t.Errorf("%s: script should target Contacts by bundle identifier", name)
		}
	}
}

func TestContactsGetUsesSanitizedNotFoundMessage(t *testing.T) {
	cmd := ContactsCommands()["contacts-get"]
	if !strings.Contains(cmd.Script, "throw new Error(\"[NOT_FOUND] contact not found\");") {
		t.Error("contacts-get should use sanitized not-found sentinel")
	}
	if strings.Contains(cmd.Script, "contact not found: \" + input.name") {
		t.Error("contacts-get should not echo input.name in not-found error")
	}
}

func TestContactsCreateSaveErrorIsSanitized(t *testing.T) {
	cmd := ContactsCommands()["contacts-create"]
	if !strings.Contains(cmd.Script, "throw new Error(\"Failed to save contact\");") {
		t.Error("contacts-create should use sanitized save error message")
	}
	if strings.Contains(cmd.Script, "e.message") {
		t.Error("contacts-create should not expose internal e.message details")
	}
}

func TestContactsNoFullMaterialization(t *testing.T) {
	cmds := ContactsCommands()

	list := cmds["contacts-list"]
	if strings.Contains(list.Script, "app.people()") {
		t.Error("contacts-list: script materializes all contacts with app.people()")
	}
	if !strings.Contains(list.Script, "app.people.length") {
		t.Error("contacts-list: script should use app.people.length for index-based access")
	}
	if !strings.Contains(list.Script, "app.people[i]") {
		t.Error("contacts-list: script should use app.people[i] for index-based access")
	}

	search := cmds["contacts-search"]
	if strings.Contains(search.Script, "app.people()") {
		t.Error("contacts-search: fallback still materializes all contacts with app.people()")
	}
	if !strings.Contains(search.Script, "app.people.length") {
		t.Error("contacts-search: fallback should use app.people.length for index-based access")
	}
	if !strings.Contains(search.Script, "app.people[i]") {
		t.Error("contacts-search: fallback should use app.people[i] for index-based access")
	}

	get := cmds["contacts-get"]
	if strings.Contains(get.Script, "app.people()") {
		t.Error("contacts-get: script materializes all contacts with app.people()")
	}
	if !strings.Contains(get.Script, "app.people.length") {
		t.Error("contacts-get: script should use app.people.length for index-based access")
	}
	if !strings.Contains(get.Script, "app.people[i]") {
		t.Error("contacts-get: script should use app.people[i] for index-based access")
	}
}

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

func TestMessagesSendNormalizesHandleAndSanitizesNotFound(t *testing.T) {
	cmd := MessagesCommands()["messages-send"]
	if !strings.Contains(cmd.Script, "function normalizeHandle(value)") {
		t.Fatal("messages-send should define normalizeHandle")
	}
	if !strings.Contains(cmd.Script, "handle = handle.replace(/[\\s\\-()\\.]/g, \"\");") {
		t.Error("messages-send should normalize common phone punctuation")
	}
	if !strings.Contains(cmd.Script, "throw new Error(\"[NOT_FOUND] contact not found\");") {
		t.Error("messages-send should use sanitized not-found sentinel")
	}
	if strings.Contains(cmd.Script, "contact not found:") {
		t.Error("messages-send should not echo handle values in not-found error")
	}
}

func TestMessagesListChatsNoFullMaterialization(t *testing.T) {
	cmd := MessagesCommands()["messages-list-chats"]
	if strings.Contains(cmd.Script, "app.chats()") {
		t.Error("messages-list-chats: script materializes all chats with app.chats()")
	}
	if !strings.Contains(cmd.Script, "app.chats.length") {
		t.Error("messages-list-chats: script should use app.chats.length for index-based access")
	}
	if !strings.Contains(cmd.Script, "app.chats[i]") {
		t.Error("messages-list-chats: script should use app.chats[i] for index-based access")
	}
}

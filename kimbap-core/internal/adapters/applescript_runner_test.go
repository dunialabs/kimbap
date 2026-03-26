package adapters

import (
	"bytes"
	"context"
	"testing"
)

func TestOSAScriptRunnerImplementsInterface(t *testing.T) {
	var _ CommandRunner = (*OSAScriptRunner)(nil)
}

func TestMockRunnerImplementsInterface(t *testing.T) {
	var _ CommandRunner = (*MockRunner)(nil)
}

func TestMockRunnerRecordsCalls(t *testing.T) {
	mock := NewMockRunner([]byte(`{"ok":true}`), nil, nil)
	stdin := bytes.NewReader([]byte(`{"title":"test"}`))
	stdout, stderr, err := mock.Run(context.Background(), "/usr/bin/osascript", []string{"-l", "JavaScript", "-e", "script"}, stdin)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(stdout) != `{"ok":true}` {
		t.Errorf("stdout = %q, want %q", stdout, `{"ok":true}`)
	}
	if len(stderr) != 0 {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	if len(mock.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(mock.Calls))
	}
	call := mock.Calls[0]
	if call.Name != "/usr/bin/osascript" {
		t.Errorf("name = %q, want /usr/bin/osascript", call.Name)
	}
	if string(call.Stdin) != `{"title":"test"}` {
		t.Errorf("stdin = %q, want {\"title\":\"test\"}", call.Stdin)
	}
}

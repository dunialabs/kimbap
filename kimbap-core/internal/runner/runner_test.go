package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunnerSetsProxyEnvVars(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "env.txt")
	r := NewRunner(RunConfig{
		Command:    []string{"sh", "-c", "printf '%s|%s|%s' \"$HTTP_PROXY\" \"$HTTPS_PROXY\" \"$KIMBAP_AGENT_TOKEN\" > \"$OUT\""},
		ProxyAddr:  "127.0.0.1:18080",
		AgentToken: "agent-token-1",
		Env: map[string]string{
			"OUT": outFile,
		},
	})

	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("runner start failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	got := string(data)
	wantProxy := "http://kimbap:agent-token-1@127.0.0.1:18080"
	wantToken := "agent-token-1"
	want := wantProxy + "|" + wantProxy + "|" + wantToken
	if got != want {
		t.Fatalf("unexpected env output:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestBuildEnvEnsuresLoopbackNoProxyEntries(t *testing.T) {
	env := buildEnv([]string{"NO_PROXY=localhost"}, nil, "http://127.0.0.1:18080", "")
	value := ""
	for _, item := range env {
		if strings.HasPrefix(item, "NO_PROXY=") {
			value = strings.TrimPrefix(item, "NO_PROXY=")
			break
		}
	}
	if value == "" {
		t.Fatal("expected NO_PROXY to be set")
	}
	for _, required := range []string{"localhost", "127.0.0.1", "::1"} {
		if !strings.Contains(value, required) {
			t.Fatalf("expected NO_PROXY to include %q, got %q", required, value)
		}
	}
}

func TestRunnerExecutesSimpleCommand(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "echo.txt")
	r := NewRunner(RunConfig{
		Command: []string{"sh", "-c", "echo hello > \"$OUT\""},
		Env: map[string]string{
			"OUT": outFile,
		},
	})

	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("runner start failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "hello" {
		t.Fatalf("unexpected output: %q", string(data))
	}
}

func TestRunnerForwardsExitCode(t *testing.T) {
	r := NewRunner(RunConfig{Command: []string{"sh", "-c", "exit 7"}})
	err := r.Start(context.Background())
	if err == nil {
		t.Fatal("expected non-zero exit error")
	}

	exitErr := &ExitError{}
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T (%v)", err, err)
	}
	if exitErr.Code != 7 {
		t.Fatalf("expected exit code 7, got %d", exitErr.Code)
	}
}

func TestRunnerCancellationKillsSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group signaling differs on windows")
	}

	r := NewRunner(RunConfig{Command: []string{"sh", "-c", "sleep 30"}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Start(ctx)
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runner did not stop after cancellation")
	}
}

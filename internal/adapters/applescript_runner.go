package adapters

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"sync"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, stdin io.Reader) (stdout []byte, stderr []byte, err error)
}

type OSAScriptRunner struct{}

func NewOSAScriptRunner() *OSAScriptRunner {
	return &OSAScriptRunner{}
}

func (r *OSAScriptRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, []byte, error) {
	_ = name
	cmd := exec.CommandContext(ctx, "/usr/bin/osascript", args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdoutBuf, limit: maxCommandOutputBytes}
	cmd.Stderr = &limitedWriter{w: &stderrBuf, limit: maxCommandStderrBytes}
	err := cmd.Run()
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

type MockRunner struct {
	mu         sync.Mutex
	Calls      []MockCall
	StdoutData []byte
	StderrData []byte
	Err        error
}

type MockCall struct {
	Name  string
	Args  []string
	Stdin []byte
}

func NewMockRunner(stdout []byte, stderr []byte, err error) *MockRunner {
	return &MockRunner{
		StdoutData: stdout,
		StderrData: stderr,
		Err:        err,
	}
}

func (m *MockRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, []byte, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	var stdinBytes []byte
	if stdin != nil {
		stdinBytes, _ = io.ReadAll(stdin)
	}
	m.Calls = append(m.Calls, MockCall{Name: name, Args: args, Stdin: stdinBytes})
	return m.StdoutData, m.StderrData, m.Err
}

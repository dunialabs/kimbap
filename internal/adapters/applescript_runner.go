package adapters

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"sync"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, stdin io.Reader) (stdout []byte, stderr []byte, stdoutTruncated bool, stderrTruncated bool, err error)
}

type OSAScriptRunner struct{}

func NewOSAScriptRunner() *OSAScriptRunner {
	return &OSAScriptRunner{}
}

func (r *OSAScriptRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, []byte, bool, bool, error) {
	_ = name
	cmd := exec.CommandContext(ctx, "/usr/bin/osascript", args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutWriter := &limitedWriter{w: &stdoutBuf, limit: maxCommandOutputBytes}
	stderrWriter := &limitedWriter{w: &stderrBuf, limit: maxCommandStderrBytes}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter
	err := cmd.Run()
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), stdoutWriter.truncated, stderrWriter.truncated, err
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

func (m *MockRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, []byte, bool, bool, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	var stdinBytes []byte
	if stdin != nil {
		stdinBytes, _ = io.ReadAll(stdin)
	}
	m.Calls = append(m.Calls, MockCall{Name: name, Args: args, Stdin: stdinBytes})
	return m.StdoutData, m.StderrData, false, false, m.Err
}

package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
)

const defaultCommandAdapterTimeout = 60 * time.Second
const maxCommandOutputBytes int64 = 4 << 20
const maxCommandStderrBytes int64 = 4 << 10

type CommandAdapter struct {
	allowlist map[string]bool
	timeout   time.Duration
}

func NewCommandAdapter(allowlist []string, timeout time.Duration) *CommandAdapter {
	var allowMap map[string]bool
	if allowlist != nil {
		allowMap = make(map[string]bool, len(allowlist))
		for _, executable := range allowlist {
			trimmed := strings.TrimSpace(executable)
			if trimmed != "" {
				allowMap[trimmed] = true
			}
		}
	}
	if timeout <= 0 {
		timeout = defaultCommandAdapterTimeout
	}

	return &CommandAdapter{allowlist: allowMap, timeout: timeout}
}

func (a *CommandAdapter) Type() string {
	return "command"
}

func (a *CommandAdapter) Validate(def actions.ActionDefinition) error {
	if strings.TrimSpace(def.Adapter.ExecutablePath) == "" {
		return fmt.Errorf("command adapter requires executable path")
	}
	if strings.TrimSpace(def.Adapter.Command) == "" {
		return fmt.Errorf("command adapter requires command")
	}
	if a.allowlist != nil && !a.allowlist[strings.TrimSpace(def.Adapter.ExecutablePath)] {
		return fmt.Errorf("executable not allowed: %s", strings.TrimSpace(def.Adapter.ExecutablePath))
	}
	return nil
}

func (a *CommandAdapter) Execute(ctx context.Context, req AdapterRequest) (*AdapterResult, error) {
	start := time.Now()

	executable := strings.TrimSpace(req.Action.Adapter.ExecutablePath)
	command := strings.TrimSpace(req.Action.Adapter.Command)
	if executable == "" || command == "" {
		execErr := actions.NewExecutionError(actions.ErrValidationFailed, "command adapter requires executable and command", http.StatusBadRequest, false, nil)
		return &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}, execErr
	}
	if a.allowlist != nil && !a.allowlist[executable] {
		execErr := actions.NewExecutionError(actions.ErrUnauthorized, fmt.Sprintf("executable not allowed: %s", executable), http.StatusForbidden, false, nil)
		return &AdapterResult{HTTPStatus: http.StatusForbidden, DurationMS: time.Since(start).Milliseconds()}, execErr
	}

	args := strings.Fields(command)
	args = append(args, buildCommandArgPairs(req)...)
	jsonFlag := strings.TrimSpace(req.Action.Adapter.JSONFlag)
	if jsonFlag == "" {
		jsonFlag = "--json"
	}
	args = append(args, jsonFlag)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = req.Action.Adapter.Timeout
	}
	if timeout <= 0 {
		timeout = a.timeout
	}
	if timeout <= 0 {
		timeout = defaultCommandAdapterTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, executable, args...)
	env := safeProcessEnv()
	for k, v := range req.Action.Adapter.EnvInject {
		if strings.TrimSpace(k) == "" {
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	if req.Credentials != nil {
		credRef := strings.TrimSpace(req.Action.Auth.CredentialRef)
		if credRef != "" {
			value := strings.TrimSpace(req.Credentials.Token)
			if value == "" {
				value = strings.TrimSpace(req.Credentials.APIKey)
			}
			if value == "" {
				value = strings.TrimSpace(req.Credentials.Password)
			}
			if value != "" {
				env = append(env, fmt.Sprintf("KIMBAP_CREDENTIAL_%s=%s", sanitizeEnvSegment(credRef), value))
			}
		}
	}
	cmd.Env = env

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdoutBuf, limit: maxCommandOutputBytes}
	cmd.Stderr = &limitedWriter{w: &stderrBuf, limit: maxCommandStderrBytes}

	err := cmd.Run()
	durationMS := time.Since(start).Milliseconds()
	stdout := stdoutBuf.Bytes()
	stderr := strings.TrimSpace(stderrBuf.String())
	if err != nil {
		status := http.StatusBadGateway
		retryable := execCtx.Err() == context.DeadlineExceeded
		if retryable {
			status = http.StatusGatewayTimeout
		}
		message := fmt.Sprintf("command execution failed: %v", err)
		if stderr != "" {
			message = message + ": " + stderr
		}
		execErr := actions.NewExecutionError(actions.ErrDownstreamUnavailable, message, status, retryable, map[string]any{"stderr": stderr})
		return &AdapterResult{
			Output:     map[string]any{"error": message, "stderr": stderr},
			HTTPStatus: status,
			DurationMS: durationMS,
			Retryable:  retryable,
			RawBody:    stdout,
		}, execErr
	}

	output := map[string]any{}
	trimmed := strings.TrimSpace(string(stdout))
	if trimmed != "" {
		var parsed any
		if jsonErr := json.Unmarshal([]byte(trimmed), &parsed); jsonErr != nil {
			output["raw"] = trimmed
		} else {
			switch v := parsed.(type) {
			case map[string]any:
				output = v
			default:
				output["data"] = v
			}
		}
	}

	return &AdapterResult{
		Output:     output,
		HTTPStatus: http.StatusOK,
		DurationMS: durationMS,
		RawBody:    stdout,
	}, nil
}

func buildCommandArgPairs(req AdapterRequest) []string {
	input := req.Input
	if input == nil {
		return nil
	}

	argNames := make([]string, 0, len(input))
	if req.Action.InputSchema != nil && req.Action.InputSchema.Properties != nil {
		for name := range req.Action.InputSchema.Properties {
			if _, ok := input[name]; ok {
				argNames = append(argNames, name)
			}
		}
	} else {
		for name := range input {
			argNames = append(argNames, name)
		}
	}
	sort.Strings(argNames)

	args := make([]string, 0, len(argNames)*2)
	for _, name := range argNames {
		value, ok := input[name]
		if !ok || !isNonEmptyArgValue(value) {
			continue
		}
		flagName := strings.ReplaceAll(strings.TrimSpace(name), "_", "-")
		if flagName == "" {
			continue
		}
		args = append(args, "--"+flagName, stringifyArgValue(value))
	}
	return args
}

func isNonEmptyArgValue(v any) bool {
	switch value := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(value) != ""
	case []any:
		return len(value) > 0
	case map[string]any:
		return len(value) > 0
	default:
		return true
	}
}

func stringifyArgValue(v any) string {
	switch value := v.(type) {
	case string:
		return value
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(encoded)
	}
}

func safeProcessEnv() []string {
	allowedPrefixes := []string{"PATH=", "HOME=", "USER=", "LOGNAME=", "SHELL=", "TMPDIR=", "TMP=", "TEMP=", "LANG=", "LC_"}
	out := make([]string, 0, len(allowedPrefixes))
	for _, entry := range os.Environ() {
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(entry, prefix) {
				out = append(out, entry)
				break
			}
		}
	}
	return out
}

type limitedWriter struct {
	w     *bytes.Buffer
	limit int64
	n     int64
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	orig := len(p)
	remaining := lw.limit - lw.n
	if remaining <= 0 {
		return orig, nil
	}
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := lw.w.Write(p)
	lw.n += int64(n)
	return orig, err
}

func sanitizeEnvSegment(ref string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(ref)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

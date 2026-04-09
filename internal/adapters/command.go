package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
const maxCommandStderrBytes int64 = 1 << 20

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
	if strings.TrimSpace(def.Adapter.ExecutablePath) == "" && strings.TrimSpace(def.Adapter.Command) == "" {
		return fmt.Errorf("command adapter requires executable path or command")
	}
	if strings.TrimSpace(def.Adapter.ExecutablePath) == "" {
		return fmt.Errorf("command adapter requires executable path")
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
	if executable == "" {
		execErr := actions.NewExecutionError(actions.ErrValidationFailed, "command adapter requires executable path", http.StatusBadRequest, false, nil)
		return &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}, execErr
	}
	if a.allowlist != nil && !a.allowlist[executable] {
		execErr := actions.NewExecutionError(actions.ErrUnauthorized, fmt.Sprintf("executable not allowed: %s", executable), http.StatusForbidden, false, nil)
		return &AdapterResult{HTTPStatus: http.StatusForbidden, DurationMS: time.Since(start).Milliseconds()}, execErr
	}

	var args []string
	if command != "" {
		args = strings.Fields(command)
	}
	args = append(args, buildCommandArgPairs(req)...)
	jsonFlag := strings.TrimSpace(req.Action.Adapter.JSONFlag)
	if jsonFlag == "" {
		jsonFlag = "--json"
	}
	if !isDisabledCommandJSONFlag(jsonFlag) {
		args = append(args, strings.Fields(jsonFlag)...)
	}

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
	for _, pattern := range req.Action.Adapter.EnvPassthrough {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || pattern == "*" {
			continue
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			for _, entry := range os.Environ() {
				if strings.HasPrefix(entry, prefix) {
					env = append(env, entry)
				}
			}
		} else {
			if val, ok := os.LookupEnv(pattern); ok {
				env = append(env, pattern+"="+val)
			}
		}
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
	if wd := strings.TrimSpace(req.Action.Adapter.WorkingDir); wd != "" {
		cmd.Dir = wd
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	stdoutWriter := &limitedWriter{w: &stdoutBuf, limit: maxCommandOutputBytes}
	stderrWriter := &limitedWriter{w: &stderrBuf, limit: maxCommandStderrBytes}
	cmd.Stdout = stdoutWriter
	if req.Action.Adapter.MergeStderr {
		cmd.Stderr = stdoutWriter
	} else {
		cmd.Stderr = stderrWriter
	}

	err := cmd.Run()
	durationMS := time.Since(start).Milliseconds()
	stdout := stdoutBuf.Bytes()
	rawText := strings.TrimSpace(stdoutBuf.String())
	stderr := strings.TrimSpace(stderrBuf.String())
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	if err != nil {
		// Only check success codes for genuine process exits, not launch failures or timeouts.
		// exec.CommandContext timeout produces ExitError with code -1, so also check execCtx.Err().
		if exitErr != nil && execCtx.Err() == nil {
			successCodes := req.Action.Adapter.SuccessCodes
			if successCodes == nil {
				successCodes = []int{0}
			}
			if containsInt(successCodes, exitCode) {
				if stdoutWriter.truncated {
					execErr := actions.NewExecutionError(
						actions.ErrDownstreamUnavailable,
						fmt.Sprintf("command output exceeded %d bytes", maxCommandOutputBytes),
						http.StatusBadGateway,
						false,
						map[string]any{"max_bytes": maxCommandOutputBytes, "exit_code": exitCode},
					)
					return &AdapterResult{
						Output:     map[string]any{"error": execErr.Message, "_exit_code": float64(exitCode)},
						ExitCode:   exitCode,
						HTTPStatus: http.StatusBadGateway,
						DurationMS: durationMS,
						Retryable:  false,
						RawBody:    stdout,
					}, execErr
				}
				output := parseCommandOutput(stdout)
				output["_exit_code"] = float64(exitCode)
				return &AdapterResult{
					Output:     output,
					ExitCode:   exitCode,
					HTTPStatus: http.StatusOK,
					DurationMS: durationMS,
					RawBody:    stdout,
				}, nil
			}
		}
		status := http.StatusBadGateway
		retryable := execCtx.Err() == context.DeadlineExceeded
		if retryable {
			status = http.StatusGatewayTimeout
		}
		diagnostic := stderr
		if req.Action.Adapter.MergeStderr && diagnostic == "" {
			diagnostic = rawText
		}
		message := fmt.Sprintf("command execution failed: %v", err)
		if diagnostic != "" {
			message = message + ": " + diagnostic
		}
		details := map[string]any{"stderr": stderr, "exit_code": exitCode}
		if rawText != "" {
			details["raw"] = rawText
		}
		execErr := actions.NewExecutionError(actions.ErrDownstreamUnavailable, message, status, retryable, details)
		output := map[string]any{"error": message, "stderr": stderr, "_exit_code": float64(exitCode)}
		if rawText != "" {
			output["raw"] = rawText
		}
		return &AdapterResult{
			Output:     output,
			ExitCode:   exitCode,
			HTTPStatus: status,
			DurationMS: durationMS,
			Retryable:  retryable,
			RawBody:    stdout,
		}, execErr
	}

	if stdoutWriter.truncated {
		execErr := actions.NewExecutionError(
			actions.ErrDownstreamUnavailable,
			fmt.Sprintf("command output exceeded %d bytes", maxCommandOutputBytes),
			http.StatusBadGateway,
			false,
			map[string]any{"max_bytes": maxCommandOutputBytes},
		)
		return &AdapterResult{
			Output:     map[string]any{"error": execErr.Message},
			ExitCode:   exitCode,
			HTTPStatus: http.StatusBadGateway,
			DurationMS: durationMS,
			Retryable:  false,
			RawBody:    stdout,
		}, execErr
	}

	output := parseCommandOutput(stdout)
	output["_exit_code"] = float64(0)

	return &AdapterResult{
		Output:     output,
		ExitCode:   0,
		HTTPStatus: http.StatusOK,
		DurationMS: durationMS,
		RawBody:    stdout,
	}, nil
}

func parseCommandOutput(stdout []byte) map[string]any {
	output := map[string]any{}
	trimmed := strings.TrimSpace(string(stdout))
	if trimmed == "" {
		return output
	}

	var parsed any
	if jsonErr := json.Unmarshal([]byte(trimmed), &parsed); jsonErr != nil {
		// Attempt NDJSON: line-delimited JSON (e.g., docker --format json, go test -json)
		if items := parseNDJSON(trimmed); len(items) > 0 {
			output["data"] = items
		} else {
			output["raw"] = trimmed
		}
		return output
	}

	switch v := parsed.(type) {
	case map[string]any:
		return v
	default:
		output["data"] = v
		return output
	}
}

func parseNDJSON(s string) []any {
	lines := strings.Split(s, "\n")
	var items []any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj any
		if err := json.Unmarshal([]byte(line), &obj); err == nil {
			items = append(items, obj)
		}
	}
	return items
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func buildCommandArgPairs(req AdapterRequest) []string {
	input := req.Input
	if input == nil {
		return nil
	}

	argNames := orderedCommandArgNames(req)
	positionalArgs := make([]string, 0, len(argNames))
	flagArgs := make([]string, 0, len(argNames)*2)
	for _, name := range argNames {
		value, ok := input[name]
		if !ok {
			continue
		}

		style := ""
		if schema := req.Action.InputSchema; schema != nil && schema.Properties != nil {
			if prop := schema.Properties[name]; prop != nil {
				style = strings.ToLower(strings.TrimSpace(prop.ArgStyle))
			}
		}

		flagName := strings.ReplaceAll(strings.TrimSpace(name), "_", "-")
		if flagName == "" {
			continue
		}

		switch style {
		case "boolean":
			boolValue, ok := value.(bool)
			if !ok || !boolValue {
				continue
			}
			flagArgs = append(flagArgs, "--"+flagName)
		case "positional":
			if !isNonEmptyArgValue(value) {
				continue
			}
			positionalArgs = append(positionalArgs, stringifyArgValue(value))
		default:
			if !isNonEmptyArgValue(value) {
				continue
			}
			flagArgs = append(flagArgs, "--"+flagName, stringifyArgValue(value))
		}
	}

	return append(positionalArgs, flagArgs...)
}

func orderedCommandArgNames(req AdapterRequest) []string {
	input := req.Input
	if input == nil {
		return nil
	}

	if schema := req.Action.InputSchema; schema != nil && schema.Properties != nil && !schema.AdditionalProperties {
		argNames := make([]string, 0, len(input))
		seen := make(map[string]struct{}, len(input))

		for _, name := range schema.ArgOrder {
			if _, ok := input[name]; ok {
				argNames = append(argNames, name)
				seen[name] = struct{}{}
			}
		}

		fallback := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			if _, ok := input[name]; !ok {
				continue
			}
			if _, done := seen[name]; done {
				continue
			}
			fallback = append(fallback, name)
		}
		sort.Strings(fallback)
		return append(argNames, fallback...)
	}

	argNames := make([]string, 0, len(input))
	for name := range input {
		argNames = append(argNames, name)
	}
	sort.Strings(argNames)
	return argNames
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

func isDisabledCommandJSONFlag(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "none", "off", "false", "-":
		return true
	default:
		return false
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
	w         *bytes.Buffer
	limit     int64
	n         int64
	truncated bool
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	orig := len(p)
	remaining := lw.limit - lw.n
	if remaining <= 0 {
		lw.truncated = true
		return orig, nil
	}
	if int64(len(p)) > remaining {
		lw.truncated = true
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

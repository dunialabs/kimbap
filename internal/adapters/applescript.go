package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters/commands"
)

const defaultAppleScriptAdapterTimeout = 30 * time.Second

type AppleScriptAdapter struct {
	runner   CommandRunner
	commands map[string]commands.Command
}

func NewAppleScriptAdapter(runner CommandRunner) *AppleScriptAdapter {
	if runner == nil {
		runner = NewOSAScriptRunner()
	}

	allCmds := make(map[string]commands.Command)
	maps.Copy(allCmds, commands.NotesCommands())
	maps.Copy(allCmds, commands.CalendarCommands())
	maps.Copy(allCmds, commands.RemindersCommands())
	maps.Copy(allCmds, commands.MailCommands())
	maps.Copy(allCmds, commands.FinderCommands())
	maps.Copy(allCmds, commands.SafariCommands())
	maps.Copy(allCmds, commands.MessagesCommands())
	maps.Copy(allCmds, commands.ContactsCommands())
	maps.Copy(allCmds, commands.MSOfficeCommands())
	maps.Copy(allCmds, commands.IWorkCommands())
	maps.Copy(allCmds, commands.SpotifyCommands())
	maps.Copy(allCmds, commands.ShortcutsCommands())

	return &AppleScriptAdapter{
		runner:   runner,
		commands: allCmds,
	}
}

// Preflight checks if automation permission is granted for a target app.
// It runs a minimal JXA probe; failure is advisory (does not block registration).
func (a *AppleScriptAdapter) Preflight(ctx context.Context, targetApp string) error {
	escaped := strings.ReplaceAll(strings.ReplaceAll(targetApp, `\`, `\\`), `"`, `\"`)
	script := fmt.Sprintf(`Application("%s").name()`, escaped)
	_, stderr, _, _, err := a.runner.Run(ctx, "/usr/bin/osascript", []string{"-l", "JavaScript", "-e", script}, nil)
	if err != nil {
		stderrStr := string(stderr)
		if strings.Contains(stderrStr, "-1743") {
			return fmt.Errorf("automation permission denied for %s. Grant access in System Settings > Privacy & Security > Automation", targetApp)
		}
		return fmt.Errorf("preflight check failed for %s: %s", targetApp, strings.TrimSpace(stderrStr))
	}
	return nil
}

func (a *AppleScriptAdapter) Type() string {
	return "applescript"
}

func (a *AppleScriptAdapter) Validate(def actions.ActionDefinition) error {
	command := strings.TrimSpace(def.Adapter.Command)
	scriptSource := strings.TrimSpace(def.Adapter.ScriptSource)
	registryMode := strings.ToLower(strings.TrimSpace(def.Adapter.RegistryMode))
	if strings.TrimSpace(def.Adapter.TargetApp) == "" {
		return fmt.Errorf("applescript adapter requires target_app")
	}
	if registryMode == "manifest" && scriptSource == "" {
		return fmt.Errorf("applescript manifest registry mode requires script_source")
	}
	if scriptSource != "" {
		if _, err := resolveAppleScriptLanguage(def.Adapter.ScriptLanguage); err != nil {
			return err
		}
		if strings.TrimSpace(def.Adapter.ApprovalRef) == "" {
			return fmt.Errorf("applescript inline script requires approval_ref")
		}
		if strings.TrimSpace(def.Adapter.AuditRef) == "" {
			return fmt.Errorf("applescript inline script requires audit_ref")
		}
		return nil
	}
	if command == "" {
		return fmt.Errorf("applescript adapter requires command or script_source")
	}
	if _, ok := a.commands[command]; !ok {
		return fmt.Errorf("unknown applescript command: %s", command)
	}
	return nil
}

func (a *AppleScriptAdapter) Execute(ctx context.Context, req AdapterRequest) (*AdapterResult, error) {
	start := time.Now()
	commandName := strings.TrimSpace(req.Action.Adapter.Command)
	scriptSource := strings.TrimSpace(req.Action.Adapter.ScriptSource)
	registryMode := strings.ToLower(strings.TrimSpace(req.Action.Adapter.RegistryMode))
	if registryMode == "manifest" && scriptSource == "" {
		execErr := actions.NewExecutionError(actions.ErrValidationFailed, "manifest applescript action missing script_source", http.StatusBadRequest, false, nil)
		return &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}, execErr
	}
	runtimeArgs, langErr := resolveAppleScriptLanguage(req.Action.Adapter.ScriptLanguage)
	if langErr != nil {
		execErr := actions.NewExecutionError(actions.ErrValidationFailed, langErr.Error(), http.StatusBadRequest, false, nil)
		return &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}, execErr
	}
	if scriptSource != "" {
		if strings.TrimSpace(req.Action.Adapter.ApprovalRef) == "" {
			execErr := actions.NewExecutionError(actions.ErrValidationFailed, "inline applescript action missing approval_ref", http.StatusBadRequest, false, nil)
			return &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}, execErr
		}
		if strings.TrimSpace(req.Action.Adapter.AuditRef) == "" {
			execErr := actions.NewExecutionError(actions.ErrValidationFailed, "inline applescript action missing audit_ref", http.StatusBadRequest, false, nil)
			return &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}, execErr
		}
	}
	if scriptSource == "" {
		cmd, ok := a.commands[commandName]
		if !ok {
			execErr := actions.NewExecutionError(actions.ErrValidationFailed, fmt.Sprintf("unknown command: %s", commandName), http.StatusBadRequest, false, nil)
			return &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}, execErr
		}
		scriptSource = cmd.Script
	}
	runtimeArgs = append(runtimeArgs, scriptSource)

	input := req.Input
	if input == nil {
		input = map[string]any{}
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		result := &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}
		execErr := actions.NewExecutionError(actions.ErrValidationFailed, fmt.Sprintf("failed to marshal input: %v", err), http.StatusBadRequest, false, nil)
		return result, execErr
	}

	execCtx := ctx
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = req.Action.Adapter.Timeout
	}
	if timeout <= 0 {
		timeout = defaultAppleScriptAdapterTimeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	stdout, stderr, stdoutTruncated, _, execErr := a.runner.Run(
		execCtx,
		"/usr/bin/osascript",
		runtimeArgs,
		bytes.NewReader(inputJSON),
	)

	durationMS := time.Since(start).Milliseconds()
	if execErr != nil {
		mappedErr := mapAppleScriptError(stderr, execErr)
		return &AdapterResult{
			Output:     map[string]any{"error": mappedErr.Message, "stderr": strings.TrimSpace(string(stderr))},
			HTTPStatus: mappedErr.HTTPStatus,
			DurationMS: durationMS,
			Retryable:  mappedErr.Retryable,
		}, mappedErr
	}
	if stdoutTruncated {
		execErr := actions.NewExecutionError(
			actions.ErrDownstreamUnavailable,
			fmt.Sprintf("command output exceeded %d bytes", maxCommandOutputBytes),
			http.StatusBadGateway,
			false,
			map[string]any{"max_bytes": maxCommandOutputBytes},
		)
		return &AdapterResult{
			Output:     map[string]any{"error": execErr.Message},
			HTTPStatus: http.StatusBadGateway,
			DurationMS: durationMS,
			Retryable:  false,
			RawBody:    stdout,
		}, execErr
	}

	output := map[string]any{}
	trimmed := strings.TrimSpace(string(stdout))
	if trimmed != "" {
		switch trimmed[0] {
		case '[':
			var arr []any
			if jsonErr := json.Unmarshal([]byte(trimmed), &arr); jsonErr == nil {
				output = map[string]any{"data": arr}
			} else {
				output = map[string]any{"raw": trimmed}
			}
		case '{':
			if jsonErr := json.Unmarshal([]byte(trimmed), &output); jsonErr != nil {
				output = map[string]any{"raw": trimmed}
			}
		default:
			output = map[string]any{"raw": trimmed}
		}
	}

	return &AdapterResult{
		Output:     output,
		HTTPStatus: http.StatusOK,
		DurationMS: durationMS,
		RawBody:    stdout,
	}, nil
}

func resolveAppleScriptLanguage(raw string) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "jxa":
		return []string{"-l", "JavaScript", "-e"}, nil
	case "applescript":
		return []string{"-e"}, nil
	default:
		return nil, fmt.Errorf("unsupported applescript script language %q: must be one of jxa or applescript", raw)
	}
}

func mapAppleScriptError(stderr []byte, err error) *actions.ExecutionError {
	if errors.Is(err, context.DeadlineExceeded) {
		return actions.NewExecutionError(actions.ErrDownstreamUnavailable, "applescript execution timed out", http.StatusGatewayTimeout, true, nil)
	}
	if errors.Is(err, context.Canceled) {
		return actions.NewExecutionError(actions.ErrDownstreamUnavailable, "applescript execution canceled", 499, false, nil)
	}

	stderrStr := strings.TrimSpace(string(stderr))
	status := http.StatusInternalServerError
	retryable := false
	message := err.Error()
	code := actions.ErrDownstreamUnavailable

	if stderrStr != "" {
		message = stderrStr
	}

	switch {
	case strings.Contains(stderrStr, "-1743"):
		status = http.StatusForbidden
		code = actions.ErrUnauthorized
	case strings.Contains(stderrStr, "[NOT_SUPPORTED]"):
		status = http.StatusBadRequest
		code = actions.ErrValidationFailed
	case strings.Contains(stderrStr, "-600"):
		status = http.StatusServiceUnavailable
		retryable = true
	case strings.Contains(stderrStr, "-1708"):
		status = http.StatusBadRequest
		code = actions.ErrValidationFailed
		if strings.Contains(strings.ToLower(stderrStr), "message not understood") {
			message = "AppleScript command is not supported by the target app (Message not understood). Use a supported action or update the service command mapping."
		}
	case strings.Contains(stderrStr, "-128"):
		status = 499
	case strings.Contains(stderrStr, "-1712"):
		status = http.StatusGatewayTimeout
		retryable = true
	case strings.Contains(stderrStr, "-1728"):
		status = http.StatusNotFound
		code = actions.ErrActionNotFound
	case strings.Contains(stderrStr, "[NOT_FOUND]"):
		status = http.StatusNotFound
		code = actions.ErrActionNotFound
	case strings.Contains(stderrStr, "[AMBIGUOUS]"):
		status = http.StatusConflict
		code = actions.ErrValidationFailed
	case strings.Contains(stderrStr, "[TIMEOUT]"):
		status = http.StatusGatewayTimeout
		code = actions.ErrDownstreamUnavailable
		retryable = true
	case strings.Contains(stderrStr, "-2700"):
		lower := strings.ToLower(stderrStr)
		if strings.Contains(lower, "application can't be found") ||
			strings.Contains(lower, "application can’t be found") ||
			strings.Contains(lower, "can't get application") ||
			strings.Contains(lower, "can’t get application") {
			status = http.StatusNotFound
			code = actions.ErrActionNotFound
			message = "target application is unavailable. Install/open the app and retry."
		} else {
			status = http.StatusInternalServerError
		}
	}

	return actions.NewExecutionError(code, message, status, retryable, map[string]any{
		"stderr": stderrStr,
	})
}

package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/adapters/commands"
)

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
	script := fmt.Sprintf(`Application("%s").name()`, targetApp)
	_, stderr, err := a.runner.Run(ctx, "/usr/bin/osascript", []string{"-l", "JavaScript", "-e", script}, nil)
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
	if command == "" {
		return fmt.Errorf("applescript adapter requires command")
	}
	if strings.TrimSpace(def.Adapter.TargetApp) == "" {
		return fmt.Errorf("applescript adapter requires target_app")
	}
	if _, ok := a.commands[command]; !ok {
		return fmt.Errorf("unknown applescript command: %s", command)
	}
	return nil
}

func (a *AppleScriptAdapter) Execute(ctx context.Context, req AdapterRequest) (*AdapterResult, error) {
	start := time.Now()
	commandName := strings.TrimSpace(req.Action.Adapter.Command)

	cmd, ok := a.commands[commandName]
	if !ok {
		execErr := actions.NewExecutionError(actions.ErrValidationFailed, fmt.Sprintf("unknown command: %s", commandName), http.StatusBadRequest, false, nil)
		return &AdapterResult{HTTPStatus: http.StatusBadRequest, DurationMS: time.Since(start).Milliseconds()}, execErr
	}

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
	if timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	stdout, stderr, execErr := a.runner.Run(
		execCtx,
		"/usr/bin/osascript",
		[]string{"-l", "JavaScript", "-e", cmd.Script},
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

func mapAppleScriptError(stderr []byte, err error) *actions.ExecutionError {
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
	case strings.Contains(stderrStr, "-600"):
		status = http.StatusServiceUnavailable
		retryable = true
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
	case strings.Contains(stderrStr, "-2700"):
		status = http.StatusInternalServerError
	}

	return actions.NewExecutionError(code, message, status, retryable, map[string]any{
		"stderr": stderrStr,
	})
}

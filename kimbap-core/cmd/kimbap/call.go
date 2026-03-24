package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/spf13/cobra"
)

func newCallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "call <service.action> [--arg value...]",
		Short:              "Execute an installed action",
		DisableFlagParsing: true,
		Args:               cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			actionName := strings.TrimSpace(args[0])
			inputTokens, err := splitGlobalCallFlags(args[1:])
			if err != nil {
				return err
			}

			input, err := parseDynamicInput(inputTokens)
			if err != nil {
				return err
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			def, err := resolveActionByName(cfg, actionName)
			if err != nil {
				return err
			}

			requestID := fmt.Sprintf("req_%d", time.Now().UTC().UnixNano())
			req := actions.ExecutionRequest{
				RequestID:      requestID,
				IdempotencyKey: requestID,
				TenantID:       defaultTenantID(),
				Principal: actions.Principal{
					ID:        "cli",
					TenantID:  defaultTenantID(),
					AgentName: "kimbap-cli",
					Type:      "operator",
				},
				Action: *def,
				Input:  input,
				Mode:   actions.ModeCall,
			}

			if isDryRun() {
				preview := buildDryRunPreview(cfg, req)
				if err := printOutput(preview); err != nil {
					return err
				}
				// Exit non-zero if validation failed during dry-run
				if validErr, ok := preview["input_valid"].(bool); ok && !validErr {
					return fmt.Errorf("dry-run: input validation failed")
				}
				return nil
			}

			rt, buildErr := buildRuntimeFromConfig(cfg)
			if buildErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "warning: runtime unavailable (%v), showing dry-run preview\n", buildErr)
				preview := buildDryRunPreview(cfg, req)
				_ = printOutput(preview)
				return fmt.Errorf("runtime unavailable: %w", buildErr)
			}

			var result actions.ExecutionResult
			if isTrace() {
				var traceSteps []runtime.TraceStep
				result, traceSteps = rt.ExecuteWithTrace(contextBackground(), req)
				if err := printTraceSteps(traceSteps); err != nil {
					return err
				}
			} else {
				result = rt.Execute(contextBackground(), req)
			}
			if err := printOutput(result); err != nil {
				return err
			}
			if result.Status != actions.StatusSuccess && result.Error != nil {
				return result.Error
			}
			return nil
		},
	}
	return cmd
}

func buildDryRunPreview(cfg *config.KimbapConfig, req actions.ExecutionRequest) map[string]any {
	validationErr := actions.ValidateInput(req.Action.InputSchema, req.Input)
	credentialRef := strings.TrimSpace(req.Action.Auth.CredentialRef)

	var validationError any
	if validationErr != nil {
		validationError = validationErr.Error()
	}

	return map[string]any{
		"dry_run":                true,
		"action":                 req.Action,
		"input":                  req.Input,
		"input_valid":            validationErr == nil,
		"validation_error":       validationError,
		"credential_ref":         credentialRef,
		"credential_ready":       isCredentialReady(cfg, req),
		"policy_path":            strings.TrimSpace(cfg.Policy.Path),
		"would_require_approval": req.Action.ApprovalHint == actions.ApprovalRequired,
	}
}

func isCredentialReady(cfg *config.KimbapConfig, req actions.ExecutionRequest) bool {
	if req.Action.Auth.Type == actions.AuthTypeNone || req.Action.Auth.Optional {
		return true
	}
	credentialRef := strings.TrimSpace(req.Action.Auth.CredentialRef)
	if credentialRef == "" {
		return false
	}
	vs, err := initVaultStore(cfg)
	if err != nil {
		return false
	}
	raw, getErr := vs.GetValue(contextBackground(), defaultTenantID(), credentialRef)
	return getErr == nil && len(raw) > 0
}

func splitGlobalCallFlags(tokens []string) ([]string, error) {
	out := make([]string, 0, len(tokens))
	globalStringFlags := map[string]*string{
		"--format":    &opts.format,
		"--config":    &opts.configPath,
		"--data-dir":  &opts.dataDir,
		"--log-level": &opts.logLevel,
		"--mode":      &opts.mode,
	}
	for i := 0; i < len(tokens); i++ {
		tok := strings.TrimSpace(tokens[i])
		switch {
		case tok == "--dry-run":
			value, consumed := parseOptionalBoolFlagValue(tokens, i)
			opts.dryRun = value
			i += consumed
			continue
		case strings.HasPrefix(tok, "--dry-run="):
			value, err := strconv.ParseBool(strings.TrimSpace(strings.TrimPrefix(tok, "--dry-run=")))
			if err != nil {
				return nil, fmt.Errorf("invalid --dry-run value %q", tok)
			}
			opts.dryRun = value
			continue
		case tok == "--trace":
			value, consumed := parseOptionalBoolFlagValue(tokens, i)
			opts.trace = value
			i += consumed
			continue
		case strings.HasPrefix(tok, "--trace="):
			value, err := strconv.ParseBool(strings.TrimSpace(strings.TrimPrefix(tok, "--trace=")))
			if err != nil {
				return nil, fmt.Errorf("invalid --trace value %q", tok)
			}
			opts.trace = value
			continue
		default:
			handled := false
			if target, ok := globalStringFlags[tok]; ok && i+1 < len(tokens) && !strings.HasPrefix(strings.TrimSpace(tokens[i+1]), "--") {
				i++
				*target = strings.TrimSpace(tokens[i])
				handled = true
			} else {
				for prefix, target := range globalStringFlags {
					if strings.HasPrefix(tok, prefix+"=") {
						*target = strings.TrimSpace(strings.TrimPrefix(tok, prefix+"="))
						handled = true
						break
					}
				}
			}
			if !handled {
				out = append(out, tokens[i])
			}
		}
	}
	return out, nil
}

func parseOptionalBoolFlagValue(tokens []string, idx int) (bool, int) {
	nextIdx := idx + 1
	if nextIdx >= len(tokens) {
		return true, 0
	}
	next := strings.TrimSpace(tokens[nextIdx])
	if strings.HasPrefix(next, "--") {
		return true, 0
	}
	parsed, err := strconv.ParseBool(next)
	if err != nil {
		return true, 0
	}
	return parsed, 1
}

func printTraceSteps(steps []runtime.TraceStep) error {
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"trace": steps})
}

func parseDynamicInput(tokens []string) (map[string]any, error) {
	out := map[string]any{}

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "--" {
			continue
		}
		if !strings.HasPrefix(tok, "--") {
			return nil, fmt.Errorf("unexpected argument %q, expected --name value", tok)
		}

		nameValue := strings.TrimPrefix(tok, "--")
		if nameValue == "" {
			return nil, fmt.Errorf("empty flag name")
		}

		var (
			name  string
			value any = true
		)
		if left, right, ok := strings.Cut(nameValue, "="); ok {
			name = left
			value = parseScalar(right)
		} else {
			name = nameValue
			if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "--") {
				i++
				value = parseScalar(tokens[i])
			}
		}

		if existing, exists := out[name]; exists {
			switch typed := existing.(type) {
			case []any:
				out[name] = append(typed, value)
			default:
				out[name] = []any{typed, value}
			}
		} else {
			out[name] = value
		}
	}

	return out, nil
}

func parseScalar(v string) any {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return ""
	}

	if b, err := strconv.ParseBool(trimmed); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f
	}
	return v
}

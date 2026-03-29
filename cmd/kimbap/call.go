package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/runtime"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newCallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call <service.action> [--arg value...]",
		Short: "Execute an installed action",
		Long: `Execute an installed action by name with key-value arguments.

Arguments after <service.action> are passed to the action as input flags.

Special call flags:
  --dry-run                validate and preview without executing
  --trace                  print execution trace to stderr
  --format json            emit JSON output (place before action name)
  --json <object|-|@file>  merge JSON object into action input

Globally consumed flags (cannot be used as action parameter names):
  --json, --config, --data-dir, --log-level, --mode,
  --idempotency-key, --dry-run, --trace, --no-splash

  --format is consumed globally only when placed BEFORE the action name;
  after the action name it is passed through as an action input parameter.

Discover available actions:
  kimbap search <query>    Search actions by keyword
  kimbap actions list      List all installed actions`,
		Example: `  # Send a Slack message
  kimbap call slack.send-message --channel general --text "deployed v2.1"

  # List GitHub repos
  kimbap call github.list-repos --sort updated

  # Search Apple Notes (no credentials needed on macOS)
  kimbap call apple-notes.search-notes --query "meeting"

  # Preview without executing
  kimbap call stripe.list-charges --limit 5 --dry-run

  # Pass input as JSON from a file
  kimbap call github.create-issue --json @payload.json

  # Get JSON output for scripting
  kimbap --format json call github.list-repos --sort updated`,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			prescanCallSplashFlags(args)
			showSplashOnce()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.dryRun = false
			opts.trace = false
			opts.format = "text"
			opts.jsonInput = ""
			opts.idempotencyKey = ""
			opts.configPath = ""
			opts.dataDir = ""
			opts.logLevel = ""
			opts.mode = ""
			opts.noSplash = false
			actionName, inputTokens, showHelp, err := splitCallInvocationArgs(args)
			if err != nil {
				return err
			}
			if showHelp {
				if actionName != "" {
					cfg, cfgErr := loadAppConfigReadOnly()
					if cfgErr == nil {
						def, resolveErr := resolveActionByName(cfg, actionName)
						if resolveErr == nil {
							fmt.Print(formatActionHelp(*def))
							return nil
						}
					}
				}
				return cmd.Help()
			}
			if actionName == "" {
				return fmt.Errorf("missing action name: expected <service.action>")
			}

			input, err := parseDynamicInput(inputTokens)
			if err != nil {
				return err
			}
			if strings.TrimSpace(opts.jsonInput) != "" {
				jsonInput, parseErr := parseJSONInput(opts.jsonInput)
				if parseErr != nil {
					return parseErr
				}
				input = mergeInputMaps(input, jsonInput)
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			def, err := resolveActionByName(cfg, actionName)
			if err != nil {
				return err
			}

			requestID := "req_" + uuid.NewString()
			req := actions.ExecutionRequest{
				RequestID:      requestID,
				IdempotencyKey: strings.TrimSpace(opts.idempotencyKey),
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
				if validErr, ok := preview["input_valid"].(bool); ok && !validErr {
					return fmt.Errorf("dry-run: input validation failed")
				}
				if idemValid, ok := preview["idempotency_valid"].(bool); ok && !idemValid {
					return fmt.Errorf("dry-run: non-idempotent action requires --idempotency-key")
				}
				return nil
			}

			rt, runtimeCleanup, buildErr := buildRuntimeFromConfigWithCleanup(cfg)
			if buildErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "warning: %s, showing dry-run preview\n", unavailableMessage(componentRuntime, buildErr))
				preview := buildDryRunPreview(cfg, req)
				_ = printOutput(preview)
				return unavailableError(componentRuntime, buildErr)
			}
			defer runtimeCleanup()

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
			if result.Status == actions.StatusApprovalRequired && !outputAsJSON() {
				warning := "!"
				if isColorStdout() {
					warning = "\x1b[33m!\x1b[0m"
				}
				_, _ = fmt.Fprintf(os.Stderr, "%s Approval required for: %s\n", warning, actionName)
			}
			if err := printCallResult(result); err != nil {
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

func printCallResult(result actions.ExecutionResult) error {
	if outputAsJSON() {
		return printOutput(result)
	}

	if result.Status == actions.StatusSuccess {
		checkmark := "✓"
		if isColorStdout() {
			checkmark = "\x1b[32m✓\x1b[0m"
		}
		if len(result.Output) == 0 {
			_, _ = fmt.Fprintln(os.Stdout, checkmark+" Done")
			return nil
		}
		encoded, err := json.MarshalIndent(result.Output, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintln(os.Stdout, checkmark+" Done")
			return nil
		}
		_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", checkmark, string(encoded))
		return nil
	}

	if result.Status == actions.StatusApprovalRequired {
		approvalID, _ := result.Meta["approval_request_id"].(string)
		approvalID = strings.TrimSpace(approvalID)
		if approvalID != "" {
			_, _ = fmt.Fprintf(os.Stderr, "Run: kimbap approve accept %s\n", approvalID)
		} else {
			_, _ = fmt.Fprintln(os.Stderr, "Run: kimbap approve accept <approval_request_id>")
		}
	}

	return nil
}

func formatActionHelp(def actions.ActionDefinition) string {
	var b strings.Builder

	b.WriteString("Action: ")
	b.WriteString(strings.TrimSpace(def.Name))
	b.WriteString("\n\nDescription: ")
	b.WriteString(strings.TrimSpace(def.Description))
	if r := strings.TrimSpace(string(def.Risk)); r != "" {
		b.WriteString("\nRisk: ")
		b.WriteString(r)
	}
	b.WriteString("\n\nParameters:\n")

	if def.InputSchema == nil || len(def.InputSchema.Properties) == 0 {
		b.WriteString("  (no input parameters)\n")
		b.WriteString("\nUsage:\n  kimbap call ")
		b.WriteString(strings.TrimSpace(def.Name))
		b.WriteString("\n")
		return b.String()
	}

	required := make(map[string]bool, len(def.InputSchema.Required))
	for _, name := range def.InputSchema.Required {
		required[name] = true
	}

	propNames := make([]string, 0, len(def.InputSchema.Properties))
	for name := range def.InputSchema.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	var requiredParams, optionalParams []string
	for _, name := range propNames {
		prop := def.InputSchema.Properties[name]
		typeName := "any"
		if prop != nil {
			if t := strings.TrimSpace(prop.Type); t != "" {
				typeName = t
			}
		}

		enumStr := ""
		if prop != nil && len(prop.Enum) > 0 && len(prop.Enum) <= 8 {
			parts := make([]string, len(prop.Enum))
			for i, v := range prop.Enum {
				parts[i] = fmt.Sprintf("%v", v)
			}
			enumStr = "  (one of: " + strings.Join(parts, ", ") + ")"
		}

		if required[name] {
			b.WriteString("  ")
			b.WriteString(name)
			b.WriteString("  ")
			b.WriteString(typeName)
			b.WriteString(enumStr)
			b.WriteString("  [required]\n")
			requiredParams = append(requiredParams, fmt.Sprintf("--%s <%s>", name, typeName))
		} else {
			b.WriteString("  ")
			b.WriteString(name)
			b.WriteString("  ")
			b.WriteString(typeName)
			b.WriteString(enumStr)
			b.WriteString("\n")
			optionalParams = append(optionalParams, fmt.Sprintf("[--%s <%s>]", name, typeName))
		}
	}

	b.WriteString("\nUsage:\n  kimbap call ")
	b.WriteString(strings.TrimSpace(def.Name))
	for _, p := range requiredParams {
		b.WriteString(" ")
		b.WriteString(p)
	}
	for _, p := range optionalParams {
		b.WriteString(" ")
		b.WriteString(p)
	}
	b.WriteString("\n")

	return b.String()
}

func buildDryRunPreview(cfg *config.KimbapConfig, req actions.ExecutionRequest) map[string]any {
	validationErr := actions.ValidateInput(req.Action.InputSchema, req.Input)
	credentialRef := strings.TrimSpace(req.Action.Auth.CredentialRef)
	authReady := isCredentialReady(cfg, req)
	approvalNeeded := req.Action.ApprovalHint == actions.ApprovalRequired

	resolvedHeaders := map[string]string{}
	for k, v := range req.Action.Adapter.Headers {
		resolvedHeaders[k] = resolvePreviewTemplate(v, req.Input)
	}
	resolvedHeaders = maskSensitivePreviewHeaders(resolvedHeaders, req.Action.Auth)

	resolvedQuery := map[string]string{}
	for k, v := range req.Action.Adapter.Query {
		resolvedQuery[k] = resolvePreviewTemplate(v, req.Input)
	}

	resolvedURL := resolvePreviewURL(req.Action, req.Input, resolvedQuery)
	requestBodyPreview := buildRequestBodyPreview(req)

	var validationError any
	if validationErr != nil {
		validationError = validationErr.Error()
	}

	idempotencyValid := true
	var idempotencyError any
	if !req.Action.Idempotent && strings.TrimSpace(req.IdempotencyKey) == "" {
		idempotencyValid = false
		idempotencyError = "non-idempotent action requires --idempotency-key"
	}

	return map[string]any{
		"dry_run":                true,
		"action":                 req.Action,
		"input":                  req.Input,
		"idempotency_key":        strings.TrimSpace(req.IdempotencyKey),
		"idempotency_valid":      idempotencyValid,
		"idempotency_error":      idempotencyError,
		"input_valid":            validationErr == nil,
		"validation_error":       validationError,
		"credential_ref":         credentialRef,
		"credential_ready":       authReady,
		"auth_type":              string(req.Action.Auth.Type),
		"auth_ready":             authReady,
		"policy_path":            strings.TrimSpace(cfg.Policy.Path),
		"would_require_approval": approvalNeeded,
		"approval_needed":        approvalNeeded,
		"http_method":            strings.ToUpper(strings.TrimSpace(req.Action.Adapter.Method)),
		"resolved_url":           resolvedURL,
		"resolved_headers":       resolvedHeaders,
		"resolved_query":         resolvedQuery,
		"request_body_preview":   requestBodyPreview,
	}
}

func resolvePreviewURL(action actions.ActionDefinition, input map[string]any, query map[string]string) string {
	base := strings.TrimSuffix(strings.TrimSpace(action.Adapter.BaseURL), "/")
	path := strings.TrimSpace(resolvePreviewTemplate(action.Adapter.URLTemplate, input))
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	resolved := base + path
	if len(query) == 0 {
		return resolved
	}
	values := url.Values{}
	for k, v := range query {
		values.Set(k, v)
	}
	encoded := values.Encode()
	if encoded == "" {
		return resolved
	}
	return resolved + "?" + encoded
}

func resolvePreviewTemplate(tmpl string, input map[string]any) string {
	out := tmpl
	for key, value := range input {
		out = strings.ReplaceAll(out, "{"+key+"}", fmt.Sprintf("%v", value))
	}
	return out
}

func maskSensitivePreviewHeaders(headers map[string]string, auth actions.AuthRequirement) map[string]string {
	masked := map[string]string{}
	authHeaderName := strings.ToLower(strings.TrimSpace(auth.HeaderName))
	for k, v := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(k))
		if lowerKey == "authorization" || lowerKey == "proxy-authorization" || lowerKey == authHeaderName || strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "api-key") || strings.Contains(lowerKey, "apikey") || strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "password") {
			masked[k] = "***"
			continue
		}
		masked[k] = v
	}
	return masked
}

func buildRequestBodyPreview(req actions.ExecutionRequest) any {
	body := strings.TrimSpace(req.Action.Adapter.RequestBody)
	if body != "" {
		return truncatePreview(resolvePreviewTemplate(body, req.Input), 2048)
	}
	b, err := json.Marshal(req.Input)
	if err != nil {
		return nil
	}
	return truncatePreview(string(b), 2048)
}

func truncatePreview(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	cut := maxLen
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + "..."
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
	defer closeVaultStoreIfPossible(vs)
	raw, getErr := vs.GetValue(contextBackground(), defaultTenantID(), credentialRef)
	return getErr == nil && len(raw) > 0
}

func prescanCallSplashFlags(tokens []string) {
	for i := 0; i < len(tokens); i++ {
		tok := strings.TrimSpace(tokens[i])
		if tok == "--" {
			break
		}
		switch {
		case tok == "--no-splash":
			value, consumed := parseOptionalBoolFlagValue(tokens, i)
			opts.noSplash = value
			i += consumed
		case tok == "--format" && i+1 < len(tokens):
			next := strings.TrimSpace(tokens[i+1])
			if !strings.HasPrefix(next, "-") {
				opts.format = next
				i++
			}
		case strings.HasPrefix(tok, "--format="):
			opts.format = strings.TrimSpace(strings.TrimPrefix(tok, "--format="))
		}
	}
}

func splitGlobalCallFlags(tokens []string) ([]string, error) {
	out := make([]string, 0, len(tokens))
	globalStringFlags := map[string]*string{
		"--format":          &opts.format,
		"--json":            &opts.jsonInput,
		"--config":          &opts.configPath,
		"--data-dir":        &opts.dataDir,
		"--log-level":       &opts.logLevel,
		"--mode":            &opts.mode,
		"--idempotency-key": &opts.idempotencyKey,
	}
	// actionFlagConflicts: these global flags must NOT be consumed after the action name
	// because services may have identically-named input parameters (e.g. --format in mermaid).
	// Note: --json is intentionally excluded — it reads JSON input and cannot conflict with service args.
	actionFlagConflicts := map[string]bool{
		"--format": true,
	}
	actionSeen := false
	for i := 0; i < len(tokens); i++ {
		tok := strings.TrimSpace(tokens[i])
		if tok == "--" {
			out = append(out, tokens[i:]...)
			return out, nil
		}
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
		case tok == "--no-splash":
			value, consumed := parseOptionalBoolFlagValue(tokens, i)
			opts.noSplash = value
			i += consumed
			continue
		case strings.HasPrefix(tok, "--no-splash="):
			value, err := strconv.ParseBool(strings.TrimSpace(strings.TrimPrefix(tok, "--no-splash=")))
			if err != nil {
				return nil, fmt.Errorf("invalid --no-splash value %q", tok)
			}
			opts.noSplash = value
			continue
		default:
			handled := false
			if target, ok := globalStringFlags[tok]; ok && !(actionSeen && actionFlagConflicts[tok]) {
				next := ""
				if i+1 < len(tokens) {
					next = strings.TrimSpace(tokens[i+1])
				}
				if i+1 >= len(tokens) || (strings.HasPrefix(next, "-") && next != "-") {
					return nil, fmt.Errorf("flag %s requires a value", tok)
				}
				i++
				*target = strings.TrimSpace(tokens[i])
				handled = true
			} else if !handled {
				for prefix, target := range globalStringFlags {
					if actionSeen && actionFlagConflicts[prefix] {
						continue
					}
					if strings.HasPrefix(tok, prefix+"=") {
						*target = strings.TrimSpace(strings.TrimPrefix(tok, prefix+"="))
						handled = true
						break
					}
				}
			}
			if !handled {
				if !strings.HasPrefix(tok, "-") {
					actionSeen = true
				}
				out = append(out, tokens[i])
			}
		}
	}
	return out, nil
}

func parseJSONInput(jsonArg string) (map[string]any, error) {
	var raw string
	switch {
	case jsonArg == "-":
		const maxStdinBytes int64 = 4 << 20
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxStdinBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		if int64(len(data)) > maxStdinBytes {
			return nil, fmt.Errorf("stdin input exceeds %d bytes", maxStdinBytes)
		}
		raw = string(data)
	case strings.HasPrefix(jsonArg, "@"):
		const maxFileBytes int64 = 4 << 20
		f, openErr := os.Open(strings.TrimPrefix(jsonArg, "@"))
		if openErr != nil {
			return nil, fmt.Errorf("read json file: %w", openErr)
		}
		data, readErr := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
		_ = f.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read json file: %w", readErr)
		}
		if int64(len(data)) > maxFileBytes {
			return nil, fmt.Errorf("json file input exceeds %d bytes", maxFileBytes)
		}
		raw = string(data)
	default:
		raw = jsonArg
	}

	// Use json.Number to preserve integer precision (json.Unmarshal defaults
	// to float64 which breaks "integer" schema validation).
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var parsed map[string]any
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("parse json input: %w", err)
	}
	var extra json.RawMessage
	if dec.Decode(&extra) != io.EOF {
		return nil, fmt.Errorf("parse json input: unexpected trailing data after JSON object")
	}
	return coerceJSONNumbers(parsed), nil
}

// coerceJSONNumbers walks a map and converts json.Number values to int64 when
// the number has no fractional part, otherwise to float64. This ensures that
// integer-typed schema fields receive Go int values, not float64.
func coerceJSONNumbers(m map[string]any) map[string]any {
	for k, v := range m {
		switch val := v.(type) {
		case json.Number:
			if i, err := val.Int64(); err == nil {
				m[k] = i
			} else if f, err := val.Float64(); err == nil {
				m[k] = f
			}
		case map[string]any:
			m[k] = coerceJSONNumbers(val)
		case []any:
			m[k] = coerceJSONNumbersSlice(val)
		}
	}
	return m
}

func coerceJSONNumbersSlice(s []any) []any {
	for i, v := range s {
		switch val := v.(type) {
		case json.Number:
			if n, err := val.Int64(); err == nil {
				s[i] = n
			} else if f, err := val.Float64(); err == nil {
				s[i] = f
			}
		case map[string]any:
			s[i] = coerceJSONNumbers(val)
		case []any:
			s[i] = coerceJSONNumbersSlice(val)
		}
	}
	return s
}

func mergeInputMaps(base map[string]any, override map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range override {
		base[k] = v
	}
	return base
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

func splitCallInvocationArgs(tokens []string) (string, []string, bool, error) {
	filtered, err := splitGlobalCallFlags(tokens)
	if err != nil {
		return "", nil, false, err
	}

	showHelp := false
	parts := make([]string, 0, len(filtered))
	for _, tok := range filtered {
		trimmed := strings.TrimSpace(tok)
		if trimmed == "--help" || trimmed == "-h" {
			showHelp = true
			continue
		}
		parts = append(parts, tok)
	}

	actionIdx := -1
	for i, tok := range parts {
		trimmed := strings.TrimSpace(tok)
		if trimmed == "--" {
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			if actionIdx == -1 {
				return "", nil, false, fmt.Errorf("missing action name before argument %q", tok)
			}
			continue
		}
		actionIdx = i
		break
	}

	if actionIdx == -1 {
		if showHelp {
			return "", nil, true, nil
		}
		if len(parts) == 0 {
			return "", nil, false, nil
		}
		return "", nil, false, fmt.Errorf("missing action name: expected <service.action>")
	}

	actionName := strings.TrimSpace(parts[actionIdx])
	inputTokens := make([]string, 0, len(parts)-actionIdx-1)
	inputTokens = append(inputTokens, parts[actionIdx+1:]...)

	if showHelp {
		return actionName, inputTokens, true, nil
	}

	return actionName, inputTokens, false, nil
}

func parseScalar(v string) any {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return ""
	}

	if len(trimmed) > 1 && trimmed[0] == '0' && trimmed[1] != '.' {
		return v
	}
	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(trimmed); err == nil {
		return b
	}
	return v
}

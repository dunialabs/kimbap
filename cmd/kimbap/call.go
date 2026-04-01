package main

import (
	"encoding/json"
	"fmt"
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
  --splash-color, --idempotency-key, --dry-run, --trace, --no-splash

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
			opts = cliOptions{format: "text"}
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
				return fmt.Errorf("missing action name\n\nUsage:\n  kimbap call <service.action> [--arg value...]\n\nRun 'kimbap call --help' for examples, or 'kimbap search <query>' to find actions.")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			def, err := resolveActionByName(cfg, actionName)
			if err != nil {
				return err
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
			applyMisplacedGlobalFormat(&opts, def, input)
			coerceArrayInputsBySchema(def, input)
			coerceStringInputsBySchema(def, input)

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

			if !outputAsJSON() && !isDryRun() {
				if err := checkRequiredInputs(def, input); err != nil {
					return err
				}
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

func applyMisplacedGlobalFormat(o *cliOptions, def *actions.ActionDefinition, input map[string]any) {
	if actionHasInputField(def, "format") {
		return
	}
	fmtVal, hasFormat := input["format"]
	if !hasFormat {
		return
	}
	fmtStr, isString := fmtVal.(string)
	if !isString {
		return
	}
	fmtStr = strings.TrimSpace(fmtStr)
	if fmtStr != "text" && fmtStr != "json" {
		return
	}
	o.format = fmtStr
	delete(input, "format")
	_, _ = fmt.Fprintf(os.Stderr, "note: --format must be placed before the action name; applying it automatically\n")
}

func coerceArrayInputsBySchema(def *actions.ActionDefinition, input map[string]any) {
	if def == nil || def.InputSchema == nil || len(def.InputSchema.Properties) == 0 || len(input) == 0 {
		return
	}
	for name, prop := range def.InputSchema.Properties {
		if prop == nil || !strings.EqualFold(strings.TrimSpace(prop.Type), "array") {
			continue
		}
		val, exists := input[name]
		if !exists {
			continue
		}
		if _, isArray := val.([]any); isArray {
			continue
		}
		input[name] = []any{val}
	}
}

func coerceStringInputsBySchema(def *actions.ActionDefinition, input map[string]any) {
	if def == nil || def.InputSchema == nil || len(def.InputSchema.Properties) == 0 || len(input) == 0 {
		return
	}
	for name, prop := range def.InputSchema.Properties {
		if prop == nil || !strings.EqualFold(strings.TrimSpace(prop.Type), "string") {
			continue
		}
		val, exists := input[name]
		if !exists {
			continue
		}
		if _, isString := val.(string); isString {
			continue
		}
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64:
			input[name] = fmt.Sprintf("%v", val)
		}
	}
}

func actionHasInputField(def *actions.ActionDefinition, fieldName string) bool {
	if def == nil || def.InputSchema == nil || len(def.InputSchema.Properties) == 0 {
		return false
	}
	_, exists := def.InputSchema.Properties[fieldName]
	return exists
}

func checkRequiredInputs(def *actions.ActionDefinition, input map[string]any) error {
	if def == nil || def.InputSchema == nil || len(def.InputSchema.Required) == 0 {
		return nil
	}

	resolved := cloneInputWithDefaults(input, def.Defaults)
	missing := missingRequiredInputs(def.InputSchema.Required, resolved)
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf(
		"missing required parameters: %s\n\nUsage:\n  %s\n\nRun 'kimbap call %s --help' for details.",
		strings.Join(missing, ", "),
		buildCallUsageLine(*def),
		strings.TrimSpace(def.Name),
	)
}

func cloneInputWithDefaults(input map[string]any, defaults map[string]any) map[string]any {
	if len(input) == 0 && len(defaults) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(input)+len(defaults))
	for k, v := range input {
		cloned[k] = v
	}
	for k, v := range defaults {
		if _, exists := cloned[k]; !exists {
			cloned[k] = v
		}
	}
	return cloned
}

func missingRequiredInputs(required []string, input map[string]any) []string {
	missing := make([]string, 0, len(required))
	for _, name := range required {
		if _, exists := input[name]; !exists {
			missing = append(missing, "--"+name)
		}
	}
	sort.Strings(missing)
	return missing
}

func buildCallUsageLine(def actions.ActionDefinition) string {
	usage := []string{"kimbap", "call", strings.TrimSpace(def.Name)}
	if def.InputSchema == nil || len(def.InputSchema.Properties) == 0 {
		return strings.Join(usage, " ")
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

	requiredSegments := make([]string, 0, len(propNames))
	optionalSegments := make([]string, 0, len(propNames))
	for _, name := range propNames {
		prop := def.InputSchema.Properties[name]
		typeName := "any"
		if prop != nil {
			if t := strings.TrimSpace(prop.Type); t != "" {
				typeName = t
			}
		}
		segment := fmt.Sprintf("--%s <%s>", name, typeName)
		if required[name] {
			requiredSegments = append(requiredSegments, segment)
			continue
		}
		optionalSegments = append(optionalSegments, "["+segment+"]")
	}

	usage = append(usage, requiredSegments...)
	usage = append(usage, optionalSegments...)
	return strings.Join(usage, " ")
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
	actionSeen := false
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
		case strings.HasPrefix(tok, "--no-splash="):
			if v, err := strconv.ParseBool(strings.TrimSpace(strings.TrimPrefix(tok, "--no-splash="))); err == nil {
				opts.noSplash = v
			}
		case !actionSeen && tok == "--format" && i+1 < len(tokens):
			next := strings.TrimSpace(tokens[i+1])
			if !strings.HasPrefix(next, "-") {
				opts.format = next
				i++
			}
		case tok == "--splash-color" && i+1 < len(tokens):
			next := strings.TrimSpace(tokens[i+1])
			if !strings.HasPrefix(next, "-") {
				opts.splashColor = next
				i++
			}
		case !actionSeen && strings.HasPrefix(tok, "--format="):
			opts.format = strings.TrimSpace(strings.TrimPrefix(tok, "--format="))
		case strings.HasPrefix(tok, "--splash-color="):
			opts.splashColor = strings.TrimSpace(strings.TrimPrefix(tok, "--splash-color="))
		case !strings.HasPrefix(tok, "-"):
			actionSeen = true
		}
	}
}

func splitGlobalCallFlags(tokens []string) ([]string, error) {
	out := make([]string, 0, len(tokens))
	globalStringFlags := map[string]*string{
		"--format":          &opts.format,
		"--splash-color":    &opts.splashColor,
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

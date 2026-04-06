package services

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
)

func ToActionDefinitions(svc *ServiceManifest) ([]actions.ActionDefinition, error) {
	if svc == nil {
		return nil, fmt.Errorf("service manifest is nil")
	}
	if errs := ValidateManifest(svc); len(errs) > 0 {
		return nil, validationErrorsToError("invalid service manifest", errs)
	}

	adapterType := normalizedAdapterType(svc.Adapter)

	switch adapterType {
	case "applescript":
		return toAppleScriptDefinitions(svc)
	case "command":
		return toCommandDefinitions(svc)
	default:
		return toHTTPDefinitions(svc)
	}
}

func toHTTPDefinitions(svc *ServiceManifest) ([]actions.ActionDefinition, error) {
	u, err := url.Parse(svc.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	keys := sortedKeys(svc.Actions)

	out := make([]actions.ActionDefinition, 0, len(keys))
	for _, key := range keys {
		actionSpec := svc.Actions[key]
		retry := actions.RetryConfig{}
		if actionSpec.Retry != nil {
			retry.MaxAttempts = actionSpec.Retry.MaxAttempts
			retry.BackoffMS = actionSpec.Retry.BackoffMS
			retry.RetryOn429 = slices.Contains(actionSpec.Retry.RetryOn, 429)
			retry.RetryOn5xx = has5xx(actionSpec.Retry.RetryOn)
		}

		idempotent := resolveIdempotent(actionSpec, isIdempotent(actionSpec.Method))

		definition := actions.ActionDefinition{
			Name:         svc.Name + "." + key,
			Version:      1,
			DisplayName:  nonEmptyString(actionSpec.Description, key),
			Namespace:    svc.Name,
			Verb:         strings.ToLower(actionSpec.Method),
			Resource:     actionSpec.Path,
			Description:  actionSpec.Description,
			Risk:         mapRisk(actionSpec.Risk.Level),
			Idempotent:   idempotent,
			ApprovalHint: mapApprovalHint(actionSpec.Risk.Level),
			Auth:         mapAuth(resolveActionAuth(svc.Auth, actionSpec.Auth)),
			InputSchema:  buildInputSchema(actionSpec.Args, actionSpec.Request.PathParams, actionSpec.Pagination),
			OutputSchema: buildOutputSchema(actionSpec.Response),
			Defaults:     collectDefaults(actionSpec.Args),
			FilterConfig:    convertFilterSpec(actionSpec.Response.Filter),
			CompactTemplate: convertCompactSpec(actionSpec.Response.Compact),
			Adapter: actions.AdapterConfig{
				Type:        "http",
				Method:      strings.ToUpper(actionSpec.Method),
				BaseURL:     svc.BaseURL,
				URLTemplate: actionSpec.Path,
				Headers:     cloneStringMap(actionSpec.Request.Headers),
				Query:       cloneStringMap(actionSpec.Request.Query),
				RequestBody: marshalBody(actionSpec.Request.Body),
				Response: actions.ResponseConfig{
					Extract: actionSpec.Response.Extract,
				},
				Retry: retry,
			},
			Classifiers: []actions.ClassifierRule{{
				Service:     svc.Name,
				Action:      key,
				Method:      strings.ToUpper(actionSpec.Method),
				PathPattern: actionSpec.Path,
				HostPattern: u.Host,
			}},
			ErrorMapping: cloneIntMap(actionSpec.ErrorMapping),
			Pagination:   mapPagination(actionSpec.Pagination),
		}

		injectOutputModeParam(&definition)
		out = append(out, definition)
	}

	return out, nil
}

func toAppleScriptDefinitions(svc *ServiceManifest) ([]actions.ActionDefinition, error) {
	keys := sortedKeys(svc.Actions)
	out := make([]actions.ActionDefinition, 0, len(keys))
	mode := CurrentAppleScriptRegistryMode()
	for _, key := range keys {
		actionSpec := svc.Actions[key]
		commandName, scriptSource, scriptLanguage, scriptTimeout, approvalRef, auditRef, resolveErr := resolveAppleScriptExecutionSpec(actionSpec, mode)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve applescript execution spec for %s.%s: %w", svc.Name, key, resolveErr)
		}

		idempotent := resolveIdempotent(actionSpec, false)

		definition := actions.ActionDefinition{
			Name:         svc.Name + "." + key,
			Version:      1,
			DisplayName:  nonEmptyString(actionSpec.Description, key),
			Namespace:    svc.Name,
			Verb:         commandName,
			Resource:     svc.TargetApp,
			Description:  actionSpec.Description,
			Risk:         mapRisk(actionSpec.Risk.Level),
			Idempotent:   idempotent,
			ApprovalHint: mapApprovalHint(actionSpec.Risk.Level),
			Auth:         mapAuth(resolveActionAuth(svc.Auth, actionSpec.Auth)),
			InputSchema:  buildInputSchema(actionSpec.Args, nil, nil),
			OutputSchema: buildOutputSchema(actionSpec.Response),
			Defaults:     collectDefaults(actionSpec.Args),
			FilterConfig:    convertFilterSpec(actionSpec.Response.Filter),
			CompactTemplate: convertCompactSpec(actionSpec.Response.Compact),
			Adapter: actions.AdapterConfig{
				Type:           "applescript",
				TargetApp:      svc.TargetApp,
				Command:        commandName,
				ScriptSource:   scriptSource,
				ScriptLanguage: scriptLanguage,
				Timeout:        scriptTimeout,
				ApprovalRef:    approvalRef,
				AuditRef:       auditRef,
				RegistryMode:   string(mode),
			},
			Classifiers:  nil,
			ErrorMapping: nil,
			Pagination:   nil,
		}

		injectOutputModeParam(&definition)
		out = append(out, definition)
	}

	return out, nil
}

func resolveAppleScriptExecutionSpec(actionSpec ServiceAction, mode AppleScriptRegistryMode) (commandName string, scriptSource string, scriptLanguage string, timeout time.Duration, approvalRef string, auditRef string, err error) {
	legacyCommand := strings.TrimSpace(actionSpec.Command)

	if mode == AppleScriptRegistryModeLegacy {
		return legacyCommand, "", "", 0, "", "", nil
	}

	if actionSpec.InlineScript == nil {
		if mode == AppleScriptRegistryModeManifest {
			return "", "", "", 0, "", "", fmt.Errorf("inline_script is required in manifest mode")
		}
		return legacyCommand, "", "", 0, "", "", nil
	}

	inline := actionSpec.InlineScript
	inlineID := strings.TrimSpace(inline.ID)
	if inlineID == "" {
		if mode == AppleScriptRegistryModeManifest {
			return "", "", "", 0, "", "", fmt.Errorf("inline_script.id is required")
		}
		inlineID = legacyCommand
	}

	source := strings.TrimSpace(inline.Source)
	if source == "" {
		if mode == AppleScriptRegistryModeManifest {
			return "", "", "", 0, "", "", fmt.Errorf("inline_script.source is required")
		}
		return legacyCommand, "", "", 0, "", "", nil
	}

	language := normalizeInlineScriptLanguage(inline.Language)
	scriptTimeout := parseDurationOrZero(inline.Timeout)
	return inlineID, source, language, scriptTimeout, strings.TrimSpace(inline.ApprovalRef), strings.TrimSpace(inline.AuditRef), nil
}

func normalizeInlineScriptLanguage(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "jxa":
		return "jxa"
	case "applescript":
		return "applescript"
	default:
		return "jxa"
	}
}

func toCommandDefinitions(svc *ServiceManifest) ([]actions.ActionDefinition, error) {
	keys := sortedKeys(svc.Actions)
	out := make([]actions.ActionDefinition, 0, len(keys))

	executable := ""
	jsonFlag := ""
	timeout := time.Duration(0)
	var envInject map[string]string
	if svc.CommandSpec != nil {
		executable = strings.TrimSpace(svc.CommandSpec.Executable)
		jsonFlag = strings.TrimSpace(svc.CommandSpec.JSONFlag)
		timeout = parseDurationOrZero(svc.CommandSpec.Timeout)
		envInject = cloneStringMap(svc.CommandSpec.EnvInject)
	}

	for _, key := range keys {
		actionSpec := svc.Actions[key]

		idempotent := resolveIdempotent(actionSpec, false)

		definition := actions.ActionDefinition{
			Name:         svc.Name + "." + key,
			Version:      1,
			DisplayName:  nonEmptyString(actionSpec.Description, key),
			Namespace:    svc.Name,
			Verb:         actionSpec.Command,
			Resource:     executable,
			Description:  actionSpec.Description,
			Risk:         mapRisk(actionSpec.Risk.Level),
			Idempotent:   idempotent,
			ApprovalHint: mapApprovalHint(actionSpec.Risk.Level),
			Auth:         mapAuth(resolveActionAuth(svc.Auth, actionSpec.Auth)),
			InputSchema:  buildInputSchema(actionSpec.Args, nil, nil),
			OutputSchema: buildOutputSchema(actionSpec.Response),
			Defaults:     collectDefaults(actionSpec.Args),
			FilterConfig:    convertFilterSpec(actionSpec.Response.Filter),
			CompactTemplate: convertCompactSpec(actionSpec.Response.Compact),
			Adapter: actions.AdapterConfig{
				Type:           "command",
				ExecutablePath: executable,
				Command:        actionSpec.Command,
				JSONFlag:       jsonFlag,
				EnvInject:      cloneStringMap(envInject),
				Timeout:        timeout,
			},
			Classifiers:  nil,
			ErrorMapping: nil,
			Pagination:   nil,
		}

		injectOutputModeParam(&definition)
		out = append(out, definition)
	}

	return out, nil
}

func sortedKeys(m map[string]ServiceAction) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mapRisk(level string) actions.RiskLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "low":
		return actions.RiskRead
	case "medium":
		return actions.RiskWrite
	case "high":
		return actions.RiskAdmin
	default:
		return actions.RiskDestructive
	}
}

func mapApprovalHint(level string) actions.ApprovalHint {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "low":
		return actions.ApprovalNone
	case "medium":
		return actions.ApprovalOptional
	default:
		return actions.ApprovalRequired
	}
}

func resolveIdempotent(actionSpec ServiceAction, fallback bool) bool {
	if actionSpec.Idempotent != nil {
		return *actionSpec.Idempotent
	}
	if actionSpec.Risk.Mutating != nil {
		return !*actionSpec.Risk.Mutating
	}
	return fallback
}

func mapAuth(auth ServiceAuth) actions.AuthRequirement {
	req := actions.AuthRequirement{
		CredentialRef: auth.CredentialRef,
	}

	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "none":
		req.Type = actions.AuthTypeNone
		req.Optional = true
	case "header":
		req.Type = actions.AuthTypeHeader
		req.HeaderName = strings.TrimSpace(auth.HeaderName)
	case "bearer":
		req.Type = actions.AuthTypeBearer
		req.Prefix = "Bearer"
	case "basic":
		req.Type = actions.AuthTypeBasic
	case "query":
		req.Type = actions.AuthTypeQuery
		req.QueryName = strings.TrimSpace(auth.QueryParam)
	case "body":
		req.Type = actions.AuthTypeBody
		req.BodyField = strings.TrimSpace(auth.BodyField)
	default:
		req.Type = actions.AuthTypeNone
		req.Optional = true
	}

	return req
}

func resolveActionAuth(defaultAuth ServiceAuth, actionAuth *ServiceAuth) ServiceAuth {
	if actionAuth != nil {
		return *actionAuth
	}
	return defaultAuth
}

func buildInputSchema(args []ActionArg, pathParams map[string]string, page *PageSpec) *actions.Schema {
	if len(args) == 0 && len(pathParams) == 0 && page == nil {
		return &actions.Schema{Type: "object", AdditionalProperties: true}
	}

	properties := make(map[string]*actions.Schema, len(args)+len(pathParams)+1)
	required := make([]string, 0)
	requiredSet := make(map[string]struct{})

	for _, arg := range args {
		name := strings.TrimSpace(arg.Name)
		properties[name] = &actions.Schema{
			Type: strings.ToLower(strings.TrimSpace(arg.Type)),
			Enum: arg.Enum,
		}
		if arg.Required {
			if _, ok := requiredSet[name]; !ok {
				required = append(required, name)
				requiredSet[name] = struct{}{}
			}
		}
	}

	for name := range pathParams {
		if _, exists := properties[name]; !exists {
			properties[name] = &actions.Schema{Type: "string"}
		}
		if _, ok := requiredSet[name]; !ok {
			required = append(required, name)
			requiredSet[name] = struct{}{}
		}
	}

	if page != nil {
		properties["_max_pages"] = &actions.Schema{Type: "integer"}
	}

	sort.Strings(required)

	return &actions.Schema{
		Type:                 "object",
		Required:             required,
		Properties:           properties,
		AdditionalProperties: false,
	}
}

func collectDefaults(args []ActionArg) map[string]any {
	defaults := map[string]any{}
	for _, arg := range args {
		if arg.Required || arg.Default == nil {
			continue
		}
		if !isArgDefaultTypeCompatible(arg.Default, arg.Type) {
			continue
		}
		defaults[strings.TrimSpace(arg.Name)] = arg.Default
	}
	if len(defaults) == 0 {
		return nil
	}
	return defaults
}

func buildOutputSchema(resp ResponseSpec) *actions.Schema {
	t := strings.ToLower(strings.TrimSpace(resp.Type))
	if t == "array" {
		return &actions.Schema{Type: "array", Items: &actions.Schema{Type: "object"}}
	}
	return &actions.Schema{Type: "object", AdditionalProperties: true}
}

func mapPagination(page *PageSpec) *actions.PaginationConfig {
	if page == nil {
		return nil
	}
	cfg := &actions.PaginationConfig{
		Style:          strings.ToLower(page.Type),
		ResponseCursor: page.NextPath,
	}
	if page.MaxPages > 0 {
		cfg.MaxPages = page.MaxPages
	}
	return cfg
}

func isIdempotent(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

func has5xx(codes []int) bool {
	for _, code := range codes {
		if code >= 500 && code <= 599 {
			return true
		}
	}
	return false
}

func marshalBody(body map[string]any) string {
	if len(body) == 0 {
		return ""
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func parseDurationOrZero(raw string) time.Duration {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0
	}
	return d
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	return maps.Clone(in)
}

func cloneIntMap(in map[int]string) map[int]string {
	if len(in) == 0 {
		return nil
	}
	return maps.Clone(in)
}

func convertFilterSpec(spec *FilterSpec) *actions.FilterConfig {
	if spec == nil {
		return nil
	}
	return &actions.FilterConfig{
		Select:    spec.Select,
		Exclude:   spec.Exclude,
		MaxItems:  spec.MaxItems,
		DropNulls: spec.DropNulls,
	}
}

func injectOutputModeParam(def *actions.ActionDefinition) {
	if def == nil || def.FilterConfig == nil {
		return
	}
	if def.InputSchema == nil {
		def.InputSchema = &actions.Schema{Type: "object", Properties: map[string]*actions.Schema{}}
	}
	if def.InputSchema.Properties == nil {
		def.InputSchema.Properties = map[string]*actions.Schema{}
	}
	def.InputSchema.Properties["_output_mode"] = &actions.Schema{
		Type: "string",
		Enum: []any{"default", "raw"},
	}
	def.InputSchema.Properties["_budget"] = &actions.Schema{
		Type: "integer",
	}
}

// convertCompactSpec converts a manifest CompactSpec to a runtime CompactTemplate.
// Returns nil if spec is nil (backward compatible).
func convertCompactSpec(spec *CompactSpec) *actions.CompactTemplate {
	if spec == nil {
		return nil
	}
	return &actions.CompactTemplate{
		Header: spec.Header,
		Item:   spec.Item,
		Footer: spec.Footer,
	}
}

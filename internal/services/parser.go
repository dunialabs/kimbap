package services

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	adaptercommands "github.com/dunialabs/kimbap/internal/adapters/commands"
	"gopkg.in/yaml.v3"
)

var (
	serviceNamePattern       = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	actionKeyPattern         = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	semverLikePattern        = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)
	validRiskLevelSet        = map[string]struct{}{"low": {}, "medium": {}, "high": {}, "critical": {}}
	validAuthTypeSet         = map[string]struct{}{"header": {}, "bearer": {}, "basic": {}, "query": {}, "body": {}, "none": {}}
	validHTTPMethodSet       = map[string]struct{}{"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "HEAD": {}, "OPTIONS": {}}
	validAdapterTypeSet      = map[string]struct{}{"http": {}, "applescript": {}, "command": {}}
	validAppleScriptCommands = buildValidAppleScriptCommands()
	validArgTypeSet          = map[string]struct{}{"string": {}, "integer": {}, "number": {}, "boolean": {}, "array": {}, "object": {}}
	validPageTypeSet         = map[string]struct{}{"cursor": {}, "offset": {}}
	validResponseTypeSet     = map[string]struct{}{"object": {}, "array": {}}
	validGotchaSeveritySet   = map[string]struct{}{"low": {}, "medium": {}, "high": {}, "critical": {}, "common": {}, "rare": {}}
)

type ValidationError struct {
	Field   string
	Message string
}

func buildValidAppleScriptCommands() map[string]struct{} {
	allowed := make(map[string]struct{})
	registries := []map[string]adaptercommands.Command{
		adaptercommands.NotesCommands(),
		adaptercommands.CalendarCommands(),
		adaptercommands.RemindersCommands(),
		adaptercommands.MailCommands(),
		adaptercommands.FinderCommands(),
		adaptercommands.SafariCommands(),
		adaptercommands.MessagesCommands(),
		adaptercommands.ContactsCommands(),
		adaptercommands.MSOfficeCommands(),
		adaptercommands.IWorkCommands(),
		adaptercommands.SpotifyCommands(),
		adaptercommands.ShortcutsCommands(),
	}
	for _, registry := range registries {
		for name := range registry {
			allowed[name] = struct{}{}
		}
	}
	return allowed
}

func ParseManifest(data []byte) (*ServiceManifest, error) {
	var manifest ServiceManifest
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("parse service manifest: %w", err)
	}

	errs := ValidateManifest(&manifest)
	if len(errs) > 0 {
		return nil, validationErrorsToError("service manifest validation failed", errs)
	}

	return &manifest, nil
}

func ParseManifestFile(path string) (*ServiceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read service manifest file: %w", err)
	}
	return ParseManifest(data)
}

func ValidateManifest(m *ServiceManifest) []ValidationError {
	if m == nil {
		return []ValidationError{{Field: "manifest", Message: "manifest is required"}}
	}

	errs := make([]ValidationError, 0)

	if !serviceNamePattern.MatchString(m.Name) {
		errs = append(errs, ValidationError{Field: "name", Message: "must match [a-z][a-z0-9-]*"})
	}
	if err := ValidateServiceName(m.Name); err != nil {
		errs = append(errs, ValidationError{Field: "name", Message: err.Error()})
	}

	if !semverLikePattern.MatchString(m.Version) {
		errs = append(errs, ValidationError{Field: "version", Message: "must be semver-like (e.g. 1.2.3 or v1.2.3)"})
	}

	errs = append(errs, validateAuth(m.Auth, "auth")...)

	adapterType := normalizedAdapterType(m.Adapter)
	if _, ok := validAdapterTypeSet[adapterType]; !ok {
		errs = append(errs, ValidationError{Field: "adapter", Message: "must be one of http, applescript, command"})
		return errs
	}

	for actionKey, action := range m.Actions {
		prefix := "actions." + actionKey

		if !actionKeyPattern.MatchString(actionKey) {
			errs = append(errs, ValidationError{Field: prefix, Message: "action key must match [a-z][a-z0-9_-]*"})
		}

		risk := strings.ToLower(strings.TrimSpace(action.Risk.Level))
		if _, ok := validRiskLevelSet[risk]; !ok {
			errs = append(errs, ValidationError{Field: prefix + ".risk.level", Message: "must be one of low, medium, high, critical"})
		}

		if action.Response.Type != "" {
			if _, ok := validResponseTypeSet[strings.ToLower(action.Response.Type)]; !ok {
				errs = append(errs, ValidationError{Field: prefix + ".response.type", Message: "must be object or array"})
			}
		}

		seenArgNames := make(map[string]int, len(action.Args))
		for idx, arg := range action.Args {
			argField := fmt.Sprintf("%s.args[%d]", prefix, idx)
			argName := strings.TrimSpace(arg.Name)
			if argName == "" {
				errs = append(errs, ValidationError{Field: argField + ".name", Message: "is required"})
			} else if firstIdx, exists := seenArgNames[argName]; exists {
				errs = append(errs, ValidationError{Field: argField + ".name", Message: fmt.Sprintf("duplicates args[%d].name %q", firstIdx, argName)})
			} else {
				seenArgNames[argName] = idx
			}
			if _, ok := validArgTypeSet[strings.ToLower(strings.TrimSpace(arg.Type))]; !ok {
				errs = append(errs, ValidationError{Field: argField + ".type", Message: "must be one of string, integer, number, boolean, array, object"})
			}
			if arg.Required && arg.Default != nil {
				errs = append(errs, ValidationError{Field: argField + ".default", Message: "required args must not have defaults"})
			}
			if !arg.Required && arg.Default != nil {
				if !isArgDefaultTypeCompatible(arg.Default, arg.Type) {
					errs = append(errs, ValidationError{
						Field:   argField + ".default",
						Message: fmt.Sprintf("default value type does not match declared arg type %q", arg.Type),
					})
				}
			}
		}

		if action.Auth != nil {
			errs = append(errs, validateAuth(*action.Auth, prefix+".auth")...)
		}

		for warningIdx, warning := range action.Warnings {
			if strings.TrimSpace(warning) == "" {
				errs = append(errs, ValidationError{Field: fmt.Sprintf("%s.warnings[%d]", prefix, warningIdx), Message: "must be non-empty"})
			}
		}
	}

	errs = append(errs, validatePackMetadata(m)...)

	switch adapterType {
	case "http":
		errs = append(errs, validateHTTPManifest(m)...)
	case "applescript":
		errs = append(errs, validateAppleScriptManifest(m)...)
	case "command":
		errs = append(errs, validateCommandManifest(m)...)
	}

	if len(m.Actions) == 0 {
		errs = append(errs, ValidationError{Field: "actions", Message: "must define at least one action"})
	}

	return errs
}

func isArgDefaultTypeCompatible(v any, declaredType string) bool {
	switch strings.ToLower(strings.TrimSpace(declaredType)) {
	case "string":
		_, ok := v.(string)
		return ok
	case "integer":
		switch val := v.(type) {
		case int, int8, int16, int32, int64:
			return true
		case uint, uint8, uint16, uint32, uint64:
			return true
		case float64:
			return val == math.Trunc(val)
		default:
			return false
		}
	case "number":
		switch v.(type) {
		case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "object":
		_, ok := v.(map[string]any)
		return ok
	default:
		return true
	}
}

func validatePackMetadata(m *ServiceManifest) []ValidationError {
	errList := make([]ValidationError, 0)

	if m.Triggers != nil {
		if len(m.Triggers.TaskVerbs) == 0 {
			errList = append(errList, ValidationError{Field: "triggers.task_verbs", Message: "must define at least one task verb"})
		}
		for idx, v := range m.Triggers.TaskVerbs {
			if strings.TrimSpace(v) == "" {
				errList = append(errList, ValidationError{Field: fmt.Sprintf("triggers.task_verbs[%d]", idx), Message: "must be non-empty"})
			}
		}
		if len(m.Triggers.Objects) == 0 {
			errList = append(errList, ValidationError{Field: "triggers.objects", Message: "must define at least one object"})
		}
		for idx, o := range m.Triggers.Objects {
			if strings.TrimSpace(o) == "" {
				errList = append(errList, ValidationError{Field: fmt.Sprintf("triggers.objects[%d]", idx), Message: "must be non-empty"})
			}
		}
		for idx, it := range m.Triggers.InsteadOf {
			if strings.TrimSpace(it) == "" {
				errList = append(errList, ValidationError{Field: fmt.Sprintf("triggers.instead_of[%d]", idx), Message: "must be non-empty"})
			}
		}
		for idx, ex := range m.Triggers.Exclusions {
			if strings.TrimSpace(ex) == "" {
				errList = append(errList, ValidationError{Field: fmt.Sprintf("triggers.exclusions[%d]", idx), Message: "must be non-empty"})
			}
		}
	}

	for idx, g := range m.Gotchas {
		prefix := fmt.Sprintf("gotchas[%d]", idx)
		if strings.TrimSpace(g.Symptom) == "" {
			errList = append(errList, ValidationError{Field: prefix + ".symptom", Message: "is required"})
		}
		if strings.TrimSpace(g.LikelyCause) == "" {
			errList = append(errList, ValidationError{Field: prefix + ".likely_cause", Message: "is required"})
		}
		if strings.TrimSpace(g.Recovery) == "" {
			errList = append(errList, ValidationError{Field: prefix + ".recovery", Message: "is required"})
		}
		sev := strings.ToLower(strings.TrimSpace(g.Severity))
		if sev != "" {
			if _, ok := validGotchaSeveritySet[sev]; !ok {
				errList = append(errList, ValidationError{Field: prefix + ".severity", Message: "must be one of low, medium, high, critical, common, rare"})
			}
		}
	}

	for idx, r := range m.Recipes {
		prefix := fmt.Sprintf("recipes[%d]", idx)
		if strings.TrimSpace(r.Name) == "" {
			errList = append(errList, ValidationError{Field: prefix + ".name", Message: "is required"})
		}
		if len(r.Steps) == 0 {
			errList = append(errList, ValidationError{Field: prefix + ".steps", Message: "must include at least one step"})
		}
		for stepIdx, step := range r.Steps {
			if strings.TrimSpace(step) == "" {
				errList = append(errList, ValidationError{Field: fmt.Sprintf("%s.steps[%d]", prefix, stepIdx), Message: "must be non-empty"})
			}
		}
	}

	return errList
}

func validateHTTPManifest(m *ServiceManifest) []ValidationError {
	errs := make([]ValidationError, 0)

	if strings.TrimSpace(m.TargetApp) != "" {
		errs = append(errs, ValidationError{Field: "target_app", Message: "must not be set for http adapter"})
	}

	if m.BaseURL == "" {
		errs = append(errs, ValidationError{Field: "base_url", Message: "must be set"})
	} else if u, err := url.Parse(m.BaseURL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, ValidationError{Field: "base_url", Message: "must be a valid absolute URL"})
	} else if u.Scheme != "http" && u.Scheme != "https" {
		errs = append(errs, ValidationError{Field: "base_url", Message: "scheme must be http or https"})
	} else if u.Scheme == "http" && !isLoopbackHostname(u.Hostname()) {
		errs = append(errs, ValidationError{Field: "base_url", Message: "http is only allowed for localhost/loopback; use https for remote hosts"})
	} else if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(m.BaseURL, "#") {
		errs = append(errs, ValidationError{Field: "base_url", Message: "must not contain query string or fragment"})
	}

	for actionKey, action := range m.Actions {
		prefix := "actions." + actionKey

		if strings.TrimSpace(action.Method) == "" {
			errs = append(errs, ValidationError{Field: prefix + ".method", Message: "is required"})
		} else if _, ok := validHTTPMethodSet[strings.ToUpper(action.Method)]; !ok {
			errs = append(errs, ValidationError{Field: prefix + ".method", Message: "must be a valid HTTP method"})
		}

		if strings.TrimSpace(action.Path) == "" {
			errs = append(errs, ValidationError{Field: prefix + ".path", Message: "is required"})
		}

		if action.Pagination != nil {
			if _, ok := validPageTypeSet[strings.ToLower(strings.TrimSpace(action.Pagination.Type))]; !ok {
				errs = append(errs, ValidationError{Field: prefix + ".pagination.type", Message: "must be cursor or offset"})
			}
			if action.Pagination.MaxPages < 0 {
				errs = append(errs, ValidationError{Field: prefix + ".pagination.max_pages", Message: "must be non-negative"})
			}
		}

		trimmedPath := strings.TrimSpace(action.Path)
		if strings.Contains(trimmedPath, "://") {
			errs = append(errs, ValidationError{Field: prefix + ".path", Message: "must be a relative path, not an absolute URL"})
		}
		if strings.HasPrefix(trimmedPath, "//") {
			errs = append(errs, ValidationError{Field: prefix + ".path", Message: "must not be a protocol-relative URL"})
		}

		pathTemplateRefs := extractTemplateRefs(action.Path)
		pathTemplateRefSet := make(map[string]struct{}, len(pathTemplateRefs))
		for _, ref := range pathTemplateRefs {
			pathTemplateRefSet[ref] = struct{}{}
		}

		requiredArgs := make(map[string]struct{}, len(action.Args))
		declaredArgs := make(map[string]struct{}, len(action.Args)+len(action.Request.PathParams))
		for _, arg := range action.Args {
			declaredArgs[arg.Name] = struct{}{}
			if arg.Required {
				requiredArgs[arg.Name] = struct{}{}
			}
		}
		for paramName := range action.Request.PathParams {
			declaredArgs[paramName] = struct{}{}
			if _, ok := pathTemplateRefSet[paramName]; !ok {
				errs = append(errs, ValidationError{Field: prefix + ".request.path_params", Message: fmt.Sprintf("declares unused path param %q", paramName)})
			}
		}
		for _, ref := range pathTemplateRefs {
			if _, isPathParam := action.Request.PathParams[ref]; isPathParam {
				continue
			}
			if _, isRequired := requiredArgs[ref]; !isRequired {
				errs = append(errs, ValidationError{
					Field:   prefix + ".path",
					Message: fmt.Sprintf("path template {%s} must reference a required arg", ref),
				})
			}
		}
		templateRefs := append([]string(nil), pathTemplateRefs...)
		for k, v := range action.Request.Query {
			templateRefs = append(templateRefs, extractTemplateRefs(k)...)
			templateRefs = append(templateRefs, extractTemplateRefs(v)...)
		}
		for k, v := range action.Request.Headers {
			templateRefs = append(templateRefs, extractTemplateRefs(k)...)
			templateRefs = append(templateRefs, extractTemplateRefs(v)...)
		}
		templateRefs = append(templateRefs, extractBodyTemplateRefs(action.Request.Body)...)
		for _, ref := range templateRefs {
			if _, ok := declaredArgs[ref]; !ok {
				errs = append(errs, ValidationError{
					Field:   prefix + ".template_ref",
					Message: fmt.Sprintf("references undeclared arg %q", ref),
				})
			}
		}
	}

	return errs
}

func validateAppleScriptManifest(m *ServiceManifest) []ValidationError {
	errs := make([]ValidationError, 0)

	if strings.TrimSpace(m.TargetApp) == "" {
		errs = append(errs, ValidationError{Field: "target_app", Message: "must be set for applescript adapter"})
	}
	if strings.TrimSpace(m.BaseURL) != "" {
		errs = append(errs, ValidationError{Field: "base_url", Message: "must not be set for applescript adapter"})
	}
	if normalizedAuthType(m.Auth.Type) != "none" {
		errs = append(errs, ValidationError{Field: "auth.type", Message: "must be none for applescript adapter"})
	}

	for actionKey, action := range m.Actions {
		prefix := "actions." + actionKey

		if strings.TrimSpace(action.Command) == "" {
			errs = append(errs, ValidationError{Field: prefix + ".command", Message: "is required"})
		} else if _, ok := validAppleScriptCommands[strings.ToLower(strings.TrimSpace(action.Command))]; !ok {
			errs = append(errs, ValidationError{Field: prefix + ".command", Message: "must be a supported applescript command"})
		}

		if strings.TrimSpace(action.Method) != "" {
			errs = append(errs, ValidationError{Field: prefix + ".method", Message: "must not be set for applescript adapter"})
		}
		if strings.TrimSpace(action.Path) != "" {
			errs = append(errs, ValidationError{Field: prefix + ".path", Message: "must not be set for applescript adapter"})
		}
		if action.Auth != nil && normalizedAuthType(action.Auth.Type) != "none" {
			errs = append(errs, ValidationError{Field: prefix + ".auth.type", Message: "must be none for applescript adapter"})
		}

		if len(action.Request.Query) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".request.query", Message: "must not be set for applescript adapter"})
		}
		if len(action.Request.Headers) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".request.headers", Message: "must not be set for applescript adapter"})
		}
		if len(action.Request.Body) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".request.body", Message: "must not be set for applescript adapter"})
		}
		if len(action.Request.PathParams) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".request.path_params", Message: "must not be set for applescript adapter"})
		}

		if action.Retry != nil {
			errs = append(errs, ValidationError{Field: prefix + ".retry", Message: "must not be set for applescript adapter"})
		}
		if action.Pagination != nil {
			errs = append(errs, ValidationError{Field: prefix + ".pagination", Message: "must not be set for applescript adapter"})
		}
		if len(action.ErrorMapping) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".error_mapping", Message: "must not be set for applescript adapter"})
		}
	}

	return errs
}

func validateCommandManifest(m *ServiceManifest) []ValidationError {
	errs := make([]ValidationError, 0)

	if strings.TrimSpace(m.BaseURL) != "" {
		errs = append(errs, ValidationError{Field: "base_url", Message: "must not be set for command adapter"})
	}
	if strings.TrimSpace(m.TargetApp) != "" {
		errs = append(errs, ValidationError{Field: "target_app", Message: "must not be set for command adapter"})
	}

	authType := normalizedAuthType(m.Auth.Type)
	if authType != "none" && authType != "bearer" {
		errs = append(errs, ValidationError{Field: "auth.type", Message: "must be none or bearer for command adapter"})
	}

	if m.CommandSpec == nil {
		errs = append(errs, ValidationError{Field: "command_spec", Message: "must be set for command adapter"})
	} else {
		if strings.TrimSpace(m.CommandSpec.Executable) == "" {
			errs = append(errs, ValidationError{Field: "command_spec.executable", Message: "must be non-empty"})
		}
		if timeout := strings.TrimSpace(m.CommandSpec.Timeout); timeout != "" {
			if _, err := time.ParseDuration(timeout); err != nil {
				errs = append(errs, ValidationError{Field: "command_spec.timeout", Message: "must be a valid Go duration (e.g. 30s, 1m)"})
			}
		}
	}

	for actionKey, action := range m.Actions {
		prefix := "actions." + actionKey

		if strings.TrimSpace(action.Command) == "" {
			errs = append(errs, ValidationError{Field: prefix + ".command", Message: "is required"})
		}

		if strings.TrimSpace(action.Method) != "" {
			errs = append(errs, ValidationError{Field: prefix + ".method", Message: "must not be set for command adapter"})
		}
		if strings.TrimSpace(action.Path) != "" {
			errs = append(errs, ValidationError{Field: prefix + ".path", Message: "must not be set for command adapter"})
		}

		if len(action.Request.Query) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".request.query", Message: "must not be set for command adapter"})
		}
		if len(action.Request.Headers) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".request.headers", Message: "must not be set for command adapter"})
		}
		if len(action.Request.Body) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".request.body", Message: "must not be set for command adapter"})
		}
		if len(action.Request.PathParams) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".request.path_params", Message: "must not be set for command adapter"})
		}

		if action.Pagination != nil {
			errs = append(errs, ValidationError{Field: prefix + ".pagination", Message: "must not be set for command adapter"})
		}
		if action.Retry != nil {
			errs = append(errs, ValidationError{Field: prefix + ".retry", Message: "must not be set for command adapter"})
		}
		if len(action.ErrorMapping) > 0 {
			errs = append(errs, ValidationError{Field: prefix + ".error_mapping", Message: "must not be set for command adapter"})
		}

		if action.Auth != nil {
			actionAuthType := normalizedAuthType(action.Auth.Type)
			if actionAuthType != "none" && actionAuthType != "bearer" {
				errs = append(errs, ValidationError{Field: prefix + ".auth.type", Message: "must be none or bearer for command adapter"})
			}
		}
	}

	return errs
}

func isLoopbackHostname(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateAuth(auth ServiceAuth, fieldPrefix string) []ValidationError {
	var errs []ValidationError
	authType := strings.ToLower(strings.TrimSpace(auth.Type))
	if _, ok := validAuthTypeSet[authType]; !ok {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix + ".type",
			Message: "must be one of none, header, bearer, basic, query, body",
		})
		return errs
	}
	if authType != "none" && strings.TrimSpace(auth.CredentialRef) == "" {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix + ".credential_ref",
			Message: "must be non-empty when auth type is not none",
		})
	}
	if authType == "header" && strings.TrimSpace(auth.HeaderName) == "" {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix + ".header_name",
			Message: "must be set when auth type is header",
		})
	}
	if authType == "query" && strings.TrimSpace(auth.QueryParam) == "" {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix + ".query_param",
			Message: "must be set when auth type is query",
		})
	}
	if authType == "body" && strings.TrimSpace(auth.BodyField) == "" {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix + ".body_field",
			Message: "must be set when auth type is body",
		})
	}
	return errs
}

var templateRefPattern = regexp.MustCompile(`\{([^{}\s]+)\}`)

func extractTemplateRefs(s string) []string {
	matches := templateRefPattern.FindAllStringSubmatch(s, -1)
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		if match[1] != "" {
			refs = append(refs, match[1])
		}
	}
	return refs
}

func extractBodyTemplateRefs(body map[string]any) []string {
	var refs []string
	for _, v := range body {
		refs = append(refs, extractBodyItemRefs(v)...)
	}
	return refs
}

func extractBodyItemRefs(v any) []string {
	switch val := v.(type) {
	case string:
		return extractTemplateRefs(val)
	case map[string]any:
		return extractBodyTemplateRefs(val)
	case []any:
		var refs []string
		for _, item := range val {
			refs = append(refs, extractBodyItemRefs(item)...)
		}
		return refs
	default:
		return nil
	}
}

func validationErrorsToError(prefix string, errs []ValidationError) error {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return fmt.Errorf("%s: %s", prefix, strings.Join(parts, "; "))
}

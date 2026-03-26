package services

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	skillNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	actionKeyPattern     = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	semverLikePattern    = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)
	validRiskLevelSet    = map[string]struct{}{"low": {}, "medium": {}, "high": {}, "critical": {}}
	validAuthTypeSet     = map[string]struct{}{"header": {}, "bearer": {}, "basic": {}, "query": {}, "body": {}, "none": {}}
	validHTTPMethodSet   = map[string]struct{}{"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "HEAD": {}, "OPTIONS": {}}
	validArgTypeSet      = map[string]struct{}{"string": {}, "integer": {}, "number": {}, "boolean": {}, "array": {}, "object": {}}
	validPageTypeSet     = map[string]struct{}{"cursor": {}, "offset": {}}
	validResponseTypeSet = map[string]struct{}{"object": {}, "array": {}}
)

type ValidationError struct {
	Field   string
	Message string
}

func ParseManifest(data []byte) (*ServiceManifest, error) {
	var manifest ServiceManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse skill manifest: %w", err)
	}

	errs := ValidateManifest(&manifest)
	if len(errs) > 0 {
		return nil, validationErrorsToError("skill manifest validation failed", errs)
	}

	return &manifest, nil
}

func ParseManifestFile(path string) (*ServiceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill manifest file: %w", err)
	}
	return ParseManifest(data)
}

func ValidateManifest(m *ServiceManifest) []ValidationError {
	if m == nil {
		return []ValidationError{{Field: "manifest", Message: "manifest is required"}}
	}

	errs := make([]ValidationError, 0)

	if !skillNamePattern.MatchString(m.Name) {
		errs = append(errs, ValidationError{Field: "name", Message: "must match [a-z][a-z0-9-]*"})
	}

	if !semverLikePattern.MatchString(m.Version) {
		errs = append(errs, ValidationError{Field: "version", Message: "must be semver-like (e.g. 1.2.3 or v1.2.3)"})
	}

	if m.BaseURL == "" {
		errs = append(errs, ValidationError{Field: "base_url", Message: "must be set"})
	} else if u, err := url.Parse(m.BaseURL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, ValidationError{Field: "base_url", Message: "must be a valid absolute URL"})
	}

	authType := strings.ToLower(strings.TrimSpace(m.Auth.Type))
	if _, ok := validAuthTypeSet[authType]; !ok {
		errs = append(errs, ValidationError{Field: "auth.type", Message: "must be one of none, header, bearer, basic, query, body"})
	}
	if authType != "none" && strings.TrimSpace(m.Auth.CredentialRef) == "" {
		errs = append(errs, ValidationError{Field: "auth.credential_ref", Message: "must be non-empty when auth type is not none"})
	}

	for actionKey, action := range m.Actions {
		prefix := "actions." + actionKey

		if !actionKeyPattern.MatchString(actionKey) {
			errs = append(errs, ValidationError{Field: prefix, Message: "action key must match [a-z][a-z0-9_-]*"})
		}

		if strings.TrimSpace(action.Method) == "" {
			errs = append(errs, ValidationError{Field: prefix + ".method", Message: "is required"})
		} else if _, ok := validHTTPMethodSet[strings.ToUpper(action.Method)]; !ok {
			errs = append(errs, ValidationError{Field: prefix + ".method", Message: "must be a valid HTTP method"})
		}

		if strings.TrimSpace(action.Path) == "" {
			errs = append(errs, ValidationError{Field: prefix + ".path", Message: "is required"})
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

		for idx, arg := range action.Args {
			argField := fmt.Sprintf("%s.args[%d]", prefix, idx)
			if strings.TrimSpace(arg.Name) == "" {
				errs = append(errs, ValidationError{Field: argField + ".name", Message: "is required"})
			}
			if _, ok := validArgTypeSet[strings.ToLower(strings.TrimSpace(arg.Type))]; !ok {
				errs = append(errs, ValidationError{Field: argField + ".type", Message: "must be one of string, integer, number, boolean, array, object"})
			}
			if arg.Required && arg.Default != nil {
				errs = append(errs, ValidationError{Field: argField + ".default", Message: "required args must not have defaults"})
			}
		}

		if action.Pagination != nil {
			if _, ok := validPageTypeSet[strings.ToLower(strings.TrimSpace(action.Pagination.Type))]; !ok {
				errs = append(errs, ValidationError{Field: prefix + ".pagination.type", Message: "must be cursor or offset"})
			}
			if action.Pagination.MaxPages < 0 {
				errs = append(errs, ValidationError{Field: prefix + ".pagination.max_pages", Message: "must be non-negative"})
			}
		}
	}

	if len(m.Actions) == 0 {
		errs = append(errs, ValidationError{Field: "actions", Message: "must define at least one action"})
	}

	for actionKey, action := range m.Actions {
		prefix := "actions." + actionKey

		trimmedPath := strings.TrimSpace(action.Path)
		if strings.Contains(trimmedPath, "://") {
			errs = append(errs, ValidationError{Field: prefix + ".path", Message: "must be a relative path, not an absolute URL"})
		}
		if strings.HasPrefix(trimmedPath, "//") {
			errs = append(errs, ValidationError{Field: prefix + ".path", Message: "must not be a protocol-relative URL"})
		}

		declaredArgs := make(map[string]struct{}, len(action.Args)+len(action.Request.PathParams))
		for _, arg := range action.Args {
			declaredArgs[arg.Name] = struct{}{}
		}
		for paramName := range action.Request.PathParams {
			declaredArgs[paramName] = struct{}{}
		}
		templateRefs := extractTemplateRefs(action.Path)
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

var templateRefPattern = regexp.MustCompile(`\{([a-zA-Z_]\w*)\}`)

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
		switch val := v.(type) {
		case string:
			refs = append(refs, extractTemplateRefs(val)...)
		case map[string]any:
			refs = append(refs, extractBodyTemplateRefs(val)...)
		case []any:
			for _, item := range val {
				if s, ok := item.(string); ok {
					refs = append(refs, extractTemplateRefs(s)...)
				}
				if m, ok := item.(map[string]any); ok {
					refs = append(refs, extractBodyTemplateRefs(m)...)
				}
			}
		}
	}
	return refs
}

func validationErrorsToError(prefix string, errs []ValidationError) error {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return fmt.Errorf("%s: %s", prefix, strings.Join(parts, "; "))
}

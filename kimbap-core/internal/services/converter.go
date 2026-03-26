package services

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/actions"
)

func ToActionDefinitions(skill *ServiceManifest) ([]actions.ActionDefinition, error) {
	if skill == nil {
		return nil, fmt.Errorf("service manifest is nil")
	}
	if errs := ValidateManifest(skill); len(errs) > 0 {
		return nil, validationErrorsToError("invalid service manifest", errs)
	}

	adapterType := strings.ToLower(strings.TrimSpace(skill.Adapter))
	if adapterType == "" {
		adapterType = "http"
	}

	switch adapterType {
	case "applescript":
		return toAppleScriptDefinitions(skill)
	default:
		return toHTTPDefinitions(skill)
	}
}

func toHTTPDefinitions(skill *ServiceManifest) ([]actions.ActionDefinition, error) {
	u, err := url.Parse(skill.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	keys := sortedKeys(skill.Actions)

	out := make([]actions.ActionDefinition, 0, len(keys))
	for _, key := range keys {
		actionSpec := skill.Actions[key]
		retry := actions.RetryConfig{}
		if actionSpec.Retry != nil {
			retry.MaxAttempts = actionSpec.Retry.MaxAttempts
			retry.BackoffMS = actionSpec.Retry.BackoffMS
			retry.RetryOn429 = slices.Contains(actionSpec.Retry.RetryOn, 429)
			retry.RetryOn5xx = has5xx(actionSpec.Retry.RetryOn)
		}

		definition := actions.ActionDefinition{
			Name:         skill.Name + "." + key,
			Version:      1,
			DisplayName:  nonEmpty(actionSpec.Description, key),
			Namespace:    skill.Name,
			Verb:         strings.ToLower(actionSpec.Method),
			Resource:     actionSpec.Path,
			Description:  actionSpec.Description,
			Risk:         mapRisk(actionSpec.Risk.Level),
			Idempotent:   isIdempotent(actionSpec.Method),
			ApprovalHint: mapApprovalHint(actionSpec.Risk.Level),
			Auth:         mapAuth(resolveActionAuth(skill.Auth, actionSpec.Auth)),
			InputSchema:  buildInputSchema(actionSpec.Args, actionSpec.Request.PathParams),
			OutputSchema: buildOutputSchema(actionSpec.Response),
			Adapter: actions.AdapterConfig{
				Type:        "http",
				Method:      strings.ToUpper(actionSpec.Method),
				BaseURL:     skill.BaseURL,
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
				Service:     skill.Name,
				Action:      key,
				Method:      strings.ToUpper(actionSpec.Method),
				PathPattern: actionSpec.Path,
				HostPattern: u.Host,
			}},
			ErrorMapping: cloneIntMap(actionSpec.ErrorMapping),
			Pagination:   mapPagination(actionSpec.Pagination),
		}

		out = append(out, definition)
	}

	return out, nil
}

func toAppleScriptDefinitions(skill *ServiceManifest) ([]actions.ActionDefinition, error) {
	keys := sortedKeys(skill.Actions)
	out := make([]actions.ActionDefinition, 0, len(keys))
	for _, key := range keys {
		actionSpec := skill.Actions[key]

		idempotent := false
		if actionSpec.Idempotent != nil {
			idempotent = *actionSpec.Idempotent
		}

		definition := actions.ActionDefinition{
			Name:         skill.Name + "." + key,
			Version:      1,
			DisplayName:  nonEmpty(actionSpec.Description, key),
			Namespace:    skill.Name,
			Verb:         actionSpec.Command,
			Resource:     skill.TargetApp,
			Description:  actionSpec.Description,
			Risk:         mapRisk(actionSpec.Risk.Level),
			Idempotent:   idempotent,
			ApprovalHint: mapApprovalHint(actionSpec.Risk.Level),
			Auth:         mapAuth(resolveActionAuth(skill.Auth, actionSpec.Auth)),
			InputSchema:  buildInputSchema(actionSpec.Args, nil),
			OutputSchema: buildOutputSchema(actionSpec.Response),
			Adapter: actions.AdapterConfig{
				Type:      "applescript",
				TargetApp: skill.TargetApp,
				Command:   actionSpec.Command,
			},
			Classifiers:  nil,
			ErrorMapping: nil,
			Pagination:   nil,
		}

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
		req.HeaderName = auth.HeaderName
	case "bearer":
		req.Type = actions.AuthTypeBearer
		req.Prefix = "Bearer"
	case "basic":
		req.Type = actions.AuthTypeBasic
	case "query":
		req.Type = actions.AuthTypeQuery
		req.QueryName = auth.QueryParam
	case "body":
		req.Type = actions.AuthTypeBody
		req.BodyField = auth.BodyField
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

func buildInputSchema(args []ActionArg, pathParams map[string]string) *actions.Schema {
	if len(args) == 0 && len(pathParams) == 0 {
		return &actions.Schema{Type: "object", AdditionalProperties: true}
	}

	properties := make(map[string]*actions.Schema, len(args)+len(pathParams))
	required := make([]string, 0)
	requiredSet := make(map[string]struct{})

	for _, arg := range args {
		properties[arg.Name] = &actions.Schema{
			Type: strings.ToLower(arg.Type),
			Enum: arg.Enum,
		}
		if arg.Required {
			if _, ok := requiredSet[arg.Name]; !ok {
				required = append(required, arg.Name)
				requiredSet[arg.Name] = struct{}{}
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

	sort.Strings(required)

	return &actions.Schema{
		Type:                 "object",
		Required:             required,
		Properties:           properties,
		AdditionalProperties: true,
	}
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

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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

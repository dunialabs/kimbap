package skills

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	openAPIVersionPattern = regexp.MustCompile(`^(?:v)?(\d+)(?:\.(\d+))?(?:\.(\d+))?`)
	nonAlphanumPattern    = regexp.MustCompile(`[^a-z0-9]+`)
	multiDashPattern      = regexp.MustCompile(`-+`)
)

const (
	schemaOpaqueKey   = "x-kimbap-opaque"
	schemaWarningsKey = "x-kimbap-warnings"
)

func GenerateFromOpenAPI(spec []byte) (*SkillManifest, error) {
	root, err := parseOpenAPIRoot(spec)
	if err != nil {
		return nil, err
	}

	openapiVersion := strings.TrimSpace(stringAt(root, "openapi"))
	if openapiVersion == "" || !strings.HasPrefix(openapiVersion, "3") {
		return nil, fmt.Errorf("unsupported OpenAPI version %q: only OpenAPI 3.x is supported", openapiVersion)
	}

	resolver := &openAPIRefResolver{root: root}

	info := mapAt(root, "info")
	name := normalizeSkillName(stringAt(info, "title"))
	version := normalizeVersion(stringAt(info, "version"))
	description := strings.TrimSpace(stringAt(info, "description"))
	baseURL := normalizeBaseURL(firstServerURL(root))

	auth, err := extractAuth(root, resolver, name)
	if err != nil {
		return nil, err
	}

	actions, err := extractActions(root, resolver, name)
	if err != nil {
		return nil, err
	}

	manifest := &SkillManifest{
		Name:        name,
		Version:     version,
		Description: description,
		BaseURL:     baseURL,
		Auth:        auth,
		Actions:     actions,
	}

	if errs := ValidateManifest(manifest); len(errs) > 0 {
		return nil, validationErrorsToError("generated manifest is invalid", errs)
	}

	return manifest, nil
}

func GenerateFromOpenAPIFile(path string) (*SkillManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI file: %w", err)
	}
	return GenerateFromOpenAPI(data)
}

const maxOpenAPISpecBytes int64 = 10 << 20

func GenerateFromOpenAPIURL(rawURL string) (*SkillManifest, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create OpenAPI request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch OpenAPI URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch OpenAPI URL: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOpenAPISpecBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI response: %w", err)
	}
	if int64(len(body)) > maxOpenAPISpecBytes {
		return nil, fmt.Errorf("OpenAPI spec exceeds maximum size of %d bytes", maxOpenAPISpecBytes)
	}

	return GenerateFromOpenAPI(body)
}

type openAPIRefResolver struct {
	root map[string]any
}

func (r *openAPIRefResolver) resolveMap(in map[string]any) (map[string]any, error) {
	if in == nil {
		return nil, nil
	}

	out := in
	for range 8 {
		ref := strings.TrimSpace(stringAt(out, "$ref"))
		if ref == "" {
			return out, nil
		}

		resolvedAny, err := r.resolveRef(ref)
		if err != nil {
			return nil, err
		}

		resolvedMap, ok := resolvedAny.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("$ref %q did not resolve to an object", ref)
		}

		merged := cloneAnyMap(resolvedMap)
		for k, v := range out {
			if k == "$ref" {
				continue
			}
			merged[k] = v
		}
		out = merged
	}

	return nil, fmt.Errorf("$ref resolution depth exceeded")
}

func (r *openAPIRefResolver) resolveRef(ref string) (any, error) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, fmt.Errorf("unsupported $ref %q: only local refs are supported", ref)
	}

	current := any(r.root)
	for rawPart := range strings.SplitSeq(strings.TrimPrefix(ref, "#/"), "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(rawPart, "~1", "/"), "~0", "~")
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid $ref %q", ref)
		}
		next, ok := asMap[part]
		if !ok {
			return nil, fmt.Errorf("unresolvable $ref %q", ref)
		}
		current = next
	}

	return current, nil
}

func parseOpenAPIRoot(spec []byte) (map[string]any, error) {
	root := make(map[string]any)
	if json.Valid(spec) {
		if err := json.Unmarshal(spec, &root); err != nil {
			return nil, fmt.Errorf("parse OpenAPI JSON: %w", err)
		}
		return root, nil
	}

	var raw any
	if err := yaml.Unmarshal(spec, &raw); err != nil {
		return nil, fmt.Errorf("parse OpenAPI YAML: %w", err)
	}
	normalized := normalizeYAMLValue(raw)
	mapRoot, ok := normalized.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("OpenAPI spec must be a top-level object")
	}
	return mapRoot, nil
}

func extractAuth(root map[string]any, resolver *openAPIRefResolver, skillName string) (SkillAuth, error) {
	components := mapAt(root, "components")
	schemes := mapAt(components, "securitySchemes")
	if len(schemes) == 0 {
		return SkillAuth{Type: "none"}, nil
	}

	schemeName := firstReferencedSecurityScheme(root)
	if schemeName == "" {
		keys := sortedMapKeys(schemes)
		schemeName = keys[0]
	}

	rawScheme := mapAt(schemes, schemeName)
	if len(rawScheme) == 0 {
		return SkillAuth{Type: "none"}, nil
	}

	scheme, err := resolver.resolveMap(rawScheme)
	if err != nil {
		return SkillAuth{}, fmt.Errorf("resolve security scheme %q: %w", schemeName, err)
	}

	auth, err := authFromScheme(scheme, skillName)
	if err != nil {
		return SkillAuth{}, err
	}

	if auth.Type == "none" {
		return SkillAuth{Type: "none"}, nil
	}

	if strings.TrimSpace(auth.CredentialRef) == "" {
		auth.CredentialRef = credentialRef(skillName, "credential")
	}

	return auth, nil
}

func extractActions(root map[string]any, resolver *openAPIRefResolver, skillName string) (map[string]SkillAction, error) {
	paths := mapAt(root, "paths")
	actions := make(map[string]SkillAction)

	pathKeys := sortedMapKeys(paths)
	for _, p := range pathKeys {
		pathItem, err := resolver.resolveMap(mapAt(paths, p))
		if err != nil {
			return nil, fmt.Errorf("resolve path item %q: %w", p, err)
		}

		pathParams := anySliceAt(pathItem, "parameters")
		for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
			rawOp := mapAt(pathItem, method)
			if len(rawOp) == 0 {
				continue
			}

			op, err := resolver.resolveMap(rawOp)
			if err != nil {
				return nil, fmt.Errorf("resolve operation %s %s: %w", strings.ToUpper(method), p, err)
			}

			action, err := buildAction(strings.ToUpper(method), p, op, pathParams, root, resolver, skillName)
			if err != nil {
				return nil, err
			}

			key := normalizeActionKey(stringAt(op, "operationId"), method, p)
			key = ensureUniqueActionKey(key, method, p, actions)
			actions[key] = action
		}
	}

	return actions, nil
}

func buildAction(method, path string, op map[string]any, pathParams []any, root map[string]any, resolver *openAPIRefResolver, skillName string) (SkillAction, error) {
	description := strings.TrimSpace(stringAt(op, "summary"))
	if description == "" {
		description = strings.TrimSpace(stringAt(op, "description"))
	}

	action := SkillAction{
		Method:      method,
		Path:        path,
		Description: description,
		Args:        make([]ActionArg, 0),
		Request: RequestSpec{
			Query:      map[string]string{},
			Headers:    map[string]string{},
			Body:       map[string]any{},
			PathParams: map[string]string{},
		},
		Response: responseFromOperation(op, resolver),
		Risk:     riskFromMethod(method),
	}

	argIndex := map[string]int{}
	mergeArgsAndRequest, err := mergeParameters(pathParams, anySliceAt(op, "parameters"), resolver)
	if err != nil {
		return SkillAction{}, err
	}

	for _, p := range mergeArgsAndRequest {
		addOrUpdateArg(&action.Args, argIndex, p.arg)
		switch p.location {
		case "query":
			action.Request.Query[p.arg.Name] = "{" + p.arg.Name + "}"
		case "header":
			action.Request.Headers[p.arg.Name] = "{" + p.arg.Name + "}"
		case "path":
			action.Request.PathParams[p.arg.Name] = "{" + p.arg.Name + "}"
		}
	}

	if err := addRequestBodyArgs(&action, op, resolver, argIndex); err != nil {
		return SkillAction{}, err
	}

	action.Auth = extractOperationAuth(op, root, resolver, skillName)

	action.Args = sortArgs(action.Args)
	action.Request.Query = nilIfEmptyStringMap(action.Request.Query)
	action.Request.Headers = nilIfEmptyStringMap(action.Request.Headers)
	action.Request.PathParams = nilIfEmptyStringMap(action.Request.PathParams)
	action.Request.Body = nilIfEmptyAnyMap(action.Request.Body)

	return action, nil
}

type parameterActionArg struct {
	arg      ActionArg
	location string
}

func mergeParameters(pathLevel, operationLevel []any, resolver *openAPIRefResolver) ([]parameterActionArg, error) {
	merged := map[string]parameterActionArg{}
	order := make([]string, 0)

	apply := func(raw []any) error {
		for _, item := range raw {
			paramMap, ok := item.(map[string]any)
			if !ok {
				continue
			}

			resolved, err := resolver.resolveMap(paramMap)
			if err != nil {
				return fmt.Errorf("resolve parameter ref: %w", err)
			}

			arg, location, ok := parameterToArg(resolved, resolver)
			if !ok {
				continue
			}

			id := location + ":" + arg.Name
			if _, exists := merged[id]; !exists {
				order = append(order, id)
			}
			merged[id] = parameterActionArg{arg: arg, location: location}
		}
		return nil
	}

	if err := apply(pathLevel); err != nil {
		return nil, err
	}
	if err := apply(operationLevel); err != nil {
		return nil, err
	}

	result := make([]parameterActionArg, 0, len(order))
	for _, id := range order {
		result = append(result, merged[id])
	}

	return result, nil
}

func parameterToArg(param map[string]any, resolver *openAPIRefResolver) (ActionArg, string, bool) {
	name := strings.TrimSpace(stringAt(param, "name"))
	in := strings.ToLower(strings.TrimSpace(stringAt(param, "in")))
	if name == "" || (in != "query" && in != "path" && in != "header") {
		return ActionArg{}, "", false
	}

	schema := mapAt(param, "schema")
	if len(schema) > 0 {
		resolved, err := resolver.resolveMap(schema)
		if err == nil {
			schema = resolved
		}
		schema = resolveCompositeSchema(schema, resolver)
	}

	arg := ActionArg{
		Name:     name,
		Type:     schemaType(schema),
		Required: boolAt(param, "required") || in == "path",
	}

	if def, ok := valueAt(schema, "default"); ok {
		arg.Default = def
	}
	if enum := anySliceAt(schema, "enum"); len(enum) > 0 {
		arg.Enum = enum
	}

	return arg, in, true
}

func addRequestBodyArgs(action *SkillAction, op map[string]any, resolver *openAPIRefResolver, argIndex map[string]int) error {
	rawBody := mapAt(op, "requestBody")
	if len(rawBody) == 0 {
		return nil
	}

	requestBody, err := resolver.resolveMap(rawBody)
	if err != nil {
		return fmt.Errorf("resolve requestBody ref: %w", err)
	}

	content := mapAt(requestBody, "content")
	if len(content) == 0 {
		return nil
	}

	mediaType := pickMediaType(content)
	mediaMap := mapAt(content, mediaType)
	schema := mapAt(mediaMap, "schema")
	if len(schema) == 0 {
		return nil
	}

	resolvedSchema, err := resolver.resolveMap(schema)
	if err != nil {
		return fmt.Errorf("resolve request body schema ref: %w", err)
	}
	resolvedSchema = resolveCompositeSchema(resolvedSchema, resolver)
	appendActionWarnings(action, schemaWarnings(resolvedSchema))

	if opaque, _ := resolvedSchema[schemaOpaqueKey].(bool); opaque {
		return nil
	}

	requiredSet := make(map[string]struct{})
	for _, item := range anySliceAt(resolvedSchema, "required") {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			requiredSet[s] = struct{}{}
		}
	}

	if strings.EqualFold(schemaType(resolvedSchema), "object") {
		properties := mapAt(resolvedSchema, "properties")
		propNames := sortedMapKeys(properties)
		for _, propName := range propNames {
			propSchema := mapAt(properties, propName)
			if len(propSchema) > 0 {
				if resolved, resolveErr := resolver.resolveMap(propSchema); resolveErr == nil {
					propSchema = resolved
				}
				propSchema = resolveCompositeSchema(propSchema, resolver)
			}

			arg := ActionArg{
				Name:     propName,
				Type:     schemaType(propSchema),
				Required: hasKey(requiredSet, propName),
			}
			if def, ok := valueAt(propSchema, "default"); ok {
				arg.Default = def
			}
			if enum := anySliceAt(propSchema, "enum"); len(enum) > 0 {
				arg.Enum = enum
			}

			addOrUpdateArg(&action.Args, argIndex, arg)
			action.Request.Body[propName] = "{" + propName + "}"
		}
		return nil
	}

	arg := ActionArg{
		Name:     "body",
		Type:     schemaType(resolvedSchema),
		Required: boolAt(requestBody, "required"),
	}
	addOrUpdateArg(&action.Args, argIndex, arg)

	return nil
}

func responseFromOperation(op map[string]any, resolver *openAPIRefResolver) ResponseSpec {
	resp := ResponseSpec{Extract: "", Type: "object"}

	responses := mapAt(op, "responses")
	if len(responses) == 0 {
		return resp
	}

	code := pickResponseCode(responses)
	rawResponse := mapAt(responses, code)
	if len(rawResponse) == 0 {
		return resp
	}

	resolvedResponse, err := resolver.resolveMap(rawResponse)
	if err != nil {
		return resp
	}

	content := mapAt(resolvedResponse, "content")
	if len(content) == 0 {
		return resp
	}

	mediaType := pickMediaType(content)
	schema := mapAt(mapAt(content, mediaType), "schema")
	if len(schema) == 0 {
		return resp
	}

	resolvedSchema, err := resolver.resolveMap(schema)
	if err != nil {
		return resp
	}

	if strings.EqualFold(schemaType(resolvedSchema), "array") {
		resp.Type = "array"
	}

	return resp
}

func riskFromMethod(method string) RiskSpec {
	upper := strings.ToUpper(strings.TrimSpace(method))
	mutating := !(upper == "GET" || upper == "HEAD" || upper == "OPTIONS")

	level := "medium"
	switch upper {
	case "GET", "HEAD", "OPTIONS":
		level = "low"
	case "DELETE":
		level = "high"
	case "POST", "PUT", "PATCH":
		level = "medium"
	}

	return RiskSpec{Level: level, Mutating: mutating}
}

func normalizeSkillName(title string) string {
	name := strings.ToLower(strings.TrimSpace(title))
	name = nonAlphanumPattern.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	name = multiDashPattern.ReplaceAllString(name, "-")

	if name == "" {
		return "openapi-skill"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "skill-" + name
	}
	return name
}

func normalizeVersion(in string) string {
	v := strings.TrimSpace(in)
	if v == "" {
		return "1.0.0"
	}

	match := openAPIVersionPattern.FindStringSubmatch(v)
	if len(match) == 0 {
		return "1.0.0"
	}

	major := nonEmptyString(match[1], "1")
	minor := nonEmptyString(match[2], "0")
	patch := nonEmptyString(match[3], "0")
	return fmt.Sprintf("%s.%s.%s", major, minor, patch)
}

func firstServerURL(root map[string]any) string {
	servers := anySliceAt(root, "servers")
	if len(servers) == 0 {
		return ""
	}
	first, ok := servers[0].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringAt(first, "url"))
}

func normalizeBaseURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err == nil && u.Scheme != "" && u.Host != "" {
		return u.String()
	}
	return "https://example.com"
}

func firstReferencedSecurityScheme(root map[string]any) string {
	security := anySliceAt(root, "security")
	for _, item := range security {
		req, ok := item.(map[string]any)
		if !ok || len(req) == 0 {
			continue
		}
		keys := sortedMapKeys(req)
		return keys[0]
	}
	return ""
}

func normalizeActionKey(operationID, method, path string) string {
	base := strings.TrimSpace(operationID)
	if base == "" {
		base = strings.ToLower(method) + " " + path
	}

	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, "{", "")
	base = strings.ReplaceAll(base, "}", "")
	base = nonAlphanumPattern.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	base = multiDashPattern.ReplaceAllString(base, "-")

	if base == "" {
		return "action"
	}
	if base[0] >= '0' && base[0] <= '9' {
		return "action-" + base
	}
	return base
}

func ensureUniqueActionKey(base, method, path string, existing map[string]SkillAction) string {
	if _, ok := existing[base]; !ok {
		return base
	}

	hashInput := strings.ToLower(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
	sum := shortStableHash(hashInput)
	for size := 6; size <= len(sum); size += 2 {
		candidate := base + "-" + sum[:size]
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}

	return base + "-" + sum
}

func pickMediaType(content map[string]any) string {
	if _, ok := content["application/json"]; ok {
		return "application/json"
	}
	keys := sortedMapKeys(content)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func pickResponseCode(responses map[string]any) string {
	if _, ok := responses["200"]; ok {
		return "200"
	}

	keys := sortedMapKeys(responses)
	best := ""
	bestCode := 999
	for _, key := range keys {
		if strings.EqualFold(key, "default") {
			if best == "" {
				best = key
			}
			continue
		}
		code, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		if code >= 200 && code < 300 && code < bestCode {
			bestCode = code
			best = key
		}
	}
	if best != "" {
		return best
	}
	if len(keys) > 0 {
		return keys[0]
	}
	return ""
}

func schemaType(schema map[string]any) string {
	t := strings.ToLower(strings.TrimSpace(stringAt(schema, "type")))
	switch t {
	case "string", "integer", "boolean", "array", "object":
		return t
	case "number":
		return "number"
	default:
		if _, hasProps := schema["properties"]; hasProps {
			return "object"
		}
		if _, hasItems := schema["items"]; hasItems {
			return "array"
		}
		if _, has := schema["oneOf"]; has {
			return "string"
		}
		if _, has := schema["anyOf"]; has {
			return "string"
		}
		if subs, has := schema["allOf"]; has {
			if arr, ok := subs.([]any); ok {
				var found string
				for _, sub := range arr {
					if m, ok := sub.(map[string]any); ok {
						if st := schemaType(m); st != "string" {
							if found == "" {
								found = st
								continue
							}
							if (found == "integer" && st == "number") || (found == "number" && st == "integer") {
								found = "integer"
								continue
							}
							if found != st {
								return "string"
							}
						}
					}
				}
				if found != "" {
					return found
				}
			}
		}
		return "string"
	}
}

func extractOperationAuth(op map[string]any, root map[string]any, resolver *openAPIRefResolver, skillName string) *SkillAuth {
	securityAny, exists := op["security"]
	if !exists {
		return nil
	}

	security, ok := securityAny.([]any)
	if !ok {
		return nil
	}

	schemeName := firstReferencedSecuritySchemeFromList(security)
	if schemeName == "" {
		auth := SkillAuth{Type: "none"}
		return &auth
	}

	schemes := mapAt(mapAt(root, "components"), "securitySchemes")
	rawScheme := mapAt(schemes, schemeName)
	if len(rawScheme) == 0 {
		return nil
	}

	scheme, err := resolver.resolveMap(rawScheme)
	if err != nil {
		return nil
	}

	auth, err := authFromScheme(scheme, skillName)
	if err != nil {
		return nil
	}

	if auth.Type == "none" {
		auth.CredentialRef = ""
	}

	return &auth
}

func firstReferencedSecuritySchemeFromList(security []any) string {
	for _, item := range security {
		req, ok := item.(map[string]any)
		if !ok || len(req) == 0 {
			continue
		}
		keys := sortedMapKeys(req)
		return keys[0]
	}
	return ""
}

func authFromScheme(scheme map[string]any, skillName string) (SkillAuth, error) {
	auth := SkillAuth{}
	t := strings.ToLower(strings.TrimSpace(stringAt(scheme, "type")))
	switch t {
	case "http":
		httpScheme := strings.ToLower(strings.TrimSpace(stringAt(scheme, "scheme")))
		switch httpScheme {
		case "bearer":
			auth.Type = "bearer"
			auth.CredentialRef = credentialRef(skillName, "token")
		case "basic":
			auth.Type = "basic"
			auth.CredentialRef = credentialRef(skillName, "basic")
		default:
			return SkillAuth{}, fmt.Errorf("unsupported HTTP auth scheme %q: only bearer and basic are supported", httpScheme)
		}
	case "apikey":
		in := strings.ToLower(strings.TrimSpace(stringAt(scheme, "in")))
		name := strings.TrimSpace(stringAt(scheme, "name"))
		switch in {
		case "query":
			auth.Type = "query"
			auth.QueryParam = nonEmptyString(name, "api_key")
			auth.CredentialRef = credentialRef(skillName, "api_key")
		default:
			auth.Type = "header"
			auth.HeaderName = nonEmptyString(name, "X-API-Key")
			auth.CredentialRef = credentialRef(skillName, "api_key")
		}
	case "oauth2":
		auth.Type = "bearer"
		auth.CredentialRef = credentialRef(skillName, "oauth_token")
	case "openidconnect":
		auth.Type = "bearer"
		auth.CredentialRef = credentialRef(skillName, "oidc_token")
	default:
		return SkillAuth{}, fmt.Errorf("unsupported security scheme type %q: only http, apiKey, oauth2, and openIdConnect are supported", t)
	}
	return auth, nil
}

func resolveCompositeSchema(schema map[string]any, resolver *openAPIRefResolver) map[string]any {
	if len(schema) == 0 {
		return schema
	}

	resolved := schema
	if next, err := resolver.resolveMap(schema); err == nil && len(next) > 0 {
		resolved = next
	}

	if allOf := anySliceAt(resolved, "allOf"); len(allOf) > 0 {
		merged := cloneAnyMap(resolved)
		properties := map[string]any{}
		requiredSet := map[string]struct{}{}
		warnings := schemaWarnings(resolved)

		for _, req := range anySliceAt(resolved, "required") {
			if key, ok := req.(string); ok && strings.TrimSpace(key) != "" {
				requiredSet[key] = struct{}{}
			}
		}
		for key, value := range mapAt(resolved, "properties") {
			properties[key] = value
		}

		for _, item := range allOf {
			sub, ok := item.(map[string]any)
			if !ok {
				continue
			}
			subResolved := resolveCompositeSchema(sub, resolver)
			if disc, hasDisc := subResolved["discriminator"]; hasDisc {
				merged["discriminator"] = disc
			}
			warnings = append(warnings, schemaWarnings(subResolved)...)
			for key, value := range mapAt(subResolved, "properties") {
				if existing, hasExisting := properties[key]; hasExisting {
					existingMap, existingOK := existing.(map[string]any)
					newMap, newOK := value.(map[string]any)
					if existingOK && newOK {
						existingType := stringAt(existingMap, "type")
						newType := stringAt(newMap, "type")
						if existingType != "" && newType != "" && existingType != newType {
							warnings = append(warnings, fmt.Sprintf("allOf conflict for property %q: overriding type %q with %q (last definition wins)", key, existingType, newType))
							properties[key] = value
							continue
						}
					}
				}
				properties[key] = value
			}
			for _, req := range anySliceAt(subResolved, "required") {
				if key, ok := req.(string); ok && strings.TrimSpace(key) != "" {
					requiredSet[key] = struct{}{}
				}
			}
		}

		delete(merged, "allOf")
		if len(properties) > 0 {
			merged["properties"] = properties
			merged["type"] = "object"
		}
		required := make([]any, 0, len(requiredSet))
		for _, key := range sortedStringKeys(requiredSet) {
			required = append(required, key)
		}
		if len(required) > 0 {
			merged["required"] = required
		} else {
			delete(merged, "required")
		}
		if len(warnings) > 0 {
			warningAny := make([]any, 0, len(warnings))
			for _, warning := range dedupeStrings(warnings) {
				warningAny = append(warningAny, warning)
			}
			merged[schemaWarningsKey] = warningAny
		} else {
			delete(merged, schemaWarningsKey)
		}

		return merged
	}

	for _, keyword := range []string{"oneOf", "anyOf"} {
		variants := anySliceAt(resolved, keyword)
		if len(variants) == 0 {
			continue
		}
		if isComplexComposition(variants, resolver) || hasKey(resolved, "discriminator") {
			merged := cloneAnyMap(resolved)
			delete(merged, keyword)
			delete(merged, "required")
			merged["type"] = "object"
			merged[schemaOpaqueKey] = true
			return merged
		}
		first, ok := variants[0].(map[string]any)
		if !ok {
			break
		}
		primary := resolveCompositeSchema(first, resolver)
		merged := cloneAnyMap(resolved)
		delete(merged, "required")
		for key, value := range primary {
			merged[key] = value
		}
		delete(merged, "oneOf")
		delete(merged, "anyOf")
		return merged
	}

	return resolved
}

func isComplexComposition(variants []any, resolver *openAPIRefResolver) bool {
	if len(variants) >= 3 {
		return true
	}

	for _, item := range variants {
		v, ok := item.(map[string]any)
		if !ok {
			continue
		}
		resolved := v
		if next, err := resolver.resolveMap(v); err == nil && len(next) > 0 {
			resolved = next
		}
		if hasKey(resolved, "discriminator") {
			return true
		}

		props := mapAt(resolved, "properties")
		for _, propVal := range props {
			propMap, ok := propVal.(map[string]any)
			if !ok {
				continue
			}
			propType := stringAt(propMap, "type")
			if propType == "object" || propType == "array" {
				return true
			}
			if hasKey(propMap, "oneOf") || hasKey(propMap, "anyOf") || hasKey(propMap, "allOf") {
				return true
			}
		}
	}

	return false
}

func schemaWarnings(schema map[string]any) []string {
	raw := anySliceAt(schema, schemaWarningsKey)
	if len(raw) == 0 {
		return nil
	}
	warnings := make([]string, 0, len(raw))
	for _, item := range raw {
		warning, ok := item.(string)
		if !ok || strings.TrimSpace(warning) == "" {
			continue
		}
		warnings = append(warnings, warning)
	}
	if len(warnings) == 0 {
		return nil
	}
	return warnings
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if hasKey(seen, trimmed) {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func appendActionWarnings(action *SkillAction, warnings []string) {
	uniqueWarnings := dedupeStrings(warnings)
	if len(uniqueWarnings) == 0 {
		return
	}
	lines := make([]string, 0, len(uniqueWarnings))
	for _, warning := range uniqueWarnings {
		lines = append(lines, "WARNING: "+warning)
	}
	warningBlock := strings.Join(lines, "\n")
	description := strings.TrimSpace(action.Description)
	if description == "" {
		action.Description = warningBlock
		return
	}
	action.Description = description + "\n" + warningBlock
}

func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func shortStableHash(input string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(input))
	return fmt.Sprintf("%08x", h.Sum32())
}

func addOrUpdateArg(args *[]ActionArg, argIndex map[string]int, arg ActionArg) {
	if idx, ok := argIndex[arg.Name]; ok {
		(*args)[idx] = arg
		return
	}
	argIndex[arg.Name] = len(*args)
	*args = append(*args, arg)
}

func sortArgs(in []ActionArg) []ActionArg {
	out := append([]ActionArg(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func credentialRef(skillName, suffix string) string {
	return skillName + "." + suffix
}

func nilIfEmptyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	return in
}

func nilIfEmptyAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	return in
}

func cloneAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	maps.Copy(out, in)
	return out
}

func mapAt(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	out, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return out
}

func anySliceAt(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	out, ok := v.([]any)
	if !ok {
		return nil
	}
	return out
}

func stringAt(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func boolAt(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func valueAt(m map[string]any, key string) (any, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m[key]
	return v, ok
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func normalizeYAMLValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, value := range typed {
			out[k] = normalizeYAMLValue(value)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for k, value := range typed {
			out[fmt.Sprint(k)] = normalizeYAMLValue(value)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, value := range typed {
			out[i] = normalizeYAMLValue(value)
		}
		return out
	default:
		return v
	}
}

func hasKey[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}

func nonEmptyString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

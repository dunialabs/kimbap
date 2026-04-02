package services

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

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

type OpenAPIGenerateOptions struct {
	NameOverride string
	Tags         []string
	PathPrefixes []string
}

type normalizedOpenAPIGenerateOptions struct {
	NameOverride string
	Tags         map[string]struct{}
	PathPrefixes []string
}

func GenerateFromOpenAPI(spec []byte) (*ServiceManifest, error) {
	return GenerateFromOpenAPIWithOptions(spec, OpenAPIGenerateOptions{})
}

func GenerateFromOpenAPIWithOptions(spec []byte, opts OpenAPIGenerateOptions) (*ServiceManifest, error) {
	root, err := parseOpenAPIRoot(spec)
	if err != nil {
		return nil, err
	}
	return generateFromOpenAPIRoot(root, newOpenAPIRefResolver(root), opts)
}

func generateFromOpenAPIRoot(root map[string]any, resolver *openAPIRefResolver, opts OpenAPIGenerateOptions) (*ServiceManifest, error) {
	normalizedOpts := normalizeOpenAPIGenerateOptions(opts)

	openapiVersion := strings.TrimSpace(stringAt(root, "openapi"))
	if openapiVersion == "" || !strings.HasPrefix(openapiVersion, "3") {
		return nil, fmt.Errorf("unsupported OpenAPI version %q: only OpenAPI 3.x is supported", openapiVersion)
	}

	info := mapAt(root, "info")
	name := normalizeSkillName(stringAt(info, "title"))
	if normalizedOpts.NameOverride != "" {
		name = normalizedOpts.NameOverride
	}
	version := normalizeVersion(stringAt(info, "version"))
	description := strings.TrimSpace(stringAt(info, "description"))
	baseURL, err := normalizeBaseURL(firstServerURL(root))
	if err != nil {
		return nil, err
	}

	auth, err := extractAuth(root, resolver, name)
	if err != nil {
		return nil, err
	}

	actions, err := extractActions(root, resolver, name, normalizedOpts)
	if err != nil {
		return nil, err
	}

	manifest := &ServiceManifest{
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

func GenerateFromOpenAPIFile(path string) (*ServiceManifest, error) {
	return GenerateFromOpenAPIFileWithOptions(path, OpenAPIGenerateOptions{})
}

func GenerateFromOpenAPIFileWithOptions(path string, opts OpenAPIGenerateOptions) (*ServiceManifest, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve OpenAPI file path: %w", err)
	}

	data, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI file: %w", err)
	}
	root, err := parseOpenAPIRoot(data)
	if err != nil {
		return nil, err
	}

	return generateFromOpenAPIRoot(root, newOpenAPIFileRefResolver(absolutePath, root), opts)
}

type openAPIDocument struct {
	path string
	root map[string]any
}

type openAPIRefResolver struct {
	root    map[string]any
	rootDoc *openAPIDocument
	docs    map[string]*openAPIDocument
}

func newOpenAPIRefResolver(root map[string]any) *openAPIRefResolver {
	return &openAPIRefResolver{root: root}
}

func newOpenAPIFileRefResolver(path string, root map[string]any) *openAPIRefResolver {
	doc := &openAPIDocument{
		path: path,
		root: root,
	}
	resolver := &openAPIRefResolver{
		root:    root,
		rootDoc: doc,
		docs:    map[string]*openAPIDocument{path: doc},
	}
	return resolver
}

func (r *openAPIRefResolver) resolveMap(in map[string]any) (map[string]any, error) {
	resolved, _, err := r.resolveMapInDoc(in, r.rootDoc)
	return resolved, err
}

func (r *openAPIRefResolver) resolveMapInDoc(in map[string]any, currentDoc *openAPIDocument) (map[string]any, *openAPIDocument, error) {
	if in == nil {
		return nil, currentDoc, nil
	}

	out := in
	resolvedDoc := currentDoc
	for range 8 {
		ref := strings.TrimSpace(stringAt(out, "$ref"))
		if ref == "" {
			return out, resolvedDoc, nil
		}

		resolvedAny, nextDoc, err := r.resolveRef(resolvedDoc, ref)
		if err != nil {
			return nil, nil, err
		}

		resolvedMap, ok := resolvedAny.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("$ref %q did not resolve to an object", ref)
		}

		merged := cloneAnyMap(resolvedMap)
		for k, v := range out {
			if k == "$ref" {
				continue
			}
			merged[k] = v
		}
		out = merged
		resolvedDoc = nextDoc
	}

	return nil, nil, fmt.Errorf("$ref resolution depth exceeded")
}

func (r *openAPIRefResolver) resolveRef(currentDoc *openAPIDocument, ref string) (any, *openAPIDocument, error) {
	refPath, pointer, err := splitOpenAPIRef(ref)
	if err != nil {
		return nil, nil, err
	}

	if refPath == "" {
		targetRoot := any(r.root)
		targetDoc := currentDoc
		if currentDoc != nil {
			targetRoot = currentDoc.root
		}
		resolved, err := resolveOpenAPIJSONPointer(targetRoot, pointer, ref)
		return resolved, targetDoc, err
	}

	if parsed, parseErr := url.Parse(refPath); parseErr == nil && parsed.Scheme != "" {
		return nil, nil, fmt.Errorf("unsupported $ref %q: only local refs and relative file refs are supported", ref)
	}
	if filepath.IsAbs(refPath) {
		return nil, nil, fmt.Errorf("unsupported $ref %q: only local refs and relative file refs are supported", ref)
	}
	if currentDoc == nil || strings.TrimSpace(currentDoc.path) == "" {
		return nil, nil, fmt.Errorf("unsupported $ref %q: external file refs require OpenAPI file input", ref)
	}

	targetPath := filepath.Clean(filepath.Join(filepath.Dir(currentDoc.path), filepath.FromSlash(refPath)))
	targetDoc, err := r.loadDocument(targetPath)
	if err != nil {
		return nil, nil, err
	}

	resolved, err := resolveOpenAPIJSONPointer(targetDoc.root, pointer, ref)
	return resolved, targetDoc, err
}

func (r *openAPIRefResolver) loadDocument(path string) (*openAPIDocument, error) {
	if r.docs == nil {
		r.docs = make(map[string]*openAPIDocument)
	}
	if doc, ok := r.docs[path]; ok {
		return doc, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read referenced OpenAPI file %q: %w", path, err)
	}
	root, err := parseOpenAPIRoot(data)
	if err != nil {
		return nil, fmt.Errorf("parse referenced OpenAPI file %q: %w", path, err)
	}

	doc := &openAPIDocument{
		path: path,
		root: root,
	}
	r.docs[path] = doc
	return doc, nil
}

func splitOpenAPIRef(ref string) (string, string, error) {
	refPath, fragment, hasFragment := strings.Cut(ref, "#")
	switch {
	case !hasFragment:
		return refPath, "", nil
	case fragment == "":
		return refPath, "", nil
	case !strings.HasPrefix(fragment, "/"):
		return "", "", fmt.Errorf("unsupported $ref %q: only JSON Pointer fragments are supported", ref)
	default:
		return refPath, fragment, nil
	}
}

func resolveOpenAPIJSONPointer(root any, pointer string, ref string) (any, error) {
	if pointer == "" {
		return root, nil
	}

	current := root
	for rawPart := range strings.SplitSeq(strings.TrimPrefix(pointer, "/"), "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(rawPart, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil, fmt.Errorf("unresolvable $ref %q", ref)
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, fmt.Errorf("unresolvable $ref %q", ref)
			}
			current = typed[index]
		default:
			return nil, fmt.Errorf("invalid $ref %q", ref)
		}
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

func extractAuth(root map[string]any, resolver *openAPIRefResolver, skillName string) (ServiceAuth, error) {
	components := mapAt(root, "components")
	schemes := mapAt(components, "securitySchemes")
	if len(schemes) == 0 {
		return ServiceAuth{Type: "none"}, nil
	}

	securityAny, hasRootSecurity := root["security"]
	if hasRootSecurity {
		security, ok := securityAny.([]any)
		if !ok || len(security) == 0 {
			return ServiceAuth{Type: "none"}, nil
		}
	}

	schemeName := firstReferencedSecurityScheme(root)
	if schemeName == "" {
		return ServiceAuth{Type: "none"}, nil
	}

	rawScheme := mapAt(schemes, schemeName)
	if len(rawScheme) == 0 {
		return ServiceAuth{Type: "none"}, nil
	}

	scheme, _, err := resolver.resolveMapInDoc(rawScheme, resolver.rootDoc)
	if err != nil {
		return ServiceAuth{}, fmt.Errorf("resolve security scheme %q: %w", schemeName, err)
	}

	auth, err := authFromScheme(scheme, skillName)
	if err != nil {
		return ServiceAuth{}, err
	}

	if auth.Type == "none" {
		return ServiceAuth{Type: "none"}, nil
	}

	if strings.TrimSpace(auth.CredentialRef) == "" {
		auth.CredentialRef = credentialRef(skillName, "credential")
	}

	return auth, nil
}

func extractActions(root map[string]any, resolver *openAPIRefResolver, skillName string, opts normalizedOpenAPIGenerateOptions) (map[string]ServiceAction, error) {
	paths := mapAt(root, "paths")
	actions := make(map[string]ServiceAction)
	matchedFilteredOperation := false

	pathKeys := sortedMapKeys(paths)
	for _, p := range pathKeys {
		if len(opts.PathPrefixes) > 0 && !matchesOpenAPIPathPrefixFilter(p, opts.PathPrefixes) {
			continue
		}

		pathItem, pathDoc, err := resolver.resolveMapInDoc(mapAt(paths, p), resolver.rootDoc)
		if err != nil {
			return nil, fmt.Errorf("resolve path item %q: %w", p, err)
		}

		pathParams := anySliceAt(pathItem, "parameters")
		for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
			rawOp := mapAt(pathItem, method)
			if len(rawOp) == 0 {
				continue
			}

			op, opDoc, err := resolver.resolveMapInDoc(rawOp, pathDoc)
			if err != nil {
				return nil, fmt.Errorf("resolve operation %s %s: %w", strings.ToUpper(method), p, err)
			}
			if len(opts.Tags) > 0 && !matchesOpenAPITagFilter(op, opts.Tags) {
				continue
			}
			matchedFilteredOperation = true

			action, err := buildAction(strings.ToUpper(method), p, op, opDoc, pathParams, pathDoc, root, resolver, skillName)
			if err != nil {
				return nil, err
			}

			key := normalizeActionKey(stringAt(op, "operationId"), method, p)
			key = ensureUniqueActionKey(key, method, p, actions)
			actions[key] = action
		}
	}

	if hasOpenAPIOperationFilters(opts) && !matchedFilteredOperation {
		return nil, fmt.Errorf("no OpenAPI operations matched the requested filters")
	}

	return actions, nil
}

func normalizeOpenAPIGenerateOptions(opts OpenAPIGenerateOptions) normalizedOpenAPIGenerateOptions {
	normalized := normalizedOpenAPIGenerateOptions{
		NameOverride: normalizeOpenAPINameOverride(opts.NameOverride),
		Tags:         make(map[string]struct{}),
		PathPrefixes: make([]string, 0, len(opts.PathPrefixes)),
	}

	for _, tag := range opts.Tags {
		trimmed := strings.ToLower(strings.TrimSpace(tag))
		if trimmed == "" {
			continue
		}
		normalized.Tags[trimmed] = struct{}{}
	}

	seenPrefixes := map[string]struct{}{}
	for _, prefix := range opts.PathPrefixes {
		normalizedPrefix := normalizeOpenAPIPathPrefix(prefix)
		if normalizedPrefix == "" {
			continue
		}
		if _, exists := seenPrefixes[normalizedPrefix]; exists {
			continue
		}
		seenPrefixes[normalizedPrefix] = struct{}{}
		normalized.PathPrefixes = append(normalized.PathPrefixes, normalizedPrefix)
	}

	return normalized
}

func normalizeOpenAPINameOverride(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	return normalizeSkillName(trimmed)
}

func normalizeOpenAPIPathPrefix(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed
}

func hasOpenAPIOperationFilters(opts normalizedOpenAPIGenerateOptions) bool {
	return len(opts.Tags) > 0 || len(opts.PathPrefixes) > 0
}

func matchesOpenAPITagFilter(op map[string]any, allowed map[string]struct{}) bool {
	for _, tagAny := range anySliceAt(op, "tags") {
		tag, ok := tagAny.(string)
		if !ok {
			continue
		}
		if _, exists := allowed[strings.ToLower(strings.TrimSpace(tag))]; exists {
			return true
		}
	}
	return false
}

func matchesOpenAPIPathPrefixFilter(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func buildAction(method, path string, op map[string]any, opDoc *openAPIDocument, pathParams []any, pathDoc *openAPIDocument, root map[string]any, resolver *openAPIRefResolver, skillName string) (ServiceAction, error) {
	description := strings.TrimSpace(stringAt(op, "summary"))
	if description == "" {
		description = strings.TrimSpace(stringAt(op, "description"))
	}

	action := ServiceAction{
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
		Response: responseFromOperation(op, opDoc, resolver),
		Risk:     riskFromMethod(method),
	}

	argIndex := map[string]int{}
	mergeArgsAndRequest, err := mergeParameters(pathParams, pathDoc, anySliceAt(op, "parameters"), opDoc, resolver)
	if err != nil {
		return ServiceAction{}, err
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

	if err := addRequestBodyArgs(&action, op, opDoc, resolver, argIndex); err != nil {
		return ServiceAction{}, err
	}

	actionAuth, err := extractOperationAuth(op, root, resolver, skillName)
	if err != nil {
		return ServiceAction{}, err
	}
	action.Auth = actionAuth

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

func mergeParameters(pathLevel []any, pathDoc *openAPIDocument, operationLevel []any, operationDoc *openAPIDocument, resolver *openAPIRefResolver) ([]parameterActionArg, error) {
	merged := map[string]parameterActionArg{}
	order := make([]string, 0)

	apply := func(raw []any, currentDoc *openAPIDocument) error {
		for _, item := range raw {
			paramMap, ok := item.(map[string]any)
			if !ok {
				continue
			}

			resolved, resolvedDoc, err := resolver.resolveMapInDoc(paramMap, currentDoc)
			if err != nil {
				return fmt.Errorf("resolve parameter ref: %w", err)
			}

			arg, location, ok := parameterToArg(resolved, resolvedDoc, resolver)
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

	if err := apply(pathLevel, pathDoc); err != nil {
		return nil, err
	}
	if err := apply(operationLevel, operationDoc); err != nil {
		return nil, err
	}

	result := make([]parameterActionArg, 0, len(order))
	for _, id := range order {
		result = append(result, merged[id])
	}

	return result, nil
}

func parameterToArg(param map[string]any, currentDoc *openAPIDocument, resolver *openAPIRefResolver) (ActionArg, string, bool) {
	name := strings.TrimSpace(stringAt(param, "name"))
	in := strings.ToLower(strings.TrimSpace(stringAt(param, "in")))
	if name == "" || (in != "query" && in != "path" && in != "header") {
		return ActionArg{}, "", false
	}

	schema := mapAt(param, "schema")
	if len(schema) > 0 {
		resolved, resolvedDoc, err := resolver.resolveMapInDoc(schema, currentDoc)
		if err == nil {
			schema = resolved
			currentDoc = resolvedDoc
		}
		schema = resolveCompositeSchema(schema, currentDoc, resolver)
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

func addRequestBodyArgs(action *ServiceAction, op map[string]any, opDoc *openAPIDocument, resolver *openAPIRefResolver, argIndex map[string]int) error {
	rawBody := mapAt(op, "requestBody")
	if len(rawBody) == 0 {
		return nil
	}

	requestBody, requestBodyDoc, err := resolver.resolveMapInDoc(rawBody, opDoc)
	if err != nil {
		return fmt.Errorf("resolve requestBody ref: %w", err)
	}

	content := mapAt(requestBody, "content")
	if len(content) == 0 {
		return nil
	}

	mediaType := pickMediaType(content)
	if !strings.Contains(strings.ToLower(mediaType), "json") {
		appendActionWarnings(action, []string{fmt.Sprintf("skipping request body: unsupported media type %q (only JSON is supported)", mediaType)})
		return nil
	}
	mediaMap := mapAt(content, mediaType)
	schema := mapAt(mediaMap, "schema")
	if len(schema) == 0 {
		return nil
	}

	resolvedSchema, resolvedSchemaDoc, err := resolver.resolveMapInDoc(schema, requestBodyDoc)
	if err != nil {
		return fmt.Errorf("resolve request body schema ref: %w", err)
	}
	resolvedSchema = resolveCompositeSchema(resolvedSchema, resolvedSchemaDoc, resolver)
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
		if len(properties) == 0 {
			addOrUpdateArg(&action.Args, argIndex, ActionArg{
				Name:     "body",
				Type:     "object",
				Required: boolAt(requestBody, "required"),
			})
			return nil
		}
		propNames := sortedMapKeys(properties)
		for _, propName := range propNames {
			propSchema := mapAt(properties, propName)
			if len(propSchema) > 0 {
				resolved, propDoc, resolveErr := resolver.resolveMapInDoc(propSchema, resolvedSchemaDoc)
				if resolveErr != nil {
					return fmt.Errorf("resolve property %q schema ref: %w", propName, resolveErr)
				}
				propSchema = resolved
				propSchema = resolveCompositeSchema(propSchema, propDoc, resolver)
			}

			arg := ActionArg{
				Name:     propName,
				Type:     schemaType(propSchema),
				Required: boolAt(requestBody, "required") && hasKey(requiredSet, propName),
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

func responseFromOperation(op map[string]any, opDoc *openAPIDocument, resolver *openAPIRefResolver) ResponseSpec {
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

	resolvedResponse, responseDoc, err := resolver.resolveMapInDoc(rawResponse, opDoc)
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

	resolvedSchema, _, err := resolver.resolveMapInDoc(schema, responseDoc)
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

	level := "medium"
	switch upper {
	case "GET", "HEAD", "OPTIONS":
		level = "low"
	case "DELETE":
		level = "high"
	case "POST", "PUT", "PATCH":
		level = "medium"
	}

	return RiskSpec{Level: level}
}

func normalizeSkillName(title string) string {
	name := strings.ToLower(strings.TrimSpace(title))
	name = nonAlphanumPattern.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	name = multiDashPattern.ReplaceAllString(name, "-")

	if name == "" {
		return "openapi-service"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "service-" + name
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

func normalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("OpenAPI spec must include at least one absolute server URL")
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid OpenAPI server URL %q: %w", trimmed, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("OpenAPI server URL must be absolute: %q", trimmed)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("OpenAPI server URL must use http or https: %q", trimmed)
	}

	return u.String(), nil
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

func ensureUniqueActionKey(base, method, path string, existing map[string]ServiceAction) string {
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
	for _, key := range keys {
		if strings.Contains(strings.ToLower(key), "json") {
			return key
		}
	}
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

func extractOperationAuth(op map[string]any, root map[string]any, resolver *openAPIRefResolver, skillName string) (*ServiceAuth, error) {
	securityAny, exists := op["security"]
	if !exists {
		return nil, nil
	}

	security, ok := securityAny.([]any)
	if !ok {
		return nil, fmt.Errorf("operation security must be an array")
	}

	schemeName := firstReferencedSecuritySchemeFromList(security)
	if schemeName == "" {
		auth := ServiceAuth{Type: "none"}
		return &auth, nil
	}

	schemes := mapAt(mapAt(root, "components"), "securitySchemes")
	rawScheme := mapAt(schemes, schemeName)
	if len(rawScheme) == 0 {
		return nil, fmt.Errorf("security scheme %q not found", schemeName)
	}

	scheme, err := resolver.resolveMap(rawScheme)
	if err != nil {
		return nil, fmt.Errorf("resolve operation security scheme %q: %w", schemeName, err)
	}

	auth, err := authFromScheme(scheme, skillName)
	if err != nil {
		return nil, fmt.Errorf("convert operation security scheme %q: %w", schemeName, err)
	}

	if auth.Type == "none" {
		auth.CredentialRef = ""
	}

	return &auth, nil
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

func authFromScheme(scheme map[string]any, skillName string) (ServiceAuth, error) {
	auth := ServiceAuth{}
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
			return ServiceAuth{}, fmt.Errorf("unsupported HTTP auth scheme %q: only bearer and basic are supported", httpScheme)
		}
	case "apikey":
		in := strings.ToLower(strings.TrimSpace(stringAt(scheme, "in")))
		name := strings.TrimSpace(stringAt(scheme, "name"))
		switch in {
		case "query":
			auth.Type = "query"
			auth.QueryParam = nonEmptyString(name, "api_key")
			auth.CredentialRef = credentialRef(skillName, "api_key")
		case "header", "":
			auth.Type = "header"
			auth.HeaderName = nonEmptyString(name, "X-API-Key")
			auth.CredentialRef = credentialRef(skillName, "api_key")
		default:
			return ServiceAuth{}, fmt.Errorf("unsupported apiKey location %q: only header and query are supported", in)
		}
	case "oauth2":
		auth.Type = "bearer"
		auth.CredentialRef = credentialRef(skillName, "oauth_token")
	case "openidconnect":
		auth.Type = "bearer"
		auth.CredentialRef = credentialRef(skillName, "oidc_token")
	default:
		return ServiceAuth{}, fmt.Errorf("unsupported security scheme type %q: only http, apiKey, oauth2, and openIdConnect are supported", t)
	}
	return auth, nil
}

func resolveCompositeSchema(schema map[string]any, currentDoc *openAPIDocument, resolver *openAPIRefResolver) map[string]any {
	return resolveCompositeSchemaDepth(schema, currentDoc, resolver, 0)
}

func resolveCompositeSchemaDepth(schema map[string]any, currentDoc *openAPIDocument, resolver *openAPIRefResolver, depth int) map[string]any {
	if len(schema) == 0 || depth > 16 {
		if depth > 16 {
			merged := cloneAnyMap(schema)
			if _, hasType := merged["type"]; !hasType {
				merged["type"] = "object"
			}
			merged[schemaOpaqueKey] = true
			return merged
		}
		return schema
	}

	resolved := schema
	resolvedDoc := currentDoc
	if next, nextDoc, err := resolver.resolveMapInDoc(schema, currentDoc); err == nil && len(next) > 0 {
		resolved = next
		resolvedDoc = nextDoc
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
		maps.Copy(properties, mapAt(resolved, "properties"))

		for _, item := range allOf {
			sub, ok := item.(map[string]any)
			if !ok {
				continue
			}
			subResolved := resolveCompositeSchemaDepth(sub, resolvedDoc, resolver, depth+1)
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
		if isComplexComposition(variants, resolvedDoc, resolver) || hasKey(resolved, "discriminator") {
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
		primary := resolveCompositeSchemaDepth(first, resolvedDoc, resolver, depth+1)
		parentRequired := anySliceAt(resolved, "required")
		parentProps := mapAt(resolved, "properties")
		merged := cloneAnyMap(resolved)
		primaryProps := mapAt(primary, "properties")
		if len(parentProps) > 0 && len(primaryProps) > 0 {
			mergedProps := cloneAnyMap(parentProps)
			maps.Copy(mergedProps, primaryProps)
			primary = cloneAnyMap(primary)
			primary["properties"] = mergedProps
		}
		maps.Copy(merged, primary)
		delete(merged, "oneOf")
		delete(merged, "anyOf")
		if len(parentRequired) > 0 {
			variantRequired := anySliceAt(primary, "required")
			requiredSet := map[string]struct{}{}
			for _, r := range parentRequired {
				if s, ok := r.(string); ok && strings.TrimSpace(s) != "" {
					requiredSet[s] = struct{}{}
				}
			}
			for _, r := range variantRequired {
				if s, ok := r.(string); ok && strings.TrimSpace(s) != "" {
					requiredSet[s] = struct{}{}
				}
			}
			mergedRequired := make([]any, 0, len(requiredSet))
			for _, key := range sortedStringKeys(requiredSet) {
				mergedRequired = append(mergedRequired, key)
			}
			merged["required"] = mergedRequired
		}
		return merged
	}

	return resolved
}

func isComplexComposition(variants []any, currentDoc *openAPIDocument, resolver *openAPIRefResolver) bool {
	if len(variants) >= 3 {
		return true
	}

	for _, item := range variants {
		v, ok := item.(map[string]any)
		if !ok {
			continue
		}
		resolved := v
		if next, _, err := resolver.resolveMapInDoc(v, currentDoc); err == nil && len(next) > 0 {
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

func appendActionWarnings(action *ServiceAction, warnings []string) {
	uniqueWarnings := dedupeStrings(warnings)
	if len(uniqueWarnings) == 0 {
		return
	}
	action.Warnings = append([]string(nil), uniqueWarnings...)
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

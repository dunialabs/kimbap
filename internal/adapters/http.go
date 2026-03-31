package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dunialabs/kimbap/internal/actions"
)

type HTTPAdapter struct {
	client *http.Client
}

const defaultMaxResponseBodyBytes int64 = 4 << 20
const maxRetryAfterSeconds = 120
const defaultAdapterAttemptTimeout = 30 * time.Second

func NewHTTPAdapter(client *http.Client) *HTTPAdapter {
	if client == nil {
		client = &http.Client{}
	}
	if client.CheckRedirect == nil {
		client.CheckRedirect = secureRedirectPolicy
	}
	return &HTTPAdapter{client: client}
}

func secureRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if len(via) == 0 || req == nil || req.URL == nil {
		return nil
	}

	origin := via[0]
	if origin == nil || origin.URL == nil {
		return nil
	}

	crossHost := !strings.EqualFold(origin.URL.Hostname(), req.URL.Hostname()) || !strings.EqualFold(effectivePort(origin.URL), effectivePort(req.URL))
	downgradedHTTPS := strings.EqualFold(origin.URL.Scheme, "https") && strings.EqualFold(req.URL.Scheme, "http")
	if crossHost || downgradedHTTPS {
		stripSensitiveRedirectHeaders(req.Header)
	}

	return nil
}

func stripSensitiveRedirectHeaders(headers http.Header) {
	for key := range headers {
		if isSensitiveRedirectHeader(key) {
			headers.Del(key)
		}
	}
}

func isSensitiveRedirectHeader(key string) bool {
	name := strings.ToLower(strings.TrimSpace(key))
	if name == "authorization" || name == "proxy-authorization" {
		return true
	}
	if strings.Contains(name, "api-key") || strings.Contains(name, "apikey") {
		return true
	}
	if strings.Contains(name, "token") || strings.Contains(name, "secret") {
		return true
	}
	return false
}

func (a *HTTPAdapter) Type() string {
	return "http"
}

func (a *HTTPAdapter) Validate(def actions.ActionDefinition) error {
	tmpl := strings.TrimSpace(def.Adapter.URLTemplate)
	if tmpl == "" {
		return fmt.Errorf("adapter url template is required")
	}
	if !isHTTPURL(tmpl) {
		if strings.TrimSpace(def.Adapter.BaseURL) == "" {
			return fmt.Errorf("adapter base_url is required for relative url template %q", tmpl)
		}
	}
	return nil
}

const defaultMaxPaginationPages = 10
const hardMaxPaginationPages = 100
const hardMaxPaginationPageLimit = 1000
const hardMaxPaginationTotalItems = 5000

func (a *HTTPAdapter) Execute(ctx context.Context, req AdapterRequest) (*AdapterResult, error) {
	if req.Action.Pagination != nil {
		style := strings.ToLower(strings.TrimSpace(req.Action.Pagination.Style))
		if style != "" {
			if style != "cursor" && style != "offset" {
				return nil, actions.NewExecutionError(actions.ErrValidationFailed, fmt.Sprintf("unsupported pagination style %q", req.Action.Pagination.Style), http.StatusBadRequest, false, nil)
			}
			return a.executeWithPagination(ctx, req)
		}
	}
	return a.executeSingle(ctx, req)
}

func (a *HTTPAdapter) executeWithPagination(ctx context.Context, req AdapterRequest) (*AdapterResult, error) {
	start := time.Now()
	pageCfg := req.Action.Pagination
	maxPages := defaultMaxPaginationPages
	if pageCfg.MaxPages > 0 {
		maxPages = pageCfg.MaxPages
	}
	if v, ok := req.Input["_max_pages"]; ok {
		if n, ok := positiveIntFromAny(v); ok {
			maxPages = n
		}
	}
	if maxPages > hardMaxPaginationPages {
		maxPages = hardMaxPaginationPages
	}

	var allItems []any
	cursor := ""
	offset := 0

	for page := 0; page < maxPages; page++ {
		select {
		case <-ctx.Done():
			return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, ctx.Err().Error(), http.StatusGatewayTimeout, true, nil)
		default:
		}

		pageInput := cloneAnyMap(req.Input)
		if pageInput == nil {
			pageInput = map[string]any{}
		}
		delete(pageInput, "_max_pages")

		limitParam := pageCfg.LimitParam
		if limitParam == "" {
			limitParam = "limit"
		}
		limit := pageCfg.DefaultLimit
		if limit <= 0 {
			limit = 20
		}
		if _, exists := pageInput[limitParam]; !exists {
			pageInput[limitParam] = limit
		} else {
			if n, ok := positiveIntFromAny(pageInput[limitParam]); ok {
				limit = n
			}
		}
		if limit > hardMaxPaginationPageLimit {
			limit = hardMaxPaginationPageLimit
			pageInput[limitParam] = limit
		}

		style := strings.ToLower(strings.TrimSpace(pageCfg.Style))
		switch style {
		case "cursor":
			if cursor != "" {
				param := pageCfg.CursorParam
				if param == "" {
					param = "cursor"
				}
				pageInput[param] = cursor
			}
		case "offset":
			param := pageCfg.OffsetParam
			if param == "" {
				param = "offset"
			}
			pageInput[param] = offset
		}

		pageReq := req
		pageReq.Input = pageInput
		result, err := a.executeSingle(ctx, pageReq)
		if err != nil {
			return nil, err
		}

		pageItems, found := extractResponseItems(result.Output)
		if !found {
			allItems = append(allItems, result.Output)
			return &AdapterResult{
				Output:     map[string]any{"items": allItems, "_pagination": map[string]any{"pages": page + 1, "total_items": len(allItems)}},
				HTTPStatus: result.HTTPStatus,
				Headers:    result.Headers,
				DurationMS: time.Since(start).Milliseconds(),
			}, nil
		}
		allItems = append(allItems, pageItems...)
		if len(allItems) > hardMaxPaginationTotalItems {
			return nil, actions.NewExecutionError(
				actions.ErrDownstreamUnavailable,
				fmt.Sprintf("paginated response exceeded %d total items", hardMaxPaginationTotalItems),
				http.StatusBadGateway,
				false,
				map[string]any{"max_total_items": hardMaxPaginationTotalItems},
			)
		}

		switch style {
		case "cursor":
			cursorPath := pageCfg.ResponseCursor
			if cursorPath == "" {
				cursorPath = "next_cursor"
			}
			nextCursor, _ := extractByPath(result.Output, cursorPath)
			if s, ok := nextCursor.(string); ok && s != "" {
				cursor = s
			} else {
				return paginatedResult(allItems, page+1, result, start), nil
			}
		case "offset":
			offset += len(pageItems)
			if len(pageItems) < limit {
				return paginatedResult(allItems, page+1, result, start), nil
			}
		}
	}

	return paginatedResult(allItems, maxPages, nil, start), nil
}

func extractResponseItems(output map[string]any) ([]any, bool) {
	for _, key := range []string{"result", "items", "data"} {
		if items, ok := output[key].([]any); ok {
			return items, true
		}
	}
	return nil, false
}

func paginatedResult(items []any, pages int, lastResult *AdapterResult, start time.Time) *AdapterResult {
	headers := map[string]string{}
	httpStatus := 200
	if lastResult != nil {
		headers = lastResult.Headers
		httpStatus = lastResult.HTTPStatus
	}
	return &AdapterResult{
		Output:     map[string]any{"items": items, "_pagination": map[string]any{"pages": pages, "total_items": len(items)}},
		HTTPStatus: httpStatus,
		Headers:    headers,
		DurationMS: time.Since(start).Milliseconds(),
	}
}

func (a *HTTPAdapter) executeSingle(ctx context.Context, req AdapterRequest) (*AdapterResult, error) {
	start := time.Now()
	method := strings.ToUpper(strings.TrimSpace(req.Action.Adapter.Method))
	if method == "" {
		method = http.MethodGet
	}

	resolvedURL, err := resolveURL(req.Action.Adapter.BaseURL, req.Action.Adapter.URLTemplate, req.Input)
	if err != nil {
		return nil, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil)
	}

	payload := filterInputBySchema(req.Input, req.Action.InputSchema)
	if payload == nil {
		payload = map[string]any{}
	}

	if err := injectBodyCredential(payload, req.Action.Auth, req.Credentials); err != nil {
		return nil, err
	}

	queryValues, queryErr := mergeQuery(req.Action.Adapter.Query, req.Input, req.Action.Auth, req.Credentials)
	if queryErr != nil {
		return nil, queryErr
	}

	u, err := url.Parse(resolvedURL)
	if err != nil {
		return nil, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil)
	}
	if u.Scheme != "https" && !req.Action.Adapter.AllowInsecure {
		if !isLoopbackHost(u.Hostname()) {
			return nil, actions.NewExecutionError(actions.ErrValidationFailed, fmt.Sprintf("insecure URL scheme %q: only https is allowed for remote hosts (set allow_insecure: true to override)", u.Scheme), http.StatusBadRequest, false, nil)
		}
	}
	if len(queryValues) > 0 {
		existing := u.Query()
		for k, v := range queryValues {
			existing.Set(k, v)
		}
		u.RawQuery = existing.Encode()
	}

	bodyBytes, err := buildBody(method, payload, req.Action.Adapter.RequestBody)
	if err != nil {
		return nil, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil)
	}

	retryCfg := req.Action.Adapter.Retry
	maxAttempts := retryCfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	backoff := retryCfg.BackoffMS
	if backoff <= 0 {
		backoff = 100
	}

	if req.Timeout > 0 {
		var overallCancel context.CancelFunc
		ctx, overallCancel = context.WithTimeout(ctx, req.Timeout)
		defer overallCancel()
	}

	var lastStatus int
	var lastBody []byte
	var lastHeaders map[string]string

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		perAttemptTimeout := req.Action.Adapter.Timeout
		if perAttemptTimeout <= 0 {
			perAttemptTimeout = defaultAdapterAttemptTimeout
		}
		attemptCtx, cancel := context.WithTimeout(ctx, perAttemptTimeout)

		httpReq, reqErr := http.NewRequestWithContext(attemptCtx, method, u.String(), bytes.NewReader(bodyBytes))
		if reqErr != nil {
			cancel()
			return nil, actions.NewExecutionError(actions.ErrValidationFailed, reqErr.Error(), http.StatusBadRequest, false, nil)
		}

		if len(bodyBytes) > 0 {
			httpReq.Header.Set("Content-Type", "application/json")
		}

		for key, value := range req.Action.Adapter.Headers {
			if templateHasUnresolvedRef(key, req.Input) || templateHasUnresolvedRef(value, req.Input) {
				continue
			}
			resolvedKey := strings.TrimSpace(resolveTemplateString(key, req.Input, templateContextHeader))
			if resolvedKey == "" {
				continue
			}
			resolved := resolveTemplateString(value, req.Input, templateContextHeader)
			httpReq.Header.Set(resolvedKey, resolved)
		}

		if authErr := injectHeaders(httpReq, req.Action.Auth, req.Credentials); authErr != nil {
			cancel()
			return nil, authErr
		}

		resp, doErr := a.client.Do(httpReq)
		if doErr != nil {
			cancel()
			isTimeout := attemptCtx.Err() == context.DeadlineExceeded || isTimeoutError(doErr)
			if attempt < maxAttempts {
				if !sleepWithContext(ctx, time.Duration(backoff*attempt)*time.Millisecond) {
					return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, ctx.Err().Error(), http.StatusGatewayTimeout, true, nil)
				}
				continue
			}
			if isTimeout {
				return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "adapter request timed out", http.StatusGatewayTimeout, true, nil)
			}
			return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, doErr.Error(), http.StatusBadGateway, true, nil)
		}

		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, defaultMaxResponseBodyBytes+1))
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, readErr.Error(), http.StatusBadGateway, true, nil)
		}
		if int64(len(respBody)) > defaultMaxResponseBodyBytes {
			return nil, actions.NewExecutionError(
				actions.ErrDownstreamUnavailable,
				fmt.Sprintf("adapter response exceeded %d bytes", defaultMaxResponseBodyBytes),
				http.StatusBadGateway,
				false,
				map[string]any{"max_response_body_bytes": defaultMaxResponseBodyBytes},
			)
		}

		lastStatus = resp.StatusCode
		lastBody = respBody
		lastHeaders = toHeaderMap(resp.Header)

		if shouldRetry(resp.StatusCode, retryCfg) && attempt < maxAttempts {
			retryWait := time.Duration(backoff*attempt) * time.Millisecond
			if resp.StatusCode == http.StatusTooManyRequests {
				if parsed := parseRetryAfter(resp.Header.Get("Retry-After")); parsed > 0 {
					retryWait = parsed
				}
			}
			if !sleepWithContext(ctx, retryWait) {
				return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, ctx.Err().Error(), http.StatusGatewayTimeout, true, nil)
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			code := mapHTTPError(resp.StatusCode, req.Action.ErrorMapping)
			return nil, actions.NewExecutionError(
				code,
				errorMessage(respBody, resp.StatusCode, req.Action.ErrorMapping),
				resp.StatusCode,
				resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500,
				map[string]any{"status": resp.StatusCode},
			)
		}

		output, parseErr := normalizeOutput(respBody, req.Action.Adapter.Response.Extract)
		if parseErr != nil {
			return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, parseErr.Error(), http.StatusBadGateway, false, nil)
		}

		return &AdapterResult{
			Output:     output,
			HTTPStatus: resp.StatusCode,
			Headers:    lastHeaders,
			DurationMS: time.Since(start).Milliseconds(),
			Retryable:  false,
			RawBody:    respBody,
		}, nil
	}

	code := mapHTTPError(lastStatus, req.Action.ErrorMapping)
	return nil, actions.NewExecutionError(code, errorMessage(lastBody, lastStatus, req.Action.ErrorMapping), lastStatus, lastStatus == 429 || lastStatus >= 500, nil)
}

func resolveURL(baseURL, tmpl string, values map[string]any) (string, error) {
	resolved := resolveTemplateString(tmpl, values, templateContextPath)
	if strings.Contains(resolved, "{") || strings.Contains(resolved, "}") {
		return "", fmt.Errorf("missing url template variables")
	}

	if isHTTPURL(resolved) {
		if strings.TrimSpace(baseURL) != "" {
			if err := validateResolvedHostMatchesBase(baseURL, resolved); err != nil {
				return "", err
			}
		}
		return resolved, nil
	}

	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("base url is required for relative templates")
	}
	base := strings.TrimSuffix(baseURL, "/")
	path := resolved
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path, nil
}

func validateResolvedHostMatchesBase(baseURL, resolved string) error {
	baseU, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	resolvedU, err := url.Parse(resolved)
	if err != nil {
		return fmt.Errorf("invalid resolved URL: %w", err)
	}
	if !strings.EqualFold(baseU.Scheme, resolvedU.Scheme) {
		return fmt.Errorf("resolved URL scheme %q does not match base URL scheme %q", resolvedU.Scheme, baseU.Scheme)
	}
	if !strings.EqualFold(baseU.Hostname(), resolvedU.Hostname()) {
		return fmt.Errorf("resolved URL host %q does not match base URL host %q", resolvedU.Hostname(), baseU.Hostname())
	}
	if !strings.EqualFold(effectivePort(baseU), effectivePort(resolvedU)) {
		return fmt.Errorf("resolved URL port %q does not match base URL port %q", resolvedU.Port(), baseU.Port())
	}
	return nil
}

type templateContext string

const (
	templateContextPath   templateContext = "path"
	templateContextQuery  templateContext = "query"
	templateContextHeader templateContext = "header"
)

var templateRefPattern = regexp.MustCompile(`\{([a-zA-Z_]\w*)\}`)

func templateHasUnresolvedRef(tmpl string, input map[string]any) bool {
	for _, match := range templateRefPattern.FindAllStringSubmatch(tmpl, -1) {
		key := match[1]
		if _, ok := input[key]; !ok {
			return true
		}
	}
	return false
}

func resolveTemplateString(tmpl string, values map[string]any, context templateContext) string {
	out := tmpl
	for key, value := range values {
		placeholder := "{" + key + "}"
		if !strings.Contains(tmpl, placeholder) {
			continue
		}
		replacement := toString(value)
		if context == templateContextPath {
			replacement = url.PathEscape(replacement)
		}
		out = strings.ReplaceAll(out, placeholder, replacement)
	}
	return out
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func positiveIntFromAny(v any) (int, bool) {
	maxInt := int(^uint(0) >> 1)
	switch n := v.(type) {
	case int:
		return n, n > 0
	case int64:
		if n <= 0 || n > int64(maxInt) {
			return 0, false
		}
		return int(n), true
	case float64:
		if n <= 0 || math.IsNaN(n) || math.IsInf(n, 0) || n > float64(maxInt) {
			return 0, false
		}
		return int(n), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil || parsed <= 0 || parsed > int64(maxInt) {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func buildBody(method string, payload map[string]any, requestBodyTemplate string) ([]byte, error) {
	switch method {
	case http.MethodGet, http.MethodHead:
		return nil, nil
	default:
		if strings.TrimSpace(requestBodyTemplate) != "" {
			return resolveBodyTemplate(requestBodyTemplate, payload)
		}
		if payload == nil {
			payload = map[string]any{}
		}
		if raw, ok := payload["body"]; ok {
			return json.Marshal(raw)
		}
		return json.Marshal(payload)
	}
}

func filterInputBySchema(input map[string]any, schema *actions.Schema) map[string]any {
	if input == nil {
		return nil
	}
	if schema != nil && schema.AdditionalProperties {
		return cloneAnyMap(input)
	}
	if schema == nil || schema.Properties == nil {
		return cloneAnyMap(input)
	}
	out := make(map[string]any, len(schema.Properties))
	for key := range schema.Properties {
		if value, ok := input[key]; ok {
			out[key] = value
		}
	}
	return out
}

func isHTTPURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	return scheme == "http" || scheme == "https"
}

func resolveBodyTemplate(tmpl string, input map[string]any) ([]byte, error) {
	var parsed any
	if err := json.Unmarshal([]byte(tmpl), &parsed); err != nil {
		return nil, fmt.Errorf("invalid request body template: %w", err)
	}
	resolved := resolveTemplateAny(parsed, input)
	return json.Marshal(resolved)
}

func resolveTemplateValues(tmpl map[string]any, input map[string]any) map[string]any {
	out := make(map[string]any, len(tmpl))
	for k, v := range tmpl {
		resolved := resolveTemplateAny(v, input)
		if resolved != nil {
			out[k] = resolved
		}
	}
	return out
}

func resolveTemplateAny(v any, input map[string]any) any {
	switch val := v.(type) {
	case string:
		return resolveBodyTemplateValue(val, input)
	case map[string]any:
		return resolveTemplateValues(val, input)
	case []any:
		resolved := make([]any, len(val))
		for i, item := range val {
			resolved[i] = resolveTemplateAny(item, input)
		}
		return resolved
	default:
		return v
	}
}

func resolveBodyTemplateValue(s string, input map[string]any) any {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") && strings.Count(trimmed, "{") == 1 {
		key := trimmed[1 : len(trimmed)-1]
		if val, ok := input[key]; ok {
			return val
		}
		return nil
	}
	out := s
	for key, value := range input {
		placeholder := "{" + key + "}"
		if strings.Contains(s, placeholder) {
			out = strings.ReplaceAll(out, placeholder, fmt.Sprintf("%v", value))
		}
	}
	return out
}

func injectHeaders(req *http.Request, auth actions.AuthRequirement, creds *actions.ResolvedCredentialSet) *actions.ExecutionError {
	auth.Type = normalizeAuthType(auth.Type)
	if creds != nil {
		for k, v := range creds.Headers {
			req.Header.Set(k, v)
		}
	}

	if auth.Type == actions.AuthTypeNone {
		return nil
	}
	if creds == nil {
		if auth.Optional {
			return nil
		}
		return actions.NewExecutionError(actions.ErrCredentialMissing, "credentials not resolved", http.StatusUnauthorized, false, nil)
	}

	switch auth.Type {
	case actions.AuthTypeBearer, actions.AuthTypeOAuth2, actions.AuthTypeSession:
		token := creds.Token
		if token == "" {
			token = creds.APIKey
		}
		if token == "" {
			return actions.NewExecutionError(actions.ErrCredentialMissing, "missing bearer token", http.StatusUnauthorized, false, nil)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	case actions.AuthTypeAPIKey:
		apiKey := creds.APIKey
		if apiKey == "" {
			apiKey = creds.Token
		}
		if apiKey == "" {
			return actions.NewExecutionError(actions.ErrCredentialMissing, "missing api key", http.StatusUnauthorized, false, nil)
		}
		name := auth.HeaderName
		if name == "" {
			name = "X-API-Key"
		}
		prefix := auth.Prefix
		req.Header.Set(name, prefix+apiKey)
	case actions.AuthTypeHeader:
		name := auth.HeaderName
		if name == "" {
			return actions.NewExecutionError(actions.ErrValidationFailed, "auth header name required", http.StatusBadRequest, false, nil)
		}
		value := creds.Token
		if value == "" {
			value = creds.APIKey
		}
		if value == "" {
			return actions.NewExecutionError(actions.ErrCredentialMissing, "missing header credential", http.StatusUnauthorized, false, nil)
		}
		req.Header.Set(name, auth.Prefix+value)
	case actions.AuthTypeBasic:
		if creds.Username == "" && creds.Password == "" {
			return actions.NewExecutionError(actions.ErrCredentialMissing, "missing basic auth credentials", http.StatusUnauthorized, false, nil)
		}
		req.SetBasicAuth(creds.Username, creds.Password)
	}

	return nil
}

func injectBodyCredential(payload map[string]any, auth actions.AuthRequirement, creds *actions.ResolvedCredentialSet) *actions.ExecutionError {
	auth.Type = normalizeAuthType(auth.Type)
	if auth.Type != actions.AuthTypeBody {
		if creds != nil {
			maps.Copy(payload, creds.Body)
		}
		return nil
	}
	if creds == nil {
		if auth.Optional {
			return nil
		}
		return actions.NewExecutionError(actions.ErrCredentialMissing, "credentials not resolved", http.StatusUnauthorized, false, nil)
	}
	field := auth.BodyField
	if field == "" {
		field = "token"
	}
	value := creds.Token
	if value == "" {
		value = creds.APIKey
	}
	if value == "" {
		return actions.NewExecutionError(actions.ErrCredentialMissing, "missing body credential", http.StatusUnauthorized, false, nil)
	}
	maps.Copy(payload, creds.Body)
	payload[field] = auth.Prefix + value
	return nil
}

func mergeQuery(config map[string]string, input map[string]any, auth actions.AuthRequirement, creds *actions.ResolvedCredentialSet) (map[string]string, *actions.ExecutionError) {
	auth.Type = normalizeAuthType(auth.Type)
	out := map[string]string{}
	for key, value := range config {
		if templateHasUnresolvedRef(key, input) || templateHasUnresolvedRef(value, input) {
			continue
		}
		resolvedKey := strings.TrimSpace(resolveTemplateString(key, input, templateContextQuery))
		if resolvedKey == "" {
			continue
		}
		resolved := resolveTemplateString(value, input, templateContextQuery)
		out[resolvedKey] = resolved
	}
	if creds != nil {
		maps.Copy(out, creds.Query)
	}
	if auth.Type == actions.AuthTypeQuery {
		if creds == nil {
			if auth.Optional {
				return out, nil
			}
			return nil, actions.NewExecutionError(actions.ErrCredentialMissing, "credentials not resolved", http.StatusUnauthorized, false, nil)
		}
		key := auth.QueryName
		if key == "" {
			key = "api_key"
		}
		value := creds.APIKey
		if value == "" {
			value = creds.Token
		}
		if value == "" {
			return nil, actions.NewExecutionError(actions.ErrCredentialMissing, "missing query credential", http.StatusUnauthorized, false, nil)
		}
		out[key] = auth.Prefix + value
	}
	return out, nil
}

func normalizeOutput(body []byte, extract string) (map[string]any, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return map[string]any{"raw": string(body)}, nil
	}

	if strings.TrimSpace(extract) != "" {
		value, ok := extractByPath(parsed, extract)
		if !ok {
			return nil, fmt.Errorf("extract path %q not found", extract)
		}
		if out, ok := value.(map[string]any); ok {
			return out, nil
		}
		return map[string]any{"result": value}, nil
	}

	if out, ok := parsed.(map[string]any); ok {
		return out, nil
	}
	return map[string]any{"result": parsed}, nil
}

func extractByPath(value any, path string) (any, bool) {
	parts := strings.Split(strings.TrimPrefix(path, "."), ".")
	current := value
	for _, part := range parts {
		if part == "" {
			continue
		}
		next, ok := extractSegment(current, part)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func extractSegment(value any, segment string) (any, bool) {
	key := segment
	index := -1
	if open := strings.Index(segment, "["); open >= 0 && strings.HasSuffix(segment, "]") {
		key = segment[:open]
		idx, err := strconv.Atoi(segment[open+1 : len(segment)-1])
		if err != nil {
			return nil, false
		}
		index = idx
	}

	var current any = value
	if key != "" {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := obj[key]
		if !ok {
			return nil, false
		}
		current = next
	}

	if index >= 0 {
		arr, ok := current.([]any)
		if !ok || index < 0 || index >= len(arr) {
			return nil, false
		}
		current = arr[index]
	}

	return current, true
}

func mapHTTPError(status int, _ map[int]string) string {
	switch status {
	case http.StatusUnauthorized:
		return actions.ErrUnauthenticated
	case http.StatusForbidden:
		return actions.ErrUnauthorized
	case http.StatusTooManyRequests:
		return actions.ErrRateLimited
	default:
		if status >= 500 {
			return actions.ErrDownstreamUnavailable
		}
		return actions.ErrValidationFailed
	}
}

func errorMessage(respBody []byte, status int, custom map[int]string) string {
	if custom != nil {
		if msg, ok := custom[status]; ok && msg != "" {
			return msg
		}
	}
	return readErrorMessage(respBody)
}

func shouldRetry(status int, cfg actions.RetryConfig) bool {
	if cfg.MaxAttempts <= 1 {
		return false
	}
	if status == http.StatusTooManyRequests {
		return cfg.RetryOn429
	}
	if status >= 500 {
		return cfg.RetryOn5xx
	}
	return false
}

func toHeaderMap(h http.Header) map[string]string {
	out := map[string]string{}
	for key, values := range h {
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	return out
}

func parseRetryAfter(value string) time.Duration {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	cap := time.Duration(maxRetryAfterSeconds) * time.Second
	if seconds, err := strconv.Atoi(trimmed); err == nil && seconds > 0 {
		d := time.Duration(seconds) * time.Second
		if d > cap {
			d = cap
		}
		return d
	}
	if t, err := http.ParseTime(trimmed); err == nil {
		delay := time.Until(t)
		if delay <= 0 {
			return 0
		}
		if delay > cap {
			delay = cap
		}
		return delay
	}
	return 0
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	case []any:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprint(val)
		}
		return string(b)
	case map[string]any:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprint(val)
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	maps.Copy(out, in)
	return out
}

func readErrorMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "downstream request failed"
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err == nil {
		if msg, ok := parsed["message"].(string); ok && msg != "" {
			return msg
		}
		if errMsg, ok := parsed["error"].(string); ok && errMsg != "" {
			return errMsg
		}
	}
	for len(trimmed) > 300 {
		_, size := utf8.DecodeLastRuneInString(trimmed)
		trimmed = trimmed[:len(trimmed)-size]
	}
	return trimmed
}

func normalizeAuthType(authType actions.AuthType) actions.AuthType {
	if authType == "" {
		return actions.AuthTypeNone
	}
	return authType
}

func isTimeoutError(err error) bool {
	if netErr, ok := err.(interface{ Timeout() bool }); ok {
		return netErr.Timeout()
	}
	return false
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

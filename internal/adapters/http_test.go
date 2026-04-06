package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
)

func TestNewHTTPAdapterDefaultTimeout(t *testing.T) {
	adapter := NewHTTPAdapter(nil)
	if adapter.client == nil {
		t.Fatal("expected default http client")
	}
	if adapter.client.Timeout != 0 {
		t.Fatalf("expected no client-level timeout (context controls deadline), got %s", adapter.client.Timeout)
	}
}

func TestHTTPAdapterSuccessGetWithBearer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.Client())
	res, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{Type: "http", Method: "GET", URLTemplate: server.URL + "/me"},
			Auth:    actions.AuthRequirement{Type: actions.AuthTypeBearer},
		},
		Credentials: &actions.ResolvedCredentialSet{Token: "token-123"},
		Input:       map[string]any{},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("expected 200, got %d", res.HTTPStatus)
	}
}

func TestSecureRedirectPolicyStripsCredentialLikeHeaders(t *testing.T) {
	originReq := httptest.NewRequest(http.MethodGet, "https://api.example.com/start", nil)
	redirectReq := httptest.NewRequest(http.MethodGet, "https://other.example.com/next", nil)
	redirectReq.Header.Set("Authorization", "Bearer secret")
	redirectReq.Header.Set("X-Access-Token", "tok-123")
	redirectReq.Header.Set("Cookie", "session=keep")
	redirectReq.Header.Set("Content-Type", "application/json")

	if err := secureRedirectPolicy(redirectReq, []*http.Request{originReq}); err != nil {
		t.Fatalf("secureRedirectPolicy() error = %v", err)
	}
	if got := redirectReq.Header.Get("Authorization"); got != "" {
		t.Fatalf("expected Authorization stripped, got %q", got)
	}
	if got := redirectReq.Header.Get("X-Access-Token"); got != "" {
		t.Fatalf("expected X-Access-Token stripped, got %q", got)
	}
	if got := redirectReq.Header.Get("Cookie"); got != "" {
		t.Fatalf("expected Cookie stripped on cross-host redirect, got %q", got)
	}
	if got := redirectReq.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type preserved, got %q", got)
	}
}

func TestHTTPAdapterSuccessPostJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if payload["title"] != "hello" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.Client())
	res, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{Type: "http", Method: "POST", URLTemplate: server.URL + "/issues"},
		},
		Input: map[string]any{"title": "hello"},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.HTTPStatus != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.HTTPStatus)
	}
}

func TestHTTPAdapterPostBodyOmitsUndeclaredSchemaFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if payload["title"] != "hello" {
			t.Fatalf("unexpected title payload: %+v", payload)
		}
		if _, ok := payload["inject"]; ok {
			t.Fatalf("did not expect undeclared input field in body payload: %+v", payload)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.Client())
	res, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			InputSchema: &actions.Schema{Properties: map[string]*actions.Schema{
				"title": {Type: "string"},
			}},
			Adapter: actions.AdapterConfig{Type: "http", Method: "POST", URLTemplate: server.URL + "/issues"},
		},
		Input: map[string]any{"title": "hello", "inject": true},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.HTTPStatus != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.HTTPStatus)
	}
}

func TestHTTPAdapterPostBodyPreservesUndeclaredFieldsForFreeformSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if payload["title"] != "hello" {
			t.Fatalf("unexpected title payload: %+v", payload)
		}
		if _, ok := payload["inject"]; !ok {
			t.Fatalf("expected undeclared input field in freeform payload: %+v", payload)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.Client())
	res, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			InputSchema: &actions.Schema{Properties: map[string]*actions.Schema{
				"title": {Type: "string"},
			}, AdditionalProperties: true},
			Adapter: actions.AdapterConfig{Type: "http", Method: "POST", URLTemplate: server.URL + "/issues"},
		},
		Input: map[string]any{"title": "hello", "inject": true},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.HTTPStatus != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.HTTPStatus)
	}
}

func TestHTTPAdapterURLTemplateAndCustomHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octo/kimbap/issues" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Tenant") != "tenant-1" {
			t.Fatalf("missing custom header")
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.Client())
	_, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				Type:        "http",
				Method:      "GET",
				URLTemplate: server.URL + "/repos/{owner}/{repo}/issues",
				Headers:     map[string]string{"X-Tenant": "{tenant}"},
			},
		},
		Input: map[string]any{"owner": "octo", "repo": "kimbap", "tenant": "tenant-1"},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHTTPAdapterErrorStatusMapping(t *testing.T) {
	t.Run("401 maps to unauthenticated", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"bad token"}`))
		}))
		defer server.Close()

		adapter := NewHTTPAdapter(server.Client())
		_, err := adapter.Execute(context.Background(), AdapterRequest{
			Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Type: "http", URLTemplate: server.URL}},
		})
		execErr := actions.AsExecutionError(err)
		if execErr == nil || execErr.Code != actions.ErrUnauthenticated {
			t.Fatalf("expected unauthenticated, got %+v", execErr)
		}
	})

	t.Run("429 maps to rate_limited", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"slow down"}`))
		}))
		defer server.Close()

		adapter := NewHTTPAdapter(server.Client())
		_, err := adapter.Execute(context.Background(), AdapterRequest{
			Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Type: "http", URLTemplate: server.URL}},
		})
		execErr := actions.AsExecutionError(err)
		if execErr == nil || execErr.Code != actions.ErrRateLimited {
			t.Fatalf("expected rate limited, got %+v", execErr)
		}
	})
}

func TestHTTPAdapterTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.Client())
	_, err := adapter.Execute(context.Background(), AdapterRequest{
		Action:  actions.ActionDefinition{Adapter: actions.AdapterConfig{Type: "http", URLTemplate: server.URL}},
		Timeout: 10 * time.Millisecond,
	})

	if err == nil {
		t.Fatal("expected timeout error")
	}
	execErr := actions.AsExecutionError(err)
	if execErr.Code != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected downstream unavailable, got %+v", execErr)
	}
}

func TestHTTPAdapterRejectsOversizedResponseBody(t *testing.T) {
	oversized := bytes.Repeat([]byte("x"), int(defaultMaxResponseBodyBytes)+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(oversized)
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(server.Client())
	_, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Type: "http", URLTemplate: server.URL}},
	})
	if err == nil {
		t.Fatal("expected oversized response error")
	}
	execErr := actions.AsExecutionError(err)
	if execErr == nil {
		t.Fatal("expected execution error")
	}
	if execErr.Code != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected downstream unavailable, got %s", execErr.Code)
	}
	if execErr.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", execErr.HTTPStatus)
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"", 0},
		{"0", 0},
		{"-5", 0},
		{"5", 5 * time.Second},
		{"60", 60 * time.Second},
		{"300", time.Duration(maxRetryAfterSeconds) * time.Second},
		{"not-a-number", 0},
	}
	for _, tt := range tests {
		got := parseRetryAfter(tt.input)
		if got != tt.expected {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestPaginationUsesMaxPages(t *testing.T) {
	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		if page <= 2 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items":       []any{map[string]any{"id": page}},
				"next_cursor": "cursor-next",
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []any{},
			})
		}
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(nil)
	result, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				Type:        "http",
				Method:      "GET",
				BaseURL:     server.URL,
				URLTemplate: "/items",
			},
			Pagination: &actions.PaginationConfig{
				Style:          "cursor",
				MaxPages:       2,
				ResponseCursor: "next_cursor",
			},
		},
		Input: map[string]any{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pagination, ok := result.Output["_pagination"].(map[string]any)
	if !ok {
		t.Fatal("expected _pagination in output")
	}
	pages, _ := pagination["pages"].(int)
	if pages != 2 {
		t.Fatalf("expected 2 pages (MaxPages cap), got %d", pages)
	}
}

func TestPaginationCapsRequestedPageLimit(t *testing.T) {
	seenLimit := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenLimit = r.URL.Query().Get("limit")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(nil)
	_, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				Type:        "http",
				Method:      "GET",
				BaseURL:     server.URL,
				URLTemplate: "/items",
				Query: map[string]string{
					"limit":  "{limit}",
					"offset": "{offset}",
				},
			},
			Pagination: &actions.PaginationConfig{Style: "offset", MaxPages: 1, LimitParam: "limit", OffsetParam: "offset"},
		},
		Input: map[string]any{"limit": hardMaxPaginationPageLimit + 100},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seenLimit != strconv.Itoa(hardMaxPaginationPageLimit) {
		t.Fatalf("expected limit query capped to %d, got %q", hardMaxPaginationPageLimit, seenLimit)
	}
}

func TestPaginationFailsWhenTotalItemsExceedHardCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		items := make([]any, 0, hardMaxPaginationPageLimit)
		for i := 0; i < hardMaxPaginationPageLimit; i++ {
			items = append(items, map[string]any{"id": i})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	}))
	defer server.Close()

	adapter := NewHTTPAdapter(nil)
	_, err := adapter.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter:    actions.AdapterConfig{Type: "http", Method: "GET", BaseURL: server.URL, URLTemplate: "/items"},
			Pagination: &actions.PaginationConfig{Style: "offset", MaxPages: 6, LimitParam: "limit", OffsetParam: "offset"},
		},
		Input: map[string]any{"limit": hardMaxPaginationPageLimit},
	})
	if err == nil {
		t.Fatal("expected error when total paginated items exceed hard cap")
	}
	execErr := actions.AsExecutionError(err)
	if execErr == nil {
		t.Fatalf("expected execution error, got %v", err)
	}
	if execErr.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", execErr.HTTPStatus)
	}
}

func TestBuildBodyWithTemplate(t *testing.T) {
	input := map[string]any{
		"name":  "alice",
		"age":   30,
		"tags":  []string{"admin", "user"},
		"extra": "ignored",
	}

	body, err := buildBody("POST", input, `{"user_name":"{name}","user_age":"{age}","user_tags":"{tags}"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if parsed["user_name"] != "alice" {
		t.Fatalf("expected user_name=alice, got %v", parsed["user_name"])
	}
	if parsed["user_age"] != float64(30) {
		t.Fatalf("expected user_age=30, got %v (type %T)", parsed["user_age"], parsed["user_age"])
	}
}

func TestHTTPAdapterValidateAcceptsUppercaseHTTPSAbsoluteTemplate(t *testing.T) {
	adapter := NewHTTPAdapter(nil)
	err := adapter.Validate(actions.ActionDefinition{Adapter: actions.AdapterConfig{URLTemplate: "HTTPS://example.com/items"}})
	if err != nil {
		t.Fatalf("expected uppercase HTTPS template to be treated as absolute URL, got %v", err)
	}
}

func TestResolveURLAcceptsUppercaseHTTPSAbsoluteTemplate(t *testing.T) {
	resolved, err := resolveURL("", "HTTPS://example.com/items", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "HTTPS://example.com/items" {
		t.Fatalf("expected resolved URL to remain absolute, got %q", resolved)
	}
}

func TestBuildBodyWithTemplateArray(t *testing.T) {
	input := map[string]any{"id": "item-1"}
	body, err := buildBody("POST", input, `{"items":[{"item_id":"{id}"}]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	items, ok := parsed["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected items array with 1 element, got %v", parsed["items"])
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first item to be object, got %T", items[0])
	}
	if first["item_id"] != "item-1" {
		t.Fatalf("expected item_id=item-1, got %v", first["item_id"])
	}
}

func TestBuildBodyInvalidTemplate(t *testing.T) {
	_, err := buildBody("POST", map[string]any{}, `not-valid-json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON template")
	}
}

func TestBuildBodyEmptyTemplateUsesInput(t *testing.T) {
	input := map[string]any{"key": "value"}
	body, err := buildBody("POST", input, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if parsed["key"] != "value" {
		t.Fatalf("expected key=value, got %v", parsed["key"])
	}
}

func TestBuildBodyTemplateOmitsMissingOptional(t *testing.T) {
	input := map[string]any{"name": "alice"}
	body, err := buildBody("POST", input, `{"name":"{name}","age":"{age}"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if parsed["name"] != "alice" {
		t.Fatalf("expected name=alice, got %v", parsed["name"])
	}
	if _, exists := parsed["age"]; exists {
		t.Fatalf("expected age to be omitted when not in input, got %v", parsed["age"])
	}
}

func TestMergeQuery_SkipsUnresolvedPlaceholders(t *testing.T) {
	config := map[string]string{"offset": "{offset}", "limit": "{limit}"}
	input := map[string]any{"limit": 50}

	out, err := mergeQuery(config, input, actions.AuthRequirement{}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, exists := out["offset"]; exists {
		t.Fatalf("expected unresolved placeholder key to be skipped, got %q", out["offset"])
	}
	if out["limit"] != "50" {
		t.Fatalf("expected limit=50, got %q", out["limit"])
	}
}

func TestMergeQuery_IncludesResolvedValues(t *testing.T) {
	config := map[string]string{"q": "{q}", "count": "{count}"}
	input := map[string]any{"q": "hello", "count": 10}

	out, err := mergeQuery(config, input, actions.AuthRequirement{}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out["q"] != "hello" {
		t.Fatalf("expected q=hello, got %q", out["q"])
	}
	if out["count"] != "10" {
		t.Fatalf("expected count=10, got %q", out["count"])
	}
}

func TestToString_ArrayGetsJSONEncoded(t *testing.T) {
	got := toString([]any{"message", "callback_query"})
	want := `["message","callback_query"]`
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestToString_MapGetsJSONEncoded(t *testing.T) {
	got := toString(map[string]any{"key": "val"})
	want := `{"key":"val"}`
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestMergeQuery_LiteralValuesPassThrough(t *testing.T) {
	config := map[string]string{"stream": "false"}

	out, err := mergeQuery(config, nil, actions.AuthRequirement{}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out["stream"] != "false" {
		t.Fatalf("expected literal query value to pass through, got %q", out["stream"])
	}
}

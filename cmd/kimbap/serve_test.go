package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/api"
	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/webhooks"
)

func TestBuildServeServerOptionsEnablesWebhookRoutes(t *testing.T) {
	withoutDispatcher := api.NewServer(":0", nil)
	withoutTS := httptest.NewServer(withoutDispatcher.Router())
	defer withoutTS.Close()

	resp, err := http.Get(withoutTS.URL + "/v1/webhooks")
	if err != nil {
		t.Fatalf("request without dispatcher: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 without webhook dispatcher, got %d", resp.StatusCode)
	}

	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "serve-test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}

	withDispatcher := api.NewServer(":0", st, buildServeServerOptions(nil, nil, nil, false)...)
	withTS := httptest.NewServer(withDispatcher.Router())
	defer withTS.Close()

	resp2, err := http.Get(withTS.URL + "/v1/webhooks")
	if err != nil {
		t.Fatalf("request with dispatcher: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for protected webhook route when dispatcher is configured, got %d", resp2.StatusCode)
	}
}

func TestBuildServeServerOptionsDisablesConsoleByDefault(t *testing.T) {
	server := api.NewServer(":0", nil, buildServeServerOptions(nil, nil, nil, false)...)
	ts := httptest.NewServer(server.Router())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/console")
	if err != nil {
		t.Fatalf("request console route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when console is disabled, got %d", resp.StatusCode)
	}
}

func TestBuildServeServerOptionsEnablesConsoleWhenRequested(t *testing.T) {
	server := api.NewServer(":0", nil, buildServeServerOptions(nil, nil, nil, true)...)
	ts := httptest.NewServer(server.Router())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/console")
	if err != nil {
		t.Fatalf("request console route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when console is enabled, got %d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected text/html content type for console route, got %q", contentType)
	}
}

func TestBuildServeServerOptionsEnablesConsoleDeepLinkWithDot(t *testing.T) {
	server := api.NewServer(":0", nil, buildServeServerOptions(nil, nil, nil, true)...)
	ts := httptest.NewServer(server.Router())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/console/releases/v1.2", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request console deep link: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when console deep link is enabled, got %d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected text/html content type for console deep link, got %q", contentType)
	}
}

func TestBuildServeServerOptionsHydratesWebhookDataFromStore(t *testing.T) {
	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "serve-hydrate.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}

	if err := st.UpsertWebhookSubscription(context.Background(), &store.WebhookSubscriptionRecord{
		ID:         "wh_hydrated",
		TenantID:   "tenant-a",
		URL:        "https://example.com/hook",
		Secret:     "secret",
		EventsJSON: `[]`,
		Active:     true,
	}); err != nil {
		t.Fatalf("upsert webhook subscription: %v", err)
	}
	if err := st.WriteWebhookEvent(context.Background(), &store.WebhookEventRecord{
		ID:        "evt_hydrated",
		TenantID:  "tenant-a",
		Type:      "approval.expired",
		Timestamp: time.Now().UTC(),
		DataJSON:  `{"approval_id":"apr_1"}`,
	}); err != nil {
		t.Fatalf("write webhook event: %v", err)
	}

	rawToken := "ktk_hydrate_bootstrap"
	if err := st.CreateToken(context.Background(), newBootstrapTokenRecordForServeTest("tenant-a", "bootstrap-agent", rawToken)); err != nil {
		t.Fatalf("seed bootstrap token: %v", err)
	}

	dispatcher := webhooks.NewDispatcher()
	cleanupWebhookSink := configureWebhookDispatcherFromStore(context.Background(), dispatcher, st)
	defer cleanupWebhookSink()
	server := api.NewServer(":0", st, buildServeServerOptions(nil, nil, dispatcher, false)...)
	ts := httptest.NewServer(server.Router())
	defer ts.Close()

	reqWebhooks, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/webhooks", nil)
	reqWebhooks.Header.Set("Authorization", "Bearer "+rawToken)
	respWebhooks, err := http.DefaultClient.Do(reqWebhooks)
	if err != nil {
		t.Fatalf("list webhooks request: %v", err)
	}
	defer respWebhooks.Body.Close()
	if respWebhooks.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(respWebhooks.Body)
		t.Fatalf("expected 200, got %d body=%s", respWebhooks.StatusCode, string(b))
	}
	var webhooksPayload map[string]any
	if err := json.NewDecoder(respWebhooks.Body).Decode(&webhooksPayload); err != nil {
		t.Fatalf("decode webhooks payload: %v", err)
	}
	data, _ := webhooksPayload["data"].(map[string]any)
	items, _ := data["webhooks"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 hydrated webhook, got %d", len(items))
	}

	reqEvents, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/webhooks/events", nil)
	reqEvents.Header.Set("Authorization", "Bearer "+rawToken)
	respEvents, err := http.DefaultClient.Do(reqEvents)
	if err != nil {
		t.Fatalf("list webhook events request: %v", err)
	}
	defer respEvents.Body.Close()
	if respEvents.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(respEvents.Body)
		t.Fatalf("expected 200, got %d body=%s", respEvents.StatusCode, string(b))
	}
	var eventsPayload map[string]any
	if err := json.NewDecoder(respEvents.Body).Decode(&eventsPayload); err != nil {
		t.Fatalf("decode events payload: %v", err)
	}
	eventsData, _ := eventsPayload["data"].(map[string]any)
	events, _ := eventsData["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("expected 1 hydrated event, got %d", len(events))
	}
}

func TestStoreApprovalStoreAdapterPreservesInputRoundTrip(t *testing.T) {
	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "serve-approval-adapter.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}

	adapter := &storeApprovalStoreAdapter{st: st}
	now := time.Now().UTC()
	input := map[string]any{"title": "hello", "nested": map[string]any{"count": 1}}
	if err := adapter.Create(context.Background(), &approvals.ApprovalRequest{
		ID:                "apr_roundtrip",
		TenantID:          "tenant-a",
		RequestID:         "req_roundtrip",
		AgentName:         "agent-a",
		Service:           "github",
		Action:            "issues.create",
		Input:             input,
		Status:            approvals.StatusPending,
		RequiredApprovals: 2,
		CreatedAt:         now,
		ExpiresAt:         now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	got, err := adapter.Get(context.Background(), "apr_roundtrip")
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if got.Input == nil || got.Input["title"] != "hello" {
		t.Fatalf("expected input payload to round-trip, got %+v", got.Input)
	}
	if got.RequiredApprovals != 2 {
		t.Fatalf("expected required approvals 2, got %d", got.RequiredApprovals)
	}

	nowVote := time.Now().UTC()
	got.Votes = []approvals.ApprovalVote{{ApproverID: "approver-1", Decision: approvals.StatusApproved, VotedAt: nowVote}}
	if err := adapter.Update(context.Background(), got); err != nil {
		t.Fatalf("update approval with vote: %v", err)
	}

	updated, err := adapter.Get(context.Background(), "apr_roundtrip")
	if err != nil {
		t.Fatalf("get updated approval: %v", err)
	}
	if len(updated.Votes) != 1 || updated.Votes[0].ApproverID != "approver-1" {
		t.Fatalf("expected votes to persist, got %+v", updated.Votes)
	}

	items, err := adapter.ListPending(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("list pending approvals: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Input == nil || items[0].Input["title"] != "hello" {
		t.Fatalf("expected list input payload to round-trip, got %+v", items[0].Input)
	}
	if items[0].RequiredApprovals != 2 {
		t.Fatalf("expected list required approvals 2, got %d", items[0].RequiredApprovals)
	}
}

func TestStoreApprovalStoreAdapterListAllRespectsFilter(t *testing.T) {
	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "serve-approval-list-filter.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}

	adapter := &storeApprovalStoreAdapter{st: st}
	now := time.Now().UTC()
	seed := []*approvals.ApprovalRequest{
		{ID: "apr_1", TenantID: "tenant-a", RequestID: "req_1", AgentName: "agent-a", Service: "github", Action: "issues.create", Status: approvals.StatusPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour)},
		{ID: "apr_2", TenantID: "tenant-a", RequestID: "req_2", AgentName: "agent-b", Service: "slack", Action: "chat.post", Status: approvals.StatusPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour)},
		{ID: "apr_3", TenantID: "tenant-a", RequestID: "req_3", AgentName: "agent-a", Service: "github", Action: "repos.list", Status: approvals.StatusPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour)},
	}
	for _, item := range seed {
		if err := adapter.Create(context.Background(), item); err != nil {
			t.Fatalf("create approval %s: %v", item.ID, err)
		}
	}

	items, err := adapter.ListAll(context.Background(), "tenant-a", approvals.ApprovalFilter{AgentName: "agent-a", Service: "github", Limit: 1})
	if err != nil {
		t.Fatalf("list all approvals: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected filtered+limited list of 1, got %d", len(items))
	}
	if items[0].AgentName != "agent-a" || items[0].Service != "github" {
		t.Fatalf("unexpected filtered item: %+v", items[0])
	}
}

func newBootstrapTokenRecordForServeTest(tenantID string, agentName string, rawToken string) *store.TokenRecord {
	now := time.Now().UTC()
	sum := sha256.Sum256([]byte(rawToken))
	hintStart := max(len(rawToken)-4, 0)
	return &store.TokenRecord{
		ID:          "st_bootstrap_serve_test",
		TenantID:    tenantID,
		AgentName:   agentName,
		TokenHash:   hex.EncodeToString(sum[:]),
		DisplayHint: rawToken[hintStart:],
		Scopes:      `["*"]`,
		CreatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
		CreatedBy:   "bootstrap",
	}
}

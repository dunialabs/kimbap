package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
	"github.com/dunialabs/kimbap/internal/auth"
	"github.com/dunialabs/kimbap/internal/authstore"
	"github.com/dunialabs/kimbap/internal/config"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/dunialabs/kimbap/internal/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestServerHealthAndRequestID(t *testing.T) {
	ts, _ := newTestAPIServer(t)

	resp, err := http.Get(ts.URL + "/v1/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header")
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true, got %v", body["success"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", body["data"])
	}
	if data["status"] != "ok" {
		t.Fatalf("expected data.status=ok, got %v", data["status"])
	}
	expectedVersion := config.CLIVersion()
	if data["version"] != expectedVersion {
		t.Fatalf("expected data.version=%q, got %v", expectedVersion, data["version"])
	}
}

func TestStoreTokenAdapterValidateAndResolveUsesTokenIDAsPrincipalID(t *testing.T) {
	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-token-adapter-principal.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	rawToken := "ktk_test_token_value"
	hash := sha256.Sum256([]byte(rawToken))
	record := &store.TokenRecord{
		ID:          "st_token_identity",
		TenantID:    "tenant-a",
		AgentName:   "agent-a",
		TokenHash:   hex.EncodeToString(hash[:]),
		DisplayHint: "xxxx",
		Scopes:      `["read"]`,
		CreatedAt:   time.Now().UTC().Add(-1 * time.Minute),
		ExpiresAt:   time.Now().UTC().Add(30 * time.Minute),
		CreatedBy:   "test",
	}
	if err := st.CreateToken(context.Background(), record); err != nil {
		t.Fatalf("create token: %v", err)
	}

	adapter := authstore.NewTokenStoreAdapter(st)
	principal, err := adapter.ValidateAndResolve(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("ValidateAndResolve error: %v", err)
	}
	if principal == nil {
		t.Fatal("expected principal")
	}
	if principal.ID != record.ID {
		t.Fatalf("expected principal.ID=%q, got %q", record.ID, principal.ID)
	}
	if principal.TokenID != record.ID {
		t.Fatalf("expected principal.TokenID=%q, got %q", record.ID, principal.TokenID)
	}
	if principal.AgentName != record.AgentName {
		t.Fatalf("expected principal.AgentName=%q, got %q", record.AgentName, principal.AgentName)
	}
}

func TestServerListActions(t *testing.T) {
	ts, _ := newTestAPIServer(t)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/actions", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("actions request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode actions response: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("expected success=false, got %v", body["success"])
	}
	errBody, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", body["error"])
	}
	if errBody["code"] != "ERR_DOWNSTREAM_UNAVAILABLE" {
		t.Fatalf("expected ERR_DOWNSTREAM_UNAVAILABLE, got %v", errBody["code"])
	}
	if errBody["message"] != "internal server error" {
		t.Fatalf("expected sanitized internal server message, got %v", errBody["message"])
	}
}

func TestServerListActionsWithRuntime(t *testing.T) {
	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-actions-runtime.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	if err := st.CreateToken(context.Background(), newBootstrapTokenRecord("tenant-a", "bootstrap-agent", "ktk_bootstrap_token_for_tests")); err != nil {
		t.Fatalf("seed bootstrap token: %v", err)
	}

	server := NewServer(":0", st, WithRuntime(&runtimepkg.Runtime{ActionRegistry: staticActionRegistry{items: []actions.ActionDefinition{{Name: "apple-notes.list-notes", Namespace: "apple-notes"}}}}))
	ts := httptest.NewServer(server.Router())
	t.Cleanup(func() {
		ts.Close()
		_ = st.Close()
	})

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/actions", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("actions request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode actions response: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true, got %v", body["success"])
	}
	data, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", body["data"])
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 action, got %d", len(data))
	}
	item, ok := data[0].(map[string]any)
	if !ok {
		t.Fatalf("expected action item object, got %T", data[0])
	}
	auth, ok := item["Auth"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth object, got %T", item["Auth"])
	}
	if auth["Type"] != "" && auth["Type"] != "none" {
		t.Fatalf("expected public auth type only, got %v", auth["Type"])
	}
	if _, exists := auth["CredentialRef"]; exists {
		t.Fatalf("expected CredentialRef to be omitted from public list payload")
	}
	if _, exists := item["Adapter"]; exists {
		t.Fatalf("expected Adapter to be omitted from public list payload")
	}
}

func TestServerDescribeActionIsPublic(t *testing.T) {
	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-action-detail-runtime.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	server := NewServer(":0", st, WithRuntime(&runtimepkg.Runtime{ActionRegistry: staticActionRegistry{items: []actions.ActionDefinition{{
		Name:        "apple-notes.list-notes",
		Namespace:   "apple-notes",
		Description: "List Apple Notes",
	}}}}))
	ts := httptest.NewServer(server.Router())
	t.Cleanup(func() {
		ts.Close()
		_ = st.Close()
	})

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/actions/apple-notes/list-notes", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("describe action request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode action detail response: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true, got %v", body["success"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", body["data"])
	}
	if data["Name"] != "apple-notes.list-notes" {
		t.Fatalf("expected action name, got %v", data["Name"])
	}
	auth, ok := data["Auth"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth object, got %T", data["Auth"])
	}
	if auth["Type"] != "none" {
		t.Fatalf("expected public auth type none, got %v", auth["Type"])
	}
	if _, exists := auth["CredentialRef"]; exists {
		t.Fatalf("expected CredentialRef to be omitted from public detail payload")
	}
	if _, exists := data["Adapter"]; exists {
		t.Fatalf("expected Adapter to be omitted from public detail payload")
	}
	if _, exists := data["OutputSchema"]; exists {
		t.Fatalf("expected OutputSchema to be omitted from public detail payload")
	}
}

func TestServerRejectsTrailingJSONPayload(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	body := `{"schema":{"type":"object"},"input":{}}{"extra":1}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/actions/validate", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("validate request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload["success"] != false {
		t.Fatalf("expected success=false, got %v", payload["success"])
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", payload["error"])
	}
	if errBody["code"] != "ERR_VALIDATION_FAILED" {
		t.Fatalf("expected ERR_VALIDATION_FAILED, got %v", errBody["code"])
	}
}

func TestServerRejectsInvalidAuditTimestamp(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/audit?from=not-a-timestamp", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("audit request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload["success"] != false {
		t.Fatalf("expected success=false, got %v", payload["success"])
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", payload["error"])
	}
	if errBody["code"] != "ERR_VALIDATION_FAILED" {
		t.Fatalf("expected ERR_VALIDATION_FAILED, got %v", errBody["code"])
	}
}

func TestServerRejectsNegativeAuditPagination(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/audit?limit=-1&offset=-2", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("audit request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", payload["error"])
	}
	if errBody["code"] != "ERR_VALIDATION_FAILED" {
		t.Fatalf("expected ERR_VALIDATION_FAILED, got %v", errBody["code"])
	}
}

func TestServerRejectsNegativeActionLimit(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/actions?limit=-1", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("actions request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", payload["error"])
	}
	if errBody["code"] != "ERR_VALIDATION_FAILED" {
		t.Fatalf("expected ERR_VALIDATION_FAILED, got %v", errBody["code"])
	}
}

func TestServerAcceptsTrailingWhitespaceAfterJSONPayload(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	body := "{\"schema\":{\"type\":\"object\"},\"input\":{}}\n\t  "
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/actions/validate", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("validate request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}
}

func TestServerApproveAlreadyResolvedWithoutRuntimeIsIdempotent(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)

	approval := &store.ApprovalRecord{
		ID:        "apr_conflict_test",
		TenantID:  "tenant-a",
		RequestID: "req_conflict_test",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
	if err := st.CreateApproval(context.Background(), approval); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	approveURL := ts.URL + "/v1/approvals/" + approval.ID + ":approve"
	firstReq, _ := http.NewRequest(http.MethodPost, approveURL, nil)
	firstReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	firstResp, err := http.DefaultClient.Do(firstReq)
	if err != nil {
		t.Fatalf("first approve request: %v", err)
	}
	defer firstResp.Body.Close()
	if firstResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(firstResp.Body)
		t.Fatalf("expected first approve=200, got %d body=%s", firstResp.StatusCode, string(b))
	}

	secondReq, _ := http.NewRequest(http.MethodPost, approveURL, nil)
	secondReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	secondResp, err := http.DefaultClient.Do(secondReq)
	if err != nil {
		t.Fatalf("second approve request: %v", err)
	}
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("expected second approve=200, got %d", secondResp.StatusCode)
	}
}

func TestServerApproveExpiredReturnsGone(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)

	approval := &store.ApprovalRecord{
		ID:        "apr_expired_test",
		TenantID:  "tenant-a",
		RequestID: "req_expired_test",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		CreatedAt: time.Now().UTC().Add(-20 * time.Minute),
		ExpiresAt: time.Now().UTC().Add(-10 * time.Minute),
	}
	if err := st.CreateApproval(context.Background(), approval); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	approveURL := ts.URL + "/v1/approvals/" + approval.ID + ":approve"
	req, _ := http.NewRequest(http.MethodPost, approveURL, nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGone {
		t.Fatalf("expected approve=410 for expired request, got %d", resp.StatusCode)
	}
}

func TestServerApproveRacePathAlreadyResolvedMappedToConflict(t *testing.T) {
	base, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-race-conflict.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := base.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	approvalID := "apr_race_conflict"
	shim := &forcedApprovalStore{
		Store:       base,
		forcedGetID: approvalID,
		forcedGetRecord: &store.ApprovalRecord{
			ID:        approvalID,
			TenantID:  "tenant-a",
			RequestID: "req_race_conflict",
			AgentName: "agent-a",
			Service:   "github",
			Action:    "issues.create",
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		},
		forcedUpdateID:  approvalID,
		forcedUpdateErr: store.ErrApprovalAlreadyResolved,
	}

	ts, rawBootstrap := newTestAPIServerFromStore(t, shim)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approvalID+":approve", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 409, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", payload["error"])
	}
	if errBody["code"] != "ERR_VALIDATION_FAILED" {
		t.Fatalf("expected ERR_VALIDATION_FAILED, got %v", errBody["code"])
	}
}

func TestServerApproveRacePathExpiredMappedToGone(t *testing.T) {
	base, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-race-expired.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := base.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	approvalID := "apr_race_expired"
	shim := &forcedApprovalStore{
		Store:       base,
		forcedGetID: approvalID,
		forcedGetRecord: &store.ApprovalRecord{
			ID:        approvalID,
			TenantID:  "tenant-a",
			RequestID: "req_race_expired",
			AgentName: "agent-a",
			Service:   "github",
			Action:    "issues.create",
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		},
		forcedUpdateID:  approvalID,
		forcedUpdateErr: store.ErrApprovalExpired,
	}

	ts, rawBootstrap := newTestAPIServerFromStore(t, shim)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approvalID+":approve", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusGone {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 410, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", payload["error"])
	}
	if errBody["code"] != "ERR_APPROVAL_TIMEOUT" {
		t.Fatalf("expected ERR_APPROVAL_TIMEOUT, got %v", errBody["code"])
	}
}

func TestHandleExecuteActionEmitsApprovalRequestedWebhook(t *testing.T) {
	dispatcher := webhooks.NewDispatcher()
	server := &Server{
		runtime: &runtimepkg.Runtime{
			PolicyEvaluator:    staticPolicyEvaluator{},
			ApprovalManager:    staticApprovalManager{requestID: "apr_req_evt_1"},
			HeldExecutionStore: testHeldExecutionStore{},
		},
		webhookDispatcher: dispatcher,
	}

	body := strings.NewReader(`{"input": {}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/actions/github/issues.create:execute", body)
	req.Header.Set("Idempotency-Key", "idem-api-approval-test")
	req = req.WithContext(context.WithValue(req.Context(), contextKeyPrincipal, &auth.Principal{
		ID:        "approver-a",
		TenantID:  "tenant-a",
		AgentName: "agent-a",
		Scopes:    []string{"*"},
	}))
	req = req.WithContext(context.WithValue(req.Context(), contextKeyTenant, "tenant-a"))
	req = req.WithContext(context.WithValue(req.Context(), contextKeyRequestID, "req_evt_1"))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("service", "github")
	rctx.URLParams.Add("action", "issues.create")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	server.handleExecuteAction(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	events := dispatcher.RecentEventsByTenant("tenant-a", 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != webhooks.EventApprovalRequested {
		t.Fatalf("expected %q, got %q", webhooks.EventApprovalRequested, events[0].Type)
	}
	data, ok := events[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected event data map, got %T", events[0].Data)
	}
	if data["approval_id"] != "apr_req_evt_1" {
		t.Fatalf("expected approval_id apr_req_evt_1, got %v", data["approval_id"])
	}
}

func TestServerApproveEmitsWebhookEvent(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)
	dispatcher := webhooks.NewDispatcher()
	server := NewServer(":0", st, WithWebhookDispatcher(dispatcher))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	approval := &store.ApprovalRecord{
		ID:        "apr_emit_approved",
		TenantID:  "tenant-a",
		RequestID: "req_emit_approved",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
	if err := st.CreateApproval(context.Background(), approval); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approval.ID+":approve", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}

	events := dispatcher.RecentEventsByTenant("tenant-a", 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != webhooks.EventApprovalApproved {
		t.Fatalf("expected %q, got %q", webhooks.EventApprovalApproved, events[0].Type)
	}
}

func TestServerDenyEmitsWebhookEvent(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)
	dispatcher := webhooks.NewDispatcher()
	server := NewServer(":0", st, WithWebhookDispatcher(dispatcher))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	approval := &store.ApprovalRecord{
		ID:        "apr_emit_denied",
		TenantID:  "tenant-a",
		RequestID: "req_emit_denied",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
	if err := st.CreateApproval(context.Background(), approval); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approval.ID+":deny", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("deny request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}

	events := dispatcher.RecentEventsByTenant("tenant-a", 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != webhooks.EventApprovalDenied {
		t.Fatalf("expected %q, got %q", webhooks.EventApprovalDenied, events[0].Type)
	}
}

func TestServerExpiredApprovalEmitsWebhookEvent(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)
	dispatcher := webhooks.NewDispatcher()
	server := NewServer(":0", st, WithWebhookDispatcher(dispatcher))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	approval := &store.ApprovalRecord{
		ID:        "apr_emit_expired",
		TenantID:  "tenant-a",
		RequestID: "req_emit_expired",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		CreatedAt: time.Now().UTC().Add(-20 * time.Minute),
		ExpiresAt: time.Now().UTC().Add(-10 * time.Minute),
	}
	if err := st.CreateApproval(context.Background(), approval); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approval.ID+":approve", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGone {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 410, got %d body=%s", resp.StatusCode, string(b))
	}

	events := dispatcher.RecentEventsByTenant("tenant-a", 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != webhooks.EventApprovalExpired {
		t.Fatalf("expected %q, got %q", webhooks.EventApprovalExpired, events[0].Type)
	}
	data, ok := events[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected event data map, got %T", events[0].Data)
	}
	if data["status"] != "expired" {
		t.Fatalf("expected expired status payload, got %v", data["status"])
	}
	rec, err := st.GetApproval(context.Background(), approval.ID)
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if rec.Status != "expired" {
		t.Fatalf("expected persisted status expired, got %q", rec.Status)
	}
}

func TestServerApproveExpiredReturnsInternalErrorWhenExpirePersistenceFails(t *testing.T) {
	base, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-expire-fail.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := base.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	approvalID := "apr_expire_persist_fail"
	shim := &forcedApprovalStore{
		Store:       base,
		forcedGetID: approvalID,
		forcedGetRecord: &store.ApprovalRecord{
			ID:        approvalID,
			TenantID:  "tenant-a",
			RequestID: "req_expire_persist_fail",
			AgentName: "agent-a",
			Service:   "github",
			Action:    "issues.create",
			Status:    "pending",
			CreatedAt: time.Now().UTC().Add(-20 * time.Minute),
			ExpiresAt: time.Now().UTC().Add(-10 * time.Minute),
		},
		forcedExpireID:  approvalID,
		forcedExpireErr: errors.New("db unavailable"),
	}

	ts, rawBootstrap := newTestAPIServerFromStore(t, shim)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approvalID+":approve", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 500, got %d body=%s", resp.StatusCode, string(b))
	}
}

func TestServerListPendingApprovalsExpiresStaleBeforeResponse(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)

	stale := &store.ApprovalRecord{
		ID:        "apr_pending_stale",
		TenantID:  "tenant-a",
		RequestID: "req_pending_stale",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		CreatedAt: time.Now().UTC().Add(-20 * time.Minute),
		ExpiresAt: time.Now().UTC().Add(-10 * time.Minute),
	}
	active := &store.ApprovalRecord{
		ID:        "apr_pending_active",
		TenantID:  "tenant-a",
		RequestID: "req_pending_active",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
	if err := st.CreateApproval(context.Background(), stale); err != nil {
		t.Fatalf("create stale approval: %v", err)
	}
	if err := st.CreateApproval(context.Background(), active); err != nil {
		t.Fatalf("create active approval: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/approvals?status=pending", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list approvals request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	data, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", payload["data"])
	}
	if len(data) != 1 {
		t.Fatalf("expected exactly 1 pending approval, got %d", len(data))
	}
	item, ok := data[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first item object, got %T", data[0])
	}
	approvalID, _ := item["ID"].(string)
	if approvalID == "" {
		approvalID, _ = item["id"].(string)
	}
	if approvalID == "" {
		approvalID, _ = item["approval_id"].(string)
	}
	if approvalID != active.ID {
		t.Fatalf("expected active approval id %q, got %q", active.ID, approvalID)
	}

	staleRecord, err := st.GetApproval(context.Background(), stale.ID)
	if err != nil {
		t.Fatalf("get stale approval: %v", err)
	}
	if staleRecord.Status != "expired" {
		t.Fatalf("expected stale approval to be expired, got %q", staleRecord.Status)
	}
}

func TestServerApproveRequiresMultipleVotesWhenConfigured(t *testing.T) {
	ts, rawBootstrap, stAny := newTestAPIServerWithStore(t)
	st, ok := stAny.(*store.SQLStore)
	if !ok {
		t.Fatalf("expected *store.SQLStore, got %T", stAny)
	}

	approvalID := "apr_multi_vote"
	if err := st.CreateApproval(context.Background(), &store.ApprovalRecord{
		ID:                approvalID,
		TenantID:          "tenant-a",
		RequestID:         "req_multi_vote",
		AgentName:         "agent-a",
		Service:           "github",
		Action:            "issues.create",
		Status:            "pending",
		InputJSON:         `{}`,
		RequiredApprovals: 2,
		VotesJSON:         `[]`,
		CreatedAt:         time.Now().UTC(),
		ExpiresAt:         time.Now().UTC().Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	rawApprover2 := "ktk_second_approver_for_tests"
	if err := st.CreateToken(context.Background(), newBootstrapTokenRecord("tenant-a", "approver-2", rawApprover2)); err != nil {
		t.Fatalf("seed second approver token: %v", err)
	}

	firstReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approvalID+":approve", nil)
	firstReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	firstResp, err := http.DefaultClient.Do(firstReq)
	if err != nil {
		t.Fatalf("first approve request: %v", err)
	}
	defer firstResp.Body.Close()
	if firstResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(firstResp.Body)
		t.Fatalf("expected first approve status 200, got %d body=%s", firstResp.StatusCode, string(b))
	}
	var firstPayload map[string]any
	if err := json.NewDecoder(firstResp.Body).Decode(&firstPayload); err != nil {
		t.Fatalf("decode first payload: %v", err)
	}
	firstData, _ := firstPayload["data"].(map[string]any)
	if approved, _ := firstData["approved"].(bool); approved {
		t.Fatalf("expected first vote not to fully approve, payload=%+v", firstData)
	}

	intermediate, err := st.GetApproval(context.Background(), approvalID)
	if err != nil {
		t.Fatalf("get intermediate approval: %v", err)
	}
	if intermediate.Status != "pending" {
		t.Fatalf("expected intermediate status pending, got %q", intermediate.Status)
	}

	secondReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approvalID+":approve", nil)
	secondReq.Header.Set("Authorization", "Bearer "+rawApprover2)
	secondResp, err := http.DefaultClient.Do(secondReq)
	if err != nil {
		t.Fatalf("second approve request: %v", err)
	}
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(secondResp.Body)
		t.Fatalf("expected second approve status 200, got %d body=%s", secondResp.StatusCode, string(b))
	}
	var secondPayload map[string]any
	if err := json.NewDecoder(secondResp.Body).Decode(&secondPayload); err != nil {
		t.Fatalf("decode second payload: %v", err)
	}
	secondData, _ := secondPayload["data"].(map[string]any)
	if approved, _ := secondData["approved"].(bool); !approved {
		t.Fatalf("expected second vote to approve, payload=%+v", secondData)
	}

	finalRecord, err := st.GetApproval(context.Background(), approvalID)
	if err != nil {
		t.Fatalf("get final approval: %v", err)
	}
	if finalRecord.Status != "approved" {
		t.Fatalf("expected final status approved, got %q", finalRecord.Status)
	}
	var votes []map[string]any
	if err := json.Unmarshal([]byte(finalRecord.VotesJSON), &votes); err != nil {
		t.Fatalf("decode votes json: %v", err)
	}
	if len(votes) != 2 {
		t.Fatalf("expected 2 votes persisted, got %d (%s)", len(votes), finalRecord.VotesJSON)
	}
}

func TestServerApprovePropagatesResumedExecutionHTTPStatus(t *testing.T) {
	ts, rawBootstrap, stAny := newTestAPIServerWithStore(t)
	st, ok := stAny.(*store.SQLStore)
	if !ok {
		t.Fatalf("expected *store.SQLStore, got %T", stAny)
	}

	approvalID := "apr_resume_failure"
	if err := st.CreateApproval(context.Background(), &store.ApprovalRecord{
		ID:        approvalID,
		TenantID:  "tenant-a",
		RequestID: "req_resume_failure",
		AgentName: "agent-a",
		Service:   "command-skill",
		Action:    "run",
		Status:    "pending",
		InputJSON: `{}`,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	held := &recordingHeldExecutionStore{held: map[string]actions.ExecutionRequest{
		approvalID: {
			RequestID: "req_resume_failure",
			TraceID:   "tr_resume_failure",
			TenantID:  "tenant-a",
			Principal: actions.Principal{ID: "principal-1", TenantID: "tenant-a", AgentName: "agent-a"},
			Action: actions.ActionDefinition{
				Name: "command-skill.run",
				Adapter: actions.AdapterConfig{
					Type:           "command",
					ExecutablePath: "/bin/failing",
				},
			},
		},
	}}
	server := NewServer(":0", st, WithRuntime(&runtimepkg.Runtime{
		HeldExecutionStore: held,
		Adapters: map[string]adapters.Adapter{
			"command": failingApprovalAdapter{},
		},
	}))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/"+approvalID+":approve", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 502 when resumed execution fails, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["success"] != true {
		t.Fatalf("expected envelope success=true with execution details, got %+v", payload)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", payload["data"])
	}
	execution, ok := data["execution"].(map[string]any)
	if !ok {
		t.Fatalf("expected execution payload, got %+v", data)
	}
	if got := execution["http_status"]; got != float64(http.StatusBadGateway) {
		t.Fatalf("expected execution http_status 502, got %v", got)
	}
	errMap, ok := execution["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected execution error payload, got %+v", execution)
	}
	if errMap["code"] != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected downstream unavailable code, got %v", errMap["code"])
	}
}

func TestServerTokensCreateAndList(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	body := map[string]any{
		"agent_name":  "agent-created",
		"scopes":      []string{"tools:read"},
		"ttl_seconds": 3600,
	}
	b, _ := json.Marshal(body)
	createReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/tokens", bytes.NewReader(b))
	createReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create token request: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.StatusCode)
	}

	listReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/tokens", nil)
	listReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("list token request: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.StatusCode)
	}
}

func TestServerCreateTokenRejectsMismatchedTenantID(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	body := map[string]any{
		"tenant_id":   "tenant-b",
		"agent_name":  "agent-created",
		"scopes":      []string{"tools:read"},
		"ttl_seconds": 3600,
	}
	b, _ := json.Marshal(body)
	createReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/tokens", bytes.NewReader(b))
	createReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create token request: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", createResp.StatusCode)
	}
}

func TestServerCreateTokenRejectsNegativeTTL(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	body := map[string]any{
		"agent_name":  "agent-created",
		"ttl_seconds": -1,
	}
	b, _ := json.Marshal(body)
	createReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/tokens", bytes.NewReader(b))
	createReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create token request: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected 400, got %d body=%s", createResp.StatusCode, string(b))
	}
}

func TestServerCreateTokenRejectsOverflowTTL(t *testing.T) {
	ts, rawBootstrap := newTestAPIServer(t)

	body := map[string]any{
		"agent_name":  "agent-created",
		"ttl_seconds": int64(31536001),
	}
	b, _ := json.Marshal(body)
	createReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/tokens", bytes.NewReader(b))
	createReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create token request: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected 400, got %d body=%s", createResp.StatusCode, string(b))
	}
}

func TestEffectiveTenantIDFromEnvironmentFallback(t *testing.T) {
	t.Setenv("KIMBAP_TENANT_ID", "")
	t.Setenv("KIMBAP_API_DEFAULT_TENANT_ID", "")

	if got := effectiveTenantID(&auth.Principal{TenantID: "tenant-a"}); got != "tenant-a" {
		t.Fatalf("expected principal tenant, got %q", got)
	}

	t.Setenv("KIMBAP_API_DEFAULT_TENANT_ID", "api-default")
	if got := effectiveTenantID(&auth.Principal{}); got != "api-default" {
		t.Fatalf("expected api-default tenant, got %q", got)
	}

	t.Setenv("KIMBAP_API_DEFAULT_TENANT_ID", "")
	t.Setenv("KIMBAP_TENANT_ID", "global-default")
	if got := effectiveTenantID(&auth.Principal{}); got != "global-default" {
		t.Fatalf("expected global-default tenant, got %q", got)
	}

	t.Setenv("KIMBAP_TENANT_ID", "")
	if got := effectiveTenantID(&auth.Principal{}); got != "" {
		t.Fatalf("expected empty tenant without explicit fallback env, got %q", got)
	}
}

func TestTenantContextUsesFallbackTenantForEmptyPrincipalTenant(t *testing.T) {
	t.Setenv("KIMBAP_API_DEFAULT_TENANT_ID", "tenant-fallback")

	h := TenantContext()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tenant := tenantFromContext(r.Context()); tenant != "tenant-fallback" {
			t.Fatalf("expected tenant-fallback, got %q", tenant)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), contextKeyPrincipal, &auth.Principal{ID: "agent-1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestTenantContextRejectsServicePrincipalWithoutTenant(t *testing.T) {
	t.Setenv("KIMBAP_API_DEFAULT_TENANT_ID", "tenant-fallback")

	h := TenantContext()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), contextKeyPrincipal, &auth.Principal{ID: "svc-1", Type: auth.PrincipalTypeService})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); !strings.Contains(got, `Bearer realm="kimbap"`) {
		t.Fatalf("expected bearer challenge, got %q", got)
	}
}

func TestHandleExecuteActionRejectsPrincipalTenantMismatch(t *testing.T) {
	server := &Server{runtime: &runtimepkg.Runtime{}}
	body := strings.NewReader(`{"input": {}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/actions/github/issues.create:execute", body)
	ctx := context.WithValue(req.Context(), contextKeyPrincipal, &auth.Principal{
		ID:        "svc-1",
		Type:      auth.PrincipalTypeService,
		TenantID:  "tenant-a",
		AgentName: "agent-a",
	})
	ctx = context.WithValue(ctx, contextKeyTenant, "tenant-b")
	ctx = context.WithValue(ctx, contextKeyRequestID, "req-1")
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("service", "github")
	rctx.URLParams.Add("action", "issues.create")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	server.handleExecuteAction(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestHandleExecuteActionRequiresExplicitIdempotencyKey(t *testing.T) {
	server := &Server{runtime: &runtimepkg.Runtime{}}
	body := strings.NewReader(`{"input": {"name":"item"}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/actions/github/issues.create:execute", body)
	req = req.WithContext(context.WithValue(req.Context(), contextKeyPrincipal, &auth.Principal{
		ID:        "agent-1",
		TenantID:  "tenant-a",
		AgentName: "agent-a",
		Scopes:    []string{"*"},
	}))
	req = req.WithContext(context.WithValue(req.Context(), contextKeyTenant, "tenant-a"))
	req = req.WithContext(context.WithValue(req.Context(), contextKeyRequestID, "req-idempotency-required"))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("service", "github")
	rctx.URLParams.Add("action", "issues.create")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	server.handleExecuteAction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", payload["error"])
	}
	if errBody["code"] != "ERR_IDEMPOTENCY_REQUIRED" {
		t.Fatalf("expected ERR_IDEMPOTENCY_REQUIRED, got %v", errBody["code"])
	}
}

func TestHandleExecuteActionUsesServeModeForPolicyEvaluation(t *testing.T) {
	policyCapture := &captureExecuteModePolicyEvaluator{}
	server := &Server{runtime: &runtimepkg.Runtime{PolicyEvaluator: policyCapture}}

	body := strings.NewReader(`{"input": {"name":"item"}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/actions/github/issues.create:execute", body)
	req.Header.Set("Idempotency-Key", "idem-serve-mode")
	req = req.WithContext(context.WithValue(req.Context(), contextKeyPrincipal, &auth.Principal{
		ID:        "agent-1",
		TenantID:  "tenant-a",
		AgentName: "agent-a",
		Scopes:    []string{"*"},
	}))
	req = req.WithContext(context.WithValue(req.Context(), contextKeyTenant, "tenant-a"))
	req = req.WithContext(context.WithValue(req.Context(), contextKeyRequestID, "req-serve-mode"))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("service", "github")
	rctx.URLParams.Add("action", "issues.create")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	server.handleExecuteAction(rr, req)

	if !policyCapture.called {
		t.Fatal("expected policy evaluator to be called")
	}
	if policyCapture.lastMode != actions.ModeServe {
		t.Fatalf("expected execution mode %q, got %q", actions.ModeServe, policyCapture.lastMode)
	}
}

func TestHandleListTokensRequiresTenantContext(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/v1/tokens", nil)
	rr := httptest.NewRecorder()

	server.handleListTokens(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); !strings.Contains(got, `Bearer realm="kimbap"`) {
		t.Fatalf("expected bearer challenge, got %q", got)
	}
}

func TestTenantScopedReadHandlersRequireTenantContext(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		method  string
		url     string
	}{
		{name: "get policy", handler: server.handleGetPolicy, method: http.MethodGet, url: "/v1/policies"},
		{name: "list approvals", handler: server.handleListApprovals, method: http.MethodGet, url: "/v1/approvals"},
		{name: "query audit", handler: server.handleQueryAudit, method: http.MethodGet, url: "/v1/audit"},
		{name: "export audit", handler: server.handleExportAudit, method: http.MethodGet, url: "/v1/audit/export"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			rr := httptest.NewRecorder()
			tc.handler(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rr.Code)
			}
			if got := rr.Header().Get("WWW-Authenticate"); !strings.Contains(got, `Bearer realm="kimbap"`) {
				t.Fatalf("expected bearer challenge, got %q", got)
			}
		})
	}
}

func TestHandleListWebhooksRequiresTenantContext(t *testing.T) {
	server := &Server{webhookDispatcher: webhooks.NewDispatcher()}
	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks", nil)
	rr := httptest.NewRecorder()

	server.handleListWebhooks(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); !strings.Contains(got, `Bearer realm="kimbap"`) {
		t.Fatalf("expected bearer challenge, got %q", got)
	}
}

func TestWebhookHandlersRequireTenantContext(t *testing.T) {
	server := &Server{webhookDispatcher: webhooks.NewDispatcher()}

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		method  string
		url     string
		body    io.Reader
	}{
		{name: "create webhook", handler: server.handleCreateWebhook, method: http.MethodPost, url: "/v1/webhooks", body: strings.NewReader(`{"url":"https://example.com/hook"}`)},
		{name: "delete webhook", handler: server.handleDeleteWebhook, method: http.MethodDelete, url: "/v1/webhooks/wh_1", body: nil},
		{name: "list webhook events", handler: server.handleListRecentEvents, method: http.MethodGet, url: "/v1/webhooks/events", body: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, tc.body)
			if strings.Contains(tc.url, "/v1/webhooks/") && tc.method == http.MethodDelete {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", "wh_1")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}
			rr := httptest.NewRecorder()
			tc.handler(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rr.Code)
			}
			if got := rr.Header().Get("WWW-Authenticate"); !strings.Contains(got, `Bearer realm="kimbap"`) {
				t.Fatalf("expected bearer challenge, got %q", got)
			}
		})
	}
}

func TestCreateWebhookRejectsTrailingJSONPayload(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)
	server := NewServer(":0", st, WithWebhookDispatcher(webhooks.NewDispatcher()))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	body := `{"url":"https://example.com/hook"}{"extra":1}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/webhooks", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, string(b))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error envelope, got %v", payload["error"])
	}
	if errBody["message"] != "unexpected trailing content after JSON body" {
		t.Fatalf("expected trailing JSON error, got %v", errBody["message"])
	}
}

func TestCreateWebhookRejectsUnknownEventType(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)
	server := NewServer(":0", st, WithWebhookDispatcher(webhooks.NewDispatcher()))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	body := `{"url":"https://example.com/hook","events":["approval.requested","unknown.event"]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/webhooks", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, string(b))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	errBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error envelope, got %v", payload["error"])
	}
	if errBody["message"] != "events contains unknown or inactive event type" {
		t.Fatalf("unexpected error message: %v", errBody["message"])
	}
	details, ok := errBody["details"].(map[string]any)
	if !ok {
		t.Fatalf("expected error details, got %T", errBody["details"])
	}
	if details["event"] != "unknown.event" {
		t.Fatalf("expected unknown.event details, got %v", details["event"])
	}
}

func TestCreateWebhookRejectsInactiveReservedEventType(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)
	server := NewServer(":0", st, WithWebhookDispatcher(webhooks.NewDispatcher()))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	body := `{"url":"https://example.com/hook","events":["connector.unknown"]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/webhooks", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, string(b))
	}
}

func TestCreateWebhookRejectsHTTPURL(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)
	server := NewServer(":0", st, WithWebhookDispatcher(webhooks.NewDispatcher()))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	body := `{"url":"http://example.com/hook"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/webhooks", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, string(b))
	}
}

func TestServerListPendingApprovalsSweepRemovesHeldAndEmitsExpiredEvent(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)
	dispatcher := webhooks.NewDispatcher()
	held := &recordingHeldExecutionStore{held: map[string]actions.ExecutionRequest{"apr_pending_stale_cleanup": {RequestID: "req_pending_stale_cleanup"}}}
	server := NewServer(":0", st, WithRuntime(&runtimepkg.Runtime{HeldExecutionStore: held}), WithWebhookDispatcher(dispatcher))
	ts.Close()
	ts = httptest.NewServer(server.Router())
	t.Cleanup(func() { ts.Close() })

	stale := &store.ApprovalRecord{
		ID:        "apr_pending_stale_cleanup",
		TenantID:  "tenant-a",
		RequestID: "req_pending_stale_cleanup",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		CreatedAt: time.Now().UTC().Add(-20 * time.Minute),
		ExpiresAt: time.Now().UTC().Add(-10 * time.Minute),
	}
	if err := st.CreateApproval(context.Background(), stale); err != nil {
		t.Fatalf("create stale approval: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/approvals?status=pending", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list approvals request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}
	if held.removeCalls == 0 {
		t.Fatal("expected stale held execution to be removed during expiry sweep")
	}
	if _, ok := held.held[stale.ID]; ok {
		t.Fatal("expected held execution entry to be removed")
	}
	events := dispatcher.RecentEventsByTenant("tenant-a", 10)
	if len(events) == 0 {
		t.Fatal("expected approval expired webhook event")
	}
	if events[len(events)-1].Type != webhooks.EventApprovalExpired {
		t.Fatalf("expected last event %q, got %q", webhooks.EventApprovalExpired, events[len(events)-1].Type)
	}
}

func TestHandleListRecentEventsLimitValidation(t *testing.T) {
	dispatcher := webhooks.NewDispatcher()
	server := &Server{webhookDispatcher: dispatcher}

	reqWithTenant := func(url string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		return req.WithContext(context.WithValue(req.Context(), contextKeyTenant, "tenant-a"))
	}

	// invalid (non-numeric) limit → 400
	rr := httptest.NewRecorder()
	server.handleListRecentEvents(rr, reqWithTenant("/v1/webhooks/events?limit=abc"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("non-numeric limit: expected 400, got %d", rr.Code)
	}

	// zero limit → 400
	rr = httptest.NewRecorder()
	server.handleListRecentEvents(rr, reqWithTenant("/v1/webhooks/events?limit=0"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("zero limit: expected 400, got %d", rr.Code)
	}

	// negative limit → 400
	rr = httptest.NewRecorder()
	server.handleListRecentEvents(rr, reqWithTenant("/v1/webhooks/events?limit=-5"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("negative limit: expected 400, got %d", rr.Code)
	}

	// oversized limit → 200 (capped, not rejected)
	rr = httptest.NewRecorder()
	server.handleListRecentEvents(rr, reqWithTenant("/v1/webhooks/events?limit=99999"))
	if rr.Code != http.StatusOK {
		t.Fatalf("oversized limit: expected 200, got %d", rr.Code)
	}

	// valid limit → 200
	rr = httptest.NewRecorder()
	server.handleListRecentEvents(rr, reqWithTenant("/v1/webhooks/events?limit=10"))
	if rr.Code != http.StatusOK {
		t.Fatalf("valid limit: expected 200, got %d", rr.Code)
	}

	// no limit param → 200 (defaults to 50)
	rr = httptest.NewRecorder()
	server.handleListRecentEvents(rr, reqWithTenant("/v1/webhooks/events"))
	if rr.Code != http.StatusOK {
		t.Fatalf("no limit param: expected 200, got %d", rr.Code)
	}
}

func TestHandleListRecentEventsSkipsMalformedPersistedPayload(t *testing.T) {
	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-webhook-events-malformed.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	rawBootstrap := "ktk_webhook_events_bootstrap"
	if err := st.CreateToken(context.Background(), newBootstrapTokenRecord("tenant-a", "bootstrap-agent", rawBootstrap)); err != nil {
		t.Fatalf("seed bootstrap token: %v", err)
	}

	if err := st.WriteWebhookEvent(context.Background(), &store.WebhookEventRecord{
		ID:        "evt_good",
		TenantID:  "tenant-a",
		Type:      "approval.expired",
		Timestamp: time.Now().UTC().Add(-time.Second),
		DataJSON:  `{"approval_id":"apr_good"}`,
	}); err != nil {
		t.Fatalf("write good webhook event: %v", err)
	}
	if err := st.WriteWebhookEvent(context.Background(), &store.WebhookEventRecord{
		ID:        "evt_bad",
		TenantID:  "tenant-a",
		Type:      "approval.expired",
		Timestamp: time.Now().UTC(),
		DataJSON:  `{`,
	}); err != nil {
		t.Fatalf("write malformed webhook event: %v", err)
	}

	server := NewServer(":0", st, WithWebhookDispatcher(webhooks.NewDispatcher()))
	ts := httptest.NewServer(server.Router())
	t.Cleanup(ts.Close)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/webhooks/events", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list webhook events request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	events, _ := data["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("expected only 1 event after skipping malformed persisted payload, got %d", len(events))
	}
	first, _ := events[0].(map[string]any)
	if id, _ := first["id"].(string); id != "evt_good" {
		t.Fatalf("expected event id evt_good, got %q", id)
	}
}

func TestServerInspectAndRevokeTokenHideCrossTenantExistence(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)

	foreignRaw := "ktk_foreign_token_for_tests"
	foreign := newBootstrapTokenRecord("tenant-b", "foreign-agent", foreignRaw)
	if err := st.CreateToken(context.Background(), foreign); err != nil {
		t.Fatalf("seed foreign token: %v", err)
	}

	inspectReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/tokens/"+foreign.ID, nil)
	inspectReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	inspectResp, err := http.DefaultClient.Do(inspectReq)
	if err != nil {
		t.Fatalf("inspect request: %v", err)
	}
	defer inspectResp.Body.Close()
	if inspectResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected inspect 404, got %d", inspectResp.StatusCode)
	}

	revokeReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/v1/tokens/"+foreign.ID, nil)
	revokeReq.Header.Set("Authorization", "Bearer "+rawBootstrap)
	revokeResp, err := http.DefaultClient.Do(revokeReq)
	if err != nil {
		t.Fatalf("revoke request: %v", err)
	}
	defer revokeResp.Body.Close()
	if revokeResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected revoke 404, got %d", revokeResp.StatusCode)
	}
}

func TestServerUnauthenticatedRequestReturns401(t *testing.T) {
	ts, _ := newTestAPIServer(t)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/tokens", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unauth request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, `Bearer realm="kimbap"`) {
		t.Fatalf("expected WWW-Authenticate bearer challenge, got %q", got)
	}
	if got := resp.Header.Get("WWW-Authenticate"); strings.Contains(got, `error="invalid_request"`) {
		t.Fatalf("expected missing-credential challenge without invalid_request, got %q", got)
	}
}

func TestServerMalformedAuthorizationReturns400WithInvalidRequestChallenge(t *testing.T) {
	ts, _ := newTestAPIServer(t)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/tokens", nil)
	req.Header.Set("Authorization", "Bearer")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("malformed auth request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, string(b))
	}
	challenge := resp.Header.Get("WWW-Authenticate")
	if !strings.Contains(challenge, `error="invalid_request"`) {
		t.Fatalf("expected invalid_request challenge, got %q", challenge)
	}
}

func TestServerInvalidTokenReturns401WithBearerChallenge(t *testing.T) {
	ts, _ := newTestAPIServer(t)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/tokens", nil)
	req.Header.Set("Authorization", "Bearer ktk_invalid_token_value")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("invalid token request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	challenge := resp.Header.Get("WWW-Authenticate")
	if !strings.Contains(challenge, `error="invalid_token"`) {
		t.Fatalf("expected invalid_token challenge, got %q", challenge)
	}
}

func TestServerInsufficientScopeIncludesBearerScopeHint(t *testing.T) {
	ts, _, st := newTestAPIServerWithStore(t)

	rawLimited := "ktk_limited_scope_token_for_tests"
	limited := newBootstrapTokenRecord("tenant-a", "limited-agent", rawLimited)
	limited.Scopes = `["tools:read"]`
	if err := st.CreateToken(context.Background(), limited); err != nil {
		t.Fatalf("seed limited token: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/tokens", nil)
	req.Header.Set("Authorization", "Bearer "+rawLimited)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("insufficient scope request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	challenge := resp.Header.Get("WWW-Authenticate")
	if !strings.Contains(challenge, `error="insufficient_scope"`) {
		t.Fatalf("expected insufficient_scope challenge, got %q", challenge)
	}
	if !strings.Contains(challenge, `scope="tokens:read"`) {
		t.Fatalf("expected required scope hint, got %q", challenge)
	}
}

func TestServerExecuteActionRequiresExecuteScope(t *testing.T) {
	ts, _, st := newTestAPIServerWithStore(t)

	rawLimited := "ktk_execute_scope_limited_token"
	limited := newBootstrapTokenRecord("tenant-a", "limited-execute-agent", rawLimited)
	limited.Scopes = `["tools:read"]`
	if err := st.CreateToken(context.Background(), limited); err != nil {
		t.Fatalf("seed limited token: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/actions/github/issues.create:execute", strings.NewReader(`{"input":{}}`))
	req.Header.Set("Authorization", "Bearer "+rawLimited)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 403, got %d body=%s", resp.StatusCode, string(b))
	}
	challenge := resp.Header.Get("WWW-Authenticate")
	if !strings.Contains(challenge, `error="insufficient_scope"`) {
		t.Fatalf("expected insufficient_scope challenge, got %q", challenge)
	}
	if !strings.Contains(challenge, `scope="actions:execute"`) {
		t.Fatalf("expected actions:execute scope hint, got %q", challenge)
	}
}

func TestProtectedRoutesRequireBearerAuth(t *testing.T) {
	ts, _ := newTestAPIServer(t)

	tests := []struct {
		name   string
		method string
		path   string
		body   io.Reader
	}{
		{name: "execute action", method: http.MethodPost, path: "/v1/actions/github/issues.create:execute", body: strings.NewReader(`{"input":{}}`)},
		{name: "validate action", method: http.MethodPost, path: "/v1/actions/validate", body: strings.NewReader(`{"schema":{"type":"object"},"input":{}}`)},
		{name: "list vault", method: http.MethodGet, path: "/v1/vault"},
		{name: "create token", method: http.MethodPost, path: "/v1/tokens", body: strings.NewReader(`{"agent_name":"a"}`)},
		{name: "list tokens", method: http.MethodGet, path: "/v1/tokens"},
		{name: "inspect token", method: http.MethodGet, path: "/v1/tokens/st_missing"},
		{name: "revoke token", method: http.MethodDelete, path: "/v1/tokens/st_missing"},
		{name: "get policy", method: http.MethodGet, path: "/v1/policies"},
		{name: "set policy", method: http.MethodPut, path: "/v1/policies", body: strings.NewReader(`{"document":"allow"}`)},
		{name: "evaluate policy", method: http.MethodPost, path: "/v1/policies:evaluate", body: strings.NewReader(`{"agent_name":"a"}`)},
		{name: "list approvals", method: http.MethodGet, path: "/v1/approvals"},
		{name: "approve", method: http.MethodPost, path: "/v1/approvals/apr_test:approve"},
		{name: "deny", method: http.MethodPost, path: "/v1/approvals/apr_test:deny"},
		{name: "query audit", method: http.MethodGet, path: "/v1/audit"},
		{name: "export audit", method: http.MethodGet, path: "/v1/audit/export"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, ts.URL+tc.path, tc.body)
			if tc.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				b, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 401, got %d body=%s", resp.StatusCode, string(b))
			}
			challenge := resp.Header.Get("WWW-Authenticate")
			if !strings.Contains(challenge, `Bearer realm="kimbap"`) {
				t.Fatalf("expected bearer challenge, got %q", challenge)
			}
			if strings.Contains(challenge, `error="invalid_request"`) {
				t.Fatalf("expected missing-credential challenge without invalid_request, got %q", challenge)
			}
		})
	}
}

func TestHandleListVaultKeysReturnsMetadataItems(t *testing.T) {
	now := time.Now().UTC()
	vs := &stubVaultStore{
		items: []vault.SecretRecord{{
			ID:             "sec_1",
			TenantID:       "tenant-a",
			Name:           "github.token",
			Type:           vault.SecretTypeBearerToken,
			VersionCount:   1,
			CurrentVersion: 1,
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
	}
	server := &Server{vaultStore: vs}
	req := httptest.NewRequest(http.MethodGet, "/v1/vault?limit=5&offset=1&type=bearer_token", nil)
	req = req.WithContext(context.WithValue(req.Context(), contextKeyTenant, "tenant-a"))
	rr := httptest.NewRecorder()

	server.handleListVaultKeys(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if vs.lastTenantID != "tenant-a" {
		t.Fatalf("expected tenant-a list call, got %q", vs.lastTenantID)
	}
	if vs.lastOpts.Limit != 5 || vs.lastOpts.Offset != 1 {
		t.Fatalf("unexpected pagination options: %+v", vs.lastOpts)
	}
	if vs.lastOpts.Type == nil || *vs.lastOpts.Type != vault.SecretTypeBearerToken {
		t.Fatalf("expected type filter bearer_token, got %+v", vs.lastOpts.Type)
	}

	var payload map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected object data payload, got %T", payload["data"])
	}
	items, ok := data["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got %T", data["items"])
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestServerTokenNotFoundParityMissingVsCrossTenant(t *testing.T) {
	ts, rawBootstrap, st := newTestAPIServerWithStore(t)

	foreignRaw := "ktk_foreign_token_for_parity_tests"
	foreign := newBootstrapTokenRecord("tenant-b", "foreign-agent", foreignRaw)
	if err := st.CreateToken(context.Background(), foreign); err != nil {
		t.Fatalf("seed foreign token: %v", err)
	}

	request := func(path string) (int, map[string]any) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		req.Header.Set("Authorization", "Bearer "+rawBootstrap)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		return resp.StatusCode, payload
	}

	missingStatus, missingPayload := request("/v1/tokens/st_missing_token")
	foreignStatus, foreignPayload := request("/v1/tokens/" + foreign.ID)

	if missingStatus != http.StatusNotFound || foreignStatus != http.StatusNotFound {
		t.Fatalf("expected both statuses 404, got missing=%d foreign=%d", missingStatus, foreignStatus)
	}

	missingError, _ := missingPayload["error"].(map[string]any)
	foreignError, _ := foreignPayload["error"].(map[string]any)
	if missingError["code"] != foreignError["code"] {
		t.Fatalf("expected same error code, got missing=%v foreign=%v", missingError["code"], foreignError["code"])
	}
	if missingError["message"] != foreignError["message"] {
		t.Fatalf("expected same error message, got missing=%v foreign=%v", missingError["message"], foreignError["message"])
	}
}

func TestServerGetPolicyNonNotFoundReturnsDownstreamUnavailable(t *testing.T) {
	baseStore, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-policy-get-error.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := baseStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	wrapped := &forcedPolicyStore{Store: baseStore, forcedTenantID: "tenant-a", forcedGetErr: errors.New("db unavailable")}

	ts, rawBootstrap := newTestAPIServerFromStore(t, wrapped)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/policies", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get policy request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 500, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	errBody, _ := payload["error"].(map[string]any)
	if errBody["code"] != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected %s, got %v", actions.ErrDownstreamUnavailable, errBody["code"])
	}
}

func TestServerEvalPolicyNonNotFoundReturnsDownstreamUnavailable(t *testing.T) {
	baseStore, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-policy-eval-error.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := baseStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	wrapped := &forcedPolicyStore{Store: baseStore, forcedTenantID: "tenant-a", forcedGetErr: errors.New("db unavailable")}

	ts, rawBootstrap := newTestAPIServerFromStore(t, wrapped)
	body := strings.NewReader(`{"agent_name":"agent-a","service":"github","action":"issues.create","risk":"medium","mutating":true,"args":{}}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/policies:evaluate", body)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("evaluate policy request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 500, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	errBody, _ := payload["error"].(map[string]any)
	if errBody["code"] != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected %s, got %v", actions.ErrDownstreamUnavailable, errBody["code"])
	}
}

func TestServerExportAuditStreamsOnSuccess(t *testing.T) {
	baseStore, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-audit-export-stream.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := baseStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	wrapped := &forcedAuditExportStore{Store: baseStore, exportFn: func(w io.Writer) error {
		_, err := io.WriteString(w, `{"request_id":"req_stream"}`+"\n")
		return err
	}}

	ts, rawBootstrap := newTestAPIServerFromStore(t, wrapped)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/audit/export?format=jsonl&from=2020-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("export audit request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "application/x-ndjson") {
		t.Fatalf("expected ndjson content type, got %q", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if !strings.Contains(string(body), `"request_id":"req_stream"`) {
		t.Fatalf("expected streamed audit payload, got %q", string(body))
	}
}

func TestServerExportAuditFailureReturnsErrorEnvelope(t *testing.T) {
	baseStore, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-audit-export-error.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := baseStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	wrapped := &forcedAuditExportStore{Store: baseStore, forcedExportErr: errors.New("db unavailable")}

	ts, rawBootstrap := newTestAPIServerFromStore(t, wrapped)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/audit/export?format=csv&from=2020-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("export audit request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 500, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	errBody, _ := payload["error"].(map[string]any)
	if errBody["code"] != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected %s, got %v", actions.ErrDownstreamUnavailable, errBody["code"])
	}
}

func TestHandleSetPolicyInvalidatesRuntimePolicyCache(t *testing.T) {
	_, _, st := newTestAPIServerWithStore(t)
	policyEval := &capturePolicyCacheInvalidator{}
	server := NewServer(":0", st, WithRuntime(&runtimepkg.Runtime{PolicyEvaluator: policyEval}))

	body := strings.NewReader(`{"document":"version: \"1.0.0\"\nrules:\n  - id: allow-all\n    priority: 1\n    match:\n      actions: [\"*\"]\n    decision: allow\n"}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/policies", body)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), contextKeyTenant, "tenant-a"))
	rr := httptest.NewRecorder()

	server.handleSetPolicy(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if policyEval.calls != 1 {
		t.Fatalf("expected one invalidation call, got %d", policyEval.calls)
	}
	if policyEval.lastTenant != "tenant-a" {
		t.Fatalf("expected tenant-a invalidation, got %q", policyEval.lastTenant)
	}
}

func TestServerApproveApprovalLookupFailureReturnsDownstreamUnavailable(t *testing.T) {
	baseStore, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-approval-get-error.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := baseStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	wrapped := &forcedApprovalStore{Store: baseStore, forcedGetID: "apr_store_error", forcedGetErr: errors.New("db unavailable")}

	ts, rawBootstrap := newTestAPIServerFromStore(t, wrapped)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/approvals/apr_store_error:approve", nil)
	req.Header.Set("Authorization", "Bearer "+rawBootstrap)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 500, got %d body=%s", resp.StatusCode, string(b))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	errBody, _ := payload["error"].(map[string]any)
	if errBody["code"] != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected %s, got %v", actions.ErrDownstreamUnavailable, errBody["code"])
	}
}

func TestNewHTTPServerTimeoutDefaults(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := newHTTPServer("127.0.0.1:0", h)

	if srv.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Fatalf("expected ReadHeaderTimeout %s, got %s", defaultReadHeaderTimeout, srv.ReadHeaderTimeout)
	}
	if srv.IdleTimeout != defaultIdleTimeout {
		t.Fatalf("expected IdleTimeout %s, got %s", defaultIdleTimeout, srv.IdleTimeout)
	}
}

func newTestAPIServer(t *testing.T) (*httptest.Server, string) {
	ts, rawBootstrap, _ := newTestAPIServerWithStore(t)
	return ts, rawBootstrap
}

type forcedApprovalStore struct {
	store.Store
	forcedGetID     string
	forcedGetErr    error
	forcedGetRecord *store.ApprovalRecord
	forcedUpdateID  string
	forcedUpdateErr error
	forcedExpireID  string
	forcedExpireErr error
}

type staticActionRegistry struct {
	items []actions.ActionDefinition
}

func (s staticActionRegistry) Lookup(_ context.Context, name string) (*actions.ActionDefinition, error) {
	for i := range s.items {
		if s.items[i].Name == name {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, actions.ErrLookupNotFound
}

func (s staticActionRegistry) List(_ context.Context, _ runtimepkg.ListOptions) ([]actions.ActionDefinition, error) {
	return append([]actions.ActionDefinition(nil), s.items...), nil
}

type forcedAuditExportStore struct {
	store.Store
	forcedExportErr error
	exportFn        func(io.Writer) error
}

func (s *forcedAuditExportStore) ExportAuditEvents(_ context.Context, _ store.AuditFilter, _ string, w io.Writer) error {
	if s.forcedExportErr != nil {
		return s.forcedExportErr
	}
	if s.exportFn != nil {
		return s.exportFn(w)
	}
	return nil
}

func (s *forcedApprovalStore) GetApproval(ctx context.Context, id string) (*store.ApprovalRecord, error) {
	if s.forcedGetErr != nil && id == s.forcedGetID {
		return nil, s.forcedGetErr
	}
	if s.forcedGetRecord != nil && id == s.forcedGetID {
		rec := *s.forcedGetRecord
		return &rec, nil
	}
	return s.Store.GetApproval(ctx, id)
}

type forcedPolicyStore struct {
	store.Store
	forcedTenantID string
	forcedGetErr   error
}

func (s *forcedPolicyStore) GetPolicy(ctx context.Context, tenantID string) ([]byte, error) {
	if s.forcedGetErr != nil && tenantID == s.forcedTenantID {
		return nil, s.forcedGetErr
	}
	return s.Store.GetPolicy(ctx, tenantID)
}

func (s *forcedApprovalStore) UpdateApprovalStatus(ctx context.Context, id string, status string, resolvedBy string, reason string) error {
	if s.forcedUpdateErr != nil && id == s.forcedUpdateID {
		return s.forcedUpdateErr
	}
	return s.Store.UpdateApprovalStatus(ctx, id, status, resolvedBy, reason)
}

func (s *forcedApprovalStore) UpdateApproval(ctx context.Context, req *store.ApprovalRecord) error {
	if req != nil && s.forcedUpdateErr != nil && req.ID == s.forcedUpdateID {
		return s.forcedUpdateErr
	}
	updater, ok := s.Store.(interface {
		UpdateApproval(context.Context, *store.ApprovalRecord) error
	})
	if !ok {
		return errors.New("update approval unsupported")
	}
	return updater.UpdateApproval(ctx, req)
}

func (s *forcedApprovalStore) ExpireApproval(ctx context.Context, id string) (bool, error) {
	if s.forcedExpireErr != nil && id == s.forcedExpireID {
		return false, s.forcedExpireErr
	}
	expirer, ok := s.Store.(interface {
		ExpireApproval(context.Context, string) (bool, error)
	})
	if !ok {
		return false, nil
	}
	return expirer.ExpireApproval(ctx, id)
}

func newTestAPIServerFromStore(t *testing.T, st store.Store) (*httptest.Server, string) {
	t.Helper()

	rawBootstrap := "ktk_bootstrap_token_for_tests"
	if err := st.CreateToken(context.Background(), newBootstrapTokenRecord("tenant-a", "bootstrap-agent", rawBootstrap)); err != nil {
		t.Fatalf("seed bootstrap token: %v", err)
	}

	server := NewServer(":0", st)
	ts := httptest.NewServer(server.Router())
	t.Cleanup(func() {
		ts.Close()
		_ = st.Close()
	})

	return ts, rawBootstrap
}

func newTestAPIServerWithStore(t *testing.T) (*httptest.Server, string, store.Store) {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "api.sqlite")
	st, err := store.OpenSQLiteStore(dsn)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	rawBootstrap := "ktk_bootstrap_token_for_tests"
	if err := st.CreateToken(context.Background(), newBootstrapTokenRecord("tenant-a", "bootstrap-agent", rawBootstrap)); err != nil {
		t.Fatalf("seed bootstrap token: %v", err)
	}

	server := NewServer(":0", st)
	ts := httptest.NewServer(server.Router())
	t.Cleanup(func() {
		ts.Close()
		_ = st.Close()
	})

	return ts, rawBootstrap, st
}

func TestNewServerDisablesConsoleByDefault(t *testing.T) {
	server := NewServer(":0", nil)
	ts := httptest.NewServer(server.Router())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/console")
	if err != nil {
		t.Fatalf("request console route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for console route by default, got %d", resp.StatusCode)
	}
}

func TestNewServerServesConsoleWhenEnabled(t *testing.T) {
	server := NewServer(":0", nil, WithConsole())
	ts := httptest.NewServer(server.Router())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/console")
	if err != nil {
		t.Fatalf("request console route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when console route is enabled, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected text/html content type for console route, got %q", got)
	}
}

func TestNewServerServesConsoleDeepLinkWithDotWhenEnabled(t *testing.T) {
	server := NewServer(":0", nil, WithConsole())
	ts := httptest.NewServer(server.Router())
	t.Cleanup(ts.Close)

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
		t.Fatalf("expected 200 for console deep link when enabled, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected text/html content type for console deep link, got %q", got)
	}
}

type staticPolicyEvaluator struct{}

func (staticPolicyEvaluator) Evaluate(context.Context, runtimepkg.PolicyRequest) (*runtimepkg.PolicyDecision, error) {
	return &runtimepkg.PolicyDecision{Decision: "require_approval"}, nil
}

type staticApprovalManager struct {
	requestID string
}

type captureExecuteModePolicyEvaluator struct {
	called   bool
	lastMode actions.ExecutionMode
}

type capturePolicyCacheInvalidator struct {
	calls      int
	lastTenant string
}

func (m *capturePolicyCacheInvalidator) Evaluate(context.Context, runtimepkg.PolicyRequest) (*runtimepkg.PolicyDecision, error) {
	return &runtimepkg.PolicyDecision{Decision: "allow"}, nil
}

func (m *capturePolicyCacheInvalidator) InvalidateTenantPolicyCache(tenantID string) {
	m.calls++
	m.lastTenant = tenantID
}

func (m *captureExecuteModePolicyEvaluator) Evaluate(_ context.Context, req runtimepkg.PolicyRequest) (*runtimepkg.PolicyDecision, error) {
	m.called = true
	m.lastMode = req.Mode
	return &runtimepkg.PolicyDecision{Decision: "allow"}, nil
}

func (m staticApprovalManager) CreateRequest(context.Context, runtimepkg.ApprovalRequest) (*runtimepkg.ApprovalResult, error) {
	return &runtimepkg.ApprovalResult{Approved: false, RequestID: m.requestID}, nil
}

func (m staticApprovalManager) CancelRequest(context.Context, string, string) error {
	return nil
}

type noopHeldExecutionStore struct{}

func (noopHeldExecutionStore) Hold(_ context.Context, _ string, _ actions.ExecutionRequest) error {
	return nil
}
func (noopHeldExecutionStore) Resume(_ context.Context, _ string) (*actions.ExecutionRequest, error) {
	return nil, nil
}
func (noopHeldExecutionStore) Remove(_ context.Context, _ string) error {
	return nil
}

type testHeldExecutionStore struct{}

func (testHeldExecutionStore) Hold(_ context.Context, _ string, _ actions.ExecutionRequest) error {
	return nil
}
func (testHeldExecutionStore) Resume(_ context.Context, _ string) (*actions.ExecutionRequest, error) {
	return nil, nil
}
func (testHeldExecutionStore) Remove(_ context.Context, _ string) error {
	return nil
}

type recordingHeldExecutionStore struct {
	held        map[string]actions.ExecutionRequest
	removeCalls int
}

func (s *recordingHeldExecutionStore) Hold(_ context.Context, approvalRequestID string, req actions.ExecutionRequest) error {
	if s.held == nil {
		s.held = map[string]actions.ExecutionRequest{}
	}
	s.held[approvalRequestID] = req
	return nil
}

func (s *recordingHeldExecutionStore) Resume(_ context.Context, approvalRequestID string) (*actions.ExecutionRequest, error) {
	req, ok := s.held[approvalRequestID]
	if !ok {
		return nil, nil
	}
	delete(s.held, approvalRequestID)
	return &req, nil
}

func (s *recordingHeldExecutionStore) Remove(_ context.Context, approvalRequestID string) error {
	s.removeCalls++
	delete(s.held, approvalRequestID)
	return nil
}

type failingApprovalAdapter struct{}

func (failingApprovalAdapter) Type() string { return "command" }

func (failingApprovalAdapter) Validate(actions.ActionDefinition) error { return nil }

func (failingApprovalAdapter) Execute(context.Context, adapters.AdapterRequest) (*adapters.AdapterResult, error) {
	return &adapters.AdapterResult{HTTPStatus: http.StatusBadGateway}, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "resume execution failed", http.StatusBadGateway, true, nil)
}

type stubVaultStore struct {
	items        []vault.SecretRecord
	err          error
	lastTenantID string
	lastOpts     vault.ListOptions
}

func (s *stubVaultStore) Create(context.Context, string, string, vault.SecretType, []byte, map[string]string, string) (*vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *stubVaultStore) Upsert(context.Context, string, string, vault.SecretType, []byte, map[string]string, string) (*vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *stubVaultStore) GetMeta(context.Context, string, string) (*vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *stubVaultStore) GetValue(context.Context, string, string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (s *stubVaultStore) List(_ context.Context, tenantID string, opts vault.ListOptions) ([]vault.SecretRecord, error) {
	s.lastTenantID = tenantID
	s.lastOpts = opts
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

func (s *stubVaultStore) Delete(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *stubVaultStore) Rotate(context.Context, string, string, []byte, string) (*vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *stubVaultStore) GetVersion(context.Context, string, string, int) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (s *stubVaultStore) MarkUsed(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *stubVaultStore) Exists(context.Context, string, string) (bool, error) {
	return false, errors.New("not implemented")
}

func newBootstrapTokenRecord(tenantID string, agentName string, rawToken string) *store.TokenRecord {
	now := time.Now().UTC()
	sum := sha256.Sum256([]byte(rawToken))
	hintStart := max(len(rawToken)-4, 0)
	return &store.TokenRecord{
		ID:          "st_bootstrap_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
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

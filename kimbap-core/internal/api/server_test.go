package api

import (
	"bytes"
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

	"github.com/dunialabs/kimbap-core/internal/store"
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
}

func TestServerListActions(t *testing.T) {
	ts, _ := newTestAPIServer(t)

	resp, err := http.Get(ts.URL + "/v1/actions")
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
	if _, ok := body["data"].([]any); !ok {
		t.Fatalf("expected data array, got %T", body["data"])
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

func TestServerApproveAlreadyResolvedReturnsConflict(t *testing.T) {
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
	if secondResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected second approve=409, got %d", secondResp.StatusCode)
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
	forcedGetRecord *store.ApprovalRecord
	forcedUpdateID  string
	forcedUpdateErr error
}

func (s *forcedApprovalStore) GetApproval(ctx context.Context, id string) (*store.ApprovalRecord, error) {
	if s.forcedGetRecord != nil && id == s.forcedGetID {
		rec := *s.forcedGetRecord
		return &rec, nil
	}
	return s.Store.GetApproval(ctx, id)
}

func (s *forcedApprovalStore) UpdateApprovalStatus(ctx context.Context, id string, status string, resolvedBy string, reason string) error {
	if s.forcedUpdateErr != nil && id == s.forcedUpdateID {
		return s.forcedUpdateErr
	}
	return s.Store.UpdateApprovalStatus(ctx, id, status, resolvedBy, reason)
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

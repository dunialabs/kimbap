package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

	return ts, rawBootstrap
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

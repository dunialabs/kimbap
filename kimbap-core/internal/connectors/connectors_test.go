package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dunialabs/kimbap-core/internal/security"
)

func TestManagerLoginDeviceFlowAndList(t *testing.T) {
	ctx := context.Background()
	store := newMemConnectorStore()
	manager := NewManager(store)
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", "connector-test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/device":
			_, _ = w.Write([]byte(`{"device_code":"dev-123","verification_uri":"https://verify.example","user_code":"ABCD-123","expires_in":600,"interval":1}`))
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if got := r.Form.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:device_code" {
				t.Fatalf("unexpected grant_type: %s", got)
			}
			if got := r.Form.Get("device_code"); got != "dev-123" {
				t.Fatalf("unexpected device code: %s", got)
			}
			_, _ = w.Write([]byte(`{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600,"token_type":"Bearer","scope":"mail.read profile"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	manager.RegisterConfig(ConnectorConfig{
		Name:      "gmail",
		Provider:  "google",
		ClientID:  "client-id",
		TokenURL:  server.URL + "/token",
		DeviceURL: server.URL + "/device",
		Scopes:    []string{"mail.read"},
	})

	flow, err := manager.Login(ctx, "tenant-1", "gmail")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if flow.VerificationURL != "https://verify.example" {
		t.Fatalf("unexpected verification url: %s", flow.VerificationURL)
	}
	if flow.UserCode != "ABCD-123" {
		t.Fatalf("unexpected user code: %s", flow.UserCode)
	}

	if err := manager.CompleteLogin(ctx, "tenant-1", "gmail", ""); err != nil {
		t.Fatalf("complete login: %v", err)
	}

	states, err := manager.List(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(states))
	}
	if states[0].Name != "gmail" {
		t.Fatalf("unexpected connector name: %s", states[0].Name)
	}
	if states[0].Status != StatusHealthy {
		t.Fatalf("unexpected connector status: %s", states[0].Status)
	}

	stored, err := store.Get(ctx, "tenant-1", "gmail")
	if err != nil {
		t.Fatalf("store get: %v", err)
	}
	if stored == nil || stored.AccessToken == "" {
		t.Fatal("stored connector token is missing")
	}
	if strings.Contains(stored.AccessToken, "access-1") {
		t.Fatal("access token should be encrypted at rest")
	}
}

func TestRefreshAccessTokenMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Fatalf("unexpected grant_type: %s", got)
		}
		if got := r.Form.Get("refresh_token"); got != "refresh-abc" {
			t.Fatalf("unexpected refresh token: %s", got)
		}
		_, _ = w.Write([]byte(`{"access_token":"new-token","refresh_token":"new-refresh","expires_in":7200,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	token, err := RefreshAccessToken(ConnectorConfig{ClientID: "client-id", TokenURL: server.URL}, "refresh-abc")
	if err != nil {
		t.Fatalf("refresh access token: %v", err)
	}
	if token.AccessToken != "new-token" {
		t.Fatalf("unexpected access token: %s", token.AccessToken)
	}
}

func TestGetAccessTokenAutoRefreshExpiredToken(t *testing.T) {
	ctx := context.Background()
	store := newMemConnectorStore()
	manager := NewManager(store)
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", "connector-test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Fatalf("unexpected grant_type: %s", got)
		}
		_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"fresh-refresh","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	manager.RegisterConfig(ConnectorConfig{
		Name:      "slack",
		Provider:  "slack",
		ClientID:  "client-id",
		TokenURL:  server.URL,
		DeviceURL: server.URL + "/device",
	})

	expired := time.Now().Add(-2 * time.Minute)
	if err := store.Save(ctx, &ConnectorState{
		Name:         "slack",
		TenantID:     "tenant-1",
		Provider:     "slack",
		Status:       StatusOldExpired,
		AccessToken:  mustEncryptToken(t, "old-access", "connector-test-key"),
		RefreshToken: mustEncryptToken(t, "old-refresh", "connector-test-key"),
		ExpiresAt:    &expired,
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	token, err := manager.GetAccessToken(ctx, "tenant-1", "slack")
	if err != nil {
		t.Fatalf("get access token: %v", err)
	}
	if token != "fresh-access" {
		t.Fatalf("unexpected token: %s", token)
	}

	updated, err := store.Get(ctx, "tenant-1", "slack")
	if err != nil {
		t.Fatalf("store get: %v", err)
	}
	if updated.LastRefresh == nil {
		t.Fatal("expected last_refresh to be set")
	}
}

func TestListConnectorStatusAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := newMemConnectorStore()
	manager := NewManager(store)
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", "connector-test-key")

	now := time.Now()
	expired := now.Add(-time.Minute)
	expiring := now.Add(2 * time.Minute)
	healthy := now.Add(20 * time.Minute)

	seed := []ConnectorState{
		{Name: "github", TenantID: "tenant-a", Provider: "github", AccessToken: mustEncryptToken(t, "tok-a", "connector-test-key"), ExpiresAt: &healthy, CreatedAt: now, UpdatedAt: now},
		{Name: "gmail", TenantID: "tenant-a", Provider: "google", AccessToken: mustEncryptToken(t, "tok-b", "connector-test-key"), ExpiresAt: &expiring, CreatedAt: now, UpdatedAt: now},
		{Name: "slack", TenantID: "tenant-a", Provider: "slack", AccessToken: mustEncryptToken(t, "tok-c", "connector-test-key"), ExpiresAt: &expired, CreatedAt: now, UpdatedAt: now},
		{Name: "github", TenantID: "tenant-b", Provider: "github", AccessToken: mustEncryptToken(t, "tok-d", "connector-test-key"), ExpiresAt: &healthy, CreatedAt: now, UpdatedAt: now},
	}

	for i := range seed {
		copyState := seed[i]
		if err := store.Save(ctx, &copyState); err != nil {
			t.Fatalf("seed state: %v", err)
		}
	}

	listA, err := manager.List(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("list tenant-a: %v", err)
	}
	if len(listA) != 3 {
		t.Fatalf("expected 3 connectors, got %d", len(listA))
	}
	statusByName := map[string]ConnectorStatus{}
	for _, item := range listA {
		statusByName[item.Name] = item.Status
	}
	if statusByName["github"] != StatusHealthy {
		t.Fatalf("expected github healthy, got %s", statusByName["github"])
	}
	if statusByName["gmail"] != StatusExpiring {
		t.Fatalf("expected gmail expiring, got %s", statusByName["gmail"])
	}
	if statusByName["slack"] != StatusOldExpired {
		t.Fatalf("expected slack expired, got %s", statusByName["slack"])
	}

	listB, err := manager.List(ctx, "tenant-b")
	if err != nil {
		t.Fatalf("list tenant-b: %v", err)
	}
	if len(listB) != 1 || listB[0].Name != "github" {
		t.Fatalf("tenant isolation failed: %+v", listB)
	}
}

type memConnectorStore struct {
	mu    sync.RWMutex
	items map[string]ConnectorState
}

func newMemConnectorStore() *memConnectorStore {
	return &memConnectorStore{items: map[string]ConnectorState{}}
}

func (s *memConnectorStore) Save(_ context.Context, state *ConnectorState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyState := *state
	s.items[s.key(state.TenantID, state.Name)] = copyState
	return nil
}

func (s *memConnectorStore) Get(_ context.Context, tenantID, name string) (*ConnectorState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[s.key(tenantID, name)]
	if !ok {
		return nil, nil
	}
	copyState := item
	return &copyState, nil
}

func (s *memConnectorStore) List(_ context.Context, tenantID string) ([]ConnectorState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ConnectorState, 0)
	for _, item := range s.items {
		if item.TenantID != tenantID {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *memConnectorStore) Delete(_ context.Context, tenantID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, s.key(tenantID, name))
	return nil
}

func (s *memConnectorStore) key(tenantID, name string) string {
	return tenantID + "::" + name
}

func mustEncryptToken(t *testing.T, value, key string) string {
	t.Helper()
	encrypted, err := security.EncryptData(value, key)
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	return encrypted
}

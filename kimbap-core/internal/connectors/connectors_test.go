package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		Name:      "mailbox",
		Provider:  "mailbox",
		ClientID:  "client-id",
		TokenURL:  server.URL + "/token",
		DeviceURL: server.URL + "/device",
		Scopes:    []string{"mail.read"},
	})

	flow, err := manager.Login(ctx, "tenant-1", "mailbox")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if flow.VerificationURL != "https://verify.example" {
		t.Fatalf("unexpected verification url: %s", flow.VerificationURL)
	}
	if flow.UserCode != "ABCD-123" {
		t.Fatalf("unexpected user code: %s", flow.UserCode)
	}

	if err := manager.CompleteLogin(ctx, "tenant-1", "mailbox", ""); err != nil {
		t.Fatalf("complete login: %v", err)
	}

	states, err := manager.List(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(states))
	}
	if states[0].Name != "mailbox" {
		t.Fatalf("unexpected connector name: %s", states[0].Name)
	}
	if states[0].Status != StatusHealthy {
		t.Fatalf("unexpected connector status: %s", states[0].Status)
	}

	stored, err := store.Get(ctx, "tenant-1", "mailbox")
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

func TestGetAccessTokenAutoRefreshExpiredClientCredentialsToken(t *testing.T) {
	ctx := context.Background()
	store := newMemConnectorStore()
	manager := NewManager(store)
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", "connector-test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "client_credentials" {
			t.Fatalf("unexpected grant_type: %s", got)
		}
		if got := r.Form.Get("client_id"); got != "client-id" {
			t.Fatalf("unexpected client_id: %s", got)
		}
		_, _ = w.Write([]byte(`{"access_token":"cc-fresh-access","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	manager.RegisterConfig(ConnectorConfig{
		Name:         "internal-api",
		Provider:     "internal-api",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     server.URL,
	})

	expired := time.Now().Add(-2 * time.Minute)
	if err := store.Save(ctx, &ConnectorState{
		Name:        "internal-api",
		TenantID:    "tenant-1",
		Provider:    "internal-api",
		Status:      StatusOldExpired,
		AccessToken: mustEncryptToken(t, "cc-old-access", "connector-test-key"),
		ExpiresAt:   &expired,
		FlowUsed:    FlowClientCredentials,
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	token, err := manager.GetAccessToken(ctx, "tenant-1", "internal-api")
	if err != nil {
		t.Fatalf("get access token: %v", err)
	}
	if token != "cc-fresh-access" {
		t.Fatalf("unexpected token: %s", token)
	}

	updated, err := store.Get(ctx, "tenant-1", "internal-api")
	if err != nil {
		t.Fatalf("store get: %v", err)
	}
	if updated.LastRefresh == nil {
		t.Fatal("expected last_refresh to be set")
	}
	if updated.LastRefreshError != "" {
		t.Fatalf("expected empty refresh error, got %q", updated.LastRefreshError)
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
		{Name: "service-a", TenantID: "tenant-a", Provider: "service-a", AccessToken: mustEncryptToken(t, "tok-a", "connector-test-key"), ExpiresAt: &healthy, CreatedAt: now, UpdatedAt: now},
		{Name: "service-b", TenantID: "tenant-a", Provider: "service-b", AccessToken: mustEncryptToken(t, "tok-b", "connector-test-key"), ExpiresAt: &expiring, CreatedAt: now, UpdatedAt: now},
		{Name: "slack", TenantID: "tenant-a", Provider: "slack", AccessToken: mustEncryptToken(t, "tok-c", "connector-test-key"), ExpiresAt: &expired, CreatedAt: now, UpdatedAt: now},
		{Name: "service-a", TenantID: "tenant-b", Provider: "service-a", AccessToken: mustEncryptToken(t, "tok-d", "connector-test-key"), ExpiresAt: &healthy, CreatedAt: now, UpdatedAt: now},
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
	if statusByName["service-a"] != StatusHealthy {
		t.Fatalf("expected service-a healthy, got %s", statusByName["service-a"])
	}
	if statusByName["service-b"] != StatusExpiring {
		t.Fatalf("expected service-b expiring, got %s", statusByName["service-b"])
	}
	if statusByName["slack"] != StatusOldExpired {
		t.Fatalf("expected slack expired, got %s", statusByName["slack"])
	}

	listB, err := manager.List(ctx, "tenant-b")
	if err != nil {
		t.Fatalf("list tenant-b: %v", err)
	}
	if len(listB) != 1 || listB[0].Name != "service-a" {
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

func TestRefreshNoTokenRotation_NoDoubleEncryption(t *testing.T) {
	ctx := context.Background()
	store := newMemConnectorStore()
	manager := NewManager(store)
	const encKey = "connector-test-key"
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", encKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("refresh_token"); got != "original-refresh" {
			t.Fatalf("expected original refresh token, got: %s", got)
		}
		_, _ = w.Write([]byte(`{"access_token":"new-access","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	manager.RegisterConfig(ConnectorConfig{
		Name:     "source-control",
		Provider: "source-control",
		ClientID: "client-id",
		TokenURL: server.URL,
	})

	expired := time.Now().Add(-10 * time.Minute)
	if err := store.Save(ctx, &ConnectorState{
		Name:         "source-control",
		TenantID:     "tenant-1",
		Provider:     "source-control",
		Status:       StatusOldExpired,
		AccessToken:  mustEncryptToken(t, "old-access", encKey),
		RefreshToken: mustEncryptToken(t, "original-refresh", encKey),
		ExpiresAt:    &expired,
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	if err := manager.Refresh(ctx, "tenant-1", "source-control"); err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	stored, err := store.Get(ctx, "tenant-1", "source-control")
	if err != nil || stored == nil {
		t.Fatalf("store get after first refresh: %v", err)
	}

	decryptedRefresh, err := security.DecryptDataFromString(stored.RefreshToken, encKey)
	if err != nil {
		t.Fatalf("decrypt refresh token after first refresh: %v", err)
	}
	if decryptedRefresh != "original-refresh" {
		t.Fatalf("refresh token corrupted after first refresh: got %q, want %q", decryptedRefresh, "original-refresh")
	}

	decryptedAccess, err := security.DecryptDataFromString(stored.AccessToken, encKey)
	if err != nil {
		t.Fatalf("decrypt access token after first refresh: %v", err)
	}
	if decryptedAccess != "new-access" {
		t.Fatalf("access token wrong after first refresh: got %q, want %q", decryptedAccess, "new-access")
	}

	if err := manager.Refresh(ctx, "tenant-1", "source-control"); err != nil {
		t.Fatalf("second refresh (proves no double-encryption): %v", err)
	}

	stored2, _ := store.Get(ctx, "tenant-1", "source-control")
	decryptedRefresh2, err := security.DecryptDataFromString(stored2.RefreshToken, encKey)
	if err != nil {
		t.Fatalf("decrypt refresh token after second refresh: %v", err)
	}
	if decryptedRefresh2 != "original-refresh" {
		t.Fatalf("refresh token corrupted after second refresh: got %q, want %q", decryptedRefresh2, "original-refresh")
	}
}

func TestLoginWithExistingState_NoDoubleEncryption(t *testing.T) {
	ctx := context.Background()
	store := newMemConnectorStore()
	manager := NewManager(store)
	const encKey = "connector-test-key"
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", encKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"device_code":"dev-1","verification_uri":"https://v.example","user_code":"CODE","expires_in":600,"interval":1}`))
	}))
	defer server.Close()

	manager.RegisterConfig(ConnectorConfig{
		Name:      "source-control",
		Provider:  "source-control",
		ClientID:  "client-id",
		DeviceURL: server.URL,
		TokenURL:  server.URL + "/token",
	})

	if err := store.Save(ctx, &ConnectorState{
		Name:         "source-control",
		TenantID:     "tenant-1",
		Provider:     "source-control",
		Status:       StatusHealthy,
		AccessToken:  mustEncryptToken(t, "existing-access", encKey),
		RefreshToken: mustEncryptToken(t, "existing-refresh", encKey),
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	if _, err := manager.Login(ctx, "tenant-1", "source-control"); err != nil {
		t.Fatalf("login: %v", err)
	}

	stored, _ := store.Get(ctx, "tenant-1", "source-control")
	if stored == nil {
		t.Fatal("stored state is nil")
	}

	decryptedAccess, err := security.DecryptDataFromString(stored.AccessToken, encKey)
	if err != nil {
		t.Fatalf("decrypt access token after login: %v (double-encrypted?)", err)
	}
	if decryptedAccess != "existing-access" {
		t.Fatalf("access token corrupted: got %q, want %q", decryptedAccess, "existing-access")
	}

	decryptedRefresh, err := security.DecryptDataFromString(stored.RefreshToken, encKey)
	if err != nil {
		t.Fatalf("decrypt refresh token after login: %v (double-encrypted?)", err)
	}
	if decryptedRefresh != "existing-refresh" {
		t.Fatalf("refresh token corrupted: got %q, want %q", decryptedRefresh, "existing-refresh")
	}
}

func TestStoreConnectionOverExisting_NoDoubleEncryption(t *testing.T) {
	ctx := context.Background()
	store := newMemConnectorStore()
	manager := NewManager(store)
	const encKey = "connector-test-key"
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", encKey)

	if err := store.Save(ctx, &ConnectorState{
		Name:         "notion",
		TenantID:     "tenant-1",
		Provider:     "notion",
		Status:       StatusHealthy,
		AccessToken:  mustEncryptToken(t, "old-access", encKey),
		RefreshToken: mustEncryptToken(t, "old-refresh", encKey),
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	if err := manager.StoreConnection(ctx, "tenant-1", "notion", "notion", "new-access", "new-refresh", 3600, "read write", FlowBrowser, ScopeWorkspace, "ws-1"); err != nil {
		t.Fatalf("store connection: %v", err)
	}

	stored, _ := store.Get(ctx, "tenant-1", "notion")
	decryptedAccess, err := security.DecryptDataFromString(stored.AccessToken, encKey)
	if err != nil {
		t.Fatalf("decrypt access: %v", err)
	}
	if decryptedAccess != "new-access" {
		t.Fatalf("access wrong: got %q, want %q", decryptedAccess, "new-access")
	}

	decryptedRefresh, err := security.DecryptDataFromString(stored.RefreshToken, encKey)
	if err != nil {
		t.Fatalf("decrypt refresh: %v", err)
	}
	if decryptedRefresh != "new-refresh" {
		t.Fatalf("refresh wrong: got %q, want %q", decryptedRefresh, "new-refresh")
	}
}

func TestPollForTokenSlowDownIncrementsBy5(t *testing.T) {
	originalWait := pollIntervalWait
	t.Cleanup(func() { pollIntervalWait = originalWait })
	var waited time.Duration
	pollIntervalWait = func(_ context.Context, d time.Duration) error {
		waited = d
		return nil
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"slow_down"}`))
		case 2:
			_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":3600,"token_type":"Bearer"}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	token, err := PollForToken(ConnectorConfig{ClientID: "cid", TokenURL: server.URL}, "dev-code", 1, 30*time.Second)

	if err != nil {
		t.Fatalf("PollForToken: %v", err)
	}
	if token.AccessToken != "tok" {
		t.Fatalf("unexpected token: %s", token.AccessToken)
	}
	if waited != 6*time.Second {
		t.Fatalf("expected wait of 6s after slow_down (+5), got %v", waited)
	}
}

func TestPollForTokenWithContextCancelsDuringWait(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)

	_, err := PollForTokenWithContext(ctx, ConnectorConfig{ClientID: "cid", TokenURL: server.URL}, "dev-code", 5, 30*time.Second)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected canceled error, got %v", err)
	}
}

func mustEncryptToken(t *testing.T, value, key string) string {
	t.Helper()
	encrypted, err := security.EncryptData(value, key)
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	return encrypted
}

func TestPostFormWithAuth_BasicSendsHeaderNotBody(t *testing.T) {
	var capturedAuth string
	var capturedBodySecret string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		capturedBodySecret = r.Form.Get("client_secret")
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer"}`))
	}))
	defer server.Close()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", "test-client")
	_, err := postFormWithAuth(context.Background(), server.URL, form, "basic", "test-client", "test-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBodySecret != "" {
		t.Errorf("client_secret should NOT be in body for basic auth, got: %q", capturedBodySecret)
	}
	if !strings.Contains(capturedAuth, "Basic ") {
		t.Errorf("expected Basic auth header, got: %q", capturedAuth)
	}
}

func TestPostFormWithAuth_BodySendsSecretInForm(t *testing.T) {
	var capturedBodySecret string
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		capturedBodySecret = r.Form.Get("client_secret")
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer"}`))
	}))
	defer server.Close()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", "test-client")
	_, err := postFormWithAuth(context.Background(), server.URL, form, "body", "test-client", "test-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBodySecret != "test-secret" {
		t.Errorf("expected client_secret in body, got: %q", capturedBodySecret)
	}
	if capturedAuth != "" {
		t.Errorf("expected no Authorization header for body auth, got: %q", capturedAuth)
	}
}

func TestPostFormWithAuth_NoSecretPublicClient(t *testing.T) {
	var capturedBodySecret string
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		capturedBodySecret = r.Form.Get("client_secret")
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer"}`))
	}))
	defer server.Close()

	form := url.Values{}
	form.Set("grant_type", "device_code")
	form.Set("client_id", "public-id")
	_, err := postFormWithAuth(context.Background(), server.URL, form, "body", "public-id", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBodySecret != "" {
		t.Errorf("expected no client_secret in body for public client, got: %q", capturedBodySecret)
	}
	if capturedAuth != "" {
		t.Errorf("expected no Authorization header for public client, got: %q", capturedAuth)
	}
}

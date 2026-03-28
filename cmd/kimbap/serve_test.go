package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/api"
	"github.com/dunialabs/kimbap/internal/store"
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

	withDispatcher := api.NewServer(":0", st, buildServeServerOptions(nil, nil, false)...)
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
	server := api.NewServer(":0", nil, buildServeServerOptions(nil, nil, false)...)
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
	server := api.NewServer(":0", nil, buildServeServerOptions(nil, nil, true)...)
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
	server := api.NewServer(":0", nil, buildServeServerOptions(nil, nil, true)...)
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

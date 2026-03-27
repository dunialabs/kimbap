package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dunialabs/kimbap/internal/api"
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

	withDispatcher := api.NewServer(":0", nil, buildServeServerOptions(nil, nil, false)...)
	withTS := httptest.NewServer(withDispatcher.Router())
	defer withTS.Close()

	resp2, err := http.Get(withTS.URL + "/v1/webhooks")
	if err != nil {
		t.Fatalf("request with dispatcher: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode == http.StatusNotFound {
		t.Fatalf("expected webhook route to be registered when dispatcher is configured")
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
	if resp.StatusCode == http.StatusNotFound {
		t.Fatalf("expected console route to be served when enabled")
	}
}

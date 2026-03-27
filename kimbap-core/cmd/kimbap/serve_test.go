//go:build ignore

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dunialabs/kimbap-core/internal/api"
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

	withDispatcher := api.NewServer(":0", nil, buildServeServerOptions(nil, nil)...)
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

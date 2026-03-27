package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesIndexAtRoot(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
		t.Fatalf("expected HTML index content, got: %s", body)
	}
	if !strings.Contains(body, "Kimbap Embedded Console") {
		t.Fatalf("expected embedded console page content")
	}
}

func TestHandlerReturnsNotFoundForMissingAssetFile(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want 404", rr.Code)
	}
}

func TestHandlerFallsBackToIndexForRouteWithoutExtension(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/console/dashboard", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rr.Code)
	}
	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if !strings.Contains(string(body), "Kimbap Embedded Console") {
		t.Fatalf("expected SPA fallback index content")
	}
}

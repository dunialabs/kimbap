package webui

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func readEmbeddedIndexHTML(t *testing.T) string {
	t.Helper()
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		t.Fatalf("fs.Sub(distFS, dist) error: %v", err)
	}
	b, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		t.Fatalf("read embedded index.html: %v", err)
	}
	return string(b)
}

func TestHandlerServesIndexAtRoot(t *testing.T) {
	h := Handler()
	indexHTML := readEmbeddedIndexHTML(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type = %q, want text/html", ct)
	}
	body := rr.Body.String()
	if body != indexHTML {
		t.Fatalf("root response should equal embedded index.html")
	}
}

func TestHandlerReturnsNotFoundForMissingAssetFile(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	req.Header.Set("Accept", "application/javascript")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want 404", rr.Code)
	}
}

func TestHandlerFallsBackToIndexForRouteWithoutExtension(t *testing.T) {
	h := Handler()
	indexHTML := readEmbeddedIndexHTML(t)
	req := httptest.NewRequest(http.MethodGet, "/console/dashboard", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rr.Code)
	}
	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if got := string(body); got != indexHTML {
		t.Fatalf("expected SPA fallback to return embedded index.html")
	}
}

func TestHandlerReturnsNotFoundForMissingAssetWithHTMLAccept(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want 404 for missing asset even with text/html Accept", rr.Code)
	}
}

func TestHandlerFallsBackToIndexForRouteWithDotWhenHTMLNavigation(t *testing.T) {
	h := Handler()
	indexHTML := readEmbeddedIndexHTML(t)
	req := httptest.NewRequest(http.MethodGet, "/console/releases/v1.2", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rr.Code)
	}
	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if got := string(body); got != indexHTML {
		t.Fatalf("expected SPA fallback to return embedded index.html for dotted route")
	}
}

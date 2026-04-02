package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateFromOpenAPIURLRejectsUppercaseHTTPSInsecureHTTP(t *testing.T) {
	_, err := GenerateFromOpenAPIURL(context.Background(), "HTTP://example.com/openapi.yaml")
	if err == nil {
		t.Fatal("expected insecure HTTP URL to be rejected")
	}
	if !strings.Contains(err.Error(), "insecure URL") {
		t.Fatalf("expected insecure URL error, got %v", err)
	}
}

func TestGenerateFromOpenAPIURLAllowsLoopbackHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`openapi: 3.0.0
info:
  title: Loopback API
  version: 1.0.0
servers:
  - url: /api
paths:
  /health:
    get:
      operationId: health
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`))
	}))
	defer server.Close()

	manifest, err := GenerateFromOpenAPIURL(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("expected loopback HTTP URL to be accepted, got %v", err)
	}
	if manifest.BaseURL != server.URL+"/api" {
		t.Fatalf("expected resolved relative baseURL, got %q", manifest.BaseURL)
	}
}

func TestGenerateFromOpenAPIURLRejectsRedirectToRemoteHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/openapi.yaml", http.StatusFound)
	}))
	defer server.Close()

	_, err := GenerateFromOpenAPIURL(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected remote HTTP redirect to be rejected")
	}
	if !strings.Contains(err.Error(), "redirect to insecure URL") {
		t.Fatalf("expected redirect security error, got %v", err)
	}
}

package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallRevocationEndpoint_BasicAuthOmitClientSecretInBody(t *testing.T) {
	var authHeader string
	var body string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		payload, _ := io.ReadAll(r.Body)
		authHeader = r.Header.Get("Authorization")
		body = string(payload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := callRevocationEndpoint(server.URL, "client-id", "client-secret", "tok-1", "basic")
	if err != nil {
		t.Fatalf("callRevocationEndpoint returned error: %v", err)
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Fatalf("expected Basic authorization header, got %q", authHeader)
	}
	if strings.Contains(body, "client_secret=") {
		t.Fatalf("did not expect client_secret in body for basic auth, body=%q", body)
	}
	if strings.Contains(body, "client_id=") {
		t.Fatalf("did not expect client_id in body for basic auth, body=%q", body)
	}
	if !strings.Contains(body, "token=tok-1") {
		t.Fatalf("expected token in body, got %q", body)
	}
}

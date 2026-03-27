package clientcred

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRunClientCredentialsFlow_ValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		cfg        ClientCredentialsConfig
		wantErrSub string
	}{
		{
			name:       "missing token endpoint",
			cfg:        ClientCredentialsConfig{ClientID: "client"},
			wantErrSub: "token endpoint is required",
		},
		{
			name:       "missing client id",
			cfg:        ClientCredentialsConfig{TokenEndpoint: "https://example.com/token"},
			wantErrSub: "client id is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RunClientCredentialsFlow(context.Background(), tc.cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}

func TestRunClientCredentialsFlow_Success_PassesScopesAuthMethodAndMapsResponse(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
			t.Fatalf("expected basic auth header, got %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
			t.Fatalf("grant_type = %q, want client_credentials", got)
		}
		if got := r.PostForm.Get("client_id"); got != "client-id" {
			t.Fatalf("client_id = %q, want client-id", got)
		}
		if got := r.PostForm.Get("scope"); got != "read write" {
			t.Fatalf("scope = %q, want joined scopes", got)
		}
		if got := r.PostForm.Get("client_secret"); got != "" {
			t.Fatalf("client_secret should not be in body for basic auth, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access-123","expires_in":3600,"scope":"read write"}`))
	}))
	defer srv.Close()

	result, err := RunClientCredentialsFlow(context.Background(), ClientCredentialsConfig{
		TokenEndpoint: srv.URL,
		ClientID:      "client-id",
		ClientSecret:  "secret",
		Scopes:        []string{"read", "write"},
		AuthMethod:    "basic",
	})
	if err != nil {
		t.Fatalf("RunClientCredentialsFlow() error = %v", err)
	}
	if result.AccessToken != "access-123" {
		t.Fatalf("access token = %q, want access-123", result.AccessToken)
	}
	if result.ExpiresIn != 3600 {
		t.Fatalf("expires_in = %d, want 3600", result.ExpiresIn)
	}
	if result.Scope != "read write" {
		t.Fatalf("scope = %q, want read write", result.Scope)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("call count = %d, want 1", got)
	}
}

func TestRunClientCredentialsFlow_PropagatesTokenEndpointError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"bad credentials"}`))
	}))
	defer srv.Close()

	_, err := RunClientCredentialsFlow(context.Background(), ClientCredentialsConfig{
		TokenEndpoint: srv.URL,
		ClientID:      "client-id",
		ClientSecret:  "wrong-secret",
	})
	if err == nil {
		t.Fatal("expected token endpoint error")
	}
	if !strings.Contains(err.Error(), "oauth endpoint returned status 401") {
		t.Fatalf("error = %v, want oauth 401 message", err)
	}
}

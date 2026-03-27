package device

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunDeviceFlow_ValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		cfg        DeviceFlowConfig
		wantErrSub string
	}{
		{
			name:       "missing device endpoint",
			cfg:        DeviceFlowConfig{TokenEndpoint: "https://example.com/token", ClientID: "client"},
			wantErrSub: "device endpoint is required",
		},
		{
			name:       "missing token endpoint",
			cfg:        DeviceFlowConfig{DeviceEndpoint: "https://example.com/device", ClientID: "client"},
			wantErrSub: "token endpoint is required",
		},
		{
			name:       "missing client id",
			cfg:        DeviceFlowConfig{DeviceEndpoint: "https://example.com/device", TokenEndpoint: "https://example.com/token"},
			wantErrSub: "client id is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RunDeviceFlow(context.Background(), tc.cfg, nil)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}

func TestRunDeviceFlow_Success_PrintsInstructionsAndReturnsToken(t *testing.T) {
	var deviceCalls int32
	var tokenCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/device":
			atomic.AddInt32(&deviceCalls, 1)
			if r.Method != http.MethodPost {
				t.Fatalf("device endpoint method = %s, want POST", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse device form: %v", err)
			}
			if got := r.PostForm.Get("client_id"); got != "client-id" {
				t.Fatalf("device client_id = %q, want client-id", got)
			}
			if got := r.PostForm.Get("scope"); got != "scope.read scope.write" {
				t.Fatalf("device scope = %q, want joined scopes", got)
			}
			if got := r.PostForm.Get("client_secret"); got != "secret" {
				t.Fatalf("device client_secret = %q, want secret", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"device_code":"dev-code","verification_uri":"https://verify.example.com","user_code":"ABCD-1234","expires_in":600,"interval":1}`))
		case "/token":
			atomic.AddInt32(&tokenCalls, 1)
			if r.Method != http.MethodPost {
				t.Fatalf("token endpoint method = %s, want POST", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			if got := r.PostForm.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:device_code" {
				t.Fatalf("grant_type = %q", got)
			}
			if got := r.PostForm.Get("device_code"); got != "dev-code" {
				t.Fatalf("device_code = %q, want dev-code", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token-123","refresh_token":"refresh-456","expires_in":3600,"scope":"scope.read scope.write"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer func() {
		srv.CloseClientConnections()
		srv.Close()
	}()

	var out bytes.Buffer
	result, err := RunDeviceFlow(context.Background(), DeviceFlowConfig{
		DeviceEndpoint: srv.URL + "/device",
		TokenEndpoint:  srv.URL + "/token",
		ClientID:       "client-id",
		ClientSecret:   "secret",
		Scopes:         []string{"scope.read", "scope.write"},
		AuthMethod:     "body",
		Timeout:        2 * time.Second,
	}, &out)
	if err != nil {
		t.Fatalf("RunDeviceFlow() error = %v", err)
	}

	if result.AccessToken != "token-123" {
		t.Fatalf("access token = %q, want token-123", result.AccessToken)
	}
	if result.RefreshToken != "refresh-456" {
		t.Fatalf("refresh token = %q, want refresh-456", result.RefreshToken)
	}
	if result.ExpiresIn != 3600 {
		t.Fatalf("expires_in = %d, want 3600", result.ExpiresIn)
	}

	if got := atomic.LoadInt32(&deviceCalls); got != 1 {
		t.Fatalf("device endpoint call count = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 1 {
		t.Fatalf("token endpoint call count = %d, want 1", got)
	}

	output := out.String()
	if !strings.Contains(output, "Open this URL in any browser") {
		t.Fatalf("expected instructions in output, got: %q", output)
	}
	if !strings.Contains(output, "https://verify.example.com") {
		t.Fatalf("expected verification URL in output, got: %q", output)
	}
	if !strings.Contains(output, "ABCD-1234") {
		t.Fatalf("expected user code in output, got: %q", output)
	}
}

func TestRunDeviceFlow_ContextCancellation(t *testing.T) {
	tokenCalled := make(chan struct{})
	var closeOnce sync.Once

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/device":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"device_code":"dev-code","verification_uri":"https://verify.example.com","user_code":"ABCD-1234","expires_in":600,"interval":1}`))
		case "/token":
			closeOnce.Do(func() { close(tokenCalled) })
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-tokenCalled:
			cancel()
		case <-ctx.Done():
		}
	}()

	_, err := RunDeviceFlow(ctx, DeviceFlowConfig{
		DeviceEndpoint: srv.URL + "/device",
		TokenEndpoint:  srv.URL + "/token",
		ClientID:       "client-id",
		Timeout:        5 * time.Second,
	}, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("error = %v, want context cancellation", err)
	}
}

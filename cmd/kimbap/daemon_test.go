package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
	"github.com/dunialabs/kimbap/internal/runtime"
)

func buildTestRuntime() *runtime.Runtime {
	return runtime.NewRuntime(runtime.Runtime{
		Adapters: map[string]adapters.Adapter{
			"http": &mockTestAdapter{},
		},
	})
}

type mockTestAdapter struct{}

func (m *mockTestAdapter) Type() string { return "http" }

func (m *mockTestAdapter) Validate(_ actions.ActionDefinition) error { return nil }

func (m *mockTestAdapter) Execute(_ context.Context, _ adapters.AdapterRequest) (*adapters.AdapterResult, error) {
	return &adapters.AdapterResult{
		Output:     map[string]any{"ok": true},
		HTTPStatus: 200,
		DurationMS: 1,
	}, nil
}

func TestDaemonCallHandlerMethodNotAllowed(t *testing.T) {
	rt := buildTestRuntime()
	handler := daemonCallHandler(nil, rt)

	req := httptest.NewRequest(http.MethodGet, "/call", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestDaemonCallHandlerEmptyAction(t *testing.T) {
	rt := buildTestRuntime()
	handler := daemonCallHandler(nil, rt)

	body, _ := json.Marshal(daemonCallRequest{Action: "", Input: nil})
	req := httptest.NewRequest(http.MethodPost, "/call", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDaemonCallHandlerInvalidJSON(t *testing.T) {
	rt := buildTestRuntime()
	handler := daemonCallHandler(nil, rt)

	req := httptest.NewRequest(http.MethodPost, "/call", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDaemonAuthMiddlewareNoToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := daemonAuthMiddleware("", inner)
	req := httptest.NewRequest(http.MethodPost, "/call", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with no token, got %d", rec.Code)
	}
}

func TestDaemonAuthMiddlewareValidToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := daemonAuthMiddleware("secret-token", inner)
	req := httptest.NewRequest(http.MethodPost, "/call", nil)
	req.Header.Set("X-Kimbap-Token", "secret-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", rec.Code)
	}
}

func TestDaemonAuthMiddlewareInvalidToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := daemonAuthMiddleware("secret-token", inner)
	req := httptest.NewRequest(http.MethodPost, "/call", nil)
	req.Header.Set("X-Kimbap-Token", "wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong token, got %d", rec.Code)
	}
}

func TestDaemonAuthMiddlewareHealthBypass(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := daemonAuthMiddleware("secret-token", inner)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for health without token, got %d", rec.Code)
	}
}

func TestDaemonShutdownRequiresPost(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/shutdown", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET /shutdown, got %d", rec.Code)
	}
}

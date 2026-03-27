package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthHandlerReturnsJSONStatus(t *testing.T) {
	h := HealthHandler("1.2.3")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}

	var payload HealthStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode health payload: %v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("status = %q, want ok", payload.Status)
	}
	if payload.Version != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", payload.Version)
	}
	if payload.Timestamp.IsZero() {
		t.Fatal("timestamp should be populated")
	}
	if payload.Timestamp.Location() != time.UTC {
		t.Fatalf("timestamp location = %v, want UTC", payload.Timestamp.Location())
	}
}

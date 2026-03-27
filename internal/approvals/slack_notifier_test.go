package approvals

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSlackNotifierSendsCorrectPayload(t *testing.T) {
	var (
		capturedMethod      string
		capturedContentType string
		capturedBody        []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedContentType = r.Header.Get("Content-Type")
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewSlackNotifier(srv.URL)
	req := &ApprovalRequest{
		ID:        "appr_test123",
		Service:   "github",
		Action:    "delete_repo",
		AgentName: "test-agent",
		Risk:      "high",
	}
	err := notifier.Notify(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if capturedContentType != "application/json" {
		t.Errorf("expected application/json, got %s", capturedContentType)
	}
	var payload map[string]string
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	text, ok := payload["text"]
	if !ok {
		t.Fatal("missing 'text' field in payload")
	}
	for _, want := range []string{"appr_test123", "github", "delete_repo", "test-agent", "kimbap approve"} {
		if !strings.Contains(text, want) {
			t.Errorf("text %q missing expected substring %q", text, want)
		}
	}
}

func TestSlackNotifierErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	notifier := NewSlackNotifier(srv.URL)
	err := notifier.Notify(context.Background(), &ApprovalRequest{ID: "x", Service: "s", Action: "a"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestSlackNotifierErrorOnEmptyURL(t *testing.T) {
	notifier := NewSlackNotifier("")
	err := notifier.Notify(context.Background(), &ApprovalRequest{ID: "x"})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

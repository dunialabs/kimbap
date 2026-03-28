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

func TestTelegramNotifierSendsCorrectPayload(t *testing.T) {
	var capturedPath string
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewTelegramNotifier("mytoken123", "mychat456", withBaseURL(srv.URL))
	req := &ApprovalRequest{ID: "appr_tg1", Service: "stripe", Action: "refund", AgentName: "billing-bot", Risk: "high"}
	if err := notifier.Notify(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPath, "/botmytoken123/sendMessage") {
		t.Errorf("unexpected path %q", capturedPath)
	}
	var body map[string]string
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body["chat_id"] != "mychat456" {
		t.Errorf("wrong chat_id: %q", body["chat_id"])
	}
	if body["parse_mode"] != "" {
		t.Errorf("expected no parse_mode (plain text), got %q", body["parse_mode"])
	}
	if !strings.Contains(body["text"], "appr_tg1") {
		t.Errorf("text missing approval ID")
	}
	if !strings.Contains(body["text"], "kimbap approve") {
		t.Errorf("text missing approve instruction")
	}
}

func TestTelegramNotifierErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	notifier := NewTelegramNotifier("tok", "chat", withBaseURL(srv.URL))
	if err := notifier.Notify(context.Background(), &ApprovalRequest{ID: "x", Service: "s", Action: "a"}); err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestTelegramNotifierErrorOnEmptyToken(t *testing.T) {
	notifier := NewTelegramNotifier("", "chatid")
	if err := notifier.Notify(context.Background(), &ApprovalRequest{ID: "x"}); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestTelegramNotifierErrorOnEmptyChatID(t *testing.T) {
	notifier := NewTelegramNotifier("token", "")
	if err := notifier.Notify(context.Background(), &ApprovalRequest{ID: "x"}); err == nil {
		t.Fatal("expected error for empty chat ID")
	}
}

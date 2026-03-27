package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJSONLWriterRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	w, err := NewJSONLWriter(path)
	if err != nil {
		t.Fatalf("new jsonl writer: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	event := AuditEvent{
		ID:        "evt-1",
		Timestamp: time.Now().UTC(),
		TenantID:  "tenant-a",
		Action:    "tools.call",
		Status:    AuditStatusSuccess,
		Input:     map[string]any{"tool": "github"},
	}
	if err := w.Write(context.Background(), event); err != nil {
		t.Fatalf("write audit event: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close jsonl writer: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read jsonl output: %v", err)
	}

	var decoded AuditEvent
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode jsonl row: %v", err)
	}
	if decoded.ID != event.ID {
		t.Fatalf("event id mismatch: got=%q want=%q", decoded.ID, event.ID)
	}
	if decoded.Status != event.Status {
		t.Fatalf("event status mismatch: got=%q want=%q", decoded.Status, event.Status)
	}
}

func TestRedactorRemovesMatchingFields(t *testing.T) {
	r := NewRedactor("token", "password")

	event := AuditEvent{
		Input: map[string]any{
			"api_token": "secret",
			"payload": map[string]any{
				"password": "hidden",
				"safe":     "ok",
			},
		},
		Meta: map[string]any{"token_hint": "abcd", "request_id": "r-1"},
	}

	redacted := r.Redact(event)
	if redacted.Input["api_token"] != redactedValue {
		t.Fatalf("expected api_token to be redacted")
	}
	payload, ok := redacted.Input["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload to remain a map")
	}
	if payload["password"] != redactedValue {
		t.Fatalf("expected nested password to be redacted")
	}
	if payload["safe"] != "ok" {
		t.Fatalf("expected non-sensitive key to remain untouched")
	}
	if redacted.Meta["token_hint"] != redactedValue {
		t.Fatalf("expected token_hint in meta to be redacted")
	}
}

func TestMultiWriterFansOutToTwoWriters(t *testing.T) {
	left := &captureWriter{}
	right := &captureWriter{}
	mw := NewMultiWriter(left, right)

	event := AuditEvent{ID: "evt-2", Status: AuditStatusSuccess}
	if err := mw.Write(context.Background(), event); err != nil {
		t.Fatalf("multiwriter write: %v", err)
	}

	if len(left.events) != 1 || len(right.events) != 1 {
		t.Fatalf("expected fanout to 2 writers, got left=%d right=%d", len(left.events), len(right.events))
	}
	if left.events[0].ID != event.ID || right.events[0].ID != event.ID {
		t.Fatalf("fanout event mismatch")
	}
}

type captureWriter struct {
	events []AuditEvent
}

func (c *captureWriter) Write(_ context.Context, event AuditEvent) error {
	c.events = append(c.events, event)
	return nil
}

func (c *captureWriter) Close() error {
	return nil
}

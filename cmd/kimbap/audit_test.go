package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/audit"
)

func TestForEachAuditEventReadsLargeJSONLRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	largeInput := strings.Repeat("x", 5<<20)
	event := audit.AuditEvent{
		ID:        "evt_large",
		Timestamp: time.Now().UTC(),
		AgentName: "agent-a",
		Service:   "svc",
		Action:    "act",
		Status:    audit.AuditStatusSuccess,
		Input: map[string]any{
			"payload": largeInput,
		},
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal large event: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write audit file: %v", err)
	}

	seen := 0
	err = forEachAuditEvent(path, func(item audit.AuditEvent) error {
		seen++
		if item.ID != "evt_large" {
			t.Fatalf("unexpected event id %q", item.ID)
		}
		payload, _ := item.Input["payload"].(string)
		if len(payload) != len(largeInput) {
			t.Fatalf("payload length mismatch: got=%d want=%d", len(payload), len(largeInput))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("forEachAuditEvent failed: %v", err)
	}
	if seen != 1 {
		t.Fatalf("expected 1 event, got %d", seen)
	}
}

func TestForEachAuditEventReturnsNilForMissingFile(t *testing.T) {
	err := forEachAuditEvent(filepath.Join(t.TempDir(), "missing.jsonl"), func(audit.AuditEvent) error {
		t.Fatal("handler should not be called for missing file")
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil for missing file, got %v", err)
	}
}

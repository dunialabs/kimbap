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

func TestParseAuditSinceDurationSupportsDays(t *testing.T) {
	dur, err := parseAuditSinceDuration("7d")
	if err != nil {
		t.Fatalf("parseAuditSinceDuration returned error: %v", err)
	}
	if want := 7 * 24 * time.Hour; dur != want {
		t.Fatalf("expected %s, got %s", want, dur)
	}
}

func TestResolveAuditQueryWindowRejectsMixedSinceAndRange(t *testing.T) {
	_, err := resolveAuditQueryWindow("24h", "2026-04-19", "", 24*time.Hour, time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected mixed --since/--from to fail")
	}
}

func TestSummarizeAuditEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	events := []audit.AuditEvent{
		newAuditFixtureEvent("evt1", "2026-04-19T10:00:00Z", "codex", "github", "list-issues", "call", audit.AuditStatusSuccess, 50, ""),
		newAuditFixtureEvent("evt2", "2026-04-19T10:05:00Z", "codex", "github", "list-repos", "call", audit.AuditStatusError, 120, "ERR_DOWNSTREAM"),
		newAuditFixtureEvent("evt3", "2026-04-19T10:10:00Z", "claude", "slack", "post-message", "serve", audit.AuditStatusApprovalRequired, 300, ""),
		newAuditFixtureEvent("evt4", "2026-04-19T10:15:00Z", "codex", "github", "list-issues", "call", audit.AuditStatusValidationFailed, 20, "ERR_VALIDATION"),
		newAuditFixtureEvent("evt5", "2026-04-19T10:20:00Z", "codex", "notion", "query", "call", audit.AuditStatusTimeout, 500, "ERR_TIMEOUT"),
		newAuditFixtureEvent("evt6", "2026-04-19T10:25:00Z", "claude", "github", "list-issues", "call", audit.AuditStatusDenied, 10, ""),
		newAuditFixtureEvent("evt7", "2026-04-19T10:30:00Z", "codex", "stripe", "charge", "call", audit.AuditStatusCancelled, 40, "ERR_CANCELLED"),
	}
	writeAuditFixture(t, path, events)

	fromTime, err := parseAuditTime("2026-04-19")
	if err != nil {
		t.Fatalf("parseAuditTime returned error: %v", err)
	}
	toTime, err := parseAuditTimeTo("2026-04-20")
	if err != nil {
		t.Fatalf("parseAuditTimeTo returned error: %v", err)
	}
	summary, err := summarizeAuditEvents(path, auditQueryWindow{
		Kind: "range",
		From: &fromTime,
		To:   &toTime,
	}, newAuditEventFilter("", "", "", ""))
	if err != nil {
		t.Fatalf("summarizeAuditEvents returned error: %v", err)
	}

	if summary.MatchedEvents != 7 {
		t.Fatalf("expected 7 matched events, got %d", summary.MatchedEvents)
	}
	if summary.StatusCounts.Success != 1 || summary.StatusCounts.Error != 1 || summary.StatusCounts.Denied != 1 ||
		summary.StatusCounts.ApprovalRequired != 1 || summary.StatusCounts.ValidationFailed != 1 ||
		summary.StatusCounts.Timeout != 1 || summary.StatusCounts.Cancelled != 1 {
		t.Fatalf("unexpected status counts: %+v", summary.StatusCounts)
	}
	if summary.LatencyMS.Avg != 148 || summary.LatencyMS.P50 != 50 || summary.LatencyMS.P95 != 500 || summary.LatencyMS.Max != 500 {
		t.Fatalf("unexpected latency summary: %+v", summary.LatencyMS)
	}
	if len(summary.TopServices) == 0 || summary.TopServices[0].Key != "github" || summary.TopServices[0].Count != 4 {
		t.Fatalf("unexpected top services: %+v", summary.TopServices)
	}
	if len(summary.TopAgents) == 0 || summary.TopAgents[0].Key != "codex" || summary.TopAgents[0].Count != 5 {
		t.Fatalf("unexpected top agents: %+v", summary.TopAgents)
	}
	if len(summary.ModeCounts) != 2 || summary.ModeCounts[0].Key != "call" || summary.ModeCounts[0].Count != 6 {
		t.Fatalf("unexpected mode counts: %+v", summary.ModeCounts)
	}
	if len(summary.TopErrorCodes) != 3 {
		t.Fatalf("expected top error preview limit of 3, got %+v", summary.TopErrorCodes)
	}
}

func TestRenderAuditSummaryText(t *testing.T) {
	summary := auditSummaryResult{
		Window:        auditWindowOutput{Kind: "since", Value: "24h", From: "2026-04-19T00:00:00Z", To: "2026-04-20T00:00:00Z"},
		MatchedEvents: 2,
		StatusCounts: auditStatusCounts{
			Success: 1,
			Error:   1,
		},
		StatusRatios: auditStatusRatios{
			Success: 0.5,
			Error:   0.5,
		},
		LatencyMS: auditLatencySummary{Avg: 85, P50: 50, P95: 120, Max: 120},
		TopServices: []auditCountItem{
			{Key: "github", Count: 2},
		},
	}

	text := renderAuditSummaryText(summary)
	if !strings.Contains(text, "Window:") || !strings.Contains(text, "last 24h") {
		t.Fatalf("expected window header, got:\n%s", text)
	}
	if !strings.Contains(text, "Status:") || !strings.Contains(text, "success") || !strings.Contains(text, "error") {
		t.Fatalf("expected status section, got:\n%s", text)
	}
	if !strings.Contains(text, "Latency:") || !strings.Contains(text, "p95 120ms") {
		t.Fatalf("expected latency section, got:\n%s", text)
	}
	if !strings.Contains(text, "Top Services:") || !strings.Contains(text, "github") {
		t.Fatalf("expected top services section, got:\n%s", text)
	}
}

func TestAuditSummaryCommandOutputsJSON(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "json"

	dataDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeMinimalConfig(t, configPath, dataDir, t.TempDir())
	writeAuditFixture(t, filepath.Join(dataDir, "audit.jsonl"), []audit.AuditEvent{
		newAuditFixtureEvent("evt1", "2026-04-19T10:00:00Z", "codex", "github", "list-issues", "call", audit.AuditStatusSuccess, 50, ""),
		newAuditFixtureEvent("evt2", "2026-04-19T10:05:00Z", "codex", "github", "list-repos", "call", audit.AuditStatusError, 120, "ERR_DOWNSTREAM"),
	})

	opts.configPath = configPath
	output, err := captureStdout(t, func() error {
		cmd := newAuditSummaryCommand()
		cmd.SetArgs([]string{"--from", "2026-04-19", "--to", "2026-04-20"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("summary command failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("summary output is not valid JSON: %v\noutput=%s", err, output)
	}
	if got := int(payload["matched_events"].(float64)); got != 2 {
		t.Fatalf("expected matched_events=2, got %d", got)
	}
	filters, _ := payload["filters"].(map[string]any)
	if filters == nil {
		t.Fatalf("expected filters object, got %v", payload["filters"])
	}
}

func newAuditFixtureEvent(id string, timestamp string, agent string, service string, action string, mode string, status audit.AuditStatus, durationMS int64, errorCode string) audit.AuditEvent {
	eventTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		panic(err)
	}
	event := audit.AuditEvent{
		ID:         id,
		Timestamp:  eventTime.UTC(),
		AgentName:  agent,
		Service:    service,
		Action:     action,
		Mode:       mode,
		Status:     status,
		DurationMS: durationMS,
	}
	if errorCode != "" {
		event.Error = &audit.AuditError{Code: errorCode}
	}
	return event
}

func writeAuditFixture(t *testing.T, path string, events []audit.AuditEvent) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create audit fixture: %v", err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			t.Fatalf("encode audit fixture: %v", err)
		}
	}
}

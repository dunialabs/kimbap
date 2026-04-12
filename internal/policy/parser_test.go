package policy

import (
	"strings"
	"testing"
)

func TestParseDocumentValidMinimal(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: allow-all
    priority: 1
    match:
      actions: ["*"]
    decision: allow
`)
	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if len(doc.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(doc.Rules))
	}
}

func TestParseDocumentRejectsDuplicateIDs(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: dup
    priority: 1
    match: { actions: ["*"] }
    decision: allow
  - id: dup
    priority: 2
    match: { actions: ["*"] }
    decision: deny
`)
	_, err := ParseDocument(data)
	if err == nil {
		t.Fatal("expected duplicate ID validation error")
	}
	if !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected unique error, got %v", err)
	}
}

func TestParseDocumentRejectsUnknownFields(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: typo
    priority: 1
    match: { actions: ["*"] }
    decsion: allow
`)
	_, err := ParseDocument(data)
	if err == nil {
		t.Fatal("expected unknown-field parse error")
	}
	if !strings.Contains(err.Error(), "decsion") {
		t.Fatalf("expected unknown field in error, got %v", err)
	}
}

func TestParseDocumentRejectsInvalidTimeWindowAfter(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: r1
    priority: 1
    match: { actions: ["*"] }
    decision: allow
    time_window:
      after: "25:00"
`)
	_, err := ParseDocument(data)
	if err == nil {
		t.Fatal("expected time_window.after validation error")
	}
	if !strings.Contains(err.Error(), "HH:MM") {
		t.Fatalf("expected HH:MM message, got %v", err)
	}
}

func TestParseDocumentRejectsInvalidTimeWindowBefore(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: r1
    priority: 1
    match: { actions: ["*"] }
    decision: allow
    time_window:
      before: "not-a-time"
`)
	_, err := ParseDocument(data)
	if err == nil {
		t.Fatal("expected time_window.before validation error")
	}
	if !strings.Contains(err.Error(), "HH:MM") {
		t.Fatalf("expected HH:MM message, got %v", err)
	}
}

func TestParseDocumentRejectsInvalidTimezone(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: r1
    priority: 1
    match: { actions: ["*"] }
    decision: allow
    time_window:
      after: "09:00"
      timezone: "Not/A/Timezone"
`)
	_, err := ParseDocument(data)
	if err == nil {
		t.Fatal("expected time_window.timezone validation error")
	}
	if !strings.Contains(err.Error(), "IANA") {
		t.Fatalf("expected IANA message, got %v", err)
	}
}

func TestParseDocumentRejectsInvalidWeekday(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: r1
    priority: 1
    match: { actions: ["*"] }
    decision: allow
    time_window:
      after: "09:00"
      before: "17:00"
      weekdays: ["monday", "notaday"]
`)
	_, err := ParseDocument(data)
	if err == nil {
		t.Fatal("expected weekday validation error")
	}
	if !strings.Contains(err.Error(), "weekday") {
		t.Fatalf("expected weekday message, got %v", err)
	}
}

func TestParseDocumentAcceptsValidTimeWindow(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: business-hours
    priority: 1
    match: { actions: ["*"] }
    decision: allow
    time_window:
      after: "09:00"
      before: "17:00"
      timezone: "America/New_York"
      weekdays: ["mon", "tue", "wed", "thu", "fri"]
`)
	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if doc.Rules[0].TimeWindow == nil {
		t.Fatal("expected time_window to be parsed")
	}
	if doc.Rules[0].TimeWindow.After != "09:00" {
		t.Fatalf("unexpected after: %s", doc.Rules[0].TimeWindow.After)
	}
}

func TestParseDocumentAcceptsNightShiftTimeWindow(t *testing.T) {
	data := []byte(`
version: "1.0.0"
rules:
  - id: night-shift
    priority: 1
    match: { actions: ["*"] }
    decision: allow
    time_window:
      after: "22:00"
      before: "06:00"
      weekdays: ["monday", "tuesday", "wednesday", "thursday", "friday"]
`)
	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("expected valid night-shift window, got %v", err)
	}
	if doc.Rules[0].TimeWindow.After != "22:00" || doc.Rules[0].TimeWindow.Before != "06:00" {
		t.Fatalf("unexpected window: %+v", doc.Rules[0].TimeWindow)
	}
}

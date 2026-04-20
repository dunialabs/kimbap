package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/audit"
)

type auditEventFilter struct {
	Agent   string
	Service string
	Action  string
	Status  string
}

type auditQueryWindow struct {
	Kind  string
	Value string
	From  *time.Time
	To    *time.Time
}

type auditWindowOutput struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
}

func newAuditEventFilter(agent string, service string, action string, status string) auditEventFilter {
	return auditEventFilter{
		Agent:   strings.TrimSpace(agent),
		Service: strings.TrimSpace(service),
		Action:  strings.TrimSpace(action),
		Status:  strings.TrimSpace(status),
	}
}

func (f auditEventFilter) matches(event audit.AuditEvent) bool {
	if f.Agent != "" && !strings.EqualFold(strings.TrimSpace(event.AgentName), f.Agent) {
		return false
	}
	if f.Service != "" && !strings.EqualFold(strings.TrimSpace(event.Service), f.Service) {
		return false
	}
	if f.Action != "" && !strings.EqualFold(strings.TrimSpace(event.Action), f.Action) {
		return false
	}
	if f.Status != "" && !strings.EqualFold(string(event.Status), f.Status) {
		return false
	}
	return true
}

func (f auditEventFilter) output() auditSummaryFilters {
	return auditSummaryFilters{
		Agent:   f.Agent,
		Service: f.Service,
		Action:  f.Action,
		Status:  f.Status,
	}
}

func parseAuditSinceDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return 0, fmt.Errorf("--since is required")
	}
	if strings.HasSuffix(raw, "d") {
		daysRaw := strings.TrimSuffix(raw, "d")
		days, err := strconv.Atoi(daysRaw)
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("unsupported --since value %q", raw)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	dur, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse --since: %w", err)
	}
	if dur <= 0 {
		return 0, fmt.Errorf("--since must be greater than 0")
	}
	return dur, nil
}

func resolveAuditQueryWindow(since string, from string, to string, defaultSince time.Duration, now time.Time) (auditQueryWindow, error) {
	since = strings.TrimSpace(since)
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	now = now.UTC()

	if since != "" && (from != "" || to != "") {
		return auditQueryWindow{}, fmt.Errorf("--since cannot be combined with --from or --to")
	}

	if since == "" && from == "" && to == "" {
		since = defaultSince.String()
	}

	if since != "" {
		dur, err := parseAuditSinceDuration(since)
		if err != nil {
			return auditQueryWindow{}, err
		}
		fromTime := now.Add(-dur)
		toTime := now
		return auditQueryWindow{
			Kind:  "since",
			Value: since,
			From:  &fromTime,
			To:    &toTime,
		}, nil
	}

	window := auditQueryWindow{Kind: "range"}
	if from != "" {
		fromTime, err := parseAuditTime(from)
		if err != nil {
			return auditQueryWindow{}, fmt.Errorf("parse --from: %w", err)
		}
		window.From = &fromTime
	}
	if to != "" {
		toTime, err := parseAuditTimeTo(to)
		if err != nil {
			return auditQueryWindow{}, fmt.Errorf("parse --to: %w", err)
		}
		window.To = &toTime
	}
	if window.From != nil && window.To == nil {
		toTime := now
		window.To = &toTime
	}
	if window.From != nil && window.To != nil && window.To.Before(*window.From) {
		return auditQueryWindow{}, fmt.Errorf("--to must be after or equal to --from")
	}
	return window, nil
}

func (w auditQueryWindow) contains(ts time.Time) bool {
	if w.From != nil && ts.Before(*w.From) {
		return false
	}
	if w.To != nil && ts.After(*w.To) {
		return false
	}
	return true
}

func (w auditQueryWindow) output() auditWindowOutput {
	out := auditWindowOutput{
		Kind:  w.Kind,
		Value: w.Value,
	}
	if w.From != nil {
		out.From = w.From.Format(time.RFC3339)
	}
	if w.To != nil {
		out.To = w.To.Format(time.RFC3339)
	}
	return out
}

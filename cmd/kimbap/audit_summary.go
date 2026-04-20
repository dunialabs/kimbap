package main

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/audit"
)

const auditSummaryTopPreviewLimit = 3

type auditSummaryResult struct {
	Window        auditWindowOutput   `json:"window"`
	Filters       auditSummaryFilters `json:"filters"`
	MatchedEvents int                 `json:"matched_events"`
	StatusCounts  auditStatusCounts   `json:"status_counts"`
	StatusRatios  auditStatusRatios   `json:"status_ratios"`
	LatencyMS     auditLatencySummary `json:"latency_ms"`
	ModeCounts    []auditCountItem    `json:"mode_counts"`
	TopServices   []auditCountItem    `json:"top_services"`
	TopActions    []auditCountItem    `json:"top_actions"`
	TopAgents     []auditCountItem    `json:"top_agents"`
	TopErrorCodes []auditCountItem    `json:"top_error_codes"`
}

type auditSummaryFilters struct {
	Agent   string `json:"agent"`
	Service string `json:"service"`
	Action  string `json:"action"`
	Status  string `json:"status"`
}

type auditStatusCounts struct {
	Success          int `json:"success"`
	Error            int `json:"error"`
	Denied           int `json:"denied"`
	ApprovalRequired int `json:"approval_required"`
	ValidationFailed int `json:"validation_failed"`
	Timeout          int `json:"timeout"`
	Cancelled        int `json:"cancelled"`
}

type auditStatusRatios struct {
	Success          float64 `json:"success"`
	Error            float64 `json:"error"`
	Denied           float64 `json:"denied"`
	ApprovalRequired float64 `json:"approval_required"`
	ValidationFailed float64 `json:"validation_failed"`
	Timeout          float64 `json:"timeout"`
	Cancelled        float64 `json:"cancelled"`
}

type auditLatencySummary struct {
	Avg int64 `json:"avg"`
	P50 int64 `json:"p50"`
	P95 int64 `json:"p95"`
	Max int64 `json:"max"`
}

type auditCountItem struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type auditSummaryAggregate struct {
	matched     int
	statuses    auditStatusCounts
	sumDuration int64
	maxDuration int64
	durations   []int64
	modeCounts  map[string]int
	serviceHits map[string]int
	actionHits  map[string]int
	agentHits   map[string]int
	errorHits   map[string]int
}

func summarizeAuditEvents(path string, window auditQueryWindow, filter auditEventFilter) (auditSummaryResult, error) {
	agg := auditSummaryAggregate{
		modeCounts:  map[string]int{},
		serviceHits: map[string]int{},
		actionHits:  map[string]int{},
		agentHits:   map[string]int{},
		errorHits:   map[string]int{},
	}
	if err := forEachAuditEvent(path, func(event audit.AuditEvent) error {
		if !window.contains(event.Timestamp) || !filter.matches(event) {
			return nil
		}
		agg.consume(event)
		return nil
	}); err != nil {
		return auditSummaryResult{}, err
	}

	return auditSummaryResult{
		Window:        window.output(),
		Filters:       filter.output(),
		MatchedEvents: agg.matched,
		StatusCounts:  agg.statuses,
		StatusRatios:  agg.statusRatios(),
		LatencyMS:     agg.latencySummary(),
		ModeCounts:    topAuditCounts(agg.modeCounts, 0),
		TopServices:   topAuditCounts(agg.serviceHits, auditSummaryTopPreviewLimit),
		TopActions:    topAuditCounts(agg.actionHits, auditSummaryTopPreviewLimit),
		TopAgents:     topAuditCounts(agg.agentHits, auditSummaryTopPreviewLimit),
		TopErrorCodes: topAuditCounts(agg.errorHits, auditSummaryTopPreviewLimit),
	}, nil
}

func (a *auditSummaryAggregate) consume(event audit.AuditEvent) {
	a.matched++
	switch event.Status {
	case audit.AuditStatusSuccess:
		a.statuses.Success++
	case audit.AuditStatusError:
		a.statuses.Error++
	case audit.AuditStatusDenied:
		a.statuses.Denied++
	case audit.AuditStatusApprovalRequired:
		a.statuses.ApprovalRequired++
	case audit.AuditStatusValidationFailed:
		a.statuses.ValidationFailed++
	case audit.AuditStatusTimeout:
		a.statuses.Timeout++
	case audit.AuditStatusCancelled:
		a.statuses.Cancelled++
	}

	a.sumDuration += event.DurationMS
	if event.DurationMS > a.maxDuration {
		a.maxDuration = event.DurationMS
	}
	a.durations = append(a.durations, event.DurationMS)

	if key := strings.TrimSpace(event.Mode); key != "" {
		a.modeCounts[key]++
	}
	if key := strings.TrimSpace(event.Service); key != "" {
		a.serviceHits[key]++
	}
	if key := auditActionKey(event); key != "" {
		a.actionHits[key]++
	}
	if key := strings.TrimSpace(event.AgentName); key != "" {
		a.agentHits[key]++
	}
	if event.Error != nil {
		if key := strings.TrimSpace(event.Error.Code); key != "" {
			a.errorHits[key]++
		}
	}
}

func (a auditSummaryAggregate) statusRatios() auditStatusRatios {
	total := float64(a.matched)
	if total == 0 {
		return auditStatusRatios{}
	}
	return auditStatusRatios{
		Success:          float64(a.statuses.Success) / total,
		Error:            float64(a.statuses.Error) / total,
		Denied:           float64(a.statuses.Denied) / total,
		ApprovalRequired: float64(a.statuses.ApprovalRequired) / total,
		ValidationFailed: float64(a.statuses.ValidationFailed) / total,
		Timeout:          float64(a.statuses.Timeout) / total,
		Cancelled:        float64(a.statuses.Cancelled) / total,
	}
}

func (a auditSummaryAggregate) latencySummary() auditLatencySummary {
	if a.matched == 0 {
		return auditLatencySummary{}
	}
	sorted := append([]int64(nil), a.durations...)
	slices.Sort(sorted)
	return auditLatencySummary{
		Avg: a.sumDuration / int64(a.matched),
		P50: percentileNearestRank(sorted, 50),
		P95: percentileNearestRank(sorted, 95),
		Max: a.maxDuration,
	}
}

func auditActionKey(event audit.AuditEvent) string {
	service := strings.TrimSpace(event.Service)
	action := strings.TrimSpace(event.Action)
	switch {
	case service != "" && action != "":
		return service + "." + action
	case action != "":
		return action
	default:
		return ""
	}
}

func topAuditCounts(counts map[string]int, limit int) []auditCountItem {
	items := make([]auditCountItem, 0, len(counts))
	for key, count := range counts {
		if strings.TrimSpace(key) == "" || count <= 0 {
			continue
		}
		items = append(items, auditCountItem{Key: key, Count: count})
	}
	slices.SortFunc(items, func(left auditCountItem, right auditCountItem) int {
		if left.Count != right.Count {
			if left.Count > right.Count {
				return -1
			}
			return 1
		}
		return strings.Compare(left.Key, right.Key)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func percentileNearestRank(sorted []int64, percentile int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil(float64(percentile) / 100 * float64(len(sorted))))
	if rank <= 0 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

func renderAuditSummaryText(summary auditSummaryResult) string {
	lines := []string{
		fmt.Sprintf("%-14s%s", "Window:", formatAuditWindowLineFromOutput(summary.Window)),
		fmt.Sprintf("%-14s%s", "Range:", formatAuditRangeLineFromOutput(summary.Window)),
		fmt.Sprintf("%-14s%d events", "Matched:", summary.MatchedEvents),
	}

	if summary.MatchedEvents == 0 {
		lines = append(lines, "", "No audit events matched.")
		return strings.Join(lines, "\n")
	}

	lines = append(lines, "", "Status:")
	for _, row := range auditStatusTextRows(summary.StatusCounts, summary.StatusRatios) {
		lines = append(lines, row)
	}

	lines = append(lines,
		"",
		"Latency:",
		fmt.Sprintf("  avg %dms   p50 %dms   p95 %dms   max %dms",
			summary.LatencyMS.Avg,
			summary.LatencyMS.P50,
			summary.LatencyMS.P95,
			summary.LatencyMS.Max,
		),
	)

	appendAuditCountSection(&lines, "Modes:", summary.ModeCounts)
	appendAuditCountSection(&lines, "Top Services:", summary.TopServices)
	appendAuditCountSection(&lines, "Top Actions:", summary.TopActions)
	appendAuditCountSection(&lines, "Top Agents:", summary.TopAgents)
	appendAuditCountSection(&lines, "Top Error Codes:", summary.TopErrorCodes)

	return strings.Join(lines, "\n")
}

func auditStatusTextRows(counts auditStatusCounts, ratios auditStatusRatios) []string {
	rows := make([]string, 0, 7)
	add := func(label string, count int, ratio float64) {
		if count == 0 {
			return
		}
		rows = append(rows, fmt.Sprintf("  %-20s %5d  %5.1f%%", label, count, ratio*100))
	}
	add("success", counts.Success, ratios.Success)
	add("error", counts.Error, ratios.Error)
	add("denied", counts.Denied, ratios.Denied)
	add("approval_required", counts.ApprovalRequired, ratios.ApprovalRequired)
	add("validation_failed", counts.ValidationFailed, ratios.ValidationFailed)
	add("timeout", counts.Timeout, ratios.Timeout)
	add("cancelled", counts.Cancelled, ratios.Cancelled)
	if len(rows) == 0 {
		rows = append(rows, "  (none)")
	}
	return rows
}

func appendAuditCountSection(lines *[]string, title string, items []auditCountItem) {
	if len(items) == 0 {
		return
	}
	*lines = append(*lines, "", title)
	for _, item := range items {
		*lines = append(*lines, fmt.Sprintf("  %-20s %5d", item.Key, item.Count))
	}
}

func formatAuditWindowLineFromOutput(window auditWindowOutput) string {
	if window.Kind == "since" && strings.TrimSpace(window.Value) != "" {
		return "last " + strings.TrimSpace(window.Value)
	}
	return "custom"
}

func formatAuditRangeLineFromOutput(window auditWindowOutput) string {
	return formatAuditTextTime(window.From, "beginning") + " -> " + formatAuditTextTime(window.To, "now")
}

func formatAuditTextTime(raw string, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return ts.UTC().Format("2006-01-02 15:04:05Z07:00")
}

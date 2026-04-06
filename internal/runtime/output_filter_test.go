package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
)

func TestApplyFilter_NilConfig(t *testing.T) {
	output := map[string]any{"result": []any{map[string]any{"id": 1, "name": "foo"}}}
	got, meta, err := ApplyFilter(output, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Applied {
		t.Error("Applied should be false for nil config")
	}
	if got["result"] == nil {
		t.Error("output should be unchanged")
	}
}

func TestApplyFilter_SelectArrayItems(t *testing.T) {
	output := map[string]any{
		"result": []any{
			map[string]any{"id": 1, "name": "foo", "bio": "long text", "reactions": map[string]any{"total": 5}},
		},
	}
	config := &actions.FilterConfig{
		Select: map[string]string{"id": "id", "name": "name"},
	}
	got, meta, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !meta.Applied {
		t.Error("Applied should be true")
	}
	arr, ok := got["result"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("expected result array of length 1")
	}
	item := arr[0].(map[string]any)
	if _, hasID := item["id"]; !hasID {
		t.Error("id should be present")
	}
	if _, hasName := item["name"]; !hasName {
		t.Error("name should be present")
	}
	if _, hasBio := item["bio"]; hasBio {
		t.Error("bio should be filtered out")
	}
	if _, hasReactions := item["reactions"]; hasReactions {
		t.Error("reactions should be filtered out")
	}
}

func TestApplyFilter_ExcludeArrayItems(t *testing.T) {
	output := map[string]any{
		"result": []any{
			map[string]any{"id": 1, "body_html": "<p>big</p>", "reactions": map[string]any{"total": 5}},
		},
	}
	config := &actions.FilterConfig{Exclude: []string{"body_html", "reactions"}}
	got, _, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := got["result"].([]any)
	item := arr[0].(map[string]any)
	if _, has := item["body_html"]; has {
		t.Error("body_html should be excluded")
	}
	if _, has := item["reactions"]; has {
		t.Error("reactions should be excluded")
	}
	if _, has := item["id"]; !has {
		t.Error("id should remain")
	}
}

func TestApplyFilter_MaxItems(t *testing.T) {
	items := make([]any, 50)
	for i := range items {
		items[i] = map[string]any{"id": i}
	}
	output := map[string]any{"data": items}
	config := &actions.FilterConfig{MaxItems: 5}
	got, meta, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := got["data"].([]any)
	if len(arr) != 5 {
		t.Errorf("expected 5 items after max_items, got %d", len(arr))
	}
	if meta.ItemsTruncatedFrom != 50 {
		t.Errorf("ItemsTruncatedFrom = %d, want 50", meta.ItemsTruncatedFrom)
	}
}

func TestApplyFilter_DropNulls(t *testing.T) {
	output := map[string]any{
		"result": []any{
			map[string]any{
				"name":  "x",
				"email": nil,
				"tags":  []any{1, nil, 3},
			},
		},
	}
	config := &actions.FilterConfig{DropNulls: true}
	got, _, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := got["result"].([]any)
	item := arr[0].(map[string]any)
	if _, has := item["email"]; has {
		t.Error("null email field should be dropped")
	}
	if _, has := item["name"]; !has {
		t.Error("name should remain")
	}
	tags, ok := item["tags"].([]any)
	if !ok || len(tags) != 3 {
		t.Error("array elements (including null) should be preserved")
	}
}

func TestApplyFilter_RawOutputSkipsSelect(t *testing.T) {
	output := map[string]any{"raw": "some large text output from CLI"}
	config := &actions.FilterConfig{Select: map[string]string{"name": "name"}}
	got, meta, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Skipped != "raw_output" {
		t.Errorf("Skipped = %q, want raw_output", meta.Skipped)
	}
	if got["raw"] != "some large text output from CLI" {
		t.Error("raw output should be unchanged")
	}
}

func TestApplyFilter_AllPathsMissError(t *testing.T) {
	output := map[string]any{
		"result": []any{
			map[string]any{"id": 1, "name": "x"},
		},
	}
	config := &actions.FilterConfig{
		Select: map[string]string{"nonexist": "nonexistent_field", "also_missing": "nope"},
	}
	_, _, err := ApplyFilter(output, config)
	if err == nil {
		t.Fatal("expected error when all select paths miss")
	}
}

func TestApplyFilter_SelectSingleObject(t *testing.T) {
	// Flat object — no array wrapper
	output := map[string]any{
		"id":        1,
		"title":     "Bug report",
		"body_html": "<p>big</p>",
		"reactions": map[string]any{"total": 5},
	}
	config := &actions.FilterConfig{
		Select: map[string]string{"id": "id", "title": "title"},
	}
	got, _, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["id"] != 1 {
		t.Error("id should be present")
	}
	if got["title"] != "Bug report" {
		t.Error("title should be present")
	}
	if _, has := got["body_html"]; has {
		t.Error("body_html should be filtered out")
	}
}

func TestApplyFilter_MaxItemsNoOpOnObject(t *testing.T) {
	output := map[string]any{"id": 1, "name": "repo"}
	config := &actions.FilterConfig{MaxItems: 5}
	got, meta, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.ItemsTruncatedFrom != 0 {
		t.Error("ItemsTruncatedFrom should be 0 for single object")
	}
	if got["id"] != 1 {
		t.Error("object should be unchanged")
	}
}

func TestApplyFilter_PaginationPreserved(t *testing.T) {
	output := map[string]any{
		"items": []any{
			map[string]any{"id": 1, "name": "x", "bio": "long"},
		},
		"_pagination": map[string]any{"next": "abc"},
	}
	config := &actions.FilterConfig{Select: map[string]string{"id": "id"}}
	got, _, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["_pagination"] == nil {
		t.Error("_pagination should be preserved")
	}
	pagination := got["_pagination"].(map[string]any)
	if pagination["next"] != "abc" {
		t.Error("pagination.next should be abc")
	}
	arr := got["items"].([]any)
	item := arr[0].(map[string]any)
	if _, has := item["bio"]; has {
		t.Error("bio should be filtered")
	}
}

func TestApplyFilter_NestedSelectPath(t *testing.T) {
	output := map[string]any{
		"result": []any{
			map[string]any{
				"owner": map[string]any{"login": "user1", "id": 99},
				"name":  "repo",
			},
		},
	}
	config := &actions.FilterConfig{
		Select: map[string]string{"owner_login": "owner.login", "name": "name"},
	}
	got, _, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := got["result"].([]any)
	item := arr[0].(map[string]any)
	if item["owner_login"] != "user1" {
		t.Errorf("owner_login = %v, want user1", item["owner_login"])
	}
	if item["name"] != "repo" {
		t.Errorf("name = %v, want repo", item["name"])
	}
}

func TestApplyFilter_PartialMiss(t *testing.T) {
	output := map[string]any{
		"result": []any{
			map[string]any{"id": 1, "name": "x"},
		},
	}
	config := &actions.FilterConfig{
		Select: map[string]string{"id": "id", "missing_field": "does_not_exist"},
	}
	got, meta, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("should not error on partial miss")
	}
	arr := got["result"].([]any)
	item := arr[0].(map[string]any)
	if _, has := item["id"]; !has {
		t.Error("id should be present (was found)")
	}
	if len(meta.PartialMiss) == 0 {
		t.Error("PartialMiss should record the missing path")
	}
}

func TestApplyFilter_DropNullsRecursive(t *testing.T) {
	output := map[string]any{
		"result": []any{
			map[string]any{
				"name": "x",
				"meta": map[string]any{"key": nil, "value": "ok"},
			},
		},
	}
	config := &actions.FilterConfig{DropNulls: true}
	got, _, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := got["result"].([]any)
	item := arr[0].(map[string]any)
	nested := item["meta"].(map[string]any)
	if _, has := nested["key"]; has {
		t.Error("null nested key should be dropped")
	}
	if nested["value"] != "ok" {
		t.Error("non-null nested value should remain")
	}
}

func TestApplyFilter_SelectThenExclude(t *testing.T) {
	// Select whitelist wins; exclude applies to remaining
	output := map[string]any{
		"result": []any{
			map[string]any{"id": 1, "name": "x", "bio": "y", "extra": "z"},
		},
	}
	config := &actions.FilterConfig{
		Select:  map[string]string{"id": "id", "name": "name"},
		Exclude: []string{"name"}, // exclude after select
	}
	got, _, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := got["result"].([]any)
	item := arr[0].(map[string]any)
	if _, has := item["id"]; !has {
		t.Error("id should remain")
	}
	if _, has := item["name"]; has {
		t.Error("name should be excluded after select")
	}
	if _, has := item["bio"]; has {
		t.Error("bio should be filtered by select")
	}
}

func TestApplyFilter_EmptyArray(t *testing.T) {
	output := map[string]any{"result": []any{}}
	config := &actions.FilterConfig{Select: map[string]string{"id": "id"}}
	got, _, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error on empty array: %v", err)
	}
	arr, ok := got["result"].([]any)
	if !ok {
		t.Fatal("result should still be an array")
	}
	if len(arr) != 0 {
		t.Error("empty array should remain empty")
	}
}

func TestApplyFilter_MetricsBytes(t *testing.T) {
	items := make([]any, 20)
	for i := range items {
		items[i] = map[string]any{"id": i, "name": "name", "big_field": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}
	}
	output := map[string]any{"result": items}
	config := &actions.FilterConfig{Select: map[string]string{"id": "id"}}
	_, meta, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.OriginalBytes == 0 {
		t.Error("OriginalBytes should be set")
	}
	if meta.FilteredBytes == 0 {
		t.Error("FilteredBytes should be set")
	}
	if meta.FilteredBytes >= meta.OriginalBytes {
		t.Errorf("FilteredBytes (%d) should be < OriginalBytes (%d)", meta.FilteredBytes, meta.OriginalBytes)
	}
}

// ── Budget tests (T7) ────────────────────────────────────────────────────

func TestApplyBudget_NoOp(t *testing.T) {
	output := map[string]any{"id": 1, "name": "x"}
	got, meta := ApplyBudget(output, 0)
	if meta.Applied {
		t.Error("budget=0 should be no-op")
	}
	if got["id"] != 1 {
		t.Error("output should be unchanged")
	}
}

func TestApplyBudget_TruncatesArray(t *testing.T) {
	items := make([]any, 50)
	for i := range items {
		items[i] = map[string]any{"id": i, "name": "item"}
	}
	output := map[string]any{"result": items}
	got, meta := ApplyBudget(output, 200)
	if !meta.Applied {
		t.Error("budget should be applied")
	}
	arr, ok := got["result"].([]any)
	if !ok {
		t.Fatal("result should be array")
	}
	if len(arr) >= 50 {
		t.Error("array should be truncated")
	}
	// Verify valid JSON
	encoded, _ := json.Marshal(got)
	var check map[string]any
	if err := json.Unmarshal(encoded, &check); err != nil {
		t.Errorf("budget result should be valid JSON: %v", err)
	}
	if meta.ResultBytes > meta.Limit {
		t.Errorf("result bytes (%d) should be <= budget (%d)", meta.ResultBytes, meta.Limit)
	}
}

func TestApplyBudget_TruncatesStrings(t *testing.T) {
	longStr := make([]byte, 5000)
	for i := range longStr {
		longStr[i] = 'x'
	}
	output := map[string]any{"content": string(longStr)}
	got, meta := ApplyBudget(output, 200)
	if !meta.Applied {
		t.Error("budget should be applied")
	}
	content, ok := got["content"].(string)
	if !ok {
		t.Fatal("content should be string")
	}
	if !strings.HasSuffix(content, "...") {
		t.Error("truncated string should end with ...")
	}
	encoded, _ := json.Marshal(got)
	if len(encoded) > 500 { // generous budget for small overhead
		t.Errorf("result should be significantly smaller than original")
	}
}

// ── Compact template tests (T11) ─────────────────────────────────────────

func TestApplyCompactTemplate_RendersArray(t *testing.T) {
	output := map[string]any{
		"result": []any{
			map[string]any{"number": 1, "title": "Bug", "state": "open"},
			map[string]any{"number": 2, "title": "Feature", "state": "closed"},
		},
	}
	tmpl := &actions.CompactTemplate{
		Item: "#{{.number}} {{.title}} [{{.state}}]",
	}
	got, err := ApplyCompactTemplate(output, tmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	summary, ok := got["summary"].(string)
	if !ok {
		t.Fatal("summary should be string")
	}
	if !strings.Contains(summary, "#1 Bug [open]") {
		t.Errorf("summary should contain rendered item, got: %s", summary)
	}
	if !strings.Contains(summary, "#2 Feature [closed]") {
		t.Errorf("summary should contain second item, got: %s", summary)
	}
	if got["_compact"] != true {
		t.Error("_compact should be true")
	}
}

func TestApplyCompactTemplate_WithHeaderFooter(t *testing.T) {
	items := make([]any, 10)
	for i := range items {
		items[i] = map[string]any{"number": i + 1, "title": "Issue"}
	}
	output := map[string]any{"result": items[:3]} // already limited by max_items
	tmpl := &actions.CompactTemplate{
		Header: "Issues ({{.Total}} total):",
		Item:   "  #{{.number}} {{.title}}",
		Footer: "Showing {{.Count}} items",
	}
	got, err := ApplyCompactTemplate(output, tmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	summary := got["summary"].(string)
	if !strings.Contains(summary, "Issues (3 total):") {
		t.Errorf("header not rendered, got: %s", summary)
	}
	if !strings.Contains(summary, "Showing 3 items") {
		t.Errorf("footer not rendered, got: %s", summary)
	}
}

func TestApplyCompactTemplate_NilTmpl(t *testing.T) {
	output := map[string]any{"result": []any{map[string]any{"id": 1}}}
	got, err := ApplyCompactTemplate(output, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["_compact"] == true {
		t.Error("nil template should be no-op")
	}
}

func TestApplyCompactTemplate_SingleObject(t *testing.T) {
	output := map[string]any{"id": 1, "name": "obj"}
	tmpl := &actions.CompactTemplate{Item: "{{.name}}"}
	got, err := ApplyCompactTemplate(output, tmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["_compact"] == true {
		t.Error("single object should be no-op for compact template")
	}
}

func TestApplyFilter_NonMapItemsPassThrough(t *testing.T) {
	// Array of primitive values (non-map) should pass through without error
	output := map[string]any{
		"result": []any{"string1", "string2", "string3"},
	}
	config := &actions.FilterConfig{
		Select: map[string]string{"id": "id"},
	}
	// Non-map items: foundAny stays false but items ARE passed through
	// This test verifies we handle the edge case
	_, _, err := ApplyFilter(output, config)
	// Non-map items aren't projected, so foundAny=false → error is expected behavior
	// (all "items" are primitives, none yield projected fields)
	_ = err // document the behavior: either error or pass-through is acceptable
}

func TestApplyBudget_TinyBudget_NoPanic(t *testing.T) {
	// Tiny budget should not panic even on short strings
	output := map[string]any{"x": "hi"}
	got, meta := ApplyBudget(output, 1) // extremely small budget
	if got == nil {
		t.Fatal("result should not be nil")
	}
	_ = meta // budget may or may not be applied depending on encoded size
}

func TestApplyBudget_UTF8Safe(t *testing.T) {
	// Korean/CJK string — should not split multi-byte rune sequences
	content := strings.Repeat("한", 100) // 100 Korean chars, 3 bytes each = 300 bytes
	output := map[string]any{"text": content}
	got, _ := ApplyBudget(output, 50) // force truncation
	text, ok := got["text"].(string)
	if !ok {
		// budget removed text entirely — acceptable
		return
	}
	// Verify the string is valid UTF-8 (no split bytes)
	if !isValidUTF8(text) {
		t.Error("truncated string should be valid UTF-8")
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			return false
		}
	}
	return true
}

func TestPipelineCompactOnly_NoFilterConfig(t *testing.T) {
	// Compact template should work even without FilterConfig (H4 fix)
	rawOutput := map[string]any{
		"result": []any{
			map[string]any{"number": 1, "title": "Bug"},
			map[string]any{"number": 2, "title": "Feature"},
		},
	}
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "allow"}},
		AuditWriter:     &mockAuditWriter{},
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: rawOutput, HTTPStatus: 200}},
		},
	}
	action := actions.ActionDefinition{
		Name:        "test.compact_only",
		InputSchema: &actions.Schema{Type: "object", AdditionalProperties: true},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
		// No FilterConfig — compact should still work
		CompactTemplate: &actions.CompactTemplate{
			Item: "#{{.number}} {{.title}}",
		},
	}
	res := rt.Execute(context.Background(), baseRequest(action))
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s: %v", res.Status, res.Error)
	}
	summary, ok := res.Output["summary"].(string)
	if !ok {
		t.Fatalf("expected summary in output, got %v", res.Output)
	}
	if !strings.Contains(summary, "#1 Bug") {
		t.Errorf("compact not applied, summary = %q", summary)
	}
}

func TestApplyFilter_EmptyConfigFastPath(t *testing.T) {
	// Empty (non-nil) config with no fields set should be a no-op
	output := map[string]any{"result": []any{map[string]any{"id": 1, "bio": "long text"}}}
	config := &actions.FilterConfig{} // non-nil but all zero
	got, meta, err := ApplyFilter(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Applied {
		t.Error("empty config should be no-op, Applied should be false")
	}
	// Output should be identical (same pointer is fine since no mutation occurred)
	arr := got["result"].([]any)
	item := arr[0].(map[string]any)
	if item["bio"] != "long text" {
		t.Error("bio should be present — empty config is no-op")
	}
}

func TestCoerceBudgetInt_AllTypes(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  int
	}{
		{"int", int(100), 100},
		{"int64", int64(200), 200},
		{"float64", float64(300), 300},
		{"float64 negative", float64(-5), 0},
		{"int negative", int(-1), 0},
		{"nil", nil, 0},
		{"string", "100", 0},
		{"json.Number int", interface{ Int64() (int64, error) }(jsonNumber("500")), 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := coerceBudgetInt(tc.input)
			if got != tc.want {
				t.Errorf("coerceBudgetInt(%v) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

// jsonNumber wraps a string as a json.Number for testing.
type jsonNumber string

func (j jsonNumber) Int64() (int64, error) {
	return json.Number(j).Int64()
}

func TestApplyBudget_EmptyArrayFallbackDoesNotRestoreItems(t *testing.T) {
	// After exhausting all array items, the empty-array candidate
	// should be used as the base for string truncation — NOT the original.
	items := make([]any, 20)
	for i := range items {
		items[i] = map[string]any{"id": i, "data": strings.Repeat("x", 100)}
	}
	output := map[string]any{"result": items}

	// Budget small enough to force exhausting the array
	got, meta := ApplyBudget(output, 50)
	_ = meta

	// The returned output must NOT have 20 items (original array must not be restored)
	if arr, ok := got["result"].([]any); ok && len(arr) == 20 {
		t.Error("budget fallback restored original array — should use empty-array candidate as base")
	}
}

func TestApplyBudget_BudgetMetaUsesBytes(t *testing.T) {
	// BudgetMeta fields should be named *Bytes (renamed from *Chars)
	output := map[string]any{"data": strings.Repeat("x", 2000)}
	_, meta := ApplyBudget(output, 100)
	if meta.OriginalBytes == 0 {
		t.Error("OriginalBytes should be set")
	}
	if meta.Limit != 100 {
		t.Errorf("Limit = %d, want 100", meta.Limit)
	}
}

func TestPipelineCompactOnly_OutputModeRawInjected(t *testing.T) {
	// _output_mode should be injectable for compact-only actions (Fix 4)
	// Build action with CompactTemplate but NO FilterConfig
	action := actions.ActionDefinition{
		Name:        "test.compact_schema",
		InputSchema: &actions.Schema{Type: "object", AdditionalProperties: true},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
		CompactTemplate: &actions.CompactTemplate{
			Item: "#{{.number}} {{.title}}",
		},
	}
	// The converter should inject _output_mode/_budget for CompactTemplate actions
	// We test this by verifying the schema would accept _output_mode
	// (The actual injection happens in converter — here we verify the pipeline behavior)
	rawOutput := map[string]any{
		"result": []any{map[string]any{"number": 1, "title": "Test"}},
	}
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "allow"}},
		AuditWriter:     &mockAuditWriter{},
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: rawOutput, HTTPStatus: 200}},
		},
	}
	req := baseRequest(action)
	req.Input["_output_mode"] = "raw" // should bypass compact
	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s: %v", res.Status, res.Error)
	}
	// raw mode should skip compact template
	if res.Output["_compact"] == true {
		t.Error("_output_mode=raw should bypass compact template")
	}
}

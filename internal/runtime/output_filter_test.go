package runtime

import (
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
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

//go:build darwin && integration

package adapters

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const testNotePrefix = "[KIMBAP-TEST]"

func skipIfTCCDenied(t *testing.T, adapter *AppleScriptAdapter) {
	t.Helper()
	if err := adapter.Preflight(context.Background(), "Notes"); err != nil {
		t.Skipf("TCC denied, skipping: %v", err)
	}
}

func skipIfCalendarTCCDenied(t *testing.T, adapter *AppleScriptAdapter) {
	t.Helper()
	if err := adapter.Preflight(context.Background(), "Calendar"); err != nil {
		t.Skipf("Calendar TCC denied, skipping: %v", err)
	}
}

func newTestAdapter() *AppleScriptAdapter {
	return NewAppleScriptAdapter(nil) // real OSAScriptRunner
}

func makeRequest(command, targetApp string, input map[string]any) AdapterRequest {
	return AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				Type:      "applescript",
				Command:   command,
				TargetApp: targetApp,
			},
		},
		Input: input,
	}
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func uniqueTitle(suffix string) string {
	return fmt.Sprintf("%s %s %d", testNotePrefix, suffix, time.Now().UnixNano())
}

// requireOK asserts the result has HTTP 200 and no error.
func requireOK(t *testing.T, result *AdapterResult, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200; output = %+v", result.HTTPStatus, result.Output)
	}
}

// dataArray extracts the "data" key as []any from the result output.
func dataArray(t *testing.T, result *AdapterResult) []any {
	t.Helper()
	raw, ok := result.Output["data"]
	if !ok {
		t.Fatalf("output has no 'data' key: %+v", result.Output)
	}
	arr, ok := raw.([]any)
	if !ok {
		t.Fatalf("output['data'] is %T, want []any", raw)
	}
	return arr
}

// ---------------------------------------------------------------------------
// Basic integration tests (Notes)
// ---------------------------------------------------------------------------

func TestIntegration_ListFolders(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)
	ctx := testCtx(t)

	result, err := adapter.Execute(ctx, makeRequest("list-folders", "Notes", nil))
	requireOK(t, result, err)

	arr := dataArray(t, result)
	t.Logf("output: %+v", result.Output)

	if len(arr) < 1 {
		t.Fatalf("expected at least 1 folder, got %d", len(arr))
	}

	// Each entry should have a "name" field.
	first, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("first folder is %T, want map[string]any", arr[0])
	}
	if _, hasName := first["name"]; !hasName {
		t.Fatalf("first folder missing 'name' key: %+v", first)
	}
}

func TestIntegration_CreateAndGetNote(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)
	ctx := testCtx(t)

	title := uniqueTitle("CreateAndGet")
	body := "Integration test body. Created at " + time.Now().Format(time.RFC3339)

	// Create
	createResult, err := adapter.Execute(ctx, makeRequest("create-note", "Notes", map[string]any{
		"title": title,
		"body":  body,
	}))
	requireOK(t, createResult, err)
	t.Logf("create output: %+v", createResult.Output)

	// Get it back
	getResult, err := adapter.Execute(ctx, makeRequest("get-note", "Notes", map[string]any{
		"name": title,
	}))
	requireOK(t, getResult, err)
	t.Logf("get output: %+v", getResult.Output)

	// Verify title matches
	gotName, _ := getResult.Output["name"].(string)
	if gotName != title {
		t.Fatalf("get-note name = %q, want %q", gotName, title)
	}

	// Verify body contains the text we wrote (Notes.app may add HTML wrapping)
	gotBody, _ := getResult.Output["body"].(string)
	if !strings.Contains(gotBody, "Integration test body") {
		t.Fatalf("get-note body = %q, want it to contain 'Integration test body'", gotBody)
	}

	// NOTE: Apple Notes JXA has no reliable delete. Test note remains with
	// [KIMBAP-TEST] prefix so it's identifiable and can be manually cleaned.
}

func TestIntegration_SearchNotes(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)
	ctx := testCtx(t)

	// Create a note with a unique keyword we can search for.
	keyword := fmt.Sprintf("kimbapsearch%d", time.Now().UnixNano())
	title := uniqueTitle(keyword)

	_, err := adapter.Execute(ctx, makeRequest("create-note", "Notes", map[string]any{
		"title": title,
		"body":  "Body contains " + keyword,
	}))
	if err != nil {
		t.Fatalf("create-note failed: %v", err)
	}

	// Search for the unique keyword
	searchResult, err := adapter.Execute(ctx, makeRequest("search-notes", "Notes", map[string]any{
		"query": keyword,
	}))
	requireOK(t, searchResult, err)
	t.Logf("search output: %+v", searchResult.Output)

	arr := dataArray(t, searchResult)
	if len(arr) < 1 {
		t.Fatalf("search for %q returned 0 results, want >= 1", keyword)
	}

	// Verify our note is in the results
	found := false
	for _, item := range arr {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if name, _ := entry["name"].(string); strings.Contains(name, keyword) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("search results don't contain note with keyword %q", keyword)
	}
}

func TestIntegration_ListNotes(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)
	ctx := testCtx(t)

	result, err := adapter.Execute(ctx, makeRequest("list-notes", "Notes", map[string]any{}))
	requireOK(t, result, err)
	t.Logf("output: %+v (truncated; %d items)", result.Output, len(dataArray(t, result)))

	// Just verify it's a valid array — don't assert on count (user may have any number).
	arr := dataArray(t, result)
	if len(arr) > 0 {
		first, ok := arr[0].(map[string]any)
		if !ok {
			t.Fatalf("first note is %T, want map[string]any", arr[0])
		}
		for _, key := range []string{"name", "folder", "modifiedDate"} {
			if _, has := first[key]; !has {
				t.Fatalf("first note missing %q key: %+v", key, first)
			}
		}
		if _, has := first["snippet"]; has {
			t.Fatalf("list-notes should not include snippet in list output: %+v", first)
		}
	}
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestEdgeCase_SpecialCharacters(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)
	ctx := testCtx(t)

	specialTitle := uniqueTitle(`"quotes" & 'apostrophes' <angle> $(not-a-command)`)
	body := `Body with "double" and 'single' quotes & <html>tags</html> $(echo nope)`

	// Create note with special characters
	createResult, err := adapter.Execute(ctx, makeRequest("create-note", "Notes", map[string]any{
		"title": specialTitle,
		"body":  body,
	}))
	requireOK(t, createResult, err)
	t.Logf("create output: %+v", createResult.Output)

	// Search for a substring of the special title
	searchResult, err := adapter.Execute(ctx, makeRequest("search-notes", "Notes", map[string]any{
		"query": "apostrophes",
	}))
	requireOK(t, searchResult, err)
	t.Logf("search output: %+v", searchResult.Output)

	// Verify we can find the note (search may return others too)
	arr := dataArray(t, searchResult)
	found := false
	for _, item := range arr {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		if strings.Contains(name, testNotePrefix) && strings.Contains(name, "apostrophes") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("special character note not found in search results")
	}
}

func TestEdgeCase_EmptySearch(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)
	ctx := testCtx(t)

	// Search for a UUID-like string that won't match anything
	nonsense := "kimbap-no-match-e7b3c9a4f2d1"
	result, err := adapter.Execute(ctx, makeRequest("search-notes", "Notes", map[string]any{
		"query": nonsense,
	}))
	requireOK(t, result, err)
	t.Logf("output: %+v", result.Output)

	arr := dataArray(t, result)
	if len(arr) != 0 {
		t.Fatalf("expected 0 results for nonsense query, got %d", len(arr))
	}
}

func TestEdgeCase_NonexistentFolder(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)
	ctx := testCtx(t)

	result, err := adapter.Execute(ctx, makeRequest("list-notes", "Notes", map[string]any{
		"folder": "KIMBAP_NONEXISTENT_FOLDER_12345",
	}))

	// The JXA script returns an empty array for a non-existent folder,
	// not an error. We accept either empty array (HTTP 200) or a graceful error.
	if err != nil {
		// Graceful error is acceptable — just should not panic/crash.
		t.Logf("list-notes for nonexistent folder returned error (acceptable): %v", err)
		if result != nil {
			t.Logf("output: %+v", result.Output)
		}
		return
	}

	requireOK(t, result, err)
	t.Logf("output: %+v", result.Output)

	arr := dataArray(t, result)
	if len(arr) != 0 {
		t.Fatalf("expected 0 notes in nonexistent folder, got %d", len(arr))
	}
}

func TestEdgeCase_ConcurrentCalls(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)

	const parallelism = 5
	var wg sync.WaitGroup
	wg.Add(parallelism)

	results := make([]*AdapterResult, parallelism)
	errs := make([]error, parallelism)

	for i := 0; i < parallelism; i++ {
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			results[idx], errs[idx] = adapter.Execute(ctx, makeRequest("list-folders", "Notes", nil))
		}(i)
	}

	wg.Wait()

	for i := 0; i < parallelism; i++ {
		if errs[i] != nil {
			t.Errorf("goroutine %d: error = %v", i, errs[i])
			continue
		}
		if results[i] == nil || results[i].HTTPStatus != 200 {
			status := 0
			if results[i] != nil {
				status = results[i].HTTPStatus
			}
			t.Errorf("goroutine %d: HTTPStatus = %d, want 200", i, status)
			continue
		}
		arr := dataArray(t, results[i])
		if len(arr) < 1 {
			t.Errorf("goroutine %d: got 0 folders, want >= 1", i)
		}
		t.Logf("goroutine %d: %d folders, duration %dms", i, len(arr), results[i].DurationMS)
	}
}

// ---------------------------------------------------------------------------
// Cross-app test
// ---------------------------------------------------------------------------

func TestEdgeCase_CrossAppNotes(t *testing.T) {
	adapter := newTestAdapter()
	skipIfTCCDenied(t, adapter)
	ctx := testCtx(t)

	// Notes: list folders
	notesResult, err := adapter.Execute(ctx, makeRequest("list-folders", "Notes", nil))
	requireOK(t, notesResult, err)
	notesFolders := dataArray(t, notesResult)
	t.Logf("Notes folders: %d", len(notesFolders))

	// Calendar: list calendars (skip if TCC not granted for Calendar)
	skipIfCalendarTCCDenied(t, adapter)
	calResult, err := adapter.Execute(ctx, makeRequest("list-calendars", "Calendar", nil))
	requireOK(t, calResult, err)
	calendars := dataArray(t, calResult)
	t.Logf("Calendars: %d", len(calendars))

	// Both should have returned valid results
	if len(notesFolders) < 1 {
		t.Fatalf("Notes should have at least 1 folder")
	}
	if len(calendars) < 1 {
		t.Fatalf("Calendar should have at least 1 calendar")
	}
}

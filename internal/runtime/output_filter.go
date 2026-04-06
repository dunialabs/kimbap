package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/pathutil"
)

// ApplyFilter applies FilterConfig transformations to an output map.
// Returns error only when ALL select paths miss for every item (API schema change).
// Partial misses are recorded in FilterMeta.PartialMiss, not returned as errors.
func ApplyFilter(output map[string]any, config *actions.FilterConfig) (map[string]any, actions.FilterMeta, error) {
	if config == nil {
		return output, actions.FilterMeta{}, nil
	}
	// Fast-path: empty config is a no-op (e.g. filter: {} in YAML)
	if len(config.Select) == 0 && len(config.Exclude) == 0 && config.MaxItems == 0 && !config.DropNulls {
		return output, actions.FilterMeta{}, nil
	}

	rawBytes, _ := json.Marshal(output)
	filtered, filterMeta, err := detectAndFilter(output, config)
	if err != nil {
		return output, filterMeta, err
	}

	filteredBytes, _ := json.Marshal(filtered)
	filterMeta.OriginalBytes = len(rawBytes)
	filterMeta.FilteredBytes = len(filteredBytes)
	filterMeta.Applied = true

	return filtered, filterMeta, nil
}

// detectAndFilter uses DetectPayloadRoot to find the data, applies filters,
// and reassembles the output preserving the wrapper key and _pagination.
func detectAndFilter(output map[string]any, config *actions.FilterConfig) (map[string]any, actions.FilterMeta, error) {
	// Handle raw text output — skip field selection
	if isRawOutput(output) {
		meta := actions.FilterMeta{Skipped: "raw_output"}
		result := dropNullsFromMap(output, config.DropNulls)
		return result, meta, nil
	}

	wrapperKey, payload := pathutil.DetectPayloadRoot(output)

	// Single object (no array wrapper found)
	if wrapperKey == "" {
		obj, ok := payload.(map[string]any)
		if !ok {
			return output, actions.FilterMeta{}, nil
		}
		filtered, meta, err := filterObject(obj, config)
		if err != nil {
			return output, meta, err
		}
		return filtered, meta, nil
	}

	// Array payload
	if arr, ok := payload.([]any); ok {
		filtered, meta, err := filterArray(arr, config)
		if err != nil {
			return output, meta, err
		}
		// Rebuild output preserving wrapper key and any other keys (_pagination etc)
		result := make(map[string]any, len(output))
		for k, v := range output {
			if k != wrapperKey {
				result[k] = v
			}
		}
		result[wrapperKey] = filtered
		return result, meta, nil
	}

	return output, actions.FilterMeta{}, nil
}

// filterArray applies all filter operations to an array payload.
func filterArray(items []any, config *actions.FilterConfig) ([]any, actions.FilterMeta, error) {
	meta := actions.FilterMeta{}
	result := items

	// 1. Limit items first
	if config.MaxItems > 0 && len(result) > config.MaxItems {
		meta.ItemsTruncatedFrom = len(result)
		result = result[:config.MaxItems]
	}

	// 2. Select (whitelist)
	if len(config.Select) > 0 {
		projected, partialMiss, err := projectArray(result, config.Select)
		if err != nil {
			// Return original untruncated items with empty meta so caller gets a clean fallback.
			return items, actions.FilterMeta{}, err
		}
		meta.PartialMiss = partialMiss
		meta.FieldsSelected = len(config.Select)
		result = projected
	}

	// 3. Exclude (blacklist on remaining fields)
	if len(config.Exclude) > 0 {
		result = excludeArray(result, config.Exclude)
		meta.FieldsExcluded = len(config.Exclude)
	}

	// 4. Drop nulls
	if config.DropNulls {
		result = dropNullsFromArray(result)
	}

	return result, meta, nil
}

// filterObject applies filter operations to a single map object.
func filterObject(obj map[string]any, config *actions.FilterConfig) (map[string]any, actions.FilterMeta, error) {
	meta := actions.FilterMeta{}
	result := obj

	// Select
	if len(config.Select) > 0 {
		projected, missing := projectItem(result, config.Select)
		if len(projected) == 0 && len(missing) == len(config.Select) {
			return obj, meta, fmt.Errorf("all select paths missing from object: %v", missing)
		}
		meta.PartialMiss = missing
		meta.FieldsSelected = len(config.Select)
		result = projected
	}

	// Exclude
	if len(config.Exclude) > 0 {
		result = excludeItem(result, config.Exclude)
		meta.FieldsExcluded = len(config.Exclude)
	}

	// Drop nulls
	if config.DropNulls {
		result = dropNullsFromItem(result)
	}

	return result, meta, nil
}

// projectArray applies select to each item in the array.
// Returns error only if ALL paths miss for EVERY item.
func projectArray(items []any, selectMap map[string]string) ([]any, []string, error) {
	if len(items) == 0 {
		return items, nil, nil
	}

	result := make([]any, 0, len(items))
	allMissing := make(map[string]bool)
	foundAny := false
	hadMapItem := false

	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			result = append(result, item)
			continue
		}
		hadMapItem = true
		projected, missing := projectItem(m, selectMap)
		result = append(result, projected)
		if len(projected) > 0 {
			foundAny = true
		}
		for _, p := range missing {
			allMissing[p] = true
		}
	}

	// Error only when we had map items but none produced any projected output
	// (primitive items pass through and do not count toward the "all paths missing" error)
	if hadMapItem && !foundAny {
		missing := make([]string, 0, len(allMissing))
		for p := range allMissing {
			missing = append(missing, p)
		}
		return items, nil, fmt.Errorf("all select paths missing from all items: %v", missing)
	}

	// Partial miss: source paths that never appeared in any item
	var partialMiss []string
	for _, sourcePath := range selectMap {
		found := false
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if _, ok2 := pathutil.ExtractByPath(m, sourcePath); ok2 {
				found = true
				break
			}
		}
		if !found {
			partialMiss = append(partialMiss, sourcePath)
		}
	}

	return result, partialMiss, nil
}

// projectItem applies a select map to a single item map.
// outputKey: sourcePath — extracts sourcePath from item, stores as outputKey.
func projectItem(item map[string]any, selectMap map[string]string) (map[string]any, []string) {
	result := make(map[string]any, len(selectMap))
	var missing []string
	for outputKey, sourcePath := range selectMap {
		val, ok := pathutil.ExtractByPath(item, sourcePath)
		if ok {
			result[outputKey] = val
		} else {
			missing = append(missing, sourcePath)
		}
	}
	return result, missing
}

// excludeArray applies excludeItem to every array element.
func excludeArray(items []any, excludeList []string) []any {
	result := make([]any, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			result[i] = item
			continue
		}
		result[i] = excludeItem(m, excludeList)
	}
	return result
}

// excludeItem returns a new map without the specified fields.
func excludeItem(item map[string]any, excludeList []string) map[string]any {
	excluded := make(map[string]bool, len(excludeList))
	for _, k := range excludeList {
		excluded[k] = true
	}
	result := make(map[string]any, len(item))
	for k, v := range item {
		if !excluded[k] {
			result[k] = v
		}
	}
	return result
}

// dropNullsFromArray applies dropNullsFromItem to every array element.
func dropNullsFromArray(items []any) []any {
	result := make([]any, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			result[i] = item // preserve non-map elements (including null array slots)
			continue
		}
		result[i] = dropNullsFromItem(m)
	}
	return result
}

// dropNullsFromItem recursively removes nil-valued fields from a map.
// Array elements are NOT touched — only object fields.
func dropNullsFromItem(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if v == nil {
			continue // drop null field
		}
		// Recurse into nested maps
		if nested, ok := v.(map[string]any); ok {
			result[k] = dropNullsFromItem(nested)
		} else {
			result[k] = v // arrays pass through as-is
		}
	}
	return result
}

// dropNullsFromMap applies dropNulls to a top-level map (used for raw output wrapper).
func dropNullsFromMap(m map[string]any, apply bool) map[string]any {
	if !apply {
		return m
	}
	return dropNullsFromItem(m)
}

// isRawOutput returns true when the output map contains exactly one key ("raw").
// The Command and AppleScript adapters produce {"raw": "..."} when stdout is not valid JSON.
func isRawOutput(output map[string]any) bool {
	if len(output) != 1 {
		return false
	}
	_, hasRaw := output["raw"]
	return hasRaw
}

// ── Budget enforcement (T7) ───────────────────────────────────────────────

// BudgetMeta records what happened during budget enforcement.
type BudgetMeta struct {
	Applied       bool
	Limit         int
	OriginalBytes int // serialized JSON bytes before budget enforcement
	ResultBytes   int // serialized JSON bytes after budget enforcement
}

// ApplyBudget enforces a maximum serialized byte count on output.
// For arrays: progressively removes items from the end until under budget.
// For objects with long strings: rune-safely truncates the longest values.
// Always produces valid JSON. Note: limit is measured in JSON bytes, not characters.
func ApplyBudget(output map[string]any, maxBytes int) (map[string]any, BudgetMeta) {
	if maxBytes <= 0 {
		return output, BudgetMeta{}
	}

	raw, _ := json.Marshal(output)
	meta := BudgetMeta{
		Applied:       false,
		Limit:         maxBytes,
		OriginalBytes: len(raw),
	}

	if len(raw) <= maxBytes {
		meta.ResultBytes = len(raw)
		return output, meta
	}

	meta.Applied = true

	// Determine the best starting point for string-truncation fallback.
	// Start from the original output; if array trimming reduces it, update the base.
	base := output

	// Try to trim arrays first
	wrapperKey, payload := pathutil.DetectPayloadRoot(output)
	if arr, ok := payload.([]any); ok && wrapperKey != "" {
		// Remove items from end until under budget
		trimmed := make([]any, len(arr))
		copy(trimmed, arr)
		for {
			candidate := make(map[string]any, len(output))
			for k, v := range output {
				if k == wrapperKey {
					candidate[k] = trimmed
				} else {
					candidate[k] = v
				}
			}
			encoded, _ := json.Marshal(candidate)
			if len(encoded) <= maxBytes {
				meta.ResultBytes = len(encoded)
				return candidate, meta
			}
			if len(trimmed) == 0 {
				// Even empty array is over budget — continue to string truncation
				// but use this candidate (empty array) as our base, not the original.
				base = candidate
				break
			}
			trimmed = trimmed[:len(trimmed)-1]
		}
	}

	// Truncate long strings in the current base (which may have an empty array).
	// Each iteration lowers the per-string threshold to guarantee monotonic shrinkage.
	result := base
	threshold := maxBytes / 2
	if threshold < 10 {
		threshold = 10
	}
	for range 10 {
		prevEncoded, _ := json.Marshal(result)
		prevSize := len(prevEncoded)
		result = truncateLongStrings(result, threshold)
		encoded, _ := json.Marshal(result)
		if len(encoded) <= maxBytes {
			meta.ResultBytes = len(encoded)
			return result, meta
		}
		// If no progress was made, lower threshold aggressively
		if len(encoded) >= prevSize {
			threshold = threshold / 2
			if threshold < 10 {
				break // can't shrink further
			}
		}
	}
	// Best effort — record final size even if still over budget
	encoded, _ := json.Marshal(result)
	meta.ResultBytes = len(encoded)
	return result, meta
}

// truncateLongStrings recursively truncates string values exceeding threshold runes.
// Uses rune-aware truncation to avoid splitting multi-byte UTF-8 characters.
func truncateLongStrings(m map[string]any, threshold int) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			runes := []rune(val)
			// Only truncate if the string is longer than threshold AND truncation
			// actually reduces length (cutoff + 3 for '...' must be < original).
			if len(runes) > threshold && threshold+3 < len(runes) {
				result[k] = string(runes[:threshold]) + "..."
			} else {
				result[k] = v
			}
		case map[string]any:
			result[k] = truncateLongStrings(val, threshold)
		default:
			result[k] = v
		}
	}
	return result
}

// ── Compact text templates (T11) ─────────────────────────────────────────

// ApplyCompactTemplate renders structured array data into a text summary.
// Applied AFTER filter (select/max_items), so template sees already-filtered fields.
// Only applies to array payloads — single objects are a no-op.
// Returns {"summary": "...", "_compact": true, "_original_items": N}.
func ApplyCompactTemplate(output map[string]any, tmpl *actions.CompactTemplate) (map[string]any, error) {
	if tmpl == nil || tmpl.Item == "" {
		return output, nil
	}

	wrapperKey, payload := pathutil.DetectPayloadRoot(output)
	arr, ok := payload.([]any)
	if !ok || wrapperKey == "" {
		return output, nil // single object or unrecognized — no-op
	}

	itemTmpl, err := compileTemplate("item", tmpl.Item)
	if err != nil {
		return output, fmt.Errorf("compact item template: %w", err)
	}

	var lines []string

	if tmpl.Header != "" {
		headerTmpl, err := compileTemplate("header", tmpl.Header)
		if err != nil {
			return output, fmt.Errorf("compact header template: %w", err)
		}
		headerData := map[string]any{"Total": len(arr), "Count": len(arr)}
		hLine, err := renderTemplate(headerTmpl, headerData)
		if err != nil {
			return output, fmt.Errorf("compact header render: %w", err)
		}
		lines = append(lines, hLine)
	}

	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		line, err := renderTemplate(itemTmpl, m)
		if err != nil {
			return output, fmt.Errorf("compact item render: %w", err)
		}
		lines = append(lines, line)
	}

	if tmpl.Footer != "" {
		footerTmpl, err := compileTemplate("footer", tmpl.Footer)
		if err != nil {
			return output, fmt.Errorf("compact footer template: %w", err)
		}
		footerData := map[string]any{"Count": len(arr), "Total": len(arr), "Remaining": 0}
		fLine, err := renderTemplate(footerTmpl, footerData)
		if err != nil {
			return output, fmt.Errorf("compact footer render: %w", err)
		}
		lines = append(lines, fLine)
	}

	summary := joinLines(lines)
	return map[string]any{
		"summary":         summary,
		"_compact":        true,
		"_original_items": len(arr),
	}, nil
}

// ── Template helpers ──────────────────────────────────────────────────────

func compileTemplate(name, src string) (*template.Template, error) {
	return template.New(name).Option("missingkey=zero").Parse(src)
}

func renderTemplate(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// coerceBudgetInt extracts a budget integer from any numeric type that input
// parsing might produce:
//   - int: Go test literals and direct API callers
//   - int64: CLI parseScalar path (strconv.ParseInt returns int64)
//   - float64: standard json.Unmarshal into map[string]any
//   - json.Number: dec.UseNumber() path before coerceJSONNumbers is called
//
// Returns 0 if the value is absent, non-numeric, or negative.
func coerceBudgetInt(v any) int {
	var n int
	switch val := v.(type) {
	case int:
		n = val
	case int64:
		n = int(val)
	case float64:
		n = int(val)
	case interface{ Int64() (int64, error) }: // handles json.Number
		if i, err := val.Int64(); err == nil {
			n = int(i)
		}
	default:
		return 0
	}
	if n < 0 {
		return 0
	}
	return n
}

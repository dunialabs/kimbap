package runtime

import (
	"encoding/json"
	"fmt"

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
			return items, meta, err
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

	// Track which paths ever succeeded across all items
	everFound := make(map[string]bool)

	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			result = append(result, item)
			continue
		}
		projected, missing := projectItem(m, selectMap)
		result = append(result, projected)
		if len(projected) > 0 {
			foundAny = true
		}
		for _, p := range missing {
			allMissing[p] = true
		}
		for k := range projected {
			_ = k
			everFound[k] = true
		}
	}

	// If no item produced any output at all → error
	if !foundAny && len(items) > 0 {
		missing := make([]string, 0, len(allMissing))
		for p := range allMissing {
			missing = append(missing, p)
		}
		return items, nil, fmt.Errorf("all select paths missing from all items: %v", missing)
	}

	// Partial miss: paths that never appeared in any item
	partialMiss := make([]string, 0)
	for outputKey, sourcePath := range selectMap {
		_ = outputKey
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

// isRawOutput returns true when the output map contains only a "raw" key with a string value.
func isRawOutput(output map[string]any) bool {
	if len(output) != 1 {
		return false
	}
	_, hasRaw := output["raw"]
	return hasRaw
}

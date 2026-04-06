package pathutil

import (
	"sort"
	"strconv"
	"strings"
)

// ExtractByPath traverses a nested structure (maps and slices) following a dot-delimited path.
// Example: ExtractByPath(data, "owner.login") will return the value at data["owner"]["login"].
// Returns (value, true) when found, or (nil, false) when not found.
func ExtractByPath(value interface{}, path string) (interface{}, bool) {
	if path == "" {
		return value, true
	}
	parts := strings.Split(strings.TrimPrefix(path, "."), ".")
	current := value
	for _, part := range parts {
		if part == "" {
			continue
		}
		next, ok := ExtractSegment(current, part)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

// ExtractSegment extracts a single segment from a value.
// It supports keys like "name" and array indices like "items[0]".
// Returns (value, true) if found, otherwise (nil, false).
func ExtractSegment(value interface{}, segment string) (interface{}, bool) {
	key := segment
	index := -1
	if open := strings.Index(segment, "["); open >= 0 && strings.HasSuffix(segment, "]") {
		key = segment[:open]
		idxStr := segment[open+1 : len(segment)-1]
		// tolerate invalid or negative indices by returning not found
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 {
			return nil, false
		}
		index = idx
	}

	current := value
	if key != "" {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		next, ok := m[key]
		if !ok {
			return nil, false
		}
		current = next
	}

	if index >= 0 {
		arr, ok := current.([]interface{})
		if !ok {
			return nil, false
		}
		if index >= len(arr) {
			return nil, false
		}
		current = arr[index]
	}

	return current, true
}

// DetectPayloadRoot inspects a payload (as map[string]interface{}) to determine the root wrapper key
// and the corresponding payload. It follows the priority rules described in the task:
// 1) "items" if present and is an array
// 2) "result" if present and is an array
// 3) "data" if present and is an array
// 4) first key whose value is []interface{} (deterministic order by key name)
// 5) if no wrapper found, return empty key and the original output as payload
func DetectPayloadRoot(output map[string]interface{}) (string, interface{}) {
	// 1-3: explicit wrappers
	if v, ok := output["items"]; ok {
		if arr, ok2 := v.([]interface{}); ok2 {
			return "items", arr
		}
	}
	if v, ok := output["result"]; ok {
		if arr, ok2 := v.([]interface{}); ok2 {
			return "result", arr
		}
	}
	if v, ok := output["data"]; ok {
		if arr, ok2 := v.([]interface{}); ok2 {
			return "data", arr
		}
	}

	// 4: first key whose value is []interface{} deterministically by key name
	var arrayKeys []string
	for k, v := range output {
		if _, ok := v.([]interface{}); ok {
			arrayKeys = append(arrayKeys, k)
		}
	}
	if len(arrayKeys) > 0 {
		sort.Strings(arrayKeys)
		first := arrayKeys[0]
		return first, output[first]
	}

	// 5: no wrapper; return whole map as payload
	return "", output
}


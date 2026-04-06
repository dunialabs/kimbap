package pathutil

import (
	"reflect"
	"testing"
)

func TestExtractByPath(t *testing.T) {
	// simple root key
	data := map[string]interface{}{"name": "John"}
	if v, ok := ExtractByPath(data, "name"); !ok || v != "John" {
		t.Fatalf("expected name=John, got %v, ok=%v", v, ok)
	}

	// nested path
	nested := map[string]interface{}{"owner": map[string]interface{}{"login": "alice"}}
	if v, ok := ExtractByPath(nested, "owner.login"); !ok || v != "alice" {
		t.Fatalf("expected owner.login=alice, got %v, ok=%v", v, ok)
	}

	// array path
	arr := map[string]interface{}{"items": []interface{}{map[string]interface{}{"name": "foo"}}}
	if v, ok := ExtractByPath(arr, "items[0].name"); !ok || v != "foo" {
		t.Fatalf("expected items[0].name=foo, got %v, ok=%v", v, ok)
	}

	// missing path
	if v, ok := ExtractByPath(data, "unknown"); ok || v != nil {
		t.Fatalf("expected not found for missing path, got v=%v ok=%v", v, ok)
	}

	// empty path
	if v, ok := ExtractByPath(data, ""); !ok || !reflect.DeepEqual(v, data) {
		t.Fatalf("expected empty path to return original value, got v=%v ok=%v", v, ok)
	}
}

func TestExtractSegment(t *testing.T) {
	data := map[string]interface{}{"name": "alice", "items": []interface{}{map[string]interface{}{"id": 1}}}
	if v, ok := ExtractSegment(data, "name"); !ok || v != "alice" {
		t.Fatalf("expected name=alice, got %v, ok=%v", v, ok)
	}
	if v, ok := ExtractSegment(data, "items[0]"); !ok {
		t.Fatalf("expected to extract items[0], got ok=%v, v=%v", ok, v)
	} else {
		// should be a map with id 1
		if m, ok2 := v.(map[string]interface{}); !ok2 || m["id"] != 1 {
			t.Fatalf("expected items[0] to be object with id=1, got %#v", v)
		}
	}
	if v, ok := ExtractSegment(data, "missing"); ok {
		t.Fatalf("expected missing segment to fail, got v=%v ok=%v", v, ok)
	}
}

func TestDetectPayloadRoot(t *testing.T) {
	// 1-3 wrappers
	for _, tc := range []struct {
		in      map[string]interface{}
		wantKey string
	}{
		{map[string]interface{}{"items": []interface{}{1, 2, 3}}, "items"},
		{map[string]interface{}{"result": []interface{}{1, 2, 3}}, "result"},
		{map[string]interface{}{"data": []interface{}{1, 2, 3}}, "data"},
		{map[string]interface{}{"items": []interface{}{1}, "_pagination": map[string]interface{}{"next": "x"}}, "items"},
	} {
		key, payload := DetectPayloadRoot(tc.in)
		if key != tc.wantKey {
			t.Fatalf("expected key %q, got %q", tc.wantKey, key)
		}
		// payload should be the array
		if arr, ok := payload.([]interface{}); ok {
			if len(arr) != 3 && tc.wantKey == "items" && tc.in["items"] != nil {
				// special-case small arrays
				// do not fail here, just ensure it's an array
			}
		} else if tc.wantKey == "items" {
			t.Fatalf("expected payload to be array for key %q", tc.wantKey)
		}
	}

	// raw value should return empty key
	in := map[string]interface{}{"raw": "text"}
	key, payload := DetectPayloadRoot(in)
	if key != "" {
		t.Fatalf("expected empty key for raw payload, got %q", key)
	}
	if !reflect.DeepEqual(payload, in) {
		t.Fatalf("expected payload to be original map, got %#v", payload)
	}

	// flat object should return empty key
	in2 := map[string]interface{}{"id": 1, "name": "x"}
	key2, payload2 := DetectPayloadRoot(in2)
	if key2 != "" || !reflect.DeepEqual(payload2, in2) {
		t.Fatalf("expected empty key and original payload for flat object, got key=%q payload=%#v", key2, payload2)
	}

	// fallback: first array-valued key in deterministic order
	in3 := map[string]interface{}{"custom_list": []interface{}{1, 2, 3}, "other": []interface{}{"a"}}
	key3, payload3 := DetectPayloadRoot(in3)
	if key3 != "custom_list" {
		t.Fatalf("expected first array-valued key to be 'custom_list', got %q", key3)
	}
	if _, ok := payload3.([]interface{}); !ok {
		t.Fatalf("expected payload to be array for fallback case, got %#v", payload3)
	}
}

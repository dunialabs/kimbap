package user

import "testing"

func TestParseJSONFieldNullReturnsEmptyMap(t *testing.T) {
	result, err := parseJSONField("null", "launch configs")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got len=%d", len(result))
	}
}

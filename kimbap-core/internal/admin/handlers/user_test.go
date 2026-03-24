package handlers

import (
	"strings"
	"testing"
)

func TestToJSONStringTreatsNullStringAsFallback(t *testing.T) {
	got := toJSONString("null", "{}")
	if got != "{}" {
		t.Fatalf("expected fallback {}, got %s", got)
	}

	got = toJSONString("  null  ", "{}")
	if got != "{}" {
		t.Fatalf("expected fallback {} for trimmed null, got %s", got)
	}
}

func TestNormalizePermissionsJSONRejectsInvalidInput(t *testing.T) {
	_, _, err := normalizePermissionsJSON(nil)
	if err == nil {
		t.Fatal("expected invalid permissions error for nil input")
	}

	_, _, err = normalizePermissionsJSON("{")
	if err == nil {
		t.Fatal("expected invalid permissions error")
	}
}

func TestNormalizePermissionsJSONInitializesNestedMaps(t *testing.T) {
	normalized, perms, err := normalizePermissionsJSON(map[string]any{
		"server-a": map[string]any{"enabled": true},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	entry, ok := perms["server-a"]
	if !ok {
		t.Fatal("expected server-a entry")
	}
	if entry.Tools == nil || entry.Resources == nil || entry.Prompts == nil {
		t.Fatal("expected nested maps to be initialized")
	}
	if strings.Contains(normalized, `"tools":null`) || strings.Contains(normalized, `"resources":null`) || strings.Contains(normalized, `"prompts":null`) {
		t.Fatal("expected normalized JSON to avoid null nested maps")
	}

}

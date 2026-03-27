package main

import "testing"

func TestParseJSONInputInline(t *testing.T) {
	result, err := parseJSONInput(`{"name": "test", "count": 42}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["name"] != "test" {
		t.Fatalf("expected name=test, got %v", result["name"])
	}
	count := result["count"]
	if _, ok := count.(int64); !ok {
		t.Fatalf("expected count to be int64, got %T (%v)", count, count)
	}
}

func TestParseJSONInputFloat(t *testing.T) {
	result, err := parseJSONInput(`{"price": 19.99}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	price := result["price"]
	if _, ok := price.(float64); !ok {
		t.Fatalf("expected price to be float64, got %T (%v)", price, price)
	}
}

func TestParseJSONInputNestedObjects(t *testing.T) {
	result, err := parseJSONInput(`{"outer": {"inner": 10, "nested_float": 3.14}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outer, ok := result["outer"].(map[string]any)
	if !ok {
		t.Fatalf("expected outer to be map, got %T", result["outer"])
	}
	if _, ok := outer["inner"].(int64); !ok {
		t.Fatalf("expected nested integer to be int64, got %T", outer["inner"])
	}
	if _, ok := outer["nested_float"].(float64); !ok {
		t.Fatalf("expected nested float to be float64, got %T", outer["nested_float"])
	}
}

func TestParseJSONInputArray(t *testing.T) {
	result, err := parseJSONInput(`{"tags": ["a", "b"], "ids": [1, 2, 3]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids, ok := result["ids"].([]any)
	if !ok {
		t.Fatalf("expected ids to be slice, got %T", result["ids"])
	}
	for i, id := range ids {
		if _, ok := id.(int64); !ok {
			t.Fatalf("expected ids[%d] to be int64, got %T", i, id)
		}
	}
}

func TestParseJSONInputInvalid(t *testing.T) {
	_, err := parseJSONInput(`not json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMergeInputMaps(t *testing.T) {
	base := map[string]any{"a": "1", "b": "2"}
	override := map[string]any{"b": "override", "c": "3"}
	merged := mergeInputMaps(base, override)
	if merged["a"] != "1" {
		t.Fatalf("expected a=1, got %v", merged["a"])
	}
	if merged["b"] != "override" {
		t.Fatalf("expected b=override, got %v", merged["b"])
	}
	if merged["c"] != "3" {
		t.Fatalf("expected c=3, got %v", merged["c"])
	}
}

func TestMergeInputMapsNilBase(t *testing.T) {
	merged := mergeInputMaps(nil, map[string]any{"a": "1"})
	if merged["a"] != "1" {
		t.Fatalf("expected a=1, got %v", merged["a"])
	}
}

func TestCoerceJSONNumbersLargeInt(t *testing.T) {
	input := `{"big": 9007199254740993}`
	result, err := parseJSONInput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	big := result["big"]
	if _, ok := big.(int64); !ok {
		t.Fatalf("expected big int to be int64, got %T (%v)", big, big)
	}
}

func TestCoerceJSONNumbersBoolString(t *testing.T) {
	input := `{"flag": true, "label": "hello"}`
	result, err := parseJSONInput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["flag"] != true {
		t.Fatalf("expected flag=true, got %v (%T)", result["flag"], result["flag"])
	}
	if result["label"] != "hello" {
		t.Fatalf("expected label=hello, got %v", result["label"])
	}
}

func TestSplitCallInvocationArgs_HelpWithGlobalConfigBeforeAction(t *testing.T) {
	action, input, showHelp, err := splitCallInvocationArgs([]string{"--config", "/tmp/config.yaml", "--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "" {
		t.Fatalf("expected empty action for help-only invocation, got %q", action)
	}
	if len(input) != 0 {
		t.Fatalf("expected no input tokens, got %+v", input)
	}
	if !showHelp {
		t.Fatal("expected showHelp=true")
	}
}

func TestSplitCallInvocationArgs_GlobalFlagsAroundAction(t *testing.T) {
	action, input, showHelp, err := splitCallInvocationArgs([]string{"--format", "json", "slack.list-channels", "--dry-run", "--limit", "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "slack.list-channels" {
		t.Fatalf("expected action slack.list-channels, got %q", action)
	}
	if showHelp {
		t.Fatal("expected showHelp=false")
	}
	if len(input) != 2 || input[0] != "--limit" || input[1] != "1" {
		t.Fatalf("unexpected input tokens: %+v", input)
	}
}

func TestSplitCallInvocationArgs_RejectsInputBeforeAction(t *testing.T) {
	_, _, _, err := splitCallInvocationArgs([]string{"--limit", "1", "slack.list-channels"})
	if err == nil {
		t.Fatal("expected error when input flag appears before action")
	}
}

func TestParseScalarNumericOneIsInteger(t *testing.T) {
	v := parseScalar("1")
	if _, ok := v.(int64); !ok {
		t.Fatalf("expected int64 for scalar '1', got %T (%v)", v, v)
	}
}

func TestSplitCallInvocationArgs_StringFlagMissingValue(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
	}{
		{"format at end of args", []string{"slack.list-channels", "--format"}},
		{"format followed by another flag", []string{"slack.list-channels", "--format", "--dry-run"}},
		{"json at end of args", []string{"slack.list-channels", "--json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := splitCallInvocationArgs(tt.args)
			if err == nil {
				t.Fatalf("expected error for %v, got nil", tt.args)
			}
		})
	}
}

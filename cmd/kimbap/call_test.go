package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
)

func resetOptsForTest(t *testing.T) {
	t.Helper()
	prev := opts
	t.Cleanup(func() { opts = prev })
	opts = cliOptions{}
}

func captureStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = w

	runErr := fn()
	_ = w.Close()
	os.Stderr = old

	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("read captured stderr: %v", readErr)
	}
	return string(out), runErr
}

func TestCallOutputTextMode_Success(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "text"

	result := actions.ExecutionResult{
		Status: actions.StatusSuccess,
		Output: map[string]any{"message": "all good"},
	}

	output, err := captureStdout(t, func() error { return printCallResult(result) })
	if err != nil {
		t.Fatalf("printCallResult returned error: %v", err)
	}

	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "{") {
		t.Fatalf("expected human-readable text output, got raw json blob: %q", output)
	}
	if !strings.HasPrefix(trimmed, "✓") {
		t.Fatalf("expected output to start with checkmark, got %q", output)
	}
	if !strings.Contains(output, "all good") {
		t.Fatalf("expected output to contain meaningful content, got %q", output)
	}
}

func TestCallOutputTextMode_SuccessNilOutput(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "text"

	for _, name := range []string{"nil output", "empty output"} {
		t.Run(name, func(t *testing.T) {
			var out map[string]any
			if name == "empty output" {
				out = map[string]any{}
			}
			result := actions.ExecutionResult{Status: actions.StatusSuccess, Output: out}
			output, err := captureStdout(t, func() error { return printCallResult(result) })
			if err != nil {
				t.Fatalf("printCallResult returned error: %v", err)
			}
			if !strings.Contains(output, "✓ Done") {
				t.Fatalf("expected '✓ Done' for %s, got %q", name, output)
			}
			if strings.Contains(output, "{}") {
				t.Fatalf("expected no raw '{}' for %s, got %q", name, output)
			}
		})
	}
}

func TestCallOutputTextMode_Error(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "text"

	result := actions.ExecutionResult{
		Status: actions.StatusError,
		Error:  &actions.ExecutionError{Message: "permission denied"},
	}

	stderr, err := captureStderr(t, func() error { return printCallResult(result) })
	if err != nil {
		t.Fatalf("printCallResult returned error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("printCallResult should not print errors in text mode (Cobra handles it), got %q", stderr)
	}
}

func TestCallOutputTextMode_ApprovalRequiredPrintsHint(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "text"

	result := actions.ExecutionResult{
		Status: actions.StatusApprovalRequired,
		Error:  &actions.ExecutionError{Message: "approval required"},
		Meta: map[string]any{
			"approval_request_id": "apr_123",
		},
	}

	stderr, err := captureStderr(t, func() error { return printCallResult(result) })
	if err != nil {
		t.Fatalf("printCallResult returned error: %v", err)
	}
	if !strings.Contains(stderr, "Run: kimbap approve accept apr_123") {
		t.Fatalf("expected approval next-step hint, got %q", stderr)
	}
}

func TestCallHelpContainsReservedFlags(t *testing.T) {
	long := newCallCommand().Long
	if !strings.Contains(long, "Globally consumed flags") {
		t.Fatalf("expected call help to include globally consumed flags section, got %q", long)
	}
	if !strings.Contains(long, "--idempotency-key") {
		t.Fatalf("expected call help to include --idempotency-key, got %q", long)
	}
	if !strings.Contains(long, "--format") {
		t.Fatalf("expected call help to mention --format behavior, got %q", long)
	}
}

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
	// After the action name, --format is passed through as an action arg.
	// Only flags that remain global after the action name should still error without a value.
	tests := []struct {
		name string
		args []string
	}{
		{"json at end of args", []string{"slack.list-channels", "--json"}},
		{"idempotency key at end of args", []string{"slack.list-channels", "--idempotency-key"}},
		{"idempotency key followed by short flag", []string{"slack.list-channels", "--idempotency-key", "-h"}},
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

func TestSplitCallInvocationArgs_FormatPassedThroughAfterAction(t *testing.T) {
	resetOptsForTest(t)

	// --format after action name must be passed as action input, not consumed as global flag.
	_, input, _, err := splitCallInvocationArgs([]string{"mermaid.render-url", "--diagram_id", "x", "--format", "png"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.format != "" {
		t.Fatalf("global format should not be set when --format appears after action; got %q", opts.format)
	}
	found := false
	for i, tok := range input {
		if tok == "--format" && i+1 < len(input) && input[i+1] == "png" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected --format png in action input tokens, got %v", input)
	}
}

func TestSplitCallInvocationArgs_NoSplashOptionalFalse(t *testing.T) {
	resetOptsForTest(t)

	action, input, showHelp, err := splitCallInvocationArgs([]string{"slack.list-channels", "--no-splash", "false", "--limit", "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if showHelp {
		t.Fatal("expected showHelp=false")
	}
	if action != "slack.list-channels" {
		t.Fatalf("expected action slack.list-channels, got %q", action)
	}
	if opts.noSplash {
		t.Fatal("expected --no-splash false to keep noSplash=false")
	}
	if len(input) != 2 || input[0] != "--limit" || input[1] != "1" {
		t.Fatalf("unexpected input tokens: %+v", input)
	}
}

func TestPrescanCallSplashFlags_NoSplashOptionalFalse(t *testing.T) {
	resetOptsForTest(t)

	prescanCallSplashFlags([]string{"slack.list-channels", "--no-splash", "false", "--format", "json"})

	if opts.noSplash {
		t.Fatal("expected noSplash=false when --no-splash false is provided")
	}
	if opts.format != "json" {
		t.Fatalf("expected format=json, got %q", opts.format)
	}
}

func TestPrescanCallSplashFlags_SplashColorParsed(t *testing.T) {
	resetOptsForTest(t)

	prescanCallSplashFlags([]string{"slack.list-channels", "--splash-color", "ansi256"})

	if opts.splashColor != "ansi256" {
		t.Fatalf("expected splashColor=ansi256, got %q", opts.splashColor)
	}
}

func TestPrescanRawSplashFlags_NoSplashOptionalFalse(t *testing.T) {
	resetOptsForTest(t)
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	os.Args = []string{"kimbap", "call", "slack.list-channels", "--no-splash", "false", "--format", "json"}
	prescanRawSplashFlags()

	if opts.noSplash {
		t.Fatal("expected noSplash=false when --no-splash false is provided")
	}
	if opts.format != "json" {
		t.Fatalf("expected format=json, got %q", opts.format)
	}
}

func TestPrescanRawSplashFlags_FormatMissingValueIgnored(t *testing.T) {
	resetOptsForTest(t)
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	opts.format = "text"
	os.Args = []string{"kimbap", "call", "slack.list-channels", "--format", "--dry-run"}
	prescanRawSplashFlags()

	if opts.format != "text" {
		t.Fatalf("expected format to remain unchanged when --format lacks value, got %q", opts.format)
	}
}

func TestPrescanRawSplashFlags_SplashColorParsed(t *testing.T) {
	resetOptsForTest(t)
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	os.Args = []string{"kimbap", "call", "slack.list-channels", "--splash-color=none"}
	prescanRawSplashFlags()

	if opts.splashColor != "none" {
		t.Fatalf("expected splashColor=none, got %q", opts.splashColor)
	}
}

func TestPrescanCallSplashFlags_FormatShortFlagValueIgnored(t *testing.T) {
	resetOptsForTest(t)

	opts.format = "text"
	prescanCallSplashFlags([]string{"slack.list-channels", "--format", "-h"})

	if opts.format != "text" {
		t.Fatalf("expected format to remain unchanged when --format is followed by short flag, got %q", opts.format)
	}
}

func TestPrescanRawSplashFlags_FormatShortFlagValueIgnored(t *testing.T) {
	resetOptsForTest(t)
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	opts.format = "text"
	os.Args = []string{"kimbap", "call", "slack.list-channels", "--format", "-h"}
	prescanRawSplashFlags()

	if opts.format != "text" {
		t.Fatalf("expected format to remain unchanged when --format is followed by short flag, got %q", opts.format)
	}
}

func TestSplitCallInvocationArgs_ParsesIdempotencyKeyFlag(t *testing.T) {
	resetOptsForTest(t)

	action, input, showHelp, err := splitCallInvocationArgs([]string{"slack.list-channels", "--idempotency-key", "idem-123", "--limit", "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if showHelp {
		t.Fatal("expected showHelp=false")
	}
	if action != "slack.list-channels" {
		t.Fatalf("expected action slack.list-channels, got %q", action)
	}
	if opts.idempotencyKey != "idem-123" {
		t.Fatalf("expected idempotency key idem-123, got %q", opts.idempotencyKey)
	}
	if len(input) != 2 || input[0] != "--limit" || input[1] != "1" {
		t.Fatalf("unexpected input tokens: %+v", input)
	}
}

func TestSplitCallInvocationArgs_JsonStdinDash(t *testing.T) {
	resetOptsForTest(t)

	action, input, showHelp, err := splitCallInvocationArgs([]string{"svc.action", "--json", "-"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if showHelp {
		t.Fatal("expected showHelp=false")
	}
	if action != "svc.action" {
		t.Fatalf("expected action svc.action, got %q", action)
	}
	if opts.jsonInput != "-" {
		t.Fatalf("expected json input '-', got %q", opts.jsonInput)
	}
	if len(input) != 0 {
		t.Fatalf("unexpected remaining input tokens: %+v", input)
	}
}

func TestBuildDryRunPreview_IdempotencyCheck(t *testing.T) {
	cfg := config.DefaultConfig()

	nonIdempotentReq := actions.ExecutionRequest{
		RequestID:      "req-1",
		IdempotencyKey: "",
		Action: actions.ActionDefinition{
			Name:       "svc.create",
			Idempotent: false,
		},
		Input: map[string]any{},
	}
	preview := buildDryRunPreview(cfg, nonIdempotentReq)
	if idemValid, ok := preview["idempotency_valid"].(bool); !ok || idemValid {
		t.Fatalf("expected idempotency_valid=false for non-idempotent action without key, got %v", preview["idempotency_valid"])
	}
	if preview["idempotency_error"] == nil {
		t.Fatal("expected idempotency_error to be set")
	}

	withKeyReq := nonIdempotentReq
	withKeyReq.IdempotencyKey = "idem-1"
	previewOK := buildDryRunPreview(cfg, withKeyReq)
	if idemValid, ok := previewOK["idempotency_valid"].(bool); !ok || !idemValid {
		t.Fatalf("expected idempotency_valid=true with key, got %v", previewOK["idempotency_valid"])
	}

	idempotentReq := actions.ExecutionRequest{
		RequestID: "req-2",
		Action: actions.ActionDefinition{
			Name:       "svc.list",
			Idempotent: true,
		},
		Input: map[string]any{},
	}
	previewIdem := buildDryRunPreview(cfg, idempotentReq)
	if idemValid, ok := previewIdem["idempotency_valid"].(bool); !ok || !idemValid {
		t.Fatalf("expected idempotency_valid=true for idempotent action, got %v", previewIdem["idempotency_valid"])
	}
}

func TestSplitGlobalCallFlags_DoubleDashStopsParsing(t *testing.T) {
	resetOptsForTest(t)

	out, err := splitGlobalCallFlags([]string{"--format", "json", "--", "--format", "csv", "--json", "-"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.format != "json" {
		t.Fatalf("expected format=json before --, got %q", opts.format)
	}
	expected := []string{"--", "--format", "csv", "--json", "-"}
	if len(out) != len(expected) {
		t.Fatalf("expected output %v, got %v", expected, out)
	}
	for i, v := range expected {
		if out[i] != v {
			t.Fatalf("expected out[%d]=%q, got %q", i, v, out[i])
		}
	}
}

func TestSplitGlobalCallFlags_ParsesSplashColor(t *testing.T) {
	resetOptsForTest(t)

	out, err := splitGlobalCallFlags([]string{"--splash-color", "ansi256", "svc.act", "--name", "kim"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.splashColor != "ansi256" {
		t.Fatalf("expected splashColor=ansi256, got %q", opts.splashColor)
	}
	expected := []string{"svc.act", "--name", "kim"}
	if len(out) != len(expected) {
		t.Fatalf("expected output %v, got %v", expected, out)
	}
	for i, v := range expected {
		if out[i] != v {
			t.Fatalf("expected out[%d]=%q, got %q", i, v, out[i])
		}
	}
}

func TestPrescanCallSplashFlags_DoubleDashStopsParsing(t *testing.T) {
	resetOptsForTest(t)

	opts.format = "text"
	prescanCallSplashFlags([]string{"svc.act", "--", "--format", "json", "--no-splash"})

	if opts.format != "text" {
		t.Fatalf("expected format to remain text after --, got %q", opts.format)
	}
	if opts.noSplash {
		t.Fatal("expected noSplash to remain false after --")
	}
}

func TestNormalizeCallInputTokensForGlobalFormatConsumesJsonWhenActionHasNoFormatField(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "text"

	def := actions.ActionDefinition{
		InputSchema: &actions.Schema{Properties: map[string]*actions.Schema{
			"name":  {Type: "string"},
			"count": {Type: "integer"},
		}},
	}

	input := []string{"--name", "Seoul", "--count", "1", "--format", "json"}
	out := normalizeCallInputTokensForGlobalFormat(input, def)

	if opts.format != "json" {
		t.Fatalf("expected global format json, got %q", opts.format)
	}
	if len(out) != 4 || out[0] != "--name" || out[1] != "Seoul" || out[2] != "--count" || out[3] != "1" {
		t.Fatalf("expected --format json consumed from input tokens, got %v", out)
	}
}

func TestNormalizeCallInputTokensForGlobalFormatKeepsFormatWhenActionDefinesFormatField(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "text"

	def := actions.ActionDefinition{
		InputSchema: &actions.Schema{Properties: map[string]*actions.Schema{
			"format": {Type: "string"},
			"query":  {Type: "string"},
		}},
	}

	input := []string{"--query", "diagram", "--format", "png"}
	out := normalizeCallInputTokensForGlobalFormat(input, def)

	if opts.format != "text" {
		t.Fatalf("expected global format to remain text, got %q", opts.format)
	}
	if strings.Join(out, " ") != strings.Join(input, " ") {
		t.Fatalf("expected input tokens unchanged, got %v", out)
	}
}

func TestNormalizeCallInputTokensForGlobalFormatKeepsUnknownFormatValueForActionInput(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "text"

	def := actions.ActionDefinition{
		InputSchema: &actions.Schema{Properties: map[string]*actions.Schema{
			"name": {Type: "string"},
		}},
	}

	input := []string{"--name", "foo", "--format", "png"}
	out := normalizeCallInputTokensForGlobalFormat(input, def)

	if opts.format != "text" {
		t.Fatalf("expected global format to remain text, got %q", opts.format)
	}
	if strings.Join(out, " ") != strings.Join(input, " ") {
		t.Fatalf("expected input tokens unchanged, got %v", out)
	}
}

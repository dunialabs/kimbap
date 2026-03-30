package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
)

func parseJSONInput(jsonArg string) (map[string]any, error) {
	var raw string
	switch {
	case jsonArg == "-":
		const maxStdinBytes int64 = 4 << 20
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxStdinBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		if int64(len(data)) > maxStdinBytes {
			return nil, fmt.Errorf("stdin input exceeds %d bytes", maxStdinBytes)
		}
		raw = string(data)
	case strings.HasPrefix(jsonArg, "@"):
		const maxFileBytes int64 = 4 << 20
		f, openErr := os.Open(strings.TrimPrefix(jsonArg, "@"))
		if openErr != nil {
			return nil, fmt.Errorf("read json file: %w", openErr)
		}
		data, readErr := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
		_ = f.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read json file: %w", readErr)
		}
		if int64(len(data)) > maxFileBytes {
			return nil, fmt.Errorf("json file input exceeds %d bytes", maxFileBytes)
		}
		raw = string(data)
	default:
		raw = jsonArg
	}

	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var parsed map[string]any
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("parse json input: %w", err)
	}
	var extra json.RawMessage
	if dec.Decode(&extra) != io.EOF {
		return nil, fmt.Errorf("parse json input: unexpected trailing data after JSON object")
	}
	return coerceJSONNumbers(parsed), nil
}

func coerceJSONNumbers(m map[string]any) map[string]any {
	for k, v := range m {
		switch val := v.(type) {
		case json.Number:
			if i, err := val.Int64(); err == nil {
				m[k] = i
			} else if f, err := val.Float64(); err == nil {
				m[k] = f
			}
		case map[string]any:
			m[k] = coerceJSONNumbers(val)
		case []any:
			m[k] = coerceJSONNumbersSlice(val)
		}
	}
	return m
}

func coerceJSONNumbersSlice(s []any) []any {
	for i, v := range s {
		switch val := v.(type) {
		case json.Number:
			if n, err := val.Int64(); err == nil {
				s[i] = n
			} else if f, err := val.Float64(); err == nil {
				s[i] = f
			}
		case map[string]any:
			s[i] = coerceJSONNumbers(val)
		case []any:
			s[i] = coerceJSONNumbersSlice(val)
		}
	}
	return s
}

func mergeInputMaps(base map[string]any, override map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range override {
		base[k] = v
	}
	return base
}

func parseOptionalBoolFlagValue(tokens []string, idx int) (bool, int) {
	nextIdx := idx + 1
	if nextIdx >= len(tokens) {
		return true, 0
	}
	next := strings.TrimSpace(tokens[nextIdx])
	if strings.HasPrefix(next, "--") {
		return true, 0
	}
	parsed, err := strconv.ParseBool(next)
	if err != nil {
		return true, 0
	}
	return parsed, 1
}

func printTraceSteps(steps []runtimepkg.TraceStep) error {
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"trace": steps})
}

func parseDynamicInput(tokens []string) (map[string]any, error) {
	out := map[string]any{}

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "--" {
			continue
		}
		if !strings.HasPrefix(tok, "--") {
			return nil, fmt.Errorf("unexpected argument %q, expected --name value", tok)
		}

		nameValue := strings.TrimPrefix(tok, "--")
		if nameValue == "" {
			return nil, fmt.Errorf("empty flag name")
		}

		var (
			name  string
			value any = true
		)
		if left, right, ok := strings.Cut(nameValue, "="); ok {
			name = left
			value = parseScalar(right)
		} else {
			name = nameValue
			if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "--") {
				i++
				value = parseScalar(tokens[i])
			}
		}

		if existing, exists := out[name]; exists {
			switch typed := existing.(type) {
			case []any:
				out[name] = append(typed, value)
			default:
				out[name] = []any{typed, value}
			}
		} else {
			out[name] = value
		}
	}

	return out, nil
}

func normalizeCallInputTokensForGlobalFormat(tokens []string, def actions.ActionDefinition) []string {
	_ = def
	return tokens
}

func splitCallInvocationArgs(tokens []string) (string, []string, bool, error) {
	filtered, err := splitGlobalCallFlags(tokens)
	if err != nil {
		return "", nil, false, err
	}

	showHelp := false
	parts := make([]string, 0, len(filtered))
	for _, tok := range filtered {
		trimmed := strings.TrimSpace(tok)
		if trimmed == "--help" || trimmed == "-h" {
			showHelp = true
			continue
		}
		parts = append(parts, tok)
	}

	actionIdx := -1
	for i, tok := range parts {
		trimmed := strings.TrimSpace(tok)
		if trimmed == "--" {
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			if actionIdx == -1 {
				return "", nil, false, fmt.Errorf("missing action name before argument %q", tok)
			}
			continue
		}
		actionIdx = i
		break
	}

	if actionIdx == -1 {
		if showHelp {
			return "", nil, true, nil
		}
		if len(parts) == 0 {
			return "", nil, false, nil
		}
		return "", nil, false, fmt.Errorf("missing action name: expected <service.action>")
	}

	actionName := strings.TrimSpace(parts[actionIdx])
	inputTokens := make([]string, 0, len(parts)-actionIdx-1)
	inputTokens = append(inputTokens, parts[actionIdx+1:]...)

	if showHelp {
		return actionName, inputTokens, true, nil
	}

	return actionName, inputTokens, false, nil
}

func parseScalar(v string) any {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return ""
	}

	if len(trimmed) > 1 && trimmed[0] == '0' && trimmed[1] != '.' {
		return v
	}
	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(trimmed); err == nil {
		return b
	}
	return v
}

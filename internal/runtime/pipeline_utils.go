package runtime

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/headerutil"
)

func cloneInputMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	cloned, ok := deepCloneValue(input).(map[string]any)
	if !ok {
		return nil
	}
	return cloned
}

func cloneMetaMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func stripRuntimeKeys(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	cleaned := make(map[string]any, len(input))
	for key, value := range input {
		cleaned[key] = value
	}
	delete(cleaned, "_output_mode")
	delete(cleaned, "_budget")
	delete(cleaned, "_max_pages")
	return cleaned
}

func applyRuntimeActionOverrides(action actions.ActionDefinition, input map[string]any) actions.ActionDefinition {
	if action.Pagination == nil {
		return action
	}
	requested := coercePositiveInt(input["_max_pages"])
	if requested <= 0 {
		return action
	}
	effective := requested
	if action.Pagination.MaxPages > 0 && requested > action.Pagination.MaxPages {
		effective = action.Pagination.MaxPages
	}
	overridden := action
	pagination := *action.Pagination
	pagination.MaxPages = effective
	overridden.Pagination = &pagination
	return overridden
}

func coercePositiveInt(value any) int {
	maxInt := int(^uint(0) >> 1)
	switch v := value.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 && v <= int64(maxInt) {
			return int(v)
		}
	case float64:
		if v > 0 && v <= float64(maxInt) {
			return int(v)
		}
	case string:
		parsed := strings.TrimSpace(v)
		if parsed == "" {
			return 0
		}
		n, err := strconv.ParseInt(parsed, 10, 64)
		if err == nil && n > 0 && n <= int64(maxInt) {
			return int(n)
		}
	}
	return 0
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func deepCloneValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = deepCloneValue(vv)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = deepCloneValue(vv)
		}
		return out
	default:
		return v
	}
}

func normalizePolicyDecision(decision string) string {
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "allow":
		return "allow"
	case "deny":
		return "deny"
	case "require_approval":
		return "require_approval"
	default:
		return "deny"
	}
}

func (r *Runtime) now() time.Time {
	if r.Now == nil {
		return time.Now()
	}
	return r.Now()
}

func redactHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}

	redacted := make(map[string]string, len(headers))
	for key, value := range headers {
		if headerutil.IsSensitiveAuditHeader(key) {
			continue
		}
		redacted[key] = value
	}
	return redacted
}

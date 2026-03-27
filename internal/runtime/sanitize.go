package runtime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/dunialabs/kimbap/internal/actions"
)

const (
	defaultMaxTotalInputBytes = 4 * 1024 * 1024
	defaultMaxStringBytes     = 1 * 1024 * 1024
)

var (
	pathTraversalPattern      = regexp.MustCompile(`(?i)\.\.\s*(?:/|\\|%2f|%5c)`)
	ansiEscapePattern         = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	escapedANSIStringPattern  = regexp.MustCompile(`(?i)\\x1b|\\033|\\u001b`)
	dangerousShellCmdPattern  = regexp.MustCompile(`(?i)(?:^|[;&])\s*(?:rm|cat|curl|wget|bash|sh|nc|netcat|python|perl|powershell|pwsh)\b`)
	dangerousPipeCmdPattern   = regexp.MustCompile(`(?i)\|\s*(?:cat|sh|bash|nc|netcat|python|perl|powershell|pwsh)\b`)
	subshellPattern           = regexp.MustCompile(`\$\([^\)]*\)`)
	sqlUnionSelectPattern     = regexp.MustCompile(`(?i)\bunion\s+select\b`)
	sqlAlwaysTruePattern      = regexp.MustCompile(`(?i)\bor\s+['"]?1['"]?\s*=\s*['"]?1['"]?`)
	sqlDropTablePattern       = regexp.MustCompile(`(?i);\s*drop\s+table\b`)
	sqlCommentTerminatorRegex = regexp.MustCompile(`(?m)--\s*$`)
)

type SanitizeOptions struct {
	MaxTotalInputBytes int
	MaxStringBytes     int

	DisablePathTraversalCheck bool
	DisableControlCharCheck   bool
	DisableSizeCheck          bool

	// EnableDangerousPatternCheck opts in to shell/SQL injection pattern matching.
	// Off by default because legitimate payloads (code, SQL queries, scripts) will
	// false-positive. Enable per-action when inputs are known to be plain text.
	EnableDangerousPatternCheck bool
}

func DefaultSanitizeOptions() SanitizeOptions {
	return SanitizeOptions{
		MaxTotalInputBytes: defaultMaxTotalInputBytes,
		MaxStringBytes:     defaultMaxStringBytes,
	}
}

func SanitizeInput(input map[string]any) error {
	if err := checkInputSize(input); err != nil {
		return err
	}

	opts := DefaultSanitizeOptions()
	opts.DisableSizeCheck = true
	return SanitizeInputWithOptions(input, opts)
}

func SanitizeInputWithOptions(input map[string]any, opts SanitizeOptions) error {
	if input == nil {
		return nil
	}

	normalized := normalizeSanitizeOptions(opts)

	if !normalized.DisableSizeCheck {
		if err := checkInputSizeWithOptions(input, normalized); err != nil {
			return err
		}
	}

	return walkValue("input", input, func(path string, value string) error {
		if !normalized.DisablePathTraversalCheck {
			if err := checkPathTraversal(value); err != nil {
				return addSanitizePath(err, path)
			}
		}
		if !normalized.DisableControlCharCheck {
			if err := checkControlChars(value); err != nil {
				return addSanitizePath(err, path)
			}
		}
		if normalized.EnableDangerousPatternCheck {
			if err := checkDangerousPatterns(value); err != nil {
				return addSanitizePath(err, path)
			}
		}
		return nil
	})
}

func normalizeSanitizeOptions(opts SanitizeOptions) SanitizeOptions {
	normalized := opts
	if normalized.MaxTotalInputBytes <= 0 {
		normalized.MaxTotalInputBytes = defaultMaxTotalInputBytes
	}
	if normalized.MaxStringBytes <= 0 {
		normalized.MaxStringBytes = defaultMaxStringBytes
	}
	return normalized
}

func addSanitizePath(err error, path string) error {
	execErr, ok := err.(*actions.ExecutionError)
	if !ok || execErr == nil {
		return err
	}
	if execErr.Details == nil {
		execErr.Details = map[string]any{}
	}
	if _, hasPath := execErr.Details["path"]; !hasPath {
		execErr.Details["path"] = path
	}
	return execErr
}

func walkValue(path string, value any, stringFn func(path string, value string) error) error {
	if value == nil {
		return nil
	}

	switch typed := value.(type) {
	case string:
		return stringFn(path, typed)
	case map[string]any:
		for key, child := range typed {
			if err := stringFn(path+"."+key+"(key)", key); err != nil {
				return err
			}
			if err := walkValue(path+"."+key, child, stringFn); err != nil {
				return err
			}
		}
		return nil
	case []any:
		for i, child := range typed {
			if err := walkValue(fmt.Sprintf("%s[%d]", path, i), child, stringFn); err != nil {
				return err
			}
		}
		return nil
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil
	}

	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.String:
		return stringFn(path, rv.String())
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if err := walkValue(fmt.Sprintf("%s[%d]", path, i), rv.Index(i).Interface(), stringFn); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key()
			if key.Kind() != reflect.String {
				continue
			}
			keyStr := key.String()
			if err := stringFn(path+"."+keyStr+"(key)", keyStr); err != nil {
				return err
			}
			if err := walkValue(path+"."+keyStr, iter.Value().Interface(), stringFn); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkPathTraversal(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	decoded := trimmed
	for range 8 {
		next, decodeErr := url.PathUnescape(decoded)
		if decodeErr != nil {
			break
		}
		if next == decoded {
			break
		}
		decoded = next
	}

	normalized := strings.ToLower(strings.ReplaceAll(decoded, "\\", "/"))
	if pathTraversalPattern.MatchString(normalized) ||
		strings.Contains(normalized, "../") ||
		strings.HasPrefix(normalized, "../") ||
		strings.Contains(normalized, "..%2f") ||
		strings.Contains(normalized, "%2e%2e/") ||
		strings.Contains(normalized, "%2e%2e%2f") ||
		strings.Contains(normalized, "%2e%2e%5c") {
		return validationFailure("potential path traversal sequence detected", map[string]any{"check": "path_traversal"})
	}

	targets := []string{
		"~/.ssh",
		"/.ssh/",
		"/etc/passwd",
		"/etc/shadow",
		"/proc/self/environ",
		"c:/windows/system32",
	}
	for _, target := range targets {
		if strings.Contains(normalized, target) {
			return validationFailure("sensitive filesystem path detected", map[string]any{"check": "path_traversal"})
		}
	}

	return nil
}

func checkControlChars(value string) error {
	if strings.IndexByte(value, 0) >= 0 {
		return validationFailure("null byte detected in input", map[string]any{"check": "control_chars"})
	}
	if ansiEscapePattern.MatchString(value) || escapedANSIStringPattern.MatchString(value) {
		return validationFailure("escape sequence detected in input", map[string]any{"check": "control_chars"})
	}

	for _, r := range value {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if unicode.IsControl(r) || !unicode.IsPrint(r) {
			return validationFailure("non-printable control character detected", map[string]any{"check": "control_chars"})
		}
	}

	return nil
}

func checkInputSize(input map[string]any) error {
	return checkInputSizeWithOptions(input, DefaultSanitizeOptions())
}

func checkInputSizeWithOptions(input map[string]any, opts SanitizeOptions) error {
	normalized := normalizeSanitizeOptions(opts)
	if input == nil {
		return nil
	}

	total := estimateValueSize(input)
	if total > normalized.MaxTotalInputBytes {
		return validationFailure(
			"input payload too large",
			map[string]any{
				"check":       "input_size",
				"size_bytes":  total,
				"limit_bytes": normalized.MaxTotalInputBytes,
			},
		)
	}

	return walkValue("input", input, func(path string, value string) error {
		if len(value) > normalized.MaxStringBytes {
			return validationFailure(
				"input string too large",
				map[string]any{
					"check":       "input_size",
					"path":        path,
					"size_bytes":  len(value),
					"limit_bytes": normalized.MaxStringBytes,
				},
			)
		}
		return nil
	})
}

func checkDangerousPatterns(value string) error {
	if strings.Contains(value, "`") {
		return validationFailure("backtick command substitution pattern detected", map[string]any{"check": "dangerous_patterns"})
	}

	patterns := []*regexp.Regexp{
		dangerousShellCmdPattern,
		dangerousPipeCmdPattern,
		subshellPattern,
		sqlUnionSelectPattern,
		sqlAlwaysTruePattern,
		sqlDropTablePattern,
		sqlCommentTerminatorRegex,
	}

	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return validationFailure("dangerous shell or SQL pattern detected", map[string]any{"check": "dangerous_patterns"})
		}
	}

	return nil
}

func estimateValueSize(value any) int {
	if value == nil {
		return 0
	}

	switch typed := value.(type) {
	case string:
		return len(typed)
	case bool:
		return 1
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return 8
	case map[string]any:
		total := 0
		for key, child := range typed {
			total += len(key)
			total += estimateValueSize(child)
		}
		return total
	case []any:
		total := 0
		for _, child := range typed {
			total += estimateValueSize(child)
		}
		return total
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return 0
	}

	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return 0
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.String:
		return len(rv.String())
	case reflect.Map:
		total := 0
		iter := rv.MapRange()
		for iter.Next() {
			if iter.Key().Kind() == reflect.String {
				total += len(iter.Key().String())
			}
			total += estimateValueSize(iter.Value().Interface())
		}
		return total
	case reflect.Slice, reflect.Array:
		total := 0
		for i := 0; i < rv.Len(); i++ {
			total += estimateValueSize(rv.Index(i).Interface())
		}
		return total
	default:
		blob, err := json.Marshal(value)
		if err != nil {
			return 0
		}
		return len(blob)
	}
}

func validationFailure(message string, details map[string]any) *actions.ExecutionError {
	return actions.NewExecutionError(actions.ErrValidationFailed, message, 400, false, details)
}

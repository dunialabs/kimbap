package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dunialabs/kimbap-core/internal/database"
)

var maxResponseLength = parseResponseMaxLength()

const (
	defaultResponseMaxLength = 300
	maxAllowedResponseLength = 100000
)

var errJSONTruncated = errors.New("json output truncated")

const truncationMarker = "\n[TRUNCATED]"

func parseResponseMaxLength() int {
	v := os.Getenv("LOG_RESPONSE_MAX_LENGTH")
	if v == "" {
		return defaultResponseMaxLength
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultResponseMaxLength
	}
	if n <= 0 {
		return defaultResponseMaxLength
	}
	if n > maxAllowedResponseLength {
		return maxAllowedResponseLength
	}
	return n
}

type SessionLogger struct {
	userID    string
	sessionID string
	tokenMask string
	ip        string
	userAgent string
	service   *LogService
}

func NewSessionLogger(userID string, sessionID string, tokenMask string, ip string, userAgent string) *SessionLogger {
	return &SessionLogger{
		userID:    userID,
		sessionID: sessionID,
		tokenMask: tokenMask,
		ip:        ip,
		userAgent: userAgent,
		service:   GetLogService(),
	}
}

type sessionClientRequestLog struct {
	Action            int
	UpstreamRequestID string
	UniformRequestID  string
	ServerID          *string
	RequestParams     any
	ResponseResult    any
	Error             string
	Duration          *int
	StatusCode        *int
}

func (l *SessionLogger) logClientRequest(data sessionClientRequestLog) {
	l.service.EnqueueLog(l.buildEntry(data.Action, data.ServerID, data.UpstreamRequestID, data.UniformRequestID, nil, nil, data.RequestParams, data.ResponseResult, data.Error, data.Duration, data.StatusCode))
}

type sessionReverseRequestLog struct {
	Action                 int
	ServerID               string
	UpstreamRequestID      string
	UniformRequestID       string
	ParentUniformRequestID string
	ProxyRequestID         string
	RequestParams          any
	ResponseResult         any
	Error                  string
	Duration               *int
	StatusCode             *int
}

func (l *SessionLogger) logReverseRequest(data sessionReverseRequestLog) {
	serverID := data.ServerID
	parent := data.ParentUniformRequestID
	proxyReq := data.ProxyRequestID
	l.service.EnqueueLog(l.buildEntry(data.Action, &serverID, data.UpstreamRequestID, data.UniformRequestID, &parent, &proxyReq, data.RequestParams, data.ResponseResult, data.Error, data.Duration, data.StatusCode))
}

func (l *SessionLogger) LogClientRequest(_ context.Context, entry map[string]any) error {
	data, err := parseSessionClientRequestLog(entry)
	if err != nil {
		return err
	}
	l.logClientRequest(data)
	return nil
}

func (l *SessionLogger) LogReverseRequest(_ context.Context, entry map[string]any) error {
	data, err := parseSessionReverseRequestLog(entry)
	if err != nil {
		return err
	}
	l.logReverseRequest(data)
	return nil
}

func (l *SessionLogger) LogServerLifecycle(_ context.Context, entry map[string]any) error {
	action, err := intField(entry, "action", true)
	if err != nil {
		return err
	}
	serverID := stringPtrField(entry, "serverId")
	l.service.EnqueueLog(database.Log{
		Action:    action,
		UserID:    l.userID,
		SessionID: l.sessionID,
		ServerID:  serverID,
		IP:        l.ip,
		UA:        l.userAgent,
		TokenMask: l.tokenMask,
		Error:     stringField(entry, "error"),
	})
	return nil
}

func (l *SessionLogger) LogError(_ context.Context, entry map[string]any) error {
	action, err := intField(entry, "action", true)
	if err != nil {
		return err
	}
	l.service.EnqueueLog(database.Log{
		Action:            action,
		UserID:            l.userID,
		SessionID:         l.sessionID,
		ServerID:          stringPtrField(entry, "serverId"),
		UpstreamRequestID: stringField(entry, "upstreamRequestId"),
		UniformRequestID:  strPtrIfNonEmpty(stringField(entry, "uniformRequestId")),
		IP:                l.ip,
		UA:                l.userAgent,
		TokenMask:         l.tokenMask,
		Error:             stringField(entry, "error"),
	})
	return nil
}

func (l *SessionLogger) IP() string {
	return l.ip
}

func (l *SessionLogger) UserAgent() string {
	return l.userAgent
}

func (l *SessionLogger) LogSessionLifecycle(action int, errMsg string) {
	l.service.EnqueueLog(database.Log{
		Action:    action,
		UserID:    l.userID,
		SessionID: l.sessionID,
		IP:        l.ip,
		UA:        l.userAgent,
		TokenMask: l.tokenMask,
		Error:     errMsg,
	})
}

func (l *SessionLogger) LogAuth(action int, errMsg string, requestParams any) {
	reqParams := truncateJSONValue(requestParams, maxResponseLength)
	l.service.EnqueueLog(database.Log{
		Action:        action,
		UserID:        l.userID,
		SessionID:     l.sessionID,
		IP:            l.ip,
		UA:            l.userAgent,
		TokenMask:     l.tokenMask,
		Error:         errMsg,
		RequestParams: reqParams,
	})
}

func parseSessionClientRequestLog(entry map[string]any) (sessionClientRequestLog, error) {
	action, err := intField(entry, "action", true)
	if err != nil {
		return sessionClientRequestLog{}, err
	}
	return sessionClientRequestLog{
		Action:            action,
		UpstreamRequestID: stringField(entry, "upstreamRequestId"),
		UniformRequestID:  stringField(entry, "uniformRequestId"),
		ServerID:          stringPtrField(entry, "serverId"),
		RequestParams:     entry["requestParams"],
		ResponseResult:    entry["responseResult"],
		Error:             stringField(entry, "error"),
		Duration:          intPtrField(entry, "duration"),
		StatusCode:        intPtrField(entry, "statusCode"),
	}, nil
}

func parseSessionReverseRequestLog(entry map[string]any) (sessionReverseRequestLog, error) {
	action, err := intField(entry, "action", true)
	if err != nil {
		return sessionReverseRequestLog{}, err
	}
	serverID := stringField(entry, "serverId")
	if serverID == "" {
		return sessionReverseRequestLog{}, fmt.Errorf("missing required field: serverId")
	}
	return sessionReverseRequestLog{
		Action:                 action,
		ServerID:               serverID,
		UpstreamRequestID:      stringField(entry, "upstreamRequestId"),
		UniformRequestID:       stringField(entry, "uniformRequestId"),
		ParentUniformRequestID: stringField(entry, "parentUniformRequestId"),
		ProxyRequestID:         stringField(entry, "proxyRequestId"),
		RequestParams:          entry["requestParams"],
		ResponseResult:         entry["responseResult"],
		Error:                  stringField(entry, "error"),
		Duration:               intPtrField(entry, "duration"),
		StatusCode:             intPtrField(entry, "statusCode"),
	}, nil
}

func intField(entry map[string]any, key string, required bool) (int, error) {
	v, ok := entry[key]
	if !ok || v == nil {
		if required {
			return 0, fmt.Errorf("missing required field: %s", key)
		}
		return 0, nil
	}
	switch n := v.(type) {
	case int:
		return n, nil
	case int32:
		return int(n), nil
	case int64:
		return int(n), nil
	case float64:
		return int(n), nil
	case float32:
		return int(n), nil
	case json.Number:
		iv, err := n.Int64()
		if err != nil {
			return 0, err
		}
		return int(iv), nil
	case string:
		if n == "" {
			if required {
				return 0, fmt.Errorf("missing required field: %s", key)
			}
			return 0, nil
		}
		parsed, err := strconv.Atoi(n)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("invalid field type for %s", key)
	}
}

func intPtrField(entry map[string]any, key string) *int {
	v, err := intField(entry, key, false)
	if err != nil {
		return nil
	}
	if raw, ok := entry[key]; !ok || raw == nil {
		return nil
	}
	return &v
}

func stringField(entry map[string]any, key string) string {
	v, ok := entry[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

func stringPtrField(entry map[string]any, key string) *string {
	v := stringField(entry, key)
	if v == "" {
		return nil
	}
	return &v
}

func (l *SessionLogger) buildEntry(action int, serverID *string, upstreamRequestID string, uniformRequestID string, parentUniformRequestID *string, proxyRequestID *string, requestParams any, responseResult any, errMsg string, duration *int, statusCode *int) database.Log {
	reqParams := truncateJSONValue(redactSensitiveFields(requestParams), maxResponseLength)

	resp := ""
	if responseResult != nil {
		resp = truncateJSONValue(redactSensitiveFields(responseResult), maxResponseLength)
	}

	return database.Log{
		Action:                 action,
		UserID:                 l.userID,
		SessionID:              l.sessionID,
		ServerID:               serverID,
		UpstreamRequestID:      upstreamRequestID,
		UniformRequestID:       strPtrIfNonEmpty(uniformRequestID),
		ParentUniformRequestID: parentUniformRequestID,
		ProxyRequestID:         proxyRequestID,
		IP:                     l.ip,
		UA:                     l.userAgent,
		TokenMask:              l.tokenMask,
		RequestParams:          reqParams,
		ResponseResult:         resp,
		Error:                  errMsg,
		Duration:               duration,
		StatusCode:             statusCode,
	}
}

func strPtrIfNonEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func truncateJSONValue(data any, maxLength int) string {
	if data == nil {
		return ""
	}
	if maxLength <= 0 {
		maxLength = defaultResponseMaxLength
	}
	w := &boundedJSONWriter{max: maxLength}
	err := writeJSONLimited(w, reflect.ValueOf(data))
	if err == nil {
		return w.String()
	}
	if errors.Is(err, errJSONTruncated) {
		truncated := w.String()
		if maxLength <= 0 {
			return truncated + truncationMarker
		}
		if len(truncated)+len(truncationMarker) <= maxLength {
			return truncated + truncationMarker
		}
		allowed := maxLength - len(truncationMarker)
		if allowed <= 0 {
			return utf8SafePrefix(truncationMarker, maxLength)
		}
		return utf8SafePrefix(truncated, allowed) + truncationMarker
	}
	return ""
}

func utf8SafePrefix(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	n = min(n, len(s))
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	if n == 0 {
		return ""
	}
	return s[:n]
}

type boundedJSONWriter struct {
	buf bytes.Buffer
	max int
}

func (w *boundedJSONWriter) String() string {
	return w.buf.String()
}

func (w *boundedJSONWriter) remaining() int {
	if w.max <= 0 {
		return 1<<31 - 1
	}
	return w.max - w.buf.Len()
}

func (w *boundedJSONWriter) writeToken(token string) error {
	if token == "" {
		return nil
	}
	if len(token) > w.remaining() {
		return errJSONTruncated
	}
	_, err := w.buf.WriteString(token)
	if err != nil {
		return err
	}
	return nil
}

func writeJSONLimited(w *boundedJSONWriter, v reflect.Value) error {
	if !v.IsValid() {
		return w.writeToken("null")
	}

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return w.writeToken("null")
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return w.writeToken("true")
		}
		return w.writeToken("false")
	case reflect.String:
		return writeJSONStringLimited(w, v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return w.writeToken(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return w.writeToken(strconv.FormatUint(v.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return w.writeToken("null")
		}
		bitSize := 64
		if v.Kind() == reflect.Float32 {
			bitSize = 32
		}
		return w.writeToken(strconv.FormatFloat(f, 'f', -1, bitSize))
	case reflect.Slice, reflect.Array:
		if err := w.writeToken("["); err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				if err := w.writeToken(","); err != nil {
					return err
				}
			}
			if err := writeJSONLimited(w, v.Index(i)); err != nil {
				return err
			}
		}
		return w.writeToken("]")
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return errors.New("unsupported map key type")
		}
		if err := w.writeToken("{"); err != nil {
			return err
		}
		keys := v.MapKeys()
		for i, key := range keys {
			if i > 0 {
				if err := w.writeToken(","); err != nil {
					return err
				}
			}
			if err := writeJSONStringLimited(w, key.String()); err != nil {
				return err
			}
			if err := w.writeToken(":"); err != nil {
				return err
			}
			if err := writeJSONLimited(w, v.MapIndex(key)); err != nil {
				return err
			}
		}
		return w.writeToken("}")
	case reflect.Struct:
		if err := w.writeToken("{"); err != nil {
			return err
		}
		written := 0
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			name := f.Name
			if tag := f.Tag.Get("json"); tag != "" {
				if tag == "-" {
					continue
				}
				if comma := bytes.IndexByte([]byte(tag), ','); comma >= 0 {
					if comma > 0 {
						name = tag[:comma]
					}
				} else {
					name = tag
				}
			}
			if name == "" {
				continue
			}
			if written > 0 {
				if err := w.writeToken(","); err != nil {
					return err
				}
			}
			if err := writeJSONStringLimited(w, name); err != nil {
				return err
			}
			if err := w.writeToken(":"); err != nil {
				return err
			}
			if err := writeJSONLimited(w, v.Field(i)); err != nil {
				return err
			}
			written++
		}
		return w.writeToken("}")
	default:
		return errors.New("unsupported value type")
	}
}

func writeJSONStringLimited(w *boundedJSONWriter, s string) error {
	if w.remaining() <= 2 {
		return errJSONTruncated
	}
	if err := w.writeToken("\""); err != nil {
		return err
	}

	i := 0
	truncated := false
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		piece := ""
		switch r {
		case utf8.RuneError:
			if size == 1 {
				piece = "\\ufffd"
			} else {
				piece = s[i : i+size]
			}
		case '\\':
			piece = "\\\\"
		case '"':
			piece = "\\\""
		case '\b':
			piece = "\\b"
		case '\f':
			piece = "\\f"
		case '\n':
			piece = "\\n"
		case '\r':
			piece = "\\r"
		case '\t':
			piece = "\\t"
		default:
			if r < 0x20 {
				piece = fmt.Sprintf("\\u%04x", r)
			} else {
				piece = s[i : i+size]
			}
		}

		if len(piece)+1 > w.remaining() {
			truncated = true
			break
		}
		if err := w.writeToken(piece); err != nil {
			return err
		}
		i += size
	}

	if err := w.writeToken("\""); err != nil {
		return err
	}
	if truncated {
		return errJSONTruncated
	}
	return nil
}

var logSensitiveKeys = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey", "api-key",
	"authorization", "credential", "private_key", "privatekey", "private-key",
	"access_key", "accesskey", "refresh_token", "client_secret", "clientsecret",
}

func isLogSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range logSensitiveKeys {
		if lower == s {
			return true
		}
		// Match as a whole word segment separated by _, -, or .
		if strings.HasPrefix(lower, s+"_") || strings.HasPrefix(lower, s+"-") || strings.HasPrefix(lower, s+".") ||
			strings.HasSuffix(lower, "_"+s) || strings.HasSuffix(lower, "-"+s) || strings.HasSuffix(lower, "."+s) ||
			strings.Contains(lower, "_"+s+"_") || strings.Contains(lower, "-"+s+"-") || strings.Contains(lower, "."+s+".") ||
			strings.Contains(lower, "_"+s+"-") || strings.Contains(lower, "-"+s+"_") {
			return true
		}
	}
	return false
}

func redactSensitiveFields(v any) any {
	if v == nil {
		return nil
	}
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, val := range typed {
			if isLogSensitiveKey(k) {
				out[k] = "[REDACTED]"
			} else {
				out[k] = redactSensitiveFields(val)
			}
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = redactSensitiveFields(item)
		}
		return out
	default:
		return v
	}
}

package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const redactedValue = "[REDACTED]"

type Writer interface {
	Write(ctx context.Context, event AuditEvent) error
	Close() error
}

type JSONLWriter struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

func NewJSONLWriter(path string) (*JSONLWriter, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("audit jsonl path is required")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create audit directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open audit jsonl file: %w", err)
	}

	return &JSONLWriter{
		file: f,
		enc:  json.NewEncoder(f),
	}, nil
}

func (w *JSONLWriter) Write(ctx context.Context, event AuditEvent) error {
	if w == nil {
		return errors.New("jsonl writer is not initialized")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.enc == nil {
		return errors.New("jsonl writer is closed")
	}

	if err := w.enc.Encode(event); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

func (w *JSONLWriter) Close() error {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.enc = nil
	return err
}

type MultiWriter struct {
	writers []Writer
}

func NewMultiWriter(writers ...Writer) *MultiWriter {
	flattened := make([]Writer, 0, len(writers))
	for _, w := range writers {
		if w != nil {
			flattened = append(flattened, w)
		}
	}
	return &MultiWriter{writers: flattened}
}

func (m *MultiWriter) Write(ctx context.Context, event AuditEvent) error {
	if m == nil {
		return nil
	}

	var errs []error
	for _, w := range m.writers {
		if err := w.Write(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *MultiWriter) Close() error {
	if m == nil {
		return nil
	}

	var errs []error
	for _, w := range m.writers {
		if err := w.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type TailWriter struct {
	mu          sync.RWMutex
	subscribers map[chan AuditEvent]struct{}
	bufferSize  int
	closed      bool
}

func NewTailWriter(bufferSize int) *TailWriter {
	if bufferSize <= 0 {
		bufferSize = 64
	}
	return &TailWriter{
		subscribers: make(map[chan AuditEvent]struct{}),
		bufferSize:  bufferSize,
	}
}

func (t *TailWriter) Write(_ context.Context, event AuditEvent) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.closed {
		return errors.New("tail writer is closed")
	}
	for ch := range t.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	return nil
}

func (t *TailWriter) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	for ch := range t.subscribers {
		close(ch)
		delete(t.subscribers, ch)
	}
	return nil
}

func (t *TailWriter) Subscribe() <-chan AuditEvent {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		ch := make(chan AuditEvent)
		close(ch)
		return ch
	}
	ch := make(chan AuditEvent, t.bufferSize)
	t.subscribers[ch] = struct{}{}
	return ch
}

func (t *TailWriter) Unsubscribe(ch <-chan AuditEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for sub := range t.subscribers {
		if sub == ch {
			close(sub)
			delete(t.subscribers, sub)
			return
		}
	}
}

type CSVExporter struct{}

var csvHeader = []string{
	"id", "timestamp", "request_id", "trace_id", "tenant_id",
	"principal_id", "agent_name", "service", "action", "mode",
	"status", "policy_decision", "duration_ms", "error_code", "error_message",
}

func (e *CSVExporter) Export(events []AuditEvent) ([]byte, error) {
	var buf strings.Builder
	buf.WriteString(strings.Join(csvHeader, ","))
	buf.WriteByte('\n')
	for _, ev := range events {
		errCode, errMsg := "", ""
		if ev.Error != nil {
			errCode = ev.Error.Code
			errMsg = ev.Error.Message
		}
		row := []string{
			csvEscape(ev.ID),
			ev.Timestamp.UTC().Format(time.RFC3339),
			csvEscape(ev.RequestID),
			csvEscape(ev.TraceID),
			csvEscape(ev.TenantID),
			csvEscape(ev.PrincipalID),
			csvEscape(ev.AgentName),
			csvEscape(ev.Service),
			csvEscape(ev.Action),
			csvEscape(ev.Mode),
			csvEscape(string(ev.Status)),
			csvEscape(ev.PolicyDecision),
			fmt.Sprintf("%d", ev.DurationMS),
			csvEscape(errCode),
			csvEscape(errMsg),
		}
		buf.WriteString(strings.Join(row, ","))
		buf.WriteByte('\n')
	}
	return []byte(buf.String()), nil
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

type Redactor struct {
	patterns []string
}

func NewRedactor(patterns ...string) *Redactor {
	normalized := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			normalized = append(normalized, p)
		}
	}
	return &Redactor{patterns: normalized}
}

func (r *Redactor) Redact(event AuditEvent) AuditEvent {
	if r == nil || len(r.patterns) == 0 {
		return event
	}

	cloned := event
	cloned.Input = redactMap(event.Input, r.patterns)
	cloned.Meta = redactMap(event.Meta, r.patterns)
	if event.Error != nil {
		errCopy := *event.Error
		errCopy.Meta = redactMap(event.Error.Meta, r.patterns)
		cloned.Error = &errCopy
	}
	return cloned
}

func redactMap(input map[string]any, patterns []string) map[string]any {
	if input == nil {
		return nil
	}

	out := make(map[string]any, len(input))
	for key, value := range input {
		if shouldRedact(key, patterns) {
			out[key] = redactedValue
			continue
		}
		out[key] = redactValue(value, patterns)
	}
	return out
}

func redactValue(value any, patterns []string) any {
	switch v := value.(type) {
	case map[string]any:
		return redactMap(v, patterns)
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = redactValue(v[i], patterns)
		}
		return out
	default:
		return value
	}
}

func shouldRedact(key string, patterns []string) bool {
	lower := strings.ToLower(key)
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

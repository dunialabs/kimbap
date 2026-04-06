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

type redactor struct {
	patterns []string
}

func newRedactor(patterns ...string) *redactor {
	normalized := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			normalized = append(normalized, p)
		}
	}
	return &redactor{patterns: normalized}
}

func (r *redactor) Redact(event AuditEvent) AuditEvent {
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

var DefaultRedactPatterns = []string{
	"password", "secret", "token", "key", "credential",
	"authorization", "cookie", "api_key", "apikey", "private",
	"masterPwd", "master_password",
}

type RedactingWriter struct {
	inner Writer
	r     *redactor
}

func NewRedactingWriter(inner Writer, patterns ...string) *RedactingWriter {
	if len(patterns) == 0 {
		patterns = DefaultRedactPatterns
	}
	return &RedactingWriter{inner: inner, r: newRedactor(patterns...)}
}

func (w *RedactingWriter) Write(ctx context.Context, event AuditEvent) error {
	return w.inner.Write(ctx, w.r.Redact(event))
}

func (w *RedactingWriter) Close() error {
	return w.inner.Close()
}

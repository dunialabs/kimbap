package observability

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
)

var stderrCaptureMu sync.Mutex

func captureStderrOutput(t *testing.T, fn func()) string {
	t.Helper()
	stderrCaptureMu.Lock()
	defer stderrCaptureMu.Unlock()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr
	}()

	fn()

	_ = w.Close()
	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("read stderr output: %v", readErr)
	}

	return string(out)
}

func TestNewLoggerLevelAndFormatSelection(t *testing.T) {
	traceLogger := NewLogger("trace", "json")
	if !traceLogger.Enabled(context.Background(), traceLevel) {
		t.Fatal("trace logger should enable trace level")
	}
	if !traceLogger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("trace logger should enable debug level")
	}

	jsonLogger := NewLogger("debug", "json")
	if !jsonLogger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("debug json logger should enable debug level")
	}

	textLogger := NewLogger("warn", "text")
	if textLogger.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("warn text logger should not enable info level")
	}
	if !textLogger.Enabled(context.Background(), slog.LevelWarn) {
		t.Fatal("warn text logger should enable warn level")
	}

	fallback := NewLogger("unknown", "unknown")
	if !fallback.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("fallback logger should default to info level")
	}
}

func TestNewLoggerJSONOutputContainsLevelMessageAndFields(t *testing.T) {
	out := captureStderrOutput(t, func() {
		logger := NewLogger("info", "json")
		logger.Info("json-log", "component", "observability", "status", "ok")
	})

	line := strings.TrimSpace(out)
	if line == "" {
		t.Fatal("expected non-empty JSON logger output")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("decode JSON logger output: %v\nraw=%s", err, line)
	}
	if got := payload["level"]; got != "INFO" {
		t.Fatalf("level field = %v, want INFO", got)
	}
	if got := payload["msg"]; got != "json-log" {
		t.Fatalf("msg field = %v, want json-log", got)
	}
	if got := payload["component"]; got != "observability" {
		t.Fatalf("component field = %v, want observability", got)
	}
	if got := payload["status"]; got != "ok" {
		t.Fatalf("status field = %v, want ok", got)
	}
}

func TestNewLoggerTextOutputContainsLevelAndMessage(t *testing.T) {
	out := captureStderrOutput(t, func() {
		logger := NewLogger("info", "text")
		logger.Info("text-log", "component", "observability")
	})

	line := strings.TrimSpace(out)
	if line == "" {
		t.Fatal("expected non-empty text logger output")
	}
	if !strings.Contains(line, "level=INFO") {
		t.Fatalf("text output missing level field: %s", line)
	}
	if !strings.Contains(line, "msg=text-log") {
		t.Fatalf("text output missing message field: %s", line)
	}
	if !strings.Contains(line, "component=observability") {
		t.Fatalf("text output missing custom field: %s", line)
	}
}

func TestNewLoggerWarnLevelSuppressesInfoLogs(t *testing.T) {
	out := captureStderrOutput(t, func() {
		logger := NewLogger("warn", "json")
		logger.Info("should-not-appear")
	})

	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected info log to be suppressed at warn level, got output: %s", out)
	}

	warnOut := captureStderrOutput(t, func() {
		logger := NewLogger("warn", "json")
		logger.Warn("warn-visible")
	})
	if strings.TrimSpace(warnOut) == "" {
		t.Fatal("expected warn log to be emitted at warn level")
	}
}

func TestNewLoggerUnknownFormatFallsBackToText(t *testing.T) {
	out := captureStderrOutput(t, func() {
		logger := NewLogger("info", "unknown")
		logger.Info("fallback-text-log")
	})

	line := strings.TrimSpace(out)
	if line == "" {
		t.Fatal("expected fallback logger output")
	}
	if strings.HasPrefix(line, "{") {
		t.Fatalf("unknown format should fall back to text handler, got JSON output: %s", line)
	}
	if !strings.Contains(line, "msg=fallback-text-log") {
		t.Fatalf("fallback text output missing message: %s", line)
	}
}

func TestNewLoggerNormalizesLevelAndFormatInput(t *testing.T) {
	logger := NewLogger(" WARN ", " JSON ")
	if logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("normalized warn logger should suppress info level")
	}

	out := captureStderrOutput(t, func() {
		logger := NewLogger(" WARN ", " JSON ")
		logger.Info("suppressed")
		logger.Warn("normalized-json-log")
	})

	line := strings.TrimSpace(out)
	if line == "" {
		t.Fatal("expected normalized logger output")
	}
	if !strings.HasPrefix(line, "{") {
		t.Fatalf("expected JSON output after format normalization, got: %s", line)
	}
}

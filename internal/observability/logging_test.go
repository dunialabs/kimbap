package observability

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewLoggerLevelAndFormatSelection(t *testing.T) {
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

package logger

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestResolveLogLevelFromEnvironment(t *testing.T) {
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("NODE_ENV", "production")
	if lvl := resolveLogLevel(); lvl != zerolog.WarnLevel {
		t.Fatalf("resolveLogLevel() = %s, want warn", lvl)
	}
}

func TestResolveLogLevelDefaultsByEnvironment(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("NODE_ENV", "development")
	if lvl := resolveLogLevel(); lvl != zerolog.TraceLevel {
		t.Fatalf("development default level = %s, want trace", lvl)
	}

	t.Setenv("NODE_ENV", "production")
	if lvl := resolveLogLevel(); lvl != zerolog.InfoLevel {
		t.Fatalf("production default level = %s, want info", lvl)
	}
}

func TestResolveEnvPriorityAndPrettyToggle(t *testing.T) {
	t.Setenv("NODE_ENV", "")
	t.Setenv("APP_ENV", "staging")
	if env := resolveEnv(); env != "staging" {
		t.Fatalf("resolveEnv() = %q, want staging", env)
	}
	if isDevelopment() {
		t.Fatal("staging should not be treated as development")
	}

	if !isPrettyEnabled("true") {
		t.Fatal("isPrettyEnabled(true) should be true")
	}
	if isPrettyEnabled("off") {
		t.Fatal("isPrettyEnabled(off) should be false")
	}

	t.Setenv("NODE_ENV", "development")
	if !isPrettyEnabled(" ") {
		t.Fatal("empty LOG_PRETTY should default to development=true")
	}

	t.Setenv("NODE_ENV", "production")
	if isPrettyEnabled(" ") {
		t.Fatal("empty LOG_PRETTY should default to development=false in production")
	}
}

func TestCreateLoggerEmitsModuleAndExtraFields(t *testing.T) {
	t.Setenv("NODE_ENV", "production")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_PRETTY", "false")

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	logger := CreateLogger("unit-test", map[string]any{"component": "logger"})
	logger.Info().Str("event", "emit").Msg("hello")

	_ = w.Close()
	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("read logger output: %v", readErr)
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode logger JSON output: %v\nraw=%s", err, string(out))
	}
	if got := payload["module"]; got != "unit-test" {
		t.Fatalf("module field = %v, want unit-test", got)
	}
	if got := payload["component"]; got != "logger" {
		t.Fatalf("component field = %v, want logger", got)
	}
	if _, ok := payload["pid"]; !ok {
		t.Fatal("expected pid field in logger payload")
	}
}

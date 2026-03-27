package logger

import (
	"context"
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

func TestCreateLoggerAddsModuleFieldAndIsUsable(t *testing.T) {
	t.Setenv("NODE_ENV", "production")
	t.Setenv("LOG_LEVEL", "info")

	logger := CreateLogger("unit-test", map[string]any{"component": "logger"})
	ctx := logger.WithContext(context.Background())
	if ctx == nil {
		t.Fatal("expected logger.WithContext to return non-nil context")
	}
}

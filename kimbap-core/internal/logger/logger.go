package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func CreateLogger(module string, fields ...map[string]any) zerolog.Logger {
	level := resolveLogLevel()
	zerolog.SetGlobalLevel(level)

	var out io.Writer = os.Stdout
	if isPrettyEnabled(os.Getenv("LOG_PRETTY")) {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = os.Getenv("HOSTNAME")
		if hostname == "" {
			hostname = "unknown"
		}
	}

	ctx := zerolog.New(out).Level(level).With().Timestamp().
		Int("pid", os.Getpid()).
		Str("hostname", hostname).
		Str("env", resolveEnv()).
		Str("module", module)
	for _, extra := range fields {
		for key, val := range extra {
			ctx = ctx.Interface(key, val)
		}
	}

	return ctx.Logger()
}

func resolveLogLevel() zerolog.Level {
	if rawLevel := strings.TrimSpace(os.Getenv("LOG_LEVEL")); rawLevel != "" {
		if parsedLevel, err := zerolog.ParseLevel(strings.ToLower(rawLevel)); err == nil {
			return parsedLevel
		}
	}
	if isDevelopment() {
		return zerolog.TraceLevel
	}
	return zerolog.InfoLevel
}

func resolveEnv() string {
	if env := os.Getenv("NODE_ENV"); env != "" {
		return env
	}
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}
	return "development"
}

func isDevelopment() bool {
	env := resolveEnv()
	return env != "production" && env != "staging"
}

func isPrettyEnabled(v string) bool {
	val := strings.TrimSpace(strings.ToLower(v))
	switch val {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return isDevelopment()
	}
}

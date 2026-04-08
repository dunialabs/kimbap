package config

import (
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

var version = "0.0.1-dev"

var (
	commit = "unknown"
	date   = ""
)

var AppInfo = struct {
	Name        string
	Version     string
	Commit      string
	Date        string
	Description string
}{
	Name:        "kimbap",
	Version:     version,
	Commit:      commit,
	Date:        date,
	Description: "Kimbap - Secure action runtime for AI agents",
}

var dotenvOnce sync.Once

func CLIVersion() string {
	v := strings.TrimSpace(AppInfo.Version)
	if v == "" {
		return "0.0.1-dev"
	}
	return v
}

func CLIVersionDisplay() string {
	v := CLIVersion()

	meta := make([]string, 0, 2)
	if c := strings.TrimSpace(AppInfo.Commit); c != "" && !strings.EqualFold(c, "unknown") {
		if len(c) > 12 {
			c = c[:12]
		}
		meta = append(meta, "commit "+c)
	}
	if d := strings.TrimSpace(AppInfo.Date); d != "" {
		meta = append(meta, d)
	}

	if len(meta) == 0 {
		return v
	}
	return v + " (" + strings.Join(meta, ", ") + ")"
}

func ensureDotenvLoaded() {
	dotenvOnce.Do(func() {
		_ = godotenv.Load()
	})
}

func Env(key string, defaultVal ...string) string {
	ensureDotenvLoaded()
	if val := os.Getenv(key); val != "" {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

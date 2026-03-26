package config

import (
	"os"

	"github.com/joho/godotenv"
)

// version is set at build time via -ldflags "-X github.com/dunialabs/kimbap-core/internal/config.version=X.Y.Z"
// Falls back to "1.1.0" if not set.
var version = "1.1.0"

var AppInfo = struct {
	Name        string
	Version     string
	Description string
}{
	Name:        "kimbap-core",
	Version:     version,
	Description: "Kimbap - Secure action runtime for AI agents",
}

func init() {
	_ = godotenv.Load()
}

func Env(key string, defaultVal ...string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

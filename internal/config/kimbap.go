package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type KimbapConfig struct {
	Mode          string              `yaml:"mode"`
	DataDir       string              `yaml:"data_dir"`
	ListenAddr    string              `yaml:"listen_addr"`
	ProxyAddr     string              `yaml:"proxy_addr"`
	LogLevel      string              `yaml:"log_level"`
	LogFormat     string              `yaml:"log_format"`
	Console       ConsoleConfig       `yaml:"console"`
	Vault         VaultConfig         `yaml:"vault"`
	Auth          AuthConfig          `yaml:"auth"`
	Audit         AuditConfig         `yaml:"audit"`
	Policy        PolicyConfig        `yaml:"policy"`
	Services      ServicesConfig      `yaml:"services"`
	Database      DatabaseConfig      `yaml:"database"`
	Notifications NotificationsConfig `yaml:"notifications"`
	Aliases       map[string]string   `yaml:"aliases,omitempty"`
}

type VaultConfig struct {
	Backend string `yaml:"backend"`
	Path    string `yaml:"path"`
}

type ConsoleConfig struct {
	Enabled bool `yaml:"enabled"`
}

type AuthConfig struct {
	TokenTTL   string `yaml:"token_ttl"`
	SessionTTL string `yaml:"session_ttl"`
	ServerURL  string `yaml:"server_url"`
}

type AuditConfig struct {
	Sink string `yaml:"sink"`
	Path string `yaml:"path"`
}

type PolicyConfig struct {
	Path string `yaml:"path"`
}

type ServicesConfig struct {
	Dir             string `yaml:"dir"`
	RegistryURL     string `yaml:"registry_url"`
	Verify          string `yaml:"verify"`
	SignaturePolicy string `yaml:"signature_policy"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type SlackNotificationConfig struct {
	WebhookURL string `yaml:"webhook_url"`
}

type TelegramNotificationConfig struct {
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

type EmailNotificationConfig struct {
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
}

type WebhookNotificationConfig struct {
	URL     string `yaml:"url"`
	SignKey string `yaml:"sign_key"`
}

type NotificationsConfig struct {
	Slack    SlackNotificationConfig    `yaml:"slack"`
	Telegram TelegramNotificationConfig `yaml:"telegram"`
	Email    EmailNotificationConfig    `yaml:"email"`
	Webhook  WebhookNotificationConfig  `yaml:"webhook"`
}

func DefaultConfig() *KimbapConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".kimbap")

	return &KimbapConfig{
		Mode:       "embedded",
		DataDir:    dataDir,
		ListenAddr: ":8080",
		ProxyAddr:  "127.0.0.1:7788",
		LogLevel:   "info",
		LogFormat:  "text",
		Console: ConsoleConfig{
			Enabled: false,
		},
		Aliases: map[string]string{},
		Vault: VaultConfig{
			Backend: "sqlite",
			Path:    filepath.Join(dataDir, "vault.db"),
		},
		Auth: AuthConfig{
			TokenTTL:   "720h",
			SessionTTL: "15m",
			ServerURL:  "",
		},
		Audit: AuditConfig{
			Sink: "jsonl",
			Path: filepath.Join(dataDir, "audit.jsonl"),
		},
		Policy: PolicyConfig{Path: filepath.Join(dataDir, "policy.yaml")},
		Services: ServicesConfig{
			Dir:             filepath.Join(dataDir, "services"),
			RegistryURL:     "https://services.kimbap.ai",
			Verify:          "warn",
			SignaturePolicy: "optional",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    filepath.Join(dataDir, "kimbap.db"),
		},
	}
}

func LoadKimbapConfig(paths ...string) (*KimbapConfig, error) {
	return loadKimbapConfig(true, paths...)
}

// LoadKimbapConfigWithoutDefault loads defaults, env overrides, and explicit files without reading the discovered default config file.
func LoadKimbapConfigWithoutDefault(paths ...string) (*KimbapConfig, error) {
	return loadKimbapConfig(false, paths...)
}

func loadKimbapConfig(includeDefault bool, paths ...string) (*KimbapConfig, error) {
	cfg := DefaultConfig()

	if includeDefault {
		defaultPath, err := defaultKimbapConfigPath()
		if err != nil {
			return nil, err
		}
		prev := *cfg
		if err := mergeConfigFromFile(cfg, defaultPath, false); err != nil {
			return nil, err
		}
		rebaseDerivedPathsForDataDir(&prev, cfg)
	}

	prev := *cfg
	applyKimbapEnv(cfg)
	rebaseDerivedPathsForDataDir(&prev, cfg)

	for _, p := range paths {
		if p == "" {
			continue
		}
		prev = *cfg
		if err := mergeConfigFromFile(cfg, p, true); err != nil {
			return nil, err
		}
		rebaseDerivedPathsForDataDir(&prev, cfg)
	}

	verifyMode, normErr := normalizeServiceVerifyMode(cfg.Services.Verify)
	if normErr != nil {
		return nil, normErr
	}
	cfg.Services.Verify = verifyMode

	sigPolicy, normErr := normalizeServiceSignaturePolicy(cfg.Services.SignaturePolicy)
	if normErr != nil {
		return nil, normErr
	}
	cfg.Services.SignaturePolicy = sigPolicy

	return cfg, nil
}

func ResolveConfigPath(explicitPath string) (string, error) {
	if trimmed := strings.TrimSpace(explicitPath); trimmed != "" {
		return trimmed, nil
	}

	xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	xdgPath := ""
	xdgIsDir := false
	if xdg != "" {
		xdgPath = filepath.Join(xdg, "kimbap", "config.yaml")
		if st, err := os.Stat(xdgPath); err == nil {
			if !st.IsDir() {
				return xdgPath, nil
			}
			xdgIsDir = true
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat xdg config path %q: %w", xdgPath, err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		if xdgPath != "" {
			if xdgIsDir {
				return "", fmt.Errorf("config path is a directory: %s", xdgPath)
			}
			return xdgPath, nil
		}
		return "", errors.New("resolve user home directory")
	}

	legacyPath := filepath.Join(homeDir, ".kimbap", "config.yaml")
	if xdgPath != "" {
		if st, err := os.Stat(legacyPath); err == nil {
			if st.IsDir() {
				return "", fmt.Errorf("legacy config path is a directory: %s", legacyPath)
			}
			return legacyPath, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat legacy config path %q: %w", legacyPath, err)
		}
		if xdgIsDir {
			return "", fmt.Errorf("config path is a directory: %s", xdgPath)
		}
		return xdgPath, nil
	}

	if st, err := os.Stat(legacyPath); err == nil && st.IsDir() {
		return "", fmt.Errorf("legacy config path is a directory: %s", legacyPath)
	}
	return legacyPath, nil
}

func defaultKimbapConfigPath() (string, error) {
	return ResolveConfigPath("")
}

func ApplyDataDirOverride(cfg *KimbapConfig, dataDir string) {
	if cfg == nil {
		return
	}
	trimmed := strings.TrimSpace(dataDir)
	if trimmed == "" {
		return
	}
	prev := *cfg
	cfg.DataDir = trimmed
	rebaseDerivedPathsForDataDir(&prev, cfg)
}

func mergeConfigFromFile(cfg *KimbapConfig, path string, required bool) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !required {
			return nil
		}
		return fmt.Errorf("read config file %q: %w", path, err)
	}

	var loaded KimbapConfig
	if err := yaml.Unmarshal(content, &loaded); err != nil {
		return fmt.Errorf("parse config file %q: %w", path, err)
	}
	var boolPresence struct {
		Console struct {
			Enabled *bool `yaml:"enabled"`
		} `yaml:"console"`
		Notifications struct {
			Email struct {
				To *[]string `yaml:"to"`
			} `yaml:"email"`
		} `yaml:"notifications"`
	}
	if err := yaml.Unmarshal(content, &boolPresence); err != nil {
		return fmt.Errorf("parse config file %q: %w", path, err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(content, &raw); err == nil {
		if _, hasLegacy := raw["skills"]; hasLegacy {
			return fmt.Errorf("config contains unsupported 'skills:' key — use 'services:'")
		}
		warnUnknownConfigKeys(raw, path)
		if _, hasAliases := raw["aliases"]; hasAliases && len(loaded.Aliases) == 0 {
			cfg.Aliases = nil
		}
	}

	mergeConfig(cfg, &loaded)
	if boolPresence.Console.Enabled != nil {
		cfg.Console.Enabled = *boolPresence.Console.Enabled
	}
	if boolPresence.Notifications.Email.To != nil {
		cfg.Notifications.Email.To = loaded.Notifications.Email.To
	}
	return nil
}

func warnUnknownConfigKeys(raw map[string]any, path string) {
	topLevel := map[string]bool{
		"mode": true, "data_dir": true, "listen_addr": true,
		"proxy_addr": true, "log_level": true, "log_format": true,
		"console": true,
		"vault":   true, "auth": true, "audit": true, "policy": true,
		"services": true, "database": true, "notifications": true,
		"aliases": true,
	}
	for key := range raw {
		if !topLevel[key] {
			_, _ = fmt.Fprintf(os.Stderr,
				"warning: unknown config key %q in %s (known: mode, data_dir, console, vault, auth, audit, policy, services, database, notifications, aliases)\n",
				key, path)
		}
	}
	sectionKeys := map[string]map[string]bool{
		"console":       {"enabled": true},
		"vault":         {"backend": true, "path": true},
		"auth":          {"token_ttl": true, "session_ttl": true, "server_url": true},
		"audit":         {"sink": true, "path": true},
		"policy":        {"path": true},
		"services":      {"dir": true, "registry_url": true, "verify": true, "signature_policy": true},
		"database":      {"driver": true, "dsn": true},
		"notifications": {"slack": true, "telegram": true, "email": true, "webhook": true},
	}
	for section, knownKeys := range sectionKeys {
		val, ok := raw[section]
		if !ok {
			continue
		}
		nested, ok := val.(map[string]any)
		if !ok {
			continue
		}
		for key := range nested {
			if !knownKeys[key] {
				_, _ = fmt.Fprintf(os.Stderr,
					"warning: unknown config key %q in %s under %q\n", key, path, section)
			}
		}
	}
	notifSubKeys := map[string]map[string]bool{
		"slack":    {"webhook_url": true},
		"telegram": {"bot_token": true, "chat_id": true},
		"email":    {"smtp_host": true, "smtp_port": true, "from": true, "to": true, "username": true, "password": true},
		"webhook":  {"url": true, "sign_key": true},
	}
	if notif, ok := raw["notifications"].(map[string]any); ok {
		for sub, knownKeys := range notifSubKeys {
			if subMap, ok := notif[sub].(map[string]any); ok {
				for key := range subMap {
					if !knownKeys[key] {
						_, _ = fmt.Fprintf(os.Stderr,
							"warning: unknown config key %q in %s under notifications.%s\n", key, path, sub)
					}
				}
			}
		}
	}
}

func rebaseDerivedPathsForDataDir(prev *KimbapConfig, cfg *KimbapConfig) {
	if prev == nil || cfg == nil || prev.DataDir == cfg.DataDir || cfg.DataDir == "" {
		return
	}

	if cfg.Vault.Path == prev.Vault.Path && prev.Vault.Path == filepath.Join(prev.DataDir, "vault.db") {
		cfg.Vault.Path = filepath.Join(cfg.DataDir, "vault.db")
	}
	if cfg.Audit.Path == prev.Audit.Path && prev.Audit.Path == filepath.Join(prev.DataDir, "audit.jsonl") {
		cfg.Audit.Path = filepath.Join(cfg.DataDir, "audit.jsonl")
	}
	if cfg.Policy.Path == prev.Policy.Path && prev.Policy.Path == filepath.Join(prev.DataDir, "policy.yaml") {
		cfg.Policy.Path = filepath.Join(cfg.DataDir, "policy.yaml")
	}
	if cfg.Services.Dir == prev.Services.Dir && prev.Services.Dir == filepath.Join(prev.DataDir, "services") {
		cfg.Services.Dir = filepath.Join(cfg.DataDir, "services")
	}
	if cfg.Database.DSN == prev.Database.DSN && prev.Database.DSN == filepath.Join(prev.DataDir, "kimbap.db") {
		cfg.Database.DSN = filepath.Join(cfg.DataDir, "kimbap.db")
	}
}

func applyKimbapEnv(cfg *KimbapConfig) {
	setIfNotEmpty(&cfg.Mode, os.Getenv("KIMBAP_MODE"))
	setIfNotEmpty(&cfg.DataDir, os.Getenv("KIMBAP_DATA_DIR"))
	setIfNotEmpty(&cfg.ListenAddr, os.Getenv("KIMBAP_LISTEN_ADDR"))
	setIfNotEmpty(&cfg.ProxyAddr, os.Getenv("KIMBAP_PROXY_ADDR"))
	setIfNotEmpty(&cfg.LogLevel, os.Getenv("KIMBAP_LOG_LEVEL"))
	setIfNotEmpty(&cfg.LogFormat, os.Getenv("KIMBAP_LOG_FORMAT"))
	if v := strings.TrimSpace(os.Getenv("KIMBAP_CONSOLE_ENABLED")); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.Console.Enabled = enabled
		}
	}

	setIfNotEmpty(&cfg.Vault.Backend, os.Getenv("KIMBAP_VAULT_BACKEND"))
	setIfNotEmpty(&cfg.Vault.Path, os.Getenv("KIMBAP_VAULT_PATH"))

	setIfNotEmpty(&cfg.Auth.TokenTTL, os.Getenv("KIMBAP_AUTH_TOKEN_TTL"))
	setIfNotEmpty(&cfg.Auth.SessionTTL, os.Getenv("KIMBAP_AUTH_SESSION_TTL"))
	setIfNotEmpty(&cfg.Auth.ServerURL, os.Getenv("KIMBAP_AUTH_SERVER_URL"))

	setIfNotEmpty(&cfg.Audit.Sink, os.Getenv("KIMBAP_AUDIT_SINK"))
	setIfNotEmpty(&cfg.Audit.Path, os.Getenv("KIMBAP_AUDIT_PATH"))

	setIfNotEmpty(&cfg.Policy.Path, os.Getenv("KIMBAP_POLICY_PATH"))

	setIfNotEmpty(&cfg.Services.Dir, os.Getenv("KIMBAP_SERVICES_DIR"))
	setIfNotEmpty(&cfg.Services.RegistryURL, os.Getenv("KIMBAP_SERVICES_REGISTRY_URL"))
	setIfNotEmpty(&cfg.Services.Verify, os.Getenv("KIMBAP_SERVICES_VERIFY"))
	setIfNotEmpty(&cfg.Services.SignaturePolicy, os.Getenv("KIMBAP_SERVICES_SIGNATURE_POLICY"))

	setIfNotEmpty(&cfg.Database.Driver, os.Getenv("KIMBAP_DATABASE_DRIVER"))
	setIfNotEmpty(&cfg.Database.DSN, os.Getenv("KIMBAP_DATABASE_DSN"))

	setIfNotEmpty(&cfg.Notifications.Slack.WebhookURL, os.Getenv("KIMBAP_NOTIFICATIONS_SLACK_WEBHOOK_URL"))

	setIfNotEmpty(&cfg.Notifications.Telegram.BotToken, os.Getenv("KIMBAP_NOTIFICATIONS_TELEGRAM_BOT_TOKEN"))
	setIfNotEmpty(&cfg.Notifications.Telegram.ChatID, os.Getenv("KIMBAP_NOTIFICATIONS_TELEGRAM_CHAT_ID"))

	setIfNotEmpty(&cfg.Notifications.Email.SMTPHost, os.Getenv("KIMBAP_NOTIFICATIONS_EMAIL_SMTP_HOST"))
	if v := os.Getenv("KIMBAP_NOTIFICATIONS_EMAIL_SMTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Notifications.Email.SMTPPort = port
		}
	}
	setIfNotEmpty(&cfg.Notifications.Email.From, os.Getenv("KIMBAP_NOTIFICATIONS_EMAIL_FROM"))
	if v := os.Getenv("KIMBAP_NOTIFICATIONS_EMAIL_TO"); v != "" {
		parts := strings.Split(v, ",")
		trimmed := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				trimmed = append(trimmed, s)
			}
		}
		cfg.Notifications.Email.To = trimmed
	}
	setIfNotEmpty(&cfg.Notifications.Email.Username, os.Getenv("KIMBAP_NOTIFICATIONS_EMAIL_USERNAME"))
	setIfNotEmpty(&cfg.Notifications.Email.Password, os.Getenv("KIMBAP_NOTIFICATIONS_EMAIL_PASSWORD"))

	setIfNotEmpty(&cfg.Notifications.Webhook.URL, os.Getenv("KIMBAP_NOTIFICATIONS_WEBHOOK_URL"))
	setIfNotEmpty(&cfg.Notifications.Webhook.SignKey, os.Getenv("KIMBAP_NOTIFICATIONS_WEBHOOK_SIGN_KEY"))
}

func mergeConfig(dst, src *KimbapConfig) {
	setIfNotEmpty(&dst.Mode, src.Mode)
	setIfNotEmpty(&dst.DataDir, src.DataDir)
	setIfNotEmpty(&dst.ListenAddr, src.ListenAddr)
	setIfNotEmpty(&dst.ProxyAddr, src.ProxyAddr)
	setIfNotEmpty(&dst.LogLevel, src.LogLevel)
	setIfNotEmpty(&dst.LogFormat, src.LogFormat)
	if src.Console.Enabled {
		dst.Console.Enabled = true
	}
	if len(src.Aliases) > 0 {
		if dst.Aliases == nil {
			dst.Aliases = make(map[string]string, len(src.Aliases))
		}
		for k, v := range src.Aliases {
			dst.Aliases[k] = v
		}
	}

	setIfNotEmpty(&dst.Vault.Backend, src.Vault.Backend)
	setIfNotEmpty(&dst.Vault.Path, src.Vault.Path)

	setIfNotEmpty(&dst.Auth.TokenTTL, src.Auth.TokenTTL)
	setIfNotEmpty(&dst.Auth.SessionTTL, src.Auth.SessionTTL)
	setIfNotEmpty(&dst.Auth.ServerURL, src.Auth.ServerURL)

	setIfNotEmpty(&dst.Audit.Sink, src.Audit.Sink)
	setIfNotEmpty(&dst.Audit.Path, src.Audit.Path)

	setIfNotEmpty(&dst.Policy.Path, src.Policy.Path)

	setIfNotEmpty(&dst.Services.Dir, src.Services.Dir)
	setIfNotEmpty(&dst.Services.RegistryURL, src.Services.RegistryURL)
	setIfNotEmpty(&dst.Services.Verify, src.Services.Verify)
	setIfNotEmpty(&dst.Services.SignaturePolicy, src.Services.SignaturePolicy)

	setIfNotEmpty(&dst.Database.Driver, src.Database.Driver)
	setIfNotEmpty(&dst.Database.DSN, src.Database.DSN)

	setIfNotEmpty(&dst.Notifications.Slack.WebhookURL, src.Notifications.Slack.WebhookURL)

	setIfNotEmpty(&dst.Notifications.Telegram.BotToken, src.Notifications.Telegram.BotToken)
	setIfNotEmpty(&dst.Notifications.Telegram.ChatID, src.Notifications.Telegram.ChatID)

	setIfNotEmpty(&dst.Notifications.Email.SMTPHost, src.Notifications.Email.SMTPHost)
	if src.Notifications.Email.SMTPPort != 0 {
		dst.Notifications.Email.SMTPPort = src.Notifications.Email.SMTPPort
	}
	setIfNotEmpty(&dst.Notifications.Email.From, src.Notifications.Email.From)
	if len(src.Notifications.Email.To) > 0 {
		dst.Notifications.Email.To = src.Notifications.Email.To
	}
	setIfNotEmpty(&dst.Notifications.Email.Username, src.Notifications.Email.Username)
	setIfNotEmpty(&dst.Notifications.Email.Password, src.Notifications.Email.Password)

	setIfNotEmpty(&dst.Notifications.Webhook.URL, src.Notifications.Webhook.URL)
	setIfNotEmpty(&dst.Notifications.Webhook.SignKey, src.Notifications.Webhook.SignKey)

}

func setIfNotEmpty(dst *string, value string) {
	if value != "" {
		*dst = value
	}
}

func normalizeServiceVerifyMode(mode string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "", "warn":
		return "warn", nil
	case "off", "strict":
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid services.verify value %q: must be one of: off, warn, strict", mode)
	}
}

func normalizeServiceSignaturePolicy(policy string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(policy))
	switch normalized {
	case "", "optional":
		return "optional", nil
	case "off", "required":
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid services.signature_policy value %q: must be one of: off, optional, required", policy)
	}
}

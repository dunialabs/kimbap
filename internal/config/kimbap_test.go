package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfigHasCoreDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Mode != "embedded" {
		t.Fatalf("unexpected mode default: %q", cfg.Mode)
	}
	if cfg.Console.Enabled {
		t.Fatal("expected console to be disabled by default")
	}
	if cfg.Auth.TokenTTL != "720h" {
		t.Fatalf("unexpected auth token ttl default: %q", cfg.Auth.TokenTTL)
	}
	if cfg.Auth.SessionTTL != "15m" {
		t.Fatalf("unexpected session ttl default: %q", cfg.Auth.SessionTTL)
	}
}

func TestLoadKimbapConfigAppliesConsoleEnabledFromEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_CONSOLE_ENABLED", "true")

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("LoadKimbapConfigWithoutDefault: %v", err)
	}
	if !cfg.Console.Enabled {
		t.Fatal("expected console enabled from env")
	}
}

func TestLoadKimbapConfigExplicitConsoleFalseOverridesEnvTrue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_CONSOLE_ENABLED", "true")

	explicitPath := filepath.Join(t.TempDir(), "override.yaml")
	if err := os.WriteFile(explicitPath, []byte("console:\n  enabled: false\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(explicitPath)
	if err != nil {
		t.Fatalf("LoadKimbapConfigWithoutDefault: %v", err)
	}
	if cfg.Console.Enabled {
		t.Fatal("expected explicit config to disable console")
	}
}

func TestLoadKimbapConfigPrecedenceDefaultEnvExplicit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultConfigDir := filepath.Join(home, ".kimbap")
	if err := os.MkdirAll(defaultConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir default config dir: %v", err)
	}

	defaultPath := filepath.Join(defaultConfigDir, "config.yaml")
	if err := os.WriteFile(defaultPath, []byte("mode: connected\nauth:\n  token_ttl: 48h\nlog_level: warn\n"), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	t.Setenv("KIMBAP_MODE", "embedded")
	t.Setenv("KIMBAP_LOG_LEVEL", "error")

	explicitPath := filepath.Join(t.TempDir(), "override.yaml")
	if err := os.WriteFile(explicitPath, []byte("mode: connected\nlog_level: debug\nauth:\n  token_ttl: 24h\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfig(explicitPath)
	if err != nil {
		t.Fatalf("load kimbap config: %v", err)
	}

	if cfg.Mode != "connected" {
		t.Fatalf("explicit file should win for mode, got %q", cfg.Mode)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("explicit file should win for log level, got %q", cfg.LogLevel)
	}
	if cfg.Auth.TokenTTL != "24h" {
		t.Fatalf("explicit file should win for auth token ttl, got %q", cfg.Auth.TokenTTL)
	}
}

func TestResolveConfigPathUsesExplicitPath(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "explicit.yaml")
	path, err := ResolveConfigPath(explicit)
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if path != explicit {
		t.Fatalf("expected explicit path %q, got %q", explicit, path)
	}
}

func TestResolveConfigPathUsesKimbapConfigEnvWhenExplicitMissing(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "env-config.yaml")
	t.Setenv("KIMBAP_CONFIG", envPath)

	path, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if path != envPath {
		t.Fatalf("expected KIMBAP_CONFIG path %q, got %q", envPath, path)
	}
}

func TestResolveConfigPathExplicitOverridesKimbapConfigEnv(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "explicit.yaml")
	t.Setenv("KIMBAP_CONFIG", filepath.Join(t.TempDir(), "env-config.yaml"))

	path, err := ResolveConfigPath(explicit)
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if path != explicit {
		t.Fatalf("expected explicit path %q, got %q", explicit, path)
	}
}

func TestResolveConfigPathWithDataDirUsesDataDirWhenExplicitMissing(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "isolated-data")
	path, err := ResolveConfigPathWithDataDir("", dataDir)
	if err != nil {
		t.Fatalf("ResolveConfigPathWithDataDir: %v", err)
	}
	want := filepath.Join(dataDir, "config.yaml")
	if path != want {
		t.Fatalf("expected data-dir config path %q, got %q", want, path)
	}
}

func TestResolveConfigPathWithDataDirPrefersExplicitPath(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "explicit.yaml")
	dataDir := filepath.Join(t.TempDir(), "isolated-data")
	path, err := ResolveConfigPathWithDataDir(explicit, dataDir)
	if err != nil {
		t.Fatalf("ResolveConfigPathWithDataDir: %v", err)
	}
	if path != explicit {
		t.Fatalf("expected explicit path %q, got %q", explicit, path)
	}
}

func TestApplyDataDirOverrideRebasesDerivedDefaults(t *testing.T) {
	cfg := DefaultConfig()
	override := filepath.Join(t.TempDir(), "override-data")
	ApplyDataDirOverride(cfg, override)

	if cfg.DataDir != override {
		t.Fatalf("expected data dir %q, got %q", override, cfg.DataDir)
	}
	if cfg.Vault.Path != filepath.Join(override, "vault.db") {
		t.Fatalf("expected vault path rebased, got %q", cfg.Vault.Path)
	}
	if cfg.Services.Dir != filepath.Join(override, "services") {
		t.Fatalf("expected services dir rebased, got %q", cfg.Services.Dir)
	}
	if cfg.Database.DSN != filepath.Join(override, "kimbap.db") {
		t.Fatalf("expected database dsn rebased, got %q", cfg.Database.DSN)
	}
}

func TestApplyDataDirOverridePreservesExplicitNonDefaultPaths(t *testing.T) {
	cfg := DefaultConfig()
	customVault := filepath.Join(t.TempDir(), "custom-vault.db")
	cfg.Vault.Path = customVault
	override := filepath.Join(t.TempDir(), "override-data")
	ApplyDataDirOverride(cfg, override)

	if cfg.Vault.Path != customVault {
		t.Fatalf("expected custom vault path preserved, got %q", cfg.Vault.Path)
	}
	if cfg.Policy.Path != filepath.Join(override, "policy.yaml") {
		t.Fatalf("expected policy path rebased, got %q", cfg.Policy.Path)
	}
}

func TestDefaultKimbapConfigPathPrefersExistingXDGPath(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	xdgPath := filepath.Join(xdg, "kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(xdgPath), 0o755); err != nil {
		t.Fatalf("mkdir xdg config dir: %v", err)
	}
	if err := os.WriteFile(xdgPath, []byte("mode: embedded\n"), 0o644); err != nil {
		t.Fatalf("write xdg config file: %v", err)
	}

	path, err := defaultKimbapConfigPath()
	if err != nil {
		t.Fatalf("defaultKimbapConfigPath: %v", err)
	}
	if path != xdgPath {
		t.Fatalf("expected xdg path %q, got %q", xdgPath, path)
	}
}

func TestDefaultKimbapConfigPathFallsBackToLegacyWhenXDGMissing(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	legacyPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("mkdir legacy config dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("mode: embedded\n"), 0o644); err != nil {
		t.Fatalf("write legacy config file: %v", err)
	}

	path, err := defaultKimbapConfigPath()
	if err != nil {
		t.Fatalf("defaultKimbapConfigPath: %v", err)
	}
	if path != legacyPath {
		t.Fatalf("expected legacy path %q, got %q", legacyPath, path)
	}
}

func TestDefaultKimbapConfigPathIgnoresDirectoryEntries(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	xdgPath := filepath.Join(xdg, "kimbap", "config.yaml")
	if err := os.MkdirAll(xdgPath, 0o755); err != nil {
		t.Fatalf("mkdir xdg config directory entry: %v", err)
	}

	legacyPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("mkdir legacy config dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("mode: embedded\n"), 0o644); err != nil {
		t.Fatalf("write legacy config file: %v", err)
	}

	path, err := defaultKimbapConfigPath()
	if err != nil {
		t.Fatalf("defaultKimbapConfigPath: %v", err)
	}
	if path != legacyPath {
		t.Fatalf("expected legacy file path %q, got %q", legacyPath, path)
	}
}

func TestDefaultKimbapConfigPathErrorsWhenXDGEntryIsDirectoryWithoutLegacy(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	xdgPath := filepath.Join(xdg, "kimbap", "config.yaml")
	if err := os.MkdirAll(xdgPath, 0o755); err != nil {
		t.Fatalf("mkdir xdg config directory entry: %v", err)
	}

	_, err := defaultKimbapConfigPath()
	if err == nil {
		t.Fatal("expected error when xdg config path is a directory and no legacy file exists")
	}
}

func TestDefaultKimbapConfigPathReturnsXDGPathWhenXDGMissingAndLegacyMissing(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	path, err := defaultKimbapConfigPath()
	if err != nil {
		t.Fatalf("defaultKimbapConfigPath: %v", err)
	}
	expected := filepath.Join(xdg, "kimbap", "config.yaml")
	if path != expected {
		t.Fatalf("expected xdg path %q when both files are missing, got %q", expected, path)
	}
}

func TestLoadKimbapConfigWithoutDefaultIgnoresBrokenDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultPath := filepath.Join(home, ".kimbap", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("mkdir default config dir: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte("mode: [\n"), 0o644); err != nil {
		t.Fatalf("write broken default config: %v", err)
	}

	explicitPath := filepath.Join(t.TempDir(), "explicit.yaml")
	if err := os.WriteFile(explicitPath, []byte("mode: connected\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(explicitPath)
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}
	if cfg.Mode != "connected" {
		t.Fatalf("expected explicit config mode, got %q", cfg.Mode)
	}
}

func TestLoadKimbapConfigRebasesDerivedPathsWhenExplicitDataDirChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dataDir := filepath.Join(t.TempDir(), "custom-data")
	explicitPath := filepath.Join(t.TempDir(), "explicit.yaml")
	if err := os.WriteFile(explicitPath, []byte("data_dir: "+dataDir+"\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(explicitPath)
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}

	if cfg.Vault.Path != filepath.Join(dataDir, "vault.db") {
		t.Fatalf("expected rebased vault path, got %q", cfg.Vault.Path)
	}
	if cfg.Audit.Path != filepath.Join(dataDir, "audit.jsonl") {
		t.Fatalf("expected rebased audit path, got %q", cfg.Audit.Path)
	}
	if cfg.Policy.Path != filepath.Join(dataDir, "policy.yaml") {
		t.Fatalf("expected rebased policy path, got %q", cfg.Policy.Path)
	}
	if cfg.Services.Dir != filepath.Join(dataDir, "services") {
		t.Fatalf("expected rebased services dir, got %q", cfg.Services.Dir)
	}
	if cfg.Database.DSN != filepath.Join(dataDir, "kimbap.db") {
		t.Fatalf("expected rebased database dsn, got %q", cfg.Database.DSN)
	}
}

func TestLoadKimbapConfigRebasesDerivedPathsWhenEnvDataDirChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dataDir := filepath.Join(t.TempDir(), "env-data")
	t.Setenv("KIMBAP_DATA_DIR", dataDir)

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}

	if cfg.Vault.Path != filepath.Join(dataDir, "vault.db") {
		t.Fatalf("expected rebased vault path, got %q", cfg.Vault.Path)
	}
	if cfg.Services.Dir != filepath.Join(dataDir, "services") {
		t.Fatalf("expected rebased services dir, got %q", cfg.Services.Dir)
	}
}

func TestLoadKimbapConfigPreservesExplicitPathOverridesWhenDataDirChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dataDir := filepath.Join(t.TempDir(), "custom-data")
	customVaultPath := filepath.Join(t.TempDir(), "custom-vault.db")
	explicitPath := filepath.Join(t.TempDir(), "explicit.yaml")
	content := "data_dir: " + dataDir + "\n" +
		"vault:\n" +
		"  path: " + customVaultPath + "\n"
	if err := os.WriteFile(explicitPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(explicitPath)
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}

	if cfg.Vault.Path != customVaultPath {
		t.Fatalf("expected custom vault path preserved, got %q", cfg.Vault.Path)
	}
	if cfg.Policy.Path != filepath.Join(dataDir, "policy.yaml") {
		t.Fatalf("expected policy path rebased, got %q", cfg.Policy.Path)
	}
}

func TestLoadKimbapConfigAppliesServicesEnvOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_SERVICES_DIR", "/tmp/custom-services")
	t.Setenv("KIMBAP_SERVICES_REGISTRY_URL", "https://services.example.com")
	t.Setenv("KIMBAP_SERVICES_VERIFY", "strict")
	t.Setenv("KIMBAP_SERVICES_SIGNATURE_POLICY", "required")
	t.Setenv("KIMBAP_SERVICES_APPLESCRIPT_REGISTRY_MODE", "manifest")

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}

	if cfg.Services.Dir != "/tmp/custom-services" {
		t.Fatalf("expected services dir override, got %q", cfg.Services.Dir)
	}
	if cfg.Services.RegistryURL != "https://services.example.com" {
		t.Fatalf("expected services registry_url override, got %q", cfg.Services.RegistryURL)
	}
	if cfg.Services.Verify != "strict" {
		t.Fatalf("expected services verify override, got %q", cfg.Services.Verify)
	}
	if cfg.Services.SignaturePolicy != "required" {
		t.Fatalf("expected services signature policy override, got %q", cfg.Services.SignaturePolicy)
	}
	if cfg.Services.AppleScriptRegistryMode != "manifest" {
		t.Fatalf("expected services applescript_registry_mode override, got %q", cfg.Services.AppleScriptRegistryMode)
	}
}

func TestLoadKimbapConfigRejectsInvalidAppleScriptRegistryMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_SERVICES_APPLESCRIPT_REGISTRY_MODE", "fast-and-loose")

	_, err := LoadKimbapConfigWithoutDefault()
	if err == nil {
		t.Fatal("expected invalid services.applescript_registry_mode error, got nil")
	}
	if !strings.Contains(err.Error(), "applescript_registry_mode") {
		t.Fatalf("expected applescript_registry_mode in error, got %v", err)
	}
}

func TestLoadKimbapConfigIgnoresUnknownLegacyEnvOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_LEGACY_SERVICES_DIR", "/tmp/legacy-services")

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("load config without default: %v", err)
	}

	if cfg.Services.Dir == "/tmp/legacy-services" {
		t.Fatalf("expected unknown legacy env var to be ignored")
	}
}

func TestLoadKimbapConfigRejectsLegacySkillsKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	content := "skills:\n  dir: /tmp/old-skills\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadKimbapConfigWithoutDefault(cfgPath)
	if err == nil {
		t.Fatal("expected error for legacy 'skills:' key, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") || !strings.Contains(err.Error(), "skills") {
		t.Errorf("expected unsupported legacy skills key error, got: %v", err)
	}
}

func TestLoadKimbapConfigWithoutDefault_AllowsClearingAliasesWithExplicitEmptyMap(t *testing.T) {
	base := filepath.Join(t.TempDir(), "base.yaml")
	override := filepath.Join(t.TempDir(), "override.yaml")

	if err := os.WriteFile(base, []byte("aliases:\n  gh: github\n  ls: list\n"), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	if err := os.WriteFile(override, []byte("aliases: {}\n"), 0o644); err != nil {
		t.Fatalf("write override config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(base, override)
	if err != nil {
		t.Fatalf("LoadKimbapConfigWithoutDefault: %v", err)
	}
	if len(cfg.Aliases) != 0 {
		t.Fatalf("expected aliases to be cleared by explicit empty map, got %+v", cfg.Aliases)
	}
}

func TestLoadKimbapConfigWithoutDefault_AllowsClearingCommandAliasesWithExplicitEmptyMap(t *testing.T) {
	base := filepath.Join(t.TempDir(), "base.yaml")
	override := filepath.Join(t.TempDir(), "override.yaml")

	if err := os.WriteFile(base, []byte("command_aliases:\n  geosearch: open-meteo-geocoding.search\n"), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	if err := os.WriteFile(override, []byte("command_aliases: {}\n"), 0o644); err != nil {
		t.Fatalf("write override config: %v", err)
	}

	cfg, err := LoadKimbapConfigWithoutDefault(base, override)
	if err != nil {
		t.Fatalf("LoadKimbapConfigWithoutDefault: %v", err)
	}
	if len(cfg.CommandAliases) != 0 {
		t.Fatalf("expected command_aliases to be cleared by explicit empty map, got %+v", cfg.CommandAliases)
	}
}

func TestNotificationConfigYAMLParsing(t *testing.T) {
	yamlContent := `
notifications:
  slack:
    webhook_url: https://hooks.slack.com/test
  telegram:
    bot_token: bot123
    chat_id: "-100456"
  email:
    smtp_host: smtp.example.com
    smtp_port: 587
    from: sender@example.com
    to:
      - alice@example.com
      - bob@example.com
    username: user
    password: pass
  webhook:
    url: https://webhook.example.com/hook
    sign_key: mysecret
`
	var cfg KimbapConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if cfg.Notifications.Slack.WebhookURL != "https://hooks.slack.com/test" {
		t.Errorf("slack.webhook_url: got %q", cfg.Notifications.Slack.WebhookURL)
	}
	if cfg.Notifications.Telegram.BotToken != "bot123" {
		t.Errorf("telegram.bot_token: got %q", cfg.Notifications.Telegram.BotToken)
	}
	if cfg.Notifications.Telegram.ChatID != "-100456" {
		t.Errorf("telegram.chat_id: got %q", cfg.Notifications.Telegram.ChatID)
	}
	if cfg.Notifications.Email.SMTPHost != "smtp.example.com" {
		t.Errorf("email.smtp_host: got %q", cfg.Notifications.Email.SMTPHost)
	}
	if cfg.Notifications.Email.SMTPPort != 587 {
		t.Errorf("email.smtp_port: got %d", cfg.Notifications.Email.SMTPPort)
	}
	if cfg.Notifications.Email.From != "sender@example.com" {
		t.Errorf("email.from: got %q", cfg.Notifications.Email.From)
	}
	if len(cfg.Notifications.Email.To) != 2 || cfg.Notifications.Email.To[0] != "alice@example.com" || cfg.Notifications.Email.To[1] != "bob@example.com" {
		t.Errorf("email.to: got %v", cfg.Notifications.Email.To)
	}
	if cfg.Notifications.Email.Username != "user" {
		t.Errorf("email.username: got %q", cfg.Notifications.Email.Username)
	}
	if cfg.Notifications.Email.Password != "pass" {
		t.Errorf("email.password: got %q", cfg.Notifications.Email.Password)
	}
	if cfg.Notifications.Webhook.URL != "https://webhook.example.com/hook" {
		t.Errorf("webhook.url: got %q", cfg.Notifications.Webhook.URL)
	}
	if cfg.Notifications.Webhook.SignKey != "mysecret" {
		t.Errorf("webhook.sign_key: got %q", cfg.Notifications.Webhook.SignKey)
	}
}

func TestNotificationEnvVarSlack(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_NOTIFICATIONS_SLACK_WEBHOOK_URL", "https://hooks.slack.com/env-test")

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("LoadKimbapConfigWithoutDefault: %v", err)
	}
	if cfg.Notifications.Slack.WebhookURL != "https://hooks.slack.com/env-test" {
		t.Errorf("slack webhook url: got %q, want %q", cfg.Notifications.Slack.WebhookURL, "https://hooks.slack.com/env-test")
	}
}

func TestNotificationEnvVarTelegram(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_NOTIFICATIONS_TELEGRAM_BOT_TOKEN", "token-abc")
	t.Setenv("KIMBAP_NOTIFICATIONS_TELEGRAM_CHAT_ID", "-100789")

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("LoadKimbapConfigWithoutDefault: %v", err)
	}
	if cfg.Notifications.Telegram.BotToken != "token-abc" {
		t.Errorf("telegram bot token: got %q, want %q", cfg.Notifications.Telegram.BotToken, "token-abc")
	}
	if cfg.Notifications.Telegram.ChatID != "-100789" {
		t.Errorf("telegram chat id: got %q, want %q", cfg.Notifications.Telegram.ChatID, "-100789")
	}
}

func TestNotificationEnvVarEmail(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_NOTIFICATIONS_EMAIL_SMTP_HOST", "smtp.test.com")
	t.Setenv("KIMBAP_NOTIFICATIONS_EMAIL_SMTP_PORT", "465")
	t.Setenv("KIMBAP_NOTIFICATIONS_EMAIL_FROM", "from@test.com")
	t.Setenv("KIMBAP_NOTIFICATIONS_EMAIL_TO", "a@b.com,c@d.com")
	t.Setenv("KIMBAP_NOTIFICATIONS_EMAIL_USERNAME", "testuser")
	t.Setenv("KIMBAP_NOTIFICATIONS_EMAIL_PASSWORD", "testpass")

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("LoadKimbapConfigWithoutDefault: %v", err)
	}
	if cfg.Notifications.Email.SMTPHost != "smtp.test.com" {
		t.Errorf("email smtp host: got %q", cfg.Notifications.Email.SMTPHost)
	}
	if cfg.Notifications.Email.SMTPPort != 465 {
		t.Errorf("email smtp port: got %d, want 465", cfg.Notifications.Email.SMTPPort)
	}
	if cfg.Notifications.Email.From != "from@test.com" {
		t.Errorf("email from: got %q", cfg.Notifications.Email.From)
	}
	if len(cfg.Notifications.Email.To) != 2 || cfg.Notifications.Email.To[0] != "a@b.com" || cfg.Notifications.Email.To[1] != "c@d.com" {
		t.Errorf("email to: got %v, want [a@b.com c@d.com]", cfg.Notifications.Email.To)
	}
	if cfg.Notifications.Email.Username != "testuser" {
		t.Errorf("email username: got %q", cfg.Notifications.Email.Username)
	}
	if cfg.Notifications.Email.Password != "testpass" {
		t.Errorf("email password: got %q", cfg.Notifications.Email.Password)
	}
}

func TestNotificationEnvVarWebhook(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KIMBAP_NOTIFICATIONS_WEBHOOK_URL", "https://webhook.test.com/hook")
	t.Setenv("KIMBAP_NOTIFICATIONS_WEBHOOK_SIGN_KEY", "signkey123")

	cfg, err := LoadKimbapConfigWithoutDefault()
	if err != nil {
		t.Fatalf("LoadKimbapConfigWithoutDefault: %v", err)
	}
	if cfg.Notifications.Webhook.URL != "https://webhook.test.com/hook" {
		t.Errorf("webhook url: got %q", cfg.Notifications.Webhook.URL)
	}
	if cfg.Notifications.Webhook.SignKey != "signkey123" {
		t.Errorf("webhook sign key: got %q", cfg.Notifications.Webhook.SignKey)
	}
}

func TestNotificationConfigEmpty(t *testing.T) {
	var cfg KimbapConfig
	if err := yaml.Unmarshal([]byte("mode: embedded\n"), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if cfg.Notifications.Slack.WebhookURL != "" {
		t.Errorf("expected empty slack webhook url, got %q", cfg.Notifications.Slack.WebhookURL)
	}
	if cfg.Notifications.Telegram.BotToken != "" {
		t.Errorf("expected empty telegram bot token, got %q", cfg.Notifications.Telegram.BotToken)
	}
	if cfg.Notifications.Email.SMTPHost != "" {
		t.Errorf("expected empty email smtp host, got %q", cfg.Notifications.Email.SMTPHost)
	}
	if cfg.Notifications.Email.SMTPPort != 0 {
		t.Errorf("expected 0 email smtp port, got %d", cfg.Notifications.Email.SMTPPort)
	}
	if len(cfg.Notifications.Email.To) != 0 {
		t.Errorf("expected empty email to, got %v", cfg.Notifications.Email.To)
	}
	if cfg.Notifications.Webhook.URL != "" {
		t.Errorf("expected empty webhook url, got %q", cfg.Notifications.Webhook.URL)
	}
}

func TestGetReverseRequestTimeoutTrimsRequestType(t *testing.T) {
	if got := GetReverseRequestTimeout(" sampling "); got != REVERSE_REQUEST_TIMEOUTS["sampling"] {
		t.Fatalf("expected sampling timeout %d, got %d", REVERSE_REQUEST_TIMEOUTS["sampling"], got)
	}
}

func TestGetReverseRequestTimeoutUsesTrimmedEnvValue(t *testing.T) {
	t.Setenv("REVERSE_REQUEST_TIMEOUT_SAMPLING", " 12345 ")

	if got := GetReverseRequestTimeout("sampling"); got != 12345 {
		t.Fatalf("expected env override 12345, got %d", got)
	}
}

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type KimbapConfig struct {
	Mode       string `yaml:"mode"`
	DataDir    string `yaml:"data_dir"`
	ListenAddr string `yaml:"listen_addr"`
	ProxyAddr  string `yaml:"proxy_addr"`
	LogLevel   string `yaml:"log_level"`
	LogFormat  string `yaml:"log_format"`

	Vault    VaultConfig    `yaml:"vault"`
	Auth     AuthConfig     `yaml:"auth"`
	Audit    AuditConfig    `yaml:"audit"`
	Policy   PolicyConfig   `yaml:"policy"`
	Skills   SkillsConfig   `yaml:"skills"`
	Database DatabaseConfig `yaml:"database"`
}

type VaultConfig struct {
	Backend string `yaml:"backend"`
	Path    string `yaml:"path"`
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

type SkillsConfig struct {
	Dir             string `yaml:"dir"`
	Official        string `yaml:"official"`
	Verify          string `yaml:"verify"`
	SignaturePolicy string `yaml:"signature_policy"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
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
		ListenAddr: ":3002",
		ProxyAddr:  ":7788",
		LogLevel:   "info",
		LogFormat:  "text",
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
		Skills: SkillsConfig{
			Dir:             filepath.Join(dataDir, "skills"),
			Official:        "https://skills.kimbap.ai",
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
	cfg := DefaultConfig()

	defaultPath, err := defaultKimbapConfigPath()
	if err != nil {
		return nil, err
	}

	if err := mergeConfigFromFile(cfg, defaultPath, false); err != nil {
		return nil, err
	}
	applyKimbapEnv(cfg)

	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := mergeConfigFromFile(cfg, p, true); err != nil {
			return nil, err
		}
	}

	cfg.Skills.Verify = normalizeSkillVerifyMode(cfg.Skills.Verify)
	cfg.Skills.SignaturePolicy = normalizeSkillSignaturePolicy(cfg.Skills.SignaturePolicy)

	return cfg, nil
}

func defaultKimbapConfigPath() (string, error) {
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

	mergeConfig(cfg, &loaded)
	return nil
}

func applyKimbapEnv(cfg *KimbapConfig) {
	setIfNotEmpty(&cfg.Mode, os.Getenv("KIMBAP_MODE"))
	setIfNotEmpty(&cfg.DataDir, os.Getenv("KIMBAP_DATA_DIR"))
	setIfNotEmpty(&cfg.ListenAddr, os.Getenv("KIMBAP_LISTEN_ADDR"))
	setIfNotEmpty(&cfg.ProxyAddr, os.Getenv("KIMBAP_PROXY_ADDR"))
	setIfNotEmpty(&cfg.LogLevel, os.Getenv("KIMBAP_LOG_LEVEL"))
	setIfNotEmpty(&cfg.LogFormat, os.Getenv("KIMBAP_LOG_FORMAT"))

	setIfNotEmpty(&cfg.Vault.Backend, os.Getenv("KIMBAP_VAULT_BACKEND"))
	setIfNotEmpty(&cfg.Vault.Path, os.Getenv("KIMBAP_VAULT_PATH"))

	setIfNotEmpty(&cfg.Auth.TokenTTL, os.Getenv("KIMBAP_AUTH_TOKEN_TTL"))
	setIfNotEmpty(&cfg.Auth.SessionTTL, os.Getenv("KIMBAP_AUTH_SESSION_TTL"))
	setIfNotEmpty(&cfg.Auth.ServerURL, os.Getenv("KIMBAP_AUTH_SERVER_URL"))

	setIfNotEmpty(&cfg.Audit.Sink, os.Getenv("KIMBAP_AUDIT_SINK"))
	setIfNotEmpty(&cfg.Audit.Path, os.Getenv("KIMBAP_AUDIT_PATH"))

	setIfNotEmpty(&cfg.Policy.Path, os.Getenv("KIMBAP_POLICY_PATH"))

	setIfNotEmpty(&cfg.Skills.Dir, os.Getenv("KIMBAP_SKILLS_DIR"))
	setIfNotEmpty(&cfg.Skills.Official, os.Getenv("KIMBAP_SKILLS_OFFICIAL"))
	setIfNotEmpty(&cfg.Skills.Verify, os.Getenv("KIMBAP_SKILLS_VERIFY"))
	setIfNotEmpty(&cfg.Skills.SignaturePolicy, os.Getenv("KIMBAP_SKILLS_SIGNATURE_POLICY"))

	setIfNotEmpty(&cfg.Database.Driver, os.Getenv("KIMBAP_DATABASE_DRIVER"))
	setIfNotEmpty(&cfg.Database.DSN, os.Getenv("KIMBAP_DATABASE_DSN"))
}

func mergeConfig(dst, src *KimbapConfig) {
	setIfNotEmpty(&dst.Mode, src.Mode)
	setIfNotEmpty(&dst.DataDir, src.DataDir)
	setIfNotEmpty(&dst.ListenAddr, src.ListenAddr)
	setIfNotEmpty(&dst.ProxyAddr, src.ProxyAddr)
	setIfNotEmpty(&dst.LogLevel, src.LogLevel)
	setIfNotEmpty(&dst.LogFormat, src.LogFormat)

	setIfNotEmpty(&dst.Vault.Backend, src.Vault.Backend)
	setIfNotEmpty(&dst.Vault.Path, src.Vault.Path)

	setIfNotEmpty(&dst.Auth.TokenTTL, src.Auth.TokenTTL)
	setIfNotEmpty(&dst.Auth.SessionTTL, src.Auth.SessionTTL)
	setIfNotEmpty(&dst.Auth.ServerURL, src.Auth.ServerURL)

	setIfNotEmpty(&dst.Audit.Sink, src.Audit.Sink)
	setIfNotEmpty(&dst.Audit.Path, src.Audit.Path)

	setIfNotEmpty(&dst.Policy.Path, src.Policy.Path)

	setIfNotEmpty(&dst.Skills.Dir, src.Skills.Dir)
	setIfNotEmpty(&dst.Skills.Official, src.Skills.Official)
	setIfNotEmpty(&dst.Skills.Verify, src.Skills.Verify)
	setIfNotEmpty(&dst.Skills.SignaturePolicy, src.Skills.SignaturePolicy)

	setIfNotEmpty(&dst.Database.Driver, src.Database.Driver)
	setIfNotEmpty(&dst.Database.DSN, src.Database.DSN)
}

func setIfNotEmpty(dst *string, value string) {
	if value != "" {
		*dst = value
	}
}

func normalizeSkillVerifyMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "off", "strict", "warn":
		return normalized
	default:
		return "warn"
	}
}

func normalizeSkillSignaturePolicy(policy string) string {
	normalized := strings.ToLower(strings.TrimSpace(policy))
	switch normalized {
	case "off", "optional", "required":
		return normalized
	default:
		return "optional"
	}
}

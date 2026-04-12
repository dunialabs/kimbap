package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
)

func loadAppConfig() (*config.KimbapConfig, error) {
	cfg, err := loadAppConfigReadOnly()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return cfg, nil
}

func loadAppConfigReadOnly() (*config.KimbapConfig, error) {
	cfg, err := loadBaseConfigForCLI()
	if err != nil {
		return nil, err
	}

	config.ApplyDataDirOverride(cfg, opts.dataDir)
	if strings.TrimSpace(opts.logLevel) != "" {
		cfg.LogLevel = opts.logLevel
	}
	if strings.TrimSpace(opts.mode) != "" {
		cfg.Mode = opts.mode
	}

	if err := services.SetAppleScriptRegistryMode(cfg.Services.AppleScriptRegistryMode); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadBaseConfigForCLI() (*config.KimbapConfig, error) {
	var (
		cfg *config.KimbapConfig
		err error
	)
	explicitConfigProvided := strings.TrimSpace(opts.configPath) != ""
	if !explicitConfigProvided && strings.TrimSpace(opts.dataDir) == "" {
		cfg, err = config.LoadKimbapConfig()
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	configPath, pathErr := resolveConfigPath()
	if pathErr != nil {
		return nil, pathErr
	}
	st, statErr := os.Stat(configPath)
	if statErr == nil {
		if st.IsDir() {
			return nil, fmt.Errorf("config path is a directory: %s", configPath)
		}
		cfg, err = config.LoadKimbapConfigWithoutDefault(configPath)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}
	if os.IsNotExist(statErr) {
		if explicitConfigProvided {
			return nil, fmt.Errorf("read config file %q: %w", configPath, statErr)
		}
		cfg, err = config.LoadKimbapConfigWithoutDefault()
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}
	return nil, fmt.Errorf("stat config path %q: %w", configPath, statErr)
}

func outputAsJSON() bool {
	return strings.EqualFold(strings.TrimSpace(opts.format), "json")
}

func successCheck() string {
	if isColorStdout() {
		return "\x1b[32m✓\x1b[0m"
	}
	return "✓"
}

func isColorStdout() bool {
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func isDryRun() bool {
	return opts.dryRun
}

func isTrace() bool {
	return opts.trace
}

func printOutput(v any) error {
	if outputAsJSON() {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	if s, ok := v.(string); ok {
		_, err := fmt.Fprintln(os.Stdout, s)
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(b))
	return err
}

func defaultTenantID() string {
	if tenant := strings.TrimSpace(os.Getenv("KIMBAP_TENANT_ID")); tenant != "" {
		return tenant
	}
	return "default"
}

func installerFromConfig(cfg *config.KimbapConfig) *services.LocalInstaller {
	return services.NewLocalInstaller(cfg.Services.Dir)
}

func loadEnabledInstalledServices(cfg *config.KimbapConfig) ([]services.InstalledService, error) {
	return installerFromConfig(cfg).ListEnabled()
}

func loadInstalledActions(cfg *config.KimbapConfig) ([]actions.ActionDefinition, error) {
	installed, err := loadEnabledInstalledServices(cfg)
	if err != nil {
		return nil, err
	}

	out := make([]actions.ActionDefinition, 0)
	for _, installedService := range installed {
		defs, convErr := services.ToActionDefinitions(&installedService.Manifest)
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, defs...)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func decodeMasterKeyHexEnv() ([]byte, error, bool) {
	hexKey := strings.TrimSpace(os.Getenv("KIMBAP_MASTER_KEY_HEX"))
	if hexKey == "" {
		return nil, nil, false
	}
	decoded, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode KIMBAP_MASTER_KEY_HEX: %w", err), true
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("KIMBAP_MASTER_KEY_HEX must decode to 32 bytes"), true
	}
	return decoded, nil, true
}

func readPersistedDevMasterKey(path string) ([]byte, error) {
	existing, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, readErr
	}
	if len(existing) == 32 {
		return existing, nil
	}
	for range 10 {
		time.Sleep(50 * time.Millisecond)
		existing, readErr = os.ReadFile(path)
		if readErr == nil && len(existing) == 32 {
			return existing, nil
		}
	}
	if readErr != nil {
		return nil, readErr
	}
	return nil, fmt.Errorf("invalid dev master key at %s: expected 32 bytes, got %d", path, len(existing))
}

func parseJSONMap(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func sanitizeDisplayURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return rawURL
	}
	u.User = nil
	return u.String()
}

func contextBackground() context.Context {
	return context.Background()
}

func commandContext(cmd *cobra.Command) context.Context {
	if cmd != nil {
		if ctx := cmd.Context(); ctx != nil {
			return ctx
		}
	}
	return contextBackground()
}

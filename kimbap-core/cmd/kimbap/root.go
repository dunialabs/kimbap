package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/app"
	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/audit"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	corecrypto "github.com/dunialabs/kimbap-core/internal/crypto"
	"github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/dunialabs/kimbap-core/internal/skills"
	"github.com/dunialabs/kimbap-core/internal/splash"
	"github.com/dunialabs/kimbap-core/internal/vault"
	"github.com/spf13/cobra"
)

type cliOptions struct {
	configPath string
	dataDir    string
	logLevel   string
	mode       string
	format     string
	jsonInput  string
	dryRun     bool
	trace      bool
	noSplash   bool
}

const defaultApprovalTTL = 30 * time.Minute

var opts = cliOptions{}

var rootCmd = &cobra.Command{
	Use:          "kimbap",
	Short:        "Secure action runtime for AI agents",
	Long:         "Kimbap lets AI agents use external services without handling raw credentials.",
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		showSplashOnce()
	},
}

var splashShown bool

func init() {
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&opts.configPath, "config", "", "config file path")
	flags.StringVar(&opts.dataDir, "data-dir", "", "data directory (default ~/.kimbap)")
	flags.StringVar(&opts.logLevel, "log-level", "", "log level (debug, info, warn, error)")
	flags.StringVar(&opts.mode, "mode", "", "execution mode (embedded, connected)")
	flags.StringVar(&opts.format, "format", "text", "output format (text, json)")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "validate and preview without executing (returns JSON)")
	flags.BoolVar(&opts.trace, "trace", false, "print execution pipeline trace to stderr")
	flags.BoolVar(&opts.noSplash, "no-splash", false, "disable startup splash output")
	rootCmd.Version = strings.TrimSpace(config.AppInfo.Version)
	rootCmd.SetVersionTemplate("kimbap {{.Version}}\n")
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		showSplashOnce()
		defaultHelp(cmd, args)
	})

	rootCmd.AddCommand(newCallCommand())
	rootCmd.AddCommand(newActionsCommand())
	rootCmd.AddCommand(newVaultCommand())
	rootCmd.AddCommand(newTokenCommand())
	rootCmd.AddCommand(newPolicyCommand())
	rootCmd.AddCommand(newDoctorCommand())
	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newServiceCommand())
	rootCmd.AddCommand(newConnectorCommand())
	rootCmd.AddCommand(newAuthCommand())
	rootCmd.AddCommand(newApproveCommand())
	rootCmd.AddCommand(newAuditCommand())
	rootCmd.AddCommand(newRunCommand())
	rootCmd.AddCommand(newProxyCommand())
	rootCmd.AddCommand(newServeCommand())
	rootCmd.AddCommand(newDaemonCommand())
	rootCmd.AddCommand(newAgentProfileCommand())
	rootCmd.AddCommand(newAgentsCommand())
}

func showSplashOnce() {
	if splashShown || opts.noSplash {
		return
	}
	if outputAsJSON() {
		return
	}
	if fi, err := os.Stdout.Stat(); err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return
	}
	splash.Print(splashOptions())
	splashShown = true
}

func splashOptions() splash.Options {
	cfg, err := loadSplashConfig()
	if err != nil {
		return splash.Options{
			Version:     strings.TrimSpace(config.AppInfo.Version),
			Mode:        modeFromRaw(opts.mode),
			VaultStatus: vaultStatusFromRaw(opts.mode),
			Server:      strings.TrimSpace(os.Getenv("KIMBAP_AUTH_SERVER_URL")),
		}
	}

	if strings.TrimSpace(opts.mode) != "" {
		cfg.Mode = strings.TrimSpace(opts.mode)
	}

	return splash.Options{
		Version:     strings.TrimSpace(config.AppInfo.Version),
		Mode:        modeFromRaw(cfg.Mode),
		VaultStatus: vaultStatusFromConfig(cfg),
		Server:      strings.TrimSpace(cfg.Auth.ServerURL),
	}
}

func loadSplashConfig() (*config.KimbapConfig, error) {
	if strings.TrimSpace(opts.configPath) == "" {
		return config.LoadKimbapConfig()
	}
	return config.LoadKimbapConfig(opts.configPath)
}

func modeFromRaw(raw string) splash.Mode {
	if strings.EqualFold(strings.TrimSpace(raw), string(splash.ModeConnected)) {
		return splash.ModeConnected
	}
	return splash.ModeEmbedded
}

func vaultStatusFromRaw(mode string) string {
	if strings.TrimSpace(os.Getenv("KIMBAP_MASTER_KEY_HEX")) != "" {
		return "ready"
	}
	devEnabled := strings.EqualFold(strings.TrimSpace(mode), "dev")
	if !devEnabled {
		if rawDev, ok := os.LookupEnv("KIMBAP_DEV"); ok {
			parsed, err := strconv.ParseBool(strings.TrimSpace(rawDev))
			if err == nil {
				devEnabled = parsed
			}
		}
	}
	if devEnabled {
		return "ready"
	}
	return "locked"
}

func vaultStatusFromConfig(cfg *config.KimbapConfig) string {
	if cfg == nil {
		return vaultStatusFromRaw("")
	}
	return vaultStatusFromRaw(cfg.Mode)
}

func loadAppConfig() (*config.KimbapConfig, error) {
	cfg, err := loadAppConfigReadOnly()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return cfg, nil
}

func loadAppConfigReadOnly() (*config.KimbapConfig, error) {
	var (
		cfg *config.KimbapConfig
		err error
	)
	if strings.TrimSpace(opts.configPath) == "" {
		cfg, err = config.LoadKimbapConfig()
	} else {
		cfg, err = config.LoadKimbapConfig(opts.configPath)
	}
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(opts.dataDir) != "" {
		cfg.DataDir = opts.dataDir
		cfg.Vault.Path = filepath.Join(cfg.DataDir, "vault.db")
		cfg.Skills.Dir = filepath.Join(cfg.DataDir, "skills")
		cfg.Audit.Path = filepath.Join(cfg.DataDir, "audit.jsonl")
		cfg.Database.DSN = filepath.Join(cfg.DataDir, "kimbap.db")
		cfg.Policy.Path = filepath.Join(cfg.DataDir, "policy.yaml")
	}
	if strings.TrimSpace(opts.logLevel) != "" {
		cfg.LogLevel = opts.logLevel
	}
	if strings.TrimSpace(opts.mode) != "" {
		cfg.Mode = opts.mode
	}

	return cfg, nil
}

func outputAsJSON() bool {
	return strings.EqualFold(strings.TrimSpace(opts.format), "json")
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

func installerFromConfig(cfg *config.KimbapConfig) *skills.LocalInstaller {
	return skills.NewLocalInstaller(cfg.Skills.Dir)
}

func loadInstalledActions(cfg *config.KimbapConfig) ([]actions.ActionDefinition, error) {
	installer := installerFromConfig(cfg)
	installed, err := installer.List()
	if err != nil {
		return nil, err
	}

	out := make([]actions.ActionDefinition, 0)
	for _, installedSkill := range installed {
		defs, convErr := skills.ToActionDefinitions(&installedSkill.Manifest)
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, defs...)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func resolveActionByName(cfg *config.KimbapConfig, name string) (*actions.ActionDefinition, error) {
	defs, err := loadInstalledActions(cfg)
	if err != nil {
		return nil, err
	}
	for i := range defs {
		if defs[i].Name == name {
			return &defs[i], nil
		}
	}
	return nil, fmt.Errorf("action %q not found in installed skills", name)
}

func splitActionName(actionName string) (service string, action string) {
	service, action, ok := strings.Cut(actionName, ".")
	if !ok {
		return "", actionName
	}
	return service, action
}

func initVaultStore(cfg *config.KimbapConfig) (vault.Store, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Vault.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create vault db dir: %w", err)
	}

	masterKey, err := resolveVaultMasterKey(cfg)
	if err != nil {
		return nil, err
	}
	envelope, err := corecrypto.NewEnvelopeService(masterKey)
	if err != nil {
		return nil, err
	}

	store, err := vault.OpenSQLiteStore(cfg.Vault.Path, envelope)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func resolveVaultMasterKey(cfg *config.KimbapConfig) ([]byte, error) {
	hexKey := strings.TrimSpace(os.Getenv("KIMBAP_MASTER_KEY_HEX"))
	if hexKey != "" {
		decoded, err := hex.DecodeString(hexKey)
		if err != nil {
			return nil, fmt.Errorf("decode KIMBAP_MASTER_KEY_HEX: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("KIMBAP_MASTER_KEY_HEX must decode to 32 bytes")
		}
		return decoded, nil
	}

	devEnabled := strings.EqualFold(strings.TrimSpace(cfg.Mode), "dev")
	if !devEnabled {
		if rawDev, ok := os.LookupEnv("KIMBAP_DEV"); ok {
			parsed, err := strconv.ParseBool(strings.TrimSpace(rawDev))
			if err != nil {
				return nil, fmt.Errorf("parse KIMBAP_DEV: %w", err)
			}
			devEnabled = parsed
		}
	}

	if !devEnabled {
		return nil, fmt.Errorf("vault master key is required: set KIMBAP_MASTER_KEY_HEX or enable dev mode (--mode dev or KIMBAP_DEV=true)")
	}
	devKeyPath := filepath.Join(cfg.DataDir, ".dev-master-key")
	if existing, err := os.ReadFile(devKeyPath); err == nil {
		if len(existing) != 32 {
			return nil, fmt.Errorf("invalid dev master key at %s: expected 32 bytes, got %d", devKeyPath, len(existing))
		}
		return existing, nil
	}
	key, err := corecrypto.GenerateRandomKey(32)
	if err != nil {
		return nil, fmt.Errorf("generate dev master key: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(devKeyPath, key, 0o600); err != nil {
		return nil, fmt.Errorf("persist dev master key: %w", err)
	}
	return key, nil
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

func contextBackground() context.Context {
	return context.Background()
}

func buildRuntimeFromConfig(cfg *config.KimbapConfig) (*runtime.Runtime, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required to build runtime")
	}
	vaultStore, err := initVaultStore(cfg)
	if err != nil {
		return nil, err
	}

	var auditWriter runtime.AuditWriter
	auditPath := strings.TrimSpace(cfg.Audit.Path)
	if auditPath != "" {
		jw, jwErr := audit.NewJSONLWriter(auditPath)
		if jwErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: audit writer init failed (%v), audit events will not be recorded\n", jwErr)
		} else {
			auditWriter = app.NewAuditWriterAdapter(jw)
		}
	}

	var approvalManager runtime.ApprovalManager
	if runtimeStore, rsErr := openRuntimeStore(cfg); rsErr == nil {
		approvalMgr := approvals.NewApprovalManager(
			&storeApprovalStoreAdapter{st: runtimeStore},
			buildNotifierFromConfig(cfg),
			defaultApprovalTTL,
		)
		approvalManager = app.NewApprovalManagerAdapter(approvalMgr)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "warning: approval store unavailable (%v); approval-requiring actions will fail\n", rsErr)
	}

	var connStore connectors.ConnectorStore
	var connConfigs []connectors.ConnectorConfig
	if cs, csErr := openConnectorStore(cfg); csErr == nil {
		connStore = cs
		for _, prov := range providers.ListProviders() {
			creds := resolveOAuthCreds(cfg, prov.ID)
			connConfigs = append(connConfigs, connectors.ConnectorConfig{
				Name:         prov.ID,
				Provider:     prov.ID,
				ClientID:     creds.ClientID,
				ClientSecret: creds.ClientSecret,
				AuthMethod:   creds.AuthMethod,
				TokenURL:     prov.TokenEndpoint,
				DeviceURL:    prov.DeviceEndpoint,
				Scopes:       prov.DefaultScopes,
			})
		}
	}

	return app.BuildRuntime(app.RuntimeDeps{
		Config:           cfg,
		VaultStore:       vaultStore,
		ConnectorStore:   connStore,
		ConnectorConfigs: connConfigs,
		PolicyPath:       cfg.Policy.Path,
		SkillsDir:        cfg.Skills.Dir,
		AuditWriter:      auditWriter,
		ApprovalManager:  approvalManager,
	})
}

// buildNotifierFromConfig constructs an approval notifier from the kimbap configuration.
// If no notification adapters are configured, it returns a LogNotifier as fallback.
// Uses best-effort delivery: individual adapter failures are logged but do not block approval creation.
func buildNotifierFromConfig(cfg *config.KimbapConfig) approvals.Notifier {
	if cfg == nil {
		return &approvals.LogNotifier{}
	}
	n := cfg.Notifications
	var notifiers []approvals.Notifier

	if strings.TrimSpace(n.Slack.WebhookURL) != "" {
		notifiers = append(notifiers, approvals.NewSlackNotifier(n.Slack.WebhookURL))
	}
	if strings.TrimSpace(n.Telegram.BotToken) != "" && strings.TrimSpace(n.Telegram.ChatID) != "" {
		notifiers = append(notifiers, approvals.NewTelegramNotifier(n.Telegram.BotToken, n.Telegram.ChatID))
	}
	if strings.TrimSpace(n.Email.SMTPHost) != "" && strings.TrimSpace(n.Email.From) != "" && len(n.Email.To) > 0 {
		notifiers = append(notifiers, approvals.NewEmailNotifier(
			n.Email.SMTPHost, n.Email.SMTPPort, n.Email.From, n.Email.To,
			n.Email.Username, n.Email.Password,
		))
	}
	if strings.TrimSpace(n.Webhook.URL) != "" {
		notifiers = append(notifiers, approvals.NewWebhookNotifier(n.Webhook.URL, []byte(n.Webhook.SignKey)))
	}

	if len(notifiers) == 0 {
		return &approvals.LogNotifier{}
	}
	return approvals.NewMultiNotifier(notifiers...)
}

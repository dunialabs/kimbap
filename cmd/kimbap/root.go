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
	"sync"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/app"
	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	corecrypto "github.com/dunialabs/kimbap/internal/crypto"
	"github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/internal/splash"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/spf13/cobra"
)

type cliOptions struct {
	configPath     string
	dataDir        string
	logLevel       string
	mode           string
	format         string
	splashColor    string
	jsonInput      string
	idempotencyKey string
	dryRun         bool
	trace          bool
	noSplash       bool
}

const defaultApprovalTTL = 30 * time.Minute

var opts = cliOptions{}

var (
	initVaultStoreForBuild     = initVaultStore
	openRuntimeStoreForBuild   = openRuntimeStore
	openConnectorStoreForBuild = openConnectorStore
	closeVaultStoreForBuild    = closeVaultStoreIfPossible
	closeRuntimeStoreForBuild  = func(st *store.SQLStore) {
		if st != nil {
			_ = st.Close()
		}
	}
	closeConnectorStoreForBuild = closeConnectorStoreIfPossible
	buildRuntimeForConfig       = app.BuildRuntime
)

func closeVaultStoreIfPossible(st vault.Store) {
	if closer, ok := st.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

func binaryName() string {
	if len(os.Args) > 0 {
		if name := filepath.Base(os.Args[0]); name != "" {
			return name
		}
	}
	return "kimbap"
}

var rootCmd = &cobra.Command{
	Use:          "kimbap",
	Short:        "Secure action runtime for AI agents",
	Long:         "Kimbap lets AI agents use external services without handling raw credentials.",
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		prescanRawSplashFlags()
		showSplashOnce()
	},
}

var splashShown bool

func init() {
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&opts.configPath, "config", "", "config file path")
	flags.StringVar(&opts.dataDir, "data-dir", "", "data directory (default ~/.kimbap)")
	flags.StringVar(&opts.logLevel, "log-level", "", "log level (debug, info, warn, error)")
	flags.StringVar(&opts.mode, "mode", "", "execution mode (dev, embedded, connected)")
	flags.StringVar(&opts.format, "format", "text", "output format (text, json)")
	flags.StringVar(&opts.splashColor, "splash-color", "", "splash color mode override (auto, truecolor, ansi256, none)")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "validate and preview without executing (returns JSON)")
	flags.BoolVar(&opts.trace, "trace", false, "print execution pipeline trace to stderr")
	flags.BoolVar(&opts.noSplash, "no-splash", false, "disable startup splash output")
	name := binaryName()
	rootCmd.Use = name
	rootCmd.Version = config.CLIVersionDisplay()
	rootCmd.SetVersionTemplate(name + " {{.Version}}\n")
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		showSplashOnce()
		defaultHelp(cmd, args)
	})

	rootCmd.AddGroup(
		&cobra.Group{ID: "core", Title: "Core Commands:"},
		&cobra.Group{ID: "setup", Title: "Setup:"},
		&cobra.Group{ID: "management", Title: "Management:"},
		&cobra.Group{ID: "advanced", Title: "Advanced:"},
	)

	addGrouped := func(cmd *cobra.Command, groupID string) {
		cmd.GroupID = groupID
		rootCmd.AddCommand(cmd)
	}
	addGroupedHidden := func(cmd *cobra.Command, groupID string) {
		cmd.GroupID = groupID
		cmd.Hidden = true
		rootCmd.AddCommand(cmd)
	}

	// Core — day-to-day workflow
	addGrouped(newCallCommand(), "core")
	addGrouped(newLinkCommand(), "core")
	addGrouped(newSearchCommand(), "core")
	addGrouped(newActionsCommand(), "core")
	addGrouped(newAliasCommand(), "core")

	// Setup — install and configure
	addGrouped(newInitCommand(), "setup")
	addGrouped(newServiceCommand(), "setup")
	addGrouped(newAgentsCommand(), "setup")

	// Management — credentials, policy, diagnostics
	addGrouped(newVaultCommand(), "management")
	addGrouped(newPolicyCommand(), "management")
	addGroupedHidden(newCheckCommand(), "management")
	addGrouped(newDoctorCommand(), "management")
	addGrouped(newStatusCommand(), "management")
	addGrouped(newAuthCommand(), "management")

	// Advanced — specialized integration modes
	addGrouped(newApproveCommand(), "advanced")
	addGrouped(newRunCommand(), "advanced")
	addGrouped(newProxyCommand(), "advanced")
	addGrouped(newServeCommand(), "advanced")

	for _, c := range []*cobra.Command{
		newConnectorCommand(),
		newAgentProfileCommand(),
		newGenerateCommand(),
		newTokenCommand(),
		newAuditCommand(),
		newDaemonCommand(),
		newCompletionCommand(),
	} {
		c.Hidden = true
		rootCmd.AddCommand(c)
	}
}

func prescanRawSplashFlags() {
	withinCall := false
	callActionSeen := false
	for i := 1; i < len(os.Args); i++ {
		tok := strings.TrimSpace(os.Args[i])
		if tok == "--" {
			break
		}
		if tok == "call" {
			withinCall = true
			callActionSeen = false
			continue
		}
		if withinCall && !callActionSeen && tok != "" && !strings.HasPrefix(tok, "-") {
			callActionSeen = true
			continue
		}
		switch {
		case tok == "--no-splash":
			value, consumed := parseOptionalBoolFlagValue(os.Args, i)
			opts.noSplash = value
			i += consumed
		case strings.HasPrefix(tok, "--no-splash="):
			if v, err := strconv.ParseBool(strings.TrimSpace(strings.TrimPrefix(tok, "--no-splash="))); err == nil {
				opts.noSplash = v
			}
		case !(withinCall && callActionSeen) && tok == "--format" && i+1 < len(os.Args):
			next := strings.TrimSpace(os.Args[i+1])
			if !strings.HasPrefix(next, "-") {
				opts.format = next
				i++
			}
		case tok == "--splash-color" && i+1 < len(os.Args):
			next := strings.TrimSpace(os.Args[i+1])
			if !strings.HasPrefix(next, "-") {
				opts.splashColor = next
				i++
			}
		case !(withinCall && callActionSeen) && strings.HasPrefix(tok, "--format="):
			opts.format = strings.TrimSpace(strings.TrimPrefix(tok, "--format="))
		case strings.HasPrefix(tok, "--splash-color="):
			opts.splashColor = strings.TrimSpace(strings.TrimPrefix(tok, "--splash-color="))
		}
	}
}

func showSplashOnce() {
	if splashShown || opts.noSplash {
		return
	}
	if shouldSuppressSplashForInvocation(os.Args[1:]) {
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

func shouldSuppressSplashForInvocation(args []string) bool {
	commandPath := primaryCommandPath(args)
	return len(commandPath) >= 2 && commandPath[0] == "audit" && commandPath[1] == "export"
}

func primaryCommandPath(args []string) []string {
	out := make([]string, 0, 2)
	stringFlags := map[string]bool{
		"--config":       true,
		"--data-dir":     true,
		"--log-level":    true,
		"--mode":         true,
		"--format":       true,
		"--splash-color": true,
	}

	for i := 0; i < len(args); i++ {
		tok := strings.TrimSpace(args[i])
		if tok == "" {
			continue
		}
		if tok == "--" {
			break
		}
		if strings.HasPrefix(tok, "-") {
			if tok == "--no-splash" || tok == "--dry-run" || tok == "--trace" {
				_, consumed := parseOptionalBoolFlagValue(args, i)
				i += consumed
				continue
			}
			if strings.Contains(tok, "=") {
				continue
			}
			if stringFlags[tok] && i+1 < len(args) {
				next := strings.TrimSpace(args[i+1])
				if next != "" && !strings.HasPrefix(next, "-") {
					i++
				}
			}
			continue
		}
		out = append(out, strings.ToLower(tok))
		if len(out) == 2 {
			break
		}
	}

	return out
}

func splashOptions() splash.Options {
	cfg, err := loadBaseConfigForCLI()
	if err != nil {
		return splash.Options{
			Version:      config.CLIVersion(),
			Mode:         modeFromRaw(opts.mode),
			VaultStatus:  vaultStatusFromRaw(opts.mode),
			Server:       strings.TrimSpace(os.Getenv("KIMBAP_AUTH_SERVER_URL")),
			ColorProfile: detectSplashColorProfile(),
			Background:   detectSplashBackgroundTone(),
		}
	}

	if strings.TrimSpace(opts.mode) != "" {
		cfg.Mode = strings.TrimSpace(opts.mode)
	}

	return splash.Options{
		Version:      config.CLIVersion(),
		Mode:         modeFromRaw(cfg.Mode),
		VaultStatus:  vaultStatusFromConfig(cfg),
		Server:       strings.TrimSpace(cfg.Auth.ServerURL),
		ColorProfile: detectSplashColorProfile(),
		Background:   detectSplashBackgroundTone(),
	}
}

func modeFromRaw(raw string) splash.Mode {
	if strings.EqualFold(strings.TrimSpace(raw), string(splash.ModeConnected)) {
		return splash.ModeConnected
	}
	return splash.ModeEmbedded
}

func vaultStatusFromRaw(mode string) string {
	if _, err, present := decodeMasterKeyHexEnv(); present {
		if err != nil {
			return "error"
		}
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

func detectSplashColorProfile() splash.ColorProfile {
	flagRaw := strings.TrimSpace(opts.splashColor)
	if strings.EqualFold(flagRaw, string(splash.ColorProfileAuto)) {
		return detectSplashColorProfileAuto()
	}
	if override, ok := parseSplashColorProfile(flagRaw); ok {
		return override
	}
	envRaw := strings.TrimSpace(os.Getenv("KIMBAP_SPLASH_COLOR"))
	if strings.EqualFold(envRaw, string(splash.ColorProfileAuto)) {
		return detectSplashColorProfileAuto()
	}
	if override, ok := parseSplashColorProfile(envRaw); ok {
		return override
	}

	return detectSplashColorProfileAuto()
}

func detectSplashColorProfileAuto() splash.ColorProfile {
	if !isColorStdout() {
		return splash.ColorProfileNone
	}
	if supportsTrueColorEnv() {
		return splash.ColorProfileTrueColor
	}
	if supportsANSI256Env() {
		return splash.ColorProfileANSI256
	}
	return splash.ColorProfileNone
}

func parseSplashColorProfile(raw string) (splash.ColorProfile, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return splash.ColorProfileAuto, false
	}
	switch v {
	case "auto":
		return splash.ColorProfileAuto, false
	case "truecolor", "24bit", "24-bit":
		return splash.ColorProfileTrueColor, true
	case "ansi256", "256", "256color", "256-color":
		return splash.ColorProfileANSI256, true
	case "none", "off", "false", "no":
		return splash.ColorProfileNone, true
	default:
		return splash.ColorProfileAuto, false
	}
}

func detectSplashBackgroundTone() splash.BackgroundTone {
	raw := strings.TrimSpace(os.Getenv("COLORFGBG"))
	if raw == "" {
		return splash.BackgroundToneDark
	}
	parts := strings.Split(raw, ";")
	last := strings.TrimSpace(parts[len(parts)-1])
	bg, err := strconv.Atoi(last)
	if err != nil {
		return splash.BackgroundToneDark
	}
	if bg == 7 || bg == 15 {
		return splash.BackgroundToneLight
	}
	if bg >= 0 && bg <= 6 {
		return splash.BackgroundToneDark
	}
	return splash.BackgroundToneDark
}

func supportsTrueColorEnv() bool {
	colorterm := strings.ToLower(strings.TrimSpace(os.Getenv("COLORTERM")))
	if strings.Contains(colorterm, "truecolor") || strings.Contains(colorterm, "24bit") {
		return true
	}

	term := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	if strings.Contains(term, "direct") {
		return true
	}

	termProgram := strings.ToLower(strings.TrimSpace(os.Getenv("TERM_PROGRAM")))
	if termProgram == "iterm.app" || termProgram == "apple_terminal" || termProgram == "wezterm" || termProgram == "vscode" {
		return true
	}

	if strings.TrimSpace(os.Getenv("WT_SESSION")) != "" || strings.TrimSpace(os.Getenv("KITTY_WINDOW_ID")) != "" {
		return true
	}

	return false
}

func supportsANSI256Env() bool {
	term := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	return strings.Contains(term, "256color")
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

func resolveActionByName(cfg *config.KimbapConfig, name string) (*actions.ActionDefinition, error) {
	resolved := resolveAliasedActionName(cfg, name)
	defs, err := loadInstalledActions(cfg)
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		allInstalled, listErr := installerFromConfig(cfg).List()
		if listErr != nil {
			return nil, listErr
		}
		if len(allInstalled) > 0 {
			return nil, fmt.Errorf("no enabled services found — run 'kimbap service list' to see installed services")
		}
		return nil, fmt.Errorf("no services installed — run 'kimbap init --services select' to choose what to install")
	}
	for i := range defs {
		if defs[i].Name == resolved {
			return &defs[i], nil
		}
	}

	if !strings.Contains(resolved, ".") {
		serviceName := strings.ToLower(resolved)
		var serviceActions []actions.ActionDefinition
		var canonicalNamespace string
		for _, d := range defs {
			if strings.EqualFold(d.Namespace, serviceName) {
				serviceActions = append(serviceActions, d)
				if canonicalNamespace == "" {
					canonicalNamespace = d.Namespace
				}
			}
		}
		if len(serviceActions) > 0 {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%q is a service name. Specify an action:\n", resolved))
			limit := 5
			if len(serviceActions) < limit {
				limit = len(serviceActions)
			}
			for i := 0; i < limit; i++ {
				desc := serviceActions[i].Description
				if desc == "" {
					desc = "-"
				}
				sb.WriteString(fmt.Sprintf("  kimbap call %-40s %s\n", serviceActions[i].Name, desc))
			}
			if len(serviceActions) > 5 {
				sb.WriteString(fmt.Sprintf("\nRun 'kimbap actions list --service %s' to see all %d actions.", canonicalNamespace, len(serviceActions)))
			}
			return nil, fmt.Errorf("%s", sb.String())
		}
	}

	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	hint := "Run 'kimbap actions list' to see available actions, or 'kimbap search <query>' to find by keyword."
	if suggestion := didYouMean(resolved, names); suggestion != "" {
		hint = fmt.Sprintf("Did you mean %q? Run 'kimbap actions list' to see all available actions.", suggestion)
	}
	return nil, fmt.Errorf("action %q not found in installed services. %s", resolved, hint)
}

func resolveAliasedActionName(cfg *config.KimbapConfig, actionName string) string {
	if cfg == nil || len(cfg.Aliases) == 0 {
		return actionName
	}
	service, action := splitActionName(actionName)
	if service == "" || action == "" {
		return actionName
	}
	if !services.IsValidServiceAlias(service) {
		return actionName
	}
	if target, ok := cfg.Aliases[service]; ok {
		target = strings.ToLower(strings.TrimSpace(target))
		if target != "" && services.ValidateServiceName(target) == nil {
			return target + "." + action
		}
	}
	return actionName
}

func splitActionName(actionName string) (service string, action string) {
	service, action, ok := strings.Cut(actionName, ".")
	if !ok {
		return "", actionName
	}
	return service, action
}

func rewriteArgsForExecutableAlias(argv []string) []string {
	if len(argv) == 0 {
		return argv
	}

	binary := strings.TrimSpace(filepath.Base(argv[0]))
	if binary == "" || strings.EqualFold(binary, "kimbap") {
		return argv
	}

	configPath, dataDir := parseConfigPathAndDataDirArgs(argv)

	var (
		cfg *config.KimbapConfig
		err error
	)
	if strings.TrimSpace(configPath) == "" && strings.TrimSpace(dataDir) == "" {
		cfg, err = config.LoadKimbapConfig()
	} else {
		resolvedPath, resolveErr := config.ResolveConfigPathWithDataDir(configPath, dataDir)
		if resolveErr != nil {
			return argv
		}
		if st, statErr := os.Stat(resolvedPath); statErr == nil {
			if st.IsDir() {
				return argv
			}
			cfg, err = config.LoadKimbapConfigWithoutDefault(resolvedPath)
		} else if os.IsNotExist(statErr) {
			cfg, err = config.LoadKimbapConfigWithoutDefault()
		} else {
			return argv
		}
	}
	if err != nil {
		return argv
	}

	return rewriteArgsForConfiguredExecutableAlias(argv, cfg.CommandAliases)
}

func parseConfigPathAndDataDirArgs(argv []string) (configPath string, dataDir string) {
	for i := 1; i < len(argv); i++ {
		tok := strings.TrimSpace(argv[i])
		if tok == "--" {
			break
		}
		if tok == "--config" && i+1 < len(argv) {
			next := strings.TrimSpace(argv[i+1])
			if next != "" && !strings.HasPrefix(next, "-") {
				configPath = next
				i++
				continue
			}
		}
		if tok == "--data-dir" && i+1 < len(argv) {
			next := strings.TrimSpace(argv[i+1])
			if next != "" && !strings.HasPrefix(next, "-") {
				dataDir = next
				i++
				continue
			}
		}
		if strings.HasPrefix(tok, "--config=") {
			configPath = strings.TrimSpace(strings.TrimPrefix(tok, "--config="))
			continue
		}
		if strings.HasPrefix(tok, "--data-dir=") {
			dataDir = strings.TrimSpace(strings.TrimPrefix(tok, "--data-dir="))
			continue
		}
	}
	return configPath, dataDir
}

func rewriteArgsForConfiguredExecutableAlias(argv []string, commandAliases map[string]string) []string {
	if len(argv) == 0 || len(commandAliases) == 0 {
		return argv
	}

	binary := strings.TrimSpace(filepath.Base(argv[0]))
	if binary == "" || strings.EqualFold(binary, "kimbap") {
		return argv
	}

	target := strings.TrimSpace(commandAliases[binary])
	if target == "" {
		target = strings.TrimSpace(commandAliases[strings.ToLower(binary)])
	}
	if target == "" || !strings.Contains(target, ".") {
		return argv
	}

	out := make([]string, 0, len(argv)+2)
	out = append(out, argv[0], "call", target)
	out = append(out, argv[1:]...)
	return out
}

func initVaultStore(cfg *config.KimbapConfig) (vault.Store, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Vault.Path), 0o700); err != nil {
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
	if decoded, err, present := decodeMasterKeyHexEnv(); present {
		if err != nil {
			return nil, err
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
	existing, readErr := readPersistedDevMasterKey(devKeyPath)
	if readErr == nil {
		return existing, nil
	}
	if !os.IsNotExist(readErr) {
		return nil, fmt.Errorf("read dev master key %s: %w", devKeyPath, readErr)
	}
	key, err := corecrypto.GenerateRandomKey(32)
	if err != nil {
		return nil, fmt.Errorf("generate dev master key: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	f, err := os.OpenFile(devKeyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			existing, readErr := readPersistedDevMasterKey(devKeyPath)
			if readErr == nil {
				return existing, nil
			}
			return nil, fmt.Errorf("read dev master key %s after concurrent create: %w", devKeyPath, readErr)
		}
		return nil, fmt.Errorf("persist dev master key: %w", err)
	}
	_, writeErr := f.Write(key)
	_ = f.Close()
	if writeErr != nil {
		_ = os.Remove(devKeyPath)
		return nil, fmt.Errorf("write dev master key: %w", writeErr)
	}
	return key, nil
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

func buildRuntimeFromConfig(cfg *config.KimbapConfig) (*runtime.Runtime, error) {
	rt, _, err := buildRuntimeFromConfigWithCleanup(cfg)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func buildRuntimeFromConfigWithCleanup(cfg *config.KimbapConfig) (*runtime.Runtime, func(), error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("config is required to build runtime")
	}
	vaultStore, err := initVaultStoreForBuild(cfg)
	if err != nil {
		return nil, nil, err
	}

	var auditWriter runtime.AuditWriter
	var auditCloser interface{ Close() error }
	auditPath := strings.TrimSpace(cfg.Audit.Path)
	if auditPath != "" {
		jw, jwErr := audit.NewJSONLWriter(auditPath)
		if jwErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: audit writer init failed (%v), audit events will not be recorded\n", jwErr)
		} else {
			auditWriter = app.NewAuditWriterAdapter(audit.NewRedactingWriter(jw))
			auditCloser = jw
		}
	}

	var approvalManager runtime.ApprovalManager
	var runtimeStoreForCleanup *store.SQLStore
	if runtimeStore, rsErr := openRuntimeStoreForBuild(cfg); rsErr == nil {
		runtimeStoreForCleanup = runtimeStore
		approvalMgr := approvals.NewApprovalManager(
			&storeApprovalStoreAdapter{st: runtimeStore},
			buildNotifierFromConfig(cfg),
			defaultApprovalTTL,
		)
		approvalManager = app.NewApprovalManagerAdapter(approvalMgr)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %s; approval-requiring actions will fail\n", unavailableMessage(componentApprovalStore, rsErr))
	}

	var connStore connectors.ConnectorStore
	var connConfigs []connectors.ConnectorConfig
	if cs, csErr := openConnectorStoreForBuild(cfg); csErr == nil {
		connStore = cs
		connConfigs = buildConnectorConfigs(cfg)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %s; oauth-backed credential resolution may fail\n", unavailableMessage(componentConnectorStore, csErr))
	}

	rt, buildErr := buildRuntimeForConfig(app.RuntimeDeps{
		Config:           cfg,
		VaultStore:       vaultStore,
		ConnectorStore:   connStore,
		ConnectorConfigs: connConfigs,
		PolicyPath:       cfg.Policy.Path,
		ServicesDir:      cfg.Services.Dir,
		AuditWriter:      auditWriter,
		ApprovalManager:  approvalManager,
		HeldStore:        runtimeStoreForCleanup,
	})
	if buildErr != nil {
		if auditCloser != nil {
			_ = auditCloser.Close()
		}
		closeVaultStoreForBuild(vaultStore)
		if runtimeStoreForCleanup != nil {
			closeRuntimeStoreForBuild(runtimeStoreForCleanup)
		}
		closeConnectorStoreForBuild(connStore)
		return nil, nil, buildErr
	}
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if auditCloser != nil {
				_ = auditCloser.Close()
			}
			closeVaultStoreForBuild(vaultStore)
			if runtimeStoreForCleanup != nil {
				closeRuntimeStoreForBuild(runtimeStoreForCleanup)
			}
			closeConnectorStoreForBuild(connStore)
		})
	}
	return rt, cleanup, nil
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

func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for kimbap (also available as kb).

Add to your shell profile:
  bash:       source <(kimbap completion bash)
  zsh:        source <(kimbap completion zsh)
  fish:       kimbap completion fish | source
  powershell: kimbap completion powershell | Out-String | Invoke-Expression`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(_ *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
}

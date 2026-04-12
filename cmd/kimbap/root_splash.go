package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/splash"
)

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
	if splashShownRecently() {
		splashShown = true
		return
	}
	splash.Print(splashOptions())
	markSplashShown()
	splashShown = true
}

func splashShownRecently() bool {
	stampPath, ok := resolveSplashStampPath()
	if !ok {
		return false
	}
	raw, err := os.ReadFile(stampPath)
	if err != nil {
		return false
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return false
	}
	shownAt := time.Unix(ts, 0)
	if shownAt.After(time.Now()) {
		return false
	}
	return time.Since(shownAt) < 2*time.Hour
}

func markSplashShown() {
	stampPath, ok := resolveSplashStampPath()
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(stampPath), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(stampPath, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0o600)
}

func resolveSplashStampPath() (string, bool) {
	cfg, err := loadBaseConfigForCLI()
	if err != nil {
		return "", false
	}
	config.ApplyDataDirOverride(cfg, opts.dataDir)
	dataDir := strings.TrimSpace(cfg.DataDir)
	if dataDir == "" {
		return "", false
	}
	return filepath.Join(dataDir, ".splash-stamp"), true
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
			Server:       sanitizeDisplayURL(strings.TrimSpace(os.Getenv("KIMBAP_AUTH_SERVER_URL"))),
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
		Server:       sanitizeDisplayURL(strings.TrimSpace(cfg.Auth.ServerURL)),
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

var _ = fmt.Sprintf

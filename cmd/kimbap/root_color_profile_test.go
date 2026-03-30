package main

import (
	"testing"

	"github.com/dunialabs/kimbap/internal/splash"
)

func TestSupportsTrueColorEnv_Colorterm(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("WT_SESSION", "")
	t.Setenv("KITTY_WINDOW_ID", "")

	if !supportsTrueColorEnv() {
		t.Fatal("expected truecolor support when COLORTERM=truecolor")
	}
}

func TestSupportsTrueColorEnv_TermProgram(t *testing.T) {
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	t.Setenv("WT_SESSION", "")
	t.Setenv("KITTY_WINDOW_ID", "")

	if !supportsTrueColorEnv() {
		t.Fatal("expected truecolor support for iTerm")
	}
}

func TestSupportsANSI256Env(t *testing.T) {
	t.Setenv("TERM", "screen-256color")
	if !supportsANSI256Env() {
		t.Fatal("expected ANSI 256 support when TERM contains 256color")
	}

	t.Setenv("TERM", "dumb")
	if supportsANSI256Env() {
		t.Fatal("did not expect ANSI 256 support for TERM=dumb")
	}
}

func TestParseSplashColorProfile(t *testing.T) {
	profile, ok := parseSplashColorProfile("ansi256")
	if !ok || profile != splash.ColorProfileANSI256 {
		t.Fatalf("expected ansi256 override, got profile=%q ok=%v", profile, ok)
	}

	profile, ok = parseSplashColorProfile("none")
	if !ok || profile != splash.ColorProfileNone {
		t.Fatalf("expected none override, got profile=%q ok=%v", profile, ok)
	}

	profile, ok = parseSplashColorProfile("auto")
	if ok || profile != splash.ColorProfileAuto {
		t.Fatalf("expected auto to disable override, got profile=%q ok=%v", profile, ok)
	}
}

func TestDetectSplashColorProfile_OptionOverride(t *testing.T) {
	resetOptsForTest(t)
	t.Setenv("NO_COLOR", "1")
	opts.splashColor = "ansi256"

	profile := detectSplashColorProfile()
	if profile != splash.ColorProfileANSI256 {
		t.Fatalf("expected ansi256 override even with NO_COLOR, got %q", profile)
	}
}

func TestDetectSplashColorProfile_EnvOverride(t *testing.T) {
	resetOptsForTest(t)
	t.Setenv("KIMBAP_SPLASH_COLOR", "none")
	t.Setenv("NO_COLOR", "")

	profile := detectSplashColorProfile()
	if profile != splash.ColorProfileNone {
		t.Fatalf("expected none from env override, got %q", profile)
	}
}

func TestDetectSplashColorProfile_InvalidFlagFallsBackToEnv(t *testing.T) {
	resetOptsForTest(t)
	opts.splashColor = "invalid-value"
	t.Setenv("KIMBAP_SPLASH_COLOR", "ansi256")

	profile := detectSplashColorProfile()
	if profile != splash.ColorProfileANSI256 {
		t.Fatalf("expected ansi256 from env when flag is invalid, got %q", profile)
	}
}

func TestDetectSplashColorProfile_ExplicitAutoIgnoresEnvOverride(t *testing.T) {
	resetOptsForTest(t)
	t.Setenv("KIMBAP_SPLASH_COLOR", "none")
	opts.splashColor = "auto"

	profile := detectSplashColorProfile()
	if profile != splash.ColorProfileNone {
		t.Fatalf("expected explicit auto to bypass env override and use auto detection, got %q", profile)
	}
}

func TestShouldSuppressSplashForInvocation_AuditExport(t *testing.T) {
	if !shouldSuppressSplashForInvocation([]string{"audit", "export", "--from", "2026-01-01", "--to", "2026-01-02"}) {
		t.Fatal("expected audit export invocation to suppress splash")
	}
}

func TestShouldSuppressSplashForInvocation_ResolvesAfterGlobalFlags(t *testing.T) {
	args := []string{"--config", "/tmp/cfg.yml", "audit", "export", "--from", "2026-01-01", "--to", "2026-01-02"}
	if !shouldSuppressSplashForInvocation(args) {
		t.Fatal("expected audit export invocation with global flags to suppress splash")
	}
}

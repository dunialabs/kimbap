package config

import "testing"

func TestCLIVersionFallback(t *testing.T) {
	prev := AppInfo
	AppInfo.Version = ""
	AppInfo.Commit = ""
	AppInfo.Date = ""
	t.Cleanup(func() { AppInfo = prev })

	if got := CLIVersion(); got != "0.0.1-dev" {
		t.Fatalf("CLIVersion() = %q, want %q", got, "0.0.1-dev")
	}
}

func TestCLIVersionDisplayMetadata(t *testing.T) {
	prev := AppInfo
	AppInfo.Version = "0.1.0"
	AppInfo.Commit = "1234567890abcdef"
	AppInfo.Date = "2026-03-30T01:02:03Z"
	t.Cleanup(func() { AppInfo = prev })

	got := CLIVersionDisplay()
	want := "0.1.0 (commit 1234567890ab, 2026-03-30T01:02:03Z)"
	if got != want {
		t.Fatalf("CLIVersionDisplay() = %q, want %q", got, want)
	}
}

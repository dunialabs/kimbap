package services

import "testing"

func TestNormalizeAppleScriptRegistryMode(t *testing.T) {
	tests := []struct {
		in      string
		want    AppleScriptRegistryMode
		wantErr bool
	}{
		{in: "", want: AppleScriptRegistryModeDual},
		{in: "dual", want: AppleScriptRegistryModeDual},
		{in: "legacy", want: AppleScriptRegistryModeLegacy},
		{in: "manifest", want: AppleScriptRegistryModeManifest},
		{in: "weird", wantErr: true},
	}

	for _, tc := range tests {
		got, err := NormalizeAppleScriptRegistryMode(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("NormalizeAppleScriptRegistryMode(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeAppleScriptRegistryMode(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeAppleScriptRegistryMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSetAppleScriptRegistryMode(t *testing.T) {
	prev := CurrentAppleScriptRegistryMode()
	t.Cleanup(func() {
		_ = SetAppleScriptRegistryMode(string(prev))
	})

	if err := SetAppleScriptRegistryMode("manifest"); err != nil {
		t.Fatalf("SetAppleScriptRegistryMode(manifest): %v", err)
	}
	if got := CurrentAppleScriptRegistryMode(); got != AppleScriptRegistryModeManifest {
		t.Fatalf("CurrentAppleScriptRegistryMode() = %q, want manifest", got)
	}
}

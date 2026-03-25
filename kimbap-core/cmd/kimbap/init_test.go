package main

import (
	"path/filepath"
	"testing"
)

func TestBuildInitConfigRebasesPolicyPathWithDataDir(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })

	opts = cliOptions{}
	opts.dataDir = t.TempDir()

	cfg := buildInitConfig()
	want := filepath.Join(opts.dataDir, "policy.yaml")
	if cfg.Policy.Path != want {
		t.Fatalf("expected policy path %q, got %q", want, cfg.Policy.Path)
	}
}

func TestBuildInitConfigKeepsDefaultModeEmbedded(t *testing.T) {
	original := opts
	t.Cleanup(func() { opts = original })

	opts = cliOptions{}
	cfg := buildInitConfig()
	if cfg.Mode != "embedded" {
		t.Fatalf("expected default init mode embedded, got %q", cfg.Mode)
	}
}

func TestValidateInitMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "embedded", mode: "embedded"},
		{name: "dev", mode: "dev"},
		{name: "connected", mode: "connected"},
		{name: "invalid", mode: "unknown", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInitMode(tc.mode)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for mode %q", tc.mode)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for mode %q, got %v", tc.mode, err)
			}
		})
	}
}

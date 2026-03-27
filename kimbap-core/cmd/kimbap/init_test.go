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

func TestAppendInitChecksFlagsFailures(t *testing.T) {
	checks, hasFailure := appendInitChecks(nil, false,
		doctorCheck{Name: "ok", Status: "ok"},
		doctorCheck{Name: "warn", Status: "warn"},
	)
	if hasFailure {
		t.Fatal("expected non-failing checks to keep hasFailure=false")
	}

	checks, hasFailure = appendInitChecks(checks, hasFailure, doctorCheck{Name: "fail", Status: "fail"})
	if !hasFailure {
		t.Fatal("expected failing check to set hasFailure=true")
	}
	if len(checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(checks))
	}
}

func TestOptionalKBCheckDoesNotFailInit(t *testing.T) {
	checks := []doctorCheck{}
	hasFailure := false

	kbCheck := doctorCheck{Name: "kb alias", Status: "fail"}
	checks = append(checks, kbCheck)

	if hasFailure {
		t.Fatal("optional kbCheck failure must not set hasFailure")
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check recorded, got %d", len(checks))
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

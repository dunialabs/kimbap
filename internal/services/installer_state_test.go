package services

import (
	"strings"
	"testing"
)

func TestLocalInstallerEnableDisableAndListEnabled(t *testing.T) {
	installer := NewLocalInstaller(t.TempDir())
	manifest := validHTTPManifest()
	manifest.Name = "toggle-service"

	if _, err := installer.Install(manifest, "test"); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	enabled, err := installer.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}
	if len(enabled) != 1 || enabled[0].Manifest.Name != "toggle-service" {
		t.Fatalf("ListEnabled() = %+v, want toggle-service", enabled)
	}

	if err := installer.Disable("toggle-service"); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	got, err := installer.Get("toggle-service")
	if err != nil {
		t.Fatalf("Get() after disable error = %v", err)
	}
	if got.Enabled {
		t.Fatal("expected disabled service state after Disable()")
	}

	enabled, err = installer.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled() after disable error = %v", err)
	}
	if len(enabled) != 0 {
		t.Fatalf("expected 0 enabled services after disable, got %d", len(enabled))
	}

	if err := installer.Enable("toggle-service"); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	got, err = installer.Get("toggle-service")
	if err != nil {
		t.Fatalf("Get() after enable error = %v", err)
	}
	if !got.Enabled {
		t.Fatal("expected enabled service state after Enable()")
	}
}

func TestInstallWithForceAndActivationCanStartDisabled(t *testing.T) {
	installer := NewLocalInstaller(t.TempDir())
	manifest := validHTTPManifest()
	manifest.Name = "starts-disabled"

	if _, err := installer.InstallWithForceAndActivation(manifest, "test", false, false); err != nil {
		t.Fatalf("InstallWithForceAndActivation() error = %v", err)
	}

	installed, err := installer.Get("starts-disabled")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if installed.Enabled {
		t.Fatal("expected installed service to start disabled")
	}

	enabledOnly, err := installer.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}
	if len(enabledOnly) != 0 {
		t.Fatalf("expected no enabled services, got %d", len(enabledOnly))
	}
}

func TestDisableReturnsHelpfulErrorWhenServiceMissing(t *testing.T) {
	installer := NewLocalInstaller(t.TempDir())
	err := installer.Disable("missing-service")
	if err == nil {
		t.Fatal("expected disable on missing service to fail")
	}
	if !strings.Contains(err.Error(), "is not installed") {
		t.Fatalf("unexpected error = %v", err)
	}
}

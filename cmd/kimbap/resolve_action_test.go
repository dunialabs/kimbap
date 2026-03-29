package main

import (
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/services"
)

func TestResolveActionByName_NoServicesInstalled_SuggestsInit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Services.Dir = t.TempDir()

	_, err := resolveActionByName(cfg, "github.list-repos")
	if err == nil {
		t.Fatal("expected error when no services are installed")
	}

	if !strings.Contains(err.Error(), "No services installed") {
		t.Fatalf("expected no-services message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "kimbap init --mode dev --services all") {
		t.Fatalf("expected init guidance in error, got %q", err.Error())
	}
}

func TestResolveActionByName_ServicesInstalled_ActionNotFound(t *testing.T) {
	servicesDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Services.Dir = servicesDir

	manifest := &services.ServiceManifest{
		Name:    "demo",
		Version: "1.0.0",
		Adapter: "http",
		BaseURL: "https://example.com",
		Auth: services.ServiceAuth{
			Type: string(actions.AuthTypeNone),
		},
		Actions: map[string]services.ServiceAction{
			"noop": {
				Method:      "GET",
				Path:        "/noop",
				Description: "noop",
				Risk:        services.RiskSpec{Level: "low"},
				Response:    services.ResponseSpec{Type: "object"},
			},
		},
	}

	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install service: %v", err)
	}

	_, err := resolveActionByName(cfg, "demo.nop")
	if err == nil {
		t.Fatal("expected action-not-found error")
	}

	errText := err.Error()
	if !strings.Contains(errText, "action \"demo.nop\" not found in installed services") {
		t.Fatalf("expected action-not-found message, got %q", errText)
	}
	if !strings.Contains(errText, "Did you mean \"demo.noop\"") {
		t.Fatalf("expected did-you-mean suggestion, got %q", errText)
	}
}

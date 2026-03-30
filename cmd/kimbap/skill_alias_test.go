package main

import (
	"testing"

	"github.com/dunialabs/kimbap/internal/config"
)

func TestConfiguredServiceCallAliasReturnsConfiguredValueOnly(t *testing.T) {
	cfg := &config.KimbapConfig{
		Aliases: map[string]string{
			"geo": "open-meteo-geocoding",
			"":    "open-meteo-geocoding",
		},
	}

	alias := configuredServiceCallAlias(cfg, "open-meteo-geocoding")
	if alias != "geo" {
		t.Fatalf("expected configured alias 'geo', got %q", alias)
	}
}

func TestConfiguredServiceCallAliasReturnsEmptyWhenNoConfiguredAlias(t *testing.T) {
	cfg := &config.KimbapConfig{
		Aliases: map[string]string{
			"github": "github",
		},
	}

	alias := configuredServiceCallAlias(cfg, "open-meteo-geocoding")
	if alias != "" {
		t.Fatalf("expected empty alias when none configured, got %q", alias)
	}
}

func TestConfiguredServiceCallAliasDeterministicSelection(t *testing.T) {
	cfg := &config.KimbapConfig{
		Aliases: map[string]string{
			"zeta": "svc",
			"beta": "svc",
			"aa":   "svc",
			"ab":   "svc",
		},
	}

	alias := configuredServiceCallAlias(cfg, "svc")
	if alias != "aa" {
		t.Fatalf("expected deterministic shortest+lexicographic alias 'aa', got %q", alias)
	}
}

func TestConfiguredServiceCallAliasSkipsInvalidConfiguredAlias(t *testing.T) {
	cfg := &config.KimbapConfig{
		Aliases: map[string]string{
			"1":      "svc",
			"valida": "svc",
		},
	}

	alias := configuredServiceCallAlias(cfg, "svc")
	if alias != "valida" {
		t.Fatalf("expected invalid configured alias to be ignored, got %q", alias)
	}
}

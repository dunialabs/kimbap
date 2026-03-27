package main

import "testing"

func TestLinkOAuthConnectionStatus_DefaultFallbackUsesLegacyRowOnly(t *testing.T) {
	states := []connectorStateRow{
		{Name: "github:work", Provider: "github", Status: "healthy"},
	}

	status := linkOAuthConnectionStatus("github", "default", states)
	if status != "not_connected" {
		t.Fatalf("expected not_connected when only non-default row exists, got %q", status)
	}
}

func TestLinkOAuthConnectionStatus_DefaultFallbackAcceptsLegacyRow(t *testing.T) {
	states := []connectorStateRow{
		{Name: "github", Provider: "github", Status: "healthy"},
	}

	status := linkOAuthConnectionStatus("github", "default", states)
	if status != "connected" {
		t.Fatalf("expected connected for default legacy row, got %q", status)
	}
}

func TestLinkOAuthConnectionStatus_NonDefaultNoProviderFallback(t *testing.T) {
	states := []connectorStateRow{
		{Name: "github", Provider: "github", Status: "healthy"},
	}

	status := linkOAuthConnectionStatus("github", "work", states)
	if status != "not_connected" {
		t.Fatalf("expected not_connected for missing non-default row, got %q", status)
	}
}

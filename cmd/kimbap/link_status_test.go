package main

import (
	"strings"
	"testing"
)

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
		{Name: "github", Provider: "github", Status: "healthy", AccessToken: "tok"},
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

func TestLinkOAuthConnectionStatus_RefreshFailedTakesPrecedence(t *testing.T) {
	last := "2026-03-27T00:00:00Z"
	states := []connectorStateRow{{
		Name:             "github",
		Provider:         "github",
		Status:           "healthy",
		LastRefresh:      &last,
		LastRefreshError: "refresh failed",
	}}

	status := linkOAuthConnectionStatus("github", "default", states)
	if status != "refresh_failed" {
		t.Fatalf("expected refresh_failed, got %q", status)
	}
}

func TestLinkOAuthConnectionStatus_RevokedTakesPrecedence(t *testing.T) {
	revoked := "2026-03-27T00:00:00Z"
	states := []connectorStateRow{{
		Name:      "github",
		Provider:  "github",
		Status:    "healthy",
		RevokedAt: &revoked,
	}}

	status := linkOAuthConnectionStatus("github", "default", states)
	if status != "revoked" {
		t.Fatalf("expected revoked, got %q", status)
	}
}

func TestLinkConnectFooterZeroActionable(t *testing.T) {
	footer := linkConnectFooter(nil)
	if !strings.Contains(footer, "<service>") {
		t.Fatalf("expected generic footer for zero actionable, got %q", footer)
	}
}

func TestLinkConnectFooterOneActionable(t *testing.T) {
	footer := linkConnectFooter([]string{"stripe"})
	if footer != "Run 'kimbap link stripe' to connect." {
		t.Fatalf("unexpected footer: %q", footer)
	}
}

func TestLinkConnectFooterTwoActionable(t *testing.T) {
	footer := linkConnectFooter([]string{"stripe", "github"})
	if !strings.Contains(footer, "stripe") || !strings.Contains(footer, "github") {
		t.Fatalf("expected both services in footer, got %q", footer)
	}
	if strings.Contains(footer, "<service>") {
		t.Fatalf("expected specific services, not placeholder, got %q", footer)
	}
}

func TestLinkConnectFooterFourPlusActionable(t *testing.T) {
	footer := linkConnectFooter([]string{"a", "b", "c", "d"})
	if !strings.Contains(footer, "4 services") {
		t.Fatalf("expected count in footer for 4+ actionable, got %q", footer)
	}
	if !strings.Contains(footer, "<service>") {
		t.Fatalf("expected generic placeholder for 4+ actionable, got %q", footer)
	}
}

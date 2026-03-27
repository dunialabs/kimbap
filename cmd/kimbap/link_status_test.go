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

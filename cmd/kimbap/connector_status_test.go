package main

import (
	"testing"
	"time"
)

func TestConnectorComputedStatus_UsesDerivedSemantics(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-10 * time.Minute).Format(timeLayoutRFC3339)
	soon := now.Add(2 * time.Minute).Format(timeLayoutRFC3339)
	future := now.Add(30 * time.Minute).Format(timeLayoutRFC3339)
	refresh := now.Add(-1 * time.Minute).Format(timeLayoutRFC3339)

	tests := []struct {
		name string
		row  connectorStateRow
		want string
	}{
		{
			name: "revoked takes precedence",
			row: connectorStateRow{
				Status:      "healthy",
				AccessToken: "tok",
				ExpiresAt:   &future,
				RevokedAt:   &refresh,
			},
			want: "revoked",
		},
		{
			name: "refresh failure takes precedence",
			row: connectorStateRow{
				Status:           "healthy",
				AccessToken:      "tok",
				ExpiresAt:        &future,
				LastRefresh:      &refresh,
				LastRefreshError: "network error",
			},
			want: "refresh_failed",
		},
		{
			name: "expired when token present and expires_at in past",
			row: connectorStateRow{
				Status:      "healthy",
				AccessToken: "tok",
				ExpiresAt:   &past,
			},
			want: "expired",
		},
		{
			name: "degraded when expiring soon",
			row: connectorStateRow{
				Status:      "healthy",
				AccessToken: "tok",
				ExpiresAt:   &soon,
			},
			want: "degraded",
		},
		{
			name: "connected when healthy and future expiry",
			row: connectorStateRow{
				Status:      "healthy",
				AccessToken: "tok",
				ExpiresAt:   &future,
			},
			want: "connected",
		},
		{
			name: "connecting when access token missing",
			row: connectorStateRow{
				Status: "healthy",
			},
			want: "connecting",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := connectorComputedStatus(tc.row); got != tc.want {
				t.Fatalf("connectorComputedStatus()=%q, want %q", got, tc.want)
			}
		})
	}
}

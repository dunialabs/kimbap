package api

import (
	"context"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/webhooks"
)

func TestValidateWebhookURLRejectsLoopbackHost(t *testing.T) {
	err := validateWebhookURL(context.Background(), "https://127.0.0.1/hook")
	if err == nil {
		t.Fatal("expected loopback webhook URL to be rejected")
	}
}

func TestValidateWebhookURLWithCanceledContextStillValidatesFormat(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := validateWebhookURL(ctx, "https://example.com/hook")
	if err != nil {
		t.Fatalf("expected canceled context to not fail URL validation for public host, got %v", err)
	}
}

func TestDedupeAndTrimWebhookEventsPreservesNewestAndLimit(t *testing.T) {
	now := time.Now().UTC()
	items := []webhooks.Event{
		{ID: "evt-1", Timestamp: now.Add(-2 * time.Second), Type: webhooks.EventTokenCreated, TenantID: "tenant-a"},
		{ID: "evt-1", Timestamp: now.Add(-1 * time.Second), Type: webhooks.EventTokenDeleted, TenantID: "tenant-a"},
		{ID: "evt-2", Timestamp: now, Type: webhooks.EventPolicyCreated, TenantID: "tenant-a"},
	}

	filtered := dedupeAndTrimWebhookEvents(items, 2)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 events, got %d", len(filtered))
	}
	if filtered[0].ID != "evt-1" || filtered[0].Type != webhooks.EventTokenDeleted {
		t.Fatalf("expected deduped older entry first, got %+v", filtered[0])
	}
	if filtered[1].ID != "evt-2" {
		t.Fatalf("expected newest event last, got %s", filtered[1].ID)
	}
}

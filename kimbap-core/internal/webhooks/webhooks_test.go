package webhooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDispatcherEmitAndDeliver(t *testing.T) {
	received := make(chan Event, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var evt Event
		_ = json.NewDecoder(r.Body).Decode(&evt)
		received <- evt
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher()
	d.Subscribe(Subscription{
		ID:  "test-1",
		URL: server.URL,
	})

	d.Emit(EventTokenCreated, map[string]string{"user_id": "u1"})

	select {
	case evt := <-received:
		if evt.Type != EventTokenCreated {
			t.Fatalf("expected token.created, got %s", evt.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("webhook delivery timed out")
	}
}

func TestDispatcherEventFiltering(t *testing.T) {
	calls := make(chan struct{}, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher()
	d.Subscribe(Subscription{
		ID:     "filtered",
		URL:    server.URL,
		Events: []EventType{EventPolicyCreated},
	})

	d.Emit(EventTokenCreated, nil)
	d.Emit(EventPolicyCreated, nil)

	select {
	case <-calls:
	case <-time.After(2 * time.Second):
		t.Fatal("expected policy event delivery")
	}

	select {
	case <-calls:
		t.Fatal("token event should not have been delivered to filtered subscription")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestDispatcherHMACSignature(t *testing.T) {
	received := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher()
	d.Subscribe(Subscription{
		ID:     "signed",
		URL:    server.URL,
		Secret: "test-secret",
	})

	d.Emit(EventSkillInstalled, map[string]string{"name": "github"})

	select {
	case headers := <-received:
		sig := headers.Get("X-Kimbap-Signature")
		if sig == "" {
			t.Fatal("expected X-Kimbap-Signature header")
		}
		if len(sig) < 10 || sig[:7] != "sha256=" {
			t.Fatalf("unexpected signature format: %s", sig)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("webhook delivery timed out")
	}
}

func TestDispatcherRecentEvents(t *testing.T) {
	d := NewDispatcher()
	d.Emit(EventTokenCreated, nil)
	d.Emit(EventTokenDeleted, nil)
	d.Emit(EventPolicyCreated, nil)

	events := d.RecentEvents(2)
	if len(events) != 2 {
		t.Fatalf("expected 2 recent events, got %d", len(events))
	}
	if events[0].Type != EventTokenDeleted {
		t.Fatalf("expected second-to-last event, got %s", events[0].Type)
	}
}

func TestDispatcherUnsubscribe(t *testing.T) {
	d := NewDispatcher()
	d.Subscribe(Subscription{ID: "x", URL: "http://example.com"})
	d.Unsubscribe("x")
	subs := d.ListSubscriptions()
	if len(subs) != 0 {
		t.Fatalf("expected no active subs, got %d", len(subs))
	}
}

func TestDispatcherTenantIsolation(t *testing.T) {
	d := NewDispatcher()
	d.Subscribe(Subscription{ID: "a1", URL: "http://example.com/a", TenantID: "tenant-a"})
	d.Subscribe(Subscription{ID: "b1", URL: "http://example.com/b", TenantID: "tenant-b"})

	subsA := d.ListSubscriptionsByTenant("tenant-a")
	if len(subsA) != 1 || subsA[0].ID != "a1" {
		t.Fatalf("expected 1 sub for tenant-a, got %+v", subsA)
	}
	subsB := d.ListSubscriptionsByTenant("tenant-b")
	if len(subsB) != 1 || subsB[0].ID != "b1" {
		t.Fatalf("expected 1 sub for tenant-b, got %+v", subsB)
	}
}

func TestDispatcherTenantEventFiltering(t *testing.T) {
	d := NewDispatcher()

	d.mu.Lock()
	d.events = append(d.events,
		Event{ID: "e1", Type: EventTokenCreated, TenantID: "t1"},
		Event{ID: "e2", Type: EventTokenDeleted, TenantID: "t2"},
		Event{ID: "e3", Type: EventPolicyCreated, TenantID: "t1"},
	)
	d.mu.Unlock()

	t1Events := d.RecentEventsByTenant("t1", 10)
	if len(t1Events) != 2 {
		t.Fatalf("expected 2 events for t1, got %d", len(t1Events))
	}
	t2Events := d.RecentEventsByTenant("t2", 10)
	if len(t2Events) != 1 {
		t.Fatalf("expected 1 event for t2, got %d", len(t2Events))
	}
}

func TestSubscribeDuplicateIDOverwrites(t *testing.T) {
	d := NewDispatcher()
	d.Subscribe(Subscription{ID: "dup", URL: "http://example.com/v1", TenantID: "t1"})
	d.Subscribe(Subscription{ID: "dup", URL: "http://example.com/v2", TenantID: "t1"})

	subs := d.ListSubscriptions()
	if len(subs) != 1 {
		t.Fatalf("expected 1 sub after duplicate, got %d", len(subs))
	}
	if subs[0].URL != "http://example.com/v2" {
		t.Fatalf("expected overwritten URL, got %s", subs[0].URL)
	}
}

func TestUnsubscribeByTenantIsolation(t *testing.T) {
	d := NewDispatcher()
	d.Subscribe(Subscription{ID: "shared", URL: "http://example.com/a", TenantID: "t1"})
	d.Subscribe(Subscription{ID: "shared-b", URL: "http://example.com/b", TenantID: "t2"})

	d.UnsubscribeByTenant("shared", "t2")
	subsT1 := d.ListSubscriptionsByTenant("t1")
	if len(subsT1) != 1 {
		t.Fatal("unsubscribe by wrong tenant should not affect other tenants")
	}

	d.UnsubscribeByTenant("shared", "t1")
	subsT1 = d.ListSubscriptionsByTenant("t1")
	if len(subsT1) != 0 {
		t.Fatal("unsubscribe by correct tenant should deactivate")
	}
}

func TestEmitForTenantOnlyDeliversToSameTenant(t *testing.T) {
	t1Calls := make(chan struct{}, 10)
	t2Calls := make(chan struct{}, 10)

	t1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t1Calls <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer t1Server.Close()

	t2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t2Calls <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer t2Server.Close()

	d := NewDispatcher()
	d.Subscribe(Subscription{ID: "s1", URL: t1Server.URL, TenantID: "tenant-1"})
	d.Subscribe(Subscription{ID: "s2", URL: t2Server.URL, TenantID: "tenant-2"})

	d.EmitForTenant("tenant-1", EventTokenCreated, nil)

	select {
	case <-t1Calls:
	case <-time.After(2 * time.Second):
		t.Fatal("expected delivery to tenant-1 subscription")
	}

	select {
	case <-t2Calls:
		t.Fatal("tenant-2 subscription should NOT receive tenant-1 event")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestEmitForTenantRecordsTenantID(t *testing.T) {
	d := NewDispatcher()
	d.EmitForTenant("t42", EventPolicyCreated, nil)

	events := d.RecentEventsByTenant("t42", 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event for t42, got %d", len(events))
	}
	if events[0].TenantID != "t42" {
		t.Fatalf("expected tenant t42, got %s", events[0].TenantID)
	}
}

func TestSubscribeCrossTenantIDDoesNotOverwrite(t *testing.T) {
	d := NewDispatcher()
	d.Subscribe(Subscription{ID: "same-id", URL: "http://example.com/t1", TenantID: "t1"})
	d.Subscribe(Subscription{ID: "same-id", URL: "http://example.com/t2", TenantID: "t2"})

	subs := d.ListSubscriptions()
	if len(subs) != 2 {
		t.Fatalf("expected 2 subs (different tenants, same ID), got %d", len(subs))
	}
}

func TestUnsubscribeDeactivatesAllMatches(t *testing.T) {
	d := NewDispatcher()
	d.mu.Lock()
	d.subscriptions = append(d.subscriptions,
		Subscription{ID: "x", URL: "http://example.com/1", Active: true},
		Subscription{ID: "x", URL: "http://example.com/2", Active: true},
	)
	d.mu.Unlock()

	d.Unsubscribe("x")
	subs := d.ListSubscriptions()
	if len(subs) != 0 {
		t.Fatalf("expected all matches deactivated, got %d active", len(subs))
	}
}

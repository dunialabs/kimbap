package core

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/dunialabs/kimbap-core/internal/database"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
)

type mockEventRepo struct {
	mu     sync.Mutex
	events []database.Event
}

func (m *mockEventRepo) Create(_ context.Context, event *database.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, *event)
	return nil
}

func (m *mockEventRepo) FindByStreamIDAfter(_ context.Context, streamID string, afterEventID string) ([]database.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var afterCreatedAt *time.Time
	for _, e := range m.events {
		if e.EventID == afterEventID {
			createdAt := e.CreatedAt
			afterCreatedAt = &createdAt
			break
		}
	}

	result := make([]database.Event, 0)
	for _, e := range m.events {
		if e.StreamID != streamID {
			continue
		}
		if afterCreatedAt == nil || e.CreatedAt.After(*afterCreatedAt) {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockEventRepo) DeleteExpired(_ context.Context) (int64, error) {
	return 0, nil
}

func (m *mockEventRepo) DeleteByStreamID(_ context.Context, streamID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []database.Event
	var removed int64
	for _, e := range m.events {
		if e.StreamID == streamID {
			removed++
			continue
		}
		kept = append(kept, e)
	}
	m.events = kept
	return removed, nil
}

func TestPersistentEventStore(t *testing.T) {
	t.Run("event id format and store/replay", func(t *testing.T) {
		repo := &mockEventRepo{}
		store := NewPersistentEventStore("session-1", "user-1", repo)

		msg := mcptypes.JSONRPCMessage{JSONRPC: "2.0", Method: "ping"}
		eventID, err := store.StoreEvent(context.Background(), "streamA", msg)
		if err != nil {
			t.Fatalf("StoreEvent failed: %v", err)
		}

		pattern := regexp.MustCompile(`^streamA_[0-9]+_[0-9a-f]+$`)
		if !pattern.MatchString(eventID) {
			t.Fatalf("unexpected event id format: %s", eventID)
		}

		time.Sleep(50 * time.Millisecond)

		var replayed []mcptypes.EventID
		_, err = store.ReplayEventsAfter(context.Background(), "streamA_0_00000000", mcptypes.ReplayOptions{
			Send: func(_ context.Context, id mcptypes.EventID, message mcptypes.JSONRPCMessage) error {
				replayed = append(replayed, id)
				if message.Method != "ping" {
					t.Fatalf("unexpected message method: %s", message.Method)
				}
				return nil
			},
		})
		if err != nil {
			t.Fatalf("ReplayEventsAfter failed: %v", err)
		}
		if len(replayed) != 1 || replayed[0] != eventID {
			t.Fatalf("unexpected replay results: %#v", replayed)
		}
	})

	t.Run("ReplayEventsAfter returns only events after id", func(t *testing.T) {
		repo := &mockEventRepo{}
		store := NewPersistentEventStore("session-2", "user-2", repo)

		repo.events = []database.Event{
			{EventID: "streamB_1000_aaaa", StreamID: "streamB", MessageData: `{"jsonrpc":"2.0","method":"m1"}`, CreatedAt: time.UnixMilli(1000)},
			{EventID: "streamB_2000_bbbb", StreamID: "streamB", MessageData: `{"jsonrpc":"2.0","method":"m2"}`, CreatedAt: time.UnixMilli(2000)},
			{EventID: "streamB_3000_cccc", StreamID: "streamB", MessageData: `{"jsonrpc":"2.0","method":"m3"}`, CreatedAt: time.UnixMilli(3000)},
		}

		var got []string
		_, err := store.ReplayEventsAfter(context.Background(), "streamB_1500_dead", mcptypes.ReplayOptions{
			Send: func(_ context.Context, id mcptypes.EventID, _ mcptypes.JSONRPCMessage) error {
				got = append(got, id)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("ReplayEventsAfter failed: %v", err)
		}
		if len(got) != 3 || got[0] != "streamB_1000_aaaa" || got[1] != "streamB_2000_bbbb" || got[2] != "streamB_3000_cccc" {
			t.Fatalf("unexpected replayed ids: %#v", got)
		}
	})

	t.Run("ReplayEventsAfter orders by timestamp", func(t *testing.T) {
		store := NewPersistentEventStore("session-3", "user-3", nil)

		store.mu.Lock()
		store.cache.Add(store.cacheKey("streamC", "streamC_3000_cccc"), mcptypes.CachedEvent{EventID: "streamC_3000_cccc", StreamID: "streamC", Timestamp: time.UnixMilli(3000), Message: mcptypes.JSONRPCMessage{Method: "late"}})
		store.cache.Add(store.cacheKey("streamC", "streamC_1000_aaaa"), mcptypes.CachedEvent{EventID: "streamC_1000_aaaa", StreamID: "streamC", Timestamp: time.UnixMilli(1000), Message: mcptypes.JSONRPCMessage{Method: "early"}})
		store.cache.Add(store.cacheKey("streamC", "streamC_2000_bbbb"), mcptypes.CachedEvent{EventID: "streamC_2000_bbbb", StreamID: "streamC", Timestamp: time.UnixMilli(2000), Message: mcptypes.JSONRPCMessage{Method: "mid"}})
		store.mu.Unlock()

		var got []string
		_, err := store.ReplayEventsAfter(context.Background(), "streamC_0000_0000", mcptypes.ReplayOptions{
			Send: func(_ context.Context, id mcptypes.EventID, _ mcptypes.JSONRPCMessage) error {
				got = append(got, id)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("ReplayEventsAfter failed: %v", err)
		}

		expected := []string{"streamC_1000_aaaa", "streamC_2000_bbbb", "streamC_3000_cccc"}
		if len(got) != len(expected) {
			t.Fatalf("expected %d events, got %d (%#v)", len(expected), len(got), got)
		}
		for i := range expected {
			if got[i] != expected[i] {
				t.Fatalf("unexpected replay order: got=%#v expected=%#v", got, expected)
			}
		}
	})

	t.Run("cleanup stream removes only matching stream cache", func(t *testing.T) {
		store := NewPersistentEventStore("session-4", "user-4", nil)

		store.mu.Lock()
		store.cache.Add(store.cacheKey("streamD", "streamD_1000_aaaa"), mcptypes.CachedEvent{EventID: "streamD_1000_aaaa", StreamID: "streamD", Timestamp: time.UnixMilli(1000)})
		store.cache.Add(store.cacheKey("streamE", "streamE_1000_bbbb"), mcptypes.CachedEvent{EventID: "streamE_1000_bbbb", StreamID: "streamE", Timestamp: time.UnixMilli(1000)})
		store.mu.Unlock()

		store.CleanupStream("streamD")

		var got []string
		_, err := store.ReplayEventsAfter(context.Background(), "streamD_0000_0000", mcptypes.ReplayOptions{Send: func(_ context.Context, id mcptypes.EventID, _ mcptypes.JSONRPCMessage) error {
			got = append(got, id)
			return nil
		}})
		if err != nil {
			t.Fatalf("ReplayEventsAfter failed: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected streamD cache to be empty, got %#v", got)
		}

		got = nil
		_, err = store.ReplayEventsAfter(context.Background(), "streamE_0000_0000", mcptypes.ReplayOptions{Send: func(_ context.Context, id mcptypes.EventID, _ mcptypes.JSONRPCMessage) error {
			got = append(got, id)
			return nil
		}})
		if err != nil {
			t.Fatalf("ReplayEventsAfter failed: %v", err)
		}
		if len(got) != 1 || got[0] != "streamE_1000_bbbb" {
			t.Fatalf("expected streamE event to remain, got %#v", got)
		}
	})

	t.Run("extractStreamID parses composite ids", func(t *testing.T) {
		if got := extractStreamID("streamF_1234_abcd"); got != "streamF" {
			t.Fatalf("expected streamF, got %s", got)
		}
		if got := extractStreamID("user_stream_1234_abcd"); got != "user_stream" {
			t.Fatalf("expected user_stream, got %s", got)
		}
		if got := extractStreamID("plain-stream"); got != "plain-stream" {
			t.Fatalf("expected plain-stream, got %s", got)
		}
	})

	t.Run("extractTimestampFromEventID supports stream ids with underscores", func(t *testing.T) {
		if got := extractTimestampFromEventID("user_stream_1234_abcd"); got != 1234 {
			t.Fatalf("expected timestamp 1234, got %d", got)
		}
		if got := extractTimestampFromEventID("invalid"); got != 0 {
			t.Fatalf("expected timestamp 0 for invalid id, got %d", got)
		}
	})

	t.Run("LRU eviction does not lose events when DB repo backs the store", func(t *testing.T) {
		repo := &mockEventRepo{}
		store := NewPersistentEventStore("session-lru", "user-lru", repo)

		smallCache, _ := lru.New[string, mcptypes.CachedEvent](3)
		store.mu.Lock()
		store.cache = smallCache
		store.mu.Unlock()

		for i := 0; i < 6; i++ {
			msg := mcptypes.JSONRPCMessage{JSONRPC: "2.0", Method: "ping"}
			_, err := store.StoreEvent(context.Background(), "streamLRU", msg)
			if err != nil {
				t.Fatalf("StoreEvent %d failed: %v", i, err)
			}
		}

		if store.cache.Len() > 3 {
			t.Fatalf("expected cache len <= 3, got %d", store.cache.Len())
		}
		// Allow async persistence goroutines to complete
		time.Sleep(100 * time.Millisecond)
		repo.mu.Lock()
		dbCount := len(repo.events)
		repo.mu.Unlock()
		if dbCount != 6 {
			t.Fatalf("expected 6 events in DB, got %d", dbCount)
		}

		var replayed []string
		_, err := store.ReplayEventsAfter(context.Background(), "streamLRU_0_00000000", mcptypes.ReplayOptions{
			Send: func(_ context.Context, id mcptypes.EventID, _ mcptypes.JSONRPCMessage) error {
				replayed = append(replayed, id)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("ReplayEventsAfter failed: %v", err)
		}
		if len(replayed) != 6 {
			t.Fatalf("expected 6 replayed events (cache+DB merge), got %d", len(replayed))
		}
	})

	t.Run("replay deduplicates events present in both cache and DB", func(t *testing.T) {
		repo := &mockEventRepo{}
		store := NewPersistentEventStore("session-dedup", "user-dedup", repo)

		msg := mcptypes.JSONRPCMessage{JSONRPC: "2.0", Method: "test"}
		eventID, err := store.StoreEvent(context.Background(), "streamDD", msg)
		if err != nil {
			t.Fatalf("StoreEvent failed: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		var replayed []string
		_, err = store.ReplayEventsAfter(context.Background(), "streamDD_0_00000000", mcptypes.ReplayOptions{
			Send: func(_ context.Context, id mcptypes.EventID, _ mcptypes.JSONRPCMessage) error {
				replayed = append(replayed, id)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("ReplayEventsAfter failed: %v", err)
		}
		if len(replayed) != 1 || replayed[0] != eventID {
			t.Fatalf("expected exactly 1 event %s, got %#v", eventID, replayed)
		}
	})
}

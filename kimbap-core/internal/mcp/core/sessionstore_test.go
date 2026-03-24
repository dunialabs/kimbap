package core

import (
	"context"
	"testing"

	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
)

type noopSessionLogger struct{}

func (noopSessionLogger) LogClientRequest(_ context.Context, _ map[string]any) error   { return nil }
func (noopSessionLogger) LogReverseRequest(_ context.Context, _ map[string]any) error  { return nil }
func (noopSessionLogger) LogServerLifecycle(_ context.Context, _ map[string]any) error { return nil }
func (noopSessionLogger) LogError(_ context.Context, _ map[string]any) error           { return nil }
func (noopSessionLogger) LogSessionLifecycle(_ int, _ string)                          {}
func (noopSessionLogger) IP() string                                                   { return "127.0.0.1" }

func newTestSessionStore() *SessionStore {
	return &SessionStore{
		sessions:       map[string]*ClientSession{},
		proxySessions:  map[string]*ProxySession{},
		userSessions:   map[string]map[string]struct{}{},
		eventStores:    map[string]*PersistentEventStore{},
		sessionLoggers: map[string]SessionLogger{},
	}
}

func TestSessionStore_Lifecycle(t *testing.T) {
	t.Run("CreateSession adds session to store and GetSession returns it", func(t *testing.T) {
		store := newTestSessionStore()
		session, err := store.CreateSession(context.Background(), "s1", "u1", "token", mcptypes.AuthContext{UserID: "u1"}, noopSessionLogger{})
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
		if session == nil {
			t.Fatalf("expected created session")
		}

		got := store.GetSession("s1")
		if got == nil || got.SessionID != "s1" || got.UserID != "u1" {
			t.Fatalf("unexpected session from GetSession: %#v", got)
		}
		if store.GetProxySession("s1") == nil {
			t.Fatalf("expected proxy session to be created")
		}
		if store.GetEventStore("s1") == nil {
			t.Fatalf("expected event store to be created")
		}
	})

	t.Run("GetUserSessions returns all sessions for a user", func(t *testing.T) {
		store := newTestSessionStore()
		_, _ = store.CreateSession(context.Background(), "s1", "u1", "t1", mcptypes.AuthContext{UserID: "u1"}, noopSessionLogger{})
		_, _ = store.CreateSession(context.Background(), "s2", "u1", "t2", mcptypes.AuthContext{UserID: "u1"}, noopSessionLogger{})
		_, _ = store.CreateSession(context.Background(), "s3", "u2", "t3", mcptypes.AuthContext{UserID: "u2"}, noopSessionLogger{})

		got := store.GetUserSessions("u1")
		if len(got) != 2 {
			t.Fatalf("expected 2 sessions for u1, got %d", len(got))
		}
	})

	t.Run("RemoveSession removes from all maps", func(t *testing.T) {
		store := newTestSessionStore()
		_, _ = store.CreateSession(context.Background(), "s1", "u1", "token", mcptypes.AuthContext{UserID: "u1"}, noopSessionLogger{})

		store.RemoveSession("s1", mcptypes.DisconnectReasonClientDisconnect, false)

		if store.GetSession("s1") != nil {
			t.Fatalf("expected session to be removed")
		}
		if store.GetProxySession("s1") != nil {
			t.Fatalf("expected proxy session to be removed")
		}
		if store.GetEventStore("s1") != nil {
			t.Fatalf("expected event store to be removed")
		}
		if store.GetSessionLogger("s1") != nil {
			t.Fatalf("expected session logger to be removed")
		}
		if len(store.userSessions) != 0 {
			t.Fatalf("expected userSessions map to be cleaned up, got %#v", store.userSessions)
		}
	})

	t.Run("TotalCreated increments on create and does not decrement on remove", func(t *testing.T) {
		store := newTestSessionStore()
		if got := store.TotalCreated(); got != 0 {
			t.Fatalf("expected initial totalCreated=0, got %d", got)
		}

		_, _ = store.CreateSession(context.Background(), "s1", "u1", "t1", mcptypes.AuthContext{UserID: "u1"}, noopSessionLogger{})
		_, _ = store.CreateSession(context.Background(), "s2", "u1", "t2", mcptypes.AuthContext{UserID: "u1"}, noopSessionLogger{})
		if got := store.TotalCreated(); got != 2 {
			t.Fatalf("expected totalCreated=2 after creates, got %d", got)
		}

		store.RemoveSession("s1", mcptypes.DisconnectReasonClientDisconnect, false)
		if got := store.TotalCreated(); got != 2 {
			t.Fatalf("expected totalCreated to stay 2 after remove, got %d", got)
		}
	})

	t.Run("Stop is safe to call multiple times", func(t *testing.T) {
		store := newTestSessionStore()
		store.startCleanupTimer()
		store.Stop()
		store.Stop()
	})
}

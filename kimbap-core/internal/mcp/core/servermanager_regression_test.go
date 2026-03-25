package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	serverlog "github.com/dunialabs/kimbap-core/internal/log"
)

func TestCloseTemporaryServerKeepsTrackingUntilCloseCompletes(t *testing.T) {
	mgr := &serverManager{
		temporaryServers: map[string]*ServerContext{},
		serverLoggers:    map[string]*serverlog.ServerLogger{},
	}

	serverID := "srv-temp"
	userID := "user-temp"
	key := tempServerKey(serverID, userID)

	ctxObj := NewServerContext(database.Server{ServerID: serverID})
	transport := &blockingTerminateTransport{release: make(chan struct{}), closed: make(chan struct{})}
	ctxObj.Transport = transport
	mgr.temporaryServers[key] = ctxObj

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mgr.CloseTemporaryServer(cancelledCtx, serverID, userID)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if got := mgr.GetTemporaryServer(serverID, userID); got == nil {
		t.Fatal("expected temporary server to remain tracked until close completes")
	}

	close(transport.release)
	select {
	case <-transport.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for transport close")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if mgr.GetTemporaryServer(serverID, userID) == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected temporary server cleanup after close completion")
}

func TestRemoveTemporaryServerIfUnchangedProtectsReplacement(t *testing.T) {
	mgr := &serverManager{
		temporaryServers: map[string]*ServerContext{},
		serverLoggers:    map[string]*serverlog.ServerLogger{},
	}

	key := tempServerKey("srv-a", "user-a")
	oldCtx := NewServerContext(database.Server{ServerID: "srv-a"})
	newCtx := NewServerContext(database.Server{ServerID: "srv-a"})
	oldLogger := serverlog.NewServerLogger(key + "-old")
	newLogger := serverlog.NewServerLogger(key + "-new")

	mgr.temporaryServers[key] = newCtx
	mgr.serverLoggers[key] = newLogger

	mgr.removeTemporaryServerIfUnchanged(key, oldCtx, oldLogger)

	if got := mgr.temporaryServers[key]; got != newCtx {
		t.Fatal("expected replacement temporary server to remain tracked")
	}
	if got := mgr.serverLoggers[key]; got != newLogger {
		t.Fatal("expected replacement server logger to remain tracked")
	}

	mgr.removeTemporaryServerIfUnchanged(key, newCtx, newLogger)

	if got := mgr.temporaryServers[key]; got != nil {
		t.Fatal("expected temporary server to be removed when pointers match")
	}
	if got := mgr.serverLoggers[key]; got != nil {
		t.Fatal("expected server logger to be removed when pointers match")
	}
}

type blockingTerminateTransport struct {
	release chan struct{}
	closed  chan struct{}
}

func (t *blockingTerminateTransport) TerminateSession() error {
	<-t.release
	return nil
}

func (t *blockingTerminateTransport) Close() error {
	select {
	case <-t.closed:
	default:
		close(t.closed)
	}
	return nil
}

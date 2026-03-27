package jobs

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type mockExpirer struct {
	mu      sync.Mutex
	calls   int
	calledC chan struct{}
}

func (m *mockExpirer) ExpireStale(context.Context) (int, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if m.calledC != nil {
		select {
		case m.calledC <- struct{}{}:
		default:
		}
	}
	return 1, nil
}

func (m *mockExpirer) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestWorkerStartStopCleanly(t *testing.T) {
	m := &mockExpirer{calledC: make(chan struct{}, 8)}
	w := NewWorker(10*time.Millisecond, m, slog.Default())
	w.Start(t.Context())

	select {
	case <-m.calledC:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("worker did not execute expiry job")
	}

	stopped := make(chan struct{})
	go func() {
		w.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("worker stop did not return")
	}
}

func TestWorkerCallsExpirerAtInterval(t *testing.T) {
	m := &mockExpirer{calledC: make(chan struct{}, 16)}
	w := NewWorker(15*time.Millisecond, m, slog.Default())
	w.Start(t.Context())
	defer w.Stop()

	deadline := time.After(350 * time.Millisecond)
	for m.callCount() < 2 {
		select {
		case <-m.calledC:
		case <-deadline:
			t.Fatalf("expected at least 2 expiry calls, got %d", m.callCount())
		}
	}
}

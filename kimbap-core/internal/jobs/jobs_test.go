package jobs

import (
	"testing"
	"time"
)

func TestNewWorkerDefaults(t *testing.T) {
	w := NewWorker(0, nil, nil)
	if w == nil {
		t.Fatal("expected worker")
	}
	if w.interval != time.Minute {
		t.Fatalf("expected default interval 1m, got %s", w.interval)
	}
}

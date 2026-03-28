package approvals

import (
	"context"
	"sync"
	"time"
)

type HeldExecution struct {
	ApprovalID string
	TenantID   string
	RequestID  string
	Payload    map[string]any
	CreatedAt  time.Time
}

type HeldExecutionStore interface {
	Save(ctx context.Context, execution HeldExecution) error
	Get(ctx context.Context, approvalID string) (*HeldExecution, error)
	Delete(ctx context.Context, approvalID string) error
}

type MemoryHeldExecutionStore struct {
	mu    sync.RWMutex
	items map[string]HeldExecution
}

func NewMemoryHeldExecutionStore() *MemoryHeldExecutionStore {
	return &MemoryHeldExecutionStore{items: map[string]HeldExecution{}}
}

func (s *MemoryHeldExecutionStore) Save(_ context.Context, execution HeldExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execution.CreatedAt.IsZero() {
		execution.CreatedAt = time.Now().UTC()
	}
	if execution.Payload != nil {
		execution.Payload = deepCopyMap(execution.Payload)
	}
	s.items[execution.ApprovalID] = execution
	return nil
}

func (s *MemoryHeldExecutionStore) Get(_ context.Context, approvalID string) (*HeldExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execution, ok := s.items[approvalID]
	if !ok {
		return nil, nil
	}
	copyValue := execution
	if execution.Payload != nil {
		copyValue.Payload = deepCopyMap(execution.Payload)
	}
	return &copyValue, nil
}

func (s *MemoryHeldExecutionStore) Delete(_ context.Context, approvalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, approvalID)
	return nil
}

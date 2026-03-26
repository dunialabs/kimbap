package app

import (
	"context"
	"sync"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/runtime"
)

type memoryHeldExecutionStore struct {
	mu   sync.Mutex
	held map[string]actions.ExecutionRequest
}

func NewMemoryHeldExecutionStore() runtime.HeldExecutionStore {
	return &memoryHeldExecutionStore{held: map[string]actions.ExecutionRequest{}}
}

func (s *memoryHeldExecutionStore) Hold(_ context.Context, approvalRequestID string, req actions.ExecutionRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held[approvalRequestID] = req
	return nil
}

func (s *memoryHeldExecutionStore) Resume(_ context.Context, approvalRequestID string) (*actions.ExecutionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.held[approvalRequestID]
	if !ok {
		return nil, nil
	}
	delete(s.held, approvalRequestID)
	return &req, nil
}

func (s *memoryHeldExecutionStore) Remove(_ context.Context, approvalRequestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.held, approvalRequestID)
	return nil
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
)

type sqlHeldExecutionStore struct {
	st store.HeldExecutionStore
}

func NewSQLHeldExecutionStore(st store.HeldExecutionStore) runtime.HeldExecutionStore {
	return &sqlHeldExecutionStore{st: st}
}

func (s *sqlHeldExecutionStore) Hold(ctx context.Context, approvalRequestID string, req actions.ExecutionRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal held execution: %w", err)
	}
	return s.st.HoldExecution(ctx, approvalRequestID, b)
}

func (s *sqlHeldExecutionStore) Resume(ctx context.Context, approvalRequestID string) (*actions.ExecutionRequest, error) {
	b, err := s.st.ResumeExecution(ctx, approvalRequestID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var req actions.ExecutionRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return nil, fmt.Errorf("unmarshal held execution: %w", err)
	}
	return &req, nil
}

func (s *sqlHeldExecutionStore) Remove(ctx context.Context, approvalRequestID string) error {
	return s.st.RemoveExecution(ctx, approvalRequestID)
}

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

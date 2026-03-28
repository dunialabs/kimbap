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
	if s == nil || s.st == nil {
		return fmt.Errorf("held execution store is not initialized")
	}
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal held execution: %w", err)
	}
	return s.st.HoldExecution(ctx, approvalRequestID, b)
}

func (s *sqlHeldExecutionStore) Resume(ctx context.Context, approvalRequestID string) (*actions.ExecutionRequest, error) {
	if s == nil || s.st == nil {
		return nil, fmt.Errorf("held execution store is not initialized")
	}
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
	if s == nil || s.st == nil {
		return fmt.Errorf("held execution store is not initialized")
	}
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
	if s == nil {
		return fmt.Errorf("held execution store is not initialized")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held == nil {
		s.held = map[string]actions.ExecutionRequest{}
	}
	s.held[approvalRequestID] = req
	return nil
}

func (s *memoryHeldExecutionStore) Resume(_ context.Context, approvalRequestID string) (*actions.ExecutionRequest, error) {
	if s == nil {
		return nil, fmt.Errorf("held execution store is not initialized")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held == nil {
		return nil, nil
	}
	req, ok := s.held[approvalRequestID]
	if !ok {
		return nil, nil
	}
	delete(s.held, approvalRequestID)
	return &req, nil
}

func (s *memoryHeldExecutionStore) Remove(_ context.Context, approvalRequestID string) error {
	if s == nil {
		return fmt.Errorf("held execution store is not initialized")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held == nil {
		return nil
	}
	delete(s.held, approvalRequestID)
	return nil
}

package approvals

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MemoryApprovalStore struct {
	mu    sync.RWMutex
	items map[string]ApprovalRequest
}

func NewMemoryApprovalStore() *MemoryApprovalStore {
	return &MemoryApprovalStore{items: map[string]ApprovalRequest{}}
}

func (s *MemoryApprovalStore) Create(_ context.Context, req *ApprovalRequest) error {
	if req == nil {
		return fmt.Errorf("approval request is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[req.ID] = deepCopyApprovalRequest(*req)
	return nil
}

func (s *MemoryApprovalStore) Get(_ context.Context, id string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	cp := deepCopyApprovalRequest(req)
	return &cp, nil
}

func (s *MemoryApprovalStore) Update(_ context.Context, req *ApprovalRequest) error {
	if req == nil {
		return fmt.Errorf("approval request is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[req.ID] = deepCopyApprovalRequest(*req)
	return nil
}

func deepCopyApprovalRequest(src ApprovalRequest) ApprovalRequest {
	cp := src
	if src.Input != nil {
		cp.Input = make(map[string]any, len(src.Input))
		for k, v := range src.Input {
			cp.Input[k] = v
		}
	}
	if src.Votes != nil {
		cp.Votes = make([]ApprovalVote, len(src.Votes))
		copy(cp.Votes, src.Votes)
	}
	return cp
}

func (s *MemoryApprovalStore) ListPending(_ context.Context, tenantID string) ([]ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ApprovalRequest, 0)
	for _, item := range s.items {
		if item.TenantID == tenantID && item.Status == StatusPending {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *MemoryApprovalStore) ListAll(_ context.Context, tenantID string, filter ApprovalFilter) ([]ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ApprovalRequest, 0)
	for _, item := range s.items {
		if item.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && item.Status != *filter.Status {
			continue
		}
		if filter.AgentName != "" && item.AgentName != filter.AgentName {
			continue
		}
		if filter.Service != "" && item.Service != filter.Service {
			continue
		}
		out = append(out, item)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (s *MemoryApprovalStore) ExpireOld(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for id, item := range s.items {
		if item.Status == StatusPending && !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			item.Status = StatusExpired
			s.items[id] = item
			count++
		}
	}
	return count, nil
}

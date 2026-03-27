package approvals

import (
	"context"
	"encoding/json"
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
	if _, exists := s.items[req.ID]; exists {
		return fmt.Errorf("%w: %s", ErrAlreadyResolved, req.ID)
	}
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
	cp.Input = deepCopyMap(src.Input)
	if src.Votes != nil {
		cp.Votes = make([]ApprovalVote, len(src.Votes))
		copy(cp.Votes, src.Votes)
	}
	if src.ResolvedAt != nil {
		t := *src.ResolvedAt
		cp.ResolvedAt = &t
	}
	return cp
}

func deepCopyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	data, err := json.Marshal(src)
	if err != nil {
		cp := make(map[string]any, len(src))
		for k, v := range src {
			cp[k] = v
		}
		return cp
	}
	var dst map[string]any
	if err := json.Unmarshal(data, &dst); err != nil {
		cp := make(map[string]any, len(src))
		for k, v := range src {
			cp[k] = v
		}
		return cp
	}
	return dst
}

func (s *MemoryApprovalStore) ListPending(_ context.Context, tenantID string) ([]ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ApprovalRequest, 0)
	for _, item := range s.items {
		if item.TenantID == tenantID && item.Status == StatusPending {
			out = append(out, deepCopyApprovalRequest(item))
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
		out = append(out, deepCopyApprovalRequest(item))
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

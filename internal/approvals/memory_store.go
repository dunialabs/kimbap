package approvals

import (
	"context"
	"fmt"
	"reflect"
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
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = deepCopyAny(v)
	}
	return dst
}

func deepCopyAny(value any) any {
	if value == nil {
		return nil
	}
	return deepCopyReflectValue(reflect.ValueOf(value)).Interface()
}

func deepCopyReflectValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		inner := deepCopyReflectValue(value.Elem())
		copyValue := reflect.New(value.Type()).Elem()
		copyValue.Set(inner)
		return copyValue
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copyValue := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			copyValue.SetMapIndex(iter.Key(), deepCopyReflectValue(iter.Value()))
		}
		return copyValue
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copyValue := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			copyValue.Index(i).Set(deepCopyReflectValue(value.Index(i)))
		}
		return copyValue
	case reflect.Array:
		copyValue := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			copyValue.Index(i).Set(deepCopyReflectValue(value.Index(i)))
		}
		return copyValue
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copyValue := reflect.New(value.Type().Elem())
		copyValue.Elem().Set(deepCopyReflectValue(value.Elem()))
		return copyValue
	default:
		return value
	}
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
	now := time.Now().UTC()
	count := 0
	for id, item := range s.items {
		if item.Status == StatusPending && !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			item.Status = StatusExpired
			item.ResolvedAt = &now
			item.ResolvedBy = "system"
			s.items[id] = item
			count++
		}
	}
	return count, nil
}

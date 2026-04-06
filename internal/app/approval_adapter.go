package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/approvals"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
)

type ApprovalManagerAdapter struct {
	mgr *approvals.ApprovalManager
}

func NewApprovalManagerAdapter(mgr *approvals.ApprovalManager) runtimepkg.ApprovalManager {
	if mgr == nil {
		return nil
	}
	return &ApprovalManagerAdapter{mgr: mgr}
}

func (a *ApprovalManagerAdapter) CreateRequest(ctx context.Context, req runtimepkg.ApprovalRequest) (*runtimepkg.ApprovalResult, error) {
	if a == nil || a.mgr == nil {
		return nil, fmt.Errorf("approval manager unavailable")
	}
	approvalReq := &approvals.ApprovalRequest{
		TenantID:  req.TenantID,
		RequestID: req.RequestID,
		AgentName: req.Principal.AgentName,
		Risk:      req.Action.Risk.DocVocab(),
		Input:     req.Input,
	}
	actionName := strings.TrimSpace(req.Action.Name)
	if actionName != "" {
		if svc, action, ok := strings.Cut(actionName, "."); ok {
			approvalReq.Service = strings.TrimSpace(svc)
			actionName = strings.TrimSpace(action)
		}
	}
	if approvalReq.Service == "" {
		approvalReq.Service = strings.TrimSpace(req.Action.Namespace)
	}
	approvalReq.Action = actionName
	if err := a.mgr.Submit(ctx, approvalReq); err != nil {
		return nil, err
	}

	return &runtimepkg.ApprovalResult{
		Approved:  false,
		RequestID: approvalReq.ID,
		Reason:    "approval pending",
		Meta: map[string]any{
			"approval_id": approvalReq.ID,
			"status":      "pending",
		},
	}, nil
}

func (a *ApprovalManagerAdapter) CancelRequest(ctx context.Context, approvalRequestID string, reason string) error {
	if a == nil || a.mgr == nil {
		return fmt.Errorf("approval manager unavailable")
	}
	approvalRequestID = strings.TrimSpace(approvalRequestID)
	if approvalRequestID == "" {
		return fmt.Errorf("approval request id is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "cancelled by runtime"
	}
	return a.mgr.Deny(ctx, approvalRequestID, "system", reason)
}

type sqlHeldExecutionStore struct {
	st store.HeldExecutionStore
}

func NewSQLHeldExecutionStore(st store.HeldExecutionStore) runtimepkg.HeldExecutionStore {
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

func NewMemoryHeldExecutionStore() runtimepkg.HeldExecutionStore {
	return &memoryHeldExecutionStore{held: map[string]actions.ExecutionRequest{}}
}

func (s *memoryHeldExecutionStore) Hold(_ context.Context, approvalRequestID string, req actions.ExecutionRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("serialize held request: %w", err)
	}
	var cloned actions.ExecutionRequest
	if err := json.Unmarshal(b, &cloned); err != nil {
		return fmt.Errorf("deserialize held request: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held[approvalRequestID] = cloned
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

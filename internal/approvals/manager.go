package approvals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ApprovalStore interface {
	Create(ctx context.Context, req *ApprovalRequest) error
	Get(ctx context.Context, id string) (*ApprovalRequest, error)
	Update(ctx context.Context, req *ApprovalRequest) error
	ListPending(ctx context.Context, tenantID string) ([]ApprovalRequest, error)
	ListAll(ctx context.Context, tenantID string, filter ApprovalFilter) ([]ApprovalRequest, error)
	ExpireOld(ctx context.Context) (int, error)
}

type ApprovalFilter struct {
	Status    *ApprovalStatus
	AgentName string
	Service   string
	Limit     int
}

type ApprovalManager struct {
	store    ApprovalStore
	notifier Notifier
	ttl      time.Duration
	mu       sync.Mutex
}

func NewApprovalManager(store ApprovalStore, notifier Notifier, ttl time.Duration) *ApprovalManager {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &ApprovalManager{store: store, notifier: notifier, ttl: ttl}
}

func (m *ApprovalManager) Submit(ctx context.Context, req *ApprovalRequest) error {
	if req == nil {
		return errors.New("approval request is nil")
	}
	if req.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if req.ID == "" {
		req.ID = uuid.NewString()
	}

	now := time.Now().UTC()
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = req.CreatedAt.Add(m.ttl)
	}
	req.Status = StatusPending
	req.ResolvedAt = nil
	req.ResolvedBy = ""
	req.DenyReason = ""
	req.Votes = nil

	if err := m.store.Create(ctx, req); err != nil {
		return err
	}
	if m.notifier != nil {
		if err := m.notifier.Notify(ctx, req); err != nil {
			log.Warn().Err(err).Str("requestId", req.ID).Msg("approvals notifier failed")
		}
	}

	return nil
}

func (m *ApprovalManager) Approve(ctx context.Context, id string, approvedBy string) error {
	approvedBy = strings.TrimSpace(approvedBy)
	if approvedBy == "" {
		return errors.New("approved_by is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	req, err := m.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrNotFound
	}
	if err := validatePendingForResolution(req); err != nil {
		return err
	}

	for _, v := range req.Votes {
		if v.ApproverID == approvedBy {
			return fmt.Errorf("%w: %s", ErrDuplicateVote, approvedBy)
		}
	}

	now := time.Now().UTC()
	req.Votes = append(req.Votes, ApprovalVote{
		ApproverID: approvedBy,
		Decision:   StatusApproved,
		VotedAt:    now,
	})

	required := max(1, req.RequiredApprovals)
	approveCount := 0
	for _, v := range req.Votes {
		if v.Decision == StatusApproved {
			approveCount++
		}
	}
	if approveCount >= required {
		req.Status = StatusApproved
		req.ResolvedBy = approvedBy
		req.ResolvedAt = &now
		req.DenyReason = ""
	}

	return m.store.Update(ctx, req)
}

func (m *ApprovalManager) Deny(ctx context.Context, id string, deniedBy string, reason string) error {
	deniedBy = strings.TrimSpace(deniedBy)
	if deniedBy == "" {
		return errors.New("denied_by is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	req, err := m.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrNotFound
	}
	if err := validatePendingForResolution(req); err != nil {
		return err
	}

	for _, v := range req.Votes {
		if v.ApproverID == deniedBy {
			return fmt.Errorf("%w: %s", ErrDuplicateVote, deniedBy)
		}
	}

	now := time.Now().UTC()
	req.Votes = append(req.Votes, ApprovalVote{
		ApproverID: deniedBy,
		Decision:   StatusDenied,
		Reason:     reason,
		VotedAt:    now,
	})
	req.Status = StatusDenied
	req.ResolvedBy = deniedBy
	req.ResolvedAt = &now
	req.DenyReason = reason

	return m.store.Update(ctx, req)
}

func (m *ApprovalManager) Get(ctx context.Context, id string) (*ApprovalRequest, error) {
	return m.store.Get(ctx, id)
}

func (m *ApprovalManager) ListPending(ctx context.Context, tenantID string) ([]ApprovalRequest, error) {
	return m.store.ListPending(ctx, tenantID)
}

func (m *ApprovalManager) ExpireStale(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.ExpireOld(ctx)
}

func validatePendingForResolution(req *ApprovalRequest) error {
	if req.Status != StatusPending {
		return fmt.Errorf("%w: status is %s", ErrAlreadyResolved, req.Status)
	}
	if !req.ExpiresAt.IsZero() && time.Now().After(req.ExpiresAt) {
		return ErrExpired
	}
	return nil
}

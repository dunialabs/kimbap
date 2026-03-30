package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/storeconv"
)

type storeApprovalStoreAdapter struct {
	st *store.SQLStore
}

func (a *storeApprovalStoreAdapter) CanResolveApproval() bool {
	return a != nil && a.st != nil
}

func (a *storeApprovalStoreAdapter) Create(ctx context.Context, req *approvals.ApprovalRequest) error {
	if a.st == nil {
		return fmt.Errorf("store unavailable")
	}
	return a.st.CreateApproval(ctx, storeconv.ApprovalRecordForCreate(req))
}

func (a *storeApprovalStoreAdapter) Get(ctx context.Context, id string) (*approvals.ApprovalRequest, error) {
	rec, err := a.st.GetApproval(ctx, id)
	if err != nil {
		return nil, err
	}
	converted, err := storeconv.ApprovalRequestFromRecordStrict(*rec)
	if err != nil {
		return nil, err
	}
	result := &converted
	return result, nil
}

func (a *storeApprovalStoreAdapter) Update(ctx context.Context, req *approvals.ApprovalRequest) error {
	return a.st.UpdateApproval(ctx, storeconv.ApprovalRecordForUpdate(req))
}

func (a *storeApprovalStoreAdapter) Resolve(ctx context.Context, id string, actor string, decision approvals.ApprovalStatus, reason string) (*approvals.ApprovalRequest, error) {
	if a == nil || a.st == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	rec, err := a.st.ResolveApprovalVote(ctx, id, actor, string(decision), reason)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			return nil, approvals.ErrNotFound
		case errors.Is(err, store.ErrApprovalAlreadyResolved):
			return nil, approvals.ErrAlreadyResolved
		case errors.Is(err, store.ErrApprovalExpired):
			return nil, approvals.ErrExpired
		case errors.Is(err, store.ErrApprovalDuplicateVote):
			return nil, approvals.ErrDuplicateVote
		default:
			return nil, err
		}
	}
	converted, convErr := storeconv.ApprovalRequestFromRecordStrict(*rec)
	if convErr != nil {
		return nil, convErr
	}
	result := &converted
	return result, nil
}

func (a *storeApprovalStoreAdapter) ListPending(ctx context.Context, tenantID string) ([]approvals.ApprovalRequest, error) {
	recs, err := a.st.ListApprovals(ctx, tenantID, "pending")
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, len(recs))
	for i, r := range recs {
		converted, convErr := storeconv.ApprovalRequestFromRecordStrict(r)
		if convErr != nil {
			return nil, convErr
		}
		out[i] = converted
	}
	return out, nil
}

func (a *storeApprovalStoreAdapter) ListAll(ctx context.Context, tenantID string, filter approvals.ApprovalFilter) ([]approvals.ApprovalRequest, error) {
	status := ""
	if filter.Status != nil {
		status = string(*filter.Status)
	}
	recs, err := a.st.ListApprovals(ctx, tenantID, status)
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, 0, len(recs))
	for _, r := range recs {
		item, convErr := storeconv.ApprovalRequestFromRecordStrict(r)
		if convErr != nil {
			return nil, convErr
		}
		if filter.AgentName != "" && !strings.EqualFold(item.AgentName, filter.AgentName) {
			continue
		}
		if filter.Service != "" && !strings.EqualFold(item.Service, filter.Service) {
			continue
		}
		out = append(out, item)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (a *storeApprovalStoreAdapter) ExpireOld(ctx context.Context) (int, error) {
	return expirePendingApprovalsWithSideEffects(ctx, a.st, "", nil)
}

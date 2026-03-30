package api

import (
	"context"
	"errors"
	"strings"

	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/storeconv"
)

type approvalRecordUpdater interface {
	UpdateApproval(ctx context.Context, req *store.ApprovalRecord) error
}

type approvalVoteResolver interface {
	ResolveApprovalVote(ctx context.Context, id string, actor string, decision string, reason string) (*store.ApprovalRecord, error)
}

type approvalManagerStoreAdapter struct {
	base     store.Store
	updater  approvalRecordUpdater
	resolver approvalVoteResolver
}

func (a *approvalManagerStoreAdapter) CanResolveApproval() bool {
	return a != nil && a.resolver != nil
}

func newApprovalManager(st store.Store) (*approvals.ApprovalManager, bool) {
	updater, ok := st.(approvalRecordUpdater)
	if !ok {
		return nil, false
	}
	adapter := &approvalManagerStoreAdapter{base: st, updater: updater}
	if resolver, resolverOK := st.(approvalVoteResolver); resolverOK {
		adapter.resolver = resolver
	}
	return approvals.NewApprovalManager(adapter, nil, 0), true
}

func (a *approvalManagerStoreAdapter) Create(ctx context.Context, req *approvals.ApprovalRequest) error {
	if a == nil || a.base == nil {
		return errors.New("approval store unavailable")
	}
	return a.base.CreateApproval(ctx, storeconv.ApprovalRecordForCreate(req))
}

func (a *approvalManagerStoreAdapter) Get(ctx context.Context, id string) (*approvals.ApprovalRequest, error) {
	rec, err := a.base.GetApproval(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, approvals.ErrNotFound
		}
		return nil, err
	}
	if rec == nil {
		return nil, approvals.ErrNotFound
	}
	converted, err := storeconv.ApprovalRequestFromRecordStrict(*rec)
	if err != nil {
		return nil, err
	}
	out := &converted
	return out, nil
}

func (a *approvalManagerStoreAdapter) Update(ctx context.Context, req *approvals.ApprovalRequest) error {
	if a == nil || a.updater == nil {
		return errors.New("approval updater unavailable")
	}
	err := a.updater.UpdateApproval(ctx, storeconv.ApprovalRecordForUpdate(req))
	if errors.Is(err, store.ErrNotFound) {
		return approvals.ErrNotFound
	}
	return err
}

func (a *approvalManagerStoreAdapter) Resolve(ctx context.Context, id string, actor string, decision approvals.ApprovalStatus, reason string) (*approvals.ApprovalRequest, error) {
	if a == nil || a.resolver == nil {
		return nil, errors.New("approval resolver unavailable")
	}
	rec, err := a.resolver.ResolveApprovalVote(ctx, id, actor, string(decision), reason)
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

func (a *approvalManagerStoreAdapter) ListPending(ctx context.Context, tenantID string) ([]approvals.ApprovalRequest, error) {
	recs, err := a.base.ListApprovals(ctx, tenantID, string(approvals.StatusPending))
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, len(recs))
	for i := range recs {
		converted, convErr := storeconv.ApprovalRequestFromRecordStrict(recs[i])
		if convErr != nil {
			return nil, convErr
		}
		out[i] = converted
	}
	return out, nil
}

func (a *approvalManagerStoreAdapter) ListAll(ctx context.Context, tenantID string, filter approvals.ApprovalFilter) ([]approvals.ApprovalRequest, error) {
	status := ""
	if filter.Status != nil {
		status = string(*filter.Status)
	}
	recs, err := a.base.ListApprovals(ctx, tenantID, status)
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, 0, len(recs))
	for i := range recs {
		item, convErr := storeconv.ApprovalRequestFromRecordStrict(recs[i])
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

func (a *approvalManagerStoreAdapter) ExpireOld(ctx context.Context) (int, error) {
	return a.base.ExpirePendingApprovals(ctx)
}

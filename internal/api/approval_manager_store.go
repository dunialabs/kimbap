package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/store"
)

type approvalRecordUpdater interface {
	UpdateApproval(ctx context.Context, req *store.ApprovalRecord) error
}

type approvalManagerStoreAdapter struct {
	base    store.Store
	updater approvalRecordUpdater
}

func newApprovalManager(st store.Store) (*approvals.ApprovalManager, bool) {
	updater, ok := st.(approvalRecordUpdater)
	if !ok {
		return nil, false
	}
	adapter := &approvalManagerStoreAdapter{base: st, updater: updater}
	return approvals.NewApprovalManager(adapter, nil, 0), true
}

func (a *approvalManagerStoreAdapter) Create(ctx context.Context, req *approvals.ApprovalRequest) error {
	if a == nil || a.base == nil {
		return errors.New("approval store unavailable")
	}
	inputJSON := "{}"
	if req.Input != nil {
		if b, err := json.Marshal(req.Input); err == nil {
			inputJSON = string(b)
		}
	}
	return a.base.CreateApproval(ctx, &store.ApprovalRecord{
		ID:                req.ID,
		TenantID:          req.TenantID,
		RequestID:         req.RequestID,
		AgentName:         req.AgentName,
		Service:           req.Service,
		Action:            req.Action,
		Status:            string(req.Status),
		InputJSON:         inputJSON,
		RequiredApprovals: max(1, req.RequiredApprovals),
		VotesJSON:         approvalVotesToJSON(req.Votes),
		CreatedAt:         req.CreatedAt,
		ExpiresAt:         req.ExpiresAt,
		ResolvedAt:        req.ResolvedAt,
		ResolvedBy:        req.ResolvedBy,
		Reason:            req.DenyReason,
	})
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
	out := &approvals.ApprovalRequest{
		ID:                rec.ID,
		TenantID:          rec.TenantID,
		RequestID:         rec.RequestID,
		AgentName:         rec.AgentName,
		Service:           rec.Service,
		Action:            rec.Action,
		Input:             parseApprovalInputJSON(rec.InputJSON),
		Status:            approvals.ApprovalStatus(rec.Status),
		CreatedAt:         rec.CreatedAt,
		ExpiresAt:         rec.ExpiresAt,
		ResolvedAt:        rec.ResolvedAt,
		ResolvedBy:        rec.ResolvedBy,
		DenyReason:        rec.Reason,
		RequiredApprovals: max(1, rec.RequiredApprovals),
		Votes:             parseApprovalVotesJSON(rec.VotesJSON),
	}
	return out, nil
}

func (a *approvalManagerStoreAdapter) Update(ctx context.Context, req *approvals.ApprovalRequest) error {
	if a == nil || a.updater == nil {
		return errors.New("approval updater unavailable")
	}
	err := a.updater.UpdateApproval(ctx, &store.ApprovalRecord{
		ID:                req.ID,
		Status:            string(req.Status),
		ResolvedAt:        req.ResolvedAt,
		ResolvedBy:        req.ResolvedBy,
		Reason:            req.DenyReason,
		RequiredApprovals: max(1, req.RequiredApprovals),
		VotesJSON:         approvalVotesToJSON(req.Votes),
	})
	if errors.Is(err, store.ErrNotFound) {
		return approvals.ErrNotFound
	}
	return err
}

func (a *approvalManagerStoreAdapter) ListPending(ctx context.Context, tenantID string) ([]approvals.ApprovalRequest, error) {
	recs, err := a.base.ListApprovals(ctx, tenantID, string(approvals.StatusPending))
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, len(recs))
	for i := range recs {
		out[i] = approvalFromRecord(recs[i])
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
		item := approvalFromRecord(recs[i])
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

func parseApprovalInputJSON(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return nil
	}
	return input
}

func parseApprovalVotesJSON(raw string) []approvals.ApprovalVote {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var votes []approvals.ApprovalVote
	if err := json.Unmarshal([]byte(raw), &votes); err != nil {
		return nil
	}
	return votes
}

func approvalVotesToJSON(votes []approvals.ApprovalVote) string {
	if len(votes) == 0 {
		return "[]"
	}
	b, err := json.Marshal(votes)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func approvalFromRecord(rec store.ApprovalRecord) approvals.ApprovalRequest {
	return approvals.ApprovalRequest{
		ID:                rec.ID,
		TenantID:          rec.TenantID,
		RequestID:         rec.RequestID,
		AgentName:         rec.AgentName,
		Service:           rec.Service,
		Action:            rec.Action,
		Input:             parseApprovalInputJSON(rec.InputJSON),
		Status:            approvals.ApprovalStatus(rec.Status),
		CreatedAt:         rec.CreatedAt,
		ExpiresAt:         rec.ExpiresAt,
		ResolvedAt:        rec.ResolvedAt,
		ResolvedBy:        rec.ResolvedBy,
		DenyReason:        rec.Reason,
		RequiredApprovals: max(1, rec.RequiredApprovals),
		Votes:             parseApprovalVotesJSON(rec.VotesJSON),
	}
}

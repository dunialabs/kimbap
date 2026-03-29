package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/store"
)

type storeApprovalStoreAdapter struct {
	st *store.SQLStore
}

func (a *storeApprovalStoreAdapter) Create(ctx context.Context, req *approvals.ApprovalRequest) error {
	if a.st == nil {
		return fmt.Errorf("store unavailable")
	}
	inputJSON := "{}"
	if req.Input != nil {
		if b, err := json.Marshal(req.Input); err == nil {
			inputJSON = string(b)
		}
	}
	return a.st.CreateApproval(ctx, &store.ApprovalRecord{
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
	})
}

func (a *storeApprovalStoreAdapter) Get(ctx context.Context, id string) (*approvals.ApprovalRequest, error) {
	rec, err := a.st.GetApproval(ctx, id)
	if err != nil {
		return nil, err
	}
	result := &approvals.ApprovalRequest{
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
		ResolvedBy:        rec.ResolvedBy,
		DenyReason:        rec.Reason,
		RequiredApprovals: max(1, rec.RequiredApprovals),
		Votes:             parseApprovalVotesJSON(rec.VotesJSON),
	}
	if rec.ResolvedAt != nil {
		result.ResolvedAt = rec.ResolvedAt
	}
	return result, nil
}

func (a *storeApprovalStoreAdapter) Update(ctx context.Context, req *approvals.ApprovalRequest) error {
	return a.st.UpdateApproval(ctx, &store.ApprovalRecord{
		ID:                req.ID,
		Status:            string(req.Status),
		ResolvedAt:        req.ResolvedAt,
		ResolvedBy:        req.ResolvedBy,
		Reason:            req.DenyReason,
		RequiredApprovals: max(1, req.RequiredApprovals),
		VotesJSON:         approvalVotesToJSON(req.Votes),
	})
}

func (a *storeApprovalStoreAdapter) ListPending(ctx context.Context, tenantID string) ([]approvals.ApprovalRequest, error) {
	recs, err := a.st.ListApprovals(ctx, tenantID, "pending")
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, len(recs))
	for i, r := range recs {
		out[i] = approvals.ApprovalRequest{
			ID:                r.ID,
			TenantID:          r.TenantID,
			RequestID:         r.RequestID,
			AgentName:         r.AgentName,
			Service:           r.Service,
			Action:            r.Action,
			Input:             parseApprovalInputJSON(r.InputJSON),
			Status:            approvals.ApprovalStatus(r.Status),
			CreatedAt:         r.CreatedAt,
			ExpiresAt:         r.ExpiresAt,
			ResolvedAt:        r.ResolvedAt,
			ResolvedBy:        r.ResolvedBy,
			DenyReason:        r.Reason,
			RequiredApprovals: max(1, r.RequiredApprovals),
			Votes:             parseApprovalVotesJSON(r.VotesJSON),
		}
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
		item := approvals.ApprovalRequest{
			ID:                r.ID,
			TenantID:          r.TenantID,
			RequestID:         r.RequestID,
			AgentName:         r.AgentName,
			Service:           r.Service,
			Action:            r.Action,
			Input:             parseApprovalInputJSON(r.InputJSON),
			Status:            approvals.ApprovalStatus(r.Status),
			CreatedAt:         r.CreatedAt,
			ExpiresAt:         r.ExpiresAt,
			ResolvedAt:        r.ResolvedAt,
			ResolvedBy:        r.ResolvedBy,
			DenyReason:        r.Reason,
			RequiredApprovals: max(1, r.RequiredApprovals),
			Votes:             parseApprovalVotesJSON(r.VotesJSON),
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

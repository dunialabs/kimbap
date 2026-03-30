package storeconv

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/auth"
	"github.com/dunialabs/kimbap/internal/store"
)

func MarshalScopes(scopes []string) string {
	if len(scopes) == 0 {
		return "[]"
	}
	b, err := json.Marshal(scopes)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func ParseScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var scopes []string
	if err := json.Unmarshal([]byte(raw), &scopes); err != nil {
		return nil
	}
	return scopes
}

func TokenRecordFromServiceToken(token *auth.ServiceToken) *store.TokenRecord {
	if token == nil {
		return nil
	}
	return &store.TokenRecord{
		ID:          token.ID,
		TenantID:    token.TenantID,
		AgentName:   token.AgentName,
		TokenHash:   token.TokenHash,
		DisplayHint: token.DisplayHint,
		Scopes:      MarshalScopes(token.Scopes),
		CreatedAt:   token.CreatedAt,
		ExpiresAt:   token.ExpiresAt,
		LastUsedAt:  token.LastUsedAt,
		RevokedAt:   token.RevokedAt,
		CreatedBy:   token.CreatedBy,
	}
}

func ServiceTokenFromRecord(rec store.TokenRecord) auth.ServiceToken {
	return auth.ServiceToken{
		ID:          rec.ID,
		TenantID:    rec.TenantID,
		AgentName:   rec.AgentName,
		TokenHash:   rec.TokenHash,
		DisplayHint: rec.DisplayHint,
		Scopes:      ParseScopes(rec.Scopes),
		CreatedAt:   rec.CreatedAt,
		ExpiresAt:   rec.ExpiresAt,
		LastUsedAt:  rec.LastUsedAt,
		RevokedAt:   rec.RevokedAt,
		CreatedBy:   rec.CreatedBy,
	}
}

func PrincipalFromTokenRecord(token *store.TokenRecord) (*auth.Principal, error) {
	if token == nil {
		return nil, auth.ErrInvalidToken
	}
	scopes := ParseScopes(token.Scopes)
	if token.Scopes != "" && scopes == nil {
		return nil, auth.ErrInvalidToken
	}
	return &auth.Principal{
		ID:        token.ID,
		Type:      auth.PrincipalTypeService,
		TenantID:  token.TenantID,
		AgentName: token.AgentName,
		Scopes:    scopes,
		TokenID:   token.ID,
		IssuedAt:  token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
	}, nil
}

func ApprovalInputToJSON(input map[string]any) string {
	if input == nil {
		return "{}"
	}
	b, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func ParseApprovalInputJSON(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return nil
	}
	return input
}

func ParseApprovalVotesJSON(raw string) []approvals.ApprovalVote {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var votes []approvals.ApprovalVote
	if err := json.Unmarshal([]byte(raw), &votes); err != nil {
		return nil
	}
	return votes
}

func ParseApprovalInputJSONStrict(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return nil, err
	}
	return input, nil
}

func ParseApprovalVotesJSONStrict(raw string) ([]approvals.ApprovalVote, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var votes []approvals.ApprovalVote
	if err := json.Unmarshal([]byte(raw), &votes); err != nil {
		return nil, err
	}
	return votes, nil
}

func ApprovalVotesToJSON(votes []approvals.ApprovalVote) string {
	if len(votes) == 0 {
		return "[]"
	}
	b, err := json.Marshal(votes)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func ApprovalRecordForCreate(req *approvals.ApprovalRequest) *store.ApprovalRecord {
	if req == nil {
		return nil
	}
	return &store.ApprovalRecord{
		ID:                req.ID,
		TenantID:          req.TenantID,
		RequestID:         req.RequestID,
		AgentName:         req.AgentName,
		Service:           req.Service,
		Action:            req.Action,
		Status:            string(req.Status),
		InputJSON:         ApprovalInputToJSON(req.Input),
		RequiredApprovals: max(1, req.RequiredApprovals),
		VotesJSON:         ApprovalVotesToJSON(req.Votes),
		CreatedAt:         req.CreatedAt,
		ExpiresAt:         req.ExpiresAt,
		ResolvedAt:        req.ResolvedAt,
		ResolvedBy:        req.ResolvedBy,
		Reason:            req.DenyReason,
	}
}

func ApprovalRecordForUpdate(req *approvals.ApprovalRequest) *store.ApprovalRecord {
	if req == nil {
		return nil
	}
	return &store.ApprovalRecord{
		ID:                req.ID,
		Status:            string(req.Status),
		ResolvedAt:        req.ResolvedAt,
		ResolvedBy:        req.ResolvedBy,
		Reason:            req.DenyReason,
		RequiredApprovals: max(1, req.RequiredApprovals),
		VotesJSON:         ApprovalVotesToJSON(req.Votes),
	}
}

func ApprovalRequestFromRecord(rec store.ApprovalRecord) approvals.ApprovalRequest {
	return approvals.ApprovalRequest{
		ID:                rec.ID,
		TenantID:          rec.TenantID,
		RequestID:         rec.RequestID,
		AgentName:         rec.AgentName,
		Service:           rec.Service,
		Action:            rec.Action,
		Input:             ParseApprovalInputJSON(rec.InputJSON),
		Status:            approvals.ApprovalStatus(rec.Status),
		CreatedAt:         rec.CreatedAt,
		ExpiresAt:         rec.ExpiresAt,
		ResolvedAt:        rec.ResolvedAt,
		ResolvedBy:        rec.ResolvedBy,
		DenyReason:        rec.Reason,
		RequiredApprovals: max(1, rec.RequiredApprovals),
		Votes:             ParseApprovalVotesJSON(rec.VotesJSON),
	}
}

func ApprovalRequestFromRecordStrict(rec store.ApprovalRecord) (approvals.ApprovalRequest, error) {
	input, err := ParseApprovalInputJSONStrict(rec.InputJSON)
	if err != nil {
		return approvals.ApprovalRequest{}, fmt.Errorf("parse approval input for %q: %w", rec.ID, err)
	}
	votes, err := ParseApprovalVotesJSONStrict(rec.VotesJSON)
	if err != nil {
		return approvals.ApprovalRequest{}, fmt.Errorf("parse approval votes for %q: %w", rec.ID, err)
	}
	return approvals.ApprovalRequest{
		ID:                rec.ID,
		TenantID:          rec.TenantID,
		RequestID:         rec.RequestID,
		AgentName:         rec.AgentName,
		Service:           rec.Service,
		Action:            rec.Action,
		Input:             input,
		Status:            approvals.ApprovalStatus(rec.Status),
		CreatedAt:         rec.CreatedAt,
		ExpiresAt:         rec.ExpiresAt,
		ResolvedAt:        rec.ResolvedAt,
		ResolvedBy:        rec.ResolvedBy,
		DenyReason:        rec.Reason,
		RequiredApprovals: max(1, rec.RequiredApprovals),
		Votes:             votes,
	}, nil
}

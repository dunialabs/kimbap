package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrApprovalAlreadyResolved = errors.New("approval already resolved")
	ErrApprovalExpired         = errors.New("approval has expired")
	ErrApprovalDuplicateVote   = errors.New("approver has already voted")
)

type approvalVoteRecord struct {
	ApproverID string
	Decision   string
	Reason     string
	VotedAt    time.Time
}

func parseApprovalVotesJSONStrict(raw string) ([]approvalVoteRecord, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	var votes []approvalVoteRecord
	if err := json.Unmarshal([]byte(trimmed), &votes); err != nil {
		return nil, err
	}
	return votes, nil
}

func approvalVotesToJSON(votes []approvalVoteRecord) (string, error) {
	if len(votes) == 0 {
		return "[]", nil
	}
	encoded, err := json.Marshal(votes)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func countApprovedVotes(votes []approvalVoteRecord) int {
	count := 0
	for _, vote := range votes {
		if vote.Decision == "approved" {
			count++
		}
	}
	return count
}

func (s *SQLStore) CreateApproval(ctx context.Context, req *ApprovalRecord) error {
	if req == nil {
		return errors.New("approval is required")
	}
	req.TenantID = strings.TrimSpace(req.TenantID)
	if req.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(req.ID) == "" {
		req.ID = uuid.NewString()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = req.CreatedAt.Add(10 * time.Minute)
	}
	if req.Status == "" {
		req.Status = "pending"
	}
	if req.RequiredApprovals <= 0 {
		req.RequiredApprovals = 1
	}
	if strings.TrimSpace(req.VotesJSON) == "" {
		req.VotesJSON = "[]"
	}
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO approvals (
			id, tenant_id, request_id, agent_name, service, action,
			status, input_json, required_approvals, votes_json, created_at, expires_at, resolved_at,
			resolved_by, reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`),
		req.ID,
		req.TenantID,
		req.RequestID,
		req.AgentName,
		req.Service,
		req.Action,
		req.Status,
		req.InputJSON,
		req.RequiredApprovals,
		req.VotesJSON,
		req.CreatedAt,
		req.ExpiresAt,
		req.ResolvedAt,
		req.ResolvedBy,
		req.Reason,
	)
	return err
}

func (s *SQLStore) GetApproval(ctx context.Context, id string) (*ApprovalRecord, error) {
	id = strings.TrimSpace(id)
	row := s.db.QueryRowContext(ctx, s.bind(`
		SELECT `+approvalSelectColumns+`
		FROM approvals WHERE id = ?
	`), id)
	rec, err := scanApproval(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rec, nil
}

func (s *SQLStore) UpdateApprovalStatus(ctx context.Context, id string, status string, resolvedBy string, reason string) error {
	id = strings.TrimSpace(id)
	status = strings.TrimSpace(status)
	resolvedBy = strings.TrimSpace(resolvedBy)
	reason = strings.TrimSpace(reason)

	now := time.Now().UTC()
	// CAS: only update if still pending AND not expired
	res, err := s.db.ExecContext(ctx, s.bind(`
		UPDATE approvals
		SET status = ?, resolved_at = ?, resolved_by = ?, reason = ?
		WHERE id = ? AND status = 'pending' AND expires_at > ?
	`), status, now, resolvedBy, reason, id, now)
	if err != nil {
		return err
	}
	if affectedRows(res) == 0 {
		existing, lookupErr := s.GetApproval(ctx, id)
		if lookupErr != nil {
			return lookupErr
		}
		if existing.Status != "pending" {
			if existing.Status == "expired" {
				return ErrApprovalExpired
			}
			return ErrApprovalAlreadyResolved
		}
		if !existing.ExpiresAt.After(now) {
			return ErrApprovalExpired
		}
		return ErrNotFound
	}
	return nil
}

func (s *SQLStore) UpdateApproval(ctx context.Context, req *ApprovalRecord) error {
	if req == nil {
		return errors.New("approval is required")
	}
	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		return ErrNotFound
	}
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		req.Status = "pending"
	}
	if req.RequiredApprovals <= 0 {
		req.RequiredApprovals = 1
	}
	if strings.TrimSpace(req.VotesJSON) == "" {
		req.VotesJSON = "[]"
	}

	res, err := s.db.ExecContext(ctx, s.bind(`
		UPDATE approvals
		SET status = ?, resolved_at = ?, resolved_by = ?, reason = ?, required_approvals = ?, votes_json = ?
		WHERE id = ?
	`), req.Status, req.ResolvedAt, req.ResolvedBy, req.Reason, req.RequiredApprovals, req.VotesJSON, req.ID)
	if err != nil {
		return err
	}
	if affectedRows(res) == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLStore) ResolveApprovalVote(ctx context.Context, id string, actor string, decision string, reason string) (*ApprovalRecord, error) {
	id = strings.TrimSpace(id)
	actor = strings.TrimSpace(actor)
	decision = strings.TrimSpace(strings.ToLower(decision))
	reason = strings.TrimSpace(reason)

	if id == "" {
		return nil, ErrNotFound
	}
	if actor == "" {
		return nil, errors.New("resolved_by is required")
	}
	switch decision {
	case "approved", "denied":
	default:
		return nil, fmt.Errorf("invalid approval decision %q", decision)
	}

	for attempt := 0; attempt < 8; attempt++ {
		now := time.Now().UTC()
		rec, err := s.GetApproval(ctx, id)
		if err != nil {
			return nil, err
		}
		if rec == nil {
			return nil, ErrNotFound
		}
		if rec.Status != "pending" {
			if rec.Status == "expired" {
				return nil, ErrApprovalExpired
			}
			return nil, ErrApprovalAlreadyResolved
		}
		if !rec.ExpiresAt.After(now) {
			return nil, ErrApprovalExpired
		}

		votes, err := parseApprovalVotesJSONStrict(rec.VotesJSON)
		if err != nil {
			return nil, fmt.Errorf("parse approval votes for %q: %w", rec.ID, err)
		}
		for _, vote := range votes {
			if strings.TrimSpace(vote.ApproverID) == actor {
				return nil, ErrApprovalDuplicateVote
			}
		}

		vote := approvalVoteRecord{
			ApproverID: actor,
			Decision:   decision,
			Reason:     "",
			VotedAt:    now,
		}
		if decision == "denied" {
			vote.Reason = reason
		}
		votes = append(votes, vote)

		requiredApprovals := rec.RequiredApprovals
		if requiredApprovals <= 0 {
			requiredApprovals = 1
		}

		status := "pending"
		resolvedBy := ""
		var resolvedAt *time.Time
		resolvedReason := ""

		if decision == "denied" {
			status = "denied"
			resolvedBy = actor
			resolvedReason = reason
			resolvedNow := now
			resolvedAt = &resolvedNow
		} else if countApprovedVotes(votes) >= requiredApprovals {
			status = "approved"
			resolvedBy = actor
			resolvedNow := now
			resolvedAt = &resolvedNow
		}

		votesJSON, err := approvalVotesToJSON(votes)
		if err != nil {
			return nil, fmt.Errorf("encode approval votes for %q: %w", rec.ID, err)
		}

		res, err := s.db.ExecContext(ctx, s.bind(`
			UPDATE approvals
			SET status = ?, resolved_at = ?, resolved_by = ?, reason = ?, required_approvals = ?, votes_json = ?
			WHERE id = ? AND status = 'pending' AND expires_at > ? AND votes_json = ?
		`), status, resolvedAt, resolvedBy, resolvedReason, requiredApprovals, votesJSON, id, now, rec.VotesJSON)
		if err != nil {
			return nil, err
		}
		if affectedRows(res) == 0 {
			continue
		}

		updated := *rec
		updated.Status = status
		updated.ResolvedBy = resolvedBy
		updated.ResolvedAt = resolvedAt
		updated.Reason = resolvedReason
		updated.VotesJSON = votesJSON
		updated.RequiredApprovals = requiredApprovals
		return &updated, nil
	}

	return nil, fmt.Errorf("resolve approval vote: concurrent update conflict")
}

func (s *SQLStore) ListApprovals(ctx context.Context, tenantID string, status string) ([]ApprovalRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, ErrInvalidTenantID
	}
	query := `
		SELECT ` + approvalSelectColumns + `
		FROM approvals
		WHERE tenant_id = ?`
	args := []any{tenantID}
	status = strings.TrimSpace(status)
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ApprovalRecord, 0)
	for rows.Next() {
		rec, scanErr := scanApproval(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

func (s *SQLStore) ExpirePendingApprovals(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, s.bind(`
		UPDATE approvals SET status = 'expired', resolved_at = ?, resolved_by = 'system', reason = 'auto-expired'
		WHERE status = 'pending' AND expires_at <= ?
	`), now, now)
	if err != nil {
		return 0, err
	}
	return int(affectedRows(res)), nil
}

func (s *SQLStore) ListExpiredPendingApprovals(ctx context.Context, tenantID string) ([]ApprovalRecord, error) {
	now := time.Now().UTC()
	query := `
		SELECT ` + approvalSelectColumns + `
		FROM approvals
		WHERE status = 'pending' AND expires_at <= ?`
	args := []any{now}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID != "" {
		query += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}
	query += ` ORDER BY expires_at ASC, created_at ASC`
	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ApprovalRecord, 0)
	for rows.Next() {
		rec, scanErr := scanApproval(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

func (s *SQLStore) ExpireApproval(ctx context.Context, id string) (bool, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, s.bind(`
		UPDATE approvals
		SET status = 'expired', resolved_at = ?, resolved_by = 'system', reason = 'auto-expired'
		WHERE id = ? AND status = 'pending' AND expires_at <= ?
	`), now, id, now)
	if err != nil {
		return false, err
	}
	return affectedRows(res) > 0, nil
}

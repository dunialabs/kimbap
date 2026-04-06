package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (s *SQLStore) HoldExecution(ctx context.Context, approvalRequestID string, requestJSON []byte) error {
	approvalRequestID = strings.TrimSpace(approvalRequestID)
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO held_executions (approval_request_id, request_json, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(approval_request_id) DO UPDATE SET request_json = excluded.request_json
	`), approvalRequestID, string(requestJSON), time.Now().UTC())
	return err
}

func (s *SQLStore) ResumeExecution(ctx context.Context, approvalRequestID string) ([]byte, error) {
	approvalRequestID = strings.TrimSpace(approvalRequestID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var requestJSON string
	err = tx.QueryRowContext(ctx, s.bind(`
		SELECT request_json FROM held_executions WHERE approval_request_id = ?
	`), approvalRequestID).Scan(&requestJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	res, err := tx.ExecContext(ctx, s.bind(`DELETE FROM held_executions WHERE approval_request_id = ?`), approvalRequestID)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return []byte(requestJSON), nil
}

func (s *SQLStore) RemoveExecution(ctx context.Context, approvalRequestID string) error {
	approvalRequestID = strings.TrimSpace(approvalRequestID)
	_, err := s.db.ExecContext(ctx, s.bind(`DELETE FROM held_executions WHERE approval_request_id = ?`), approvalRequestID)
	return err
}

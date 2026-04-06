package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *SQLStore) CreateToken(ctx context.Context, token *TokenRecord) error {
	if token == nil {
		return errors.New("token is required")
	}
	if strings.TrimSpace(token.ID) == "" {
		token.ID = "st_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	token.TenantID = strings.TrimSpace(token.TenantID)
	if token.TenantID == "" {
		return ErrInvalidTenantID
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	if token.ExpiresAt.IsZero() {
		token.ExpiresAt = token.CreatedAt.Add(30 * 24 * time.Hour)
	}
	if token.Scopes == "" {
		token.Scopes = "[]"
	}

	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO service_tokens (
			id, tenant_id, agent_name, token_hash, display_hint, scopes,
			created_at, expires_at, last_used_at, revoked_at, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`),
		token.ID,
		token.TenantID,
		token.AgentName,
		token.TokenHash,
		token.DisplayHint,
		token.Scopes,
		token.CreatedAt,
		token.ExpiresAt,
		token.LastUsedAt,
		token.RevokedAt,
		token.CreatedBy,
	)
	return err
}

func (s *SQLStore) GetToken(ctx context.Context, id string) (*TokenRecord, error) {
	id = strings.TrimSpace(id)
	row := s.db.QueryRowContext(ctx, s.bind(`
		SELECT `+tokenSelectColumns+`
		FROM service_tokens WHERE id = ?
	`), id)
	rec, err := scanToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rec, nil
}

func (s *SQLStore) GetTokenByHash(ctx context.Context, hash string) (*TokenRecord, error) {
	hash = strings.TrimSpace(hash)
	row := s.db.QueryRowContext(ctx, s.bind(`
		SELECT `+tokenSelectColumns+`
		FROM service_tokens WHERE token_hash = ?
	`), hash)
	rec, err := scanToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rec, nil
}

func (s *SQLStore) ListTokens(ctx context.Context, tenantID string) ([]TokenRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, ErrInvalidTenantID
	}
	rows, err := s.db.QueryContext(ctx, s.bind(`
		SELECT `+tokenSelectColumns+`
		FROM service_tokens
		WHERE tenant_id = ?
		ORDER BY created_at DESC
	`), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TokenRecord, 0)
	for rows.Next() {
		rec, scanErr := scanToken(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

func (s *SQLStore) UpdateTokenLastUsed(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	res, err := s.db.ExecContext(ctx, s.bind(`UPDATE service_tokens SET last_used_at = ? WHERE id = ? AND revoked_at IS NULL`), time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if affectedRows(res) == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLStore) RevokeToken(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	res, err := s.db.ExecContext(ctx, s.bind(`UPDATE service_tokens SET revoked_at = COALESCE(revoked_at, ?) WHERE id = ?`), time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if affectedRows(res) == 0 {
		return ErrNotFound
	}
	return nil
}

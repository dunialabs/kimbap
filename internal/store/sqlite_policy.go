package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (s *SQLStore) SetPolicy(ctx context.Context, tenantID string, document []byte) error {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return ErrInvalidTenantID
	}
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO policies (tenant_id, document, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT (tenant_id)
		DO UPDATE SET document = EXCLUDED.document, updated_at = EXCLUDED.updated_at
	`), tenantID, document, now)
	return err
}

func (s *SQLStore) GetPolicy(ctx context.Context, tenantID string) ([]byte, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, ErrInvalidTenantID
	}
	var doc []byte
	err := s.db.QueryRowContext(ctx, s.bind(`SELECT document FROM policies WHERE tenant_id = ?`), tenantID).Scan(&doc)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return doc, nil
}

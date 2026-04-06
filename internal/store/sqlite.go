package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/sqliteutil"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const maxAuditQueryLimit = 10000

const tokenSelectColumns = "id, tenant_id, agent_name, token_hash, display_hint, scopes, created_at, expires_at, last_used_at, revoked_at, created_by"

const approvalSelectColumns = "id, tenant_id, request_id, agent_name, service, action, status, input_json, required_approvals, votes_json, created_at, expires_at, resolved_at, resolved_by, reason"

var (
	ErrNotFound        = errors.New("record not found")
	ErrInvalidTenantID = errors.New("tenant id is required")
)

type SQLStore struct {
	db      *sql.DB
	dialect string
}

func NewSQLiteStore(db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	return &SQLStore{db: db, dialect: "sqlite"}, nil
}

func OpenSQLiteStore(dsn string) (*SQLStore, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("dsn is required")
	}
	db, err := openDBWithPing("sqlite", dsn, "sqlite")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := sqliteutil.ApplyConnectionPragmas(context.Background(), db, []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	st, err := NewSQLiteStore(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return st, nil
}

func NewPostgresStore(db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	return &SQLStore{db: db, dialect: "postgres"}, nil
}

func OpenPostgresStore(dsn string) (*SQLStore, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("dsn is required")
	}
	db, err := openDBWithPing("pgx", dsn, "postgres")
	if err != nil {
		return nil, err
	}
	st, err := NewPostgresStore(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return st, nil
}

func openDBWithPing(driverName, dsn, label string) (*sql.DB, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s database: %w", label, err)
	}
	return db, nil
}

func (s *SQLStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLStore) bind(query string) string {
	if s == nil || s.dialect != "postgres" {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	idx := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			_, _ = fmt.Fprintf(&b, "$%d", idx)
			idx++
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func scanToken(scanner interface{ Scan(dest ...any) error }) (*TokenRecord, error) {
	var rec TokenRecord
	var lastUsed sql.NullTime
	var revoked sql.NullTime
	err := scanner.Scan(
		&rec.ID,
		&rec.TenantID,
		&rec.AgentName,
		&rec.TokenHash,
		&rec.DisplayHint,
		&rec.Scopes,
		&rec.CreatedAt,
		&rec.ExpiresAt,
		&lastUsed,
		&revoked,
		&rec.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		ts := lastUsed.Time
		rec.LastUsedAt = &ts
	}
	if revoked.Valid {
		ts := revoked.Time
		rec.RevokedAt = &ts
	}
	return &rec, nil
}

func scanAudit(scanner interface{ Scan(dest ...any) error }) (*AuditRecord, error) {
	var rec AuditRecord
	err := scanner.Scan(
		&rec.ID,
		&rec.Timestamp,
		&rec.RequestID,
		&rec.TraceID,
		&rec.TenantID,
		&rec.PrincipalID,
		&rec.AgentName,
		&rec.Service,
		&rec.Action,
		&rec.Mode,
		&rec.Status,
		&rec.PolicyDecision,
		&rec.DurationMS,
		&rec.ErrorCode,
		&rec.ErrorMessage,
		&rec.InputJSON,
		&rec.MetaJSON,
	)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func scanApproval(scanner interface{ Scan(dest ...any) error }) (*ApprovalRecord, error) {
	var rec ApprovalRecord
	var resolved sql.NullTime
	err := scanner.Scan(
		&rec.ID,
		&rec.TenantID,
		&rec.RequestID,
		&rec.AgentName,
		&rec.Service,
		&rec.Action,
		&rec.Status,
		&rec.InputJSON,
		&rec.RequiredApprovals,
		&rec.VotesJSON,
		&rec.CreatedAt,
		&rec.ExpiresAt,
		&resolved,
		&rec.ResolvedBy,
		&rec.Reason,
	)
	if err != nil {
		return nil, err
	}
	if resolved.Valid {
		ts := resolved.Time
		rec.ResolvedAt = &ts
	}
	return &rec, nil
}

func affectedRows(res sql.Result) int64 {
	if res == nil {
		return 0
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0
	}
	return rows
}

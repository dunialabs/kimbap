package store

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const maxAuditQueryLimit = 10000

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
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("dsn is required")
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
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
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("dsn is required")
	}
	db, err := sql.Open("pgx", dsn)
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

func (s *SQLStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLStore) Migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS audit_events (
			id TEXT PRIMARY KEY,
			timestamp TIMESTAMP NOT NULL,
			request_id TEXT NOT NULL,
			trace_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			principal_id TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			service TEXT NOT NULL,
			action TEXT NOT NULL,
			mode TEXT NOT NULL,
			status TEXT NOT NULL,
			policy_decision TEXT NOT NULL,
			duration_ms BIGINT NOT NULL,
			error_code TEXT NOT NULL,
			error_message TEXT NOT NULL,
			input_json TEXT NOT NULL,
			meta_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_tenant_timestamp ON audit_events(tenant_id, timestamp DESC)`,
		`CREATE TABLE IF NOT EXISTS approvals (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			request_id TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			service TEXT NOT NULL,
			action TEXT NOT NULL,
			status TEXT NOT NULL,
			input_json TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			resolved_at TIMESTAMP NULL,
			resolved_by TEXT NOT NULL,
			reason TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_approvals_tenant_status ON approvals(tenant_id, status, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS policies (
			tenant_id TEXT PRIMARY KEY,
			document BYTEA NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
	}

	if s.dialect == "sqlite" {
		queries[len(queries)-1] = `CREATE TABLE IF NOT EXISTS policies (
			tenant_id TEXT PRIMARY KEY,
			document BLOB NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`
	}

	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, s.bind(q)); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLStore) WriteAuditEvent(ctx context.Context, event *AuditRecord) error {
	if event == nil {
		return errors.New("audit event is required")
	}
	if strings.TrimSpace(event.ID) == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO audit_events (
			id, timestamp, request_id, trace_id, tenant_id, principal_id,
			agent_name, service, action, mode, status, policy_decision,
			duration_ms, error_code, error_message, input_json, meta_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`),
		event.ID,
		event.Timestamp,
		event.RequestID,
		event.TraceID,
		event.TenantID,
		event.PrincipalID,
		event.AgentName,
		event.Service,
		event.Action,
		event.Mode,
		event.Status,
		event.PolicyDecision,
		event.DurationMS,
		event.ErrorCode,
		event.ErrorMessage,
		event.InputJSON,
		event.MetaJSON,
	)
	return err
}

func (s *SQLStore) QueryAuditEvents(ctx context.Context, filter AuditFilter) ([]AuditRecord, error) {
	args := make([]any, 0, 12)
	query := `
		SELECT id, timestamp, request_id, trace_id, tenant_id, principal_id,
			agent_name, service, action, mode, status, policy_decision,
			duration_ms, error_code, error_message, input_json, meta_json
		FROM audit_events
		WHERE 1 = 1`

	if filter.TenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, filter.TenantID)
	}
	if filter.AgentName != "" {
		query += " AND agent_name = ?"
		args = append(args, filter.AgentName)
	}
	if filter.Service != "" {
		query += " AND service = ?"
		args = append(args, filter.Service)
	}
	if filter.Action != "" {
		query += " AND action = ?"
		args = append(args, filter.Action)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.From != nil {
		query += " AND timestamp >= ?"
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		query += " AND timestamp <= ?"
		args = append(args, *filter.To)
	}

	effectiveLimit := filter.Limit
	if effectiveLimit <= 0 || effectiveLimit > maxAuditQueryLimit {
		effectiveLimit = maxAuditQueryLimit
	}

	query += " ORDER BY timestamp DESC"
	query += " LIMIT ?"
	args = append(args, effectiveLimit)
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AuditRecord, 0)
	for rows.Next() {
		rec, scanErr := scanAudit(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

func (s *SQLStore) ExportAuditEvents(ctx context.Context, filter AuditFilter, format string, w io.Writer) error {
	if w == nil {
		return errors.New("writer is required")
	}
	records, err := s.QueryAuditEvents(ctx, filter)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json", "jsonl":
		enc := json.NewEncoder(w)
		for i := range records {
			if err := enc.Encode(records[i]); err != nil {
				return err
			}
		}
		return nil
	case "csv":
		cw := csv.NewWriter(w)
		headers := []string{"id", "timestamp", "request_id", "trace_id", "tenant_id", "principal_id", "agent_name", "service", "action", "mode", "status", "policy_decision", "duration_ms", "error_code", "error_message", "input_json", "meta_json"}
		if err := cw.Write(headers); err != nil {
			return err
		}
		for i := range records {
			r := records[i]
			row := []string{r.ID, r.Timestamp.Format(time.RFC3339Nano), r.RequestID, r.TraceID, r.TenantID, r.PrincipalID, r.AgentName, r.Service, r.Action, r.Mode, r.Status, r.PolicyDecision, fmt.Sprintf("%d", r.DurationMS), r.ErrorCode, r.ErrorMessage, r.InputJSON, r.MetaJSON}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	default:
		return fmt.Errorf("unsupported export format %q", format)
	}
}

func (s *SQLStore) CreateApproval(ctx context.Context, req *ApprovalRecord) error {
	if req == nil {
		return errors.New("approval is required")
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
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO approvals (
			id, tenant_id, request_id, agent_name, service, action,
			status, input_json, created_at, expires_at, resolved_at,
			resolved_by, reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`),
		req.ID,
		req.TenantID,
		req.RequestID,
		req.AgentName,
		req.Service,
		req.Action,
		req.Status,
		req.InputJSON,
		req.CreatedAt,
		req.ExpiresAt,
		req.ResolvedAt,
		req.ResolvedBy,
		req.Reason,
	)
	return err
}

func (s *SQLStore) GetApproval(ctx context.Context, id string) (*ApprovalRecord, error) {
	row := s.db.QueryRowContext(ctx, s.bind(`
		SELECT id, tenant_id, request_id, agent_name, service, action,
			status, input_json, created_at, expires_at, resolved_at,
			resolved_by, reason
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

var (
	ErrApprovalAlreadyResolved = errors.New("approval already resolved")
	ErrApprovalExpired         = errors.New("approval has expired")
)

func (s *SQLStore) UpdateApprovalStatus(ctx context.Context, id string, status string, resolvedBy string, reason string) error {
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
		// Distinguish not-found vs already-resolved vs expired
		existing, lookupErr := s.GetApproval(ctx, id)
		if lookupErr != nil {
			return ErrNotFound
		}
		if existing.Status != "pending" {
			return ErrApprovalAlreadyResolved
		}
		if !existing.ExpiresAt.After(now) {
			return ErrApprovalExpired
		}
		return ErrNotFound
	}
	return nil
}

func (s *SQLStore) ListApprovals(ctx context.Context, tenantID string, status string) ([]ApprovalRecord, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, ErrInvalidTenantID
	}
	query := `
		SELECT id, tenant_id, request_id, agent_name, service, action,
			status, input_json, created_at, expires_at, resolved_at,
			resolved_by, reason
		FROM approvals
		WHERE tenant_id = ?`
	args := []any{tenantID}
	if strings.TrimSpace(status) != "" {
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

func (s *SQLStore) SetPolicy(ctx context.Context, tenantID string, document []byte) error {
	if strings.TrimSpace(tenantID) == "" {
		return ErrInvalidTenantID
	}
	now := time.Now().UTC()
	if s.dialect == "postgres" {
		_, err := s.db.ExecContext(ctx, s.bind(`
			INSERT INTO policies (tenant_id, document, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT (tenant_id)
			DO UPDATE SET document = EXCLUDED.document, updated_at = EXCLUDED.updated_at
		`), tenantID, document, now)
		return err
	}
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO policies (tenant_id, document, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(tenant_id)
		DO UPDATE SET document=excluded.document, updated_at=excluded.updated_at
	`), tenantID, document, now)
	return err
}

func (s *SQLStore) GetPolicy(ctx context.Context, tenantID string) ([]byte, error) {
	if strings.TrimSpace(tenantID) == "" {
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

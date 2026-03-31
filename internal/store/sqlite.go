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

	"github.com/dunialabs/kimbap/internal/sqliteutil"
	"github.com/google/uuid"
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

func (s *SQLStore) Migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS tokens (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			display_hint TEXT NOT NULL,
			scopes TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			last_used_at TIMESTAMP NULL,
			revoked_at TIMESTAMP NULL,
			created_by TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_tenant_created ON tokens(tenant_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS service_tokens (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			display_hint TEXT NOT NULL,
			scopes TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			last_used_at TIMESTAMP NULL,
			revoked_at TIMESTAMP NULL,
			created_by TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_service_tokens_tenant_created ON service_tokens(tenant_id, created_at DESC)`,
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
			required_approvals INTEGER NOT NULL DEFAULT 1,
			votes_json TEXT NOT NULL DEFAULT '[]',
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			resolved_at TIMESTAMP NULL,
			resolved_by TEXT NOT NULL,
			reason TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_approvals_tenant_status ON approvals(tenant_id, status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_approvals_expiry ON approvals(status, expires_at)`,
		`CREATE TABLE IF NOT EXISTS policies (
			tenant_id TEXT PRIMARY KEY,
			document BYTEA NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS held_executions (
			approval_request_id TEXT PRIMARY KEY,
			request_json TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS webhook_subscriptions (
			id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			url TEXT NOT NULL,
			secret TEXT NOT NULL,
			events_json TEXT NOT NULL,
			active BOOLEAN NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (id, tenant_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_subscriptions_tenant_active ON webhook_subscriptions(tenant_id, active, updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS webhook_events (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			type TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			data_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_events_tenant_timestamp ON webhook_events(tenant_id, timestamp DESC, id DESC)`,
	}

	if s.dialect == "sqlite" {
		for i := range queries {
			if strings.Contains(queries[i], "CREATE TABLE IF NOT EXISTS policies") {
				queries[i] = `CREATE TABLE IF NOT EXISTS policies (
					tenant_id TEXT PRIMARY KEY,
					document BLOB NOT NULL,
					updated_at TIMESTAMP NOT NULL
				)`
				break
			}
		}
	}

	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, s.bind(q)); err != nil {
			return err
		}
	}

	if s.dialect == "postgres" {
		for _, q := range []string{
			`ALTER TABLE approvals ADD COLUMN IF NOT EXISTS required_approvals INTEGER NOT NULL DEFAULT 1`,
			`ALTER TABLE approvals ADD COLUMN IF NOT EXISTS votes_json TEXT NOT NULL DEFAULT '[]'`,
		} {
			if _, err := s.db.ExecContext(ctx, s.bind(q)); err != nil {
				return err
			}
		}
	} else {
		for _, q := range []string{
			`ALTER TABLE approvals ADD COLUMN required_approvals INTEGER NOT NULL DEFAULT 1`,
			`ALTER TABLE approvals ADD COLUMN votes_json TEXT NOT NULL DEFAULT '[]'`,
		} {
			if _, err := s.db.ExecContext(ctx, s.bind(q)); err != nil {
				if !isColumnAlreadyExistsError(err) {
					return err
				}
			}
		}
	}

	needsBackfill, err := s.needsServiceTokenBackfill(ctx)
	if err != nil {
		return err
	}
	if !needsBackfill {
		return nil
	}

	backfillServiceTokens := `INSERT INTO service_tokens (id, tenant_id, agent_name, token_hash, display_hint, scopes, created_at, expires_at, last_used_at, revoked_at, created_by)
		SELECT id, tenant_id, agent_name, token_hash, display_hint, scopes, created_at, expires_at, last_used_at, revoked_at, created_by
		FROM tokens
		ON CONFLICT DO NOTHING`
	if s.dialect == "sqlite" {
		backfillServiceTokens = `INSERT OR IGNORE INTO service_tokens (id, tenant_id, agent_name, token_hash, display_hint, scopes, created_at, expires_at, last_used_at, revoked_at, created_by)
			SELECT id, tenant_id, agent_name, token_hash, display_hint, scopes, created_at, expires_at, last_used_at, revoked_at, created_by
			FROM tokens`
	}
	if _, err := s.db.ExecContext(ctx, s.bind(backfillServiceTokens)); err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) needsServiceTokenBackfill(ctx context.Context) (bool, error) {
	var hasMissing bool
	err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT EXISTS (
			SELECT 1
			FROM tokens t
			LEFT JOIN service_tokens st ON st.id = t.id
			WHERE st.id IS NULL
			LIMIT 1
		)
	`)).Scan(&hasMissing)
	if err != nil {
		return false, err
	}
	return hasMissing, nil
}

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

func buildAuditQueryArgs(filter AuditFilter, unlimited bool) (string, []any) {
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

	query += " ORDER BY timestamp DESC, id DESC"
	if !unlimited {
		effectiveLimit := filter.Limit
		if effectiveLimit <= 0 || effectiveLimit > maxAuditQueryLimit {
			effectiveLimit = maxAuditQueryLimit
		}
		query += " LIMIT ?"
		args = append(args, effectiveLimit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}
	return query, args
}

func (s *SQLStore) QueryAuditEvents(ctx context.Context, filter AuditFilter) ([]AuditRecord, error) {
	query, args := buildAuditQueryArgs(filter, false)
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
	query, args := buildAuditQueryArgs(filter, true)
	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json", "jsonl":
		enc := json.NewEncoder(w)
		for rows.Next() {
			rec, scanErr := scanAudit(rows)
			if scanErr != nil {
				return scanErr
			}
			if err := enc.Encode(*rec); err != nil {
				return err
			}
		}
		return rows.Err()
	case "csv":
		cw := csv.NewWriter(w)
		headers := []string{"id", "timestamp", "request_id", "trace_id", "tenant_id", "principal_id", "agent_name", "service", "action", "mode", "status", "policy_decision", "duration_ms", "error_code", "error_message", "input_json", "meta_json"}
		if err := cw.Write(headers); err != nil {
			return err
		}
		for rows.Next() {
			r, scanErr := scanAudit(rows)
			if scanErr != nil {
				return scanErr
			}
			row := []string{r.ID, r.Timestamp.Format(time.RFC3339Nano), r.RequestID, r.TraceID, r.TenantID, r.PrincipalID, r.AgentName, r.Service, r.Action, r.Mode, r.Status, r.PolicyDecision, fmt.Sprintf("%d", r.DurationMS), r.ErrorCode, r.ErrorMessage, r.InputJSON, r.MetaJSON}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
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

func (s *SQLStore) UpsertWebhookSubscription(ctx context.Context, sub *WebhookSubscriptionRecord) error {
	if sub == nil {
		return errors.New("webhook subscription is required")
	}
	sub.ID = strings.TrimSpace(sub.ID)
	sub.TenantID = strings.TrimSpace(sub.TenantID)
	if sub.ID == "" {
		return errors.New("webhook subscription id is required")
	}
	if sub.TenantID == "" {
		return ErrInvalidTenantID
	}
	if strings.TrimSpace(sub.EventsJSON) == "" {
		sub.EventsJSON = "[]"
	}
	now := time.Now().UTC()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO webhook_subscriptions (
			id, tenant_id, url, secret, events_json, active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id, tenant_id)
		DO UPDATE SET
			url = EXCLUDED.url,
			secret = EXCLUDED.secret,
			events_json = EXCLUDED.events_json,
			active = EXCLUDED.active,
			updated_at = EXCLUDED.updated_at
	`),
		sub.ID,
		sub.TenantID,
		sub.URL,
		sub.Secret,
		sub.EventsJSON,
		sub.Active,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	return err
}

func (s *SQLStore) DeleteWebhookSubscription(ctx context.Context, id string, tenantID string) error {
	id = strings.TrimSpace(id)
	tenantID = strings.TrimSpace(tenantID)
	if id == "" {
		return errors.New("webhook subscription id is required")
	}
	if tenantID == "" {
		return ErrInvalidTenantID
	}
	_, err := s.db.ExecContext(ctx, s.bind(`DELETE FROM webhook_subscriptions WHERE id = ? AND tenant_id = ?`), id, tenantID)
	return err
}

func (s *SQLStore) ListWebhookSubscriptions(ctx context.Context, tenantID string) ([]WebhookSubscriptionRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	query := `
		SELECT id, tenant_id, url, secret, events_json, active, created_at, updated_at
		FROM webhook_subscriptions
		WHERE active = ?`
	args := []any{true}
	if tenantID != "" {
		query += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}
	query += ` ORDER BY updated_at DESC, id DESC`

	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WebhookSubscriptionRecord, 0)
	for rows.Next() {
		var rec WebhookSubscriptionRecord
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.URL, &rec.Secret, &rec.EventsJSON, &rec.Active, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *SQLStore) WriteWebhookEvent(ctx context.Context, event *WebhookEventRecord) error {
	if event == nil {
		return errors.New("webhook event is required")
	}
	event.ID = strings.TrimSpace(event.ID)
	event.TenantID = strings.TrimSpace(event.TenantID)
	event.Type = strings.TrimSpace(event.Type)
	if event.ID == "" {
		event.ID = "evt_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	if event.TenantID == "" {
		return ErrInvalidTenantID
	}
	if event.Type == "" {
		return errors.New("webhook event type is required")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(event.DataJSON) == "" {
		event.DataJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO webhook_events (id, tenant_id, type, timestamp, data_json)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			type = EXCLUDED.type,
			timestamp = EXCLUDED.timestamp,
			data_json = EXCLUDED.data_json
	`), event.ID, event.TenantID, event.Type, event.Timestamp, event.DataJSON)
	return err
}

func (s *SQLStore) ListWebhookEvents(ctx context.Context, tenantID string, limit int) ([]WebhookEventRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	query := `
		SELECT id, tenant_id, type, timestamp, data_json
		FROM webhook_events
		WHERE 1 = 1`
	args := make([]any, 0, 2)
	if tenantID != "" {
		query += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}
	query += ` ORDER BY timestamp DESC, id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tmp := make([]WebhookEventRecord, 0)
	for rows.Next() {
		var rec WebhookEventRecord
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Type, &rec.Timestamp, &rec.DataJSON); err != nil {
			return nil, err
		}
		tmp = append(tmp, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i, j := 0, len(tmp)-1; i < j; i, j = i+1, j-1 {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	}
	return tmp, nil
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

func isColumnAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}

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

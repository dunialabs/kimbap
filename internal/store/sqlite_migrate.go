package store

import (
	"context"
	"strings"
)

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

func isColumnAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}

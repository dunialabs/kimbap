package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
)

func openConnectorStore(cfg *config.KimbapConfig) (connectors.ConnectorStore, error) {
	db, dialect, err := openConnectorDB(cfg)
	if err != nil {
		return nil, err
	}
	return &sqlConnectorStore{db: db, dialect: dialect}, nil
}

type sqlConnectorStore struct {
	db      *sql.DB
	dialect string
	once    sync.Once
	initErr error
}

func (s *sqlConnectorStore) ensureConnectorSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("connector store database is required")
	}
	s.once.Do(func() {
		if err := migrateConnectorTable(ctx, s.db, s.dialect); err != nil {
			s.initErr = fmt.Errorf("migrate connector table: %w", err)
		}
	})
	return s.initErr
}

func (s *sqlConnectorStore) Save(ctx context.Context, state *connectors.ConnectorState) error {
	if err := s.ensureConnectorSchema(ctx); err != nil {
		return err
	}
	scopesJSON := strings.Join(state.Scopes, " ")
	q := `INSERT INTO connector_states (tenant_id, name, provider, status, account, expires_at, updated_at, last_refresh, scopes_json,
			access_token, refresh_token, workspace_id, connected_principal, connection_scope, revoked_at, flow_used, last_refresh_error, last_used_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, name) DO UPDATE SET
			provider=excluded.provider, status=excluded.status, account=excluded.account,
			expires_at=excluded.expires_at, updated_at=excluded.updated_at,
			last_refresh=excluded.last_refresh, scopes_json=excluded.scopes_json,
			access_token=excluded.access_token, refresh_token=excluded.refresh_token,
			workspace_id=excluded.workspace_id, connected_principal=excluded.connected_principal,
			connection_scope=excluded.connection_scope, revoked_at=excluded.revoked_at,
			flow_used=excluded.flow_used, last_refresh_error=excluded.last_refresh_error,
			last_used_at=excluded.last_used_at`
	_, err := s.db.ExecContext(ctx, bindQuery(q, s.dialect),
		state.TenantID, state.Name, state.Provider, string(state.Status),
		state.Account, state.ExpiresAt, state.UpdatedAt, state.LastRefresh, scopesJSON,
		state.AccessToken, state.RefreshToken, state.WorkspaceID, state.ConnectedPrincipal,
		string(state.ConnectionScope), state.RevokedAt, string(state.FlowUsed), state.LastRefreshError, state.LastUsedAt,
		state.CreatedAt,
	)
	return err
}

func scanConnectorRow(scanner interface{ Scan(...any) error }, tenantID string) (connectors.ConnectorState, error) {
	var (
		st          connectors.ConnectorState
		scopesJSON  string
		expiresAt   sql.NullTime
		updatedAt   sql.NullTime
		lastRefresh sql.NullTime
		revokedAt   sql.NullTime
		lastUsedAt  sql.NullTime
		createdAt   sql.NullTime
		connScope   string
		flowUsed    string
	)
	err := scanner.Scan(&st.Name, &st.Provider, &st.Status, &st.Account,
		&expiresAt, &updatedAt, &lastRefresh, &scopesJSON,
		&st.AccessToken, &st.RefreshToken, &st.WorkspaceID, &st.ConnectedPrincipal,
		&connScope, &revokedAt, &flowUsed, &st.LastRefreshError, &lastUsedAt, &createdAt,
	)
	if err != nil {
		return connectors.ConnectorState{}, err
	}

	st.TenantID = tenantID
	st.Scopes = strings.Fields(scopesJSON)
	st.Profile = connectorProfileFromName(st.Name)
	st.ConnectionScope = connectors.ConnectionScope(connScope)
	st.FlowUsed = connectors.FlowType(flowUsed)
	if expiresAt.Valid {
		t := expiresAt.Time.UTC()
		st.ExpiresAt = &t
	}
	if updatedAt.Valid {
		st.UpdatedAt = updatedAt.Time.UTC()
	}
	if lastRefresh.Valid {
		t := lastRefresh.Time.UTC()
		st.LastRefresh = &t
	}
	if revokedAt.Valid {
		t := revokedAt.Time.UTC()
		st.RevokedAt = &t
	}
	if lastUsedAt.Valid {
		t := lastUsedAt.Time.UTC()
		st.LastUsedAt = &t
	}
	if createdAt.Valid {
		st.CreatedAt = createdAt.Time.UTC()
	}
	return st, nil
}

func (s *sqlConnectorStore) Get(ctx context.Context, tenantID, name string) (*connectors.ConnectorState, error) {
	if err := s.ensureConnectorSchema(ctx); err != nil {
		return nil, err
	}
	q := `SELECT name, provider, status, account, expires_at, updated_at, last_refresh, scopes_json,
		access_token, refresh_token, workspace_id, connected_principal, connection_scope, revoked_at, flow_used, last_refresh_error, last_used_at, created_at
		FROM connector_states WHERE tenant_id = ? AND name = ?`
	row := s.db.QueryRowContext(ctx, bindQuery(q, s.dialect), tenantID, name)

	st, err := scanConnectorRow(row, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &st, nil
}

func (s *sqlConnectorStore) List(ctx context.Context, tenantID string) ([]connectors.ConnectorState, error) {
	if err := s.ensureConnectorSchema(ctx); err != nil {
		return nil, err
	}
	q := `SELECT name, provider, status, account, expires_at, updated_at, last_refresh, scopes_json,
		access_token, refresh_token, workspace_id, connected_principal, connection_scope, revoked_at, flow_used, last_refresh_error, last_used_at, created_at
		FROM connector_states WHERE tenant_id = ? ORDER BY name ASC`
	rows, err := s.db.QueryContext(ctx, bindQuery(q, s.dialect), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []connectors.ConnectorState
	for rows.Next() {
		st, scanErr := scanConnectorRow(rows, tenantID)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *sqlConnectorStore) Delete(ctx context.Context, tenantID, name string) error {
	if err := s.ensureConnectorSchema(ctx); err != nil {
		return err
	}
	q := `DELETE FROM connector_states WHERE tenant_id = ? AND name = ?`
	_, err := s.db.ExecContext(ctx, bindQuery(q, s.dialect), tenantID, name)
	return err
}

func (s *sqlConnectorStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func connectorProfileFromName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "default"
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return "default"
	}
	return strings.TrimSpace(parts[1])
}

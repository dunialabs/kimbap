package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/spf13/cobra"
)

func newConnectorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "connector",
		Short:  "Removed — use 'kimbap auth' instead",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("'kimbap connector' has been removed.\n\nUse these commands instead:\n  kimbap auth connect <provider>   (was: connector login)\n  kimbap auth list                 (was: connector list)\n  kimbap auth status <provider>    (was: connector status)\n  kimbap auth reconnect <provider> (was: connector refresh)")
		},
	}
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		fmt.Fprintln(c.ErrOrStderr(), c.RunE(c, nil))
	})
	return cmd
}

func connectorTenant(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultTenantID()
	}
	return trimmed
}

type connectorStateRow struct {
	Name               string
	Provider           string
	Status             string
	Account            string
	ConnectedPrincipal string
	ExpiresAt          *string
	RevokedAt          *string
	UpdatedAt          *string
	LastRefresh        *string
	LastRefreshError   string
	AccessToken        string
	Scopes             []string
}

func listConnectorStates(ctx context.Context, cfg *config.KimbapConfig, tenantID string) ([]connectorStateRow, error) {
	db, dialect, err := openConnectorDB(cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := migrateConnectorTable(ctx, db, dialect); err != nil {
		return nil, err
	}

	q := `SELECT name, provider, status, account, connected_principal, expires_at, updated_at, last_refresh, scopes_json, revoked_at, last_refresh_error, access_token FROM connector_states WHERE tenant_id = ? ORDER BY name ASC`
	rows, err := db.QueryContext(ctx, bindQuery(q, dialect), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]connectorStateRow, 0)
	for rows.Next() {
		item, scanErr := scanConnectorState(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func openConnectorDB(cfg *config.KimbapConfig) (*sql.DB, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("config is required")
	}
	driver := strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	dsn := strings.TrimSpace(cfg.Database.DSN)
	switch driver {
	case "", "sqlite", "sqlite3":
		if dsn == "" {
			dsn = filepath.Join(cfg.DataDir, "kimbap.db")
		}
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			return nil, "", err
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			_ = db.Close()
			return nil, "", fmt.Errorf("ping connector sqlite database: %w", err)
		}
		return db, "sqlite", nil
	case "postgres", "postgresql", "pgx":
		if dsn == "" {
			return nil, "", fmt.Errorf("database dsn is required for postgres")
		}
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, "", err
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			_ = db.Close()
			return nil, "", fmt.Errorf("ping connector postgres database: %w", err)
		}
		return db, "postgres", nil
	default:
		return nil, "", fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}
}

func openConnectorStoreReadOnly(cfg *config.KimbapConfig) (connectors.ConnectorStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	driver := strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	dsn := strings.TrimSpace(cfg.Database.DSN)
	switch driver {
	case "", "sqlite", "sqlite3":
		if dsn == "" {
			dsn = filepath.Join(cfg.DataDir, "kimbap.db")
		}
		db, err := sql.Open("sqlite", sqliteReadOnlyDSN(dsn))
		if err != nil {
			return nil, err
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("ping connector sqlite database: %w", err)
		}
		return &readOnlySQLConnectorStore{db: db, dialect: "sqlite"}, nil
	case "postgres", "postgresql", "pgx":
		if dsn == "" {
			return nil, fmt.Errorf("database dsn is required for postgres")
		}
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("ping connector postgres database: %w", err)
		}
		return &readOnlySQLConnectorStore{db: db, dialect: "postgres"}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}
}

func sqliteReadOnlyDSN(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(strings.ToLower(trimmed), "file:") {
		if strings.Contains(strings.ToLower(trimmed), "mode=ro") {
			return trimmed
		}
		if strings.Contains(trimmed, "?") {
			return trimmed + "&mode=ro"
		}
		return trimmed + "?mode=ro"
	}
	return "file:" + trimmed + "?mode=ro"
}

func migrateConnectorTable(ctx context.Context, db *sql.DB, dialect string) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}
	query := `CREATE TABLE IF NOT EXISTS connector_states (
		tenant_id TEXT NOT NULL,
		name TEXT NOT NULL,
		provider TEXT NOT NULL,
		status TEXT NOT NULL,
		account TEXT NOT NULL DEFAULT '',
		expires_at TIMESTAMP NULL,
		updated_at TIMESTAMP NULL,
		last_refresh TIMESTAMP NULL,
		scopes_json TEXT NOT NULL DEFAULT '',
		access_token TEXT NOT NULL DEFAULT '',
		refresh_token TEXT NOT NULL DEFAULT '',
		workspace_id TEXT NOT NULL DEFAULT '',
		connected_principal TEXT NOT NULL DEFAULT '',
		connection_scope TEXT NOT NULL DEFAULT 'user',
		revoked_at TIMESTAMP NULL,
		flow_used TEXT NOT NULL DEFAULT '',
		last_refresh_error TEXT NOT NULL DEFAULT '',
		last_used_at TIMESTAMP NULL,
		created_at TIMESTAMP NULL,
		PRIMARY KEY (tenant_id, name)
	)`
	if _, err := db.ExecContext(ctx, bindQuery(query, dialect)); err != nil {
		return err
	}

	addColumns := []string{
		"ALTER TABLE connector_states ADD COLUMN access_token TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE connector_states ADD COLUMN refresh_token TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE connector_states ADD COLUMN workspace_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE connector_states ADD COLUMN connected_principal TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE connector_states ADD COLUMN connection_scope TEXT NOT NULL DEFAULT 'user'",
		"ALTER TABLE connector_states ADD COLUMN revoked_at TIMESTAMP NULL",
		"ALTER TABLE connector_states ADD COLUMN flow_used TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE connector_states ADD COLUMN last_refresh_error TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE connector_states ADD COLUMN last_used_at TIMESTAMP NULL",
		"ALTER TABLE connector_states ADD COLUMN created_at TIMESTAMP NULL",
	}
	for _, ddl := range addColumns {
		if _, execErr := db.ExecContext(ctx, bindQuery(ddl, dialect)); execErr != nil {
			errMsg := strings.ToLower(execErr.Error())
			isDuplicate := strings.Contains(errMsg, "duplicate column") ||
				strings.Contains(errMsg, "already exists") ||
				strings.Contains(errMsg, "duplicate")
			if !isDuplicate {
				return fmt.Errorf("migrate connector table: %w", execErr)
			}
		}
	}

	if _, err := db.ExecContext(ctx, bindQuery(`CREATE INDEX IF NOT EXISTS idx_connector_states_tenant_name ON connector_states(tenant_id, name)`, dialect)); err != nil {
		return fmt.Errorf("migrate connector table index: %w", err)
	}
	return nil
}

func scanConnectorState(scanner interface{ Scan(dest ...any) error }) (connectorStateRow, error) {
	var (
		item        connectorStateRow
		expiresAt   sql.NullTime
		updatedAt   sql.NullTime
		lastRefresh sql.NullTime
		revokedAt   sql.NullTime
		scopesJSON  string
	)
	if err := scanner.Scan(&item.Name, &item.Provider, &item.Status, &item.Account, &item.ConnectedPrincipal, &expiresAt, &updatedAt, &lastRefresh, &scopesJSON, &revokedAt, &item.LastRefreshError, &item.AccessToken); err != nil {
		return connectorStateRow{}, err
	}
	if expiresAt.Valid {
		ts := expiresAt.Time.UTC().Format(timeLayoutRFC3339)
		item.ExpiresAt = &ts
	}
	if updatedAt.Valid {
		ts := updatedAt.Time.UTC().Format(timeLayoutRFC3339)
		item.UpdatedAt = &ts
	}
	if lastRefresh.Valid {
		ts := lastRefresh.Time.UTC().Format(timeLayoutRFC3339)
		item.LastRefresh = &ts
	}
	if revokedAt.Valid {
		ts := revokedAt.Time.UTC().Format(timeLayoutRFC3339)
		item.RevokedAt = &ts
	}
	if trimmed := strings.TrimSpace(scopesJSON); trimmed != "" {
		if json.Unmarshal([]byte(trimmed), &item.Scopes) != nil {
			item.Scopes = strings.Fields(trimmed)
		}
	}
	return item, nil
}

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"

func bindQuery(query, dialect string) string {
	if dialect != "postgres" {
		return query
	}
	var out strings.Builder
	out.Grow(len(query) + 8)
	idx := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			_, _ = fmt.Fprintf(&out, "$%d", idx)
			idx++
			continue
		}
		out.WriteByte(query[i])
	}
	return out.String()
}

func connectorComputedStatus(state connectorStateRow) string {
	if state.RevokedAt != nil {
		return "revoked"
	}
	if strings.TrimSpace(state.LastRefreshError) != "" {
		return "refresh_failed"
	}
	if state.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339, *state.ExpiresAt); err == nil {
			if t.Before(time.Now()) {
				return "expired"
			}
			if t.Before(time.Now().Add(5 * time.Minute)) {
				return "degraded"
			}
		}
	}
	if strings.TrimSpace(state.AccessToken) == "" {
		return "connecting"
	}
	mapped := connectors.MapLegacyStatus(connectors.ConnectorStatus(state.Status))
	return string(mapped)
}

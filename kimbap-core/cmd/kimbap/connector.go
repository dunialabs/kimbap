package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	"github.com/spf13/cobra"
)

func newConnectorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connector",
		Short: "Manage OAuth connectors",
	}

	cmd.AddCommand(newConnectorLoginCommand())
	cmd.AddCommand(newConnectorListCommand())
	cmd.AddCommand(newConnectorStatusCommand())
	cmd.AddCommand(newConnectorRefreshCommand())

	return cmd
}

func newConnectorLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login <name>",
		Short: "Start connector device-flow login",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("connector name is required")
			}
			return fmt.Errorf("connector login is not yet available.\n\nConnector login for %q requires a running ConnectorManager with OAuth device-flow support.\nThis feature is under development. Track progress: https://github.com/dunialabs/kimbap/issues\n\nAvailable connector commands: list, status", name)
		},
	}
	return cmd
}

func newConnectorListCommand() *cobra.Command {
	var tenant string
	cmd := &cobra.Command{
		Use:   "list [--tenant <id>]",
		Short: "List connector states",
		RunE: func(_ *cobra.Command, _ []string) error {
			activeTenant := connectorTenant(tenant)
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			states, err := listConnectorStates(contextBackground(), cfg, activeTenant)
			if err != nil {
				return printOutput(map[string]any{
					"status":    "not_configured",
					"operation": "connector.list",
					"tenant_id": activeTenant,
					"connectors": []map[string]any{
						{"name": "gmail", "status": connectors.StatusPending, "provider": "oauth2"},
					},
					"message": fmt.Sprintf("Connector state store unavailable: %v", err),
					"next":    "Configure database in ~/.kimbap/config.yaml and start runtime migrations.",
				})
			}

			connectorsOut := make([]map[string]any, 0, len(states))
			for _, state := range states {
				connectorsOut = append(connectorsOut, map[string]any{
					"name":         state.Name,
					"provider":     state.Provider,
					"status":       state.Status,
					"account":      state.Account,
					"expires_at":   state.ExpiresAt,
					"updated_at":   state.UpdatedAt,
					"last_refresh": state.LastRefresh,
					"scopes":       state.Scopes,
				})
			}

			return printOutput(map[string]any{
				"status":     "ok",
				"operation":  "connector.list",
				"tenant_id":  activeTenant,
				"count":      len(connectorsOut),
				"connectors": connectorsOut,
			})
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func newConnectorStatusCommand() *cobra.Command {
	var tenant string
	cmd := &cobra.Command{
		Use:   "status <name> [--tenant <id>]",
		Short: "Show connector status details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("connector name is required")
			}
			activeTenant := connectorTenant(tenant)
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			state, err := getConnectorState(contextBackground(), cfg, activeTenant, name)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return printOutput(map[string]any{
						"status":    "not_found",
						"operation": "connector.status",
						"tenant_id": activeTenant,
						"connector": name,
						"message":   fmt.Sprintf("No connector state found for %q.", name),
					})
				}
				return printOutput(map[string]any{
					"status":    "not_configured",
					"operation": "connector.status",
					"tenant_id": activeTenant,
					"connector": name,
					"message":   fmt.Sprintf("Connector state store unavailable: %v", err),
				})
			}

			return printOutput(map[string]any{
				"status":    "ok",
				"operation": "connector.status",
				"tenant_id": activeTenant,
				"connector": map[string]any{
					"name":         state.Name,
					"provider":     state.Provider,
					"status":       state.Status,
					"account":      state.Account,
					"expires_at":   state.ExpiresAt,
					"updated_at":   state.UpdatedAt,
					"last_refresh": state.LastRefresh,
					"scopes":       state.Scopes,
				},
			})
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func newConnectorRefreshCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh <name>",
		Short: "Force refresh connector access token",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("connector name is required")
			}
			return fmt.Errorf("connector refresh is not yet available.\n\nConnector refresh for %q requires ConnectorManager with OAuth refresh-token flow.\nThis feature is under development. Track progress: https://github.com/dunialabs/kimbap/issues\n\nAvailable connector commands: list, status", name)
		},
	}
	return cmd
}

func connectorTenant(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return defaultTenantID()
	}
	return raw
}

type connectorStateRow struct {
	Name        string
	Provider    string
	Status      string
	Account     string
	ExpiresAt   *string
	UpdatedAt   *string
	LastRefresh *string
	Scopes      []string
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

	q := `SELECT name, provider, status, account, expires_at, updated_at, last_refresh, scopes_json FROM connector_states WHERE tenant_id = ? ORDER BY name ASC`
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

func getConnectorState(ctx context.Context, cfg *config.KimbapConfig, tenantID, name string) (*connectorStateRow, error) {
	db, dialect, err := openConnectorDB(cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := migrateConnectorTable(ctx, db, dialect); err != nil {
		return nil, err
	}

	q := `SELECT name, provider, status, account, expires_at, updated_at, last_refresh, scopes_json FROM connector_states WHERE tenant_id = ? AND name = ?`
	row := db.QueryRowContext(ctx, bindQuery(q, dialect), tenantID, name)
	item, err := scanConnectorState(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
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
		return db, "sqlite", nil
	case "postgres", "postgresql", "pgx":
		if dsn == "" {
			return nil, "", fmt.Errorf("database dsn is required for postgres")
		}
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, "", err
		}
		return db, "postgres", nil
	default:
		return nil, "", fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}
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
		account TEXT NOT NULL,
		expires_at TIMESTAMP NULL,
		updated_at TIMESTAMP NULL,
		last_refresh TIMESTAMP NULL,
		scopes_json TEXT NOT NULL,
		PRIMARY KEY (tenant_id, name)
	)`
	if _, err := db.ExecContext(ctx, bindQuery(query, dialect)); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, bindQuery(`CREATE INDEX IF NOT EXISTS idx_connector_states_tenant_name ON connector_states(tenant_id, name)`, dialect)); err != nil {
		return err
	}
	return nil
}

func scanConnectorState(scanner interface{ Scan(dest ...any) error }) (connectorStateRow, error) {
	var (
		item        connectorStateRow
		expiresAt   sql.NullTime
		updatedAt   sql.NullTime
		lastRefresh sql.NullTime
		scopesJSON  string
	)
	if err := scanner.Scan(&item.Name, &item.Provider, &item.Status, &item.Account, &expiresAt, &updatedAt, &lastRefresh, &scopesJSON); err != nil {
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
	item.Scopes = strings.Fields(scopesJSON)
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

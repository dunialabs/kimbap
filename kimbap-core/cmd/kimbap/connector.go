package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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
	var (
		flow            string
		scopeInput      string
		browserName     string
		noOpen          bool
		port            int
		timeout         time.Duration
		workspace       string
		connectionScope string
		extras          []string
		tenant          string
	)
	cmd := &cobra.Command{
		Use:   "login <name>",
		Short: "Start connector OAuth login",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("connector name is required")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			extraValues, parseErr := parseExtrasStrict(extras)
			if parseErr != nil {
				return parseErr
			}

			return runAuthConnect(
				cfg,
				name,
				connectorTenant(tenant),
				flow,
				scopeInput,
				browserName,
				noOpen,
				port,
				timeout,
				workspace,
				connectionScope,
				"default",
				false,
				extraValues,
			)
		},
	}
	cmd.Flags().StringVar(&flow, "flow", "auto", "auth flow to use (auto, browser, device)")
	cmd.Flags().StringVar(&scopeInput, "scope", "", "requested scopes (space/comma separated)")
	cmd.Flags().StringVar(&scopeInput, "scopes", "", "requested scopes (space/comma separated)")
	cmd.Flags().StringVar(&browserName, "browser", "auto", "browser strategy (auto, system, none)")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "do not automatically open browser")
	cmd.Flags().IntVar(&port, "port", 0, "local callback port for browser flow")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "authorization timeout")
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace id for workspace-scoped connection")
	cmd.Flags().StringVar(&connectionScope, "connection-scope", string(connectors.ScopeUser), "connection scope (user, workspace, service)")
	cmd.Flags().StringArrayVar(&extras, "extra", nil, "provider-specific key=value pairs for placeholder endpoints")
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
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
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":     "not_configured",
						"operation":  "connector.list",
						"tenant_id":  activeTenant,
						"connectors": []map[string]any{},
						"message":    fmt.Sprintf("Connector state store unavailable: %v", err),
						"next":       "Configure database in ~/.kimbap/config.yaml and start runtime migrations.",
					})
				}
				return fmt.Errorf("connector state store unavailable: %w", err)
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
					if outputAsJSON() {
						_ = printOutput(map[string]any{
							"status":    "not_found",
							"operation": "connector.status",
							"tenant_id": activeTenant,
							"connector": name,
							"message":   fmt.Sprintf("No connector state found for %q.", name),
						})
					}
					return fmt.Errorf("connector %q not found", name)
				}
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "not_configured",
						"operation": "connector.status",
						"tenant_id": activeTenant,
						"connector": name,
						"message":   fmt.Sprintf("Connector state store unavailable: %v", err),
					})
				}
				return fmt.Errorf("connector state store unavailable: %w", err)
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
	var tenant string
	var extras []string
	cmd := &cobra.Command{
		Use:   "refresh <name> [--tenant <id>]",
		Short: "Force refresh connector access token",
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

			store, storeErr := openConnectorStore(cfg)
			if storeErr != nil {
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "not_configured",
						"operation": "connector.refresh",
						"tenant_id": activeTenant,
						"connector": name,
						"message":   fmt.Sprintf("Connector state store unavailable: %v", storeErr),
					})
				}
				return fmt.Errorf("connector state store unavailable: %w", storeErr)
			}

			mgr := connectors.NewManager(store)

			provider, provErr := providers.GetProvider(name)
			if provErr != nil {
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "not_found",
						"operation": "connector.refresh",
						"tenant_id": activeTenant,
						"connector": name,
						"message":   fmt.Sprintf("Provider %q not found in registry: %v", name, provErr),
						"next":      "Check available providers with: kimbap auth providers list",
					})
				}
				return fmt.Errorf("provider %q not found: %w", name, provErr)
			}
			name = provider.ID
			extraValues, parseErr := parseExtrasStrict(extras)
			if parseErr != nil {
				return parseErr
			}
			if valErr := validateProviderExtraValues(provider, extraValues); valErr != nil {
				return valErr
			}
			provider = substituteProviderEndpoints(provider, extraValues)
			if hasUnresolvedPlaceholders(provider) {
				missing := listUnresolvedPlaceholders(provider)
				return fmt.Errorf("provider %q has unresolved endpoint placeholders: %s (use --extra key=value)", name, strings.Join(missing, ", "))
			}
			if vErr := validateProviderEndpoints(provider); vErr != nil {
				return fmt.Errorf("provider %q endpoint validation failed: %w", name, vErr)
			}

			connCreds := resolveOAuthCreds(cfg, provider.ID)
			mgr.RegisterConfig(connectors.ConnectorConfig{
				Name:         name,
				Provider:     provider.ID,
				ClientID:     connCreds.ClientID,
				ClientSecret: connCreds.ClientSecret,
				AuthMethod:   connCreds.AuthMethod,
				TokenURL:     provider.TokenEndpoint,
				DeviceURL:    provider.DeviceEndpoint,
				Scopes:       provider.DefaultScopes,
			})

			if refreshErr := mgr.Refresh(contextBackground(), activeTenant, name); refreshErr != nil {
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "error",
						"operation": "connector.refresh",
						"tenant_id": activeTenant,
						"connector": name,
						"message":   refreshErr.Error(),
						"next":      fmt.Sprintf("If refresh token is missing or expired, run: kimbap connector login %s", name),
					})
				}
				return fmt.Errorf("connector refresh failed: %w", refreshErr)
			}

			return printOutput(map[string]any{
				"status":    "ok",
				"operation": "connector.refresh",
				"tenant_id": activeTenant,
				"connector": name,
				"refreshed": true,
			})
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	cmd.Flags().StringArrayVar(&extras, "extra", nil, "provider-specific key=value pairs for placeholder endpoints")
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
	Name             string
	Provider         string
	Status           string
	Account          string
	ExpiresAt        *string
	RevokedAt        *string
	UpdatedAt        *string
	LastRefresh      *string
	LastRefreshError string
	AccessToken      string
	Scopes           []string
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

	_, _ = db.ExecContext(ctx, bindQuery(`CREATE INDEX IF NOT EXISTS idx_connector_states_tenant_name ON connector_states(tenant_id, name)`, dialect))
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
			if t.Before(time.Now().Add(15 * time.Minute)) {
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

package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"

	_ "modernc.org/sqlite"
)

func TestSQLConnectorStoreSaveSurfacesMigrationErrors(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	store := &sqlConnectorStore{db: db, dialect: "sqlite"}
	now := time.Now().UTC()
	err = store.Save(context.Background(), &connectors.ConnectorState{
		TenantID:  "tenant-a",
		Name:      "github",
		Provider:  "github",
		Status:    connectors.StatusHealthy,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err == nil {
		t.Fatal("expected migration error when using closed db")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "migrate connector table") {
		t.Fatalf("expected migration context in error, got: %v", err)
	}
}

func TestSQLConnectorStoreRoundTripAndIdempotentMigration(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "connector.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store := &sqlConnectorStore{db: db, dialect: "sqlite"}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	state := &connectors.ConnectorState{
		TenantID:        "tenant-a",
		Name:            "github",
		Provider:        "github",
		Status:          connectors.StatusHealthy,
		Scopes:          []string{"repo", "read:user"},
		CreatedAt:       now,
		UpdatedAt:       now,
		ConnectionScope: connectors.ScopeUser,
	}

	if err := store.Save(context.Background(), state); err != nil {
		t.Fatalf("save initial state: %v", err)
	}

	state.Status = connectors.StatusExpiring
	state.UpdatedAt = now.Add(1 * time.Minute)
	if err := store.Save(context.Background(), state); err != nil {
		t.Fatalf("save updated state (idempotent migration path): %v", err)
	}

	got, err := store.Get(context.Background(), "tenant-a", "github")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got == nil {
		t.Fatal("expected state to exist")
	}
	if got.Status != connectors.StatusExpiring {
		t.Fatalf("expected status %q, got %q", connectors.StatusExpiring, got.Status)
	}

	items, err := store.List(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("list states: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one state, got %d", len(items))
	}

	if err := store.Delete(context.Background(), "tenant-a", "github"); err != nil {
		t.Fatalf("delete state: %v", err)
	}
	afterDelete, err := store.Get(context.Background(), "tenant-a", "github")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if afterDelete != nil {
		t.Fatal("expected no state after delete")
	}
}

func TestSQLConnectorStoreSaveIfUnchangedRejectsStaleSnapshot(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "connector-stale.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store := &sqlConnectorStore{db: db, dialect: "sqlite"}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC().Round(0)
	initial := &connectors.ConnectorState{
		TenantID:        "tenant-a",
		Name:            "github",
		Provider:        "github",
		Status:          connectors.StatusHealthy,
		CreatedAt:       now,
		UpdatedAt:       now,
		ConnectionScope: connectors.ScopeUser,
	}
	if err := store.Save(context.Background(), initial); err != nil {
		t.Fatalf("save initial state: %v", err)
	}

	fresh, err := store.Get(context.Background(), "tenant-a", "github")
	if err != nil {
		t.Fatalf("get fresh state: %v", err)
	}
	stale := *fresh

	fresh.Status = connectors.StatusExpiring
	fresh.UpdatedAt = now.Add(1 * time.Minute)
	saved, err := store.SaveIfUnchanged(context.Background(), &stale, fresh)
	if err != nil {
		t.Fatalf("save fresh snapshot: %v", err)
	}
	if !saved {
		t.Fatal("expected fresh snapshot save to succeed")
	}

	stale.Status = connectors.StatusReauthNeeded
	stale.UpdatedAt = now.Add(2 * time.Minute)
	saved, err = store.SaveIfUnchanged(context.Background(), &stale, &stale)
	if err != nil {
		t.Fatalf("save stale snapshot: %v", err)
	}
	if saved {
		t.Fatal("expected stale snapshot save to be rejected")
	}

	got, err := store.Get(context.Background(), "tenant-a", "github")
	if err != nil {
		t.Fatalf("get state after stale save: %v", err)
	}
	if got == nil || got.Status != connectors.StatusExpiring {
		t.Fatalf("expected latest status to remain expiring, got %+v", got)
	}
}

func TestSQLConnectorStoreSaveIfUnchangedInsertsOnlyOnce(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "connector-insert.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store := &sqlConnectorStore{db: db, dialect: "sqlite"}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC().Round(0)
	state := &connectors.ConnectorState{
		TenantID:        "tenant-a",
		Name:            "github",
		Provider:        "github",
		Status:          connectors.StatusHealthy,
		CreatedAt:       now,
		UpdatedAt:       now,
		ConnectionScope: connectors.ScopeUser,
	}

	saved, err := store.SaveIfUnchanged(context.Background(), nil, state)
	if err != nil {
		t.Fatalf("first conditional insert: %v", err)
	}
	if !saved {
		t.Fatal("expected first conditional insert to succeed")
	}

	saved, err = store.SaveIfUnchanged(context.Background(), nil, state)
	if err != nil {
		t.Fatalf("second conditional insert: %v", err)
	}
	if saved {
		t.Fatal("expected duplicate conditional insert to be rejected")
	}
}

func TestConnectorReadOnlyStoreSupportsSQLiteURIDSNOnMainAgain(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "connector.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS smoke(id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("seed sqlite file: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = "file:" + dbPath + "?cache=shared"

	connectorStore, err := openConnectorStoreReadOnly(cfg)
	if err != nil {
		t.Fatalf("open read-only connector store with sqlite URI dsn: %v", err)
	}
	closer, ok := connectorStore.(interface{ Close() error })
	if !ok {
		t.Fatalf("expected read-only connector store to implement Close")
	}
	t.Cleanup(func() {
		_ = closer.Close()
	})
}

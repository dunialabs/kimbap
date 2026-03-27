package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
)

var connectorMigrationDriverSeq uint64

type connectorMigrationDriver struct {
	execFn func(string) error
}

func (d *connectorMigrationDriver) Open(_ string) (driver.Conn, error) {
	return &connectorMigrationConn{execFn: d.execFn}, nil
}

type connectorMigrationConn struct {
	execFn func(string) error
}

func (c *connectorMigrationConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *connectorMigrationConn) Close() error {
	return nil
}

func (c *connectorMigrationConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (c *connectorMigrationConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if err := c.execFn(query); err != nil {
		return nil, err
	}
	return driver.RowsAffected(1), nil
}

func registerConnectorMigrationDriver(t *testing.T, execFn func(string) error) string {
	t.Helper()
	name := fmt.Sprintf("kimbap-connector-migrate-%d", atomic.AddUint64(&connectorMigrationDriverSeq, 1))
	sql.Register(name, &connectorMigrationDriver{execFn: execFn})
	return name
}

func TestMigrateConnectorTableReturnsIndexCreationError(t *testing.T) {
	sentinel := errors.New("index create failed")
	driverName := registerConnectorMigrationDriver(t, func(query string) error {
		if strings.Contains(query, "CREATE INDEX IF NOT EXISTS idx_connector_states_tenant_name") {
			return sentinel
		}
		return nil
	})

	db, err := sql.Open(driverName, "test")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	err = migrateConnectorTable(context.Background(), db, "sqlite")
	if err == nil {
		t.Fatal("expected migrate connector table to fail")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel error, got %v", err)
	}
	if !strings.Contains(err.Error(), "migrate connector table index") {
		t.Fatalf("expected index migration context in error, got %v", err)
	}
}

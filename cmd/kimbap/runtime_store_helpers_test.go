package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/store"
)

func TestWithRuntimeStoreWrapsOpenFailures(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Database.Driver = "unsupported-driver"

	err := withRuntimeStore(cfg, func(_ *store.SQLStore) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected openRuntimeStore failure")
	}
	if !isRuntimeStoreUnavailable(err) {
		t.Fatalf("expected runtime store unavailable classification, got %v", err)
	}
}

func TestWithRuntimeStorePreservesCallbackError(t *testing.T) {
	cfg := config.DefaultConfig()
	tmp := t.TempDir()
	cfg.DataDir = tmp
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = filepath.Join(tmp, "kimbap.db")

	sentinel := errors.New("sentinel callback failure")
	err := withRuntimeStore(cfg, func(_ *store.SQLStore) error {
		return sentinel
	})
	if err == nil {
		t.Fatal("expected callback failure")
	}
	if isRuntimeStoreUnavailable(err) {
		t.Fatalf("expected callback error to remain non-store-unavailable, got %v", err)
	}
	if !strings.Contains(err.Error(), "sentinel callback failure") {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestRunApproveAcceptPreservesDomainErrors(t *testing.T) {
	dataDir := t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfgRaw := "data_dir: " + dataDir + "\n" +
		"database:\n" +
		"  driver: sqlite\n" +
		"  dsn: " + filepath.Join(dataDir, "kimbap.db") + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgRaw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	prevOpts := opts
	opts = cliOptions{configPath: cfgPath, format: "json"}
	t.Cleanup(func() {
		opts = prevOpts
	})

	err := runApproveAccept("missing-request")
	if err == nil {
		t.Fatal("expected approve failure for missing request")
	}
	if isRuntimeStoreUnavailable(err) {
		t.Fatalf("expected domain failure, got runtime-store-unavailable: %v", err)
	}
	if !strings.Contains(err.Error(), "approve failed") {
		t.Fatalf("expected approve failure context, got %v", err)
	}
}

func TestRunApproveAcceptMaterializesExpiredApproval(t *testing.T) {
	approvalID, cfgPath := seedExpiredApprovalForCLI(t, time.Now().UTC().Add(-1*time.Minute))

	prevOpts := opts
	opts = cliOptions{configPath: cfgPath, format: "json"}
	t.Cleanup(func() {
		opts = prevOpts
	})

	err := runApproveAccept(approvalID)
	if err == nil {
		t.Fatal("expected approve failure for expired request")
	}
	if !strings.Contains(err.Error(), "approval has expired") {
		t.Fatalf("expected expired approval error, got %v", err)
	}

	rec := loadApprovalFromCLIStore(t, cfgPath, approvalID)
	if rec.Status != "expired" {
		t.Fatalf("expected expired status to be materialized, got %q", rec.Status)
	}
	if rec.ResolvedBy != "system" || rec.Reason != "auto-expired" {
		t.Fatalf("expected auto-expired resolution metadata, got resolved_by=%q reason=%q", rec.ResolvedBy, rec.Reason)
	}
	if rec.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set after auto-expiry")
	}
}

func TestApproveAcceptPrintsNextStep(t *testing.T) {
	approvalID, cfgPath := seedPendingApprovalForCLI(t, time.Now().UTC().Add(5*time.Minute))

	prevOpts := opts
	opts = cliOptions{configPath: cfgPath, format: "text"}
	t.Cleanup(func() {
		opts = prevOpts
	})

	output, err := captureStdout(t, func() error { return runApproveAccept(approvalID) })
	if err != nil {
		t.Fatalf("runApproveAccept failed: %v", err)
	}
	if !strings.Contains(output, "✓ "+approvalID+" approved") {
		t.Fatalf("expected approval success line, got %q", output)
	}
	if !strings.Contains(output, "Retry: kimbap call github.issues.create") {
		t.Fatalf("expected retry hint, got %q", output)
	}
	if strings.Contains(output, "Hint: Approval recorded.") {
		t.Fatalf("expected old hint removed, got %q", output)
	}
}

func TestApproveDenyMaterializesExpiredApproval(t *testing.T) {
	approvalID, cfgPath := seedExpiredApprovalForCLI(t, time.Now().UTC().Add(-1*time.Minute))

	prevOpts := opts
	opts = cliOptions{configPath: cfgPath, format: "json"}
	t.Cleanup(func() {
		opts = prevOpts
	})

	cmd := newApproveDenyCommand()
	cmd.SetArgs([]string{approvalID, "--reason", "too late"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected deny failure for expired request")
	}
	if !strings.Contains(err.Error(), "approval has expired") {
		t.Fatalf("expected expired approval error, got %v", err)
	}

	rec := loadApprovalFromCLIStore(t, cfgPath, approvalID)
	if rec.Status != "expired" {
		t.Fatalf("expected expired status to be materialized, got %q", rec.Status)
	}
	if rec.ResolvedBy != "system" || rec.Reason != "auto-expired" {
		t.Fatalf("expected auto-expired resolution metadata, got resolved_by=%q reason=%q", rec.ResolvedBy, rec.Reason)
	}
	if rec.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set after auto-expiry")
	}
}

func seedExpiredApprovalForCLI(t *testing.T, expiresAt time.Time) (string, string) {
	t.Helper()
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "kimbap.db")
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfgRaw := "data_dir: " + dataDir + "\n" +
		"database:\n" +
		"  driver: sqlite\n" +
		"  dsn: " + dbPath + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgRaw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	st, err := store.OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("open runtime store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate runtime store: %v", err)
	}

	req := &store.ApprovalRecord{
		ID:        "apr-expired-cli",
		TenantID:  defaultTenantID(),
		RequestID: "req-expired-cli",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		InputJSON: `{}`,
		CreatedAt: expiresAt.Add(-10 * time.Minute),
		ExpiresAt: expiresAt,
	}
	if err := st.CreateApproval(context.Background(), req); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	return req.ID, cfgPath
}

func seedPendingApprovalForCLI(t *testing.T, expiresAt time.Time) (string, string) {
	t.Helper()
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "kimbap.db")
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfgRaw := "data_dir: " + dataDir + "\n" +
		"database:\n" +
		"  driver: sqlite\n" +
		"  dsn: " + dbPath + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgRaw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	st, err := store.OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("open runtime store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate runtime store: %v", err)
	}

	req := &store.ApprovalRecord{
		ID:        "apr-pending-cli",
		TenantID:  defaultTenantID(),
		RequestID: "req-pending-cli",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		InputJSON: `{}`,
		CreatedAt: time.Now().UTC().Add(-1 * time.Minute),
		ExpiresAt: expiresAt,
	}
	if err := st.CreateApproval(context.Background(), req); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	return req.ID, cfgPath
}

func loadApprovalFromCLIStore(t *testing.T, cfgPath, approvalID string) *store.ApprovalRecord {
	t.Helper()
	cfg, err := config.LoadKimbapConfigWithoutDefault(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	st, err := openRuntimeStore(cfg)
	if err != nil {
		t.Fatalf("open runtime store: %v", err)
	}
	defer st.Close()
	rec, err := st.GetApproval(context.Background(), approvalID)
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	return rec
}

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

func TestRunApproveAcceptRequiresRequestIDWithHint(t *testing.T) {
	err := runApproveAccept("   ")
	if err == nil {
		t.Fatal("expected request-id required error")
	}
	if !strings.Contains(err.Error(), "request-id is required") {
		t.Fatalf("expected request-id required message, got %v", err)
	}
	if !strings.Contains(err.Error(), "Run: kimbap approve list --status pending") {
		t.Fatalf("expected actionable discovery hint, got %v", err)
	}
	if !strings.Contains(err.Error(), "Then run: kimbap approve accept <request-id>") {
		t.Fatalf("expected actionable next-step hint, got %v", err)
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
	if !strings.Contains(err.Error(), "approval") || !strings.Contains(err.Error(), "expired") {
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
	if !strings.Contains(err.Error(), "approval") || !strings.Contains(err.Error(), "expired") {
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

func TestApproveDenyRequiresReasonWithNextStepHint(t *testing.T) {
	cmd := newApproveDenyCommand()
	cmd.SetArgs([]string{"apr-123"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected deny command to require --reason")
	}
	if !strings.Contains(err.Error(), "--reason is required") {
		t.Fatalf("expected required reason error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Run: kimbap approve list --status pending") {
		t.Fatalf("expected discovery hint, got %v", err)
	}
	if !strings.Contains(err.Error(), "Then run: kimbap approve deny apr-123 --reason \"<why>\"") {
		t.Fatalf("expected actionable next-step hint, got %v", err)
	}
}

func TestRuntimeStoreSQLiteURIDoesNotCreateSchemeDirectoryOnMainAgain(t *testing.T) {
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir work dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	dbPath := filepath.Join(t.TempDir(), "db", "kimbap.db")
	cfg := config.DefaultConfig()
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = "file:" + dbPath + "?cache=shared"

	st, err := openRuntimeStore(cfg)
	if err != nil {
		t.Fatalf("open runtime store with sqlite URI dsn: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	if _, statErr := os.Stat(filepath.Join(workDir, "file:")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no scheme directory in cwd, got stat err=%v", statErr)
	}
}

func TestApprovalStatusValidation(t *testing.T) {
	tests := []struct {
		name            string
		in              string
		want            string
		wantErr         bool
		wantErrContains string
	}{
		{name: "pending", in: "pending", want: "pending"},
		{name: "uppercase normalized", in: "APPROVED", want: "approved"},
		{name: "trimmed", in: " denied ", want: "denied"},
		{name: "blank invalid", in: "   ", wantErr: true, wantErrContains: "--status cannot be blank"},
		{name: "invalid", in: "in_review", wantErr: true, wantErrContains: "invalid --status"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := approvalStatus(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.in)
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErrContains, err)
				}
				if !strings.Contains(err.Error(), "valid: pending, approved, denied, expired") {
					t.Fatalf("expected allowed-values hint in error, got %v", err)
				}
				if tc.name == "blank invalid" {
					if !strings.Contains(err.Error(), "Run: kimbap approve list") {
						t.Fatalf("expected blank-status default command hint, got %v", err)
					}
					if !strings.Contains(err.Error(), "Or:  kimbap approve list --status pending") {
						t.Fatalf("expected blank-status pending command hint, got %v", err)
					}
				} else if !strings.Contains(err.Error(), "Run: kimbap approve list --status pending") {
					t.Fatalf("expected actionable retry hint, got %v", err)
				} else if !strings.Contains(err.Error(), "Example: kimbap approve list --status approved") {
					t.Fatalf("expected copyable alternate-status example, got %v", err)
				} else if !strings.Contains(err.Error(), "Try one of: approved, denied, expired") {
					t.Fatalf("expected invalid-status options hint, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestApprovalNoRequestsMessage(t *testing.T) {
	pending := approvalNoRequestsMessage("pending")
	if !strings.Contains(pending, "No pending approval requests.") {
		t.Fatalf("unexpected pending message: %q", pending)
	}
	if !strings.Contains(pending, "Tip: Run kimbap approve list --status approved") {
		t.Fatalf("expected approved status tip, got %q", pending)
	}
	if !strings.Contains(pending, "Tip: Run kimbap approve list --status denied") {
		t.Fatalf("expected denied status tip, got %q", pending)
	}
	got := approvalNoRequestsMessage("approved")
	if !strings.Contains(got, `No approval requests found for status "approved".`) {
		t.Fatalf("expected status-specific message, got %q", got)
	}
	if !strings.Contains(got, "Tip: Run kimbap approve list --status pending to review pending decisions.") {
		t.Fatalf("expected actionable tip, got %q", got)
	}
}

func TestApprovalTimeRemainingLongDurationUsesDays(t *testing.T) {
	now := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	got := approvalTimeRemainingAt(now.Add(49*time.Hour), now)
	if got != "2d1h" {
		t.Fatalf("expected exact day+hour format, got %q", got)
	}
}

func TestApprovalTimeRemainingNearDayBoundaryUsesDayFormat(t *testing.T) {
	now := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	got := approvalTimeRemainingAt(now.Add(24*time.Hour-time.Second), now)
	if got != "1d0h" {
		t.Fatalf("expected boundary to round up to day format, got %q", got)
	}
}

func TestApprovalTimeRemainingLongDurationDoesNotOverstateHours(t *testing.T) {
	now := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	got := approvalTimeRemainingAt(now.Add(25*time.Hour+time.Second), now)
	if got != "1d1h" {
		t.Fatalf("expected floor-hour day format for long duration, got %q", got)
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

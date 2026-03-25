package store

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSQLiteStoreTokenCRUDLifecycle(t *testing.T) {
	st := newTestSQLiteStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	rec := &TokenRecord{
		ID:          "st_test_1",
		TenantID:    "tenant-a",
		AgentName:   "agent-a",
		TokenHash:   "hash-1",
		DisplayHint: "abcd",
		Scopes:      `["tools:read"]`,
		CreatedAt:   now,
		ExpiresAt:   now.Add(time.Hour),
		CreatedBy:   "tester",
	}
	if err := st.CreateToken(ctx, rec); err != nil {
		t.Fatalf("create token: %v", err)
	}

	got, err := st.GetToken(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if got.TenantID != "tenant-a" {
		t.Fatalf("unexpected tenant id: %s", got.TenantID)
	}

	byHash, err := st.GetTokenByHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("get token by hash: %v", err)
	}
	if byHash.ID != rec.ID {
		t.Fatalf("unexpected token by hash id: %s", byHash.ID)
	}

	if err := st.UpdateTokenLastUsed(ctx, rec.ID); err != nil {
		t.Fatalf("update token last used: %v", err)
	}
	updated, err := st.GetToken(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get updated token: %v", err)
	}
	if updated.LastUsedAt == nil {
		t.Fatal("expected last_used_at to be set")
	}

	if err := st.RevokeToken(ctx, rec.ID); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	revoked, err := st.GetToken(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get revoked token: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("expected revoked_at to be set")
	}

	list, err := st.ListTokens(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 token, got %d", len(list))
	}
}

func TestSQLiteStoreAuditWriteAndQueryWithFilters(t *testing.T) {
	st := newTestSQLiteStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	events := []*AuditRecord{
		{ID: "a1", Timestamp: now.Add(-2 * time.Minute), TenantID: "tenant-a", AgentName: "agent-a", Service: "github", Action: "issues.list", Status: "success", RequestID: "r1", TraceID: "t1", PrincipalID: "p1", Mode: "call", PolicyDecision: "allow", InputJSON: `{}`, MetaJSON: `{}`},
		{ID: "a2", Timestamp: now.Add(-1 * time.Minute), TenantID: "tenant-a", AgentName: "agent-b", Service: "github", Action: "issues.create", Status: "error", RequestID: "r2", TraceID: "t2", PrincipalID: "p2", Mode: "call", PolicyDecision: "deny", InputJSON: `{}`, MetaJSON: `{"reason":"x"}`},
		{ID: "a3", Timestamp: now, TenantID: "tenant-b", AgentName: "agent-a", Service: "slack", Action: "chat.post", Status: "success", RequestID: "r3", TraceID: "t3", PrincipalID: "p3", Mode: "call", PolicyDecision: "allow", InputJSON: `{}`, MetaJSON: `{}`},
	}
	for _, ev := range events {
		if err := st.WriteAuditEvent(ctx, ev); err != nil {
			t.Fatalf("write audit event: %v", err)
		}
	}

	got, err := st.QueryAuditEvents(ctx, AuditFilter{TenantID: "tenant-a", Service: "github", Status: "error"})
	if err != nil {
		t.Fatalf("query audit events: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].ID != "a2" {
		t.Fatalf("unexpected event id: %s", got[0].ID)
	}

	var out bytes.Buffer
	if err := st.ExportAuditEvents(ctx, AuditFilter{TenantID: "tenant-a"}, "csv", &out); err != nil {
		t.Fatalf("export csv: %v", err)
	}
	if !strings.Contains(out.String(), "tenant-a") {
		t.Fatalf("export missing expected tenant: %s", out.String())
	}
}

func TestSQLiteStoreAuditQueryOffsetWithoutLimit(t *testing.T) {
	st := newTestSQLiteStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	for i, id := range []string{"a1", "a2", "a3"} {
		if err := st.WriteAuditEvent(ctx, &AuditRecord{
			ID:             id,
			Timestamp:      now.Add(time.Duration(i) * time.Minute),
			TenantID:       "tenant-a",
			AgentName:      "agent-a",
			Service:        "github",
			Action:         "issues.list",
			Status:         "success",
			RequestID:      "r" + id,
			TraceID:        "t" + id,
			PrincipalID:    "p" + id,
			Mode:           "call",
			PolicyDecision: "allow",
			InputJSON:      `{}`,
			MetaJSON:       `{}`,
		}); err != nil {
			t.Fatalf("write audit event %s: %v", id, err)
		}
	}

	got, err := st.QueryAuditEvents(ctx, AuditFilter{TenantID: "tenant-a", Offset: 1})
	if err != nil {
		t.Fatalf("query audit events with offset only: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results after offset, got %d", len(got))
	}
	if got[0].ID != "a2" || got[1].ID != "a1" {
		t.Fatalf("unexpected offset order: %+v", got)
	}
}

func TestSQLiteStoreApprovalCreateAndUpdateStatus(t *testing.T) {
	st := newTestSQLiteStore(t)
	ctx := context.Background()

	req := &ApprovalRecord{
		ID:        "apr-1",
		TenantID:  "tenant-a",
		RequestID: "req-1",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		InputJSON: `{"title":"hello"}`,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := st.CreateApproval(ctx, req); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	if err := st.UpdateApprovalStatus(ctx, "apr-1", "approved", "operator-1", "ok"); err != nil {
		t.Fatalf("update approval status: %v", err)
	}

	got, err := st.GetApproval(ctx, "apr-1")
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if got.Status != "approved" {
		t.Fatalf("unexpected status: %s", got.Status)
	}
	if got.ResolvedAt == nil {
		t.Fatal("expected resolved_at")
	}

	list, err := st.ListApprovals(ctx, "tenant-a", "approved")
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(list))
	}
}

func TestSQLiteStoreUpdateApprovalStatusAlreadyResolvedReturnsTypedError(t *testing.T) {
	st := newTestSQLiteStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	req := &ApprovalRecord{
		ID:        "apr-cas-resolved",
		TenantID:  "tenant-a",
		RequestID: "req-cas-resolved",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		InputJSON: `{}`,
		CreatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute),
	}
	if err := st.CreateApproval(ctx, req); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	if err := st.UpdateApprovalStatus(ctx, req.ID, "approved", "operator-1", "ok"); err != nil {
		t.Fatalf("first update approval status: %v", err)
	}
	if err := st.UpdateApprovalStatus(ctx, req.ID, "denied", "operator-2", "late deny"); !errors.Is(err, ErrApprovalAlreadyResolved) {
		t.Fatalf("expected ErrApprovalAlreadyResolved, got %v", err)
	}
}

func TestSQLiteStoreUpdateApprovalStatusExpiredReturnsTypedError(t *testing.T) {
	st := newTestSQLiteStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	req := &ApprovalRecord{
		ID:        "apr-cas-expired",
		TenantID:  "tenant-a",
		RequestID: "req-cas-expired",
		AgentName: "agent-a",
		Service:   "github",
		Action:    "issues.create",
		Status:    "pending",
		InputJSON: `{}`,
		CreatedAt: now.Add(-30 * time.Minute),
		ExpiresAt: now.Add(-1 * time.Minute),
	}
	if err := st.CreateApproval(ctx, req); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	if err := st.UpdateApprovalStatus(ctx, req.ID, "approved", "operator-1", "too late"); !errors.Is(err, ErrApprovalExpired) {
		t.Fatalf("expected ErrApprovalExpired, got %v", err)
	}
}

func TestSQLiteStorePolicySetAndGet(t *testing.T) {
	st := newTestSQLiteStore(t)
	ctx := context.Background()

	doc := []byte("version: 1.0.0\nrules:\n  - id: allow-all\n")
	if err := st.SetPolicy(ctx, "tenant-a", doc); err != nil {
		t.Fatalf("set policy: %v", err)
	}

	got, err := st.GetPolicy(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if !bytes.Equal(got, doc) {
		t.Fatalf("unexpected policy payload")
	}
}

func TestSQLiteStoreTenantIsolation(t *testing.T) {
	st := newTestSQLiteStore(t)
	ctx := context.Background()

	if err := st.CreateToken(ctx, &TokenRecord{ID: "st-a", TenantID: "tenant-a", AgentName: "agent", TokenHash: "hash-a", DisplayHint: "aaaa", Scopes: "[]", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedBy: "tester"}); err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := st.CreateToken(ctx, &TokenRecord{ID: "st-b", TenantID: "tenant-b", AgentName: "agent", TokenHash: "hash-b", DisplayHint: "bbbb", Scopes: "[]", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedBy: "tester"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	listA, err := st.ListTokens(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("list tenant-a: %v", err)
	}
	if len(listA) != 1 || listA[0].TenantID != "tenant-a" {
		t.Fatalf("tenant isolation failed for tenant-a: %+v", listA)
	}

	listB, err := st.ListTokens(ctx, "tenant-b")
	if err != nil {
		t.Fatalf("list tenant-b: %v", err)
	}
	if len(listB) != 1 || listB[0].TenantID != "tenant-b" {
		t.Fatalf("tenant isolation failed for tenant-b: %+v", listB)
	}

	_, err = st.GetPolicy(ctx, "tenant-z")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing tenant policy, got %v", err)
	}
}

func newTestSQLiteStore(t *testing.T) *SQLStore {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "store.sqlite")
	st, err := OpenSQLiteStore(dsn)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return st
}

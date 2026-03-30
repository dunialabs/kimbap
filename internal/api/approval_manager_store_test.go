package api

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/store"
)

func TestApprovalManagerStoreAdapterApproveFailsOnCorruptedVotesJSON(t *testing.T) {
	st, err := store.OpenSQLiteStore(filepath.Join(t.TempDir(), "api-approval-corrupt-votes.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}

	now := time.Now().UTC()
	if err := st.CreateApproval(context.Background(), &store.ApprovalRecord{
		ID:                "apr_bad_votes",
		TenantID:          "tenant-a",
		RequestID:         "req_bad_votes",
		AgentName:         "agent-a",
		Service:           "github",
		Action:            "issues.create",
		Status:            "pending",
		InputJSON:         "{}",
		RequiredApprovals: 1,
		VotesJSON:         "{",
		CreatedAt:         now,
		ExpiresAt:         now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create approval: %v", err)
	}

	adapter := &approvalManagerStoreAdapter{base: st, updater: st}
	manager := approvals.NewApprovalManager(adapter, nil, time.Minute)
	if err := manager.Approve(context.Background(), "apr_bad_votes", "approver-1"); err == nil {
		t.Fatal("expected corrupted votes JSON to prevent approval")
	} else if !strings.Contains(err.Error(), "parse approval votes") {
		t.Fatalf("expected parse approval votes error, got %v", err)
	}
}

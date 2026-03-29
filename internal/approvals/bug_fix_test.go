package approvals

import (
	"context"
	"testing"
	"time"
)

func TestSubmitClearsPrepopulatedVotes(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{
		TenantID:          "tenant-a",
		RequiredApprovals: 2,
		Votes: []ApprovalVote{{
			ApproverID: "seeded-approver",
			Decision:   StatusApproved,
			VotedAt:    time.Now().UTC(),
		}},
	}

	if err := manager.Submit(ctx, req); err != nil {
		t.Fatalf("submit approval request: %v", err)
	}

	stored, err := store.Get(ctx, req.ID)
	if err != nil {
		t.Fatalf("get stored approval request: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored approval request")
	}
	if len(stored.Votes) != 0 {
		t.Fatalf("expected submit to clear seeded votes, got %d", len(stored.Votes))
	}
}

func TestMemoryApprovalStorePreservesInputTypesAndIsolation(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryApprovalStore()
	when := time.Now().UTC().Round(0)
	nested := map[string]any{"safe": true}
	tags := []string{"alpha", "beta"}
	blob := []byte("abc")

	req := &ApprovalRequest{
		ID:       "ap-typed",
		TenantID: "tenant-a",
		Input: map[string]any{
			"count":  3,
			"when":   when,
			"nested": nested,
			"tags":   tags,
			"blob":   blob,
		},
	}
	if err := store.Create(ctx, req); err != nil {
		t.Fatalf("create approval request: %v", err)
	}

	nested["safe"] = false
	tags[0] = "changed"
	blob[0] = 'z'

	stored, err := store.Get(ctx, req.ID)
	if err != nil {
		t.Fatalf("get approval request: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored approval request")
	}

	count, ok := stored.Input["count"].(int)
	if !ok || count != 3 {
		t.Fatalf("expected count to stay an int with value 3, got %#v", stored.Input["count"])
	}
	storedWhen, ok := stored.Input["when"].(time.Time)
	if !ok || !storedWhen.Equal(when) {
		t.Fatalf("expected when to stay a time.Time with value %v, got %#v", when, stored.Input["when"])
	}
	storedNested, ok := stored.Input["nested"].(map[string]any)
	if !ok || storedNested["safe"] != true {
		t.Fatalf("expected nested map to be copied, got %#v", stored.Input["nested"])
	}
	storedTags, ok := stored.Input["tags"].([]string)
	if !ok || len(storedTags) != 2 || storedTags[0] != "alpha" {
		t.Fatalf("expected tags slice to be copied, got %#v", stored.Input["tags"])
	}
	storedBlob, ok := stored.Input["blob"].([]byte)
	if !ok || string(storedBlob) != "abc" {
		t.Fatalf("expected blob slice to be copied, got %#v", stored.Input["blob"])
	}
}

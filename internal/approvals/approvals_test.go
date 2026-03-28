package approvals

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSubmitCreatesPendingApproval(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	notifier := &captureNotifier{}
	manager := NewApprovalManager(store, notifier, 5*time.Minute)

	req := &ApprovalRequest{
		TenantID:  "tenant-a",
		RequestID: "req-1",
		AgentName: "assistant",
		Service:   "github",
		Action:    "issues.create",
		Input:     map[string]any{"title": "hello"},
	}

	if err := manager.Submit(ctx, req); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if req.ID == "" {
		t.Fatal("expected generated id")
	}
	if req.Status != StatusPending {
		t.Fatalf("expected pending, got %s", req.Status)
	}
	if notifier.last == nil || notifier.last.ID != req.ID {
		t.Fatalf("expected notifier to receive request, got %+v", notifier.last)
	}
}

func TestApproveChangesStatusAndApprover(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{ID: "ap-1", TenantID: "tenant-a", Status: StatusPending, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
	if err := store.Create(ctx, req); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := manager.Approve(ctx, "ap-1", "operator-1"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	updated, _ := store.Get(ctx, "ap-1")
	if updated.Status != StatusApproved {
		t.Fatalf("expected approved, got %s", updated.Status)
	}
	if updated.ResolvedBy != "operator-1" {
		t.Fatalf("unexpected approver: %s", updated.ResolvedBy)
	}
	if updated.ResolvedAt == nil {
		t.Fatal("expected resolved_at")
	}
}

func TestDenyChangesStatusWithReason(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{ID: "ap-2", TenantID: "tenant-a", Status: StatusPending, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
	_ = store.Create(ctx, req)

	if err := manager.Deny(ctx, "ap-2", "operator-2", "insufficient context"); err != nil {
		t.Fatalf("deny: %v", err)
	}

	updated, _ := store.Get(ctx, "ap-2")
	if updated.Status != StatusDenied {
		t.Fatalf("expected denied, got %s", updated.Status)
	}
	if updated.DenyReason != "insufficient context" {
		t.Fatalf("unexpected deny reason: %s", updated.DenyReason)
	}
}

func TestDoubleApproveReturnsError(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{ID: "ap-3", TenantID: "tenant-a", Status: StatusPending, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
	_ = store.Create(ctx, req)

	if err := manager.Approve(ctx, "ap-3", "operator-1"); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	if err := manager.Approve(ctx, "ap-3", "operator-2"); err == nil {
		t.Fatal("expected second approve to fail")
	}
}

func TestExpiredApprovalCannotBeApproved(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{ID: "ap-4", TenantID: "tenant-a", Status: StatusPending, CreatedAt: time.Now().Add(-10 * time.Minute), ExpiresAt: time.Now().Add(-time.Second)}
	_ = store.Create(ctx, req)

	if err := manager.Approve(ctx, "ap-4", "operator-1"); err == nil {
		t.Fatal("expected expired approval to fail")
	}
}

func TestExpireStaleMarksOldPendingAsExpired(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	_ = store.Create(ctx, &ApprovalRequest{ID: "ap-5", TenantID: "tenant-a", Status: StatusPending, CreatedAt: time.Now().Add(-10 * time.Minute), ExpiresAt: time.Now().Add(-time.Minute)})
	_ = store.Create(ctx, &ApprovalRequest{ID: "ap-6", TenantID: "tenant-a", Status: StatusPending, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})

	count, err := manager.ExpireStale(ctx)
	if err != nil {
		t.Fatalf("expire stale: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 expired request, got %d", count)
	}

	stale, _ := store.Get(ctx, "ap-5")
	if stale.Status != StatusExpired {
		t.Fatalf("expected expired status, got %s", stale.Status)
	}
}

func TestMultiApproverRequiresTwoVotes(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{
		ID:                "ap-multi",
		TenantID:          "tenant-a",
		Status:            StatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(time.Minute),
		RequiredApprovals: 2,
	}
	_ = store.Create(ctx, req)

	if err := manager.Approve(ctx, "ap-multi", "operator-1"); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	after1, _ := store.Get(ctx, "ap-multi")
	if after1.Status != StatusPending {
		t.Fatalf("expected still pending after 1 vote, got %s", after1.Status)
	}
	if len(after1.Votes) != 1 {
		t.Fatalf("expected 1 vote, got %d", len(after1.Votes))
	}

	if err := manager.Approve(ctx, "ap-multi", "operator-2"); err != nil {
		t.Fatalf("second approve: %v", err)
	}
	after2, _ := store.Get(ctx, "ap-multi")
	if after2.Status != StatusApproved {
		t.Fatalf("expected approved after 2 votes, got %s", after2.Status)
	}
	if len(after2.Votes) != 2 {
		t.Fatalf("expected 2 votes, got %d", len(after2.Votes))
	}
}

func TestMultiApproverDuplicateVoteRejected(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{
		ID:                "ap-dup-vote",
		TenantID:          "tenant-a",
		Status:            StatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(time.Minute),
		RequiredApprovals: 2,
	}
	_ = store.Create(ctx, req)

	if err := manager.Approve(ctx, "ap-dup-vote", "operator-1"); err != nil {
		t.Fatalf("first vote: %v", err)
	}
	if err := manager.Approve(ctx, "ap-dup-vote", "operator-1"); err == nil {
		t.Fatal("expected duplicate vote to be rejected")
	}
}

func TestApproveRejectsEmptyApproverID(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{
		ID:        "ap-empty-approver",
		TenantID:  "tenant-a",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	_ = store.Create(ctx, req)

	if err := manager.Approve(ctx, "ap-empty-approver", "   "); err == nil {
		t.Fatal("expected empty approver ID to be rejected")
	}
}

func TestDenyRejectsEmptyApproverID(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{
		ID:        "ap-empty-denier",
		TenantID:  "tenant-a",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	_ = store.Create(ctx, req)

	if err := manager.Deny(ctx, "ap-empty-denier", "\t\n", "reason"); err == nil {
		t.Fatal("expected empty denied_by to be rejected")
	}
}

func TestDuplicateVoteRejectsWhitespaceVariantApproverID(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{
		ID:                "ap-whitespace-dup",
		TenantID:          "tenant-a",
		Status:            StatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(time.Minute),
		RequiredApprovals: 2,
	}
	_ = store.Create(ctx, req)

	if err := manager.Approve(ctx, "ap-whitespace-dup", "operator-1"); err != nil {
		t.Fatalf("first vote: %v", err)
	}
	if err := manager.Approve(ctx, "ap-whitespace-dup", "  operator-1  "); err == nil {
		t.Fatal("expected whitespace-variant duplicate vote to be rejected")
	}
}

func TestMultiApproverDenyImmediatelyResolves(t *testing.T) {
	ctx := context.Background()
	store := newMemApprovalStore()
	manager := NewApprovalManager(store, nil, time.Minute)

	req := &ApprovalRequest{
		ID:                "ap-deny-multi",
		TenantID:          "tenant-a",
		Status:            StatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(time.Minute),
		RequiredApprovals: 3,
	}
	_ = store.Create(ctx, req)

	if err := manager.Approve(ctx, "ap-deny-multi", "operator-1"); err != nil {
		t.Fatalf("first approve: %v", err)
	}

	if err := manager.Deny(ctx, "ap-deny-multi", "operator-2", "too risky"); err != nil {
		t.Fatalf("deny: %v", err)
	}
	denied, _ := store.Get(ctx, "ap-deny-multi")
	if denied.Status != StatusDenied {
		t.Fatalf("expected denied, got %s", denied.Status)
	}
	if len(denied.Votes) != 2 {
		t.Fatalf("expected 2 votes (1 approve + 1 deny), got %d", len(denied.Votes))
	}
	if denied.DenyReason != "too risky" {
		t.Fatalf("unexpected deny reason: %s", denied.DenyReason)
	}
}

func TestWebhookNotifierSendsPostWithSignature(t *testing.T) {
	signKey := []byte("super-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req ApprovalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		raw, _ := json.Marshal(&req)
		h := hmac.New(sha256.New, signKey)
		_, _ = h.Write(raw)
		expected := "sha256=" + hex.EncodeToString(h.Sum(nil))

		if got := r.Header.Get("X-Kimbap-Signature"); got != expected {
			t.Fatalf("unexpected signature: %s", got)
		}
		if req.ID != "ap-webhook" {
			t.Fatalf("unexpected payload id: %s", req.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, signKey)
	err := notifier.Notify(context.Background(), &ApprovalRequest{
		ID:       "ap-webhook",
		TenantID: "tenant-a",
		Status:   StatusPending,
	})
	if err != nil {
		t.Fatalf("notify webhook: %v", err)
	}
}

type captureNotifier struct {
	last *ApprovalRequest
}

func (n *captureNotifier) Notify(_ context.Context, req *ApprovalRequest) error {
	copyReq := *req
	n.last = &copyReq
	return nil
}

type memApprovalStore struct {
	mu    sync.RWMutex
	items map[string]ApprovalRequest
}

func newMemApprovalStore() *memApprovalStore {
	return &memApprovalStore{items: map[string]ApprovalRequest{}}
}

func (s *memApprovalStore) Create(_ context.Context, req *ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyReq := *req
	s.items[req.ID] = copyReq
	return nil
}

func (s *memApprovalStore) Get(_ context.Context, id string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyReq := req
	return &copyReq, nil
}

func (s *memApprovalStore) Update(_ context.Context, req *ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyReq := *req
	s.items[req.ID] = copyReq
	return nil
}

func (s *memApprovalStore) ListPending(_ context.Context, tenantID string) ([]ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]ApprovalRequest, 0)
	for _, item := range s.items {
		if item.TenantID == tenantID && item.Status == StatusPending {
			list = append(list, item)
		}
	}
	return list, nil
}

func (s *memApprovalStore) ListAll(_ context.Context, tenantID string, filter ApprovalFilter) ([]ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]ApprovalRequest, 0)
	for _, item := range s.items {
		if item.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && item.Status != *filter.Status {
			continue
		}
		if filter.AgentName != "" && item.AgentName != filter.AgentName {
			continue
		}
		if filter.Service != "" && item.Service != filter.Service {
			continue
		}
		list = append(list, item)
		if filter.Limit > 0 && len(list) >= filter.Limit {
			break
		}
	}
	return list, nil
}

func (s *memApprovalStore) ExpireOld(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for id, item := range s.items {
		if item.Status == StatusPending && !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			item.Status = StatusExpired
			s.items[id] = item
			count++
		}
	}
	return count, nil
}

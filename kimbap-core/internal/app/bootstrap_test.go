package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	"github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/dunialabs/kimbap-core/internal/security"
	"github.com/dunialabs/kimbap-core/internal/skills"
)

func TestBuildRuntimeNilConfig(t *testing.T) {
	rt, err := BuildRuntime(RuntimeDeps{})
	if err == nil {
		t.Fatalf("expected error for nil config, got runtime=%v", rt)
	}
}

func TestBuildRuntimeMinimalConfig(t *testing.T) {
	cfg := &config.KimbapConfig{
		Skills: config.SkillsConfig{Dir: t.TempDir()},
		Policy: config.PolicyConfig{Path: ""},
	}

	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	if rt == nil {
		t.Fatal("expected runtime to be non-nil")
	}
}

func TestBuildRuntimeWithSkills(t *testing.T) {
	skillsDir := t.TempDir()
	const manifest = `name: test-skill
version: 1.0.0
description: test skill
base_url: https://api.example.com
auth:
  type: header
  header_name: Authorization
  credential_ref: test.token
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
      mutating: false
`
	if err := os.WriteFile(filepath.Join(skillsDir, "test-skill.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write skill manifest: %v", err)
	}

	cfg := &config.KimbapConfig{Skills: config.SkillsConfig{Dir: skillsDir}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	actions, err := rt.ActionRegistry.List(context.Background(), runtime.ListOptions{})
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action from loaded skill, got %d", len(actions))
	}
	if actions[0].Name != "test-skill.ping" {
		t.Fatalf("unexpected action name: %s", actions[0].Name)
	}
}

func TestBuildRuntimeWithSkillsStrictVerifyFailsWithoutLock(t *testing.T) {
	skillsDir := t.TempDir()
	const manifest = `name: test-skill
version: 1.0.0
description: test skill
base_url: https://api.example.com
auth:
  type: header
  header_name: Authorization
  credential_ref: test.token
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
      mutating: false
`
	if err := os.WriteFile(filepath.Join(skillsDir, "test-skill.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write skill manifest: %v", err)
	}

	cfg := &config.KimbapConfig{Skills: config.SkillsConfig{Dir: skillsDir, Verify: "strict"}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	_, listErr := rt.ActionRegistry.List(context.Background(), runtime.ListOptions{})
	if listErr == nil {
		t.Fatalf("expected strict verification failure for unlocked skill")
	}
}

func TestBuildRuntimeWithSkillsStrictVerifyPassesWithLock(t *testing.T) {
	skillsDir := t.TempDir()
	manifest, err := skills.ParseManifest([]byte(`name: test-skill
version: 1.0.0
description: test skill
base_url: https://api.example.com
auth:
  type: header
  header_name: Authorization
  credential_ref: test.token
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
      mutating: false
`))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	installer := skills.NewLocalInstaller(skillsDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install skill with lockfile: %v", err)
	}

	cfg := &config.KimbapConfig{Skills: config.SkillsConfig{Dir: skillsDir, Verify: "strict"}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	actionsList, listErr := rt.ActionRegistry.List(context.Background(), runtime.ListOptions{})
	if listErr != nil {
		t.Fatalf("list actions: %v", listErr)
	}
	if len(actionsList) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actionsList))
	}
}

func TestBuildRuntimeWithRequiredSignatureSkipsUnsignedUnlockedSkill(t *testing.T) {
	skillsDir := t.TempDir()
	const manifest = `name: test-skill
version: 1.0.0
description: test skill
base_url: https://api.example.com
auth:
  type: header
  header_name: Authorization
  credential_ref: test.token
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
      mutating: false
`
	if err := os.WriteFile(filepath.Join(skillsDir, "test-skill.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write skill manifest: %v", err)
	}

	cfg := &config.KimbapConfig{Skills: config.SkillsConfig{Dir: skillsDir, Verify: "warn", SignaturePolicy: "required"}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	actionsList, listErr := rt.ActionRegistry.List(context.Background(), runtime.ListOptions{})
	if listErr != nil {
		t.Fatalf("list actions: %v", listErr)
	}
	if len(actionsList) != 0 {
		t.Fatalf("expected unsigned unlocked skill to be skipped under required signature policy, got %d actions", len(actionsList))
	}
}

func TestConnectorCredentialResolverRequiresOAuthSuffix(t *testing.T) {
	ctx := context.Background()
	store := newBootstrapMemConnectorStore()
	mgr := connectors.NewManager(store)
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", "connector-test-key")

	mgr.RegisterConfig(connectors.ConnectorConfig{
		Name:      "github",
		Provider:  "github",
		ClientID:  "cid",
		TokenURL:  "https://example.com/token",
		DeviceURL: "https://example.com/device",
	})

	if err := mgr.StoreConnection(ctx, "tenant-1", "github", "github", "token-123", "", 3600, "", connectors.FlowBrowser, connectors.ScopeUser, ""); err != nil {
		t.Fatalf("store connection: %v", err)
	}

	resolver := &connectorCredentialResolver{mgr: mgr}

	plainRef, err := resolver.Resolve(ctx, "tenant-1", actions.AuthRequirement{Type: actions.AuthTypeOAuth2, CredentialRef: "github"})
	if err != nil {
		t.Fatalf("resolve plain ref: %v", err)
	}
	if plainRef != nil {
		t.Fatalf("expected nil for plain ref without oauth suffix, got %+v", plainRef)
	}

	suffixRef, err := resolver.Resolve(ctx, "tenant-1", actions.AuthRequirement{Type: actions.AuthTypeOAuth2, CredentialRef: "github.token"})
	if err != nil {
		t.Fatalf("resolve suffix ref: %v", err)
	}
	if suffixRef == nil || suffixRef.Token != "token-123" {
		t.Fatalf("expected token from suffix ref, got %+v", suffixRef)
	}
}

func TestConnectorCredentialResolverSupportsProfiledConnectorRef(t *testing.T) {
	ctx := context.Background()
	store := newBootstrapMemConnectorStore()
	mgr := connectors.NewManager(store)
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", "connector-test-key")

	mgr.RegisterConfig(connectors.ConnectorConfig{
		Name:      "github:work",
		Provider:  "github",
		ClientID:  "cid",
		TokenURL:  "https://example.com/token",
		DeviceURL: "https://example.com/device",
	})

	if err := mgr.StoreConnection(ctx, "tenant-1", "github:work", "github", "work-token", "", 3600, "", connectors.FlowBrowser, connectors.ScopeUser, ""); err != nil {
		t.Fatalf("store connection: %v", err)
	}

	resolver := &connectorCredentialResolver{mgr: mgr}
	resolved, err := resolver.Resolve(ctx, "tenant-1", actions.AuthRequirement{Type: actions.AuthTypeBearer, CredentialRef: "github:work.token"})
	if err != nil {
		t.Fatalf("resolve profiled ref: %v", err)
	}
	if resolved == nil || resolved.Token != "work-token" {
		t.Fatalf("expected profiled token, got %+v", resolved)
	}
}

func TestApprovalManagerAdapterSplitsServiceAndAction(t *testing.T) {
	store := &captureApprovalStore{}
	mgr := approvals.NewApprovalManager(store, nil, time.Minute)
	adapter := NewApprovalManagerAdapter(mgr)

	_, err := adapter.CreateRequest(context.Background(), runtime.ApprovalRequest{
		TenantID:  "tenant-a",
		RequestID: "req-1",
		Principal: actions.Principal{AgentName: "agent-a"},
		Action:    actions.ActionDefinition{Name: "github.issues.create"},
	})
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}

	if store.last == nil {
		t.Fatal("expected approval request to be persisted")
	}
	if store.last.Service != "github" {
		t.Fatalf("expected service github, got %q", store.last.Service)
	}
	if store.last.Action != "issues.create" {
		t.Fatalf("expected action issues.create, got %q", store.last.Action)
	}
}

type bootstrapMemConnectorStore struct {
	items map[string]connectors.ConnectorState
}

func newBootstrapMemConnectorStore() *bootstrapMemConnectorStore {
	return &bootstrapMemConnectorStore{items: map[string]connectors.ConnectorState{}}
}

func (s *bootstrapMemConnectorStore) Save(_ context.Context, state *connectors.ConnectorState) error {
	copyState := *state
	s.items[s.key(state.TenantID, state.Name)] = copyState
	return nil
}

func (s *bootstrapMemConnectorStore) Get(_ context.Context, tenantID, name string) (*connectors.ConnectorState, error) {
	item, ok := s.items[s.key(tenantID, name)]
	if !ok {
		return nil, nil
	}
	copyState := item
	return &copyState, nil
}

func (s *bootstrapMemConnectorStore) List(_ context.Context, tenantID string) ([]connectors.ConnectorState, error) {
	out := make([]connectors.ConnectorState, 0)
	for _, item := range s.items {
		if item.TenantID == tenantID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *bootstrapMemConnectorStore) Delete(_ context.Context, tenantID, name string) error {
	delete(s.items, s.key(tenantID, name))
	return nil
}

func (s *bootstrapMemConnectorStore) key(tenantID, name string) string {
	return tenantID + "::" + name
}

func mustEncryptBootstrapToken(t *testing.T, value string) string {
	t.Helper()
	enc, err := security.EncryptData(value, "connector-test-key")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	return enc
}

func mustSeedConnectorState(t *testing.T, store *bootstrapMemConnectorStore, tenantID, name, provider, token string) {
	t.Helper()
	now := time.Now().UTC()
	expiresAt := now.Add(30 * time.Minute)
	state := connectors.ConnectorState{
		Name:        name,
		TenantID:    tenantID,
		Provider:    provider,
		AccessToken: mustEncryptBootstrapToken(t, token),
		Status:      connectors.StatusHealthy,
		ExpiresAt:   &expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.Save(context.Background(), &state); err != nil {
		t.Fatalf("seed connector state: %v", err)
	}
}

type captureApprovalStore struct {
	last *approvals.ApprovalRequest
}

func (s *captureApprovalStore) Create(_ context.Context, req *approvals.ApprovalRequest) error {
	copyReq := *req
	s.last = &copyReq
	return nil
}

func (s *captureApprovalStore) Get(context.Context, string) (*approvals.ApprovalRequest, error) {
	return nil, nil
}

func (s *captureApprovalStore) Update(context.Context, *approvals.ApprovalRequest) error {
	return nil
}

func (s *captureApprovalStore) ListPending(context.Context, string) ([]approvals.ApprovalRequest, error) {
	return nil, nil
}

func (s *captureApprovalStore) ListAll(context.Context, string, approvals.ApprovalFilter) ([]approvals.ApprovalRequest, error) {
	return nil, nil
}

func (s *captureApprovalStore) ExpireOld(context.Context) (int, error) {
	return 0, nil
}

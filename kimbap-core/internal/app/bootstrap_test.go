package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	runtimepkg "github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/dunialabs/kimbap-core/internal/services"
	"github.com/dunialabs/kimbap-core/internal/vault"
)

func TestBuildRuntimeNilConfig(t *testing.T) {
	rt, err := BuildRuntime(RuntimeDeps{})
	if err == nil {
		t.Fatalf("expected error for nil config, got runtime=%v", rt)
	}
}

func TestBuildRuntimeMinimalConfig(t *testing.T) {
	cfg := &config.KimbapConfig{
		Services: config.ServicesConfig{Dir: t.TempDir()},
		Policy:   config.PolicyConfig{Path: ""},
	}

	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	if rt == nil {
		t.Fatal("expected runtime to be non-nil")
	}
}

func TestBootstrapRegistersAppleScript(t *testing.T) {
	cfg := &config.KimbapConfig{Services: config.ServicesConfig{Dir: t.TempDir()}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	adapter, ok := rt.Adapters["applescript"]
	if goruntime.GOOS == "darwin" {
		if !ok || adapter == nil {
			t.Fatal("expected applescript adapter to be registered on darwin")
		}
		if adapter.Type() != "applescript" {
			t.Fatalf("expected applescript adapter type, got %q", adapter.Type())
		}
		return
	}
	if ok {
		t.Fatal("expected applescript adapter to be absent on non-darwin")
	}
}

func TestBuildRuntimeWithServices(t *testing.T) {
	servicesDir := t.TempDir()
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
	if err := os.WriteFile(filepath.Join(servicesDir, "test-skill.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}

	cfg := &config.KimbapConfig{Services: config.ServicesConfig{Dir: servicesDir}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	actions, err := rt.ActionRegistry.List(context.Background(), runtimepkg.ListOptions{})
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action from loaded service, got %d", len(actions))
	}
	if actions[0].Name != "test-skill.ping" {
		t.Fatalf("unexpected action name: %s", actions[0].Name)
	}
}

func TestBuildRuntimeWithServicesStrictVerifyFailsWithoutLock(t *testing.T) {
	servicesDir := t.TempDir()
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
	if err := os.WriteFile(filepath.Join(servicesDir, "test-skill.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}

	cfg := &config.KimbapConfig{Services: config.ServicesConfig{Dir: servicesDir, Verify: "strict"}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	_, listErr := rt.ActionRegistry.List(context.Background(), runtimepkg.ListOptions{})
	if listErr == nil {
		t.Fatalf("expected strict verification failure for unlocked service")
	}
}

func TestBuildRuntimeWithServicesStrictVerifyPassesWithLock(t *testing.T) {
	servicesDir := t.TempDir()
	manifest, err := services.ParseManifest([]byte(`name: test-skill
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

	installer := services.NewLocalInstaller(servicesDir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install service with lockfile: %v", err)
	}

	cfg := &config.KimbapConfig{Services: config.ServicesConfig{Dir: servicesDir, Verify: "strict"}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	actionsList, listErr := rt.ActionRegistry.List(context.Background(), runtimepkg.ListOptions{})
	if listErr != nil {
		t.Fatalf("list actions: %v", listErr)
	}
	if len(actionsList) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actionsList))
	}
}

func TestBuildRuntimeWithRequiredSignatureSkipsUnsignedUnlockedService(t *testing.T) {
	servicesDir := t.TempDir()
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
	if err := os.WriteFile(filepath.Join(servicesDir, "test-skill.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}

	cfg := &config.KimbapConfig{Services: config.ServicesConfig{Dir: servicesDir, Verify: "warn", SignaturePolicy: "required"}}
	rt, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	actionsList, listErr := rt.ActionRegistry.List(context.Background(), runtimepkg.ListOptions{})
	if listErr != nil {
		t.Fatalf("list actions: %v", listErr)
	}
	if len(actionsList) != 0 {
		t.Fatalf("expected unsigned unlocked service to be skipped under required signature policy, got %d actions", len(actionsList))
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

	_, err := adapter.CreateRequest(context.Background(), runtimepkg.ApprovalRequest{
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

func TestVaultCredentialResolverTreatsNotFoundAsSoftMiss(t *testing.T) {
	store := &bootstrapVaultStore{getValueErr: vault.ErrSecretNotFound}
	resolver := &vaultCredentialResolver{store: store}

	resolved, err := resolver.Resolve(context.Background(), "tenant-1", actions.AuthRequirement{Type: actions.AuthTypeBearer, CredentialRef: "github.token"})
	if err != nil {
		t.Fatalf("resolve credential: %v", err)
	}
	if resolved != nil {
		t.Fatalf("expected nil credential set for missing vault secret, got %+v", resolved)
	}
	if store.markUsedCalled {
		t.Fatal("expected MarkUsed not to be called when secret is missing")
	}
}

func TestChainCredentialResolverFallsBackToEnvOnVaultMiss(t *testing.T) {
	t.Setenv("KIMBAP_GITHUB_TOKEN", "env-token-123")

	chain := &chainCredentialResolver{resolvers: []runtimepkg.CredentialResolver{
		&vaultCredentialResolver{store: &bootstrapVaultStore{getValueErr: vault.ErrSecretNotFound}},
		&envCredentialResolver{},
	}}

	resolved, err := chain.Resolve(context.Background(), "tenant-1", actions.AuthRequirement{Type: actions.AuthTypeOAuth2, CredentialRef: "github.token"})
	if err != nil {
		t.Fatalf("resolve chain credential: %v", err)
	}
	if resolved == nil {
		t.Fatal("expected credential set from env resolver")
	}
	if resolved.Token != "env-token-123" {
		t.Fatalf("expected env token, got %q", resolved.Token)
	}
}

func TestEnvCredentialResolverReturnsSoftMissWhenEnvMissing(t *testing.T) {
	t.Setenv("KIMBAP_GITHUB_TOKEN", "")

	resolver := &envCredentialResolver{}
	resolved, err := resolver.Resolve(context.Background(), "tenant-1", actions.AuthRequirement{Type: actions.AuthTypeBearer, CredentialRef: "github.token"})
	if err != nil {
		t.Fatalf("resolve env credential: %v", err)
	}
	if resolved != nil {
		t.Fatalf("expected nil when env var is not configured, got %+v", resolved)
	}
}

func TestEnvCredentialResolverHandlesProfiledRef(t *testing.T) {
	t.Setenv("KIMBAP_GITHUB_WORK_TOKEN", "test-token")

	r := &envCredentialResolver{}
	creds, err := r.Resolve(context.Background(), "default", actions.AuthRequirement{
		Type:          actions.AuthTypeBearer,
		CredentialRef: "github:work.token",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds == nil {
		t.Fatal("expected creds, got nil")
	}
	if creds.Token != "test-token" {
		t.Errorf("expected test-token, got %q", creds.Token)
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

type captureApprovalStore struct {
	last *approvals.ApprovalRequest
}

type bootstrapVaultStore struct {
	value          []byte
	getValueErr    error
	markUsedErr    error
	markUsedCalled bool
}

func (s *bootstrapVaultStore) Create(context.Context, string, string, vault.SecretType, []byte, map[string]string, string) (*vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *bootstrapVaultStore) Upsert(context.Context, string, string, vault.SecretType, []byte, map[string]string, string) (*vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *bootstrapVaultStore) GetMeta(context.Context, string, string) (*vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *bootstrapVaultStore) GetValue(context.Context, string, string) ([]byte, error) {
	if s.getValueErr != nil {
		return nil, s.getValueErr
	}
	return s.value, nil
}

func (s *bootstrapVaultStore) List(context.Context, string, vault.ListOptions) ([]vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *bootstrapVaultStore) Delete(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *bootstrapVaultStore) Rotate(context.Context, string, string, []byte, string) (*vault.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *bootstrapVaultStore) GetVersion(context.Context, string, string, int) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (s *bootstrapVaultStore) MarkUsed(context.Context, string, string) error {
	s.markUsedCalled = true
	return s.markUsedErr
}

func (s *bootstrapVaultStore) Exists(context.Context, string, string) (bool, error) {
	return false, errors.New("not implemented")
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

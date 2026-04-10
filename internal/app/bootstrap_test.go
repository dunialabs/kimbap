package app

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
)

func captureBootstrapStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("read captured stderr: %v", readErr)
	}
	return string(out)
}

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
	adapter, ok := rt.Adapters["command"]
	if !ok || adapter == nil {
		t.Fatal("expected command adapter to be registered")
	}
	if adapter.Type() != "command" {
		t.Fatalf("expected command adapter type, got %q", adapter.Type())
	}
}

func TestBuildRuntimeAppliesAppleScriptRegistryMode(t *testing.T) {
	prevMode := services.CurrentAppleScriptRegistryMode()
	t.Cleanup(func() {
		_ = services.SetAppleScriptRegistryMode(string(prevMode))
	})

	cfg := &config.KimbapConfig{
		Services: config.ServicesConfig{Dir: t.TempDir(), AppleScriptRegistryMode: "legacy"},
		Policy:   config.PolicyConfig{Path: ""},
	}

	_, err := BuildRuntime(RuntimeDeps{Config: cfg})
	if err != nil {
		t.Fatalf("BuildRuntime() error: %v", err)
	}
	if got := services.CurrentAppleScriptRegistryMode(); got != services.AppleScriptRegistryModeLegacy {
		t.Fatalf("services.CurrentAppleScriptRegistryMode() = %q, want legacy", got)
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
    idempotent: true
    risk:
      level: low
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
    idempotent: true
    risk:
      level: low
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
    idempotent: true
    risk:
      level: low
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

func TestCollectCommandExecutablesReturnsUniqueCommandPaths(t *testing.T) {
	servicesDir := t.TempDir()
	installer := services.NewLocalInstaller(servicesDir)

	commandManifest, err := services.ParseManifest([]byte(`name: diagram-cli
version: 1.0.0
adapter: command
auth:
  type: none
command_spec:
  executable: /usr/local/bin/mermaid
actions:
  create_diagram:
    command: diagram create
    risk:
      level: low
  validate_diagram:
    command: diagram validate
    risk:
      level: low
`))
	if err != nil {
		t.Fatalf("parse command manifest: %v", err)
	}
	if _, err := installer.Install(commandManifest, "local"); err != nil {
		t.Fatalf("install command manifest: %v", err)
	}

	secondCommandManifest, err := services.ParseManifest([]byte(`name: image-cli
version: 1.0.0
adapter: command
auth:
  type: none
command_spec:
  executable: /opt/bin/imagetool
actions:
  render:
    command: render image
    risk:
      level: low
`))
	if err != nil {
		t.Fatalf("parse second command manifest: %v", err)
	}
	if _, err := installer.Install(secondCommandManifest, "local"); err != nil {
		t.Fatalf("install second command manifest: %v", err)
	}

	httpManifest, err := services.ParseManifest([]byte(`name: http-skill
version: 1.0.0
base_url: https://api.example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
`))
	if err != nil {
		t.Fatalf("parse http manifest: %v", err)
	}
	if _, err := installer.Install(httpManifest, "local"); err != nil {
		t.Fatalf("install http manifest: %v", err)
	}

	registry := &servicesActionRegistry{
		installer:       installer,
		verifyMode:      "off",
		signaturePolicy: "optional",
		servicesDir:     servicesDir,
	}

	got, err := collectCommandExecutables(registry)
	if err != nil {
		t.Fatalf("collectCommandExecutables() error = %v", err)
	}
	want := []string{"/usr/local/bin/mermaid", "/opt/bin/imagetool"}
	if len(got) != len(want) {
		t.Fatalf("len(collectCommandExecutables()) = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("collectCommandExecutables()[%d] = %q, want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestBuildRuntimeFailsWhenCommandAllowlistCollectionErrors(t *testing.T) {
	servicesDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(servicesDir, "bad-command.yaml"), []byte(`name: bad-command
version: 1.0.0
adapter: command
[
`), 0o644); err != nil {
		t.Fatalf("write bad command manifest: %v", err)
	}

	const httpManifest = `name: http-skill
version: 1.0.0
base_url: https://api.example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
`
	if err := os.WriteFile(filepath.Join(servicesDir, "http-skill.yaml"), []byte(httpManifest), 0o644); err != nil {
		t.Fatalf("write http manifest: %v", err)
	}

	var (
		rt       *runtimepkg.Runtime
		buildErr error
	)
	stderr := captureBootstrapStderr(t, func() {
		rt, buildErr = BuildRuntime(RuntimeDeps{Config: &config.KimbapConfig{Services: config.ServicesConfig{Dir: servicesDir}, Policy: config.PolicyConfig{Path: ""}}})
	})
	if buildErr != nil {
		t.Fatalf("expected BuildRuntime() to keep unrelated services available, got %v", buildErr)
	}
	if !strings.Contains(stderr, "command allowlist initialization failed") {
		t.Fatalf("expected bootstrap warning about command allowlist failure, got %q", stderr)
	}

	if rt == nil {
		t.Fatal("expected runtime to be created")
	}
	httpAdapter, ok := rt.Adapters["http"]
	if !ok || httpAdapter == nil {
		t.Fatal("expected unrelated http adapter to remain available")
	}

	commandAdapter, ok := rt.Adapters["command"]
	if !ok || commandAdapter == nil {
		t.Fatal("expected command adapter to be registered")
	}
	blockedReq := adapters.AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{ExecutablePath: "/bin/echo"},
		},
	}
	if _, execErr := commandAdapter.Execute(context.Background(), blockedReq); execErr == nil {
		t.Fatal("expected command adapter to deny execution when allowlist collection failed")
	}
}

func TestServicesActionRegistryRefreshesDefinitionsImmediatelyAfterManifestChange(t *testing.T) {
	servicesDir := t.TempDir()
	const manifest = `name: cached-skill
version: 1.0.0
description: cached skill
base_url: https://api.example.com
auth:
  type: header
  header_name: Authorization
  credential_ref: test.token
actions:
  ping:
    method: GET
    path: /ping
    idempotent: true
    risk:
      level: low
`
	manifestPath := filepath.Join(servicesDir, "cached-skill.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}

	registry := &servicesActionRegistry{
		installer:        services.NewLocalInstaller(servicesDir),
		verifyMode:       "off",
		signaturePolicy:  "optional",
		servicesDir:      servicesDir,
		fullScanInterval: 10 * time.Millisecond,
	}

	first, err := registry.loadDefinitions()
	if err != nil {
		t.Fatalf("load definitions first call: %v", err)
	}
	if len(first) != 1 || first[0].Name != "cached-skill.ping" {
		t.Fatalf("unexpected first definitions: %+v", first)
	}

	if err := os.Remove(manifestPath); err != nil {
		t.Fatalf("remove manifest: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	second, err := registry.loadDefinitions()
	if err != nil {
		t.Fatalf("load definitions second call: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("expected no definitions after manifest deletion, got %+v", second)
	}
}

func TestLoadDefinitionsReturnsWarmCacheWithinScanInterval(t *testing.T) {
	servicesDir := t.TempDir()
	const manifest = `name: cached-skill
version: 1.0.0
description: cached skill
base_url: https://example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    idempotent: true
    risk:
      level: low
`
	manifestPath := filepath.Join(servicesDir, "cached-skill.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}

	registry := &servicesActionRegistry{
		installer:        services.NewLocalInstaller(servicesDir),
		verifyMode:       "off",
		signaturePolicy:  "optional",
		servicesDir:      servicesDir,
		fullScanInterval: time.Hour,
	}

	first, err := registry.loadDefinitions()
	if err != nil {
		t.Fatalf("load definitions first call: %v", err)
	}
	if len(first) != 1 || first[0].Name != "cached-skill.ping" {
		t.Fatalf("unexpected first definitions: %+v", first)
	}

	if err := os.Remove(manifestPath); err != nil {
		t.Fatalf("remove manifest: %v", err)
	}

	second, err := registry.loadDefinitions()
	if err != nil {
		t.Fatalf("load definitions second call: %v", err)
	}
	if len(second) != 1 || second[0].Name != "cached-skill.ping" {
		t.Fatalf("expected warm cache definitions within scan interval, got %+v", second)
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
    idempotent: true
    risk:
      level: low
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

func TestBuildRuntimeDoesNotWarnForUnrelatedUnlockedServices(t *testing.T) {
	servicesDir := t.TempDir()
	installer := services.NewLocalInstaller(servicesDir)

	validManifest, err := services.ParseManifest([]byte(`name: target-cli
version: 1.0.0
adapter: command
auth:
  type: none
command_spec:
  executable: /usr/bin/printf
actions:
  list:
    command: list
    risk:
      level: low
`))
	if err != nil {
		t.Fatalf("parse target manifest: %v", err)
	}
	if _, err := installer.Install(validManifest, "local"); err != nil {
		t.Fatalf("install target manifest: %v", err)
	}

	if err := os.WriteFile(filepath.Join(servicesDir, "unrelated-http.yaml"), []byte(`name: unrelated-http
version: 1.0.0
base_url: https://example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
`), 0o644); err != nil {
		t.Fatalf("write unrelated manifest: %v", err)
	}

	cfg := &config.KimbapConfig{
		Services: config.ServicesConfig{Dir: servicesDir, Verify: "warn", SignaturePolicy: "optional"},
		Policy:   config.PolicyConfig{Path: ""},
	}
	stderr := captureBootstrapStderr(t, func() {
		if _, err := BuildRuntime(RuntimeDeps{Config: cfg}); err != nil {
			t.Fatalf("build runtime: %v", err)
		}
	})
	if strings.Contains(stderr, "unrelated-http") || strings.Contains(stderr, "failed digest verification") {
		t.Fatalf("expected runtime bootstrap to stay quiet for unrelated unlocked services, got stderr=%q", stderr)
	}
}

func TestServicesActionRegistryLookupOnlyWarnsForRequestedService(t *testing.T) {
	servicesDir := t.TempDir()
	installer := services.NewLocalInstaller(servicesDir)

	validManifest, err := services.ParseManifest([]byte(`name: target-http
version: 1.0.0
base_url: https://example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
`))
	if err != nil {
		t.Fatalf("parse target manifest: %v", err)
	}
	if _, err := installer.Install(validManifest, "local"); err != nil {
		t.Fatalf("install target manifest: %v", err)
	}

	if err := os.WriteFile(filepath.Join(servicesDir, "unrelated-http.yaml"), []byte(`name: unrelated-http
version: 1.0.0
base_url: https://example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
`), 0o644); err != nil {
		t.Fatalf("write unrelated manifest: %v", err)
	}

	registry := &servicesActionRegistry{
		installer:       installer,
		verifyMode:      "warn",
		signaturePolicy: "optional",
		servicesDir:     servicesDir,
	}
	stderr := captureBootstrapStderr(t, func() {
		def, err := registry.Lookup(context.Background(), "target-http.ping")
		if err != nil {
			t.Fatalf("lookup target action: %v", err)
		}
		if def == nil || def.Name != "target-http.ping" {
			t.Fatalf("unexpected lookup result: %+v", def)
		}
	})
	if strings.Contains(stderr, "unrelated-http") || strings.Contains(stderr, "failed digest verification") {
		t.Fatalf("expected lookup to avoid unrelated digest warnings, got stderr=%q", stderr)
	}
}

func TestServicesActionRegistryLookupMissingActionAvoidsUnrelatedWarnings(t *testing.T) {
	servicesDir := t.TempDir()
	installer := services.NewLocalInstaller(servicesDir)

	validManifest, err := services.ParseManifest([]byte(`name: target-http
version: 1.0.0
base_url: https://example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
`))
	if err != nil {
		t.Fatalf("parse target manifest: %v", err)
	}
	if _, err := installer.Install(validManifest, "local"); err != nil {
		t.Fatalf("install target manifest: %v", err)
	}

	if err := os.WriteFile(filepath.Join(servicesDir, "unrelated-http.yaml"), []byte(`name: unrelated-http
version: 1.0.0
base_url: https://example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
`), 0o644); err != nil {
		t.Fatalf("write unrelated manifest: %v", err)
	}

	registry := &servicesActionRegistry{
		installer:       installer,
		verifyMode:      "warn",
		signaturePolicy: "optional",
		servicesDir:     servicesDir,
	}

	var lookupErr error
	stderr := captureBootstrapStderr(t, func() {
		def, err := registry.Lookup(context.Background(), "target-http.missing")
		if def != nil {
			t.Fatalf("expected nil definition for missing action, got %+v", def)
		}
		lookupErr = err
	})
	if !errors.Is(lookupErr, actions.ErrLookupNotFound) {
		t.Fatalf("expected lookup not found error, got %v", lookupErr)
	}
	if strings.Contains(stderr, "unrelated-http") || strings.Contains(stderr, "failed digest verification") {
		t.Fatalf("expected missing-action lookup to avoid unrelated digest warnings, got stderr=%q", stderr)
	}
}

func TestServicesActionRegistryLookupMissingServiceAvoidsUnrelatedWarnings(t *testing.T) {
	servicesDir := t.TempDir()
	installer := services.NewLocalInstaller(servicesDir)

	if err := os.WriteFile(filepath.Join(servicesDir, "unrelated-http.yaml"), []byte(`name: unrelated-http
version: 1.0.0
base_url: https://example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    risk:
      level: low
`), 0o644); err != nil {
		t.Fatalf("write unrelated manifest: %v", err)
	}

	registry := &servicesActionRegistry{
		installer:       installer,
		verifyMode:      "warn",
		signaturePolicy: "optional",
		servicesDir:     servicesDir,
	}

	var lookupErr error
	stderr := captureBootstrapStderr(t, func() {
		def, err := registry.Lookup(context.Background(), "missing-service.ping")
		if def != nil {
			t.Fatalf("expected nil definition for missing service, got %+v", def)
		}
		lookupErr = err
	})
	if !errors.Is(lookupErr, actions.ErrLookupNotFound) {
		t.Fatalf("expected lookup not found error, got %v", lookupErr)
	}
	if strings.Contains(stderr, "unrelated-http") || strings.Contains(stderr, "failed digest verification") {
		t.Fatalf("expected missing-service lookup to avoid unrelated digest warnings, got stderr=%q", stderr)
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

func TestApprovalManagerAdapterCancelRequestDeniesWithSystemPrincipal(t *testing.T) {
	store := &captureApprovalStore{}
	mgr := approvals.NewApprovalManager(store, nil, time.Minute)
	adapter := NewApprovalManagerAdapter(mgr)

	res, err := adapter.CreateRequest(context.Background(), runtimepkg.ApprovalRequest{
		TenantID:  "tenant-a",
		RequestID: "req-cancel",
		Principal: actions.Principal{AgentName: "agent-a"},
		Action:    actions.ActionDefinition{Name: "github.issues.create"},
	})
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if res == nil || strings.TrimSpace(res.RequestID) == "" {
		t.Fatalf("expected approval request id, got %+v", res)
	}

	if err := adapter.CancelRequest(context.Background(), res.RequestID, "hold failed"); err != nil {
		t.Fatalf("CancelRequest: %v", err)
	}

	req, err := store.Get(context.Background(), res.RequestID)
	if err != nil {
		t.Fatalf("Get approval: %v", err)
	}
	if req == nil {
		t.Fatal("expected approval request after cancel")
	}
	if req.Status != approvals.StatusDenied {
		t.Fatalf("expected denied status after cancel, got %q", req.Status)
	}
	if req.ResolvedBy != "system" {
		t.Fatalf("expected resolved_by system, got %q", req.ResolvedBy)
	}
	if req.DenyReason != "hold failed" {
		t.Fatalf("expected deny reason 'hold failed', got %q", req.DenyReason)
	}
}

func TestMemoryHeldExecutionStoreCopiesNestedPayload(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryHeldExecutionStore()
	nested := map[string]any{"count": 1}
	tags := []string{"alpha", "beta"}
	req := actions.ExecutionRequest{
		RequestID: "req-held",
		TenantID:  "tenant-a",
		Input: map[string]any{
			"nested": nested,
			"tags":   tags,
		},
	}

	if err := store.Hold(ctx, "apr-held", req); err != nil {
		t.Fatalf("hold execution: %v", err)
	}

	nested["count"] = 2
	tags[0] = "changed"

	resumed, err := store.Resume(ctx, "apr-held")
	if err != nil {
		t.Fatalf("resume held execution: %v", err)
	}
	if resumed == nil {
		t.Fatal("expected resumed held execution")
	}
	storedNested, ok := resumed.Input["nested"].(map[string]any)
	if !ok || storedNested["count"] != float64(1) {
		t.Fatalf("expected nested payload copy, got %#v", resumed.Input["nested"])
	}
	storedTags, ok := resumed.Input["tags"].([]any)
	if !ok || len(storedTags) != 2 || storedTags[0] != "alpha" {
		t.Fatalf("expected tags payload copy, got %#v", resumed.Input["tags"])
	}
}

func TestAuditWriterAdapterPreservesInputPayload(t *testing.T) {
	capture := &captureAuditWriter{}
	adapter := NewAuditWriterAdapter(capture)

	err := adapter.Write(context.Background(), runtimepkg.AuditEvent{
		RequestID:  "req-1",
		ActionName: "github.issues.create",
		Input:      map[string]any{"title": "hello"},
		Mode:       actions.ModeServe,
		Status:     actions.StatusSuccess,
		Timestamp:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("write audit event: %v", err)
	}
	if capture.last == nil {
		t.Fatal("expected captured audit event")
	}
	if capture.last.Input == nil || capture.last.Input["title"] != "hello" {
		t.Fatalf("expected input payload preserved, got %+v", capture.last.Input)
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

func TestStorePolicyEvaluatorFallsBackOnTenantPolicyNotFound(t *testing.T) {
	fallback := &stubPolicyEvaluator{decision: &runtimepkg.PolicyDecision{Decision: "allow"}}
	evaluator := &storePolicyEvaluator{
		policyStore: &stubPolicyStore{getErr: store.ErrNotFound},
		fallback:    fallback,
	}

	decision, err := evaluator.Evaluate(context.Background(), runtimepkg.PolicyRequest{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("evaluate policy: %v", err)
	}
	if decision == nil || decision.Decision != "allow" {
		t.Fatalf("expected fallback decision allow, got %+v", decision)
	}
	if fallback.calls != 1 {
		t.Fatalf("expected fallback evaluator called once, got %d", fallback.calls)
	}
}

func TestStorePolicyEvaluatorFailsClosedOnPolicyStoreError(t *testing.T) {
	fallback := &stubPolicyEvaluator{decision: &runtimepkg.PolicyDecision{Decision: "allow"}}
	evaluator := &storePolicyEvaluator{
		policyStore: &stubPolicyStore{getErr: errors.New("db unavailable")},
		fallback:    fallback,
	}

	decision, err := evaluator.Evaluate(context.Background(), runtimepkg.PolicyRequest{TenantID: "tenant-a"})
	if err == nil {
		t.Fatalf("expected error, got decision=%+v", decision)
	}
	if fallback.calls != 0 {
		t.Fatalf("expected fallback evaluator not called, got %d", fallback.calls)
	}
}

type bootstrapMemConnectorStore struct {
	items map[string]connectors.ConnectorState
}

type stubPolicyStore struct {
	data   []byte
	getErr error
}

func (s *stubPolicyStore) SetPolicy(context.Context, string, []byte) error {
	return nil
}

func (s *stubPolicyStore) GetPolicy(context.Context, string) ([]byte, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.data, nil
}

type stubPolicyEvaluator struct {
	decision *runtimepkg.PolicyDecision
	err      error
	calls    int
}

func (s *stubPolicyEvaluator) Evaluate(context.Context, runtimepkg.PolicyRequest) (*runtimepkg.PolicyDecision, error) {
	s.calls++
	return s.decision, s.err
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
	last  *approvals.ApprovalRequest
	items map[string]approvals.ApprovalRequest
}

type captureAuditWriter struct {
	last *audit.AuditEvent
}

func (w *captureAuditWriter) Write(_ context.Context, event audit.AuditEvent) error {
	copyEvent := event
	w.last = &copyEvent
	return nil
}

func (w *captureAuditWriter) Close() error { return nil }

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
	if s.items == nil {
		s.items = make(map[string]approvals.ApprovalRequest)
	}
	s.items[copyReq.ID] = copyReq
	return nil
}

func (s *captureApprovalStore) Get(_ context.Context, id string) (*approvals.ApprovalRequest, error) {
	if s.items == nil {
		return nil, nil
	}
	req, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyReq := req
	return &copyReq, nil
}

func (s *captureApprovalStore) Update(_ context.Context, req *approvals.ApprovalRequest) error {
	if req == nil {
		return nil
	}
	copyReq := *req
	s.last = &copyReq
	if s.items == nil {
		s.items = make(map[string]approvals.ApprovalRequest)
	}
	s.items[copyReq.ID] = copyReq
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

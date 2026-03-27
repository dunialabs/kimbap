package services

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap-core/internal/actions"
)

func loadManifestFixture(t *testing.T, fixtureName string) *ServiceManifest {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}
	fixturePath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", fixtureName)
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixturePath, err)
	}
	manifest, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("parse fixture %q: %v", fixtureName, err)
	}
	return manifest
}

const braveSearchFixture = `name: brave-search
version: 1.0.0
description: Brave Search API integration
base_url: https://api.search.brave.com
auth:
  type: header
  header_name: X-Subscription-Token
  credential_ref: brave.api_key
actions:
  web_search:
    method: GET
    path: /res/v1/web/search
    description: Search the web
    args:
      - name: q
        type: string
        required: true
      - name: count
        type: integer
        required: false
        default: 10
    request:
      query:
        q: "{q}"
        count: "{count}"
      headers:
        Accept: application/json
    response:
      extract: data.results
      type: array
    idempotent: true
    risk:
      level: low
    retry:
      max_attempts: 3
      backoff_ms: 200
      retry_on: [429, 500, 502, 503]
    pagination:
      type: offset
      max_pages: 5
      next_path: meta.next_offset
    error_mapping:
      401: unauthenticated
      429: rate_limited
`

const manifestWithAdapterFixture = `name: notes-service
version: 1.0.0
description: Notes service
adapter: applescript
target_app: Notes
auth:
  type: none
actions:
  list_notes:
    command: list-notes
    risk:
      level: low
`

const manifestWithoutAdapterFixture = `name: notes-service
version: 1.0.0
description: Notes service
base_url: https://example.com
auth:
  type: none
actions:
  list_notes:
    method: GET
    path: /notes
    risk:
      level: low
`

const actionWithCommandFixture = `name: notes-service
version: 1.0.0
description: Notes service
adapter: applescript
target_app: Notes
auth:
  type: none
actions:
  list_notes:
    description: List notes
    command: list-notes
    idempotent: true
    risk:
      level: low
`

func TestParseManifestValidFixture(t *testing.T) {
	m, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if m.Name != "brave-search" {
		t.Fatalf("unexpected name: %s", m.Name)
	}
	if m.Actions["web_search"].Method != "GET" {
		t.Fatalf("unexpected action method: %s", m.Actions["web_search"].Method)
	}
	if m.Auth.CredentialRef != "brave.api_key" {
		t.Fatalf("unexpected credential ref: %s", m.Auth.CredentialRef)
	}
}

func TestParseManifestWithAdapterField(t *testing.T) {
	m, err := ParseManifest([]byte(manifestWithAdapterFixture))
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if m.Adapter != "applescript" {
		t.Fatalf("unexpected adapter: %s", m.Adapter)
	}
	if m.TargetApp != "Notes" {
		t.Fatalf("unexpected target app: %s", m.TargetApp)
	}
}

func TestParseManifestDefaultsToEmptyAdapter(t *testing.T) {
	m, err := ParseManifest([]byte(manifestWithoutAdapterFixture))
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if m.Adapter != "" {
		t.Fatalf("expected empty adapter, got %q", m.Adapter)
	}
}

func TestParseActionWithCommandField(t *testing.T) {
	m, err := ParseManifest([]byte(actionWithCommandFixture))
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	action := m.Actions["list_notes"]
	if action.Command != "list-notes" {
		t.Fatalf("unexpected command: %s", action.Command)
	}
	if action.Idempotent == nil || !*action.Idempotent {
		t.Fatalf("expected idempotent=true, got %v", action.Idempotent)
	}
}

func TestValidateManifestMissingFields(t *testing.T) {
	manifest := &ServiceManifest{
		Name:    "Invalid Name",
		Version: "x",
		BaseURL: "not-a-url",
		Auth: ServiceAuth{
			Type:          "header",
			CredentialRef: "",
		},
		Actions: map[string]ServiceAction{
			"bad": {
				Method: "",
				Path:   "",
				Args: []ActionArg{{
					Name:     "q",
					Type:     "string",
					Required: true,
					Default:  "x",
				}},
				Risk: RiskSpec{Level: "extreme"},
			},
		},
	}

	errs := ValidateManifest(manifest)
	if len(errs) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestValidateServiceNameRejectsReserved(t *testing.T) {
	err := ValidateServiceName("kimbap")
	if err == nil {
		t.Error("expected error for reserved name 'kimbap', got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "reserved") {
		t.Errorf("expected 'reserved' in error message, got: %v", err)
	}
}

func TestInstallRejectsReservedName(t *testing.T) {
	tmpDir := t.TempDir()
	inst := NewLocalInstaller(tmpDir)
	m := &ServiceManifest{
		Name:    "kimbap",
		Version: "1.0.0",
		BaseURL: "https://example.com",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"get": {Method: "GET", Path: "/v1/items", Risk: RiskSpec{Level: "low"}},
		},
	}
	_, err := inst.Install(m, "test")
	if err == nil {
		t.Error("expected error installing reserved name 'kimbap', got nil")
	}
}

func TestValidateManifestHeaderAuthRequiresHeaderName(t *testing.T) {
	m := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "header", CredentialRef: "svc.key"},
		Actions: map[string]ServiceAction{
			"get": {Method: "GET", Path: "/v1/get", Risk: RiskSpec{Level: "low"}},
		},
	}
	err := ValidateManifest(m)
	found := false
	for _, e := range err {
		if e.Field == "auth.header_name" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected auth.header_name validation error, got: %v", err)
	}
}

func TestValidateManifestQueryAuthRequiresQueryParam(t *testing.T) {
	m := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "query", CredentialRef: "svc.key"},
		Actions: map[string]ServiceAction{
			"get": {Method: "GET", Path: "/v1/get", Risk: RiskSpec{Level: "low"}},
		},
	}
	err := ValidateManifest(m)
	found := false
	for _, e := range err {
		if e.Field == "auth.query_param" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected auth.query_param validation error, got: %v", err)
	}
}

func TestValidateManifestBodyAuthRequiresBodyField(t *testing.T) {
	m := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "body", CredentialRef: "svc.key"},
		Actions: map[string]ServiceAction{
			"post": {Method: "POST", Path: "/v1/post", Risk: RiskSpec{Level: "low"}},
		},
	}
	err := ValidateManifest(m)
	found := false
	for _, e := range err {
		if e.Field == "auth.body_field" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected auth.body_field validation error, got: %v", err)
	}
}

func TestValidateManifestActionLevelAuthIsValidated(t *testing.T) {
	m := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"get": {
				Method: "GET",
				Path:   "/v1/get",
				Risk:   RiskSpec{Level: "low"},
				Auth:   &ServiceAuth{Type: "header"},
			},
		},
	}
	err := ValidateManifest(m)
	fieldNames := make([]string, 0, len(err))
	for _, e := range err {
		fieldNames = append(fieldNames, e.Field)
	}
	if !slices.Contains(fieldNames, "actions.get.auth.credential_ref") {
		t.Errorf("expected action auth.credential_ref error, got fields: %v", fieldNames)
	}
	if !slices.Contains(fieldNames, "actions.get.auth.header_name") {
		t.Errorf("expected action auth.header_name error, got fields: %v", fieldNames)
	}
}

func TestLocalInstallerLifecycle(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)

	installed, err := installer.Install(manifest, "local")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "brave-search.yaml")); err != nil {
		t.Fatalf("expected service file to exist: %v", err)
	}
	if installed.Manifest.Name != "brave-search" {
		t.Fatalf("unexpected installed service name: %s", installed.Manifest.Name)
	}

	list, err := installer.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 installed service, got %d", len(list))
	}

	got, err := installer.Get("brave-search")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Manifest.Version != "1.0.0" {
		t.Fatalf("unexpected version: %s", got.Manifest.Version)
	}

	if err := installer.Remove("brave-search"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	list, err = installer.List()
	if err != nil {
		t.Fatalf("list after remove failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 skills after remove, got %d", len(list))
	}
}

func TestInstallDuplicateRejectedWithoutForce(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)

	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("first install: %v", err)
	}

	_, err = installer.Install(manifest, "local")
	if err != ErrServiceAlreadyInstalled {
		t.Fatalf("expected ErrServiceAlreadyInstalled, got %v", err)
	}
}

func TestInstallWithForceOverwritesExisting(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)

	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("first install: %v", err)
	}

	reinstalled, err := installer.InstallWithForce(manifest, "local", true)
	if err != nil {
		t.Fatalf("force reinstall failed: %v", err)
	}
	if reinstalled.Manifest.Name != "brave-search" {
		t.Fatalf("unexpected name after force: %s", reinstalled.Manifest.Name)
	}
}

func TestInstallWithForcePreservesExistingManifestOnLockfileWriteFailure(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("initial install: %v", err)
	}

	manifestPath := filepath.Join(dir, "brave-search.yaml")
	originalData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read original manifest: %v", err)
	}

	updated := *manifest
	updated.Description = "Updated description"

	lockPath := installer.lockfilePath()
	if err := os.Chmod(lockPath, 0o444); err != nil {
		t.Fatalf("chmod lockfile read-only: %v", err)
	}
	defer os.Chmod(lockPath, 0o644)

	if _, err := installer.InstallWithForce(&updated, "local", true); err == nil {
		t.Fatal("expected force install to fail when lockfile cannot be written")
	}

	restoredData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read restored manifest: %v", err)
	}
	if string(restoredData) != string(originalData) {
		t.Fatalf("expected original manifest restored after lockfile failure\nwant:\n%s\n\ngot:\n%s", string(originalData), string(restoredData))
	}
}

func TestInstallWritesLockfileAndVerifyPasses(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)

	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	lockPath := filepath.Join(dir, "kimbap-services.lock")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Errorf("expected kimbap-services.lock to exist at %s", lockPath)
	}

	lf, err := installer.readLockfile()
	if err != nil {
		t.Fatalf("read lockfile failed: %v", err)
	}
	entry, ok := lf.Services["brave-search"]
	if !ok {
		t.Fatal("expected lock entry for brave-search")
	}
	if entry.Digest == "" {
		t.Fatal("expected non-empty lock digest")
	}

	result, err := installer.Verify("brave-search")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !result.Locked {
		t.Fatal("expected locked result")
	}
	if !result.Verified {
		t.Fatalf("expected verified=true, got %+v", result)
	}
	if result.ExpectedDigest != result.ActualDigest {
		t.Fatalf("expected digest match, got expected=%q actual=%q", result.ExpectedDigest, result.ActualDigest)
	}
}

func TestRemoveAndVerifyGone(t *testing.T) {
	tmpDir := t.TempDir()
	inst := NewLocalInstaller(tmpDir)
	manifest := &ServiceManifest{
		Name:    "myservice",
		Version: "1.0.0",
		BaseURL: "https://example.com",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"get": {Method: "GET", Path: "/v1/items", Risk: RiskSpec{Level: "low"}},
		},
	}

	if _, err := inst.Install(manifest, "test"); err != nil {
		t.Fatalf("install: %v", err)
	}

	if err := inst.Remove("myservice"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if _, err := inst.Verify("myservice"); err == nil {
		t.Fatalf("expected verify to fail for removed service")
	}

	installed, err := inst.List()
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if len(installed) != 0 {
		t.Errorf("expected empty list after remove, got %d entries", len(installed))
	}
}

func TestListReturnsErrorOnInvalidInstalledManifest(t *testing.T) {
	tmpDir := t.TempDir()
	inst := NewLocalInstaller(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "broken.yaml"), []byte("name: broken\n"), 0o644); err != nil {
		t.Fatalf("write broken manifest: %v", err)
	}

	_, err := inst.List()
	if err == nil {
		t.Fatal("expected list to fail on invalid installed manifest")
	}
}

func TestVerifyDetectsDigestMismatch(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)

	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	skillPath := filepath.Join(dir, "brave-search.yaml")
	if err := os.WriteFile(skillPath, []byte(braveSearchFixture+"\n# tampered\n"), 0o644); err != nil {
		t.Fatalf("tamper service file failed: %v", err)
	}

	result, err := installer.Verify("brave-search")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !result.Locked {
		t.Fatal("expected locked=true for installed lock entry")
	}
	if result.Verified {
		t.Fatalf("expected verified=false for digest mismatch, got %+v", result)
	}
}

func TestRemoveDeletesLockEntry(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)

	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if err := installer.Remove("brave-search"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	lf, err := installer.readLockfile()
	if err != nil {
		t.Fatalf("read lockfile failed: %v", err)
	}
	if _, ok := lf.Services["brave-search"]; ok {
		t.Fatal("expected brave-search lock entry to be removed")
	}
}

func TestRemovePreservesManifestOnLockfileWriteFailure(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	manifestPath := filepath.Join(dir, "brave-search.yaml")
	originalData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read original manifest: %v", err)
	}

	lockPath := installer.lockfilePath()
	if err := os.Chmod(lockPath, 0o444); err != nil {
		t.Fatalf("chmod lockfile read-only: %v", err)
	}
	defer os.Chmod(lockPath, 0o644)

	if err := installer.Remove("brave-search"); err == nil {
		t.Fatal("expected remove to fail when lockfile cannot be written")
	}

	restoredData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read restored manifest: %v", err)
	}
	if string(restoredData) != string(originalData) {
		t.Fatalf("expected original manifest restored after remove failure\nwant:\n%s\n\ngot:\n%s", string(originalData), string(restoredData))
	}

	lf, err := installer.readLockfile()
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	if _, ok := lf.Services["brave-search"]; !ok {
		t.Fatal("expected lock entry to remain when remove rollback occurs")
	}
}

func TestToActionDefinitions(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}

	if len(defs) != 1 {
		t.Fatalf("expected one action definition, got %d", len(defs))
	}

	def := defs[0]
	if def.Name != "brave-search.web_search" {
		t.Fatalf("unexpected action name: %s", def.Name)
	}
	if def.Risk != actions.RiskRead {
		t.Fatalf("unexpected risk level: %s", def.Risk)
	}
	if def.Auth.Type != actions.AuthTypeHeader || def.Auth.HeaderName != "X-Subscription-Token" {
		t.Fatalf("unexpected auth mapping: %+v", def.Auth)
	}
	if def.Adapter.Type != "http" || def.Adapter.BaseURL != "https://api.search.brave.com" {
		t.Fatalf("unexpected adapter config: %+v", def.Adapter)
	}
	if def.Adapter.URLTemplate != "/res/v1/web/search" {
		t.Fatalf("unexpected adapter path: %s", def.Adapter.URLTemplate)
	}
}

func TestConvertAppleScriptManifest(t *testing.T) {
	manifest := loadManifestFixture(t, "apple-notes.yaml")
	if manifest.BaseURL != "" {
		t.Fatalf("expected applescript fixture base_url to be empty, got %q", manifest.BaseURL)
	}

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert applescript manifest failed: %v", err)
	}

	if len(defs) != 5 {
		t.Fatalf("expected 5 action definitions, got %d", len(defs))
	}

	gotNames := make([]string, 0, len(defs))
	for _, def := range defs {
		gotNames = append(gotNames, def.Name)
	}
	wantNames := []string{
		"apple-notes.create-note",
		"apple-notes.get-note",
		"apple-notes.list-folders",
		"apple-notes.list-notes",
		"apple-notes.search-notes",
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("unexpected applescript action names:\n got: %v\nwant: %v", gotNames, wantNames)
	}
}

func TestInstallAppleNotesSkill(t *testing.T) {
	manifest, err := ParseManifestFile("../../skills/official/apple-notes.yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if manifest.Adapter != "applescript" {
		t.Errorf("adapter = %q, want applescript", manifest.Adapter)
	}
	if manifest.TargetApp != "Notes" {
		t.Errorf("target_app = %q, want Notes", manifest.TargetApp)
	}
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(defs) != 5 {
		t.Errorf("got %d actions, want 5", len(defs))
	}
}

func TestConvertAppleScriptManifest_AdapterConfig(t *testing.T) {
	manifest := loadManifestFixture(t, "apple-notes.yaml")

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert applescript manifest failed: %v", err)
	}

	wantCommandByName := map[string]string{
		"apple-notes.create-note":  "create-note",
		"apple-notes.get-note":     "get-note",
		"apple-notes.list-folders": "list-folders",
		"apple-notes.list-notes":   "list-notes",
		"apple-notes.search-notes": "search-notes",
	}

	for _, def := range defs {
		if def.Adapter.Type != "applescript" {
			t.Fatalf("expected adapter type applescript for %s, got %q", def.Name, def.Adapter.Type)
		}
		if def.Adapter.TargetApp != "Notes" {
			t.Fatalf("expected adapter target app Notes for %s, got %q", def.Name, def.Adapter.TargetApp)
		}
		wantCommand, ok := wantCommandByName[def.Name]
		if !ok {
			t.Fatalf("unexpected action definition name %q", def.Name)
		}
		if def.Adapter.Command != wantCommand {
			t.Fatalf("expected adapter command %q for %s, got %q", wantCommand, def.Name, def.Adapter.Command)
		}
	}
}

func TestConvertAppleScriptManifest_NoClassifiers(t *testing.T) {
	manifest := loadManifestFixture(t, "apple-notes.yaml")

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert applescript manifest failed: %v", err)
	}

	for _, def := range defs {
		if def.Classifiers != nil {
			t.Fatalf("expected nil classifiers for %s, got %+v", def.Name, def.Classifiers)
		}
	}
}

func TestConvertHTTPManifest_Unchanged(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	got, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions failed: %v", err)
	}

	want, err := toHTTPDefinitions(manifest)
	if err != nil {
		t.Fatalf("toHTTPDefinitions failed: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected HTTP conversion to remain unchanged")
	}
}

func TestToActionDefinitionsIncludesPathParamsInInputSchema(t *testing.T) {
	manifest := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth: ServiceAuth{
			Type:          "header",
			HeaderName:    "Authorization",
			CredentialRef: "svc.token",
		},
		Actions: map[string]ServiceAction{
			"get_item": {
				Method: "GET",
				Path:   "/items/{item_id}",
				Request: RequestSpec{
					PathParams: map[string]string{"item_id": "{item_id}"},
				},
				Risk: RiskSpec{Level: "low"},
			},
		},
	}

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected one action definition, got %d", len(defs))
	}

	input := defs[0].InputSchema
	if input == nil || input.Properties == nil {
		t.Fatalf("expected input schema properties")
	}
	field, ok := input.Properties["item_id"]
	if !ok {
		t.Fatalf("expected path param field to be present")
	}
	if field.Type != "string" {
		t.Fatalf("expected path param type string, got %q", field.Type)
	}
	if len(input.Required) != 1 || input.Required[0] != "item_id" {
		t.Fatalf("expected item_id to be required, got %+v", input.Required)
	}
}

func TestParseManifestAuthNone(t *testing.T) {
	manifest := `name: public-api
version: 1.0.0
description: Public API without authentication
base_url: https://api.example.com
auth:
  type: none
actions:
  get_status:
    method: GET
    path: /status
    description: Get system status
    risk:
      level: low
    response:
      type: object
`
	m, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("expected auth:none to parse successfully, got: %v", err)
	}
	if m.Auth.Type != "none" {
		t.Fatalf("expected auth type none, got %q", m.Auth.Type)
	}
	if m.Auth.CredentialRef != "" {
		t.Fatalf("expected empty credential_ref for none auth, got %q", m.Auth.CredentialRef)
	}
}

func TestParseManifestAcceptsNumberArgType(t *testing.T) {
	manifest := `name: numeric-api
version: 1.0.0
description: Numeric API
base_url: https://api.example.com
auth:
  type: none
actions:
  create_price:
    method: POST
    path: /prices
    idempotent: false
    args:
      - name: amount
        type: number
        required: true
    request:
      body:
        amount: "{amount}"
    response:
      type: object
    risk:
      level: medium
`

	m, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("expected number arg type to be valid, got: %v", err)
	}
	if got := m.Actions["create_price"].Args[0].Type; got != "number" {
		t.Fatalf("expected parsed type number, got %q", got)
	}
}

func TestConvertAuthNone(t *testing.T) {
	manifest := &ServiceManifest{
		Name:    "public-api",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"status": {
				Method: "GET",
				Path:   "/status",
				Risk:   RiskSpec{Level: "low"},
			},
		},
	}
	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 action, got %d", len(defs))
	}
	if defs[0].Auth.Type != actions.AuthTypeNone {
		t.Fatalf("expected auth type none, got %s", defs[0].Auth.Type)
	}
	if !defs[0].Auth.Optional {
		t.Fatal("expected auth optional=true for none type")
	}
}

func TestToActionDefinitionsUsesActionLevelAuthOverride(t *testing.T) {
	manifest := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth: ServiceAuth{
			Type:          "header",
			HeaderName:    "Authorization",
			CredentialRef: "svc.token",
		},
		Actions: map[string]ServiceAction{
			"search": {
				Method: "GET",
				Path:   "/search",
				Auth: &ServiceAuth{
					Type:          "query",
					QueryParam:    "api_key",
					CredentialRef: "svc.api_key",
				},
				Risk: RiskSpec{Level: "low"},
			},
		},
	}

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected one action definition, got %d", len(defs))
	}

	if defs[0].Auth.Type != actions.AuthTypeQuery {
		t.Fatalf("expected action-level query auth, got %s", defs[0].Auth.Type)
	}
	if defs[0].Auth.QueryName != "api_key" || defs[0].Auth.CredentialRef != "svc.api_key" {
		t.Fatalf("unexpected overridden auth mapping: %+v", defs[0].Auth)
	}
}

func TestMapAuthBodyUsesBodyField(t *testing.T) {
	m := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth: ServiceAuth{
			Type:          "body",
			CredentialRef: "svc.token",
			BodyField:     "api_token",
		},
		Actions: map[string]ServiceAction{
			"post": {Method: "POST", Path: "/v1/post", Risk: RiskSpec{Level: "low"}},
		},
	}
	defs, err := ToActionDefinitions(m)
	if err != nil {
		t.Fatalf("ToActionDefinitions error: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("expected at least one action definition")
	}
	if defs[0].Auth.BodyField != "api_token" {
		t.Errorf("expected Auth.BodyField=api_token, got %q", defs[0].Auth.BodyField)
	}
	if defs[0].Auth.QueryName != "" {
		t.Errorf("expected Auth.QueryName to be empty for body auth, got %q", defs[0].Auth.QueryName)
	}
}

func TestToActionDefinitionsBearerAuth(t *testing.T) {
	m := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "bearer", CredentialRef: "svc.token"},
		Actions: map[string]ServiceAction{
			"get": {Method: "GET", Path: "/v1/items", Risk: RiskSpec{Level: "low"}},
		},
	}
	defs, err := ToActionDefinitions(m)
	if err != nil {
		t.Fatalf("ToActionDefinitions: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("expected at least one action")
	}
	if defs[0].Auth.Prefix != "Bearer" {
		t.Errorf("expected Auth.Prefix=Bearer, got %q", defs[0].Auth.Prefix)
	}
}

func TestToActionDefinitionsBasicAuth(t *testing.T) {
	m := &ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "basic", CredentialRef: "svc.creds"},
		Actions: map[string]ServiceAction{
			"get": {Method: "GET", Path: "/v1/items", Risk: RiskSpec{Level: "low"}},
		},
	}
	defs, err := ToActionDefinitions(m)
	if err != nil {
		t.Fatalf("ToActionDefinitions: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("expected at least one action")
	}
	if defs[0].Auth.Type != actions.AuthTypeBasic {
		t.Errorf("expected Auth.Type=basic, got %v", defs[0].Auth.Type)
	}
}

func TestGenerateAgentSkillMDIncludesActionLevelCredentials(t *testing.T) {
	manifest := &ServiceManifest{
		Name:    "multi-auth",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"search": {
				Method: "GET",
				Path:   "/search",
				Auth: &ServiceAuth{
					Type:          "query",
					QueryParam:    "api_key",
					CredentialRef: "multi-auth.api_key",
				},
				Risk: RiskSpec{Level: "low"},
			},
		},
	}

	content, err := GenerateAgentSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMD: %v", err)
	}
	if !strings.Contains(content, "kimbap vault set multi-auth.api_key") {
		t.Error("GenerateAgentSkillMD must list action-level credential refs in prerequisites")
	}
}

func TestSignAndVerifyRoundtrip(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install: %v", err)
	}

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	if err := installer.Sign(priv); err != nil {
		t.Fatalf("sign: %v", err)
	}

	result, err := installer.VerifyWithKey("brave-search", pub)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !result.Verified {
		t.Fatal("expected digest verified")
	}
	if !result.Signed {
		t.Fatal("expected signed")
	}
	if !result.SignatureValid {
		t.Fatal("expected valid signature with pinned key")
	}
}

func TestVerifyFailsOnMalformedEmbeddedPublicKey(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install: %v", err)
	}

	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	if err := installer.Sign(priv); err != nil {
		t.Fatalf("sign: %v", err)
	}

	lf, err := installer.readLockfile()
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	lf.PublicKey = "not-hex"
	if err := installer.writeLockfile(lf); err != nil {
		t.Fatalf("write malformed lockfile: %v", err)
	}

	result, err := installer.Verify("brave-search")
	if err == nil {
		t.Fatal("expected malformed embedded public key to fail verify")
	}
	if result == nil {
		t.Fatal("expected verify result alongside error")
	}
	if !strings.Contains(err.Error(), "decode embedded public key") {
		t.Fatalf("unexpected verify error: %v", err)
	}
	if result.SignatureValid {
		t.Fatal("expected invalid signature state for malformed embedded public key")
	}
}

func TestVerifyWithWrongPinnedKeyFails(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install: %v", err)
	}

	_, priv, _ := ed25519.GenerateKey(nil)
	if err := installer.Sign(priv); err != nil {
		t.Fatalf("sign: %v", err)
	}

	wrongPub, _, _ := ed25519.GenerateKey(nil)
	result, err := installer.VerifyWithKey("brave-search", wrongPub)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !result.Verified {
		t.Fatal("digest should still be verified")
	}
	if !result.Signed {
		t.Fatal("entry should be marked signed")
	}
	if result.SignatureValid {
		t.Fatal("signature should be invalid with wrong pinned key")
	}
}

func TestGenerateAgentSkillMDContainsExpectedSections(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	content, err := GenerateAgentSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMD: %v", err)
	}

	checks := []string{
		"name: brave-search",
		"allowed-tools: Bash",
		"## Available Actions",
		"### brave-search.web_search",
		"kimbap call brave-search.web_search",
		"## Discovery",
		"kimbap actions list --service brave-search --format json",
		"kimbap actions describe brave-search.<action> --format json",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("GenerateAgentSkillMD output missing %q", want)
		}
	}
}

func TestGenerateAgentSkillMDAppleScript(t *testing.T) {
	manifest, err := ParseManifest([]byte(actionWithCommandFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	content, err := GenerateAgentSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMD: %v", err)
	}

	checks := []string{
		"Use when you need to control Notes via AppleScript.",
		"**Command**: `list-notes`",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("GenerateAgentSkillMD AppleScript output missing %q", want)
		}
	}
	if strings.Contains(content, "**HTTP**:") {
		t.Fatalf("GenerateAgentSkillMD AppleScript output must not contain HTTP label:\n%s", content)
	}
}

func TestGenerateAgentSkillMDHTTPUnchanged(t *testing.T) {
	manifest, err := ParseManifest([]byte(manifestWithoutAdapterFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	content, err := GenerateAgentSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMD: %v", err)
	}

	expected := "---\n" +
		"name: notes-service\n" +
		"description: |\n" +
		"  Notes service\n" +
		"  Use when you need to interact with the notes-service API.\n" +
		"allowed-tools: Bash\n" +
		"---\n\n" +
		"# notes-service\n\n" +
		"Notes service\n\n" +
		"## Prerequisites\n\n" +
		"- Kimbap CLI installed and in PATH\n" +
		"- Service installed: `kimbap service install notes-service.yaml`\n\n" +
		"## Available Actions\n\n" +
		"### notes-service.list_notes\n\n" +
		"**HTTP**: `GET /notes`\n" +
		"**Risk**: low\n\n" +
		"**Usage**:\n" +
		"```bash\n" +
		"kimbap call notes-service.list_notes\n" +
		"```\n\n" +
		"## Discovery\n\n" +
		"```bash\n" +
		"kimbap actions list --service notes-service --format json\n" +
		"kimbap actions describe notes-service.<action> --format json\n" +
		"kimbap call notes-service.<action> --dry-run --format json\n" +
		"```\n"

	if content != expected {
		t.Fatalf("HTTP SKILL.md output changed unexpectedly:\nexpected:\n%s\nactual:\n%s", expected, content)
	}
}

func TestGenerateAgentSkillPackEscapesMarkdownTableCells(t *testing.T) {
	manifest := &ServiceManifest{
		Name:        "notes-service",
		Version:     "1.0.0",
		Description: "Notes service",
		BaseURL:     "https://example.com",
		Auth:        ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"list_notes": {
				Method:      "GET",
				Path:        "/notes",
				Description: "List notes | summarize\nSecond line",
				Risk:        RiskSpec{Level: "low"},
			},
		},
	}

	pack, err := GenerateAgentSkillPack(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillPack: %v", err)
	}
	skill := pack["SKILL.md"]
	if !strings.Contains(skill, "| `notes-service.list_notes` | List notes \\| summarize<br>Second line | low |") {
		t.Fatalf("expected markdown table cells escaped, got:\n%s", skill)
	}
}

func TestGenerateAgentSkillPack(t *testing.T) {
	manifest := &ServiceManifest{
		Name:        "notes-service",
		Version:     "1.0.0",
		Description: "Notes service",
		BaseURL:     "https://example.com",
		Auth:        ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"list_notes": {
				Method:      "GET",
				Path:        "/notes",
				Description: "List notes",
				Risk:        RiskSpec{Level: "low"},
				Warnings:    []string{"May return large result sets"},
			},
		},
		Gotchas: []ServiceGotcha{
			{Symptom: "429 responses", LikelyCause: "Rate limit", Recovery: "Retry with backoff", Severity: "medium"},
			{Symptom: "Auth failures", LikelyCause: "Missing token", Recovery: "Set credential ref"},
		},
		Recipes: []ServiceRecipe{{
			Name:        "List and summarize",
			Description: "Fetch notes then summarize",
			Steps:       []string{"Run list action", "Summarize results"},
		}},
		Triggers: &TriggerConfig{
			TaskVerbs:  []string{"list", "summarize"},
			Objects:    []string{"notes"},
			InsteadOf:  []string{"calling notes API directly"},
			Exclusions: []string{"editing local files"},
		},
	}

	pack, err := GenerateAgentSkillPack(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillPack: %v", err)
	}
	if len(pack) != 3 {
		t.Fatalf("expected 3 pack files, got %d", len(pack))
	}

	skill := pack["SKILL.md"]
	if !strings.Contains(skill, "## Files in This Pack") {
		t.Fatalf("SKILL.md missing pack file section:\n%s", skill)
	}
	if !strings.Contains(skill, "GOTCHAS.md") || !strings.Contains(skill, "RECIPES.md") {
		t.Fatalf("SKILL.md missing pack file references:\n%s", skill)
	}
	if !strings.Contains(skill, "Use when you need to list, summarize notes through approved Kimbap actions.") {
		t.Fatalf("SKILL.md missing trigger-based description:\n%s", skill)
	}

	gotchas := pack["GOTCHAS.md"]
	if !strings.Contains(gotchas, "## Service-Level Gotchas") || !strings.Contains(gotchas, "## Action-Specific Warnings") {
		t.Fatalf("GOTCHAS.md missing expected sections:\n%s", gotchas)
	}

	recipes := pack["RECIPES.md"]
	if !strings.Contains(recipes, "## List and summarize") || !strings.Contains(recipes, "1. Run list action") {
		t.Fatalf("RECIPES.md missing expected content:\n%s", recipes)
	}

	nilResult, nilErr := GenerateAgentSkillPack(nil)
	if nilErr == nil || nilResult != nil {
		t.Fatalf("expected nil manifest to return error and nil result, got result=%v err=%v", nilResult, nilErr)
	}
}

func TestManifestGotchasParsing(t *testing.T) {
	manifest := `name: github
version: 1.0.0
description: GitHub API integration
base_url: https://api.github.com
auth:
  type: bearer
  credential_ref: github.token
triggers:
  task_verbs: [list, create]
  objects: [issues, pull requests]
gotchas:
  - symptom: 422 validation failed
    likely_cause: payload shape mismatch
    recovery: validate body schema before retrying
    severity: medium
recipes:
  - name: Create issue safely
    description: Validate, then create an issue
    steps:
      - Validate payload
      - Call create issue action
actions:
  list_issues:
    method: GET
    path: /issues
    risk:
      level: low
`

	m, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(m.Gotchas) != 1 {
		t.Fatalf("expected 1 gotcha, got %d", len(m.Gotchas))
	}
	if m.Gotchas[0].LikelyCause != "payload shape mismatch" {
		t.Fatalf("unexpected gotcha likely_cause: %q", m.Gotchas[0].LikelyCause)
	}
	if m.Triggers == nil || len(m.Triggers.TaskVerbs) != 2 {
		t.Fatalf("expected triggers parsed, got %+v", m.Triggers)
	}
	if len(m.Recipes) != 1 || len(m.Recipes[0].Steps) != 2 {
		t.Fatalf("expected recipes parsed, got %+v", m.Recipes)
	}
}

func TestManifestBackwardCompat(t *testing.T) {
	manifest := `name: legacy-notes
version: 1.0.0
description: Legacy notes API manifest without pack metadata
base_url: https://api.example.com
auth:
  type: none
actions:
  list_notes:
    method: GET
    path: /notes
    description: List notes
    risk:
      level: low
`

	m, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(m.Gotchas) != 0 {
		t.Fatalf("expected no gotchas for legacy manifest, got %+v", m.Gotchas)
	}
	if m.Triggers != nil {
		t.Fatalf("expected nil triggers for legacy manifest, got %+v", m.Triggers)
	}
	if len(m.Recipes) != 0 {
		t.Fatalf("expected no recipes for legacy manifest, got %+v", m.Recipes)
	}

	content, genErr := GenerateAgentSkillMD(m)
	if genErr != nil {
		t.Fatalf("GenerateAgentSkillMD: %v", genErr)
	}
	if !strings.Contains(content, "### legacy-notes.list_notes") {
		t.Fatalf("expected legacy action output unchanged, got:\n%s", content)
	}
}

func TestManifestPackMetadataValidation(t *testing.T) {
	manifestYAML := []byte(`name: invalid-pack
version: 1.0.0
description: invalid pack metadata fixture
base_url: https://api.example.com
auth:
  type: bearer
  credential_ref: token
triggers:
  task_verbs: []
  objects:
    - ""
gotchas:
  - symptom: ""
    likely_cause: ""
    recovery: ""
    severity: impossible
recipes:
  - name: ""
    steps:
      - ""
actions:
  list:
    method: GET
    path: /items
    description: list items
    warnings:
      - ""
    args: []
    request: {}
    response:
      type: array
    risk:
      level: low
`)

	_, err := ParseManifest(manifestYAML)
	if err == nil {
		t.Fatal("expected parse to fail for invalid pack metadata")
	}

	errText := err.Error()
	checks := []string{
		"triggers.task_verbs",
		"triggers.objects[0]",
		"gotchas[0].symptom",
		"gotchas[0].likely_cause",
		"gotchas[0].recovery",
		"gotchas[0].severity",
		"recipes[0].name",
		"recipes[0].steps[0]",
		"actions.list.warnings[0]",
	}
	for _, want := range checks {
		if !strings.Contains(errText, want) {
			t.Fatalf("expected validation error to include %q, got: %s", want, errText)
		}
	}
}

func TestGenerateMetaAgentSkillPack(t *testing.T) {
	pack := GenerateMetaAgentSkillPack()
	if len(pack) != 1 {
		t.Fatalf("expected exactly one file in meta pack, got %d", len(pack))
	}
	skill, ok := pack["SKILL.md"]
	if !ok {
		t.Fatal("expected SKILL.md in meta pack")
	}
	if skill != GenerateMetaAgentSkillMD() {
		t.Fatal("expected meta pack SKILL.md to match GenerateMetaAgentSkillMD output")
	}
	if !strings.Contains(skill, "name: kimbap") {
		t.Fatalf("meta SKILL.md missing expected frontmatter:\n%s", skill)
	}
}

func TestGenerateAgentSkillMDCommand(t *testing.T) {
	manifest := &ServiceManifest{
		Name:    "ffmpeg",
		Adapter: "command",
		Actions: map[string]ServiceAction{
			"convert": {
				Command:     "ffmpeg -i {{input}} {{output}}",
				Description: "Convert media file",
				Risk:        RiskSpec{Level: "low"},
			},
		},
	}
	content, err := GenerateAgentSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMD: %v", err)
	}
	if !strings.Contains(content, "**Command**:") {
		t.Errorf("command adapter must emit **Command** label, got:\n%s", content)
	}
	if strings.Contains(content, "**HTTP**:") {
		t.Errorf("command adapter must not emit **HTTP** label, got:\n%s", content)
	}
	if !strings.Contains(content, "Use when you need to run ffmpeg commands.") {
		t.Errorf("command adapter description missing, got:\n%s", content)
	}
}

func TestBuildAgentSkillDescriptionConditionalTriggerPhrases(t *testing.T) {
	noDesc := &ServiceManifest{
		Name: "svc",
		Actions: map[string]ServiceAction{
			"do": {Method: "GET", Path: "/do", Risk: RiskSpec{Level: "low"}},
		},
	}
	got := buildAgentSkillDescription(noDesc)
	if strings.Contains(got, "Trigger phrases:") {
		t.Errorf("expected no 'Trigger phrases:' when no action has a description, got:\n%s", got)
	}

	withDesc := &ServiceManifest{
		Name: "svc",
		Actions: map[string]ServiceAction{
			"search": {Method: "GET", Path: "/search", Description: "Search for items", Risk: RiskSpec{Level: "low"}},
		},
	}
	got = buildAgentSkillDescription(withDesc)
	if !strings.Contains(got, "Trigger phrases:") {
		t.Errorf("expected 'Trigger phrases:' when action has description, got:\n%s", got)
	}
	if !strings.Contains(got, `"search": Search for items`) {
		t.Errorf("expected humanized trigger line, got:\n%s", got)
	}
}

func TestGenerateAgentSkillPackBeforeExecuteVariants(t *testing.T) {
	base := ServiceManifest{
		Name:    "svc",
		Version: "1.0.0",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"do": {Method: "GET", Path: "/do", Risk: RiskSpec{Level: "low"}},
		},
	}
	tests := []struct {
		name        string
		gotchas     []ServiceGotcha
		recipes     []ServiceRecipe
		wantGotchas bool
		wantRecipes bool
	}{
		{name: "neither", wantGotchas: false, wantRecipes: false},
		{
			name:        "gotchas only",
			gotchas:     []ServiceGotcha{{Symptom: "err", LikelyCause: "x", Recovery: "y"}},
			wantGotchas: true, wantRecipes: false,
		},
		{
			name:        "recipes only",
			recipes:     []ServiceRecipe{{Name: "r", Steps: []string{"step 1"}}},
			wantGotchas: false, wantRecipes: true,
		},
		{
			name:        "both",
			gotchas:     []ServiceGotcha{{Symptom: "err", LikelyCause: "x", Recovery: "y"}},
			recipes:     []ServiceRecipe{{Name: "r", Steps: []string{"step 1"}}},
			wantGotchas: true, wantRecipes: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := base
			m.Gotchas = tt.gotchas
			m.Recipes = tt.recipes
			pack, err := GenerateAgentSkillPack(&m)
			if err != nil {
				t.Fatalf("GenerateAgentSkillPack: %v", err)
			}
			skill := pack["SKILL.md"]
			if tt.wantGotchas {
				if !strings.Contains(skill, "Read GOTCHAS.md") {
					t.Errorf("expected GOTCHAS.md line in Before Execute:\n%s", skill)
				}
			} else {
				if strings.Contains(skill, "Read GOTCHAS.md") {
					t.Errorf("unexpected GOTCHAS.md line when no gotchas:\n%s", skill)
				}
			}
			if tt.wantRecipes {
				if !strings.Contains(skill, "Read RECIPES.md") {
					t.Errorf("expected RECIPES.md line in Before Execute:\n%s", skill)
				}
			} else {
				if strings.Contains(skill, "Read RECIPES.md") {
					t.Errorf("unexpected RECIPES.md line when no recipes:\n%s", skill)
				}
			}
		})
	}
}

func TestBuildPackDescriptionTriggerlessFallback(t *testing.T) {
	tests := []struct {
		name     string
		triggers *TriggerConfig
		wantSub  string
	}{
		{name: "nil triggers", triggers: nil, wantSub: "Use for approved svc actions through Kimbap."},
		{name: "empty triggers", triggers: &TriggerConfig{}, wantSub: "Use for approved svc actions through Kimbap."},
		{name: "verbs without objects", triggers: &TriggerConfig{TaskVerbs: []string{"list"}}, wantSub: "Use for approved svc actions through Kimbap."},
		{name: "full triggers", triggers: &TriggerConfig{TaskVerbs: []string{"list"}, Objects: []string{"items"}}, wantSub: "Use when you need to list items through approved Kimbap actions."},
		{name: "instead_of only", triggers: &TriggerConfig{InsteadOf: []string{"direct API calls"}}, wantSub: "Use instead of: direct API calls."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ServiceManifest{
				Name:     "svc",
				Triggers: tt.triggers,
				Actions: map[string]ServiceAction{
					"do": {Method: "GET", Path: "/do", Risk: RiskSpec{Level: "low"}},
				},
			}
			got := buildPackDescription(m)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("buildPackDescription() = %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

func TestGenerateAgentSkillMDRiskLevelHints(t *testing.T) {
	cases := []struct {
		riskLevel        string
		wantDryRunHint   bool
		wantApprovalHint bool
	}{
		{"low", false, false},
		{"medium", true, false},
		{"high", true, true},
		{"critical", true, true},
		{"", true, false},
	}

	for _, tc := range cases {
		t.Run("risk_"+tc.riskLevel, func(t *testing.T) {
			m := &ServiceManifest{
				Name:    "test",
				Version: "1.0.0",
				BaseURL: "https://example.com",
				Auth:    ServiceAuth{Type: "none"},
				Actions: map[string]ServiceAction{
					"do_it": {
						Method: "POST",
						Path:   "/do",
						Risk:   RiskSpec{Level: tc.riskLevel},
					},
				},
			}
			out, err := GenerateAgentSkillMD(m)
			if err != nil {
				t.Fatalf("GenerateAgentSkillMD error: %v", err)
			}
			hasDryRun := strings.Contains(out, "--dry-run --format json first to preview")
			hasApproval := strings.Contains(out, "kimbap approve list")
			if hasDryRun != tc.wantDryRunHint {
				t.Errorf("risk=%q: dry-run hint = %v, want %v\nOutput:\n%s", tc.riskLevel, hasDryRun, tc.wantDryRunHint, out)
			}
			if hasApproval != tc.wantApprovalHint {
				t.Errorf("risk=%q: approval hint = %v, want %v\nOutput:\n%s", tc.riskLevel, hasApproval, tc.wantApprovalHint, out)
			}
		})
	}
}

func TestBuildSkillDescriptionHumanizesActionKeys(t *testing.T) {
	m := &ServiceManifest{
		Name: "search",
		Actions: map[string]ServiceAction{
			"web_search": {Description: "Search the web"},
		},
	}
	desc := buildAgentSkillDescription(m)
	if strings.Contains(desc, "web_search") {
		t.Error("expected humanized action key 'web search', got raw 'web_search'")
	}
	if !strings.Contains(desc, "web search") {
		t.Errorf("expected 'web search' in description, got:\n%s", desc)
	}
}

func TestGenerateAgentSkillMDCriticalRisk(t *testing.T) {
	manifest := &ServiceManifest{
		Name:    "critical-svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth: ServiceAuth{
			Type: "none",
		},
		Actions: map[string]ServiceAction{
			"destroy": {
				Method: "DELETE",
				Path:   "/items",
				Risk:   RiskSpec{Level: "critical"},
			},
		},
	}

	content, err := GenerateAgentSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMD: %v", err)
	}

	if !strings.Contains(content, "⚠️ This action is risk level: critical. Use --dry-run --format json first to preview.") {
		t.Fatalf("expected critical dry-run warning, got:\n%s", content)
	}
	if !strings.Contains(content, "🔒 Approval may be required. Check: `kimbap approve list`") {
		t.Fatalf("expected approval hint for critical risk, got:\n%s", content)
	}
}

func TestToActionDefinitionsCriticalRiskRequiresApprovalHint(t *testing.T) {
	manifest := &ServiceManifest{
		Name:    "critical-svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"destroy": {
				Method: "DELETE",
				Path:   "/items",
				Risk:   RiskSpec{Level: "critical"},
			},
		},
	}

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected one action definition, got %d", len(defs))
	}
	if defs[0].ApprovalHint != actions.ApprovalRequired {
		t.Fatalf("expected approval hint %q, got %q", actions.ApprovalRequired, defs[0].ApprovalHint)
	}
}

func TestToActionDefinitionsRespectsExplicitHTTPIdempotentOverride(t *testing.T) {
	idempotent := true
	manifest := &ServiceManifest{
		Name:    "http-service",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"create": {
				Method:     "POST",
				Path:       "/items",
				Risk:       RiskSpec{Level: "medium"},
				Idempotent: &idempotent,
			},
		},
	}

	defs, err := ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected one action definition, got %d", len(defs))
	}
	if !defs[0].Idempotent {
		t.Fatal("expected explicit HTTP idempotent override to be preserved")
	}
}

func TestGenerateAgentSkillMDNormalizesAdapterAndAuthTypes(t *testing.T) {
	manifest := &ServiceManifest{
		Name:        "apple-notes",
		Version:     "1.0.0",
		Adapter:     " AppleScript ",
		TargetApp:   "Notes",
		Description: "Manage Apple Notes",
		Auth: ServiceAuth{
			Type:          " NONE ",
			CredentialRef: "should-not-appear",
		},
		Actions: map[string]ServiceAction{
			"list_notes": {
				Command:     "list-notes",
				Description: "List notes",
				Risk:        RiskSpec{Level: "low"},
			},
		},
	}

	content, err := GenerateAgentSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMD: %v", err)
	}
	if !strings.Contains(content, "Use when you need to control Notes via AppleScript.") {
		t.Fatalf("expected AppleScript description, got:\n%s", content)
	}
	if !strings.Contains(content, "**Command**: `list-notes`") {
		t.Fatalf("expected AppleScript command block, got:\n%s", content)
	}
	if strings.Contains(content, "Credential configured:") {
		t.Fatalf("did not expect credential prerequisite for auth type none, got:\n%s", content)
	}
}

func TestGenerateMetaAgentSkillMDContainsServiceActionSyntax(t *testing.T) {
	content := GenerateMetaAgentSkillMD()

	checks := []string{
		"name: kimbap",
		"allowed-tools: Bash",
		"<when_to_use>",
		"<protocol>",
		"kimbap search",
		"kimbap actions list --format json",
		"kimbap actions describe <service.action> --format json",
		"kimbap call <service.action>",
		"<troubleshooting>",
		"<rules>",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("GenerateMetaAgentSkillMD output missing %q", want)
		}
	}

	if strings.Contains(content, "kimbap actions describe <action>") {
		t.Error("GenerateMetaAgentSkillMD must not use bare <action> — use <service.action>")
	}
}

func TestVerifyTamperedDigestDetected(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	dir := t.TempDir()
	installer := NewLocalInstaller(dir)
	if _, err := installer.Install(manifest, "local"); err != nil {
		t.Fatalf("install: %v", err)
	}

	lf, _ := installer.readLockfile()
	entry := lf.Services["brave-search"]
	entry.Digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	lf.Services["brave-search"] = entry
	_ = installer.writeLockfile(lf)

	result, err := installer.Verify("brave-search")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if result.Verified {
		t.Fatal("tampered digest should fail verification")
	}
}

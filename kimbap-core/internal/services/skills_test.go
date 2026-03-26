package services

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap-core/internal/actions"
)

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
    risk:
      level: low
      mutating: false
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
		t.Fatalf("expected skill file to exist: %v", err)
	}
	if installed.Manifest.Name != "brave-search" {
		t.Fatalf("unexpected installed skill name: %s", installed.Manifest.Name)
	}

	list, err := installer.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 installed skill, got %d", len(list))
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
		t.Fatalf("tamper skill file failed: %v", err)
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
      mutating: true
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

func TestGenerateSkillMDIncludesActionLevelCredentials(t *testing.T) {
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

	content, err := GenerateSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateSkillMD: %v", err)
	}
	if !strings.Contains(content, "kimbap vault set multi-auth.api_key") {
		t.Error("GenerateSkillMD must list action-level credential refs in prerequisites")
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

func TestGenerateSkillMDContainsExpectedSections(t *testing.T) {
	manifest, err := ParseManifest([]byte(braveSearchFixture))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	content, err := GenerateSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateSkillMD: %v", err)
	}

	checks := []string{
		"name: brave-search",
		"allowed-tools: Bash",
		"## Prerequisites",
		"kimbap service install brave-search.yaml",
		"## Available Actions",
		"### brave-search.web_search",
		"kimbap call brave-search.web_search",
		"## Discovery",
		"kimbap actions list --service brave-search --format json",
		"kimbap actions describe brave-search.<action> --format json",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("GenerateSkillMD output missing %q", want)
		}
	}
}

func TestGenerateSkillMDRiskLevelHints(t *testing.T) {
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
			out, err := GenerateSkillMD(m)
			if err != nil {
				t.Fatalf("GenerateSkillMD error: %v", err)
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
	desc := buildSkillDescription(m)
	if strings.Contains(desc, "web_search") {
		t.Error("expected humanized action key 'web search', got raw 'web_search'")
	}
	if !strings.Contains(desc, "web search") {
		t.Errorf("expected 'web search' in description, got:\n%s", desc)
	}
}

func TestGenerateSkillMDCriticalRisk(t *testing.T) {
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

	content, err := GenerateSkillMD(manifest)
	if err != nil {
		t.Fatalf("GenerateSkillMD: %v", err)
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

func TestGenerateMetaSkillMDContainsServiceActionSyntax(t *testing.T) {
	content := GenerateMetaSkillMD()

	checks := []string{
		"name: kimbap",
		"allowed-tools: Bash",
		"kimbap actions list --format json",
		"kimbap actions describe <service.action> --format json",
		"kimbap call <service>.<action>",
		"kimbap service list",
		"## Decision Protocol",
		"## Troubleshooting",
		"kimbap agents setup",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("GenerateMetaSkillMD output missing %q", want)
		}
	}

	if strings.Contains(content, "kimbap actions describe <action>") {
		t.Error("GenerateMetaSkillMD must not use bare <action> — use <service.action>")
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

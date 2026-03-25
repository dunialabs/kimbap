package skills

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
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
	manifest := &SkillManifest{
		Name:    "Invalid Name",
		Version: "x",
		BaseURL: "not-a-url",
		Auth: SkillAuth{
			Type:          "header",
			CredentialRef: "",
		},
		Actions: map[string]SkillAction{
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
	if err != ErrSkillAlreadyInstalled {
		t.Fatalf("expected ErrSkillAlreadyInstalled, got %v", err)
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

	lf, err := installer.readLockfile()
	if err != nil {
		t.Fatalf("read lockfile failed: %v", err)
	}
	entry, ok := lf.Skills["brave-search"]
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
	if _, ok := lf.Skills["brave-search"]; ok {
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
	manifest := &SkillManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth: SkillAuth{
			Type:          "header",
			HeaderName:    "Authorization",
			CredentialRef: "svc.token",
		},
		Actions: map[string]SkillAction{
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
	manifest := &SkillManifest{
		Name:    "public-api",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    SkillAuth{Type: "none"},
		Actions: map[string]SkillAction{
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
	manifest := &SkillManifest{
		Name:    "svc",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth: SkillAuth{
			Type:          "header",
			HeaderName:    "Authorization",
			CredentialRef: "svc.token",
		},
		Actions: map[string]SkillAction{
			"search": {
				Method: "GET",
				Path:   "/search",
				Auth: &SkillAuth{
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

func TestGenerateSkillMDIncludesActionLevelCredentials(t *testing.T) {
	manifest := &SkillManifest{
		Name:    "multi-auth",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    SkillAuth{Type: "none"},
		Actions: map[string]SkillAction{
			"search": {
				Method: "GET",
				Path:   "/search",
				Auth: &SkillAuth{
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
		"## Available Actions",
		"### brave-search.web_search",
		"kimbap call brave-search.web_search",
		"## Discovery",
		"kimbap actions list --service brave-search",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("GenerateSkillMD output missing %q", want)
		}
	}
}

func TestGenerateMetaSkillMDContainsServiceActionSyntax(t *testing.T) {
	content := GenerateMetaSkillMD()

	checks := []string{
		"name: kimbap",
		"allowed-tools: Bash",
		"kimbap actions list",
		"kimbap actions describe <service.action>",
		"kimbap call <service>.<action>",
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
	entry := lf.Skills["brave-search"]
	entry.Digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	lf.Skills["brave-search"] = entry
	_ = installer.writeLockfile(lf)

	result, err := installer.Verify("brave-search")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if result.Verified {
		t.Fatal("tampered digest should fail verification")
	}
}

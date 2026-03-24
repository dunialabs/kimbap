package skills

import (
	"os"
	"path/filepath"
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

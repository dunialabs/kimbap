package providers_test

import (
	"testing"
	"testing/fstest"

	"github.com/dunialabs/kimbap-core/internal/connectors/providers"
)

var githubYAML = []byte(`
id: github
display_name: GitHub (test)
supported_flows: [browser, device]
auth_endpoint: https://github.com/login/oauth/authorize
token_endpoint: https://github.com/login/oauth/access_token
device_endpoint: https://github.com/login/device/code
auth_lanes: [public-client, byo]
embedded_client_id: test-client-id
token_exchange:
  auth_method: body
`)

func TestLoadProvider_YAMLPrecedence(t *testing.T) {
	testFS := fstest.MapFS{
		"official/github.yaml": {Data: githubYAML},
	}

	def, err := providers.LoadProvider("github", testFS)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if def.DisplayName != "GitHub (test)" {
		t.Errorf("expected YAML display name 'GitHub (test)', got %q (should not be hardcoded value)", def.DisplayName)
	}
}

func TestLoadProvider_UnknownProviderReturnsError(t *testing.T) {
	testFS := fstest.MapFS{}
	_, err := providers.LoadProvider("unknown-provider", testFS)
	if err == nil {
		t.Fatal("expected error for unknown provider in empty FS, got nil")
	}
}

func TestLoadAllProviders_YAMLWins(t *testing.T) {
	testFS := fstest.MapFS{
		"official/github.yaml": {Data: githubYAML},
	}

	providersList, err := providers.LoadAllProviders(testFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(providersList) == 0 {
		t.Fatal("expected at least one provider")
	}

	found := false
	for _, p := range providersList {
		if p.ID == "github" && p.DisplayName == "GitHub (test)" {
			found = true
		}
	}

	if !found {
		t.Error("expected github from YAML (display name 'GitHub (test)') to win over hardcoded")
	}
}

func TestLoadProvider_GitHub_Parity(t *testing.T) {
	def, err := providers.LoadProvider("github", providers.EmbeddedProviders)
	if err != nil {
		t.Fatalf("YAML load failed: %v", err)
	}
	if def.AuthEndpoint != "https://github.com/login/oauth/authorize" {
		t.Errorf("unexpected auth_endpoint: %q", def.AuthEndpoint)
	}
	if def.TokenEndpoint != "https://github.com/login/oauth/access_token" {
		t.Errorf("unexpected token_endpoint: %q", def.TokenEndpoint)
	}
	if len(def.SupportedFlows) != 2 {
		t.Errorf("expected 2 supported flows, got %d", len(def.SupportedFlows))
	}
}

func TestLoadProvider_Slack_Parity(t *testing.T) {
	def, err := providers.LoadProvider("slack", providers.EmbeddedProviders)
	if err != nil {
		t.Fatalf("YAML load failed: %v", err)
	}
	if def.AuthEndpoint != "https://slack.com/oauth/v2/authorize" {
		t.Errorf("unexpected auth_endpoint: %q", def.AuthEndpoint)
	}
	if def.TokenEndpoint != "https://slack.com/api/oauth.v2.access" {
		t.Errorf("unexpected token_endpoint: %q", def.TokenEndpoint)
	}
}

func assertProviderLoadsFromYAML(t *testing.T, providerID string) {
	t.Helper()
	def, err := providers.LoadProvider(providerID, providers.EmbeddedProviders)
	if err != nil {
		t.Fatalf("YAML load failed for %s: %v", providerID, err)
	}
	if def.ID != providerID {
		t.Errorf("id mismatch: expected %q, got %q", providerID, def.ID)
	}
	if def.DisplayName == "" {
		t.Errorf("display_name is empty for %s", providerID)
	}
	if def.AuthEndpoint == "" {
		t.Errorf("auth_endpoint is empty for %s", providerID)
	}
	if def.TokenEndpoint == "" {
		t.Errorf("token_endpoint is empty for %s", providerID)
	}
	if len(def.SupportedFlows) == 0 {
		t.Errorf("supported_flows is empty for %s", providerID)
	}
	if len(def.AuthLanes) == 0 {
		t.Errorf("auth_lanes is empty for %s", providerID)
	}
}

func TestLoadProvider_Google_Parity(t *testing.T) {
	assertProviderLoadsFromYAML(t, "google")
}

func TestLoadProvider_Notion_Parity(t *testing.T) {
	assertProviderLoadsFromYAML(t, "notion")
}

func TestLoadProvider_Figma_Parity(t *testing.T) {
	assertProviderLoadsFromYAML(t, "figma")
}

func TestLoadProvider_Stripe_Parity(t *testing.T) {
	assertProviderLoadsFromYAML(t, "stripe")
}

func TestLoadProvider_Zendesk_Parity(t *testing.T) {
	assertProviderLoadsFromYAML(t, "zendesk")
}

func TestLoadProvider_Canvas_Parity(t *testing.T) {
	assertProviderLoadsFromYAML(t, "canvas")
}

func TestLoadProvider_Canva_Parity(t *testing.T) {
	assertProviderLoadsFromYAML(t, "canva")
}

func TestLoadAllProviders_AllNine(t *testing.T) {
	all, err := providers.LoadAllProviders(providers.EmbeddedProviders)
	if err != nil {
		t.Fatalf("LoadAllProviders failed: %v", err)
	}
	if len(all) != 9 {
		ids := make([]string, len(all))
		for i, p := range all {
			ids[i] = p.ID
		}
		t.Errorf("expected 9 providers, got %d: %v", len(all), ids)
	}
}

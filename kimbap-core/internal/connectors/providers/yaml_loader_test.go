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

func TestLoadProvider_HardcodedFallback(t *testing.T) {
	testFS := fstest.MapFS{}

	def, err := providers.LoadProvider("github", testFS)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if def.ID != "github" {
		t.Errorf("expected github from hardcoded fallback, got %q", def.ID)
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

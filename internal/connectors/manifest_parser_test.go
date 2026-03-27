package connectors_test

import (
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/connectors"
)

var validGitHubManifestYAML = []byte(`
id: github
display_name: GitHub
supported_flows:
  - browser
  - device
auth_endpoint: https://github.com/login/oauth/authorize
token_endpoint: https://github.com/login/oauth/access_token
device_endpoint: https://github.com/login/device/code
user_info_endpoint: https://api.github.com/user
default_scopes:
  - repo
  - read:org
auth_lanes:
  - public-client
  - byo
embedded_client_id: 178c6fc778ccc68e1d6a
token_exchange:
  auth_method: body
`)

func TestParseProviderManifest_Valid(t *testing.T) {
	m, err := connectors.ParseProviderManifest(validGitHubManifestYAML)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if m.ID != "github" {
		t.Errorf("expected ID=github, got %q", m.ID)
	}
	if len(m.SupportedFlows) != 2 {
		t.Errorf("expected 2 supported flows, got %d", len(m.SupportedFlows))
	}
	if m.AuthEndpoint != "https://github.com/login/oauth/authorize" {
		t.Errorf("unexpected auth_endpoint: %q", m.AuthEndpoint)
	}
}

func TestParseProviderManifest_MissingAuthEndpoint(t *testing.T) {
	yaml := []byte(`
id: test
display_name: Test
supported_flows: [browser]
token_endpoint: https://example.com/token
auth_lanes: [byo]
`)
	_, err := connectors.ParseProviderManifest(yaml)
	if err == nil {
		t.Fatal("expected error for missing auth_endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "auth_endpoint") {
		t.Errorf("expected error to mention auth_endpoint, got: %v", err)
	}
}

func TestParseProviderManifest_InvalidAuthLane(t *testing.T) {
	yaml := []byte(`
id: test
display_name: Test
supported_flows: [browser]
auth_endpoint: https://example.com/auth
token_endpoint: https://example.com/token
auth_lanes: [invalid-lane]
`)
	_, err := connectors.ParseProviderManifest(yaml)
	if err == nil {
		t.Fatal("expected error for invalid auth lane, got nil")
	}
	if !strings.Contains(err.Error(), "invalid auth lane") {
		t.Errorf("expected 'invalid auth lane' in error, got: %v", err)
	}
}

func TestParseProviderManifest_PublicClientLaneRequiresEmbeddedClientID(t *testing.T) {
	yaml := []byte(`
id: test
display_name: Test
supported_flows: [browser]
auth_endpoint: https://example.com/auth
token_endpoint: https://example.com/token
auth_lanes: [public-client, byo]
`)
	_, err := connectors.ParseProviderManifest(yaml)
	if err == nil {
		t.Fatal("expected error for missing embedded_client_id with public-client lane, got nil")
	}
	if !strings.Contains(err.Error(), "public-client auth lane requires embedded_client_id") {
		t.Errorf("expected embedded_client_id lane validation error, got: %v", err)
	}
}

func TestParseProviderManifest_ManagedConfidentialLaneRequiresManagedClientID(t *testing.T) {
	yaml := []byte(`
id: test
display_name: Test
supported_flows: [browser]
auth_endpoint: https://example.com/auth
token_endpoint: https://example.com/token
auth_lanes: [managed-confidential, byo]
`)
	_, err := connectors.ParseProviderManifest(yaml)
	if err == nil {
		t.Fatal("expected error for missing managed_client_id with managed-confidential lane, got nil")
	}
	if !strings.Contains(err.Error(), "managed-confidential auth lane requires managed_client_id") {
		t.Errorf("expected managed_client_id lane validation error, got: %v", err)
	}
}

func TestParseProviderManifest_MalformedYAML(t *testing.T) {
	_, err := connectors.ParseProviderManifest([]byte(":::not valid yaml:::"))
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

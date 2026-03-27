package connectors_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/connectors"
	"gopkg.in/yaml.v3"
)

func TestProviderManifest_RoundTrip(t *testing.T) {
	original := connectors.ProviderManifest{
		ID:                 "test-provider",
		DisplayName:        "Test Provider",
		SupportedFlows:     []string{"browser", "device"},
		AuthEndpoint:       "https://example.com/oauth/authorize",
		TokenEndpoint:      "https://example.com/oauth/token",
		DeviceEndpoint:     "https://example.com/oauth/device",
		RevocationEndpoint: "https://example.com/oauth/revoke",
		UserInfoEndpoint:   "https://example.com/oauth/userinfo",
		DefaultScopes:      []string{"read", "write"},
		ScopePresets: map[string]string{
			"minimal": "read",
			"full":    "read write admin",
		},
		ConnectionScopeModel: []string{"user"},
		PKCERequired:         true,
		Notes:                "test notes",
		AuthLanes:            []string{"public-client", "byo"},
		EmbeddedClientID:     "test-client-id",
		ManagedClientID:      "managed-client-id",
		TokenExchange:        connectors.TokenExchangeConfig{AuthMethod: "body"},
		EndpointPlaceholders: []string{"subdomain"},
	}

	data, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var unmarshaled connectors.ProviderManifest
	if err := yaml.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(original, unmarshaled) {
		t.Errorf("round-trip mismatch:\noriginal:   %+v\nunmarshaled: %+v", original, unmarshaled)
	}
}

func TestProviderManifest_Validate_InvalidLane(t *testing.T) {
	m := connectors.ProviderManifest{
		ID:             "test",
		DisplayName:    "Test",
		SupportedFlows: []string{"browser"},
		AuthEndpoint:   "https://example.com/auth",
		TokenEndpoint:  "https://example.com/token",
		AuthLanes:      []string{"invalid-lane"},
	}
	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid auth lane, got nil")
	}
	if !strings.Contains(err.Error(), "invalid auth lane") {
		t.Errorf("expected error to contain 'invalid auth lane', got: %v", err)
	}
}

func TestProviderManifest_Validate_MissingRequired(t *testing.T) {
	m := connectors.ProviderManifest{
		ID:             "test",
		DisplayName:    "Test",
		SupportedFlows: []string{"browser"},
		AuthEndpoint:   "",
		TokenEndpoint:  "https://example.com/token",
		AuthLanes:      []string{"byo"},
	}
	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing auth_endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "auth_endpoint") {
		t.Errorf("expected error to mention auth_endpoint, got: %v", err)
	}
}

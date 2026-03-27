package connectors

import (
	"fmt"
	"strings"
)

// TokenExchangeConfig defines how tokens are exchanged with the provider.
type TokenExchangeConfig struct {
	// AuthMethod controls how client credentials are sent during token exchange.
	// Values: "basic" (HTTP Basic auth header), "body" (form body params). Defaults to "body" if empty.
	AuthMethod string `yaml:"auth_method,omitempty" json:"auth_method,omitempty"`
}

// ProviderManifest is the declarative YAML definition for an OAuth provider.
// Stored in internal/connectors/providers/official/*.yaml.
// This is SEPARATE from SkillManifest — do not merge.
type ProviderManifest struct {
	ID                   string            `yaml:"id"                            json:"id"`
	DisplayName          string            `yaml:"display_name"                  json:"display_name"`
	SupportedFlows       []string          `yaml:"supported_flows"               json:"supported_flows"`
	AuthEndpoint         string            `yaml:"auth_endpoint"                 json:"auth_endpoint"`
	TokenEndpoint        string            `yaml:"token_endpoint"                json:"token_endpoint"`
	DeviceEndpoint       string            `yaml:"device_endpoint,omitempty"     json:"device_endpoint,omitempty"`
	RevocationEndpoint   string            `yaml:"revocation_endpoint,omitempty" json:"revocation_endpoint,omitempty"`
	UserInfoEndpoint     string            `yaml:"user_info_endpoint,omitempty"  json:"user_info_endpoint,omitempty"`
	DefaultScopes        []string          `yaml:"default_scopes,omitempty"      json:"default_scopes,omitempty"`
	ScopePresets         map[string]string `yaml:"scope_presets,omitempty"       json:"scope_presets,omitempty"`
	ConnectionScopeModel []string          `yaml:"connection_scope_model,omitempty" json:"connection_scope_model,omitempty"`
	PKCERequired         bool              `yaml:"pkce_required,omitempty"       json:"pkce_required,omitempty"`
	Notes                string            `yaml:"notes,omitempty"               json:"notes,omitempty"`
	// AuthLanes lists supported authentication lanes: "public-client", "managed-confidential", "byo".
	// REQUIRED. Controls how credentials are resolved when connecting.
	AuthLanes []string `yaml:"auth_lanes"                    json:"auth_lanes"`
	// EmbeddedClientID is the OAuth app client ID for the public-client lane.
	// Safe to commit — this is not a secret. Used only with device/PKCE flows.
	EmbeddedClientID string `yaml:"embedded_client_id,omitempty"  json:"embedded_client_id,omitempty"`
	// ManagedClientID is the OAuth app client ID for the managed-confidential lane.
	// Safe to commit — the corresponding client_secret is stored in the vault only.
	ManagedClientID string `yaml:"managed_client_id,omitempty"   json:"managed_client_id,omitempty"`
	// TokenExchange configures how token requests are made to the provider.
	TokenExchange TokenExchangeConfig `yaml:"token_exchange,omitempty"      json:"token_exchange,omitempty"`
	// EndpointPlaceholders lists URL template variables (e.g. "subdomain" for Zendesk).
	EndpointPlaceholders []string `yaml:"endpoint_placeholders,omitempty" json:"endpoint_placeholders,omitempty"`
}

// validAuthLanes is the set of allowed auth lane values.
var validAuthLanes = map[string]bool{
	"public-client":        true,
	"managed-confidential": true,
	"byo":                  true,
}

// validAuthMethods is the set of allowed token exchange auth method values.
var validAuthMethods = map[string]bool{
	"":      true,
	"basic": true,
	"body":  true,
}

// Validate checks required fields and enum values on the manifest.
func (m *ProviderManifest) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("provider manifest: id is required")
	}
	if m.DisplayName == "" {
		return fmt.Errorf("provider manifest %q: display_name is required", m.ID)
	}
	if len(m.SupportedFlows) == 0 {
		return fmt.Errorf("provider manifest %q: supported_flows must not be empty", m.ID)
	}
	if m.AuthEndpoint == "" {
		return fmt.Errorf("provider manifest %q: auth_endpoint is required", m.ID)
	}
	if m.TokenEndpoint == "" {
		return fmt.Errorf("provider manifest %q: token_endpoint is required", m.ID)
	}
	if len(m.AuthLanes) == 0 {
		return fmt.Errorf("provider manifest %q: auth_lanes must not be empty", m.ID)
	}
	for _, lane := range m.AuthLanes {
		if !validAuthLanes[lane] {
			return fmt.Errorf("provider manifest %q: invalid auth lane: %q (allowed: public-client, managed-confidential, byo)", m.ID, lane)
		}
	}
	if containsAuthLane(m.AuthLanes, "public-client") && strings.TrimSpace(m.EmbeddedClientID) == "" {
		return fmt.Errorf("provider manifest %q: public-client auth lane requires embedded_client_id", m.ID)
	}
	if containsAuthLane(m.AuthLanes, "managed-confidential") && strings.TrimSpace(m.ManagedClientID) == "" {
		return fmt.Errorf("provider manifest %q: managed-confidential auth lane requires managed_client_id", m.ID)
	}
	if !validAuthMethods[m.TokenExchange.AuthMethod] {
		return fmt.Errorf("provider manifest %q: invalid token_exchange.auth_method: %q (allowed: basic, body)", m.ID, m.TokenExchange.AuthMethod)
	}
	return nil
}

func containsAuthLane(lanes []string, target string) bool {
	for _, lane := range lanes {
		if strings.TrimSpace(lane) == target {
			return true
		}
	}
	return false
}

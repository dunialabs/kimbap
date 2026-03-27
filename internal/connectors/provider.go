package connectors

// ProviderDefinition is the PRD "OAuth Definition" layer — static provider
// template and capability metadata exposed to CLI and flow selection.
type ProviderDefinition struct {
	ID                   string            `json:"id" yaml:"id"`
	DisplayName          string            `json:"display_name" yaml:"display_name"`
	SupportedFlows       []FlowType        `json:"supported_flows" yaml:"supported_flows"`
	AuthEndpoint         string            `json:"auth_endpoint" yaml:"auth_endpoint"`
	TokenEndpoint        string            `json:"token_endpoint" yaml:"token_endpoint"`
	DeviceEndpoint       string            `json:"device_endpoint,omitempty" yaml:"device_endpoint,omitempty"`
	RevocationEndpoint   string            `json:"revocation_endpoint,omitempty" yaml:"revocation_endpoint,omitempty"`
	UserInfoEndpoint     string            `json:"userinfo_endpoint,omitempty" yaml:"userinfo_endpoint,omitempty"`
	DefaultScopes        []string          `json:"default_scopes" yaml:"default_scopes"`
	ScopePresets         map[string]string `json:"scope_presets,omitempty" yaml:"scope_presets,omitempty"`
	ConnectionScopeModel []ConnectionScope `json:"connection_scope_model" yaml:"connection_scope_model"`
	PKCERequired         bool              `json:"pkce_required" yaml:"pkce_required"`
	Notes                string            `json:"notes,omitempty" yaml:"notes,omitempty"`
	// AuthLanes lists supported authentication lanes for this provider.
	// Values: "public-client", "managed-confidential", "byo".
	// Populated when loaded from YAML manifests.
	AuthLanes            []string            `json:"auth_lanes,omitempty" yaml:"auth_lanes,omitempty"`
	EmbeddedClientID     string              `json:"embedded_client_id,omitempty" yaml:"embedded_client_id,omitempty"`
	ManagedClientID      string              `json:"managed_client_id,omitempty" yaml:"managed_client_id,omitempty"`
	TokenExchange        TokenExchangeConfig `json:"token_exchange,omitempty" yaml:"token_exchange,omitempty"`
	EndpointPlaceholders []string            `json:"endpoint_placeholders,omitempty" yaml:"endpoint_placeholders,omitempty"`
}

func (p ProviderDefinition) SupportsFlow(flow FlowType) bool {
	for _, f := range p.SupportedFlows {
		if f == flow {
			return true
		}
	}
	return false
}

func (p ProviderDefinition) SupportsBrowserFlow() bool {
	return p.SupportsFlow(FlowBrowser)
}

func (p ProviderDefinition) SupportsDeviceFlow() bool {
	return p.SupportsFlow(FlowDevice)
}

func (p ProviderDefinition) SupportsClientCredentials() bool {
	return p.SupportsFlow(FlowClientCredentials)
}

func (p ProviderDefinition) SupportsRevocation() bool {
	return p.RevocationEndpoint != ""
}

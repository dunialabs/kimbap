package connectors

import "time"

// FlowType identifies a supported OAuth flow mechanism.
type FlowType string

const (
	FlowBrowser           FlowType = "browser"
	FlowDevice            FlowType = "device"
	FlowClientCredentials FlowType = "client_credentials"
)

// ConnectionScope defines who owns a connection.
type ConnectionScope string

const (
	ScopeUser      ConnectionScope = "user"
	ScopeWorkspace ConnectionScope = "workspace"
	ScopeService   ConnectionScope = "service"
)

// ConnectorStatus is the legacy status type kept for backward compatibility
// with existing DB rows and internal Manager state transitions.
// New code should prefer ConnectionStatus from status.go.
type ConnectorStatus string

const (
	StatusHealthy      ConnectorStatus = "healthy"
	StatusExpiring     ConnectorStatus = "expiring"
	StatusOldExpired   ConnectorStatus = "expired"
	StatusReauthNeeded ConnectorStatus = "reauth_needed"
	StatusPending      ConnectorStatus = "pending"
)

// ConnectorConfig holds environment- or tenant-specific OAuth provider
// installation settings. It maps to the PRD "OAuth Installation" layer.
type ConnectorConfig struct {
	Name              string          `yaml:"name"`
	Provider          string          `yaml:"provider"`
	ClientID          string          `yaml:"client_id"`
	ClientSecret      string          `yaml:"client_secret"`
	Scopes            []string        `yaml:"scopes"`
	AuthURL           string          `yaml:"auth_url"`
	TokenURL          string          `yaml:"token_url"`
	DeviceURL         string          `yaml:"device_url,omitempty"`
	RevocationURL     string          `yaml:"revocation_url,omitempty"`
	ConnectionScope   ConnectionScope `yaml:"connection_scope,omitempty"`
	EnableBrowserFlow bool            `yaml:"enable_browser_flow,omitempty"`
	EnableDeviceFlow  bool            `yaml:"enable_device_flow,omitempty"`
}

// ConnectorState holds the runtime token lifecycle state for an external
// OAuth connection. It maps to the PRD "OAuth Connection" layer.
type ConnectorState struct {
	Name               string
	TenantID           string
	WorkspaceID        string
	Provider           string
	Profile            string
	Status             ConnectorStatus
	AccessToken        string
	RefreshToken       string
	ExpiresAt          *time.Time
	Scopes             []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastRefresh        *time.Time
	LastRefreshError   string
	LastUsedAt         *time.Time
	Account            string
	ConnectedPrincipal string
	ConnectionScope    ConnectionScope
	RevokedAt          *time.Time
	FlowUsed           FlowType
}

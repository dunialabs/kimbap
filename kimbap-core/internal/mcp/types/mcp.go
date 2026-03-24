package types

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type BaseCapabilityConfig struct {
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

type ToolCapabilityConfig struct {
	Enabled     bool           `json:"enabled"`
	Description string         `json:"description,omitempty"`
	DangerLevel *int           `json:"dangerLevel,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type ResourceCapabilityConfig = BaseCapabilityConfig

type PromptCapabilityConfig = BaseCapabilityConfig

type ServerConfigCapabilities struct {
	Tools     map[string]ToolCapabilityConfig     `json:"tools"`
	Resources map[string]ResourceCapabilityConfig `json:"resources"`
	Prompts   map[string]PromptCapabilityConfig   `json:"prompts"`
}

func (c *ServerConfigCapabilities) EnsureInitialized() {
	if c.Tools == nil {
		c.Tools = map[string]ToolCapabilityConfig{}
	}
	if c.Resources == nil {
		c.Resources = map[string]ResourceCapabilityConfig{}
	}
	if c.Prompts == nil {
		c.Prompts = map[string]PromptCapabilityConfig{}
	}
}

type ServerConfigWithEnabled struct {
	Tools          map[string]ToolCapabilityConfig     `json:"tools"`
	Resources      map[string]ResourceCapabilityConfig `json:"resources"`
	Prompts        map[string]PromptCapabilityConfig   `json:"prompts"`
	Enabled        bool                                `json:"enabled"`
	ServerName     string                              `json:"serverName"`
	AllowUserInput bool                                `json:"allowUserInput"`
	AuthType       int                                 `json:"authType"`
	Category       *int                                `json:"category,omitempty"`
	ConfigTemplate string                              `json:"configTemplate"`
	Configured     bool                                `json:"configured"`
	Status         *int                                `json:"status,omitempty"`
}

func (s *ServerConfigWithEnabled) UnmarshalJSON(data []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("server permissions must be an object: %w", err)
	}
	if obj == nil {
		return fmt.Errorf("server permissions must be an object")
	}

	enabledRaw, ok := obj["enabled"]
	if !ok {
		return fmt.Errorf("server permissions must include enabled")
	}
	var enabled bool
	if err := json.Unmarshal(enabledRaw, &enabled); err != nil {
		return fmt.Errorf("server permissions enabled must be boolean: %w", err)
	}

	for _, field := range []string{"tools", "resources", "prompts"} {
		raw, exists := obj[field]
		if !exists {
			continue
		}
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(raw, &nested); err != nil {
			return fmt.Errorf("server permissions %s must be an object: %w", field, err)
		}
		if nested == nil {
			return fmt.Errorf("server permissions %s must be an object", field)
		}
	}

	type alias ServerConfigWithEnabled
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = ServerConfigWithEnabled(decoded)
	return nil
}

type Permissions map[string]ServerConfigWithEnabled

type MCPServerCapability = ServerConfigWithEnabled

type MCPServerCapabilities = Permissions

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ProxyContext struct {
	ProxyRequestID   string `json:"proxyRequestId"`
	UniformRequestID string `json:"uniformRequestId,omitempty"`
}

type AuthContext struct {
	Kind            string      `json:"kind,omitempty"`
	UserID          string      `json:"userId"`
	Role            int         `json:"role,omitempty"`
	Status          int         `json:"status,omitempty"`
	Permissions     Permissions `json:"permissions"`
	UserPreferences Permissions `json:"userPreferences"`
	LaunchConfigs   string      `json:"launchConfigs,omitempty"`
	ExpiresAt       int64       `json:"expiresAt,omitempty"`
	OAuthClientID   string      `json:"oauthClientId,omitempty"`
	OAuthScopes     []string    `json:"oauthScopes,omitempty"`
	UserAgent       string      `json:"userAgent,omitempty"`
	AuthenticatedAt time.Time   `json:"authenticatedAt,omitempty"`
	RateLimit       int         `json:"rateLimit,omitempty"`
}

type DisconnectReason string

const (
	DisconnectReasonClientDisconnect DisconnectReason = "CLIENT_DISCONNECT"
	DisconnectReasonUserDisabled     DisconnectReason = "USER_DISABLED"
	DisconnectReasonUserExpired      DisconnectReason = "USER_EXPIRED"

	DisconnectReasonSessionTimeout DisconnectReason = "SESSION_TIMEOUT"
	DisconnectReasonServerShutdown DisconnectReason = "SERVER_SHUTDOWN"
	DisconnectReasonSessionRemoved DisconnectReason = "SESSION_REMOVED"
	DisconnectReasonTokenRevoked   DisconnectReason = "TOKEN_REVOKED"
)

type StreamID = string

type EventID = string

type JSONRPCMessage struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Method  string        `json:"method,omitempty"`
	Params  interface{}   `json:"params,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type ReplayOptions struct {
	Send func(ctx context.Context, eventID EventID, message JSONRPCMessage) error
}

type EventStore interface {
	StoreEvent(ctx context.Context, streamID StreamID, message JSONRPCMessage) (EventID, error)
	ReplayEventsAfter(ctx context.Context, lastEventID EventID, options ReplayOptions) (StreamID, error)
}

type CachedEvent struct {
	EventID   string         `json:"eventId"`
	Message   JSONRPCMessage `json:"message"`
	Timestamp time.Time      `json:"timestamp"`
	StreamID  string         `json:"streamId"`
}

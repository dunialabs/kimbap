package providers

import (
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/connectors"
)

var registry = map[string]connectors.ProviderDefinition{}
var order []string

func init() {
	defs := []connectors.ProviderDefinition{
		{
			ID:               "github",
			DisplayName:      "GitHub",
			SupportedFlows:   []connectors.FlowType{connectors.FlowBrowser, connectors.FlowDevice},
			AuthEndpoint:     "https://github.com/login/oauth/authorize",
			TokenEndpoint:    "https://github.com/login/oauth/access_token",
			DeviceEndpoint:   "https://github.com/login/device/code",
			UserInfoEndpoint: "https://api.github.com/user",
			DefaultScopes:    []string{"repo", "read:org"},
			ScopePresets: map[string]string{
				"minimal": "read:user",
				"default": "repo read:org",
				"full":    "repo read:org admin:org",
			},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser},
			PKCERequired:         false,
		},
		{
			ID:                 "google",
			DisplayName:        "Google",
			SupportedFlows:     []connectors.FlowType{connectors.FlowBrowser, connectors.FlowDevice},
			AuthEndpoint:       "https://accounts.google.com/o/oauth2/v2/auth",
			TokenEndpoint:      "https://oauth2.googleapis.com/token",
			DeviceEndpoint:     "https://oauth2.googleapis.com/device/code",
			RevocationEndpoint: "https://oauth2.googleapis.com/revoke",
			UserInfoEndpoint:   "https://www.googleapis.com/oauth2/v2/userinfo",
			DefaultScopes:      []string{"openid", "email", "profile"},
			ScopePresets: map[string]string{
				"minimal": "openid email",
				"gmail":   "openid email https://www.googleapis.com/auth/gmail.readonly",
				"drive":   "openid email https://www.googleapis.com/auth/drive.readonly",
			},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser},
			PKCERequired:         true,
		},
		{
			ID:                   "notion",
			DisplayName:          "Notion",
			SupportedFlows:       []connectors.FlowType{connectors.FlowBrowser},
			AuthEndpoint:         "https://api.notion.com/v1/oauth/authorize",
			TokenEndpoint:        "https://api.notion.com/v1/oauth/token",
			UserInfoEndpoint:     "https://api.notion.com/v1/users/me",
			DefaultScopes:        []string{},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeWorkspace},
			PKCERequired:         false,
			Notes:                "Notion uses workspace-level OAuth. Scopes are defined during OAuth app setup, not at authorization time.",
		},
		{
			ID:               "figma",
			DisplayName:      "Figma",
			SupportedFlows:   []connectors.FlowType{connectors.FlowBrowser},
			AuthEndpoint:     "https://www.figma.com/oauth",
			TokenEndpoint:    "https://api.figma.com/v1/oauth/token",
			UserInfoEndpoint: "https://api.figma.com/v1/me",
			DefaultScopes:    []string{"files:read"},
			ScopePresets: map[string]string{
				"readonly":  "files:read",
				"readwrite": "files:read,file_variables:read,file_variables:write",
			},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser},
			PKCERequired:         false,
		},
		{
			ID:                 "stripe",
			DisplayName:        "Stripe",
			SupportedFlows:     []connectors.FlowType{connectors.FlowBrowser},
			AuthEndpoint:       "https://connect.stripe.com/oauth/authorize",
			TokenEndpoint:      "https://connect.stripe.com/oauth/token",
			RevocationEndpoint: "https://connect.stripe.com/oauth/deauthorize",
			DefaultScopes:      []string{"read_only"},
			ScopePresets: map[string]string{
				"readonly":  "read_only",
				"readwrite": "read_write",
			},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser, connectors.ScopeWorkspace},
			PKCERequired:         false,
		},
		{
			ID:             "zendesk",
			DisplayName:    "Zendesk",
			SupportedFlows: []connectors.FlowType{connectors.FlowBrowser},
			AuthEndpoint:   "https://{subdomain}.zendesk.com/oauth/authorizations/new",
			TokenEndpoint:  "https://{subdomain}.zendesk.com/oauth/tokens",
			DefaultScopes:  []string{"read"},
			ScopePresets: map[string]string{
				"readonly":  "read",
				"readwrite": "read write",
			},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeWorkspace},
			PKCERequired:         false,
			Notes:                "Zendesk requires a subdomain in the OAuth endpoints. Use --extra subdomain=<your-subdomain> when connecting.",
		},
		{
			ID:                   "canvas",
			DisplayName:          "Canvas LMS",
			SupportedFlows:       []connectors.FlowType{connectors.FlowBrowser},
			AuthEndpoint:         "https://{domain}/login/oauth2/auth",
			TokenEndpoint:        "https://{domain}/login/oauth2/token",
			DefaultScopes:        []string{},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser},
			PKCERequired:         false,
			Notes:                "Canvas LMS requires your institution domain. Use --extra domain=<your-canvas-domain> when connecting.",
		},
		{
			ID:                 "canva",
			DisplayName:        "Canva",
			SupportedFlows:     []connectors.FlowType{connectors.FlowBrowser},
			AuthEndpoint:       "https://www.canva.com/api/oauth/authorize",
			TokenEndpoint:      "https://api.canva.com/rest/v1/oauth/token",
			RevocationEndpoint: "https://api.canva.com/rest/v1/oauth/revoke",
			UserInfoEndpoint:   "https://api.canva.com/rest/v1/users/me",
			DefaultScopes:      []string{"design:content:read"},
			ScopePresets: map[string]string{
				"readonly":  "design:content:read",
				"readwrite": "design:content:read design:content:write",
			},
			ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser},
			PKCERequired:         true,
		},
	}

	for _, def := range defs {
		registry[def.ID] = def
		order = append(order, def.ID)
	}
}

func GetProvider(id string) (connectors.ProviderDefinition, error) {
	normalized := strings.ToLower(strings.TrimSpace(id))
	def, ok := registry[normalized]
	if !ok {
		return connectors.ProviderDefinition{}, fmt.Errorf("unknown provider: %s", id)
	}
	return def, nil
}

func ListProviders() []connectors.ProviderDefinition {
	out := make([]connectors.ProviderDefinition, 0, len(order))
	for _, id := range order {
		if def, ok := registry[id]; ok {
			out = append(out, def)
		}
	}
	return out
}

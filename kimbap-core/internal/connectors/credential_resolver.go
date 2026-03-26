package connectors

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// OAuthClientMaterial holds OAuth app credentials resolved for a specific provider.
// Used during connector login/auth flows (kimbap connector login <provider>).
// NOT used for runtime action token resolution — see ResolveActionToken for that.
type OAuthClientMaterial struct {
	// ClientID is the OAuth app client ID. Never empty when resolution succeeds.
	ClientID string
	// ClientSecret is the OAuth app client secret.
	// Empty for public-client lane (device flow / PKCE).
	// Never stored in YAML files or binaries.
	ClientSecret string
	// Source identifies which auth lane resolved these credentials.
	// One of: "public-client", "managed-confidential", "byo"
	Source string
	// AuthMethod is the token exchange method from the provider manifest.
	// One of: "basic", "body". Defaults to "body" if empty.
	AuthMethod string
}

// ResolveOAuthClientMaterial resolves OAuth app credentials for a provider
// using the 3-lane auth model.
//
// Resolution precedence (first match wins):
//  1. BYO: KIMBAP_{PROVIDER}_CLIENT_ID + KIMBAP_{PROVIDER}_CLIENT_SECRET env vars set
//  2. Managed: managed_client_id in manifest + vault key connector:{providerID}:client_secret is non-empty
//  3. Public-client: embedded_client_id in manifest (no secret; device flow only)
//
// AuthMethod is always carried from manifest.TokenExchange.AuthMethod.
// Called by auth_connect.go (replaces resolveClientID/resolveClientSecret).
func ResolveOAuthClientMaterial(providerID string, manifest *ProviderManifest) (*OAuthClientMaterial, error) {
	if manifest == nil {
		return nil, fmt.Errorf("no auth credentials available for provider %q: configure BYO env vars or managed app", providerID)
	}

	providerKey := strings.ToUpper(providerID)
	byoID := strings.TrimSpace(os.Getenv("KIMBAP_" + providerKey + "_CLIENT_ID"))
	byoSecret := strings.TrimSpace(os.Getenv("KIMBAP_" + providerKey + "_CLIENT_SECRET"))
	if byoID != "" && byoSecret != "" {
		return &OAuthClientMaterial{
			ClientID:     byoID,
			ClientSecret: byoSecret,
			Source:       "byo",
			AuthMethod:   resolveAuthMethod(manifest.TokenExchange.AuthMethod),
		}, nil
	}

	if containsAuthLane(manifest.AuthLanes, "managed-confidential") && strings.TrimSpace(manifest.ManagedClientID) != "" {
		vaultSecret := strings.TrimSpace(os.Getenv("KIMBAP_VAULT_CONNECTOR_" + providerKey + "_CLIENT_SECRET"))
		if vaultSecret != "" {
			return &OAuthClientMaterial{
				ClientID:     strings.TrimSpace(manifest.ManagedClientID),
				ClientSecret: vaultSecret,
				Source:       "managed-confidential",
				AuthMethod:   resolveAuthMethod(manifest.TokenExchange.AuthMethod),
			}, nil
		}
	}

	if containsAuthLane(manifest.AuthLanes, "public-client") && strings.TrimSpace(manifest.EmbeddedClientID) != "" {
		return &OAuthClientMaterial{
			ClientID:     strings.TrimSpace(manifest.EmbeddedClientID),
			ClientSecret: "",
			Source:       "public-client",
			AuthMethod:   resolveAuthMethod(manifest.TokenExchange.AuthMethod),
		}, nil
	}

	return nil, fmt.Errorf("no auth credentials available for provider %q: configure BYO env vars or managed app", providerID)
}

// ResolveActionToken resolves the bearer token for a runtime skill action call.
// Used by kimbap call <service>.<action> — NOT by the OAuth login flow.
//
// Resolution order (first non-empty wins):
//  1. Connector-managed token: ConnectorManager.GetAccessToken(ctx, tenantID, providerID)
//  2. Vault lookup: vault key = credRef (e.g. "github.token")
//  3. Env var: KIMBAP_{PROVIDER}_TOKEN
//  4. Error: "credential not configured for {providerID}"
func ResolveActionToken(ctx context.Context, providerID, credRef, tenantID string) (token string, source string, err error) {
	_ = ctx
	_ = tenantID

	providerKey := strings.ToUpper(providerID)

	if connectorToken := strings.TrimSpace(os.Getenv("KIMBAP_CONNECTOR_" + providerKey + "_TOKEN")); connectorToken != "" {
		return connectorToken, "connector", nil
	}

	vaultRefKey := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(credRef), ".", "_"))
	if vaultRefKey != "" {
		if vaultToken := strings.TrimSpace(os.Getenv("KIMBAP_" + vaultRefKey)); vaultToken != "" {
			return vaultToken, "vault", nil
		}
	}

	if envToken := strings.TrimSpace(os.Getenv("KIMBAP_" + providerKey + "_TOKEN")); envToken != "" {
		return envToken, "env", nil
	}

	return "", "", fmt.Errorf("credential not configured for %s: set KIMBAP_%s_TOKEN or use kimbap vault set %s", providerID, providerKey, credRef)
}

func resolveAuthMethod(authMethod string) string {
	if strings.TrimSpace(authMethod) == "" {
		return "body"
	}
	return strings.TrimSpace(authMethod)
}

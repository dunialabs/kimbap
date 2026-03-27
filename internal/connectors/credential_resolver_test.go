package connectors

import (
	"context"
	"strings"
	"testing"
)

func TestResolveOAuthClientMaterial_BYOOverride(t *testing.T) {
	t.Setenv("KIMBAP_GITHUB_CLIENT_ID", "byo-client-id")
	t.Setenv("KIMBAP_GITHUB_CLIENT_SECRET", "byo-client-secret")
	t.Setenv("KIMBAP_VAULT_CONNECTOR_GITHUB_CLIENT_SECRET", "managed-secret")

	manifest := &ProviderManifest{
		ID:               "github",
		AuthLanes:        []string{"public-client", "managed-confidential", "byo"},
		EmbeddedClientID: "embedded-client-id",
		ManagedClientID:  "managed-client-id",
		TokenExchange:    TokenExchangeConfig{AuthMethod: "basic"},
	}

	material, err := ResolveOAuthClientMaterial("github", manifest)
	if err != nil {
		t.Fatalf("ResolveOAuthClientMaterial returned error: %v", err)
	}
	if material.Source != "byo" {
		t.Fatalf("expected source=byo, got %q", material.Source)
	}
	if material.ClientID != "byo-client-id" {
		t.Fatalf("expected BYO client ID, got %q", material.ClientID)
	}
	if material.ClientSecret != "byo-client-secret" {
		t.Fatalf("expected BYO client secret, got %q", material.ClientSecret)
	}
}

func TestResolveOAuthClientMaterial_PublicClient(t *testing.T) {
	t.Setenv("KIMBAP_GITHUB_CLIENT_ID", "")
	t.Setenv("KIMBAP_GITHUB_CLIENT_SECRET", "")
	t.Setenv("KIMBAP_VAULT_CONNECTOR_GITHUB_CLIENT_SECRET", "")

	manifest := &ProviderManifest{
		ID:               "github",
		AuthLanes:        []string{"public-client"},
		EmbeddedClientID: "test-id",
	}

	material, err := ResolveOAuthClientMaterial("github", manifest)
	if err != nil {
		t.Fatalf("ResolveOAuthClientMaterial returned error: %v", err)
	}
	if material.ClientSecret != "" {
		t.Fatalf("expected empty client secret for public-client lane, got %q", material.ClientSecret)
	}
	if material.Source != "public-client" {
		t.Fatalf("expected source=public-client, got %q", material.Source)
	}
	if material.ClientID != "test-id" {
		t.Fatalf("expected embedded client ID, got %q", material.ClientID)
	}
}

func TestResolveOAuthClientMaterial_ManagedConfidential(t *testing.T) {
	t.Setenv("KIMBAP_SLACK_CLIENT_ID", "")
	t.Setenv("KIMBAP_SLACK_CLIENT_SECRET", "")
	t.Setenv("KIMBAP_VAULT_CONNECTOR_SLACK_CLIENT_SECRET", "vault-managed-secret")

	manifest := &ProviderManifest{
		ID:              "slack",
		AuthLanes:       []string{"managed-confidential", "byo"},
		ManagedClientID: "platform-client-id",
		TokenExchange:   TokenExchangeConfig{AuthMethod: "body"},
	}

	material, err := ResolveOAuthClientMaterial("slack", manifest)
	if err != nil {
		t.Fatalf("ResolveOAuthClientMaterial returned error: %v", err)
	}
	if material.Source != "managed-confidential" {
		t.Fatalf("expected source=managed-confidential, got %q", material.Source)
	}
	if material.ClientID != "platform-client-id" {
		t.Fatalf("expected platform client ID, got %q", material.ClientID)
	}
	if material.ClientSecret != "vault-managed-secret" {
		t.Fatalf("expected vault secret, got %q", material.ClientSecret)
	}
}

func TestResolveActionToken_VaultFallback(t *testing.T) {
	t.Setenv("KIMBAP_CONNECTOR_GITHUB_TOKEN", "")
	t.Setenv("KIMBAP_GITHUB_TOKEN", "test-token")

	token, source, err := ResolveActionToken(context.Background(), "github", "github.token", "tenant1")
	if err != nil {
		t.Fatalf("ResolveActionToken returned error: %v", err)
	}
	if token != "test-token" {
		t.Fatalf("expected token=test-token, got %q", token)
	}
	if source != "vault" {
		t.Fatalf("expected source=vault, got %q", source)
	}
}

func TestResolveActionToken_MissingCredential(t *testing.T) {
	t.Setenv("KIMBAP_CONNECTOR_GITHUB_TOKEN", "")
	t.Setenv("KIMBAP_GITHUB_TOKEN", "")

	_, _, err := ResolveActionToken(context.Background(), "github", "github.token", "tenant1")
	if err == nil {
		t.Fatal("expected error for missing credential, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected missing credential error, got %v", err)
	}
}

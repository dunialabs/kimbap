package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/vault"
)

type mockLinkVaultStore struct {
	upsertCalled bool
	upsertName   string
	upsertValue  []byte
	upsertType   vault.SecretType
	upsertErr    error
}

func (m *mockLinkVaultStore) Create(_ context.Context, _, _ string, _ vault.SecretType, _ []byte, _ map[string]string, _ string) (*vault.SecretRecord, error) {
	return &vault.SecretRecord{}, nil
}
func (m *mockLinkVaultStore) Upsert(_ context.Context, _, name string, secretType vault.SecretType, plaintext []byte, _ map[string]string, _ string) (*vault.SecretRecord, error) {
	m.upsertCalled = true
	m.upsertName = name
	m.upsertValue = plaintext
	m.upsertType = secretType
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	return &vault.SecretRecord{Name: name, Type: secretType}, nil
}
func (m *mockLinkVaultStore) GetMeta(_ context.Context, _, _ string) (*vault.SecretRecord, error) {
	return nil, vault.ErrSecretNotFound
}
func (m *mockLinkVaultStore) GetValue(_ context.Context, _, _ string) ([]byte, error) {
	return nil, vault.ErrSecretNotFound
}
func (m *mockLinkVaultStore) List(_ context.Context, _ string, _ vault.ListOptions) ([]vault.SecretRecord, error) {
	return nil, nil
}
func (m *mockLinkVaultStore) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockLinkVaultStore) Rotate(_ context.Context, _, _ string, _ []byte, _ string) (*vault.SecretRecord, error) {
	return nil, nil
}
func (m *mockLinkVaultStore) GetVersion(_ context.Context, _, _ string, _ int) ([]byte, error) {
	return nil, nil
}
func (m *mockLinkVaultStore) MarkUsed(_ context.Context, _, _ string) error { return nil }
func (m *mockLinkVaultStore) Exists(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func TestLinkStdinReadsAndStoresCredential(t *testing.T) {
	resetOptsForTest(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	if _, err := w.Write([]byte("sk-test-key\n")); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	mock := &mockLinkVaultStore{}
	info := linkServiceInfo{
		Service:       "stripe",
		AuthType:      actions.AuthTypeBearer,
		CredentialRef: "stripe.api_key",
	}

	err = linkStoreCredentialFromInput(mock, "default", "stripe.api_key", info, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.upsertCalled {
		t.Fatal("expected Upsert to be called")
	}
	if mock.upsertName != "stripe.api_key" {
		t.Fatalf("expected upsert name %q, got %q", "stripe.api_key", mock.upsertName)
	}
	if got := string(mock.upsertValue); got != "sk-test-key" {
		t.Fatalf("expected trimmed value %q, got %q", "sk-test-key", got)
	}
	if mock.upsertType != vault.SecretTypeBearerToken {
		t.Fatalf("expected secret type %q, got %q", vault.SecretTypeBearerToken, mock.upsertType)
	}
}

func TestLinkFileReadsAndStoresCredential(t *testing.T) {
	resetOptsForTest(t)

	keyFile := filepath.Join(t.TempDir(), "key.txt")
	if err := os.WriteFile(keyFile, []byte("  sk-file-key  \n"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	mock := &mockLinkVaultStore{}
	info := linkServiceInfo{
		Service:       "stripe",
		AuthType:      actions.AuthTypeAPIKey,
		CredentialRef: "stripe.api_key",
	}

	err := linkStoreCredentialFromInput(mock, "default", "stripe.api_key", info, false, keyFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.upsertCalled {
		t.Fatal("expected Upsert to be called")
	}
	if got := string(mock.upsertValue); got != "sk-file-key" {
		t.Fatalf("expected trimmed value %q, got %q", "sk-file-key", got)
	}
	if mock.upsertType != vault.SecretTypeAPIKey {
		t.Fatalf("expected secret type %q, got %q", vault.SecretTypeAPIKey, mock.upsertType)
	}
}

func TestLinkStdinOAuthServiceReturnsError(t *testing.T) {
	info := linkServiceInfo{
		Service:       "github",
		AuthType:      actions.AuthTypeOAuth2,
		CredentialRef: "github.oauth_token",
		OAuthProvider: "github",
	}
	oauthStates := []connectorStateRow{}

	err := linkRejectStdinFileForOAuth(info, oauthStates, true, "")
	if err == nil {
		t.Fatal("expected error for OAuth service with --stdin")
	}
	if !strings.Contains(err.Error(), "OAuth") {
		t.Fatalf("expected error to mention OAuth, got: %v", err)
	}
	if !strings.Contains(err.Error(), "github") {
		t.Fatalf("expected error to mention service name, got: %v", err)
	}
}

func TestLinkStdinFileOAuthReturnsErrorForFile(t *testing.T) {
	info := linkServiceInfo{
		Service:       "github",
		AuthType:      actions.AuthTypeOAuth2,
		CredentialRef: "github.oauth_token",
		OAuthProvider: "github",
	}
	oauthStates := []connectorStateRow{}

	err := linkRejectStdinFileForOAuth(info, oauthStates, false, "/path/to/key.txt")
	if err == nil {
		t.Fatal("expected error for OAuth service with --file")
	}
	if !strings.Contains(err.Error(), "OAuth") {
		t.Fatalf("expected error to mention OAuth, got: %v", err)
	}
}

func TestLinkWithoutStdinFilePreservesExistingBehavior(t *testing.T) {
	info := linkServiceInfo{
		Service:       "github",
		AuthType:      actions.AuthTypeOAuth2,
		CredentialRef: "github.oauth_token",
		OAuthProvider: "github",
	}
	oauthStates := []connectorStateRow{}

	err := linkRejectStdinFileForOAuth(info, oauthStates, false, "")
	if err != nil {
		t.Fatalf("expected no error when --stdin/--file not set, got: %v", err)
	}
}

func TestLinkAuthTypeToSecretType(t *testing.T) {
	tests := []struct {
		authType actions.AuthType
		want     vault.SecretType
	}{
		{actions.AuthTypeBearer, vault.SecretTypeBearerToken},
		{actions.AuthTypeAPIKey, vault.SecretTypeAPIKey},
		{actions.AuthTypeBasic, vault.SecretTypePassword},
		{actions.AuthTypeHeader, vault.SecretTypeAPIKey},
		{actions.AuthTypeQuery, vault.SecretTypeAPIKey},
		{actions.AuthType("unknown"), vault.SecretTypeAPIKey},
	}

	for _, tt := range tests {
		got := linkAuthTypeToSecretType(tt.authType)
		if got != tt.want {
			t.Errorf("linkAuthTypeToSecretType(%q) = %q, want %q", tt.authType, got, tt.want)
		}
	}
}

func TestLinkCommandHasStdinAndFileFlags(t *testing.T) {
	cmd := newLinkCommand()
	if cmd.Flags().Lookup("stdin") == nil {
		t.Fatal("expected --stdin flag on link command")
	}
	if cmd.Flags().Lookup("file") == nil {
		t.Fatal("expected --file flag on link command")
	}
}

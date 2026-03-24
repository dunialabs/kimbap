package vault

import (
	"time"

	corecrypto "github.com/dunialabs/kimbap-core/internal/crypto"
)

type SecretRecord struct {
	ID             string
	TenantID       string
	Name           string
	Type           SecretType
	Labels         map[string]string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastUsedAt     *time.Time
	RotatedAt      *time.Time
	VersionCount   int
	CurrentVersion int
}

type SecretVersion struct {
	ID        string
	SecretID  string
	Version   int
	Envelope  corecrypto.EncryptedEnvelope
	CreatedAt time.Time
	CreatedBy string
	Active    bool
}

type SecretType string

const (
	SecretTypeAPIKey       SecretType = "api_key"
	SecretTypeBearerToken  SecretType = "bearer_token"
	SecretTypeOAuthClient  SecretType = "oauth_client"
	SecretTypePassword     SecretType = "password"
	SecretTypeRefreshToken SecretType = "refresh_token"
	SecretTypeCertificate  SecretType = "certificate"
)

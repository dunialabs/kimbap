package vault

import (
	"time"
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

type SecretType string

const (
	SecretTypeAPIKey       SecretType = "api_key"
	SecretTypeBearerToken  SecretType = "bearer_token"
	SecretTypeOAuthClient  SecretType = "oauth_client"
	SecretTypePassword     SecretType = "password"
	SecretTypeRefreshToken SecretType = "refresh_token"
	SecretTypeCertificate  SecretType = "certificate"
)

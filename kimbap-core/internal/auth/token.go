package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const serviceTokenPrefix = "ktk_"

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
	ErrRevokedToken = errors.New("revoked token")
)

type ServiceToken struct {
	ID          string
	TenantID    string
	AgentName   string
	TokenHash   string
	DisplayHint string
	Scopes      []string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	CreatedBy   string
}

type TokenStore interface {
	Create(ctx context.Context, token *ServiceToken) error
	ValidateAndResolve(ctx context.Context, rawToken string) (*Principal, error)
	Revoke(ctx context.Context, tokenID string) error
	List(ctx context.Context, tenantID string) ([]ServiceToken, error)
	Inspect(ctx context.Context, tokenID string) (*ServiceToken, error)
	MarkUsed(ctx context.Context, tokenID string) error
}

type TokenService struct {
	store TokenStore
}

func NewTokenService(store TokenStore) *TokenService {
	return &TokenService{store: store}
}

var (
	ErrInvalidTTL = errors.New("invalid ttl: must be a positive duration no longer than 365 days")

	maxTokenTTL = 365 * 24 * time.Hour
)

func (s *TokenService) Issue(ctx context.Context, tenantID, agentName string, scopes []string, ttl time.Duration) (rawToken string, token *ServiceToken, err error) {
	if s == nil || s.store == nil {
		return "", nil, errors.New("token store is required")
	}
	if strings.TrimSpace(tenantID) == "" {
		return "", nil, errors.New("tenant id is required")
	}
	if strings.TrimSpace(agentName) == "" {
		return "", nil, errors.New("agent name is required")
	}
	if ttl <= 0 || ttl > maxTokenTTL {
		return "", nil, ErrInvalidTTL
	}

	now := time.Now().UTC()
	rawToken, err = generatePrefixedHexToken(serviceTokenPrefix, 32)
	if err != nil {
		return "", nil, err
	}

	tokenID, err := generatePrefixedHexToken("st_", 16)
	if err != nil {
		return "", nil, err
	}

	token = &ServiceToken{
		ID:          tokenID,
		TenantID:    tenantID,
		AgentName:   agentName,
		TokenHash:   hashToken(rawToken),
		DisplayHint: rawToken[len(rawToken)-4:],
		Scopes:      append([]string(nil), scopes...),
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
	}

	if err := s.store.Create(ctx, token); err != nil {
		return "", nil, err
	}

	return rawToken, token, nil
}

func (s *TokenService) Validate(ctx context.Context, rawToken string) (*Principal, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("token store is required")
	}
	if !strings.HasPrefix(rawToken, serviceTokenPrefix) {
		return nil, ErrInvalidToken
	}

	principal, err := s.store.ValidateAndResolve(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	if principal == nil {
		return nil, ErrInvalidToken
	}

	if principal.TokenID != "" {
		if err := s.store.MarkUsed(ctx, principal.TokenID); err != nil {
			return nil, err
		}
	}

	return principal, nil
}

func (s *TokenService) Rotate(ctx context.Context, oldTokenID, tenantID, agentName string, scopes []string, ttl time.Duration) (rawToken string, token *ServiceToken, err error) {
	if s == nil || s.store == nil {
		return "", nil, errors.New("token store is required")
	}
	if strings.TrimSpace(oldTokenID) == "" {
		return "", nil, errors.New("old token id is required")
	}

	rawToken, token, err = s.Issue(ctx, tenantID, agentName, scopes, ttl)
	if err != nil {
		return "", nil, fmt.Errorf("issue replacement token: %w", err)
	}

	if revokeErr := s.store.Revoke(ctx, oldTokenID); revokeErr != nil {
		return rawToken, token, fmt.Errorf("new token issued but old token revocation failed: %w", revokeErr)
	}

	return rawToken, token, nil
}

func (s *TokenService) Revoke(ctx context.Context, tokenID string) error {
	if s == nil || s.store == nil {
		return errors.New("token store is required")
	}
	if strings.TrimSpace(tokenID) == "" {
		return errors.New("token id is required")
	}
	return s.store.Revoke(ctx, tokenID)
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func generatePrefixedHexToken(prefix string, randomBytes int) (string, error) {
	b := make([]byte, randomBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return prefix + hex.EncodeToString(b), nil
}

//go:build ignore

package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTokenServiceIssueReturnsPrefixedRawToken(t *testing.T) {
	store := newInMemoryTokenStore()
	svc := NewTokenService(store)

	rawToken, token, err := svc.Issue(context.Background(), "tenant-a", "agent-alpha", []string{"tools:read"}, time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	if !strings.HasPrefix(rawToken, serviceTokenPrefix) {
		t.Fatalf("expected token prefix %q, got %q", serviceTokenPrefix, rawToken)
	}
	if token == nil {
		t.Fatal("expected token metadata")
	}
	if token.DisplayHint != rawToken[len(rawToken)-4:] {
		t.Fatalf("display hint mismatch: got=%q want=%q", token.DisplayHint, rawToken[len(rawToken)-4:])
	}
}

func TestTokenServiceValidateSucceedsWithCorrectRawToken(t *testing.T) {
	store := newInMemoryTokenStore()
	svc := NewTokenService(store)

	rawToken, issued, err := svc.Issue(context.Background(), "tenant-a", "agent-alpha", []string{"tools:read"}, time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	principal, err := svc.Validate(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if principal.TokenID != issued.ID {
		t.Fatalf("expected token id %q, got %q", issued.ID, principal.TokenID)
	}
}

func TestTokenServiceValidateFailsWithWrongToken(t *testing.T) {
	store := newInMemoryTokenStore()
	svc := NewTokenService(store)

	_, _, err := svc.Issue(context.Background(), "tenant-a", "agent-alpha", nil, time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	_, err = svc.Validate(context.Background(), "ktk_nottherighttoken")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestTokenServiceValidateFailsAfterRevocation(t *testing.T) {
	store := newInMemoryTokenStore()
	svc := NewTokenService(store)

	rawToken, issued, err := svc.Issue(context.Background(), "tenant-a", "agent-alpha", nil, time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if err := svc.Revoke(context.Background(), issued.ID); err != nil {
		t.Fatalf("revoke token: %v", err)
	}

	_, err = svc.Validate(context.Background(), rawToken)
	if !errors.Is(err, ErrRevokedToken) {
		t.Fatalf("expected ErrRevokedToken, got %v", err)
	}
}

func TestTokenServiceValidateFailsWhenExpired(t *testing.T) {
	store := newInMemoryTokenStore()
	svc := NewTokenService(store)

	rawToken, issued, err := svc.Issue(context.Background(), "tenant-a", "agent-alpha", nil, time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	store.mu.Lock()
	tok := store.tokens[issued.ID]
	tok.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	store.tokens[issued.ID] = tok
	store.mu.Unlock()

	_, err = svc.Validate(context.Background(), rawToken)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

type inMemoryTokenStore struct {
	mu     sync.Mutex
	tokens map[string]ServiceToken
}

func newInMemoryTokenStore() *inMemoryTokenStore {
	return &inMemoryTokenStore{tokens: map[string]ServiceToken{}}
}

func (s *inMemoryTokenStore) Create(_ context.Context, token *ServiceToken) error {
	if token == nil {
		return errors.New("token is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token.ID] = *token
	return nil
}

func (s *inMemoryTokenStore) ValidateAndResolve(_ context.Context, rawToken string) (*Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := hashToken(rawToken)
	now := time.Now().UTC()
	for _, token := range s.tokens {
		if token.TokenHash != hash {
			continue
		}
		if token.RevokedAt != nil {
			return nil, ErrRevokedToken
		}
		if now.After(token.ExpiresAt) {
			return nil, ErrExpiredToken
		}
		principal := &Principal{
			ID:        token.AgentName,
			Type:      PrincipalTypeService,
			TenantID:  token.TenantID,
			AgentName: token.AgentName,
			Scopes:    append([]string(nil), token.Scopes...),
			TokenID:   token.ID,
			IssuedAt:  token.CreatedAt,
			ExpiresAt: token.ExpiresAt,
		}
		return principal, nil
	}

	return nil, ErrInvalidToken
}

func (s *inMemoryTokenStore) Revoke(_ context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.tokens[tokenID]
	if !ok {
		return ErrInvalidToken
	}
	now := time.Now().UTC()
	token.RevokedAt = &now
	s.tokens[tokenID] = token
	return nil
}

func (s *inMemoryTokenStore) List(_ context.Context, tenantID string) ([]ServiceToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make([]ServiceToken, 0)
	for _, token := range s.tokens {
		if token.TenantID == tenantID {
			res = append(res, token)
		}
	}
	return res, nil
}

func (s *inMemoryTokenStore) Inspect(_ context.Context, tokenID string) (*ServiceToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.tokens[tokenID]
	if !ok {
		return nil, ErrInvalidToken
	}
	tok := token
	return &tok, nil
}

func (s *inMemoryTokenStore) MarkUsed(_ context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.tokens[tokenID]
	if !ok {
		return ErrInvalidToken
	}
	now := time.Now().UTC()
	token.LastUsedAt = &now
	s.tokens[tokenID] = token
	return nil
}

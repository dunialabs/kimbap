package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type revokeFailTokenStore struct {
	*inMemoryTokenStore
	failTokenID string
}

func (s *revokeFailTokenStore) Revoke(ctx context.Context, tokenID string) error {
	if tokenID == s.failTokenID {
		return errors.New("forced revoke failure")
	}
	return s.inMemoryTokenStore.Revoke(ctx, tokenID)
}

func TestTokenServiceRotateRollsBackReplacementTokenWhenOldRevokeFails(t *testing.T) {
	ctx := context.Background()
	baseStore := newInMemoryTokenStore()
	issuer := NewTokenService(baseStore)

	oldRawToken, oldToken, err := issuer.Issue(ctx, "tenant-a", "agent-alpha", []string{"tools:read"}, time.Hour)
	if err != nil {
		t.Fatalf("issue original token: %v", err)
	}

	rotateService := NewTokenService(&revokeFailTokenStore{
		inMemoryTokenStore: baseStore,
		failTokenID:        oldToken.ID,
	})

	rawToken, replacement, err := rotateService.Rotate(ctx, oldToken.ID, "tenant-a", "agent-alpha", []string{"tools:read"}, time.Hour)
	if err == nil {
		t.Fatal("expected rotate to fail when old token revoke fails")
	}
	if rawToken != "" || replacement != nil {
		t.Fatalf("expected failed rotation to return no replacement token, got raw=%q replacement=%#v", rawToken, replacement)
	}
	if !strings.Contains(err.Error(), "revoke old token") {
		t.Fatalf("expected error to mention old token revoke failure, got %v", err)
	}

	principal, err := rotateService.Validate(ctx, oldRawToken)
	if err != nil {
		t.Fatalf("expected original token to remain valid, got %v", err)
	}
	if principal.TokenID != oldToken.ID {
		t.Fatalf("expected original token id %q, got %q", oldToken.ID, principal.TokenID)
	}

	tokens, err := baseStore.List(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected original token plus revoked replacement record, got %d tokens", len(tokens))
	}

	var revokedReplacementCount int
	for _, token := range tokens {
		if token.ID == oldToken.ID {
			if token.RevokedAt != nil {
				t.Fatalf("expected original token to remain active, got revoked_at=%v", token.RevokedAt)
			}
			continue
		}
		if token.RevokedAt == nil {
			t.Fatalf("expected replacement token %q to be revoked during rollback", token.ID)
		}
		revokedReplacementCount++
	}
	if revokedReplacementCount != 1 {
		t.Fatalf("expected exactly one revoked replacement token, got %d", revokedReplacementCount)
	}
}

package authstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/auth"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/storeconv"
)

type TokenStoreAdapter struct {
	st store.TokenStore
}

func NewTokenStoreAdapter(st store.TokenStore) *TokenStoreAdapter {
	return &TokenStoreAdapter{st: st}
}

func (a *TokenStoreAdapter) Create(ctx context.Context, token *auth.ServiceToken) error {
	if token == nil {
		return errors.New("token is required")
	}
	return a.st.CreateToken(ctx, storeconv.TokenRecordFromServiceToken(token))
}

func (a *TokenStoreAdapter) ValidateAndResolve(ctx context.Context, rawToken string) (*auth.Principal, error) {
	hash := sha256.Sum256([]byte(rawToken))
	token, err := a.st.GetTokenByHash(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}
	now := time.Now().UTC()
	if token.RevokedAt != nil {
		return nil, auth.ErrRevokedToken
	}
	if now.After(token.ExpiresAt) {
		return nil, auth.ErrExpiredToken
	}
	return storeconv.PrincipalFromTokenRecord(token)
}

func (a *TokenStoreAdapter) Revoke(ctx context.Context, tokenID string) error {
	if strings.TrimSpace(tokenID) == "" {
		return auth.ErrInvalidToken
	}
	err := a.st.RevokeToken(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
}

func (a *TokenStoreAdapter) List(ctx context.Context, tenantID string) ([]auth.ServiceToken, error) {
	items, err := a.st.ListTokens(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]auth.ServiceToken, 0, len(items))
	for i := range items {
		out = append(out, storeconv.ServiceTokenFromRecord(items[i]))
	}
	return out, nil
}

func (a *TokenStoreAdapter) Inspect(ctx context.Context, tokenID string) (*auth.ServiceToken, error) {
	it, err := a.st.GetToken(ctx, tokenID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}
	converted := storeconv.ServiceTokenFromRecord(*it)
	return &converted, nil
}

func (a *TokenStoreAdapter) MarkUsed(ctx context.Context, tokenID string) error {
	err := a.st.UpdateTokenLastUsed(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
}

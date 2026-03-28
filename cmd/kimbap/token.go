package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/auth"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/spf13/cobra"
)

func newTokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage service tokens",
	}

	cmd.AddCommand(newTokenCreateCommand())
	cmd.AddCommand(newTokenInspectCommand())
	cmd.AddCommand(newTokenRevokeCommand())
	cmd.AddCommand(newTokenListCommand())

	return cmd
}

func newTokenCreateCommand() *cobra.Command {
	var (
		agent  string
		ttl    time.Duration
		scopes string
		tenant string
	)
	cmd := &cobra.Command{
		Use:   "create --agent <name> [--ttl <duration>] [--scopes <s1,s2>]",
		Short: "Issue a new service token",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(agent) == "" {
				return fmt.Errorf("--agent is required")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			tokenService, tokenStore, err := newTokenService(cfg)
			if err != nil {
				return err
			}
			defer tokenStore.Close()

			if strings.TrimSpace(tenant) == "" {
				tenant = defaultTenantID()
			}

			if ttl == 0 {
				ttl = 30 * 24 * time.Hour
			} else if ttl < 0 {
				return auth.ErrInvalidTTL
			}
			raw, tok, err := tokenService.Issue(contextBackground(), tenant, agent, "", parseCSV(scopes), ttl)
			if err != nil {
				return err
			}
			return printOutput(map[string]any{
				"token_id":   tok.ID,
				"tenant_id":  tok.TenantID,
				"agent":      tok.AgentName,
				"expires_at": tok.ExpiresAt,
				"scopes":     tok.Scopes,
				"raw_token":  raw,
				"note":       "Raw token is shown once. Store it securely.",
			})
		},
	}
	cmd.Flags().StringVar(&agent, "agent", "", "agent name")
	cmd.Flags().DurationVar(&ttl, "ttl", 0, "token ttl (e.g. 24h)")
	cmd.Flags().StringVar(&scopes, "scopes", "", "comma-separated scopes")
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id (default from env or 'default')")
	return cmd
}

func newTokenInspectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <token-id>",
		Short: "Inspect token metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			_, store, err := newTokenService(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			tok, err := store.Inspect(contextBackground(), args[0])
			if err != nil {
				return err
			}
			return printOutput(tok)
		},
	}
	return cmd
}

func newTokenRevokeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke <token-id>",
		Short: "Revoke a token immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			tokenService, tokenStore, err := newTokenService(cfg)
			if err != nil {
				return err
			}
			defer tokenStore.Close()
			if err := tokenService.Revoke(contextBackground(), args[0]); err != nil {
				return err
			}
			return printOutput(map[string]any{"revoked": true, "token_id": args[0]})
		},
	}
	return cmd
}

func newTokenListCommand() *cobra.Command {
	var tenant string
	cmd := &cobra.Command{
		Use:   "list [--tenant <id>]",
		Short: "List token metadata",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			_, store, err := newTokenService(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			if strings.TrimSpace(tenant) == "" {
				tenant = defaultTenantID()
			}
			tokens, err := store.List(contextBackground(), tenant)
			if err != nil {
				return err
			}
			return printOutput(tokens)
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	return cmd
}

func newTokenService(cfg *config.KimbapConfig) (*auth.TokenService, *sqlTokenStoreAdapter, error) {
	st, err := openRuntimeStore(cfg)
	if err != nil {
		return nil, nil, err
	}
	adapter := &sqlTokenStoreAdapter{st: st}
	return auth.NewTokenService(adapter), adapter, nil
}

type sqlTokenStoreAdapter struct {
	st *store.SQLStore
}

func (a *sqlTokenStoreAdapter) Close() error {
	if a == nil || a.st == nil {
		return nil
	}
	return a.st.Close()
}

func (a *sqlTokenStoreAdapter) Create(ctx context.Context, token *auth.ServiceToken) error {
	if token == nil {
		return errors.New("token is required")
	}
	return a.st.CreateToken(ctx, &store.TokenRecord{
		ID:          token.ID,
		TenantID:    token.TenantID,
		AgentName:   token.AgentName,
		TokenHash:   token.TokenHash,
		DisplayHint: token.DisplayHint,
		Scopes:      marshalScopes(token.Scopes),
		CreatedAt:   token.CreatedAt,
		ExpiresAt:   token.ExpiresAt,
		LastUsedAt:  token.LastUsedAt,
		RevokedAt:   token.RevokedAt,
		CreatedBy:   token.CreatedBy,
	})

}

func (a *sqlTokenStoreAdapter) ValidateAndResolve(ctx context.Context, rawToken string) (*auth.Principal, error) {
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
	return &auth.Principal{
		ID:        token.AgentName,
		Type:      auth.PrincipalTypeService,
		TenantID:  token.TenantID,
		AgentName: token.AgentName,
		Scopes:    parseScopes(token.Scopes),
		TokenID:   token.ID,
		IssuedAt:  token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
	}, nil
}

func (a *sqlTokenStoreAdapter) Revoke(ctx context.Context, tokenID string) error {
	if strings.TrimSpace(tokenID) == "" {
		return auth.ErrInvalidToken
	}
	err := a.st.RevokeToken(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
}

func (a *sqlTokenStoreAdapter) List(ctx context.Context, tenantID string) ([]auth.ServiceToken, error) {
	items, err := a.st.ListTokens(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]auth.ServiceToken, 0, len(items))
	for i := range items {
		it := items[i]
		out = append(out, auth.ServiceToken{
			ID:          it.ID,
			TenantID:    it.TenantID,
			AgentName:   it.AgentName,
			TokenHash:   it.TokenHash,
			DisplayHint: it.DisplayHint,
			Scopes:      parseScopes(it.Scopes),
			CreatedAt:   it.CreatedAt,
			ExpiresAt:   it.ExpiresAt,
			LastUsedAt:  it.LastUsedAt,
			RevokedAt:   it.RevokedAt,
			CreatedBy:   it.CreatedBy,
		})
	}
	return out, nil
}

func (a *sqlTokenStoreAdapter) Inspect(ctx context.Context, tokenID string) (*auth.ServiceToken, error) {
	it, err := a.st.GetToken(ctx, tokenID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}
	return &auth.ServiceToken{
		ID:          it.ID,
		TenantID:    it.TenantID,
		AgentName:   it.AgentName,
		TokenHash:   it.TokenHash,
		DisplayHint: it.DisplayHint,
		Scopes:      parseScopes(it.Scopes),
		CreatedAt:   it.CreatedAt,
		ExpiresAt:   it.ExpiresAt,
		LastUsedAt:  it.LastUsedAt,
		RevokedAt:   it.RevokedAt,
		CreatedBy:   it.CreatedBy,
	}, nil
}

func (a *sqlTokenStoreAdapter) MarkUsed(ctx context.Context, tokenID string) error {
	err := a.st.UpdateTokenLastUsed(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
}

func marshalScopes(scopes []string) string {
	if len(scopes) == 0 {
		return "[]"
	}
	b, err := json.Marshal(scopes)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func parseScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var scopes []string
	if err := json.Unmarshal([]byte(raw), &scopes); err != nil {
		return nil
	}
	return scopes
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

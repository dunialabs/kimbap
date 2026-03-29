package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"os"

	"github.com/dunialabs/kimbap/internal/auth"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/storeconv"
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
				return fmt.Errorf("--agent is required\nRun: kimbap token create --agent <name>")
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
			if outputAsJSON() {
				return printOutput(map[string]any{
					"token_id":   tok.ID,
					"tenant_id":  tok.TenantID,
					"agent":      tok.AgentName,
					"expires_at": tok.ExpiresAt,
					"scopes":     tok.Scopes,
					"raw_token":  raw,
					"note":       "Raw token is shown once. Store it securely.",
				})
			}
			scopes := strings.Join(tok.Scopes, ", ")
			if scopes == "" {
				scopes = "all"
			}
			warning := "!"
			if isColorStdout() {
				warning = "\x1b[33m!\x1b[0m"
			}
			_, _ = fmt.Fprintf(os.Stdout, "%s\n\n  Agent:    %s\n  Expires:  %s\n  Scopes:   %s\n\n%s Store this token securely — it will not be shown again.\n",
				raw,
				tok.AgentName,
				tok.ExpiresAt.UTC().Format("2006-01-02 15:04 UTC"),
				scopes,
				warning,
			)
			return nil
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
			if outputAsJSON() {
				return printOutput(tokens)
			}
			if len(tokens) == 0 {
				_, _ = fmt.Fprintln(os.Stdout, "No tokens found.")
				return nil
			}
			fmt.Printf("%-16s %-14s %-20s %-20s %s\n", "HINT", "AGENT", "EXPIRES", "LAST USED", "SCOPES")
			useColor := isColorStdout()
			now := time.Now().UTC()
			for _, tok := range tokens {
				lastUsed := "-"
				if tok.LastUsedAt != nil {
					lastUsed = tok.LastUsedAt.UTC().Format("2006-01-02 15:04")
				}
				scopes := strings.Join(tok.Scopes, ",")
				if scopes == "" {
					scopes = "all"
				}
				expiresStr := fmt.Sprintf("%-20s", tok.ExpiresAt.UTC().Format("2006-01-02 15:04"))
				if useColor {
					if tok.ExpiresAt.Before(now) {
						expiresStr = "\x1b[31m" + expiresStr + "\x1b[0m"
					} else if tok.ExpiresAt.Before(now.Add(7 * 24 * time.Hour)) {
						expiresStr = "\x1b[33m" + expiresStr + "\x1b[0m"
					}
				}
				fmt.Printf("%-16s %-14s %s %-20s %s\n",
					tok.DisplayHint,
					tok.AgentName,
					expiresStr,
					lastUsed,
					scopes,
				)
			}
			return nil
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
	return a.st.CreateToken(ctx, storeconv.TokenRecordFromServiceToken(token))

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
	return storeconv.PrincipalFromTokenRecord(token)
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
		out = append(out, storeconv.ServiceTokenFromRecord(items[i]))
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
	converted := storeconv.ServiceTokenFromRecord(*it)
	return &converted, nil
}

func (a *sqlTokenStoreAdapter) MarkUsed(ctx context.Context, tokenID string) error {
	err := a.st.UpdateTokenLastUsed(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
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

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/auth"
	"github.com/dunialabs/kimbap-core/internal/config"
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
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(agent) == "" {
				return fmt.Errorf("--agent is required")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			tokenService, _, err := newTokenService(cfg)
			if err != nil {
				return err
			}

			if strings.TrimSpace(tenant) == "" {
				tenant = defaultTenantID()
			}

			if ttl <= 0 {
				ttl = 30 * 24 * time.Hour
			}
			raw, tok, err := tokenService.Issue(contextBackground(), tenant, agent, parseCSV(scopes), ttl)
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
			tokenService, _, err := newTokenService(cfg)
			if err != nil {
				return err
			}
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
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			_, store, err := newTokenService(cfg)
			if err != nil {
				return err
			}

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

func newTokenService(cfg *config.KimbapConfig) (*auth.TokenService, *fileTokenStore, error) {
	return newTokenServiceFromConfig(cfg.DataDir)
}

type fileTokenStore struct {
	path string
	mu   sync.Mutex
}

type tokenStoreData struct {
	Tokens map[string]auth.ServiceToken `json:"tokens"`
}

func newTokenServiceFromConfig(cfgDataDir string) (*auth.TokenService, *fileTokenStore, error) {
	store := &fileTokenStore{path: filepath.Join(cfgDataDir, "tokens.json")}
	if err := store.ensureFile(); err != nil {
		return nil, nil, err
	}
	return auth.NewTokenService(store), store, nil
}

func (s *fileTokenStore) ensureFile() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		initial := tokenStoreData{Tokens: map[string]auth.ServiceToken{}}
		b, mErr := json.Marshal(initial)
		if mErr != nil {
			return mErr
		}
		return os.WriteFile(s.path, b, 0o600)
	}
	return nil
}

func (s *fileTokenStore) load() (*tokenStoreData, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	data := tokenStoreData{Tokens: map[string]auth.ServiceToken{}}
	if len(b) == 0 {
		return &data, nil
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	if data.Tokens == nil {
		data.Tokens = map[string]auth.ServiceToken{}
	}
	return &data, nil
}

func (s *fileTokenStore) save(data *tokenStoreData) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func (s *fileTokenStore) Create(_ context.Context, token *auth.ServiceToken) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return err
	}
	data.Tokens[token.ID] = *token
	return s.save(data)
}

func (s *fileTokenStore) ValidateAndResolve(_ context.Context, rawToken string) (*auth.Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return nil, err
	}

	hash := hashRawToken(rawToken)
	now := time.Now().UTC()
	for _, token := range data.Tokens {
		if token.TokenHash != hash {
			continue
		}
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
			Scopes:    append([]string(nil), token.Scopes...),
			TokenID:   token.ID,
			IssuedAt:  token.CreatedAt,
			ExpiresAt: token.ExpiresAt,
		}, nil
	}

	return nil, auth.ErrInvalidToken
}

func (s *fileTokenStore) Revoke(_ context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return err
	}
	tok, ok := data.Tokens[tokenID]
	if !ok {
		return auth.ErrInvalidToken
	}
	now := time.Now().UTC()
	tok.RevokedAt = &now
	data.Tokens[tokenID] = tok
	return s.save(data)
}

func (s *fileTokenStore) List(_ context.Context, tenantID string) ([]auth.ServiceToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return nil, err
	}

	out := make([]auth.ServiceToken, 0)
	for _, token := range data.Tokens {
		if strings.TrimSpace(tenantID) == "" || token.TenantID == tenantID {
			out = append(out, token)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *fileTokenStore) Inspect(_ context.Context, tokenID string) (*auth.ServiceToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return nil, err
	}
	tok, ok := data.Tokens[tokenID]
	if !ok {
		return nil, auth.ErrInvalidToken
	}
	out := tok
	return &out, nil
}

func (s *fileTokenStore) MarkUsed(_ context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return err
	}
	tok, ok := data.Tokens[tokenID]
	if !ok {
		return auth.ErrInvalidToken
	}
	now := time.Now().UTC()
	tok.LastUsedAt = &now
	data.Tokens[tokenID] = tok
	return s.save(data)
}

func hashRawToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
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

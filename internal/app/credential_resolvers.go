package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/connectors"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/rs/zerolog/log"
)

type vaultCredentialResolver struct {
	store vault.Store
}

func (r *vaultCredentialResolver) Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	if r == nil || r.store == nil {
		return nil, fmt.Errorf("vault store is not initialized")
	}
	secretName := strings.TrimSpace(req.CredentialRef)
	if secretName == "" {
		if req.Optional {
			return nil, nil
		}
		return nil, fmt.Errorf("credential_ref is required")
	}

	raw, err := r.store.GetValue(ctx, tenantID, secretName)
	if err != nil {
		if errors.Is(err, vault.ErrSecretNotFound) {
			return nil, nil
		}
		return nil, err
	}
	set := parseCredentialSet(raw, req)
	if set == nil && !req.Optional {
		return nil, fmt.Errorf("credential %q is empty", secretName)
	}
	if set != nil {
		if err := r.store.MarkUsed(ctx, tenantID, secretName); err != nil {
			log.Warn().Err(err).Str("tenantID", tenantID).Str("secret", secretName).Msg("failed to mark credential as used")
		}
	}
	return set, nil
}

func parseCredentialSet(raw []byte, req actions.AuthRequirement) *actions.ResolvedCredentialSet {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil
	}

	var direct actions.ResolvedCredentialSet
	if err := json.Unmarshal(raw, &direct); err == nil {
		if direct.Type == "" {
			direct.Type = string(req.Type)
		}
		return &direct
	}

	set := &actions.ResolvedCredentialSet{Type: string(req.Type)}
	if idx := strings.Index(trimmed, ":"); req.Type == actions.AuthTypeBasic && idx > 0 {
		set.Username = trimmed[:idx]
		set.Password = trimmed[idx+1:]
		return set
	}

	switch req.Type {
	case actions.AuthTypeBearer, actions.AuthTypeOAuth2, actions.AuthTypeSession:
		set.Token = trimmed
	case actions.AuthTypeAPIKey, actions.AuthTypeHeader, actions.AuthTypeQuery, actions.AuthTypeBody:
		set.APIKey = trimmed
		set.Token = trimmed
	case actions.AuthTypeBasic:
		set.Username = trimmed
	default:
		set.Token = trimmed
		set.APIKey = trimmed
	}
	return set
}

type connectorCredentialResolver struct {
	mgr *connectors.Manager
}

func (r *connectorCredentialResolver) Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	if r == nil || r.mgr == nil {
		return nil, nil
	}
	if req.Type != actions.AuthTypeBearer && req.Type != actions.AuthTypeOAuth2 && req.Type != actions.AuthTypeSession {
		return nil, nil
	}
	connectorName := strings.TrimSpace(req.CredentialRef)
	if connectorName == "" {
		return nil, nil
	}

	connectorName, ok := connectorNameFromRef(connectorName)
	if !ok {
		return nil, nil
	}

	token, err := r.mgr.GetAccessToken(ctx, tenantID, connectorName)
	if err != nil {
		if errors.Is(err, connectors.ErrConnectorNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	return &actions.ResolvedCredentialSet{
		Type:  string(req.Type),
		Token: token,
	}, nil
}

type envCredentialResolver struct{}

func (r *envCredentialResolver) Resolve(_ context.Context, _ string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	if req.Type != actions.AuthTypeBearer && req.Type != actions.AuthTypeOAuth2 && req.Type != actions.AuthTypeSession {
		return nil, nil
	}

	connectorName := strings.TrimSpace(req.CredentialRef)
	if connectorName == "" {
		return nil, nil
	}

	connectorName, ok := connectorNameFromRef(connectorName)
	if !ok {
		return nil, nil
	}

	envKey := "KIMBAP_" + toEnvSegment(connectorName) + "_TOKEN"
	token := strings.TrimSpace(os.Getenv(envKey))
	if token == "" {
		return nil, nil
	}

	return &actions.ResolvedCredentialSet{Type: string(req.Type), Token: token}, nil
}

func connectorNameFromRef(ref string) (string, bool) {
	suffixes := []string{".oauth_token", ".oidc_token", ".token", ".oauth"}
	lower := strings.ToLower(ref)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			name := strings.TrimSpace(ref[:len(ref)-len(suffix)])
			if name != "" {
				return name, true
			}
			return "", false
		}
	}
	return "", false
}

func toEnvSegment(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

type chainCredentialResolver struct {
	resolvers []runtimepkg.CredentialResolver
}

func (c *chainCredentialResolver) Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	for _, r := range c.resolvers {
		creds, err := r.Resolve(ctx, tenantID, req)
		if err != nil {
			return nil, err
		}
		if creds != nil {
			return creds, nil
		}
	}
	return nil, nil
}

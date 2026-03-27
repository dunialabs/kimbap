package clientcred

import (
	"context"
	"errors"
	"strings"

	"github.com/dunialabs/kimbap/internal/connectors"
)

type ClientCredentialsConfig struct {
	TokenEndpoint string
	ClientID      string
	ClientSecret  string
	Scopes        []string
	AuthMethod    string
}

type ClientCredentialsResult struct {
	AccessToken string
	ExpiresIn   int
	Scope       string
}

func RunClientCredentialsFlow(ctx context.Context, cfg ClientCredentialsConfig) (*ClientCredentialsResult, error) {
	if strings.TrimSpace(cfg.TokenEndpoint) == "" {
		return nil, errors.New("token endpoint is required")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, errors.New("client id is required")
	}

	token, err := connectors.RequestClientCredentialsTokenWithContext(ctx, connectors.ConnectorConfig{
		TokenURL:     cfg.TokenEndpoint,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		AuthMethod:   cfg.AuthMethod,
		Scopes:       cfg.Scopes,
	})
	if err != nil {
		return nil, err
	}

	return &ClientCredentialsResult{
		AccessToken: token.AccessToken,
		ExpiresIn:   token.ExpiresIn,
		Scope:       token.Scope,
	}, nil
}

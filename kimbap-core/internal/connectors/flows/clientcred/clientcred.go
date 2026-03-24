package clientcred

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxTokenResponseBytes int64 = 4 << 20

type ClientCredentialsConfig struct {
	TokenEndpoint string
	ClientID      string
	ClientSecret  string
	Scopes        []string
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

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	if len(cfg.Scopes) > 0 {
		form.Set("scope", strings.Join(cfg.Scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute token request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxTokenResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if int64(len(body)) > maxTokenResponseBytes {
		return nil, fmt.Errorf("token response exceeded %d bytes", maxTokenResponseBytes)
	}

	var payload struct {
		AccessToken      string `json:"access_token"`
		ExpiresIn        int    `json:"expires_in"`
		Scope            string `json:"scope"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		if payload.Error != "" && payload.ErrorDescription != "" {
			return nil, fmt.Errorf("oauth token error: %s (%s)", payload.Error, payload.ErrorDescription)
		}
		if payload.Error != "" {
			return nil, fmt.Errorf("oauth token error: %s", payload.Error)
		}
		return nil, fmt.Errorf("token endpoint returned status %d", res.StatusCode)
	}

	if payload.AccessToken == "" {
		return nil, errors.New("token response missing access_token")
	}

	return &ClientCredentialsResult{
		AccessToken: payload.AccessToken,
		ExpiresIn:   payload.ExpiresIn,
		Scope:       payload.Scope,
	}, nil
}

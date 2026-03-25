package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	zendeskDefaultExpiresIn             = int64(172800)  // 2 days
	zendeskDefaultRefreshTokenExpiresIn = int64(7776000) // 90 days
)

type ZendeskAuthStrategy struct {
	state  strategyState
	client *http.Client
	config zendeskOAuthConfig
}

type zendeskOAuthConfig struct {
	ClientID                     string
	ClientSecret                 string
	RefreshToken                 string
	TokenURL                     string
	Scope                        string
	AccessToken                  string
	ExpiresAt                    int64
	RefreshTokenExpiresAt        int64
	ExpiresInSeconds             int64
	RefreshTokenExpiresInSeconds int64
}

type zendeskTokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	TokenType             string `json:"token_type"`
	Scope                 string `json:"scope"`
	ExpiresIn             *int64 `json:"expires_in"`
	RefreshTokenExpiresIn *int64 `json:"refresh_token_expires_in"`
}

func NewZendeskAuthStrategy(config map[string]any) (*ZendeskAuthStrategy, error) {
	s := &ZendeskAuthStrategy{
		client: &http.Client{Timeout: authHTTPTimeout},
		config: zendeskOAuthConfig{
			ClientID:     getStringValue(config, "clientId"),
			ClientSecret: getStringValue(config, "clientSecret"),
			RefreshToken: getStringValue(config, "refreshToken"),
			TokenURL:     getStringValue(config, "tokenUrl"),
			Scope:        getStringValue(config, "scope"),
			AccessToken:  getStringValue(config, "accessToken"),
		},
	}
	if expiresAt, ok := getInt64Value(config, "expiresAt"); ok {
		s.config.ExpiresAt = expiresAt
	}
	if refreshTokenExpiresAt, ok := getInt64Value(config, "refreshTokenExpiresAt"); ok {
		s.config.RefreshTokenExpiresAt = refreshTokenExpiresAt
	}
	if expiresInSeconds, ok := getInt64Value(config, "expiresInSeconds"); ok {
		s.config.ExpiresInSeconds = expiresInSeconds
	}
	if refreshTokenExpiresInSeconds, ok := getInt64Value(config, "refreshTokenExpiresInSeconds"); ok {
		s.config.RefreshTokenExpiresInSeconds = refreshTokenExpiresInSeconds
	}
	if err := s.validateConfig(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ZendeskAuthStrategy) validateConfig() error {
	if strings.TrimSpace(s.config.ClientID) == "" {
		return fmt.Errorf("Zendesk OAuth: clientId is required")
	}
	if strings.TrimSpace(s.config.ClientSecret) == "" {
		return fmt.Errorf("Zendesk OAuth: clientSecret is required")
	}
	if strings.TrimSpace(s.config.RefreshToken) == "" {
		return fmt.Errorf("Zendesk OAuth: refreshToken is required")
	}
	if strings.TrimSpace(s.config.TokenURL) == "" {
		return fmt.Errorf("Zendesk OAuth: tokenUrl is required")
	}
	return nil
}

func (s *ZendeskAuthStrategy) GetInitialToken() (*TokenInfo, error) {
	return s.RefreshToken()
}

func (s *ZendeskAuthStrategy) RefreshToken() (*TokenInfo, error) {
	s.state.mu.RLock()
	if s.config.AccessToken != "" && s.config.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now < s.config.ExpiresAt-expiryBuffer.Milliseconds() {
			expiresIn := (s.config.ExpiresAt - now) / 1000
			log.Debug().Str("strategy", "zendesk").Int64("expiresInSeconds", expiresIn).Msg("using cached token")
			info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
			s.state.mu.RUnlock()
			return info, nil
		}
		log.Debug().Str("strategy", "zendesk").Msg("cached token expired, refreshing")
	}
	s.state.mu.RUnlock()

	s.state.refreshMu.Lock()
	defer s.state.refreshMu.Unlock()

	// Double-check after acquiring refreshMu
	s.state.mu.RLock()
	if s.config.AccessToken != "" && s.config.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now < s.config.ExpiresAt-expiryBuffer.Milliseconds() {
			expiresIn := (s.config.ExpiresAt - now) / 1000
			info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
			s.state.mu.RUnlock()
			return info, nil
		}
	}
	clientID := s.config.ClientID
	clientSecret := s.config.ClientSecret
	refreshToken := s.config.RefreshToken
	tokenURL := s.config.TokenURL
	scope := s.config.Scope
	expiresInSeconds := s.config.ExpiresInSeconds
	refreshTokenExpiresInSeconds := s.config.RefreshTokenExpiresInSeconds
	s.state.mu.RUnlock()

	if expiresInSeconds == 0 {
		expiresInSeconds = zendeskDefaultExpiresIn
	}
	if refreshTokenExpiresInSeconds == 0 {
		refreshTokenExpiresInSeconds = zendeskDefaultRefreshTokenExpiresIn
	}

	body := map[string]any{
		"grant_type":               "refresh_token",
		"refresh_token":            refreshToken,
		"client_id":                clientID,
		"client_secret":            clientSecret,
		"expires_in":               expiresInSeconds,
		"refresh_token_expires_in": refreshTokenExpiresInSeconds,
	}
	if scope != "" {
		body["scope"] = scope
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("Zendesk OAuth token refresh error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("Zendesk OAuth token refresh error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Zendesk OAuth token refresh error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("Zendesk OAuth token refresh error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Zendesk OAuth token refresh failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var data zendeskTokenResponse
	if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(&data); err != nil {
		return nil, fmt.Errorf("Zendesk OAuth token refresh error: %w", err)
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		return nil, fmt.Errorf("Zendesk OAuth token refresh failed: missing access_token")
	}
	if strings.TrimSpace(data.RefreshToken) == "" {
		return nil, fmt.Errorf("Zendesk OAuth token refresh failed: missing refresh_token")
	}

	expiresIn := expiresInSeconds
	if data.ExpiresIn != nil {
		expiresIn = *data.ExpiresIn
	}
	expiresAt := time.Now().UnixMilli() + expiresIn*1000

	s.state.mu.Lock()
	s.config.AccessToken = data.AccessToken
	s.config.RefreshToken = data.RefreshToken
	s.config.ExpiresAt = expiresAt
	if data.RefreshTokenExpiresIn != nil {
		s.config.RefreshTokenExpiresAt = time.Now().UnixMilli() + (*data.RefreshTokenExpiresIn * 1000)
	} else {
		s.config.RefreshTokenExpiresAt = time.Now().UnixMilli() + refreshTokenExpiresInSeconds*1000
	}
	s.state.configChanged = true
	s.state.mu.Unlock()

	log.Info().Str("strategy", "zendesk").Int64("expiresInSeconds", expiresIn).Msg("token refreshed")

	return &TokenInfo{AccessToken: data.AccessToken, ExpiresIn: expiresIn, ExpiresAt: expiresAt}, nil
}

func (s *ZendeskAuthStrategy) GetCurrentOAuthConfig() map[string]any {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	if !s.state.configChanged {
		return nil
	}
	return map[string]any{
		"clientId":                     s.config.ClientID,
		"clientSecret":                 s.config.ClientSecret,
		"refreshToken":                 s.config.RefreshToken,
		"tokenUrl":                     s.config.TokenURL,
		"scope":                        s.config.Scope,
		"accessToken":                  s.config.AccessToken,
		"expiresAt":                    s.config.ExpiresAt,
		"refreshTokenExpiresAt":        s.config.RefreshTokenExpiresAt,
		"expiresInSeconds":             s.config.ExpiresInSeconds,
		"refreshTokenExpiresInSeconds": s.config.RefreshTokenExpiresInSeconds,
	}
}

func (s *ZendeskAuthStrategy) MarkConfigAsPersisted() {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	s.state.configChanged = false
}

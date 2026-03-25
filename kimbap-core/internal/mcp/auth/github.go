package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	githubTokenEndpoint    = "https://github.com/login/oauth/access_token"
	githubDefaultExpiresIn = int64(8 * 60 * 60)
)

type GithubAuthStrategy struct {
	state  strategyState
	client *http.Client
	config githubOAuthConfig
}

type githubOAuthConfig struct {
	ClientID              string
	ClientSecret          string
	RefreshToken          string
	AccessToken           string
	ExpiresAt             int64
	RefreshTokenExpiresAt int64
}

type githubTokenResponse struct {
	AccessToken           string `json:"access_token"`
	TokenType             string `json:"token_type"`
	Scope                 string `json:"scope"`
	ExpiresIn             *int64 `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn *int64 `json:"refresh_token_expires_in"`
}

func NewGithubAuthStrategy(config map[string]any) (*GithubAuthStrategy, error) {
	s := &GithubAuthStrategy{
		client: &http.Client{Timeout: authHTTPTimeout},
		config: githubOAuthConfig{
			ClientID:              getStringValue(config, "clientId"),
			ClientSecret:          getStringValue(config, "clientSecret"),
			RefreshToken:          getStringValue(config, "refreshToken"),
			AccessToken:           getStringValue(config, "accessToken"),
			RefreshTokenExpiresAt: 0,
		},
	}
	if expiresAt, ok := getInt64Value(config, "expiresAt"); ok {
		s.config.ExpiresAt = expiresAt
	}
	if refreshTokenExpiresAt, ok := getInt64Value(config, "refreshTokenExpiresAt"); ok {
		s.config.RefreshTokenExpiresAt = refreshTokenExpiresAt
	}
	if err := s.validateConfig(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *GithubAuthStrategy) validateConfig() error {
	if strings.TrimSpace(s.config.ClientID) == "" {
		return fmt.Errorf("GitHub OAuth: clientId is required")
	}
	if strings.TrimSpace(s.config.ClientSecret) == "" {
		return fmt.Errorf("GitHub OAuth: clientSecret is required")
	}
	if strings.TrimSpace(s.config.RefreshToken) == "" {
		return fmt.Errorf("GitHub OAuth: refreshToken is required")
	}
	return nil
}

func (s *GithubAuthStrategy) GetInitialToken() (*TokenInfo, error) {
	return s.RefreshToken()
}

func (s *GithubAuthStrategy) RefreshToken() (*TokenInfo, error) {
	s.state.mu.RLock()
	if s.config.AccessToken != "" && s.config.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now < s.config.ExpiresAt-expiryBuffer.Milliseconds() {
			log.Debug().Str("strategy", "github").Msg("using cached token")
			expiresIn := (s.config.ExpiresAt - now) / 1000
			info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
			s.state.mu.RUnlock()
			return info, nil
		}
		log.Debug().Str("strategy", "github").Msg("cached token expired, refreshing")
	}
	s.state.mu.RUnlock()

	s.state.refreshMu.Lock()
	defer s.state.refreshMu.Unlock()

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
	s.state.mu.RUnlock()

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	req, err := http.NewRequest(http.MethodPost, githubTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("GitHub OAuth token refresh error: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub OAuth token refresh error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("GitHub OAuth token refresh error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub OAuth token refresh failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var data githubTokenResponse
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&data); err != nil {
		return nil, fmt.Errorf("GitHub OAuth token refresh error: %w", err)
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		return nil, fmt.Errorf("GitHub OAuth token refresh failed: missing access_token")
	}

	expiresIn := githubDefaultExpiresIn
	if data.ExpiresIn != nil {
		expiresIn = *data.ExpiresIn
	}
	expiresAt := time.Now().UnixMilli() + expiresIn*1000

	s.state.mu.Lock()
	s.config.AccessToken = data.AccessToken
	s.config.ExpiresAt = expiresAt
	rotated := data.RefreshToken != ""
	if data.RefreshToken != "" {
		s.config.RefreshToken = data.RefreshToken
	}
	if data.RefreshTokenExpiresIn != nil {
		s.config.RefreshTokenExpiresAt = time.Now().UnixMilli() + (*data.RefreshTokenExpiresIn * 1000)
	}
	s.state.configChanged = true
	s.state.mu.Unlock()

	log.Info().Str("strategy", "github").Bool("rotation", rotated).Msg("token refreshed")

	return &TokenInfo{AccessToken: data.AccessToken, ExpiresIn: expiresIn, ExpiresAt: expiresAt}, nil
}

func (s *GithubAuthStrategy) GetCurrentOAuthConfig() map[string]any {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	if !s.state.configChanged {
		return nil
	}
	return map[string]any{
		"clientId":              s.config.ClientID,
		"clientSecret":          s.config.ClientSecret,
		"refreshToken":          s.config.RefreshToken,
		"accessToken":           s.config.AccessToken,
		"expiresAt":             s.config.ExpiresAt,
		"refreshTokenExpiresAt": s.config.RefreshTokenExpiresAt,
	}
}

func (s *GithubAuthStrategy) MarkConfigAsPersisted() {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	s.state.configChanged = false
}

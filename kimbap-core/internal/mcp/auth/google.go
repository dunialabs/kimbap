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

type GoogleAuthStrategy struct {
	state  strategyState
	client *http.Client
	config googleOAuthConfig
}

type googleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	AccessToken  string
	ExpiresAt    int64
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

const googleTokenEndpoint = "https://oauth2.googleapis.com/token"

func NewGoogleAuthStrategy(config map[string]interface{}) (*GoogleAuthStrategy, error) {
	s := &GoogleAuthStrategy{
		client: &http.Client{Timeout: authHTTPTimeout},
		config: googleOAuthConfig{
			ClientID:     getStringValue(config, "clientId"),
			ClientSecret: getStringValue(config, "clientSecret"),
			RefreshToken: getStringValue(config, "refreshToken"),
			AccessToken:  getStringValue(config, "accessToken"),
		},
	}
	if expiresAt, ok := getInt64Value(config, "expiresAt"); ok {
		s.config.ExpiresAt = expiresAt
	}
	if err := s.validateConfig(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *GoogleAuthStrategy) validateConfig() error {
	if strings.TrimSpace(s.config.ClientID) == "" {
		return fmt.Errorf("Google OAuth: clientId is required")
	}
	if strings.TrimSpace(s.config.ClientSecret) == "" {
		return fmt.Errorf("Google OAuth: clientSecret is required")
	}
	if strings.TrimSpace(s.config.RefreshToken) == "" {
		return fmt.Errorf("Google OAuth: refreshToken is required")
	}
	return nil
}

func (s *GoogleAuthStrategy) GetInitialToken() (*TokenInfo, error) {
	return s.RefreshToken()
}

func (s *GoogleAuthStrategy) RefreshToken() (*TokenInfo, error) {
	s.state.mu.RLock()
	if s.config.AccessToken != "" && s.config.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now < s.config.ExpiresAt-expiryBuffer.Milliseconds() {
			hoursRemaining := float64(s.config.ExpiresAt-now) / float64(time.Hour.Milliseconds())
			log.Debug().Str("strategy", "google").Float64("hoursRemaining", hoursRemaining).Msg("using cached token")
			expiresIn := (s.config.ExpiresAt - now) / 1000
			info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
			s.state.mu.RUnlock()
			return info, nil
		}
		log.Debug().Str("strategy", "google").Msg("cached token expired, refreshing")
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
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequest(http.MethodPost, googleTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("Google OAuth token refresh error: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Google OAuth token refresh error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("Google OAuth token refresh error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Google OAuth token refresh failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var data googleTokenResponse
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&data); err != nil {
		return nil, fmt.Errorf("Google OAuth token refresh error: %w", err)
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		return nil, fmt.Errorf("Google OAuth token refresh error: missing access_token")
	}

	s.state.mu.Lock()
	s.config.AccessToken = data.AccessToken
	if data.ExpiresIn > 0 {
		s.config.ExpiresAt = time.Now().UnixMilli() + data.ExpiresIn*1000
	}
	s.state.configChanged = true
	s.state.mu.Unlock()

	expiresAt := time.Now().UnixMilli() + data.ExpiresIn*1000

	return &TokenInfo{AccessToken: data.AccessToken, ExpiresIn: data.ExpiresIn, ExpiresAt: expiresAt}, nil
}

func (s *GoogleAuthStrategy) GetCurrentOAuthConfig() map[string]interface{} {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	if !s.state.configChanged {
		return nil
	}
	return map[string]interface{}{
		"clientId":     s.config.ClientID,
		"clientSecret": s.config.ClientSecret,
		"refreshToken": s.config.RefreshToken,
		"accessToken":  s.config.AccessToken,
		"expiresAt":    s.config.ExpiresAt,
	}
}

func (s *GoogleAuthStrategy) MarkConfigAsPersisted() {
	s.state.mu.Lock()
	s.state.configChanged = false
	s.state.mu.Unlock()
}

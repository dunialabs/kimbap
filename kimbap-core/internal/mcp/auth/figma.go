package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const figmaRefreshTokenEndpoint = "https://api.figma.com/v1/oauth/refresh"

type FigmaAuthStrategy struct {
	state  strategyState
	client *http.Client
	config figmaOAuthConfig
}

type figmaOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	AccessToken  string
	ExpiresAt    int64
}

type figmaTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func NewFigmaAuthStrategy(config map[string]any) (*FigmaAuthStrategy, error) {
	s := &FigmaAuthStrategy{
		client: &http.Client{Timeout: authHTTPTimeout},
		config: figmaOAuthConfig{
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

func (s *FigmaAuthStrategy) validateConfig() error {
	if strings.TrimSpace(s.config.ClientID) == "" {
		return fmt.Errorf("Figma OAuth: clientId is required")
	}
	if strings.TrimSpace(s.config.ClientSecret) == "" {
		return fmt.Errorf("Figma OAuth: clientSecret is required")
	}
	if strings.TrimSpace(s.config.RefreshToken) == "" {
		return fmt.Errorf("Figma OAuth: refreshToken is required")
	}
	return nil
}

func (s *FigmaAuthStrategy) GetInitialToken() (*TokenInfo, error) {
	return s.RefreshToken()
}

func (s *FigmaAuthStrategy) RefreshToken() (*TokenInfo, error) {
	s.state.mu.RLock()
	if s.config.AccessToken != "" && s.config.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now < s.config.ExpiresAt-expiryBuffer.Milliseconds() {
			hoursRemaining := float64(s.config.ExpiresAt-now) / float64(time.Hour.Milliseconds())
			log.Debug().Str("strategy", "figma").Float64("hoursRemaining", hoursRemaining).Msg("using cached token")
			expiresIn := (s.config.ExpiresAt - now) / 1000
			info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
			s.state.mu.RUnlock()
			return info, nil
		}
		log.Debug().Str("strategy", "figma").Msg("cached token expired, refreshing")
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

	credentials := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	form := url.Values{}
	form.Set("refresh_token", refreshToken)

	req, err := http.NewRequest(http.MethodPost, figmaRefreshTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("Figma OAuth token refresh error: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Figma OAuth token refresh error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("Figma OAuth token refresh error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Figma OAuth token refresh failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var data figmaTokenResponse
	if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(&data); err != nil {
		return nil, fmt.Errorf("Figma OAuth token refresh error: %w", err)
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		return nil, fmt.Errorf("Figma OAuth token refresh error: missing access_token")
	}

	expiresAt := time.Now().UnixMilli() + data.ExpiresIn*1000

	s.state.mu.Lock()
	s.config.AccessToken = data.AccessToken
	s.config.ExpiresAt = expiresAt
	s.state.configChanged = true
	s.state.mu.Unlock()

	daysRemaining := float64(data.ExpiresIn) / (24 * 60 * 60)
	log.Info().Str("strategy", "figma").Float64("daysRemaining", daysRemaining).Msg("new token obtained")

	return &TokenInfo{AccessToken: data.AccessToken, ExpiresIn: data.ExpiresIn, ExpiresAt: expiresAt}, nil
}

func (s *FigmaAuthStrategy) GetCurrentOAuthConfig() map[string]any {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	if !s.state.configChanged {
		return nil
	}
	return map[string]any{
		"clientId":     s.config.ClientID,
		"clientSecret": s.config.ClientSecret,
		"refreshToken": s.config.RefreshToken,
		"accessToken":  s.config.AccessToken,
		"expiresAt":    s.config.ExpiresAt,
	}
}

func (s *FigmaAuthStrategy) MarkConfigAsPersisted() {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	s.state.configChanged = false
}

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

const (
	canvaTokenEndpoint    = "https://api.canva.com/rest/v1/oauth/token"
	canvaDefaultExpiresIn = int64(4 * 60 * 60) // 4 hours
)

type CanvaAuthStrategy struct {
	state  strategyState
	client *http.Client
	config canvaOAuthConfig
}

type canvaOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	AccessToken  string
	ExpiresAt    int64
}

type canvaTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    *int64 `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func NewCanvaAuthStrategy(config map[string]any) (*CanvaAuthStrategy, error) {
	s := &CanvaAuthStrategy{
		client: &http.Client{Timeout: authHTTPTimeout},
		config: canvaOAuthConfig{
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

func (s *CanvaAuthStrategy) validateConfig() error {
	if strings.TrimSpace(s.config.ClientID) == "" {
		return fmt.Errorf("Canva OAuth: clientId is required")
	}
	if strings.TrimSpace(s.config.ClientSecret) == "" {
		return fmt.Errorf("Canva OAuth: clientSecret is required")
	}
	if strings.TrimSpace(s.config.RefreshToken) == "" {
		return fmt.Errorf("Canva OAuth: refreshToken is required")
	}
	return nil
}

func (s *CanvaAuthStrategy) GetInitialToken() (*TokenInfo, error) {
	return s.RefreshToken()
}

func (s *CanvaAuthStrategy) RefreshToken() (*TokenInfo, error) {
	s.state.mu.RLock()
	if s.config.AccessToken != "" && s.config.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now < s.config.ExpiresAt-expiryBuffer.Milliseconds() {
			expiresIn := (s.config.ExpiresAt - now) / 1000
			log.Debug().Str("strategy", "canva").Int64("expiresInSeconds", expiresIn).Msg("using cached token")
			info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
			s.state.mu.RUnlock()
			return info, nil
		}
		log.Debug().Str("strategy", "canva").Msg("cached token expired, refreshing")
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
	s.state.mu.RUnlock()

	credentials := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	req, err := http.NewRequest(http.MethodPost, canvaTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("Canva OAuth token refresh error: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Canva OAuth token refresh error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("Canva OAuth token refresh error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Canva OAuth token refresh failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var data canvaTokenResponse
	if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(&data); err != nil {
		return nil, fmt.Errorf("Canva OAuth token refresh error: %w", err)
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		return nil, fmt.Errorf("Canva OAuth token refresh failed: missing access_token")
	}
	if strings.TrimSpace(data.RefreshToken) == "" {
		return nil, fmt.Errorf("Canva OAuth token refresh failed: missing refresh_token")
	}

	expiresIn := canvaDefaultExpiresIn
	if data.ExpiresIn != nil {
		expiresIn = *data.ExpiresIn
	}
	expiresAt := time.Now().UnixMilli() + expiresIn*1000

	s.state.mu.Lock()
	s.config.AccessToken = data.AccessToken
	s.config.RefreshToken = data.RefreshToken
	s.config.ExpiresAt = expiresAt
	s.state.configChanged = true
	s.state.mu.Unlock()

	log.Info().Str("strategy", "canva").Bool("rotation", true).Int64("expiresInSeconds", expiresIn).Msg("token refreshed")

	return &TokenInfo{AccessToken: data.AccessToken, ExpiresIn: expiresIn, ExpiresAt: expiresAt}, nil
}

func (s *CanvaAuthStrategy) GetCurrentOAuthConfig() map[string]any {
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

func (s *CanvaAuthStrategy) MarkConfigAsPersisted() {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	s.state.configChanged = false
}

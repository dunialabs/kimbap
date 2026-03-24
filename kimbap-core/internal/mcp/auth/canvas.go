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

const canvasDefaultExpiresIn = int64(2 * 60 * 60)

type CanvasAuthStrategy struct {
	state  strategyState
	client *http.Client
	config canvasOAuthConfig
}

type canvasOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	TokenURL     string
	AccessToken  string
	ExpiresAt    int64
}

type canvasTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    *int64 `json:"expires_in"`
}

func NewCanvasAuthStrategy(config map[string]any) ($$$) {
  $$$
}

func (s *CanvasAuthStrategy) validateConfig() error {
	if strings.TrimSpace(s.config.ClientID) == "" {
		return fmt.Errorf("Canvas OAuth: clientId is required")
	}
	if strings.TrimSpace(s.config.ClientSecret) == "" {
		return fmt.Errorf("Canvas OAuth: clientSecret is required")
	}
	if strings.TrimSpace(s.config.RefreshToken) == "" {
		return fmt.Errorf("Canvas OAuth: refreshToken is required")
	}
	if strings.TrimSpace(s.config.TokenURL) == "" {
		return fmt.Errorf("Canvas OAuth: tokenUrl is required")
	}
	return nil
}

func (s *CanvasAuthStrategy) GetInitialToken() (*TokenInfo, error) {
	return s.RefreshToken()
}

func (s *CanvasAuthStrategy) RefreshToken() (*TokenInfo, error) {
	s.state.mu.RLock()
	if s.config.AccessToken != "" && s.config.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now < s.config.ExpiresAt-expiryBuffer.Milliseconds() {
			expiresIn := (s.config.ExpiresAt - now) / 1000
			log.Debug().Str("strategy", "canvas").Int64("expiresInSeconds", expiresIn).Msg("using cached token")
			info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
			s.state.mu.RUnlock()
			return info, nil
		}
		log.Debug().Str("strategy", "canvas").Msg("cached token expired, refreshing")
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
	tokenURL := s.config.TokenURL
	s.state.mu.RUnlock()

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("Canvas OAuth token refresh error: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Canvas OAuth token refresh error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("Canvas OAuth token refresh error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Canvas OAuth token refresh failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var data canvasTokenResponse
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&data); err != nil {
		return nil, fmt.Errorf("Canvas OAuth token refresh error: %w", err)
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		return nil, fmt.Errorf("Canvas OAuth token refresh failed: missing access_token")
	}

	expiresIn := canvasDefaultExpiresIn
	if data.ExpiresIn != nil && *data.ExpiresIn > 0 {
		expiresIn = *data.ExpiresIn
	}
	expiresAt := time.Now().UnixMilli() + expiresIn*1000

	s.state.mu.Lock()
	s.config.AccessToken = data.AccessToken
	if strings.TrimSpace(data.RefreshToken) != "" {
		s.config.RefreshToken = data.RefreshToken
	}
	s.config.ExpiresAt = expiresAt
	s.state.configChanged = true
	s.state.mu.Unlock()

	log.Info().Str("strategy", "canvas").Int64("expiresInSeconds", expiresIn).Msg("token refreshed")

	return &TokenInfo{AccessToken: data.AccessToken, ExpiresIn: expiresIn, ExpiresAt: expiresAt}, nil
}

func (s *CanvasAuthStrategy) GetCurrentOAuthConfig() map[string]any {
  $$$
}

func (s *CanvasAuthStrategy) MarkConfigAsPersisted() {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	s.state.configChanged = false
}

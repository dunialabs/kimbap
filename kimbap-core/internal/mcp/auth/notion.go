package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	notionTokenEndpoint    = "https://api.notion.com/v1/oauth/token"
	notionDefaultExpiresIn = int64(30 * 24 * 60 * 60)
)

type NotionAuthStrategy struct {
	state  strategyState
	client *http.Client
	config notionOAuthConfig
}

type notionOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	AccessToken  string
	ExpiresAt    int64
}

type notionTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	BotID        string `json:"bot_id"`
	WorkspaceID  string `json:"workspace_id"`
}

func NewNotionAuthStrategy(config map[string]interface{}) (*NotionAuthStrategy, error) {
	s := &NotionAuthStrategy{
		client: &http.Client{Timeout: authHTTPTimeout},
		config: notionOAuthConfig{
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

func (s *NotionAuthStrategy) validateConfig() error {
	if strings.TrimSpace(s.config.ClientID) == "" {
		return fmt.Errorf("Notion OAuth: clientId is required")
	}
	if strings.TrimSpace(s.config.ClientSecret) == "" {
		return fmt.Errorf("Notion OAuth: clientSecret is required")
	}
	if strings.TrimSpace(s.config.RefreshToken) == "" {
		return fmt.Errorf("Notion OAuth: refreshToken is required")
	}
	return nil
}

func (s *NotionAuthStrategy) GetInitialToken() (*TokenInfo, error) {
	return s.RefreshToken()
}

func (s *NotionAuthStrategy) RefreshToken() (*TokenInfo, error) {
	s.state.mu.RLock()
	if s.config.AccessToken != "" && s.config.ExpiresAt > 0 {
		now := time.Now().UnixMilli()
		if now < s.config.ExpiresAt-expiryBuffer.Milliseconds() {
			hoursRemaining := float64(s.config.ExpiresAt-now) / float64(time.Hour.Milliseconds())
			log.Debug().Str("strategy", "notion").Float64("hoursRemaining", hoursRemaining).Msg("using cached token")
			expiresIn := (s.config.ExpiresAt - now) / 1000
			info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
			s.state.mu.RUnlock()
			return info, nil
		}
		log.Debug().Str("strategy", "notion").Msg("cached token expired, refreshing")
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
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, notionTokenEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("Notion OAuth token refresh error: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Notion OAuth token refresh error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("Notion OAuth token refresh error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Notion OAuth token refresh failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var data notionTokenResponse
	if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(&data); err != nil {
		return nil, fmt.Errorf("Notion OAuth token refresh error: %w", err)
	}
	if strings.TrimSpace(data.AccessToken) == "" {
		return nil, fmt.Errorf("Notion OAuth token refresh error: missing access_token")
	}
	if strings.TrimSpace(data.RefreshToken) == "" {
		return nil, fmt.Errorf("Notion OAuth token refresh error: missing refresh_token")
	}

	expiresIn := notionDefaultExpiresIn
	expiresAt := time.Now().UnixMilli() + expiresIn*1000

	s.state.mu.Lock()
	s.config.AccessToken = data.AccessToken
	s.config.RefreshToken = data.RefreshToken
	s.config.ExpiresAt = expiresAt
	s.state.configChanged = true
	s.state.mu.Unlock()

	daysRemaining := float64(expiresIn) / (24 * 60 * 60)
	log.Info().Str("strategy", "notion").Float64("daysRemaining", daysRemaining).Msg("new token obtained")

	return &TokenInfo{AccessToken: data.AccessToken, ExpiresIn: expiresIn, ExpiresAt: expiresAt}, nil
}

func (s *NotionAuthStrategy) GetCurrentOAuthConfig() map[string]interface{} {
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

func (s *NotionAuthStrategy) MarkConfigAsPersisted() {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	s.state.configChanged = false
}

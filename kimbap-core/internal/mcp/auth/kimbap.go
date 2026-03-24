package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	coretypes "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/rs/zerolog/log"
)

const kimbapAuthPathRefresh = "/v1/oauth/refresh"

type KimbapAuthStrategy struct {
	state  strategyState
	client *http.Client
	config kimbapOAuthConfig
}

type kimbapOAuthConfig struct {
	UserToken   string
	Server      *database.Server
	ClientID    string
	Key         string
	AccessToken string
	ExpiresAt   int64
}

type kimbapRefreshResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   int64  `json:"expiresAt"`
}

func NewKimbapAuthStrategy(config map[string]interface{}) (*KimbapAuthStrategy, error) {
	server, _ := config["server"].(*database.Server)
	if server == nil {
		if val, ok := config["server"].(database.Server); ok {
			server = &val
		}
	}
	s := &KimbapAuthStrategy{
		client: &http.Client{Timeout: authHTTPTimeout},
		config: kimbapOAuthConfig{
			UserToken:   getStringValue(config, "userToken"),
			Server:      server,
			ClientID:    getStringValue(config, "clientId"),
			Key:         getStringValue(config, "key"),
			AccessToken: getStringValue(config, "accessToken"),
		},
	}
	if expiresAt, ok := getInt64Value(config, "expiresAt"); ok {
		s.config.ExpiresAt = normalizeExpiresAt(expiresAt)
	}
	if err := s.validateConfig(); err != nil {
		return nil, err
	}
	return s, nil
}

func normalizeExpiresAt(expiresAt int64) int64 {
	if expiresAt < 10_000_000_000 {
		return expiresAt * 1000
	}
	return expiresAt
}

func (s *KimbapAuthStrategy) validateConfig() error {
	if strings.TrimSpace(s.config.UserToken) == "" {
		return fmt.Errorf("Kimbap OAuth: userToken is required")
	}
	if strings.TrimSpace(s.config.ClientID) == "" {
		return fmt.Errorf("Kimbap OAuth: clientId is required")
	}
	if strings.TrimSpace(s.config.Key) == "" {
		return fmt.Errorf("Kimbap OAuth: key is required")
	}
	return nil
}

func (s *KimbapAuthStrategy) isCachedTokenValid() bool {
	return s.config.AccessToken != "" && s.config.ExpiresAt > 0 && time.Now().UnixMilli() < s.config.ExpiresAt-expiryBuffer.Milliseconds()
}

func (s *KimbapAuthStrategy) GetInitialToken() (*TokenInfo, error) {
	return s.RefreshToken()
}

func (s *KimbapAuthStrategy) RefreshToken() (*TokenInfo, error) {
	s.state.mu.RLock()
	if s.isCachedTokenValid() {
		now := time.Now().UnixMilli()
		expiresIn := (s.config.ExpiresAt - now) / 1000
		log.Debug().Str("strategy", "kimbap").Int64("expiresIn", expiresIn).Msg("using cached token")
		info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
		s.state.mu.RUnlock()
		return info, nil
	}
	s.state.mu.RUnlock()

	s.state.refreshMu.Lock()
	defer s.state.refreshMu.Unlock()

	s.state.mu.RLock()
	if s.isCachedTokenValid() {
		now := time.Now().UnixMilli()
		expiresIn := (s.config.ExpiresAt - now) / 1000
		info := &TokenInfo{AccessToken: s.config.AccessToken, ExpiresIn: expiresIn, ExpiresAt: s.config.ExpiresAt}
		s.state.mu.RUnlock()
		return info, nil
	}
	s.state.mu.RUnlock()

	tokenInfo, err := s.refreshTokenFromKimbap()
	if err != nil {
		return nil, err
	}

	s.state.mu.Lock()
	s.config.AccessToken = tokenInfo.AccessToken
	s.config.ExpiresAt = tokenInfo.ExpiresAt
	s.state.mu.Unlock()

	entry := log.Info().Str("strategy", "kimbap").Int64("expiresIn", tokenInfo.ExpiresIn)
	if s.config.Server != nil {
		if provider, ok := providerFromAuthType(s.config.Server.AuthType); ok {
			entry = entry.Str("provider", provider)
		}
	}
	entry.Msg("token refreshed")

	return tokenInfo, nil
}

func (s *KimbapAuthStrategy) refreshTokenFromKimbap() (*TokenInfo, error) {
	if s.config.Server == nil {
		return nil, fmt.Errorf("Kimbap OAuth: server config is required for token refresh")
	}
	provider, ok := providerFromAuthType(s.config.Server.AuthType)
	if !ok {
		return nil, fmt.Errorf("invalid OAuth provider")
	}

	requestBody := map[string]interface{}{
		"clientId": s.config.ClientID,
		"provider": provider,
		"key":      s.config.Key,
	}

	if provider == "zendesk" || provider == "canvas" {
		tokenURL := ""
		if s.config.Server != nil && s.config.Server.ConfigTemplate != nil {
			var tpl map[string]interface{}
			if err := json.Unmarshal([]byte(*s.config.Server.ConfigTemplate), &tpl); err == nil {
				oAuthConfigRaw, ok := tpl["oAuthConfig"].(map[string]interface{})
				if ok {
					tokenURL, _ = oAuthConfigRaw["tokenUrl"].(string)
					tokenURL = strings.TrimSpace(tokenURL)
				}
			}
		}
		if tokenURL != "" {
			requestBody["tokenUrl"] = tokenURL
		} else {
			serverID := ""
			if s.config.Server != nil {
				serverID = s.config.Server.ServerID
			}
			log.Warn().Str("strategy", "kimbap").Str("provider", provider).Str("serverId", serverID).Msg("missing tokenUrl for dynamic OAuth provider")
		}
	}

	body, _ := json.Marshal(requestBody)

	url := config.GetKimbapAuthConfig().BaseURL + kimbapAuthPathRefresh
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh OAuth token")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.config.UserToken))

	resp, err := s.client.Do(req)
	if err != nil {
		log.Error().Str("strategy", "kimbap").Err(err).Msg("token refresh request failed")
		return nil, fmt.Errorf("failed to refresh OAuth token")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh OAuth token")
	}

	var result kimbapRefreshResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to refresh OAuth token")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || result.AccessToken == "" || result.ExpiresAt == 0 {
		log.Warn().Str("strategy", "kimbap").Int("statusCode", resp.StatusCode).Msg("token refresh failed")
		return nil, fmt.Errorf("failed to refresh OAuth token")
	}

	expiresAt := normalizeExpiresAt(result.ExpiresAt)
	expiresIn := (expiresAt - time.Now().UnixMilli()) / 1000
	if expiresIn < 0 {
		expiresIn = 0
	}

	return &TokenInfo{AccessToken: result.AccessToken, ExpiresIn: expiresIn, ExpiresAt: expiresAt}, nil
}
func (s *KimbapAuthStrategy) GetCurrentOAuthConfig() map[string]interface{} {
	return nil
}

func (s *KimbapAuthStrategy) MarkConfigAsPersisted() {}

func providerFromAuthType(authType int) (string, bool) {
	switch authType {
	case coretypes.ServerAuthTypeGoogleAuth, coretypes.ServerAuthTypeGoogleCalendarAuth:
		return "google", true
	case coretypes.ServerAuthTypeNotionAuth:
		return "notion", true
	case coretypes.ServerAuthTypeFigmaAuth:
		return "figma", true
	case coretypes.ServerAuthTypeGithubAuth:
		return "github", true
	case coretypes.ServerAuthTypeZendeskAuth:
		return "zendesk", true
	case coretypes.ServerAuthTypeCanvasAuth:
		return "canvas", true
	case coretypes.ServerAuthTypeCanvaAuth:
		return "canva", true
	default:
		return "", false
	}
}

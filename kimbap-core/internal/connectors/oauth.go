package connectors

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

var oauthHTTPClient = &http.Client{Timeout: 15 * time.Second}

const maxOAuthResponseBodyBytes int64 = 4 << 20

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	VerificationURI string `json:"verification_uri"`
	UserCode        string `json:"user_code"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func DeviceCodeRequest(cfg ConnectorConfig) (*DeviceFlowResult, error) {
	if cfg.DeviceURL == "" {
		return nil, errors.New("device url is required")
	}

	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	if len(cfg.Scopes) > 0 {
		form.Set("scope", strings.Join(cfg.Scopes, " "))
	}

	body, err := postForm(cfg.DeviceURL, form)
	if err != nil {
		return nil, err
	}

	var decoded deviceCodeResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode device code response: %w", err)
	}
	if decoded.DeviceCode == "" || decoded.UserCode == "" || decoded.VerificationURI == "" {
		return nil, errors.New("device code response missing required fields")
	}

	if decoded.Interval <= 0 {
		decoded.Interval = 5
	}

	return &DeviceFlowResult{
		VerificationURL: decoded.VerificationURI,
		UserCode:        decoded.UserCode,
		ExpiresIn:       decoded.ExpiresIn,
		Interval:        decoded.Interval,
		DeviceCode:      decoded.DeviceCode,
	}, nil
}

func PollForToken(cfg ConnectorConfig, deviceCode string, interval int, timeout time.Duration) (*TokenResponse, error) {
	if cfg.TokenURL == "" {
		return nil, errors.New("token url is required")
	}
	if strings.TrimSpace(deviceCode) == "" {
		return nil, errors.New("device code is required")
	}
	if interval <= 0 {
		interval = 5
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return nil, errors.New("device flow polling timed out")
		}

		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		form.Set("client_id", cfg.ClientID)
		form.Set("device_code", deviceCode)
		if cfg.ClientSecret != "" {
			form.Set("client_secret", cfg.ClientSecret)
		}

		body, err := postForm(cfg.TokenURL, form)
		if err == nil {
			return parseTokenResponse(body)
		}

		oauthErr, ok := err.(*oauthHTTPError)
		if !ok {
			return nil, err
		}

		var tokenErr tokenErrorResponse
		if json.Unmarshal(oauthErr.body, &tokenErr) != nil {
			return nil, err
		}

		switch tokenErr.Error {
		case "authorization_pending":
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		case "slow_down":
			interval += 2
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		default:
			if tokenErr.ErrorDescription != "" {
				return nil, fmt.Errorf("oauth token error: %s (%s)", tokenErr.Error, tokenErr.ErrorDescription)
			}
			if tokenErr.Error != "" {
				return nil, fmt.Errorf("oauth token error: %s", tokenErr.Error)
			}
			return nil, err
		}
	}
}

func RefreshAccessToken(cfg ConnectorConfig, refreshToken string) (*TokenResponse, error) {
	if cfg.TokenURL == "" {
		return nil, errors.New("token url is required")
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("refresh token is required")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", cfg.ClientID)
	form.Set("refresh_token", refreshToken)
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}

	body, err := postForm(cfg.TokenURL, form)
	if err != nil {
		return nil, err
	}

	return parseTokenResponse(body)
}

type oauthHTTPError struct {
	status int
	body   []byte
}

func (e *oauthHTTPError) Error() string {
	return fmt.Sprintf("oauth endpoint returned status %d", e.status)
}

func postForm(endpoint string, form url.Values) ([]byte, error) {
	ctx := context.Background()
	if oauthHTTPClient.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, oauthHTTPClient.Timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute oauth request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxOAuthResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read oauth response: %w", err)
	}
	if int64(len(body)) > maxOAuthResponseBodyBytes {
		return nil, fmt.Errorf("oauth response exceeded %d bytes", maxOAuthResponseBodyBytes)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, &oauthHTTPError{status: res.StatusCode, body: body}
	}

	return body, nil
}

func parseTokenResponse(body []byte) (*TokenResponse, error) {
	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("token response missing access_token")
	}
	return &token, nil
}

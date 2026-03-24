package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var oauthHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

const maxOAuthResponseBodyBytes = 1 << 20

func OAuthHTTPPost(url string, request ProviderRequest, provider string) (*HTTPResponse, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(request.Body))
	if err != nil {
		return nil, NewOAuthExchangeError(
			fmt.Sprintf("OAuth token exchange failed for %s: %s", provider, err.Error()),
			OAuthExchangeErrorHTTP,
			provider,
			0,
			"",
			err,
		)
	}

	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, NewOAuthExchangeError(
			fmt.Sprintf("OAuth token exchange failed for %s: %s", provider, err.Error()),
			OAuthExchangeErrorHTTP,
			provider,
			0,
			"",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.ContentLength > maxOAuthResponseBodyBytes {
		return nil, NewOAuthExchangeError(
			fmt.Sprintf("OAuth token exchange failed for %s: response body too large", provider),
			OAuthExchangeErrorHTTP,
			provider,
			resp.StatusCode,
			"",
			nil,
		)
	}

	rawBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponseBodyBytes+1))
	if err != nil {
		return nil, NewOAuthExchangeError(
			fmt.Sprintf("OAuth token exchange failed for %s: %s", provider, err.Error()),
			OAuthExchangeErrorHTTP,
			provider,
			resp.StatusCode,
			"",
			err,
		)
	}
	if len(rawBytes) > maxOAuthResponseBodyBytes {
		return nil, NewOAuthExchangeError(
			fmt.Sprintf("OAuth token exchange failed for %s: response body too large", provider),
			OAuthExchangeErrorHTTP,
			provider,
			resp.StatusCode,
			"",
			nil,
		)
	}

	raw := string(rawBytes)
	data := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal(rawBytes, &data); err != nil {
			return nil, NewOAuthParseError(provider, raw, err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, NewOAuthHTTPError(provider, resp.StatusCode, raw)
	}
	if s, exists := data["error"].(string); exists {
		s = strings.TrimSpace(s)
		if s != "" {
			return nil, NewOAuthParseError(provider, raw, errors.New(s))
		}
	}

	return &HTTPResponse{
		Status: resp.StatusCode,
		Data:   data,
		Raw:    raw,
		Header: resp.Header,
	}, nil
}

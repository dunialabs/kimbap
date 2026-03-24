package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/mcp/oauth/providers"
)

func resolveTokenURL(ctx ExchangeContext, adapter providers.ProviderAdapter) (string, error) {
	if adapter.GetTokenURL != nil {
		resolved, err := adapter.GetTokenURL(providers.ExchangeContext{
			Provider:     ctx.Provider,
			TokenURL:     ctx.TokenURL,
			ClientID:     ctx.ClientID,
			ClientSecret: ctx.ClientSecret,
			Code:         ctx.Code,
			RedirectURI:  ctx.RedirectURI,
			CodeVerifier: ctx.CodeVerifier,
			Scope:        ctx.Scope,
		})
		if err != nil {
			return "", err
		}
		return validateTokenURL(resolved)
	}
	if adapter.TokenURL != "" {
		return validateTokenURL(adapter.TokenURL)
	}
	if ctx.TokenURL != "" {
		return validateTokenURL(ctx.TokenURL)
	}
	return "", NewOAuthExchangeError(
		fmt.Sprintf("No token URL available for provider '%s'. This provider requires ctx.tokenUrl to be specified.", ctx.Provider),
		OAuthExchangeErrorHTTP,
		ctx.Provider,
		0,
		"",
		nil,
	)
}

func ExchangeAuthorizationCode(ctx ExchangeContext) (*ExchangeResult, error) {
	adapter, err := providers.GetProviderAdapter(ctx.Provider)
	if err != nil {
		var unknownProviderErr *providers.UnknownProviderError
		if errors.As(err, &unknownProviderErr) {
			return nil, NewOAuthUnknownProviderError(unknownProviderErr.Provider)
		}
		return nil, err
	}

	tokenURL, err := resolveTokenURL(ctx, adapter)
	if err != nil {
		return nil, err
	}

	request, err := adapter.BuildRequest(providers.ExchangeContext{
		Provider:     ctx.Provider,
		TokenURL:     tokenURL,
		ClientID:     ctx.ClientID,
		ClientSecret: ctx.ClientSecret,
		Code:         ctx.Code,
		RedirectURI:  ctx.RedirectURI,
		CodeVerifier: ctx.CodeVerifier,
		Scope:        ctx.Scope,
	})
	if err != nil {
		return nil, err
	}

	response, err := OAuthHTTPPost(tokenURL, ProviderRequest{Headers: request.Headers, Body: request.Body}, ctx.Provider)
	if err != nil {
		return nil, err
	}

	data := response.Data
	accessToken, ok := data["access_token"].(string)
	if !ok || strings.TrimSpace(accessToken) == "" {
		body, _ := json.Marshal(data)
		return nil, NewOAuthExchangeError(
			"No access token found in response",
			OAuthExchangeErrorParse,
			ctx.Provider,
			0,
			string(body),
			nil,
		)
	}

	refreshToken, _ := data["refresh_token"].(string)

	var responseExpiresIn *int64
	if val, exists := data["expires_in"]; exists {
		if parsed, ok := NumberToInt64(val); ok {
			responseExpiresIn = &parsed
		}
	}

	expiresIn, expiresAt := ResolveExpires(responseExpiresIn, adapter.DefaultExpiresIn)

	return &ExchangeResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		ExpiresAt:    expiresAt,
		Raw:          data,
	}, nil
}

func validateTokenURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", NewOAuthExchangeError("token URL is required", OAuthExchangeErrorHTTP, "", 0, "", nil)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", NewOAuthExchangeError("invalid token URL", OAuthExchangeErrorHTTP, "", 0, "", nil)
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme == "https" {
		return parsed.String(), nil
	}
	if scheme == "http" {
		host := strings.TrimSpace(parsed.Hostname())
		if strings.EqualFold(host, "localhost") {
			return parsed.String(), nil
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return parsed.String(), nil
		}
	}
	return "", NewOAuthExchangeError("token URL must use https (or localhost http)", OAuthExchangeErrorHTTP, "", 0, "", nil)
}

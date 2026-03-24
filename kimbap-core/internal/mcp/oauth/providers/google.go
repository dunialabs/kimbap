package providers

import "net/url"

var GoogleAdapter = ProviderAdapter{
	Name:     "google",
	TokenURL: "https://oauth2.googleapis.com/token",
	BuildRequest: func(ctx ExchangeContext) (ProviderRequest, error) {
		params := url.Values{}
		params.Set("grant_type", "authorization_code")
		params.Set("client_id", ctx.ClientID)
		params.Set("client_secret", ctx.ClientSecret)
		params.Set("code", ctx.Code)
		params.Set("redirect_uri", ctx.RedirectURI)
		if ctx.CodeVerifier != "" {
			params.Set("code_verifier", ctx.CodeVerifier)
		}
		return ProviderRequest{
			Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			Body:    params.Encode(),
		}, nil
	},
}

package providers

import "net/url"

var githubDefaultExpiresIn int64 = 8 * 60 * 60

var GithubAdapter = ProviderAdapter{
	Name:             "github",
	TokenURL:         "https://github.com/login/oauth/access_token",
	DefaultExpiresIn: &githubDefaultExpiresIn,
	BuildRequest: func(ctx ExchangeContext) (ProviderRequest, error) {
		params := url.Values{}
		params.Set("client_id", ctx.ClientID)
		params.Set("client_secret", ctx.ClientSecret)
		params.Set("code", ctx.Code)
		params.Set("redirect_uri", ctx.RedirectURI)
		if ctx.CodeVerifier != "" {
			params.Set("code_verifier", ctx.CodeVerifier)
		}
		return ProviderRequest{
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Accept":       "application/json",
			},
			Body: params.Encode(),
		}, nil
	},
}

package providers

import (
	"encoding/base64"
	"net/url"
)

var CanvaAdapter = ProviderAdapter{
	Name:     "canva",
	TokenURL: "https://api.canva.com/rest/v1/oauth/token",
	BuildRequest: func(ctx ExchangeContext) (ProviderRequest, error) {
		credentials := base64.StdEncoding.EncodeToString([]byte(ctx.ClientID + ":" + ctx.ClientSecret))

		params := url.Values{}
		params.Set("grant_type", "authorization_code")
		params.Set("code", ctx.Code)
		params.Set("redirect_uri", ctx.RedirectURI)

		if ctx.CodeVerifier != "" {
			params.Set("code_verifier", ctx.CodeVerifier)
		}


		return ProviderRequest{
			Headers: map[string]string{
				"Content-Type":  "application/x-www-form-urlencoded",
				"Authorization": "Basic " + credentials,
			},
			Body: params.Encode(),
		}, nil
	},
}

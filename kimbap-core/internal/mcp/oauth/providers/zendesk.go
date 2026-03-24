package providers

import "encoding/json"

var ZendeskAdapter = ProviderAdapter{
	Name: "zendesk",
	BuildRequest: func(ctx ExchangeContext) (ProviderRequest, error) {
		body := map[string]any{
			"grant_type":               "authorization_code",
			"client_id":                ctx.ClientID,
			"client_secret":            ctx.ClientSecret,
			"code":                     ctx.Code,
			"redirect_uri":             ctx.RedirectURI,
			"expires_in":               172800,  // 2 days
			"refresh_token_expires_in": 7776000, // 90 days
		}

		if ctx.Scope != "" {
			body["scope"] = ctx.Scope
		}

		if ctx.CodeVerifier != "" {
			body["code_verifier"] = ctx.CodeVerifier
		}

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return ProviderRequest{}, err
		}
		return ProviderRequest{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    string(bodyBytes),
		}, nil
	},
}

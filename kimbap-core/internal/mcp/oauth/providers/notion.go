package providers

import (
	"encoding/base64"
	"encoding/json"
)

var notionDefaultExpiresIn int64 = 30 * 24 * 60 * 60

var NotionAdapter = ProviderAdapter{
	Name:             "notion",
	TokenURL:         "https://api.notion.com/v1/oauth/token",
	DefaultExpiresIn: &notionDefaultExpiresIn,
	BuildRequest: func(ctx ExchangeContext) (ProviderRequest, error) {
		credentials := base64.StdEncoding.EncodeToString([]byte(ctx.ClientID + ":" + ctx.ClientSecret))
		body, err := json.Marshal(map[string]string{
			"grant_type":   "authorization_code",
			"code":         ctx.Code,
			"redirect_uri": ctx.RedirectURI,
		})
		if err != nil {
			return ProviderRequest{}, err
		}
		return ProviderRequest{
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Basic " + credentials,
			},
			Body: string(body),
		}, nil
	},
}

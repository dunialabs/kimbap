package oauth

type ExchangeContext struct {
	Provider     string `json:"provider"`
	TokenURL     string `json:"tokenUrl,omitempty"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirectUri"`
	CodeVerifier string `json:"codeVerifier,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type TokenResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token,omitempty"`
	ExpiresIn    *int64         `json:"expires_in,omitempty"`
	Raw          map[string]any `json:"raw,omitempty"`
}

type ExchangeResult struct {
	AccessToken  string         `json:"accessToken"`
	RefreshToken string         `json:"refreshToken,omitempty"`
	ExpiresIn    *int64         `json:"expiresIn,omitempty"`
	ExpiresAt    *int64         `json:"expiresAt,omitempty"`
	Raw          map[string]any `json:"raw"`
}

type ProviderRequest struct {
	Headers map[string]string
	Body    string
}

type HTTPResponse struct {
	Status int
	Data   map[string]any
	Raw    string
	Header map[string][]string
}

type ProviderAdapter struct {
	Name             string
	TokenURL         string
	GetTokenURL      func(ctx ExchangeContext) (string, error)
	BuildRequest     func(ctx ExchangeContext) (ProviderRequest, error)
	DefaultExpiresIn *int64
}

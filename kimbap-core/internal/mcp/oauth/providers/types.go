package providers

type ExchangeContext struct {
	Provider     string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Code         string
	RedirectURI  string
	CodeVerifier string
	Scope        string
}

type ProviderRequest struct {
	Headers map[string]string
	Body    string
}

type ProviderAdapter struct {
	Name             string
	TokenURL         string
	GetTokenURL      func(ctx ExchangeContext) (string, error)
	BuildRequest     func(ctx ExchangeContext) (ProviderRequest, error)
	DefaultExpiresIn *int64
}

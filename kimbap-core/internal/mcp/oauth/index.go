package oauth

import "github.com/dunialabs/kimbap-core/internal/mcp/oauth/providers"

func GetSupportedProviders() []string {
	return providers.GetSupportedProviders()
}

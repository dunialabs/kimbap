package providers

import (
	"fmt"
	"strings"
)

type UnknownProviderError struct {
	Provider string
}

func (e *UnknownProviderError) Error() string {
	return fmt.Sprintf("Unknown OAuth provider: %s", e.Provider)
}

var providerRegistry = map[string]ProviderAdapter{
	"google":  GoogleAdapter,
	"notion":  NotionAdapter,
	"figma":   FigmaAdapter,
	"github":  GithubAdapter,
	"stripe":  StripeAdapter,
	"zendesk": ZendeskAdapter,
	"canvas":  CanvasAdapter,
	"canva":   CanvaAdapter,
}

var providerOrder = []string{"google", "notion", "figma", "github", "stripe", "zendesk", "canvas", "canva"}

func GetProviderAdapter(provider string) (ProviderAdapter, error) {
	normalized := strings.ToLower(provider)
	adapter, ok := providerRegistry[normalized]
	if !ok {
		return ProviderAdapter{}, &UnknownProviderError{Provider: provider}
	}
	return adapter, nil
}

func GetSupportedProviders() []string {
	providers := make([]string, 0, len(providerOrder))
	for _, provider := range providerOrder {
		if _, ok := providerRegistry[provider]; ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

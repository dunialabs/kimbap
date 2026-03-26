package providers

import (
	"github.com/dunialabs/kimbap-core/internal/connectors"
)

// GetProvider returns a provider definition by ID.
// All providers are loaded from YAML manifests in the embedded official/ directory.
func GetProvider(id string) (connectors.ProviderDefinition, error) {
	return LoadProvider(id, EmbeddedProviders)
}

// ListProviders returns all known provider definitions.
// All providers are loaded from YAML manifests in the embedded official/ directory.
func ListProviders() []connectors.ProviderDefinition {
	all, _ := LoadAllProviders(EmbeddedProviders)
	return all
}

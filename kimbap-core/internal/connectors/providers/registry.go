package providers

import (
	"log/slog"

	"github.com/dunialabs/kimbap-core/internal/connectors"
)

func GetProvider(id string) (connectors.ProviderDefinition, error) {
	return LoadProvider(id, EmbeddedProviders)
}

func ListProviders() []connectors.ProviderDefinition {
	all, err := LoadAllProviders(EmbeddedProviders)
	if err != nil {
		slog.Error("failed to load provider manifests", "error", err)
	}
	return all
}

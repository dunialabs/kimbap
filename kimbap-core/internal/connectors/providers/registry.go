package providers

import (
	"github.com/dunialabs/kimbap-core/internal/connectors"
	"github.com/rs/zerolog/log"
)

func GetProvider(id string) (connectors.ProviderDefinition, error) {
	return LoadProvider(id, EmbeddedProviders)
}

func ListProviders() []connectors.ProviderDefinition {
	all, err := LoadAllProviders(EmbeddedProviders)
	if err != nil {
		log.Error().Err(err).Msg("failed to load provider manifests")
	}
	return all
}

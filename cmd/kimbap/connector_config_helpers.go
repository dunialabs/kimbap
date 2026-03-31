package main

import (
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	connectorproviders "github.com/dunialabs/kimbap/internal/connectors/providers"
)

func buildConnectorConfigs(cfg *config.KimbapConfig) []connectors.ConnectorConfig {
	configs := make([]connectors.ConnectorConfig, 0, len(connectorproviders.ListProviders()))
	for _, prov := range connectorproviders.ListProviders() {
		creds := resolveOAuthCreds(cfg, prov.ID)
		configs = append(configs, connectors.ConnectorConfig{
			Name:         prov.ID,
			Provider:     prov.ID,
			ClientID:     creds.ClientID,
			ClientSecret: creds.ClientSecret,
			AuthMethod:   creds.AuthMethod,
			TokenURL:     prov.TokenEndpoint,
			DeviceURL:    prov.DeviceEndpoint,
			Scopes:       prov.DefaultScopes,
		})
	}
	return configs
}

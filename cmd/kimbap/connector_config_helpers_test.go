package main

import (
	"testing"

	"github.com/dunialabs/kimbap/internal/config"
	connectorproviders "github.com/dunialabs/kimbap/internal/connectors/providers"
)

func TestBuildConnectorConfigsMatchesProviderCatalog(t *testing.T) {
	cfg := config.DefaultConfig()
	configs := buildConnectorConfigs(cfg)
	providersList := connectorproviders.ListProviders()

	if len(configs) != len(providersList) {
		t.Fatalf("expected %d connector configs, got %d", len(providersList), len(configs))
	}
	if len(configs) == 0 {
		t.Fatal("expected at least one connector config")
	}
	for i, prov := range providersList {
		if configs[i].Name != prov.ID {
			t.Fatalf("config[%d].Name = %q, want %q", i, configs[i].Name, prov.ID)
		}
		if configs[i].Provider != prov.ID {
			t.Fatalf("config[%d].Provider = %q, want %q", i, configs[i].Provider, prov.ID)
		}
		if configs[i].TokenURL != prov.TokenEndpoint {
			t.Fatalf("config[%d].TokenURL = %q, want %q", i, configs[i].TokenURL, prov.TokenEndpoint)
		}
		if configs[i].DeviceURL != prov.DeviceEndpoint {
			t.Fatalf("config[%d].DeviceURL = %q, want %q", i, configs[i].DeviceURL, prov.DeviceEndpoint)
		}
	}
}

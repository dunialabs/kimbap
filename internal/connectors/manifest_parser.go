package connectors

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseProviderManifest parses YAML bytes into a validated ProviderManifest.
// Returns error if YAML is malformed or any required fields are missing/invalid.
func ParseProviderManifest(data []byte) (*ProviderManifest, error) {
	var m ProviderManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse provider manifest: invalid YAML: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("parse provider manifest: %w", err)
	}
	return &m, nil
}

// LoadProviderFromFile reads a provider manifest YAML file from disk and parses it.
func LoadProviderFromFile(path string) (*ProviderManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load provider manifest from %q: %w", path, err)
	}
	return ParseProviderManifest(data)
}

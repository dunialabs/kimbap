package providers

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/connectors"
)

func manifestToDefinition(m *connectors.ProviderManifest) connectors.ProviderDefinition {
	flows := make([]connectors.FlowType, 0, len(m.SupportedFlows))
	for _, flow := range m.SupportedFlows {
		flows = append(flows, connectors.FlowType(flow))
	}

	scopes := make([]connectors.ConnectionScope, 0, len(m.ConnectionScopeModel))
	for _, scope := range m.ConnectionScopeModel {
		scopes = append(scopes, connectors.ConnectionScope(scope))
	}

	return connectors.ProviderDefinition{
		ID:                   m.ID,
		DisplayName:          m.DisplayName,
		SupportedFlows:       flows,
		AuthEndpoint:         m.AuthEndpoint,
		TokenEndpoint:        m.TokenEndpoint,
		DeviceEndpoint:       m.DeviceEndpoint,
		RevocationEndpoint:   m.RevocationEndpoint,
		UserInfoEndpoint:     m.UserInfoEndpoint,
		DefaultScopes:        m.DefaultScopes,
		ScopePresets:         m.ScopePresets,
		ConnectionScopeModel: scopes,
		PKCERequired:         m.PKCERequired,
		Notes:                m.Notes,
		AuthLanes:            m.AuthLanes,
		EmbeddedClientID:     m.EmbeddedClientID,
		ManagedClientID:      m.ManagedClientID,
		TokenExchange:        m.TokenExchange,
		EndpointPlaceholders: m.EndpointPlaceholders,
	}
}

// LoadProvider loads a provider definition by ID from the given fs.FS.
// Returns an error if the provider is not found or the YAML is invalid.
func LoadProvider(id string, fsys fs.FS) (connectors.ProviderDefinition, error) {
	normalized := strings.ToLower(strings.TrimSpace(id))
	if normalized == "_placeholder" || normalized == "" {
		return connectors.ProviderDefinition{}, fmt.Errorf("provider %q: %w", id, fs.ErrNotExist)
	}

	providerPath := filepath.ToSlash(filepath.Join("official", normalized+".yaml"))
	data, err := fs.ReadFile(fsys, providerPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return connectors.ProviderDefinition{}, fmt.Errorf("unknown provider: %s", id)
		}
		return connectors.ProviderDefinition{}, fmt.Errorf("read provider YAML %q: %w", providerPath, err)
	}

	manifest, err := connectors.ParseProviderManifest(data)
	if err != nil {
		return connectors.ProviderDefinition{}, fmt.Errorf("parse provider YAML %q: %w", providerPath, err)
	}

	def := manifestToDefinition(manifest)
	slog.Info("provider loaded from YAML", "provider", normalized)
	return def, nil
}

// LoadAllProviders loads all provider definitions from the given fs.FS.
// Only YAML files in the official/ directory are loaded; _placeholder.yaml is skipped.
func LoadAllProviders(fsys fs.FS) ([]connectors.ProviderDefinition, error) {
	merged := map[string]connectors.ProviderDefinition{}

	entries, err := fs.ReadDir(fsys, "official")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read providers directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".yaml" || name == "_placeholder.yaml" || name == "TEMPLATE.yaml" {
			continue
		}

		path := filepath.ToSlash(filepath.Join("official", name))
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return nil, fmt.Errorf("read provider YAML %q: %w", path, readErr)
		}

		manifest, parseErr := connectors.ParseProviderManifest(data)
		if parseErr != nil {
			return nil, fmt.Errorf("parse provider YAML %q: %w", path, parseErr)
		}

		def := manifestToDefinition(manifest)
		merged[def.ID] = def
	}

	ids := make([]string, 0, len(merged))
	for id := range merged {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]connectors.ProviderDefinition, 0, len(ids))
	for _, id := range ids {
		out = append(out, merged[id])
	}

	return out, nil
}

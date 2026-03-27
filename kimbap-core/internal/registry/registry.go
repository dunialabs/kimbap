package registry

import (
	"context"

	"github.com/dunialabs/kimbap-core/internal/services"
)

// Registry is the interface for resolving service manifests by name.
type Registry interface {
	// Name returns a human-readable name for this registry.
	Name() string
	// Resolve fetches the manifest for the named service and returns it along
	// with the canonical source string (e.g. "official:github", "remote:https://...").
	Resolve(ctx context.Context, name string) (*services.ServiceManifest, string, error)
	// List returns all service names available from this registry.
	List(ctx context.Context) ([]string, error)
}

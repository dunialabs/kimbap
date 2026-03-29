package registry

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/services"
)

type EmbeddedRegistry struct{}

func NewEmbeddedRegistry() *EmbeddedRegistry {
	return &EmbeddedRegistry{}
}

func (r *EmbeddedRegistry) Name() string { return "catalog" }

func (r *EmbeddedRegistry) Resolve(_ context.Context, name string) (*services.ServiceManifest, string, error) {
	data, err := catalog.Get(name)
	if err != nil {
		if isNotExist(err) {
			return nil, "", &ErrNotFound{Name: name, Registry: r.Name()}
		}
		return nil, "", fmt.Errorf("load catalog service %q: %w", name, err)
	}
	manifest, err := services.ParseManifest(data)
	if err != nil {
		return nil, "", fmt.Errorf("parse catalog service %q: %w", name, err)
	}
	return manifest, "registry:" + name, nil
}

func (r *EmbeddedRegistry) List(_ context.Context) ([]string, error) {
	return catalog.List()
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

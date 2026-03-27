package registry

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/dunialabs/kimbap-core/internal/services"
	"github.com/dunialabs/kimbap-core/skills"
)

type EmbeddedRegistry struct{}

func NewEmbeddedRegistry() *EmbeddedRegistry {
	return &EmbeddedRegistry{}
}

func (r *EmbeddedRegistry) Name() string { return "official" }

func (r *EmbeddedRegistry) Resolve(_ context.Context, name string) (*services.ServiceManifest, string, error) {
	data, err := skills.Get(name)
	if err != nil {
		if isNotExist(err) {
			return nil, "", &ErrNotFound{Name: name, Registry: r.Name()}
		}
		return nil, "", fmt.Errorf("load official service %q: %w", name, err)
	}
	manifest, err := services.ParseManifest(data)
	if err != nil {
		return nil, "", fmt.Errorf("parse official service %q: %w", name, err)
	}
	return manifest, "official:" + name, nil
}

func (r *EmbeddedRegistry) List(_ context.Context) ([]string, error) {
	return skills.List()
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

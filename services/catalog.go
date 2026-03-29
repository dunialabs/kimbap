package catalog

import (
	"embed"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed catalog/*.yaml
var embeddedCatalogFS embed.FS

func List() ([]string, error) {
	entries, err := fs.ReadDir(embeddedCatalogFS, "catalog")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".yaml"))
	}
	sort.Strings(names)
	return names, nil
}

func Get(name string) ([]byte, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(embeddedCatalogFS, "catalog/"+normalized+".yaml")
}

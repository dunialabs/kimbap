package skills_test

import (
	"errors"
	"io/fs"
	"sort"
	"testing"

	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/skills"
)

func TestListReturnsSortedUniqueOfficialNames(t *testing.T) {
	names, err := skills.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(names) == 0 {
		t.Fatal("expected at least one official service")
	}
	if !sort.StringsAreSorted(names) {
		t.Fatalf("List() must return sorted names, got: %v", names)
	}

	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			t.Fatal("List() returned empty service name")
		}
		if _, ok := seen[name]; ok {
			t.Fatalf("List() returned duplicate service name %q", name)
		}
		seen[name] = struct{}{}
	}
}

func TestGetLoadsAndParsesAllListedServices(t *testing.T) {
	names, err := skills.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			data, getErr := skills.Get(name)
			if getErr != nil {
				t.Fatalf("Get(%q) error = %v", name, getErr)
			}

			manifest, parseErr := services.ParseManifest(data)
			if parseErr != nil {
				t.Fatalf("ParseManifest(%q) error = %v", name, parseErr)
			}
			if manifest.Name != name {
				t.Fatalf("manifest name = %q, want %q", manifest.Name, name)
			}

			defs, convErr := services.ToActionDefinitions(manifest)
			if convErr != nil {
				t.Fatalf("ToActionDefinitions(%q) error = %v", name, convErr)
			}
			if len(defs) != len(manifest.Actions) {
				t.Fatalf("definitions count = %d, want %d", len(defs), len(manifest.Actions))
			}
		})
	}
}

func TestGetRejectsBlankAndUnknownNames(t *testing.T) {
	if _, err := skills.Get("   "); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Get(blank) error = %v, want fs.ErrNotExist", err)
	}
	if _, err := skills.Get("definitely-not-a-real-service"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Get(unknown) error = %v, want fs.ErrNotExist", err)
	}
}

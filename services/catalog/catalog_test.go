package catalog_test

import (
	"errors"
	"io/fs"
	"sort"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/services"
	catalog "github.com/dunialabs/kimbap/services/catalog"
)

func TestListReturnsSortedUniqueCatalogNames(t *testing.T) {
	names, err := catalog.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(names) == 0 {
		t.Fatal("expected at least one catalog service")
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
	names, err := catalog.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			data, getErr := catalog.Get(name)
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
	if _, err := catalog.Get("   "); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Get(blank) error = %v, want fs.ErrNotExist", err)
	}
	if _, err := catalog.Get("definitely-not-a-real-service"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Get(unknown) error = %v, want fs.ErrNotExist", err)
	}
}

func TestNotionCreatePageDefinitionIncludesFriendlyAndAdvancedFields(t *testing.T) {
	data, err := catalog.Get("notion")
	if err != nil {
		t.Fatalf("Get(notion) error = %v", err)
	}

	manifest, err := services.ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest(notion) error = %v", err)
	}
	defs, err := services.ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions(notion) error = %v", err)
	}

	var createPage *services.ServiceAction
	if action, ok := manifest.Actions["create-page"]; ok {
		createPage = &action
	}
	if createPage == nil {
		t.Fatal("notion manifest missing create-page action")
	}

	requiredArgs := map[string]bool{"parent_id": false, "database_id": false, "icon": false, "cover": false}
	for _, arg := range createPage.Args {
		if _, ok := requiredArgs[arg.Name]; ok {
			requiredArgs[arg.Name] = true
		}
	}
	for name, found := range requiredArgs {
		if !found {
			t.Fatalf("expected create-page arg %q in notion manifest, got %+v", name, createPage.Args)
		}
	}
	if body := createPage.Request.Body; body["icon"] != "{icon}" || body["cover"] != "{cover}" {
		t.Fatalf("expected create-page request body to include icon/cover placeholders, got %+v", body)
	}

	for i := range defs {
		if defs[i].Name == "notion.create-page" {
			if defs[i].InputSchema == nil {
				t.Fatal("notion.create-page input schema is nil")
			}
			for _, key := range []string{"parent_id", "database_id", "icon", "cover"} {
				if defs[i].InputSchema.Properties[key] == nil {
					t.Fatalf("expected notion.create-page input schema to include %q, got %+v", key, defs[i].InputSchema.Properties)
				}
			}
			if !strings.Contains(defs[i].Adapter.RequestBody, `"icon":"{icon}"`) || !strings.Contains(defs[i].Adapter.RequestBody, `"cover":"{cover}"`) {
				t.Fatalf("expected notion.create-page request body template to include icon/cover, got %q", defs[i].Adapter.RequestBody)
			}
			return
		}
	}

	t.Fatal("notion.create-page action definition missing after conversion")
}

func TestNotionUpdatePageDefinitionIncludesFriendlyAndAdvancedFields(t *testing.T) {
	data, err := catalog.Get("notion")
	if err != nil {
		t.Fatalf("Get(notion) error = %v", err)
	}

	manifest, err := services.ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest(notion) error = %v", err)
	}
	defs, err := services.ToActionDefinitions(manifest)
	if err != nil {
		t.Fatalf("ToActionDefinitions(notion) error = %v", err)
	}

	action, ok := manifest.Actions["update-page"]
	if !ok {
		t.Fatal("notion manifest missing update-page action")
	}

	requiredArgs := map[string]bool{"title": false, "title_property": false, "icon": false, "cover": false, "in_trash": false}
	for _, arg := range action.Args {
		if _, ok := requiredArgs[arg.Name]; ok {
			requiredArgs[arg.Name] = true
		}
	}
	for name, found := range requiredArgs {
		if !found {
			t.Fatalf("expected update-page arg %q in notion manifest, got %+v", name, action.Args)
		}
	}
	if body := action.Request.Body; body["icon"] != "{icon}" || body["cover"] != "{cover}" || body["in_trash"] != "{in_trash}" {
		t.Fatalf("expected update-page request body to include icon/cover/in_trash placeholders, got %+v", body)
	}

	for i := range defs {
		if defs[i].Name == "notion.update-page" {
			if defs[i].InputSchema == nil {
				t.Fatal("notion.update-page input schema is nil")
			}
			for _, key := range []string{"title", "title_property", "icon", "cover", "in_trash"} {
				if defs[i].InputSchema.Properties[key] == nil {
					t.Fatalf("expected notion.update-page input schema to include %q, got %+v", key, defs[i].InputSchema.Properties)
				}
			}
			if !strings.Contains(defs[i].Adapter.RequestBody, `"icon":"{icon}"`) || !strings.Contains(defs[i].Adapter.RequestBody, `"cover":"{cover}"`) || !strings.Contains(defs[i].Adapter.RequestBody, `"in_trash":"{in_trash}"`) {
				t.Fatalf("expected notion.update-page request body template to include icon/cover/in_trash, got %q", defs[i].Adapter.RequestBody)
			}
			return
		}
	}

	t.Fatal("notion.update-page action definition missing after conversion")
}

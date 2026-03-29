package services

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAllCatalogServicesValidate(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	catalogDir := filepath.Join(repoRoot, "services", "catalog")
	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		t.Fatalf("read services/catalog: %v", err)
	}

	yamlCount := 0
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}
		yamlCount++
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(catalogDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			manifest, err := ParseManifest(data)
			if err != nil {
				t.Fatalf("validation failed: %v", err)
			}
			if manifest == nil {
				t.Fatal("manifest is nil after successful parse")
			}
			if len(manifest.Actions) == 0 {
				t.Error("manifest has no actions")
			}
		})
	}

	if yamlCount == 0 {
		t.Fatal("no YAML files found in services/catalog/")
	}
	t.Logf("validated %d catalog service manifests", yamlCount)
}

func TestAllCatalogServicesHavePackMetadata(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	catalogDir := filepath.Join(repoRoot, "services", "catalog")
	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		t.Fatalf("read services/catalog: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(catalogDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			manifest, err := ParseManifest(data)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			if manifest.Triggers == nil {
				t.Error("missing triggers — required for LLM routing quality")
			} else {
				if len(manifest.Triggers.TaskVerbs) == 0 {
					t.Error("triggers.task_verbs is empty — required for description generation")
				}
				if len(manifest.Triggers.Objects) == 0 {
					t.Error("triggers.objects is empty — required for description generation")
				}
			}

			hasRiskyAction := false
			hasWarnings := false
			for _, action := range manifest.Actions {
				level := strings.ToLower(strings.TrimSpace(action.Risk.Level))
				if level != "" && level != "low" {
					hasRiskyAction = true
				}
				if len(action.Warnings) > 0 {
					hasWarnings = true
				}
			}
			if (hasRiskyAction || hasWarnings) && len(manifest.Gotchas) == 0 {
				t.Error("missing gotchas — required for services with non-low-risk or warned actions")
			}
		})
	}
}

package services

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

func TestAllCatalogServicesConvert(t *testing.T) {
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

	converted := 0
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}
		converted++
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
			defs, err := ToActionDefinitions(manifest)
			if err != nil {
				t.Fatalf("ToActionDefinitions() error: %v", err)
			}
			if len(defs) == 0 {
				t.Fatal("ToActionDefinitions() returned 0 definitions")
			}
			if len(defs) != len(manifest.Actions) {
				t.Fatalf("ToActionDefinitions() returned %d definitions, want %d", len(defs), len(manifest.Actions))
			}
			for _, def := range defs {
				if def.Adapter.Type == "" {
					t.Errorf("definition %q has empty adapter type", def.Name)
				}
				if def.Name == "" {
					t.Error("definition has empty name")
				}
				if def.Namespace == "" {
					t.Error("definition has empty namespace")
				}
			}
		})
	}

	if converted == 0 {
		t.Fatal("no YAML files found in services/catalog/")
	}
	t.Logf("converted %d catalog service manifests via ToActionDefinitions()", converted)
}

func TestValidateExistingCommandServices(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	catalogDir := filepath.Join(repoRoot, "services", "catalog")

	type actionCheck struct {
		name       string
		command    string
		argCount   int
		idempotent bool
	}

	tests := []struct {
		yamlFile     string
		serviceName  string
		executable   string
		jsonFlag     string
		timeout      time.Duration
		actionCount  int
		actionChecks []actionCheck
	}{
		{
			yamlFile:    "kitty.yaml",
			serviceName: "kitty",
			executable:  "kitten",
			jsonFlag:    "none",
			timeout:     30 * time.Second,
			actionCount: 3,
			actionChecks: []actionCheck{
				{name: "kitty.close-window", command: "@ close-window", argCount: 2, idempotent: false},
				{name: "kitty.focus-window", command: "@ focus-window", argCount: 1, idempotent: false},
				{name: "kitty.list-structure", command: "@ ls", argCount: 0, idempotent: true},
			},
		},
		{
			yamlFile:    "mermaid.yaml",
			serviceName: "mermaid",
			executable:  "mmdc",
			jsonFlag:    "off",
			timeout:     30 * time.Second,
			actionCount: 1,
			actionChecks: []actionCheck{
				{name: "mermaid.render", command: "--quiet", argCount: 2, idempotent: true},
			},
		},
		{
			yamlFile:    "blender.yaml",
			serviceName: "blender",
			executable:  "blender",
			jsonFlag:    "--json",
			timeout:     300 * time.Second,
			actionCount: 6,
			actionChecks: []actionCheck{
				{name: "blender.add-mesh", command: "mesh add", argCount: 4, idempotent: false},
				{name: "blender.create-scene", command: "scene new", argCount: 1, idempotent: false},
				{name: "blender.list-objects", command: "object list", argCount: 1, idempotent: true},
				{name: "blender.render", command: "render execute", argCount: 5, idempotent: false},
				{name: "blender.scene-info", command: "scene info", argCount: 1, idempotent: true},
				{name: "blender.set-material", command: "material set", argCount: 3, idempotent: false},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.yamlFile, func(t *testing.T) {
			path := filepath.Join(catalogDir, tc.yamlFile)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			manifest, err := ParseManifest(data)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			defs, err := ToActionDefinitions(manifest)
			if err != nil {
				t.Fatalf("ToActionDefinitions() error: %v", err)
			}

			if len(defs) != tc.actionCount {
				t.Fatalf("action count = %d, want %d", len(defs), tc.actionCount)
			}

			defMap := make(map[string]int, len(defs))
			for i, def := range defs {
				defMap[def.Name] = i
			}

			for i, def := range defs {
				if def.Adapter.Type != "command" {
					t.Errorf("defs[%d] (%s): adapter.type = %q, want command", i, def.Name, def.Adapter.Type)
				}
				if def.Adapter.ExecutablePath != tc.executable {
					t.Errorf("defs[%d] (%s): adapter.executable_path = %q, want %q", i, def.Name, def.Adapter.ExecutablePath, tc.executable)
				}
				if def.Adapter.JSONFlag != tc.jsonFlag {
					t.Errorf("defs[%d] (%s): adapter.json_flag = %q, want %q", i, def.Name, def.Adapter.JSONFlag, tc.jsonFlag)
				}
				if def.Adapter.Timeout != tc.timeout {
					t.Errorf("defs[%d] (%s): adapter.timeout = %s, want %s", i, def.Name, def.Adapter.Timeout, tc.timeout)
				}
				if def.Namespace != tc.serviceName {
					t.Errorf("defs[%d] (%s): namespace = %q, want %q", i, def.Name, def.Namespace, tc.serviceName)
				}
			}

			for _, ac := range tc.actionChecks {
				idx, exists := defMap[ac.name]
				if !exists {
					t.Errorf("expected action %q not found in definitions", ac.name)
					continue
				}
				def := defs[idx]

				if def.Adapter.Command != ac.command {
					t.Errorf("action %q: adapter.command = %q, want %q", ac.name, def.Adapter.Command, ac.command)
				}
				if def.Idempotent != ac.idempotent {
					t.Errorf("action %q: idempotent = %v, want %v", ac.name, def.Idempotent, ac.idempotent)
				}

				gotArgCount := 0
				if def.InputSchema != nil {
					gotArgCount = len(def.InputSchema.Properties)
				}
				if gotArgCount != ac.argCount {
					t.Errorf("action %q: arg count = %d, want %d", ac.name, gotArgCount, ac.argCount)
				}
			}
		})
	}
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

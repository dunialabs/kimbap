package services

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAllOfficialSkillsValidate(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	skillsDir := filepath.Join(repoRoot, "skills", "official")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("read skills/official: %v", err)
	}

	yamlCount := 0
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}
		yamlCount++
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(skillsDir, entry.Name())
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
		t.Fatal("no YAML files found in skills/official/")
	}
	t.Logf("validated %d skill manifests", yamlCount)
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSkillPackDirRemovesStaleFiles(t *testing.T) {
	base := t.TempDir()
	serviceDir := filepath.Join(base, "out", "github")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "SKILL.md"), []byte("old skill\n"), 0o644); err != nil {
		t.Fatalf("seed SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "GOTCHAS.md"), []byte("old gotchas\n"), 0o644); err != nil {
		t.Fatalf("seed GOTCHAS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "STALE.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("seed STALE.md: %v", err)
	}

	pack := map[string]string{
		"SKILL.md":   "new skill\n",
		"RECIPES.md": "new recipes\n",
	}

	written, err := writeAgentSkillPackDir(serviceDir, pack)
	if err != nil {
		t.Fatalf("writeSkillPackDir: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("written count = %d, want 2", len(written))
	}

	if _, err := os.Stat(filepath.Join(serviceDir, "GOTCHAS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected stale GOTCHAS.md removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(serviceDir, "STALE.md")); !os.IsNotExist(err) {
		t.Fatalf("expected stale STALE.md removed, stat err=%v", err)
	}

	skill, err := os.ReadFile(filepath.Join(serviceDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if string(skill) != "new skill\n" {
		t.Fatalf("SKILL.md content = %q, want %q", string(skill), "new skill\\n")
	}

	recipes, err := os.ReadFile(filepath.Join(serviceDir, "RECIPES.md"))
	if err != nil {
		t.Fatalf("read RECIPES.md: %v", err)
	}
	if string(recipes) != "new recipes\n" {
		t.Fatalf("RECIPES.md content = %q, want %q", string(recipes), "new recipes\\n")
	}
}

func TestWriteSkillPackDirRejectsUnsafeFileNames(t *testing.T) {
	base := t.TempDir()
	serviceDir := filepath.Join(base, "out", "github")

	_, err := writeAgentSkillPackDir(serviceDir, map[string]string{"../escape.md": "bad"})
	if err == nil {
		t.Fatal("expected unsafe filename error")
	}

	_, err = writeAgentSkillPackDir(serviceDir, map[string]string{"nested/child.md": "bad"})
	if err == nil {
		t.Fatal("expected nested filename error")
	}
}

func TestWriteSkillPackDirPreservesExistingDirOnUnsafeName(t *testing.T) {
	base := t.TempDir()
	serviceDir := filepath.Join(base, "out", "github")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "SKILL.md"), []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := writeAgentSkillPackDir(serviceDir, map[string]string{"../escape.md": "bad"})
	if err == nil {
		t.Fatal("expected unsafe filename error")
	}

	if _, statErr := os.Stat(filepath.Join(serviceDir, "SKILL.md")); statErr != nil {
		t.Fatalf("existing SKILL.md should not have been removed on validation error: %v", statErr)
	}
}

func TestWriteSkillPackDirReturnsFinalTargetPaths(t *testing.T) {
	base := t.TempDir()
	serviceDir := filepath.Join(base, "out", "github")

	pack := map[string]string{
		"SKILL.md":   "# skill\n",
		"GOTCHAS.md": "# gotchas\n",
	}

	written, err := writeAgentSkillPackDir(serviceDir, pack)
	if err != nil {
		t.Fatalf("writeSkillPackDir: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("written count = %d, want 2", len(written))
	}
	for _, p := range written {
		if !filepath.IsAbs(p) {
			t.Fatalf("expected absolute path, got %q", p)
		}
		if dir := filepath.Dir(p); dir != serviceDir {
			t.Fatalf("expected path under serviceDir %q, got dir %q in %q", serviceDir, dir, p)
		}
		if _, statErr := os.Stat(p); statErr != nil {
			t.Fatalf("reported path %q does not exist: %v", p, statErr)
		}
	}
}

func TestWriteSkillPackDirLeavesNoTmpDirOnSuccess(t *testing.T) {
	base := t.TempDir()
	parentDir := filepath.Join(base, "out")
	serviceDir := filepath.Join(parentDir, "github")

	_, err := writeAgentSkillPackDir(serviceDir, map[string]string{"SKILL.md": "# skill\n"})
	if err != nil {
		t.Fatalf("writeSkillPackDir: %v", err)
	}

	entries, readErr := os.ReadDir(parentDir)
	if readErr != nil {
		t.Fatalf("read parent dir: %v", readErr)
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != "github" {
			t.Fatalf("unexpected leftover directory in parent: %q", e.Name())
		}
	}
}

func TestWriteSkillPackDirLeavesNoTmpDirOnValidationFailure(t *testing.T) {
	base := t.TempDir()
	parentDir := filepath.Join(base, "out")
	serviceDir := filepath.Join(parentDir, "github")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, _ = writeAgentSkillPackDir(serviceDir, map[string]string{"../escape.md": "bad"})

	entries, readErr := os.ReadDir(parentDir)
	if readErr != nil {
		t.Fatalf("read parent dir: %v", readErr)
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != "github" {
			t.Fatalf("unexpected leftover directory in parent after validation error: %q", e.Name())
		}
	}
}

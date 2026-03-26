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

	written, err := writeSkillPackDir(serviceDir, pack)
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

	_, err := writeSkillPackDir(serviceDir, map[string]string{"../escape.md": "bad"})
	if err == nil {
		t.Fatal("expected unsafe filename error")
	}

	_, err = writeSkillPackDir(serviceDir, map[string]string{"nested/child.md": "bad"})
	if err == nil {
		t.Fatal("expected nested filename error")
	}
}

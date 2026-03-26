package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSkillPackDirRemovesStaleBackupDir(t *testing.T) {
	dir := t.TempDir()
	serviceDir := filepath.Join(dir, "export", "github")
	backup := serviceDir + ".old"

	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("create service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "old.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write old export file: %v", err)
	}
	if err := os.MkdirAll(backup, 0o755); err != nil {
		t.Fatalf("create stale backup dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backup, "stale.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write stale backup file: %v", err)
	}

	written, err := writeSkillPackDir(serviceDir, map[string]string{"SKILL.md": "# new\n"})
	if err != nil {
		t.Fatalf("writeSkillPackDir with stale backup: %v", err)
	}
	if len(written) != 1 || written[0] != filepath.Join(serviceDir, "SKILL.md") {
		t.Fatalf("unexpected written files: %+v", written)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("expected stale backup dir removed, stat err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(serviceDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read exported skill file: %v", err)
	}
	if string(data) != "# new\n" {
		t.Fatalf("unexpected SKILL.md content: %q", string(data))
	}
}

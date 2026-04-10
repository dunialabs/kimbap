package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDirectOutputPathSafeRejectsSymlinkTarget(t *testing.T) {
	base := t.TempDir()
	realTarget := filepath.Join(base, "real.txt")
	linkPath := filepath.Join(base, "out.txt")
	if err := os.WriteFile(realTarget, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write real target: %v", err)
	}
	if err := os.Symlink(realTarget, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	err := ensureDirectOutputPathSafe(linkPath)
	if err == nil {
		t.Fatal("expected symlinked output path to be rejected")
	}
	if !strings.Contains(err.Error(), "symlinked output path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

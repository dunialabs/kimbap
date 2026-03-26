package service

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceDirectoryPreservesExistingDirOnStageError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "services", "github")

	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write existing target file: %v", err)
	}

	src := filepath.Join(dir, "src")
	blocked := filepath.Join(src, "blocked")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatalf("create blocked source dir: %v", err)
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatalf("chmod blocked source dir: %v", err)
	}
	defer os.Chmod(blocked, 0o755)

	err := replaceDirectory(src, target)
	if err == nil {
		t.Fatal("expected replaceDirectory to fail when staging source dir")
	}
	if !strings.Contains(err.Error(), "stage service directory") {
		t.Fatalf("expected staging error, got %v", err)
	}
	if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup dir on staging failure, stat err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(target, "old.txt"))
	if err != nil {
		t.Fatalf("read preserved target file: %v", err)
	}
	if string(data) != "old\n" {
		t.Fatalf("unexpected preserved target content: %q", string(data))
	}
}

func TestReplaceDirectoryRemovesStaleBackupDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "services", "github")
	backup := target + ".old"
	src := filepath.Join(dir, "src")

	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	if err := os.MkdirAll(backup, 0o755); err != nil {
		t.Fatalf("create stale backup dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backup, "stale.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write stale backup file: %v", err)
	}
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("# new\n"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	if err := replaceDirectory(src, target); err != nil {
		t.Fatalf("replaceDirectory with stale backup: %v", err)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("expected stale backup dir removed, stat err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
	if err != nil {
		t.Fatalf("read replaced file: %v", err)
	}
	if string(data) != "# new\n" {
		t.Fatalf("unexpected replaced content: %q", string(data))
	}
}

func TestUploadServiceRejectsMetadataDirectoryNameMismatch(t *testing.T) {
	service := &ServicesService{servicesDir: t.TempDir()}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	entry, err := zw.Create(filepath.ToSlash(filepath.Join("github", "SKILL.md")))
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	content := `---
name: slack
description: Slack service
version: 1.0.0
---
`
	if _, err := entry.Write([]byte(content)); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	uploaded, err := service.UploadService("tenant", zipBuf.Bytes())
	if err == nil {
		t.Fatalf("expected metadata/directory mismatch error, got uploaded=%v", uploaded)
	}
	if !strings.Contains(err.Error(), `service directory "github" does not match metadata name "slack"`) {
		t.Fatalf("unexpected upload error: %v", err)
	}

	entries, readErr := os.ReadDir(filepath.Join(service.servicesDir, "tenant"))
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("read tenant service dir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no uploaded services on mismatch, found %d", len(entries))
	}
}

func TestUploadServiceRejectsDuplicateZipEntryTargets(t *testing.T) {
	service := &ServicesService{servicesDir: t.TempDir()}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	for _, body := range []string{`---
name: github
description: First
version: 1.0.0
---
`, `---
name: github
description: Second
version: 1.0.0
---
`} {
		entry, err := zw.Create(filepath.ToSlash(filepath.Join("github", "SKILL.md")))
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	uploaded, err := service.UploadService("tenant", zipBuf.Bytes())
	if err == nil {
		t.Fatalf("expected duplicate zip entry error, got uploaded=%v", uploaded)
	}
	if !strings.Contains(err.Error(), "duplicate zip entry target") {
		t.Fatalf("unexpected upload error: %v", err)
	}
	if len(uploaded) != 0 {
		t.Fatalf("expected no uploaded services on duplicate entry error, got %v", uploaded)
	}
}

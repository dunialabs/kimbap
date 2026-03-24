package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleManifest = `name: test-skill
version: 1.0.0
description: Test skill
base_url: https://api.example.com
auth:
  type: header
  header_name: Authorization
  credential_ref: test.token
actions:
  ping:
    method: GET
    path: /ping
    description: Ping endpoint
    response:
      extract: data
      type: object
    risk:
      level: low
      mutating: false
`

func TestRegistryInstallCreatesLockEntryWithSHA256(t *testing.T) {
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	manifestPath := filepath.Join(tempDir, "skill.yaml")
	if err := os.WriteFile(manifestPath, []byte(sampleManifest), 0o644); err != nil {
		t.Fatalf("write manifest fixture: %v", err)
	}

	r := NewRegistry(skillsDir)
	entry, err := r.Install(context.Background(), "local", manifestPath)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if entry == nil {
		t.Fatal("expected lock entry")
	}
	if entry.Name != "test-skill" {
		t.Fatalf("unexpected entry name: %s", entry.Name)
	}

	installedBytes, err := os.ReadFile(filepath.Join(skillsDir, "test-skill.yaml"))
	if err != nil {
		t.Fatalf("read installed manifest: %v", err)
	}
	expectedSum := sha256.Sum256(installedBytes)
	if entry.SHA256 != hex.EncodeToString(expectedSum[:]) {
		t.Fatalf("unexpected digest: %s", entry.SHA256)
	}

	lf, err := r.LoadLockfile()
	if err != nil {
		t.Fatalf("load lockfile: %v", err)
	}
	if len(lf.Entries) != 1 {
		t.Fatalf("expected 1 lock entry, got %d", len(lf.Entries))
	}
	if lf.Entries[0].SHA256 == "" {
		t.Fatal("expected lockfile digest to be set")
	}
}

func TestRegistryVerifyDetectsTamperedSkill(t *testing.T) {
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	manifestPath := filepath.Join(tempDir, "skill.yaml")
	if err := os.WriteFile(manifestPath, []byte(sampleManifest), 0o644); err != nil {
		t.Fatalf("write manifest fixture: %v", err)
	}

	r := NewRegistry(skillsDir)
	if _, err := r.Install(context.Background(), "local", manifestPath); err != nil {
		t.Fatalf("install: %v", err)
	}

	tampered := strings.Replace(sampleManifest, "1.0.0", "9.9.9", 1)
	if err := os.WriteFile(filepath.Join(skillsDir, "test-skill.yaml"), []byte(tampered), 0o644); err != nil {
		t.Fatalf("tamper installed file: %v", err)
	}

	results, err := r.Verify(context.Background())
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 verify result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Fatalf("expected fail status, got %s", results[0].Status)
	}
}

func TestRegistryVerifyPassesUntamperedFiles(t *testing.T) {
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	manifestPath := filepath.Join(tempDir, "skill.yaml")
	if err := os.WriteFile(manifestPath, []byte(sampleManifest), 0o644); err != nil {
		t.Fatalf("write manifest fixture: %v", err)
	}

	r := NewRegistry(skillsDir)
	if _, err := r.Install(context.Background(), "local", manifestPath); err != nil {
		t.Fatalf("install: %v", err)
	}

	results, err := r.Verify(context.Background())
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 verify result, got %d", len(results))
	}
	if results[0].Status != "ok" {
		t.Fatalf("expected ok status, got %s", results[0].Status)
	}
}

func TestRegistryDiffShowsMeaningfulChanges(t *testing.T) {
	r := NewRegistry(t.TempDir())
	oldManifest := []byte("name: test-skill\nversion: 1.0.0\n")
	newManifest := []byte("name: test-skill\nversion: 1.1.0\n")

	diff := r.Diff(oldManifest, newManifest)
	if !strings.Contains(diff, "- version: 1.0.0") {
		t.Fatalf("expected removed line in diff, got: %s", diff)
	}
	if !strings.Contains(diff, "+ version: 1.1.0") {
		t.Fatalf("expected added line in diff, got: %s", diff)
	}
}

package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetProfileReturnsValidTemplateForEachType(t *testing.T) {
	types := []ProfileType{ProfileClaudeCode, ProfileOpenCode, ProfileCodex, ProfileGeneric, ProfileCursor}
	for _, profileType := range types {
		profile, err := GetProfile(profileType)
		if err != nil {
			t.Fatalf("GetProfile(%s): %v", profileType, err)
		}
		if profile == nil {
			t.Fatalf("GetProfile(%s) returned nil profile", profileType)
		}
		if profile.Template == "" {
			t.Fatalf("GetProfile(%s) returned empty template", profileType)
		}
		if profile.InstallPath == "" {
			t.Fatalf("GetProfile(%s) returned empty install path", profileType)
		}
	}
}

func TestInstallProfileWritesToCorrectPath(t *testing.T) {
	targetDir := t.TempDir()

	claude, err := GetProfile(ProfileClaudeCode)
	if err != nil {
		t.Fatalf("GetProfile(claude-code): %v", err)
	}
	if err := InstallProfile(claude, targetDir); err != nil {
		t.Fatalf("InstallProfile(claude): %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, ".claude", "KIMBAP_OPERATING_RULES.md")); err != nil {
		t.Fatalf("expected claude profile file: %v", err)
	}

	generic, err := GetProfile(ProfileGeneric)
	if err != nil {
		t.Fatalf("GetProfile(generic): %v", err)
	}
	if err := InstallProfile(generic, targetDir); err != nil {
		t.Fatalf("InstallProfile(generic): %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, ".agents", "KIMBAP_OPERATING_RULES.md")); err != nil {
		t.Fatalf("expected generic profile file: %v", err)
	}
}

func TestPrintProfileReturnsNonEmptyContent(t *testing.T) {
	content, err := PrintProfile(ProfileCursor)
	if err != nil {
		t.Fatalf("PrintProfile(cursor): %v", err)
	}
	if content == "" {
		t.Fatal("expected non-empty profile content")
	}
}

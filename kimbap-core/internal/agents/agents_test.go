package agents

import (
	"os"
	"path/filepath"
	"testing"
)

type fakeInstaller struct {
	skills []InstalledSkill
	err    error
}

func (f fakeInstaller) List() ([]InstalledSkill, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.skills, nil
}

func TestDetectAgents(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string)
		expected   []AgentKind
		unexpected []AgentKind
	}{
		{
			name: "detects claude from .claude directory",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755); err != nil {
					t.Fatalf("create .claude directory: %v", err)
				}
			},
			expected:   []AgentKind{AgentClaudeCode},
			unexpected: []AgentKind{AgentOpenCode, AgentCodex, AgentCursor, AgentGeneric},
		},
		{
			name:       "empty directory detects none",
			setup:      func(t *testing.T, _ string) { t.Helper() },
			expected:   []AgentKind{},
			unexpected: []AgentKind{AgentClaudeCode, AgentOpenCode, AgentCodex, AgentCursor, AgentGeneric},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)

			detected := DetectAgents(dir)
			kinds := make(map[AgentKind]bool, len(detected))
			for _, a := range detected {
				kinds[a.Kind] = true
			}

			for _, expectedKind := range tt.expected {
				if !kinds[expectedKind] {
					t.Fatalf("expected agent %q to be detected", expectedKind)
				}
			}
			for _, unexpectedKind := range tt.unexpected {
				if kinds[unexpectedKind] {
					t.Fatalf("did not expect agent %q to be detected", unexpectedKind)
				}
			}

			if len(tt.expected) == 0 && len(detected) != 0 {
				t.Fatalf("expected zero detected agents, got %d", len(detected))
			}
		})
	}
}

func TestSyncSkills(t *testing.T) {
	tests := []struct {
		name               string
		dryRun             bool
		expectFilesWritten bool
	}{
		{name: "dry run reports writes without creating files", dryRun: true, expectFilesWritten: false},
		{name: "actual sync writes files", dryRun: false, expectFilesWritten: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			rules := "# rules\n"
			installer := fakeInstaller{skills: []InstalledSkill{{Name: "github-pr", Content: "# SKILL\n"}}}

			results, err := SyncSkills(installer, rules, SyncOptions{
				ProjectDir: dir,
				Agents:     []AgentKind{AgentClaudeCode},
				DryRun:     tt.dryRun,
			})
			if err != nil {
				t.Fatalf("sync failed: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected one result, got %d", len(results))
			}

			result := results[0]
			if result.Agent != AgentClaudeCode {
				t.Fatalf("unexpected agent result: %q", result.Agent)
			}
			if len(result.Written) != 1 || result.Written[0] != "github-pr" {
				t.Fatalf("expected github-pr in written list, got %+v", result.Written)
			}
			if !result.RulesWritten {
				t.Fatal("expected rules to be marked written")
			}

			skillPath := filepath.Join(dir, ".claude", "skills", "github-pr", "SKILL.md")
			rulesPath := filepath.Join(dir, ".claude", "KIMBAP_OPERATING_RULES.md")

			if tt.expectFilesWritten {
				skillData, err := os.ReadFile(skillPath)
				if err != nil {
					t.Fatalf("read synced skill file: %v", err)
				}
				if string(skillData) != "# SKILL\n" {
					t.Fatalf("unexpected synced skill content: %q", string(skillData))
				}

				rulesData, err := os.ReadFile(rulesPath)
				if err != nil {
					t.Fatalf("read synced rules file: %v", err)
				}
				if string(rulesData) != rules {
					t.Fatalf("unexpected rules content: %q", string(rulesData))
				}
			} else {
				if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
					t.Fatalf("expected skill file to not exist in dry run, stat err=%v", err)
				}
				if _, err := os.Stat(rulesPath); !os.IsNotExist(err) {
					t.Fatalf("expected rules file to not exist in dry run, stat err=%v", err)
				}
			}
		})
	}
}

func TestStatusAfterSync(t *testing.T) {
	dir := t.TempDir()
	installer := fakeInstaller{skills: []InstalledSkill{{Name: "github-pr", Content: "# SKILL\n"}}}

	if _, err := SyncSkills(installer, "# rules\n", SyncOptions{ProjectDir: dir, Agents: []AgentKind{AgentClaudeCode}}); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	statuses, err := Status(dir)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}

	byAgent := make(map[AgentKind]StatusResult, len(statuses))
	for _, status := range statuses {
		byAgent[status.Agent] = status
	}

	claude, ok := byAgent[AgentClaudeCode]
	if !ok {
		t.Fatal("expected claude-code status to exist")
	}
	if !claude.Detected {
		t.Fatal("expected claude-code to be detected after sync")
	}
	if !claude.RulesPresent {
		t.Fatal("expected claude rules file to exist")
	}
	if len(claude.SyncedSkills) != 1 || claude.SyncedSkills[0] != "github-pr" {
		t.Fatalf("unexpected claude synced skills: %+v", claude.SyncedSkills)
	}

	if generic, ok := byAgent[AgentGeneric]; ok {
		if generic.Detected {
			t.Fatal("did not expect generic agent to be detected")
		}
	}
}

func TestIsAgentDetectedDetailedReturnsErrorOnNonDirectoryPath(t *testing.T) {
	dir := t.TempDir()
	projectFile := filepath.Join(dir, "project-file")
	if err := os.WriteFile(projectFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write project file: %v", err)
	}

	detected, err := isAgentDetectedDetailed(projectFile, AgentConfig{DetectPaths: []string{".claude"}})
	if detected {
		t.Fatal("expected detected=false when project path is not a directory")
	}
	if err == nil {
		t.Fatal("expected detection error for non-directory project path")
	}
}

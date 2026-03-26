package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeInstaller struct {
	skills []InstalledService
	err    error
}

func (f fakeInstaller) List() ([]InstalledService, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.skills, nil
}

type fakePackInstaller struct {
	skills []InstalledService
	packs  []InstalledServicePack
	err    error
}

func (f fakePackInstaller) List() ([]InstalledService, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.skills, nil
}

func (f fakePackInstaller) ListPacks() ([]InstalledServicePack, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.packs, nil
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
			installer := fakeInstaller{skills: []InstalledService{{Name: "github-pr", Content: "# SKILL\n"}}}

			results, err := SyncServices(installer, rules, SyncOptions{
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
	installer := fakeInstaller{skills: []InstalledService{{Name: "github-pr", Content: "# SKILL\n"}}}

	if _, err := SyncServices(installer, "# rules\n", SyncOptions{ProjectDir: dir, Agents: []AgentKind{AgentClaudeCode}}); err != nil {
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
	if len(claude.SyncedServices) != 1 || claude.SyncedServices[0] != "github-pr" {
		t.Fatalf("unexpected claude synced skills: %+v", claude.SyncedServices)
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

func TestSyncSkillsRejectsFileProjectDir(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	installer := fakeInstaller{skills: []InstalledService{{Name: "s", Content: "# S\n"}}}
	_, err := SyncServices(installer, "# rules\n", SyncOptions{ProjectDir: filePath})
	if err == nil {
		t.Fatal("expected error when ProjectDir is a file")
	}
}

func TestSyncSkillsUnknownAgentReturnsError(t *testing.T) {
	dir := t.TempDir()
	installer := fakeInstaller{skills: []InstalledService{{Name: "github-pr", Content: "# SKILL\n"}}}

	results, err := SyncServices(installer, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{"nonexistent-agent"},
	})
	if err != nil {
		t.Fatalf("SyncSkills returned unexpected top-level error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Errors) == 0 {
		t.Fatal("expected errors for unknown agent kind")
	}
	if len(results[0].Written) != 0 {
		t.Fatalf("expected no written skills for unknown agent, got %+v", results[0].Written)
	}
}

func TestSyncSkillsSkipsUnchangedContent(t *testing.T) {
	dir := t.TempDir()
	installer := fakeInstaller{skills: []InstalledService{{Name: "github-pr", Content: "# SKILL\n"}}}

	first, err := SyncServices(installer, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
	})
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if len(first[0].Written) != 1 {
		t.Fatalf("expected 1 written on first sync, got %d", len(first[0].Written))
	}

	second, err := SyncServices(installer, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
	})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if len(second[0].Written) != 0 {
		t.Fatalf("expected 0 written on unchanged second sync, got %d", len(second[0].Written))
	}
	if len(second[0].Skipped) != 1 {
		t.Fatalf("expected 1 skipped on unchanged second sync, got %d", len(second[0].Skipped))
	}
}

func TestSyncSkillsForceOverwritesUnchanged(t *testing.T) {
	dir := t.TempDir()
	installer := fakeInstaller{skills: []InstalledService{{Name: "github-pr", Content: "# SKILL\n"}}}

	if _, err := SyncServices(installer, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
	}); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	forced, err := SyncServices(installer, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
		Force:      true,
	})
	if err != nil {
		t.Fatalf("force sync: %v", err)
	}
	if len(forced[0].Written) != 1 {
		t.Fatalf("expected 1 written on force sync, got %d", len(forced[0].Written))
	}
	if len(forced[0].Skipped) != 0 {
		t.Fatalf("expected 0 skipped on force sync, got %d", len(forced[0].Skipped))
	}
}

func TestSyncSkillsPrunesRemovedSkills(t *testing.T) {
	dir := t.TempDir()
	both := fakeInstaller{skills: []InstalledService{
		{Name: "github", Content: "# github\n"},
		{Name: "slack", Content: "# slack\n"},
	}}

	if _, err := SyncServices(both, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
	}); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	githubOnly := fakeInstaller{skills: []InstalledService{
		{Name: "github", Content: "# github\n"},
	}}
	results, err := SyncServices(githubOnly, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
	})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}

	if len(results[0].Pruned) != 1 || results[0].Pruned[0] != "slack" {
		t.Fatalf("expected slack to be pruned, got %+v", results[0].Pruned)
	}

	slackDir := filepath.Join(dir, ".claude", "skills", "slack")
	if _, err := os.Stat(slackDir); !os.IsNotExist(err) {
		t.Fatal("expected slack directory to be removed after prune")
	}
}

func TestSyncServicesSkipPrunePreservesStaleDir(t *testing.T) {
	dir := t.TempDir()

	staleDir := filepath.Join(dir, ".claude", "skills", "old-service")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatalf("create stale service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staleDir, "SKILL.md"), []byte("# old\n"), 0o644); err != nil {
		t.Fatalf("write stale skill file: %v", err)
	}

	installer := fakeInstaller{skills: []InstalledService{{Name: "github", Content: "# github\n"}}}

	results, err := SyncServices(installer, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
		SkipPrune:  true,
	})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if len(results[0].Pruned) != 0 {
		t.Fatalf("expected no pruned entries when SkipPrune=true, got %+v", results[0].Pruned)
	}

	if _, err := os.Stat(staleDir); err != nil {
		t.Fatalf("expected stale service dir to remain when SkipPrune=true, stat err=%v", err)
	}
}

func TestSyncServicesFallsBackWhenPackCoverageIsPartial(t *testing.T) {
	dir := t.TempDir()
	installer := fakePackInstaller{
		skills: []InstalledService{
			{Name: "github", Content: "# github\n"},
			{Name: "slack", Content: "# slack\n"},
		},
		packs: []InstalledServicePack{
			{
				Name:         "github",
				AgentSkillMD: "# github\n",
				PackFiles: map[string]string{
					"GOTCHAS.md": "# gotchas\n",
				},
			},
		},
	}

	results, err := SyncServices(installer, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
	})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	result := results[0]
	if len(result.Errors) > 0 {
		t.Fatalf("expected no sync errors, got %+v", result.Errors)
	}
	if len(result.Written) != 2 {
		t.Fatalf("expected legacy fallback to write two services, got %+v", result.Written)
	}

	for _, service := range []string{"github", "slack"} {
		skillPath := filepath.Join(dir, ".claude", "skills", service, "SKILL.md")
		if _, statErr := os.Stat(skillPath); statErr != nil {
			t.Fatalf("expected %s skill file to exist, stat err=%v", service, statErr)
		}
	}
}

func TestSyncServicesFallsBackWhenPackNamesDuplicate(t *testing.T) {
	dir := t.TempDir()
	installer := fakePackInstaller{
		skills: []InstalledService{
			{Name: "github", Content: "# github\n"},
			{Name: "slack", Content: "# slack\n"},
		},
		packs: []InstalledServicePack{
			{Name: "github", AgentSkillMD: "# g1\n"},
			{Name: "github", AgentSkillMD: "# g2\n"},
		},
	}

	results, err := SyncServices(installer, "# rules\n", SyncOptions{ProjectDir: dir, Agents: []AgentKind{AgentClaudeCode}})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if len(results[0].Errors) > 0 {
		t.Fatalf("expected no sync errors, got %+v", results[0].Errors)
	}

	for _, service := range []string{"github", "slack"} {
		skillPath := filepath.Join(dir, ".claude", "skills", service, "SKILL.md")
		if _, statErr := os.Stat(skillPath); statErr != nil {
			t.Fatalf("expected %s skill file via legacy fallback, stat err=%v", service, statErr)
		}
	}
}

func TestSyncServicesFallsBackWhenPackHasNoFiles(t *testing.T) {
	dir := t.TempDir()
	installer := fakePackInstaller{
		skills: []InstalledService{{Name: "github", Content: "# github\n"}},
		packs:  []InstalledServicePack{{Name: "github"}},
	}

	results, err := SyncServices(installer, "# rules\n", SyncOptions{ProjectDir: dir, Agents: []AgentKind{AgentClaudeCode}})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if len(results[0].Errors) > 0 {
		t.Fatalf("expected no sync errors, got %+v", results[0].Errors)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "github", "SKILL.md")
	if _, statErr := os.Stat(skillPath); statErr != nil {
		t.Fatalf("expected github SKILL.md via legacy fallback, stat err=%v", statErr)
	}
}

func TestPackNeedsWriteDetectsUnexpectedFiles(t *testing.T) {
	packDir := filepath.Join(t.TempDir(), "svc")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "SKILL.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "GOTCHAS.md"), []byte("# gotchas\n"), 0o644); err != nil {
		t.Fatalf("write gotchas: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "OLD.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write old: %v", err)
	}

	needsWrite, err := packNeedsWrite(packDir, map[string]string{
		"SKILL.md":   "# skill\n",
		"GOTCHAS.md": "# gotchas\n",
	}, false)
	if err != nil {
		t.Fatalf("packNeedsWrite error: %v", err)
	}
	if !needsWrite {
		t.Fatal("expected packNeedsWrite=true when unexpected file exists")
	}
}

func TestSyncResultAndStatusResultJSONFieldNames(t *testing.T) {
	sr := SyncResult{
		Agent:          AgentClaudeCode,
		AgentSkillsDir: "/some/path",
		Written:        []string{"svc-a"},
	}
	b, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("marshal SyncResult: %v", err)
	}
	if !strings.Contains(string(b), `"agent_skills_dir"`) {
		t.Errorf("SyncResult JSON missing agent_skills_dir field, got: %s", b)
	}

	st := StatusResult{
		Agent:          AgentOpenCode,
		SyncedServices: []string{"svc-b"},
	}
	b2, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("marshal StatusResult: %v", err)
	}
	if !strings.Contains(string(b2), `"synced_services"`) {
		t.Errorf("StatusResult JSON missing synced_services field, got: %s", b2)
	}
}

func TestPackNeedsWrite(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "github")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("create pack dir: %v", err)
	}
	packFiles := map[string]string{
		"SKILL.md":   "# skill\n",
		"GOTCHAS.md": "# gotchas\n",
	}
	for name, content := range packFiles {
		if err := os.WriteFile(filepath.Join(packDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("seed pack file %s: %v", name, err)
		}
	}

	needsWrite, err := packNeedsWrite(packDir, packFiles, false)
	if err != nil {
		t.Fatalf("packNeedsWrite: %v", err)
	}
	if needsWrite {
		t.Fatal("expected unchanged pack to not require write")
	}
}

func TestPackNeedsWriteContentDiffers(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "github")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("create pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "SKILL.md"), []byte("# old\n"), 0o644); err != nil {
		t.Fatalf("seed skill file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "GOTCHAS.md"), []byte("# gotchas\n"), 0o644); err != nil {
		t.Fatalf("seed gotchas file: %v", err)
	}

	needsWrite, err := packNeedsWrite(packDir, map[string]string{
		"SKILL.md":   "# new\n",
		"GOTCHAS.md": "# gotchas\n",
	}, false)
	if err != nil {
		t.Fatalf("packNeedsWrite: %v", err)
	}
	if !needsWrite {
		t.Fatal("expected content diff to require write")
	}
}

func TestPackNeedsWriteMissingFile(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "github")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("create pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "SKILL.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatalf("seed skill file: %v", err)
	}

	needsWrite, err := packNeedsWrite(packDir, map[string]string{
		"SKILL.md":   "# skill\n",
		"GOTCHAS.md": "# gotchas\n",
	}, false)
	if err != nil {
		t.Fatalf("packNeedsWrite: %v", err)
	}
	if !needsWrite {
		t.Fatal("expected missing pack file to require write")
	}
}

func TestSyncServicesPack(t *testing.T) {
	dir := t.TempDir()
	installer := fakePackInstaller{
		skills: []InstalledService{{Name: "github", Content: "# legacy\n"}},
		packs: []InstalledServicePack{{
			Name:         "github",
			AgentSkillMD: "# SKILL\n",
			PackFiles: map[string]string{
				"GOTCHAS.md": "# GOTCHAS\n",
				"RECIPES.md": "# RECIPES\n",
			},
		}},
	}

	results, err := SyncServices(installer, "# rules\n", SyncOptions{
		ProjectDir: dir,
		Agents:     []AgentKind{AgentClaudeCode},
	})
	if err != nil {
		t.Fatalf("sync pack services: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if len(results[0].Written) != 1 || results[0].Written[0] != "github" {
		t.Fatalf("expected github pack written, got %+v", results[0].Written)
	}
	if !results[0].RulesWritten {
		t.Fatal("expected rules to be written")
	}

	base := filepath.Join(dir, ".claude", "skills", "github")
	checks := map[string]string{
		"SKILL.md":   "# SKILL\n",
		"GOTCHAS.md": "# GOTCHAS\n",
		"RECIPES.md": "# RECIPES\n",
	}
	for name, want := range checks {
		data, readErr := os.ReadFile(filepath.Join(base, name))
		if readErr != nil {
			t.Fatalf("read pack file %s: %v", name, readErr)
		}
		if string(data) != want {
			t.Fatalf("unexpected content for %s: %q", name, string(data))
		}
	}
}

func TestPruneStaleSkillsPackDir(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".claude", "skills")
	for _, name := range []string{"github", "stale", "kimbap"} {
		path := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create skill dir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("# skill\n"), 0o644); err != nil {
			t.Fatalf("write skill file %s: %v", name, err)
		}
	}

	pruned, errs := pruneStaleServices(skillsDir, []InstalledService{{Name: "github"}}, false)
	if len(errs) != 0 {
		t.Fatalf("unexpected prune errors: %+v", errs)
	}
	if len(pruned) != 1 || pruned[0] != "stale" {
		t.Fatalf("expected stale to be pruned, got %+v", pruned)
	}

	if _, err := os.Stat(filepath.Join(skillsDir, "stale")); !os.IsNotExist(err) {
		t.Fatalf("expected stale dir removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(skillsDir, "github")); err != nil {
		t.Fatalf("expected github dir kept, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(skillsDir, "kimbap")); err != nil {
		t.Fatalf("expected kimbap dir kept, stat err=%v", err)
	}
}

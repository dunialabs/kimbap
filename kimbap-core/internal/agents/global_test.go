package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectMarkerBlock(t *testing.T) {
	makeBlock := func(body string) string {
		return markerStart + "\n" + body + markerEnd + "\n"
	}

	tests := []struct {
		name         string
		setup        func(t *testing.T, path string)
		force        bool
		dryRun       bool
		expectChange bool
		assert       func(t *testing.T, path string)
	}{
		{
			name:         "new file",
			setup:        func(t *testing.T, _ string) { t.Helper() },
			expectChange: true,
			assert: func(t *testing.T, path string) {
				t.Helper()
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read injected file: %v", err)
				}
				content := string(data)
				if !strings.Contains(content, markerStart) || !strings.Contains(content, markerEnd) {
					t.Fatalf("expected marker block in file, got %q", content)
				}
			},
		},
		{
			name: "existing file with content",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("base\n"), 0o644); err != nil {
					t.Fatalf("write base file: %v", err)
				}
			},
			expectChange: true,
			assert: func(t *testing.T, path string) {
				t.Helper()
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read injected file: %v", err)
				}
				content := string(data)
				if !strings.Contains(content, "base\n") {
					t.Fatalf("expected original content preserved, got %q", content)
				}
				if !strings.Contains(content, markerStart) {
					t.Fatalf("expected marker block appended, got %q", content)
				}
			},
		},
		{
			name: "replace existing block",
			setup: func(t *testing.T, path string) {
				t.Helper()
				old := makeBlock("old instructions\n")
				content := "before\n" + old + "after\n"
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write file with old block: %v", err)
				}
			},
			expectChange: true,
			assert: func(t *testing.T, path string) {
				t.Helper()
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read replaced file: %v", err)
				}
				content := string(data)
				if strings.Contains(content, "old instructions") {
					t.Fatalf("expected old block removed, got %q", content)
				}
				if !strings.Contains(content, globalInstructionBlock) {
					t.Fatalf("expected global instruction block, got %q", content)
				}
				if !strings.Contains(content, "before\n") || !strings.Contains(content, "after\n") {
					t.Fatalf("expected surrounding content preserved, got %q", content)
				}
			},
		},
		{
			name: "idempotent when same block exists",
			setup: func(t *testing.T, path string) {
				t.Helper()
				block := markerStart + "\n" + globalInstructionBlock + markerEnd + "\n"
				content := "before\n" + block + "after\n"
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write file with existing block: %v", err)
				}
			},
			expectChange: false,
			assert: func(t *testing.T, path string) {
				t.Helper()
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read idempotent file: %v", err)
				}
				content := string(data)
				if strings.Count(content, markerStart) != 1 {
					t.Fatalf("expected single marker block, got %q", content)
				}
			},
		},
		{
			name: "force overwrite existing same block",
			setup: func(t *testing.T, path string) {
				t.Helper()
				block := markerStart + "\n" + globalInstructionBlock + markerEnd + "\n"
				if err := os.WriteFile(path, []byte("before\n"+block+"after\n"), 0o644); err != nil {
					t.Fatalf("write file with same block: %v", err)
				}
			},
			force:        true,
			expectChange: true,
			assert: func(t *testing.T, path string) {
				t.Helper()
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read forced file: %v", err)
				}
				content := string(data)
				if strings.Count(content, markerStart) != 1 {
					t.Fatalf("expected single marker block after force, got %q", content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "AGENTS.md")
			tt.setup(t, path)

			changed, err := injectMarkerBlock(path, tt.force, tt.dryRun)
			if err != nil {
				t.Fatalf("inject marker block: %v", err)
			}
			if changed != tt.expectChange {
				t.Fatalf("expected changed=%v, got %v", tt.expectChange, changed)
			}

			tt.assert(t, path)
		})
	}
}

func TestRemoveMarkerBlock(t *testing.T) {
	block := markerStart + "\n" + globalInstructionBlock + markerEnd + "\n"

	tests := []struct {
		name         string
		initial      string
		expectChange bool
		assert       func(t *testing.T, content string)
	}{
		{
			name:         "remove existing block",
			initial:      "before\n" + block + "after\n",
			expectChange: true,
			assert: func(t *testing.T, content string) {
				t.Helper()
				if strings.Contains(content, markerStart) || strings.Contains(content, markerEnd) {
					t.Fatalf("expected marker block removed, got %q", content)
				}
				if !strings.Contains(content, "before") || !strings.Contains(content, "after") {
					t.Fatalf("expected surrounding content preserved, got %q", content)
				}
			},
		},
		{
			name:         "no-op when no block present",
			initial:      "before\nafter\n",
			expectChange: false,
			assert: func(t *testing.T, content string) {
				t.Helper()
				if content != "before\nafter\n" {
					t.Fatalf("content should be unchanged, got %q", content)
				}
			},
		},
		{
			name:         "preserve content outside markers",
			initial:      "header\n\n" + block + "\nfooter\n",
			expectChange: true,
			assert: func(t *testing.T, content string) {
				t.Helper()
				if strings.Contains(content, markerStart) {
					t.Fatalf("expected marker block removed, got %q", content)
				}
				if content != "header\n\nfooter\n" {
					t.Fatalf("expected blank line preserved between header and footer, got %q", content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "AGENTS.md")
			if err := os.WriteFile(path, []byte(tt.initial), 0o644); err != nil {
				t.Fatalf("write initial file: %v", err)
			}

			changed, err := removeMarkerBlock(path, false)
			if err != nil {
				t.Fatalf("remove marker block: %v", err)
			}
			if changed != tt.expectChange {
				t.Fatalf("expected changed=%v, got %v", tt.expectChange, changed)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read file after remove: %v", err)
			}
			tt.assert(t, string(data))
		})
	}
}

func TestGlobalSetupOneAndTeardownOne(t *testing.T) {
	dir := t.TempDir()
	cfg := GlobalAgentConfig{
		Kind:            AgentClaudeCode,
		SkillsDir:       filepath.Join(dir, "skills"),
		InstructionFile: filepath.Join(dir, "CLAUDE.md"),
		DetectDir:       filepath.Join(dir, "detect"),
	}
	if err := os.MkdirAll(cfg.DetectDir, 0o755); err != nil {
		t.Fatalf("create detect dir: %v", err)
	}

	setup := globalSetupOne(cfg, "# meta\n", GlobalSetupOptions{})
	if setup.Error != "" {
		t.Fatalf("global setup failed: %s", setup.Error)
	}
	if !setup.ServiceWritten {
		t.Fatal("expected skill to be written")
	}
	if !setup.InjectWritten {
		t.Fatal("expected instruction block to be injected")
	}

	skillPath := filepath.Join(cfg.SkillsDir, "kimbap", "SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read skill file: %v", err)
	}
	if string(skillData) != "# meta\n" {
		t.Fatalf("unexpected skill content: %q", string(skillData))
	}

	instructionData, err := os.ReadFile(cfg.InstructionFile)
	if err != nil {
		t.Fatalf("read instruction file: %v", err)
	}
	if !strings.Contains(string(instructionData), markerStart) {
		t.Fatalf("expected marker block injected, got %q", string(instructionData))
	}

	teardown := globalTeardownOne(cfg, false)
	if teardown.Error != "" {
		t.Fatalf("global teardown failed: %s", teardown.Error)
	}
	if !teardown.ServiceRemoved {
		t.Fatal("expected skill directory to be removed")
	}
	if !teardown.InjectRemoved {
		t.Fatal("expected injected marker block to be removed")
	}

	if _, err := os.Stat(filepath.Join(cfg.SkillsDir, "kimbap")); !os.IsNotExist(err) {
		t.Fatalf("expected skill directory removed, stat err=%v", err)
	}

	instructionData, err = os.ReadFile(cfg.InstructionFile)
	if err != nil {
		t.Fatalf("read instruction file after teardown: %v", err)
	}
	if strings.Contains(string(instructionData), markerStart) {
		t.Fatalf("expected marker block removed, got %q", string(instructionData))
	}
}

func TestGlobalSetupOneSkippedWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	cfg := GlobalAgentConfig{
		Kind:            AgentClaudeCode,
		SkillsDir:       filepath.Join(dir, "skills"),
		InstructionFile: filepath.Join(dir, "CLAUDE.md"),
		DetectDir:       filepath.Join(dir, "detect"),
	}
	if err := os.MkdirAll(cfg.DetectDir, 0o755); err != nil {
		t.Fatalf("create detect dir: %v", err)
	}

	first := globalSetupOne(cfg, "# meta\n", GlobalSetupOptions{})
	if first.Error != "" {
		t.Fatalf("first setup failed: %s", first.Error)
	}
	if !first.ServiceWritten || !first.InjectWritten {
		t.Fatalf("expected first setup to write skill and inject, got %+v", first)
	}

	second := globalSetupOne(cfg, "# meta\n", GlobalSetupOptions{})
	if second.Error != "" {
		t.Fatalf("second setup failed: %s", second.Error)
	}
	if !second.Skipped {
		t.Fatalf("expected skipped=true on unchanged second setup, got %+v", second)
	}
	if second.ServiceWritten {
		t.Fatalf("expected skill_written=false on unchanged second setup, got %+v", second)
	}
	if second.InjectWritten {
		t.Fatalf("expected inject_written=false on unchanged second setup, got %+v", second)
	}
}

func TestGlobalTeardownCodexArtifactWithoutDetectDir(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	codexSkillDir := filepath.Join(home, ".agents", "skills", "kimbap")
	if err := os.MkdirAll(codexSkillDir, 0o755); err != nil {
		t.Fatalf("create codex kimbap skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexSkillDir, "SKILL.md"), []byte("# codex meta\n"), 0o644); err != nil {
		t.Fatalf("write codex skill file: %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("expected ~/.codex to be absent for this test, stat err=%v", err)
	}

	results, err := GlobalTeardown(GlobalSetupOptions{})
	if err != nil {
		t.Fatalf("global teardown failed: %v", err)
	}

	foundCodex := false
	for _, r := range results {
		if r.Agent == AgentCodex {
			foundCodex = true
			if r.Error != "" {
				t.Fatalf("codex teardown reported error: %s", r.Error)
			}
			if !r.ServiceRemoved {
				t.Fatalf("expected codex skill artifact removal, got %+v", r)
			}
		}
	}
	if !foundCodex {
		t.Fatalf("expected codex to be included in auto-detected teardown targets, got %+v", results)
	}

	if _, err := os.Stat(codexSkillDir); !os.IsNotExist(err) {
		t.Fatalf("expected codex kimbap skill dir removed, stat err=%v", err)
	}
}

func TestGlobalDetectAgents(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatalf("create claude dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(xdg, "opencode"), 0o755); err != nil {
		t.Fatalf("create opencode dir: %v", err)
	}

	detected, err := GlobalDetectAgents()
	if err != nil {
		t.Fatalf("global detect agents: %v", err)
	}

	kinds := make(map[AgentKind]bool, len(detected))
	for _, cfg := range detected {
		kinds[cfg.Kind] = true
	}

	if !kinds[AgentClaudeCode] {
		t.Fatal("expected claude-code to be detected")
	}
	if !kinds[AgentOpenCode] {
		t.Fatal("expected opencode to be detected")
	}
	if kinds[AgentCodex] {
		t.Fatal("did not expect codex to be detected")
	}
	if kinds[AgentCursor] {
		t.Fatal("did not expect cursor to be detected")
	}
}

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	if err := atomicWriteFile(path, "first\n"); err != nil {
		t.Fatalf("first atomic write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after first write: %v", err)
	}
	if string(data) != "first\n" {
		t.Fatalf("unexpected first content: %q", string(data))
	}

	if err := atomicWriteFile(path, "second\n"); err != nil {
		t.Fatalf("second atomic write: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after second write: %v", err)
	}
	if string(data) != "second\n" {
		t.Fatalf("unexpected second content: %q", string(data))
	}
}

func TestGlobalStatus(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, "xdg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	claudeDir := filepath.Join(home, ".claude")
	claudeSkillPath := filepath.Join(claudeDir, "skills", "kimbap", "SKILL.md")
	claudeInstruction := filepath.Join(claudeDir, "CLAUDE.md")

	if err := os.MkdirAll(filepath.Dir(claudeSkillPath), 0o755); err != nil {
		t.Fatalf("create claude skills dir: %v", err)
	}
	if err := os.WriteFile(claudeSkillPath, []byte("# skill\n"), 0o644); err != nil {
		t.Fatalf("write claude skill file: %v", err)
	}
	if err := os.WriteFile(claudeInstruction, []byte("x\n"+markerStart+"\n"+markerEnd+"\n"), 0o644); err != nil {
		t.Fatalf("write claude instruction file: %v", err)
	}

	statuses, err := GlobalStatus()
	if err != nil {
		t.Fatalf("global status: %v", err)
	}

	byAgent := make(map[AgentKind]GlobalStatusResult, len(statuses))
	for _, status := range statuses {
		byAgent[status.Agent] = status
	}

	claude, ok := byAgent[AgentClaudeCode]
	if !ok {
		t.Fatal("expected claude-code status")
	}
	if !claude.Detected {
		t.Fatal("expected claude-code detected")
	}
	if !claude.ServicePresent {
		t.Fatal("expected claude-code skill present")
	}
	if !claude.InjectPresent {
		t.Fatal("expected claude-code injection present")
	}
	if claude.InstructionFile != claudeInstruction {
		t.Fatalf("unexpected claude instruction file: %q", claude.InstructionFile)
	}

	opencode, ok := byAgent[AgentOpenCode]
	if !ok {
		t.Fatal("expected opencode status")
	}
	if opencode.Detected {
		t.Fatal("did not expect opencode detected")
	}
	if opencode.ServicePresent {
		t.Fatal("did not expect opencode skill present")
	}
}

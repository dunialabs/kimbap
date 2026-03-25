package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCustomStdioRunnerLaunchPlanRejectsInvalidCwd(t *testing.T) {
	_, err := BuildCustomStdioRunnerLaunchPlan(map[string]any{
		"command": "node",
		"cwd":     filepath.Join(t.TempDir(), "missing-dir"),
	}, "")
	if err == nil {
		t.Fatal("expected error for invalid cwd")
	}
	if !strings.Contains(err.Error(), "cwd is invalid") {
		t.Fatalf("expected cwd validation error, got %v", err)
	}
}

func TestBuildCustomStdioRunnerLaunchPlanAcceptsValidCwd(t *testing.T) {
	cwd := t.TempDir()
	plan, err := BuildCustomStdioRunnerLaunchPlan(map[string]any{
		"command": "node",
		"args":    []any{"server.js"},
		"cwd":     cwd,
	}, "")
	if err != nil {
		t.Fatalf("expected successful plan build, got %v", err)
	}

	args, ok := plan.LaunchConfig["args"].([]any)
	if !ok {
		t.Fatalf("expected args slice, got %T", plan.LaunchConfig["args"])
	}
	joined := strings.Join(toStringSlice(args), " ")
	if !strings.Contains(joined, "-w "+cwd) {
		t.Fatalf("expected docker workdir argument for cwd %q, args=%q", cwd, joined)
	}
}

func TestBuildCustomStdioRunnerLaunchPlanRejectsRelativeCwdInDockerMode(t *testing.T) {
	t.Setenv("KIMBAP_CORE_IN_DOCKER", "true")

	_, err := BuildCustomStdioRunnerLaunchPlan(map[string]any{
		"command": "node",
		"cwd":     "relative/path",
	}, "")
	if err == nil {
		t.Fatal("expected error for relative cwd in Docker mode")
	}
	if !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("expected absolute-path error, got %v", err)
	}
}

func TestBuildCustomStdioRunnerLaunchPlanRejectsFileCwd(t *testing.T) {
	base := t.TempDir()
	cwdFile := filepath.Join(base, "cwd.txt")
	if err := os.WriteFile(cwdFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write cwd file: %v", err)
	}

	_, err := BuildCustomStdioRunnerLaunchPlan(map[string]any{
		"command": "node",
		"cwd":     cwdFile,
	}, "")
	if err == nil {
		t.Fatal("expected error for non-directory cwd")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

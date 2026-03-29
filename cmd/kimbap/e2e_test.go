package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var testBinary string

func TestMain(m *testing.M) {
	tmp, err := os.CreateTemp("", "kimbap-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp binary path: %v\n", err)
		os.Exit(1)
	}
	tmp.Close()
	testBinary = tmp.Name()

	cmd := exec.Command("go", "build", "-o", testBinary, ".")
	cmd.Dir = "."
	if out, buildErr := cmd.CombinedOutput(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n%s\n", buildErr, string(out))
		os.Exit(1)
	}

	code := m.Run()
	_ = os.Remove(testBinary)
	os.Exit(code)
}

func e2eRun(t *testing.T, env []string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(testBinary, args...)
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func e2eRunInput(t *testing.T, env []string, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(testBinary, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func e2eEnv(t *testing.T) []string {
	t.Helper()
	dataDir := t.TempDir()
	homeDir := t.TempDir()
	xdgConfigHome := filepath.Join(homeDir, "xdg")
	if err := os.MkdirAll(xdgConfigHome, 0o700); err != nil {
		t.Fatalf("mkdir xdg config home: %v", err)
	}

	return []string{
		"KIMBAP_DATA_DIR=" + dataDir,
		"KIMBAP_DEV=true",
		"HOME=" + homeDir,
		"XDG_CONFIG_HOME=" + xdgConfigHome,
	}
}

func testdataAppleNotesPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	path := filepath.Join(wd, "..", "..", "testdata", "apple-notes.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("apple-notes fixture not found at %q: %v", path, err)
	}
	return path
}

func mustRunOK(t *testing.T, env []string, args ...string) (string, string) {
	t.Helper()
	stdout, stderr, err := e2eRun(t, env, args...)
	if err != nil {
		t.Fatalf("command failed: %v\nargs: %v\nstdout:\n%s\nstderr:\n%s", err, args, stdout, stderr)
	}
	return stdout, stderr
}

func TestE2EGoldenPath(t *testing.T) {
	env := e2eEnv(t)
	fixture := testdataAppleNotesPath(t)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")
	mustRunOK(t, env, "--no-splash", "service", "install", fixture)
	stdout, _ := mustRunOK(t, env, "--no-splash", "actions", "list", "--format", "json")

	var actionsList []map[string]any
	if err := json.Unmarshal([]byte(stdout), &actionsList); err != nil {
		t.Fatalf("actions list output is not valid JSON: %v\noutput:\n%s", err, stdout)
	}

	found := false
	for _, item := range actionsList {
		name, _ := item["Name"].(string)
		namespace, _ := item["Namespace"].(string)
		if strings.EqualFold(namespace, "apple-notes") || strings.HasPrefix(strings.ToLower(name), "apple-notes.") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected installed apple-notes actions in list, got: %s", stdout)
	}
}

func TestE2EServiceInstallAndDescribe(t *testing.T) {
	env := e2eEnv(t)
	fixture := testdataAppleNotesPath(t)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")
	mustRunOK(t, env, "--no-splash", "service", "install", fixture)
	stdout, _ := mustRunOK(t, env, "--no-splash", "actions", "describe", "apple-notes.list-notes", "--format", "json")

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("describe output is not valid JSON: %v\noutput:\n%s", err, stdout)
	}
	actionRaw, ok := payload["action"].(map[string]any)
	if !ok {
		t.Fatalf("expected action object in describe payload, got: %s", stdout)
	}
	if got, _ := actionRaw["Name"].(string); got != "apple-notes.list-notes" {
		t.Fatalf("expected action.name=apple-notes.list-notes, got %q", got)
	}
	if got, _ := actionRaw["Namespace"].(string); got != "apple-notes" {
		t.Fatalf("expected action.namespace=apple-notes, got %q", got)
	}
	if _, ok := payload["credential_ready"].(bool); !ok {
		t.Fatalf("expected credential_ready bool in payload, got: %s", stdout)
	}
	if _, ok := payload["approval_required"].(bool); !ok {
		t.Fatalf("expected approval_required bool in payload, got: %s", stdout)
	}
}

func TestE2EDeprecatedConnector(t *testing.T) {
	env := e2eEnv(t)
	stdout, stderr, err := e2eRun(t, env, "--no-splash", "connector", "login", "github")
	if err == nil {
		t.Fatalf("expected non-zero exit for deprecated connector command\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	combined := strings.ToLower(stdout + "\n" + stderr)
	if !strings.Contains(combined, "removed") {
		t.Fatalf("expected deprecation output to contain 'removed'\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestE2EDeprecatedProfile(t *testing.T) {
	env := e2eEnv(t)
	stdout, stderr, err := e2eRun(t, env, "--no-splash", "profile", "install", "claude-code")
	if err == nil {
		t.Fatalf("expected non-zero exit for deprecated profile command\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	combined := strings.ToLower(stdout + "\n" + stderr)
	if !strings.Contains(combined, "removed") {
		t.Fatalf("expected deprecation output to contain 'removed'\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestE2ELinkStdin(t *testing.T) {
	env := e2eEnv(t)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")
	mustRunOK(t, env, "--no-splash", "service", "install", "github")
	stdout, stderr, err := e2eRunInput(t, env, "dummy-token\n", "--no-splash", "link", "github", "--stdin", "--format", "json")
	if err != nil {
		t.Fatalf("link --stdin failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("link output is not valid JSON: %v\noutput:\n%s", err, stdout)
	}
	if got, _ := payload["service"].(string); got != "github" {
		t.Fatalf("expected service=github, got %q", got)
	}
	if got, _ := payload["status"].(string); got != "connected" {
		t.Fatalf("expected status=connected, got %q", got)
	}
	if got, _ := payload["credential_ref"].(string); got != "github.token" {
		t.Fatalf("expected credential_ref=github.token, got %q", got)
	}
}

func TestE2EDoctor(t *testing.T) {
	env := e2eEnv(t)
	fixture := testdataAppleNotesPath(t)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")
	mustRunOK(t, env, "--no-splash", "service", "install", fixture)
	if _, stderr, err := e2eRunInput(t, env, "doctor-secret\n", "--no-splash", "vault", "set", "doctor.token", "--stdin"); err != nil {
		t.Fatalf("vault set failed before doctor check: %v\nstderr:\n%s", err, stderr)
	}
	stdout, _ := mustRunOK(t, env, "--no-splash", "doctor", "--format", "json")

	var checks []map[string]any
	if err := json.Unmarshal([]byte(stdout), &checks); err != nil {
		t.Fatalf("doctor output is not valid JSON: %v\noutput:\n%s", err, stdout)
	}
	if len(checks) == 0 {
		t.Fatalf("expected doctor checks, got empty output: %s", stdout)
	}
}

func TestE2EHelpSurface(t *testing.T) {
	env := e2eEnv(t)
	stdout, stderr, err := e2eRun(t, env, "--no-splash", "--help")
	if err != nil {
		t.Fatalf("help command failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	commands := extractAvailableCommands(stdout)
	delete(commands, "help")
	if len(commands) > 16 {
		t.Fatalf("expected <=16 visible commands, got %d: %v\nhelp:\n%s", len(commands), sortedKeys(commands), stdout)
	}

	for _, want := range []string{"call", "init", "link", "auth", "service", "vault", "status"} {
		if _, ok := commands[want]; !ok {
			t.Fatalf("expected visible command %q in help output\nhelp:\n%s", want, stdout)
		}
	}

	for _, hidden := range []string{"generate", "token", "audit", "daemon"} {
		if _, ok := commands[hidden]; ok {
			t.Fatalf("expected hidden command %q to be absent from root help", hidden)
		}
	}
}

func TestE2EHiddenCommandsAccessible(t *testing.T) {
	env := e2eEnv(t)
	stdout, stderr, err := e2eRun(t, env, "--no-splash", "generate", "--help")
	if err != nil {
		t.Fatalf("expected hidden generate command to be accessible, got err=%v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
}

func extractAvailableCommands(helpOutput string) map[string]struct{} {
	commands := map[string]struct{}{}
	lines := strings.Split(helpOutput, "\n")
	inSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
			label := strings.ToLower(trimmed)
			if strings.Contains(label, "command") || label == "setup:" || label == "management:" || label == "advanced:" || label == "additional commands:" {
				inSection = true
				continue
			}
			if label == "flags:" || label == "global flags:" {
				inSection = false
				continue
			}
		}
		if !inSection {
			continue
		}
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		if strings.HasPrefix(fields[0], "-") {
			inSection = false
			continue
		}
		commands[fields[0]] = struct{}{}
	}
	return commands
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	for i := 0; i < len(out)-1; i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
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

func e2eEnvValue(t *testing.T, env []string, key string) string {
	t.Helper()
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	t.Fatalf("environment key %q not found in %v", key, env)
	return ""
}

func testdataAppleNotesPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	path := filepath.Join(wd, "..", "..", "services", "catalog", "apple-notes.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("apple-notes fixture not found at %q: %v", path, err)
	}
	return path
}

func writeE2EHTTPManifest(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name+".yaml")
	raw := "name: " + name + "\n" +
		"version: 1.0.0\n" +
		"description: local e2e http service\n" +
		"base_url: https://api.example.com\n" +
		"auth:\n" +
		"  type: none\n" +
		"actions:\n" +
		"  ping:\n" +
		"    method: GET\n" +
		"    path: /ping\n" +
		"    description: ping endpoint\n" +
		"    risk:\n" +
		"      level: low\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write http manifest: %v", err)
	}
	return path
}

func writeE2ECommandManifest(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name+".yaml")
	raw := "name: " + name + "\n" +
		"version: 1.0.0\n" +
		"description: local e2e command service\n" +
		"adapter: command\n" +
		"command_spec:\n" +
		"  executable: /bin/echo\n" +
		"  json_flag: none\n" +
		"auth:\n" +
		"  type: none\n" +
		"actions:\n" +
		"  run:\n" +
		"    command: \"hello-from-command-service\"\n" +
		"    description: echo a stable fixture value\n" +
		"    idempotent: true\n" +
		"    response:\n" +
		"      type: object\n" +
		"      text_filter:\n" +
		"        strip_ansi: true\n" +
		"    risk:\n" +
		"      level: low\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write command manifest: %v", err)
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

func decodeJSONMap(t *testing.T, raw string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, raw)
	}
	return payload
}

func decodeJSONArray(t *testing.T, raw string) []map[string]any {
	t.Helper()
	var payload []map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("output is not valid JSON array: %v\noutput:\n%s", err, raw)
	}
	return payload
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
	if len(commands) > 17 {
		t.Fatalf("expected <=17 visible commands, got %d: %v\nhelp:\n%s", len(commands), sortedKeys(commands), stdout)
	}

	for _, want := range []string{"call", "init", "link", "auth", "service", "vault", "status", "alias"} {
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

func TestE2EServiceValidateSignVerifyFlow(t *testing.T) {
	env := e2eEnv(t)
	serviceName := "local-e2e-verify"
	manifestPath := writeE2EHTTPManifest(t, t.TempDir(), serviceName)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")

	validateOut, _ := mustRunOK(t, env, "--no-splash", "service", "validate", manifestPath)
	if !strings.Contains(validateOut, serviceName+" (1.0.0) is valid") {
		t.Fatalf("expected validate output to mention manifest validity, got:\n%s", validateOut)
	}

	mustRunOK(t, env, "--no-splash", "service", "install", manifestPath)

	verifyOut, _ := mustRunOK(t, env, "--no-splash", "--format", "json", "service", "verify", serviceName)
	verifyPayload := decodeJSONMap(t, verifyOut)
	if got, _ := verifyPayload["name"].(string); got != serviceName {
		t.Fatalf("expected verify name %q, got %q", serviceName, got)
	}
	if got, _ := verifyPayload["verified"].(bool); !got {
		t.Fatalf("expected unsigned service to verify successfully, got %+v", verifyPayload)
	}
	if got, _ := verifyPayload["locked"].(bool); !got {
		t.Fatalf("expected installed service to be lockfile-backed, got %+v", verifyPayload)
	}
	if got, _ := verifyPayload["signed"].(bool); got {
		t.Fatalf("expected unsigned service before service sign, got %+v", verifyPayload)
	}

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate ed25519 keypair: %v", err)
	}

	keyDir := t.TempDir()
	seedPath := filepath.Join(keyDir, "service-sign.key")
	if err := os.WriteFile(seedPath, []byte(hex.EncodeToString(priv.Seed())), 0o600); err != nil {
		t.Fatalf("write signing seed: %v", err)
	}
	pubPath := filepath.Join(keyDir, "service-sign.pub")
	if err := os.WriteFile(pubPath, []byte(hex.EncodeToString(pub)), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	signOut, _ := mustRunOK(t, env, "--no-splash", "service", "sign", "--key", seedPath)
	if !strings.Contains(signOut, "lockfile signed") {
		t.Fatalf("expected sign output to confirm lockfile signing, got:\n%s", signOut)
	}

	signedVerifyOut, _ := mustRunOK(t, env, "--no-splash", "--format", "json", "service", "verify", serviceName, "--key", pubPath)
	signedVerifyPayload := decodeJSONMap(t, signedVerifyOut)
	if got, _ := signedVerifyPayload["verified"].(bool); !got {
		t.Fatalf("expected signed service to remain verified, got %+v", signedVerifyPayload)
	}
	if got, _ := signedVerifyPayload["signed"].(bool); !got {
		t.Fatalf("expected signed=true after service sign, got %+v", signedVerifyPayload)
	}
	if got, _ := signedVerifyPayload["signature_valid"].(bool); !got {
		t.Fatalf("expected signature_valid=true with pinned key, got %+v", signedVerifyPayload)
	}
}

func TestE2EServiceVerifyDetectsTampering(t *testing.T) {
	env := e2eEnv(t)
	serviceName := "local-e2e-tamper"
	manifestPath := writeE2EHTTPManifest(t, t.TempDir(), serviceName)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")
	mustRunOK(t, env, "--no-splash", "service", "install", manifestPath)

	servicesDir := filepath.Join(e2eEnvValue(t, env, "KIMBAP_DATA_DIR"), "services")
	installedPath := filepath.Join(servicesDir, serviceName+".yaml")
	installedData, err := os.ReadFile(installedPath)
	if err != nil {
		t.Fatalf("read installed manifest: %v", err)
	}
	if err := os.WriteFile(installedPath, append(installedData, []byte("\n# tampered\n")...), 0o644); err != nil {
		t.Fatalf("tamper installed manifest: %v", err)
	}

	stdout, stderr, err := e2eRun(t, env, "--no-splash", "--format", "json", "service", "verify", serviceName)
	if err == nil {
		t.Fatalf("expected tampered service verify to fail\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "integrity check failed") {
		t.Fatalf("expected integrity failure in stderr\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	payload := decodeJSONMap(t, stdout)
	if got, _ := payload["verified"].(bool); got {
		t.Fatalf("expected verified=false for tampered manifest, got %+v", payload)
	}
	if got, _ := payload["locked"].(bool); !got {
		t.Fatalf("expected locked=true for tampered manifest, got %+v", payload)
	}
}

func TestE2ECommandServiceCall(t *testing.T) {
	env := e2eEnv(t)
	serviceName := "local-command-e2e"
	manifestPath := writeE2ECommandManifest(t, t.TempDir(), serviceName)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")
	mustRunOK(t, env, "--no-splash", "service", "install", manifestPath)

	stdout, _ := mustRunOK(t, env, "--no-splash", "--format", "json", "call", serviceName+".run")
	payload := decodeJSONMap(t, stdout)

	if got, _ := payload["Status"].(string); got != "success" {
		t.Fatalf("expected call status success, got %+v", payload)
	}
	output, ok := payload["Output"].(map[string]any)
	if !ok {
		t.Fatalf("expected call output object, got %+v", payload)
	}
	raw, _ := output["raw"].(string)
	if !strings.Contains(raw, "hello-from-command-service") {
		t.Fatalf("expected command output in raw payload, got %+v", output)
	}
	meta, ok := payload["Meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected call meta object, got %+v", payload)
	}
	if got, _ := meta["adapter_type"].(string); got != "command" {
		t.Fatalf("expected adapter_type=command, got %+v", meta)
	}
}

func TestE2EServiceEnableDisableLifecycle(t *testing.T) {
	env := e2eEnv(t)
	serviceName := "local-command-lifecycle"
	manifestPath := writeE2ECommandManifest(t, t.TempDir(), serviceName)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")
	mustRunOK(t, env, "--no-splash", "service", "install", manifestPath, "--no-activate")

	listOut, _ := mustRunOK(t, env, "--no-splash", "--format", "json", "service", "list")
	rows := decodeJSONArray(t, listOut)
	foundDisabled := false
	for _, row := range rows {
		if got, _ := row["name"].(string); got != serviceName {
			continue
		}
		if enabled, _ := row["enabled"].(bool); enabled {
			t.Fatalf("expected service %q to be disabled after --no-activate install, got %+v", serviceName, row)
		}
		if status, _ := row["status"].(string); status != "disabled" {
			t.Fatalf("expected service %q status=disabled, got %+v", serviceName, row)
		}
		foundDisabled = true
	}
	if !foundDisabled {
		t.Fatalf("expected service %q in service list output, got %+v", serviceName, rows)
	}

	statusOut, _ := mustRunOK(t, env, "--no-splash", "--format", "json", "status")
	statusPayload := decodeJSONMap(t, statusOut)
	if services, _ := statusPayload["services"].(float64); services != 0 {
		t.Fatalf("expected disabled-only install to keep services count at 0, got %+v", statusPayload)
	}

	stdout, stderr, err := e2eRun(t, env, "--no-splash", "call", serviceName+".run")
	if err == nil {
		t.Fatalf("expected disabled service call to fail\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	combined := stdout + "\n" + stderr
	if !strings.Contains(combined, "no enabled services found") {
		t.Fatalf("expected disabled service failure hint, got\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	mustRunOK(t, env, "--no-splash", "service", "enable", serviceName)

	statusOut, _ = mustRunOK(t, env, "--no-splash", "--format", "json", "status")
	statusPayload = decodeJSONMap(t, statusOut)
	if services, _ := statusPayload["services"].(float64); services != 1 {
		t.Fatalf("expected enabled service count to be 1, got %+v", statusPayload)
	}

	callOut, _ := mustRunOK(t, env, "--no-splash", "--format", "json", "call", serviceName+".run")
	callPayload := decodeJSONMap(t, callOut)
	if got, _ := callPayload["Status"].(string); got != "success" {
		t.Fatalf("expected enabled service call to succeed, got %+v", callPayload)
	}

	mustRunOK(t, env, "--no-splash", "service", "disable", serviceName)

	statusOut, _ = mustRunOK(t, env, "--no-splash", "--format", "json", "status")
	statusPayload = decodeJSONMap(t, statusOut)
	if services, _ := statusPayload["services"].(float64); services != 0 {
		t.Fatalf("expected disabled service count to return to 0, got %+v", statusPayload)
	}
}

func TestE2EStatusShowsLinkHintForDisconnectedService(t *testing.T) {
	env := e2eEnv(t)

	mustRunOK(t, env, "--no-splash", "init", "--mode", "dev", "--no-services")
	mustRunOK(t, env, "--no-splash", "service", "install", "github")

	jsonOut, _ := mustRunOK(t, env, "--no-splash", "--format", "json", "status")
	payload := decodeJSONMap(t, jsonOut)
	if got, _ := payload["services"].(float64); got != 1 {
		t.Fatalf("expected services=1 in status JSON, got %+v", payload)
	}
	if got, _ := payload["credentials"].(float64); got != 0 {
		t.Fatalf("expected credentials=0 in status JSON, got %+v", payload)
	}

	stdout, stderr, err := e2eRun(t, env, "--no-splash", "status")
	if err != nil {
		t.Fatalf("status command failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr from status command, got %q", stderr)
	}
	if !strings.Contains(stdout, "kimbap link github") {
		t.Fatalf("expected status footer to suggest 'kimbap link github', got:\n%s", stdout)
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

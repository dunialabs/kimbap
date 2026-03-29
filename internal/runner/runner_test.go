package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunnerSetsProxyEnvVars(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "env.txt")
	r := NewRunner(RunConfig{
		Command:    []string{"sh", "-c", "printf '%s|%s|%s|%s' \"$HTTP_PROXY\" \"$HTTPS_PROXY\" \"$ALL_PROXY\" \"$KIMBAP_AGENT_TOKEN\" > \"$OUT\""},
		ProxyAddr:  "127.0.0.1:18080",
		AgentToken: "agent-token-1",
		Env: map[string]string{
			"OUT": outFile,
		},
	})

	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("runner start failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	got := string(data)
	wantProxy := "http://kimbap:agent-token-1@127.0.0.1:18080"
	wantToken := "agent-token-1"
	want := wantProxy + "|" + wantProxy + "|" + wantProxy + "|" + wantToken
	if got != want {
		t.Fatalf("unexpected env output:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestBuildEnvEnsuresLoopbackNoProxyEntries(t *testing.T) {
	env := buildEnv([]string{"NO_PROXY=localhost"}, nil, "http://127.0.0.1:18080", "")
	value := ""
	for _, item := range env {
		if strings.HasPrefix(item, "NO_PROXY=") {
			value = strings.TrimPrefix(item, "NO_PROXY=")
			break
		}
	}
	if value == "" {
		t.Fatal("expected NO_PROXY to be set")
	}
	for _, required := range []string{"localhost", "127.0.0.1", "::1"} {
		if !strings.Contains(value, required) {
			t.Fatalf("expected NO_PROXY to include %q, got %q", required, value)
		}
	}
}

func TestBuildEnvClearsProxyVarsWhenProxyDisabled(t *testing.T) {
	env := buildEnv([]string{
		"HTTP_PROXY=http://old-proxy:8080",
		"HTTPS_PROXY=http://old-proxy:8080",
		"ALL_PROXY=http://old-proxy:8080",
		"http_proxy=http://old-proxy:8080",
		"https_proxy=http://old-proxy:8080",
		"all_proxy=http://old-proxy:8080",
	}, nil, "", "")

	for _, item := range env {
		if strings.HasPrefix(item, "HTTP_PROXY=") || strings.HasPrefix(item, "HTTPS_PROXY=") || strings.HasPrefix(item, "ALL_PROXY=") ||
			strings.HasPrefix(item, "http_proxy=") || strings.HasPrefix(item, "https_proxy=") || strings.HasPrefix(item, "all_proxy=") {
			t.Fatalf("expected proxy vars to be cleared when proxy disabled, got %q", item)
		}
	}
}

func TestRunnerExecutesSimpleCommand(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "echo.txt")
	r := NewRunner(RunConfig{
		Command: []string{"sh", "-c", "echo hello > \"$OUT\""},
		Env: map[string]string{
			"OUT": outFile,
		},
	})

	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("runner start failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "hello" {
		t.Fatalf("unexpected output: %q", string(data))
	}
}

func TestRunnerForwardsExitCode(t *testing.T) {
	r := NewRunner(RunConfig{Command: []string{"sh", "-c", "exit 7"}})
	err := r.Start(context.Background())
	if err == nil {
		t.Fatal("expected non-zero exit error")
	}

	exitErr := &ExitError{}
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T (%v)", err, err)
	}
	if exitErr.Code != 7 {
		t.Fatalf("expected exit code 7, got %d", exitErr.Code)
	}
}

func TestRunnerInjectsCACertEnvVarsForAllRuntimes(t *testing.T) {
	certDir := t.TempDir()
	certPath := filepath.Join(certDir, "ca.crt")
	if err := os.WriteFile(certPath, []byte("fake-cert"), 0o644); err != nil {
		t.Fatal(err)
	}

	original := buildMergedCABundle
	buildMergedCABundle = func(string) (string, func(), error) {
		return certPath, func() {}, nil
	}
	defer func() { buildMergedCABundle = original }()

	outFile := filepath.Join(t.TempDir(), "caenv.txt")
	r := NewRunner(RunConfig{
		Command:    []string{"sh", "-c", `printf '%s\n%s\n%s\n%s\n%s\n%s' "$SSL_CERT_FILE" "$NODE_EXTRA_CA_CERTS" "$REQUESTS_CA_BUNDLE" "$CURL_CA_BUNDLE" "$GIT_SSL_CAINFO" "$GRPC_DEFAULT_SSL_ROOTS_FILE_PATH" > "$OUT"`},
		ProxyAddr:  "127.0.0.1:19090",
		CACertPath: certPath,
		Env:        map[string]string{"OUT": outFile},
	})

	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("runner start failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	wantKeys := []string{
		"SSL_CERT_FILE",
		"NODE_EXTRA_CA_CERTS",
		"REQUESTS_CA_BUNDLE",
		"CURL_CA_BUNDLE",
		"GIT_SSL_CAINFO",
		"GRPC_DEFAULT_SSL_ROOTS_FILE_PATH",
	}
	if len(lines) != len(wantKeys) {
		t.Fatalf("expected %d lines, got %d: %q", len(wantKeys), len(lines), string(data))
	}
	for i, key := range wantKeys {
		if strings.TrimSpace(lines[i]) != certPath {
			t.Errorf("%s = %q, want %q", key, strings.TrimSpace(lines[i]), certPath)
		}
	}
}

func TestRunnerKeepsSystemCABundleEnvWhenMergedBundleUnavailable(t *testing.T) {
	certDir := t.TempDir()
	certPath := filepath.Join(certDir, "ca.crt")
	if err := os.WriteFile(certPath, []byte("fake-cert"), 0o644); err != nil {
		t.Fatal(err)
	}

	original := buildMergedCABundle
	buildMergedCABundle = func(string) (string, func(), error) {
		return "", nil, errors.New("merge failed")
	}
	defer func() { buildMergedCABundle = original }()

	outFile := filepath.Join(t.TempDir(), "caenv-fallback.txt")
	r := NewRunner(RunConfig{
		Command:    []string{"sh", "-c", `printf '%s\n%s\n%s\n%s\n%s\n%s' "$SSL_CERT_FILE" "$NODE_EXTRA_CA_CERTS" "$REQUESTS_CA_BUNDLE" "$CURL_CA_BUNDLE" "$GIT_SSL_CAINFO" "$GRPC_DEFAULT_SSL_ROOTS_FILE_PATH" > "$OUT"`},
		ProxyAddr:  "127.0.0.1:19091",
		CACertPath: certPath,
		Env: map[string]string{
			"OUT":                              outFile,
			"SSL_CERT_FILE":                    "/system/ssl.pem",
			"REQUESTS_CA_BUNDLE":               "/system/requests.pem",
			"CURL_CA_BUNDLE":                   "/system/curl.pem",
			"GIT_SSL_CAINFO":                   "/system/git.pem",
			"GRPC_DEFAULT_SSL_ROOTS_FILE_PATH": "/system/grpc.pem",
		},
	})

	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("runner start failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d: %q", len(lines), string(data))
	}
	if lines[0] != "/system/ssl.pem" {
		t.Fatalf("SSL_CERT_FILE should keep system bundle, got %q", lines[0])
	}
	if lines[1] != certPath {
		t.Fatalf("NODE_EXTRA_CA_CERTS should point to proxy cert, got %q", lines[1])
	}
	if lines[2] != "/system/requests.pem" || lines[3] != "/system/curl.pem" || lines[4] != "/system/git.pem" || lines[5] != "/system/grpc.pem" {
		t.Fatalf("expected system CA vars preserved, got %q", string(data))
	}
}

func TestStripEnvKeysRemovesAllDuplicateCAOverrides(t *testing.T) {
	env := []string{
		"A=1",
		"REQUESTS_CA_BUNDLE=/old/a.pem",
		"B=2",
		"REQUESTS_CA_BUNDLE=/old/b.pem",
		"CURL_CA_BUNDLE=/old/c.pem",
		"C=3",
	}

	got := stripEnvKeys(env, "REQUESTS_CA_BUNDLE", "CURL_CA_BUNDLE")
	for _, item := range got {
		if strings.HasPrefix(item, "REQUESTS_CA_BUNDLE=") || strings.HasPrefix(item, "CURL_CA_BUNDLE=") {
			t.Fatalf("unexpected CA override remains in env: %q", item)
		}
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 non-CA entries, got %d: %v", len(got), got)
	}
}

func TestRunnerCancellationKillsSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group signaling differs on windows")
	}

	r := NewRunner(RunConfig{Command: []string{"sh", "-c", "sleep 30"}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Start(ctx)
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runner did not stop after cancellation")
	}
}

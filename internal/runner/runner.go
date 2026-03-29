package runner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap/internal/proxy"
)

type RunConfig struct {
	Command    []string
	ProxyAddr  string
	AgentToken string
	CACertPath string
	Env        map[string]string
	WorkDir    string
	Timeout    time.Duration
}

type Runner struct {
	config RunConfig
	proxy  *proxy.ProxyServer

	mu          sync.Mutex
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	proxyCancel context.CancelFunc
}

type ExitError struct {
	Code int
}

const processWaitTimeout = 5 * time.Second

var buildMergedCABundle = buildMergedCABundleFile

func (e *ExitError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.Code)
}

func NewRunner(cfg RunConfig) *Runner {
	return &Runner{config: cfg}
}

func (r *Runner) WithProxyServer(p *proxy.ProxyServer) *Runner {
	r.proxy = p
	return r
}

func (r *Runner) Start(ctx context.Context) error {
	if len(r.config.Command) == 0 {
		return errors.New("command is required")
	}

	if r.config.Timeout > 0 {
		var cancelTimeout context.CancelFunc
		ctx, cancelTimeout = context.WithTimeout(ctx, r.config.Timeout)
		defer cancelTimeout()
	}

	runCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()
	defer cancel()

	proxyAddr, proxyStop, err := r.startProxyIfNeeded(runCtx)
	if err != nil {
		return err
	}
	defer proxyStop()

	cmd := exec.Command(r.config.Command[0], r.config.Command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = strings.TrimSpace(r.config.WorkDir)
	env := buildEnv(os.Environ(), r.config.Env, proxyAddr, r.config.AgentToken)
	if proxyAddr != "" && strings.TrimSpace(r.config.CACertPath) != "" {
		certPath := strings.TrimSpace(r.config.CACertPath)
		env = stripEnvKeys(env, "NODE_EXTRA_CA_CERTS")
		env = append(env, "NODE_EXTRA_CA_CERTS="+certPath)

		mergedBundlePath, cleanupMergedBundle, _ := buildMergedCABundle(certPath)
		if cleanupMergedBundle != nil {
			defer cleanupMergedBundle()
		}
		if mergedBundlePath != "" {
			env = stripEnvKeys(env,
				"SSL_CERT_FILE",
				"REQUESTS_CA_BUNDLE",
				"CURL_CA_BUNDLE",
				"GIT_SSL_CAINFO",
				"GRPC_DEFAULT_SSL_ROOTS_FILE_PATH",
			)
			for _, key := range []string{
				"SSL_CERT_FILE",
				"REQUESTS_CA_BUNDLE",
				"CURL_CA_BUNDLE",
				"GIT_SSL_CAINFO",
				"GRPC_DEFAULT_SSL_ROOTS_FILE_PATH",
			} {
				env = append(env, key+"="+mergedBundlePath)
			}
		}
	}
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	r.mu.Lock()
	r.cmd = cmd
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.cmd = nil
		r.mu.Unlock()
	}()

	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	waitErrCh := make(chan error, 1)
	go func() {
		waitErrCh <- cmd.Wait()
	}()

	select {
	case err := <-waitErrCh:
		return mapExitError(err)
	case <-runCtx.Done():
		if err := killProcessTree(cmd.Process); err != nil {
			return fmt.Errorf("terminate process tree: %w", err)
		}
		waitTimer := time.NewTimer(processWaitTimeout)
		defer waitTimer.Stop()
		select {
		case waitErr := <-waitErrCh:
			var exitErr *exec.ExitError
			if waitErr == nil || errors.As(waitErr, &exitErr) {
				return runCtx.Err()
			}
			return mapExitError(waitErr)
		case <-waitTimer.C:
			return fmt.Errorf("process did not exit within %s after cancellation: %w", processWaitTimeout, runCtx.Err())
		}
	}
}

func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
	if r.proxyCancel != nil {
		r.proxyCancel()
	}
	return nil
}

func (r *Runner) startProxyIfNeeded(ctx context.Context) (string, func(), error) {
	proxyAddr := strings.TrimSpace(r.config.ProxyAddr)
	if r.proxy == nil {
		return normalizeProxyAddr(proxyAddr), func() {}, nil
	}

	proxyCtx, proxyCancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.proxyCancel = proxyCancel
	r.mu.Unlock()

	proxyErrCh := make(chan error, 1)
	go func() {
		proxyErrCh <- r.proxy.Start(proxyCtx)
	}()

	resolvedAddr, err := waitProxyAddr(ctx, r.proxy, proxyErrCh)
	if err != nil {
		proxyCancel()
		return "", func() {}, err
	}
	if proxyAddr == "" {
		proxyAddr = resolvedAddr
	}

	stop := func() {
		proxyCancel()
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = r.proxy.Stop(stopCtx)
		select {
		case <-time.After(2 * time.Second):
		case <-proxyErrCh:
		}
	}

	return normalizeProxyAddr(proxyAddr), stop, nil
}

func waitProxyAddr(ctx context.Context, p *proxy.ProxyServer, proxyErrCh <-chan error) (string, error) {
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()

	for {
		if p.Ready() {
			return p.Addr(), nil
		}

		select {
		case err := <-proxyErrCh:
			if err != nil {
				return "", fmt.Errorf("proxy startup failed: %w", err)
			}
			return "", errors.New("proxy exited unexpectedly during startup")
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "", errors.New("proxy start timeout")
		case <-tick.C:
		}
	}
}

func buildEnv(base []string, extra map[string]string, proxyAddr, agentToken string) []string {
	envMap := make(map[string]string, len(base)+len(extra)+8)
	for _, item := range base {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	sensitiveKeys := []string{
		"KIMBAP_MASTER_KEY_HEX",
		"KIMBAP_AGENT_TOKEN",
		"KIMBAP_DEV",
	}
	for _, key := range sensitiveKeys {
		delete(envMap, key)
	}
	for _, key := range []string{
		"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
		"http_proxy", "https_proxy", "all_proxy",
	} {
		delete(envMap, key)
	}
	if proxyAddr != "" {
		delete(envMap, "SSL_CERT_FILE")
	}

	if proxyAddr != "" {
		proxyURL := proxyAddr
		if strings.TrimSpace(agentToken) != "" {
			proxyURL = embedProxyAuth(proxyAddr, agentToken)
		}
		envMap["HTTP_PROXY"] = proxyURL
		envMap["HTTPS_PROXY"] = proxyURL
		envMap["ALL_PROXY"] = proxyURL
		envMap["http_proxy"] = proxyURL
		envMap["https_proxy"] = proxyURL
		envMap["all_proxy"] = proxyURL
		existing := envMap["NO_PROXY"]
		if existing == "" {
			existing = envMap["no_proxy"]
		}
		existing = ensureNoProxyLoopback(existing)
		envMap["NO_PROXY"] = existing
		envMap["no_proxy"] = existing
	}
	if strings.TrimSpace(agentToken) != "" {
		envMap["KIMBAP_AGENT_TOKEN"] = strings.TrimSpace(agentToken)
	}

	maps.Copy(envMap, extra)

	out := make([]string, 0, len(envMap))
	for k, v := range envMap {
		out = append(out, k+"="+v)
	}
	return out
}

func ensureNoProxyLoopback(existing string) string {
	parts := strings.Split(existing, ",")
	out := make([]string, 0, len(parts)+3)
	seen := make(map[string]struct{}, len(parts)+3)

	for _, part := range parts {
		entry := strings.TrimSpace(part)
		if entry == "" {
			continue
		}
		key := strings.ToLower(entry)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}

	for _, loopback := range []string{"localhost", "127.0.0.1", "::1"} {
		if _, ok := seen[loopback]; ok {
			continue
		}
		seen[loopback] = struct{}{}
		out = append(out, loopback)
	}

	return strings.Join(out, ",")
}

func stripEnvKeys(env []string, keys ...string) []string {
	if len(env) == 0 || len(keys) == 0 {
		return env
	}
	deny := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		deny[k] = struct{}{}
	}
	out := make([]string, 0, len(env))
	for _, item := range env {
		name := item
		if idx := strings.IndexByte(item, '='); idx >= 0 {
			name = item[:idx]
		}
		if _, found := deny[name]; found {
			continue
		}
		out = append(out, item)
	}
	return out
}

func embedProxyAuth(addr, token string) string {
	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		return addr
	}
	u.User = url.UserPassword("kimbap", strings.TrimSpace(token))
	return u.String()
}

func normalizeProxyAddr(addr string) string {
	a := strings.TrimSpace(addr)
	if a == "" {
		return ""
	}
	if strings.HasPrefix(a, ":") {
		a = "127.0.0.1" + a
	}
	if lower := strings.ToLower(a); strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return a
	}
	return "http://" + a
}

func mapExitError(err error) error {
	if err == nil {
		return nil
	}
	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				return &ExitError{Code: 128 + int(ws.Signal())}
			}
			return &ExitError{Code: ws.ExitStatus()}
		}
		return &ExitError{Code: 1}
	}
	return err
}

func killProcessTree(p *os.Process) error {
	if p == nil {
		return nil
	}
	pid := p.Pid
	if pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	if err := p.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

func buildMergedCABundleFile(proxyCertPath string) (string, func(), error) {
	proxyCertPath = strings.TrimSpace(proxyCertPath)
	if proxyCertPath == "" {
		return "", nil, nil
	}
	proxyCert, err := os.ReadFile(proxyCertPath)
	if err != nil {
		return "", nil, nil
	}

	for _, rootBundlePath := range candidateSystemCABundlePaths() {
		rootBundle, readErr := os.ReadFile(rootBundlePath)
		if readErr != nil {
			continue
		}
		merged := make([]byte, 0, len(proxyCert)+len(rootBundle)+2)
		merged = append(merged, proxyCert...)
		if len(merged) > 0 && merged[len(merged)-1] != '\n' {
			merged = append(merged, '\n')
		}
		merged = append(merged, rootBundle...)

		tmp, createErr := os.CreateTemp("", "kimbap-ca-bundle-*.pem")
		if createErr != nil {
			return "", nil, nil
		}
		path := tmp.Name()
		if _, writeErr := tmp.Write(merged); writeErr != nil {
			_ = tmp.Close()
			_ = os.Remove(path)
			return "", nil, nil
		}
		if closeErr := tmp.Close(); closeErr != nil {
			_ = os.Remove(path)
			return "", nil, nil
		}
		_ = os.Chmod(path, 0o600)
		return path, func() { _ = os.Remove(path) }, nil
	}

	return "", nil, nil
}

func candidateSystemCABundlePaths() []string {
	return []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/cert.pem",
		filepath.Join("/private", "etc", "ssl", "cert.pem"),
	}
}

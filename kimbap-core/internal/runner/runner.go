package runner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap-core/internal/proxy"
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
		env = append(env, "SSL_CERT_FILE="+strings.TrimSpace(r.config.CACertPath))
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
		_ = killProcessTree(cmd.Process)
		<-waitErrCh
		return runCtx.Err()
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
	if r.cmd != nil && r.cmd.Process != nil {
		return killProcessTree(r.cmd.Process)
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

	resolvedAddr, err := waitProxyAddr(ctx, r.proxy)
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

func waitProxyAddr(ctx context.Context, p *proxy.ProxyServer) (string, error) {
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()

	for {
		addr := strings.TrimSpace(p.Addr())
		if addr != "" && !strings.HasSuffix(addr, ":0") {
			return addr, nil
		}

		select {
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

	if proxyAddr != "" {
		envMap["HTTP_PROXY"] = proxyAddr
		envMap["HTTPS_PROXY"] = proxyAddr
		loopback := "localhost,127.0.0.1,::1"
		if existing, ok := envMap["NO_PROXY"]; ok && existing != "" {
			if !strings.Contains(existing, "localhost") {
				envMap["NO_PROXY"] = existing + "," + loopback
			}
		} else {
			envMap["NO_PROXY"] = loopback
		}
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

func normalizeProxyAddr(addr string) string {
	a := strings.TrimSpace(addr)
	if a == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(a), "http://") || strings.HasPrefix(strings.ToLower(a), "https://") {
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

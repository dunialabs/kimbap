package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap-core/internal/classifier"
	"github.com/dunialabs/kimbap-core/internal/proxy"
	"github.com/dunialabs/kimbap-core/internal/runner"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	var (
		proxyEnabled bool
		proxyAddr    string
		proxyPort    int
		caDir        string
		agentToken   string
	)

	cmd := &cobra.Command{
		Use:   "run -- <command> [args...]",
		Short: "Run a subprocess with kimbap runtime wiring",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			command := normalizeSubprocessCommand(args)
			if len(command) == 0 {
				return fmt.Errorf("missing command after --")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			resolvedProxyAddr := strings.TrimSpace(proxyAddr)
			if resolvedProxyAddr == "" {
				resolvedProxyAddr = strings.TrimSpace(cfg.ProxyAddr)
			}
			if proxyPort > 0 {
				resolvedProxyAddr = withPort(resolvedProxyAddr, proxyPort)
			}

			resolvedCADir := strings.TrimSpace(caDir)
			if resolvedCADir == "" {
				resolvedCADir = filepath.Join(cfg.DataDir, "ca")
			}

			caCertPath := ""
			if proxyEnabled {
				caCertPath = filepath.Join(resolvedCADir, "ca.crt")
			}

			runCfg := runner.RunConfig{
				Command:    command,
				ProxyAddr:  resolvedProxyAddr,
				CACertPath: caCertPath,
			}
			if proxyEnabled && strings.TrimSpace(agentToken) == "" {
				// Generate ephemeral session token scoped to this run
				sessionToken := fmt.Sprintf("kses_%d", time.Now().UnixNano())
				runCfg.AgentToken = sessionToken
				_, _ = fmt.Fprintf(os.Stderr, "info: using ephemeral session token for this run\n")
			} else {
				runCfg.AgentToken = strings.TrimSpace(agentToken)
			}
			r := runner.NewRunner(runCfg)

			if proxyEnabled {
				rt, err := buildRuntimeFromConfig(cfg)
				if err != nil {
					return fmt.Errorf("build runtime: %w", err)
				}

				ca, err := proxy.GenerateCA(resolvedCADir)
				if err != nil {
					return fmt.Errorf("prepare proxy CA: %w", err)
				}

				c := classifier.NewClassifier()
				installer := installerFromConfig(cfg)
				installedSkills, err := installer.List()
				if err != nil {
					return fmt.Errorf("load installed skills: %w", err)
				}
				for i := range installedSkills {
					if err := c.AddRulesFromSkill(&installedSkills[i].Manifest); err != nil {
						return fmt.Errorf("register skill %q: %w", installedSkills[i].Manifest.Name, err)
					}
				}

				p := proxy.NewProxyServer(resolvedProxyAddr, ca,
					proxy.WithClassifier(c),
					proxy.WithRuntime(rt),
					proxy.WithAgentToken(runCfg.AgentToken),
					proxy.WithTenantID(defaultTenantID()),
				)
				r.WithProxyServer(p)
			}

			runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			err = r.Start(runCtx)
			if err == nil || errors.Is(err, context.Canceled) {
				return nil
			}

			exitErr := &runner.ExitError{}
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.Code)
			}

			return err
		},
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().BoolVar(&proxyEnabled, "proxy", true, "start embedded proxy and inject HTTP(S)_PROXY")
	cmd.Flags().StringVar(&proxyAddr, "proxy-addr", "", "proxy listen address (default from config)")
	cmd.Flags().IntVar(&proxyPort, "proxy-port", 0, "proxy listen port override")
	cmd.Flags().StringVar(&caDir, "ca-dir", "", "proxy CA directory (default <data-dir>/ca)")
	// Agent token is no longer auto-inherited from parent environment. Pass explicitly via --agent-token for security.
	cmd.Flags().StringVar(&agentToken, "agent-token", "", "agent token to inject as KIMBAP_AGENT_TOKEN")

	return cmd
}

func normalizeSubprocessCommand(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	if args[0] == "--" {
		return args[1:]
	}
	return args
}

func withPort(addr string, port int) string {
	if port <= 0 {
		return strings.TrimSpace(addr)
	}
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return fmt.Sprintf(":%d", port)
	}

	host := ""
	if strings.Contains(trimmed, ":") {
		if strings.HasPrefix(trimmed, ":") {
			host = ""
		} else if h, _, err := net.SplitHostPort(trimmed); err == nil {
			host = h
		} else {
			host = strings.TrimSpace(trimmed)
		}
	} else {
		host = strings.TrimSpace(trimmed)
	}

	if host == "" {
		return fmt.Sprintf(":%d", port)
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}

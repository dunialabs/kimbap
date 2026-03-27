package main

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap/internal/classifier"
	"github.com/dunialabs/kimbap/internal/proxy"
	"github.com/spf13/cobra"
)

func newProxyCommand() *cobra.Command {
	var (
		addr       string
		port       int
		caDir      string
		agentToken string
	)

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Start local HTTP/HTTPS MITM proxy",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			listenAddr := strings.TrimSpace(addr)
			if listenAddr == "" {
				listenAddr = strings.TrimSpace(cfg.ProxyAddr)
			}
			if port > 0 {
				listenAddr = withPort(listenAddr, port)
			}

			caPath := strings.TrimSpace(caDir)
			if caPath == "" {
				caPath = filepath.Join(cfg.DataDir, "ca")
			}
			ca, err := proxy.GenerateCA(caPath)
			if err != nil {
				return fmt.Errorf("prepare proxy CA: %w", err)
			}

			c := classifier.NewClassifier()
			installer := installerFromConfig(cfg)
			installedServices, err := installer.List()
			if err != nil {
				return fmt.Errorf("load installed services: %w", err)
			}
			for i := range installedServices {
				if err := c.AddRulesFromService(&installedServices[i].Manifest); err != nil {
					return fmt.Errorf("register service %q: %w", installedServices[i].Manifest.Name, err)
				}
			}

			proxyOpts := []proxy.ProxyOption{
				proxy.WithClassifier(c),
				proxy.WithAgentToken(strings.TrimSpace(agentToken)),
			}
			rt, buildErr := buildRuntimeFromConfig(cfg)
			if buildErr != nil {
				return fmt.Errorf("runtime required for proxy mode (policy/credential enforcement): %w", buildErr)
			}
			proxyOpts = append(proxyOpts, proxy.WithRuntime(rt))

			server := proxy.NewProxyServer(listenAddr, ca, proxyOpts...)

			runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() {
				errCh <- server.Start(runCtx)
			}()

			select {
			case err := <-errCh:
				if err == nil {
					return nil
				}
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return err
			case <-runCtx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Stop(shutdownCtx)
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "", "proxy listen address (default from config)")
	cmd.Flags().IntVar(&port, "port", 0, "proxy listen port override")
	cmd.Flags().StringVar(&caDir, "ca-dir", "", "CA directory (default <data-dir>/ca)")
	cmd.Flags().StringVar(&agentToken, "agent-token", "", "agent token associated with proxy traffic")

	return cmd
}

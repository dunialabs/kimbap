package service

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/rs/zerolog"
)

type ContainerStatus string

const (
	ContainerRunning  ContainerStatus = "running"
	ContainerStopped  ContainerStatus = "stopped"
	ContainerNotExist ContainerStatus = "not_exist"
)

type CloudflaredDockerService struct {
	log zerolog.Logger
}

func NewCloudflaredDockerService() *CloudflaredDockerService {
	return &CloudflaredDockerService{log: logger.CreateLogger("CloudflaredDockerService")}
}

func (s *CloudflaredDockerService) CheckDockerAvailable() bool {
	_, err := runCommand(5*time.Second, "docker", "info")
	return err == nil
}

func (s *CloudflaredDockerService) EnsureImageExists() error {
	if config.CloudflaredCfg().InDocker {
		return nil
	}
	out, err := runCommand(10*time.Second, "docker", "images", config.CloudflaredCfg().Image, "--format", "{{.Repository}}")
	if err == nil && strings.Contains(out, "cloudflare/cloudflared") {
		return nil
	}
	_, err = runCommand(5*time.Minute, "docker", "pull", config.CloudflaredCfg().Image)
	return err
}

func (s *CloudflaredDockerService) GetContainerStatus() ContainerStatus {
	name := config.CloudflaredCfg().ContainerName
	running, _ := runCommand(10*time.Second, "docker", "ps", "--filter", "name="+name, "--filter", "status=running", "--format", "{{.Names}}")
	if strings.TrimSpace(running) == name {
		return ContainerRunning
	}
	all, _ := runCommand(10*time.Second, "docker", "ps", "-a", "--filter", "name="+name, "--format", "{{.Names}}")
	if strings.TrimSpace(all) == name {
		return ContainerStopped
	}
	return ContainerNotExist
}

func (s *CloudflaredDockerService) StartContainer() error {
	name := config.CloudflaredCfg().ContainerName
	status := s.GetContainerStatus()
	switch status {
	case ContainerRunning:
		return nil
	case ContainerStopped:
		_, err := runCommand(30*time.Second, "docker", "start", name)
		return err
	default:
		_, err := runCommand(60*time.Second, "docker", "compose", "up", "-d", "cloudflared")
		return err
	}
}

func (s *CloudflaredDockerService) StopContainer() error {
	if s.GetContainerStatus() != ContainerRunning {
		return nil
	}
	name := config.CloudflaredCfg().ContainerName
	_, err := runCommand(30*time.Second, "docker", "stop", name)
	if err != nil && (strings.Contains(err.Error(), "signal: interrupt") || strings.Contains(strings.ToUpper(err.Error()), "SIGINT") || strings.Contains(err.Error(), "exit status 130")) {
		s.log.Warn().Err(err).Str("containerName", name).Msg("docker stop interrupted by SIGINT, treating as success")
		return nil
	}
	if err != nil && (strings.Contains(err.Error(), "No such container") || strings.Contains(err.Error(), "is not running")) {
		return nil
	}
	return err
}

func (s *CloudflaredDockerService) RestartContainer() error {
	name := config.CloudflaredCfg().ContainerName
	status := s.GetContainerStatus()
	switch status {
	case ContainerRunning:
		_, err := runCommand(30*time.Second, "docker", "restart", name)
		return err
	case ContainerStopped:
		_, err := runCommand(30*time.Second, "docker", "start", name)
		return err
	default:
		_, err := runCommand(60*time.Second, "docker", "compose", "up", "-d", "cloudflared")
		return err
	}
}

func (s *CloudflaredDockerService) DeleteContainer() error {
	if s.GetContainerStatus() == ContainerNotExist {
		return nil
	}
	if err := s.StopContainer(); err != nil {
		return err
	}
	name := config.CloudflaredCfg().ContainerName
	_, err := runCommand(10*time.Second, "docker", "rm", name)
	if err != nil && strings.Contains(err.Error(), "No such container") {
		return nil
	}
	return err
}

func (s *CloudflaredDockerService) GenerateConfigYAML(tunnelID string, subdomain string) string {
	credentialsPath := fmt.Sprintf("%s/%s.json", config.CloudflaredCfg().ContainerConfigDir, tunnelID)
	serviceURL := config.CloudflaredCfg().KimbapCoreServiceURL
	return fmt.Sprintf("tunnel: %s\ncredentials-file: %s\n\ningress:\n  - hostname: %s\n    service: %s\n    originRequest:\n      noTLSVerify: false\n      httpHostHeader: %s\n      connectTimeout: 30s\n      keepAliveTimeout: 90s\n      keepAliveConnections: 100\n      disableChunkedEncoding: false\n      originServerName: %s\n      proxyType: \"\"\n      tcpKeepAlive: 30s\n      noHappyEyeballs: false\n      http2Origin: false\n  - service: http_status:404", tunnelID, credentialsPath, subdomain, serviceURL, subdomain, subdomain)
}

func runCommand(timeout time.Duration, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case err := <-done:
		if err != nil {
			stderr := strings.TrimSpace(errOut.String())
			if stderr == "" {
				stderr = err.Error()
			}
			return "", fmt.Errorf("%s %v: %s", name, args, stderr)
		}
		return out.String(), nil
	case <-time.After(timeout):
		if cmd.Process == nil {
			return "", fmt.Errorf("command timed out before process start: %s %v", name, args)
		}
		if err := cmd.Process.Kill(); err != nil {
			return "", fmt.Errorf("command timed out and kill failed: %s %v: %w", name, args, err)
		}
		return "", fmt.Errorf("command timed out: %s %v", name, args)
	}
}

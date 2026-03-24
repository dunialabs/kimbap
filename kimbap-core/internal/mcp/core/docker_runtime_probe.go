package core

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

const DockerSocketPath = "/var/run/docker.sock"
const dockerProbeTimeout = 5 * time.Second

func AssertDockerRuntimeAvailable(ctxLabel string, target string) error {
	if _, err := os.Stat(DockerSocketPath); err != nil {
		return fmt.Errorf("[%s] Docker socket not found (%s) for %s", ctxLabel, DockerSocketPath, target)
	}

	conn, err := net.DialTimeout("unix", DockerSocketPath, 2*time.Second)
	if err != nil {
		return fmt.Errorf("[%s] Docker socket is not accessible (%s) for %s: %s", ctxLabel, DockerSocketPath, target, err.Error())
	}
	conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), dockerProbeTimeout)
	defer cancel()

	cliCheck := exec.CommandContext(ctx, "docker", "--version")
	if _, cliErr := cliCheck.CombinedOutput(); cliErr != nil {
		return fmt.Errorf("[%s] docker CLI is not available for %s: %s", ctxLabel, target, cliErr.Error())
	}

	daemonCtx, daemonCancel := context.WithTimeout(context.Background(), dockerProbeTimeout)
	defer daemonCancel()

	daemonCheck := exec.CommandContext(daemonCtx, "docker", "version", "--format", "{{.Server.Version}}")
	if _, daemonErr := daemonCheck.CombinedOutput(); daemonErr != nil {
		return fmt.Errorf("[%s] Docker daemon check failed for %s: %s", ctxLabel, target, daemonErr.Error())
	}

	return nil
}

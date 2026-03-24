package core

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/types"
)

type CustomStdioLaunchPlan struct {
	LaunchConfig   map[string]any
	RunnerMetadata *CustomStdioRunnerMetadata
}

type RunnerExecutionTrace struct {
	StderrWriter *StderrTailWriter
	cmd          *exec.Cmd
}

func (t *RunnerExecutionTrace) ExitCode() *int {
	if t == nil || t.cmd == nil {
		return nil
	}
	ps := t.cmd.ProcessState
	if ps == nil {
		return nil
	}
	code := ps.ExitCode()
	return &code
}

func (t *RunnerExecutionTrace) StderrTail() string {
	if t == nil || t.StderrWriter == nil {
		return ""
	}
	return t.StderrWriter.Tail()
}

type RunnerFailureDetails struct {
	Category      string
	Reason        string
	Message       string
	StderrSummary string
}

type customStdioRunnerService struct{}

var CustomStdioRunnerServiceInstance = &customStdioRunnerService{}

func (s *customStdioRunnerService) ResolveLaunchPlan(serverEntity database.Server, launchConfig map[string]any) (*CustomStdioLaunchPlan, error) {
	if serverEntity.Category != types.ServerCategoryCustomStdio {
		return &CustomStdioLaunchPlan{LaunchConfig: launchConfig}, nil
	}

	if os.Getenv("KIMBAP_CORE_IN_DOCKER") != "true" {
		return &CustomStdioLaunchPlan{LaunchConfig: launchConfig}, nil
	}

	command, _ := launchConfig["command"].(string)
	if IsExplicitDockerCommand(command) {
		return &CustomStdioLaunchPlan{LaunchConfig: launchConfig}, nil
	}

	if err := AssertDockerRuntimeAvailable("CustomStdioRunner", fmt.Sprintf("server %s", serverEntity.ServerID)); err != nil {
		return nil, err
	}

	plan, err := BuildCustomStdioRunnerLaunchPlan(launchConfig, CustomStdioRunnerImage)
	if err != nil {
		return nil, err
	}
	return &CustomStdioLaunchPlan{
		LaunchConfig:   plan.LaunchConfig,
		RunnerMetadata: &plan.Metadata,
	}, nil
}

func (s *customStdioRunnerService) AttachExecutionTrace(cmd *exec.Cmd) *RunnerExecutionTrace {
	if cmd == nil {
		return nil
	}

	stderrWriter := NewStderrTailWriter(defaultStderrTailMaxLength, os.Stderr)
	cmd.Stderr = stderrWriter

	return &RunnerExecutionTrace{
		StderrWriter: stderrWriter,
		cmd:          cmd,
	}
}

func (s *customStdioRunnerService) BuildFailureDetails(
	serverID string,
	metadata *CustomStdioRunnerMetadata,
	trace *RunnerExecutionTrace,
	err error,
) *RunnerFailureDetails {
	if metadata == nil || trace == nil {
		return nil
	}

	errorText := ""
	if err != nil {
		errorText = err.Error()
	}

	stderrSummary := trace.StderrWriter.SummarizeTail(300)
	exitCode := trace.ExitCode()
	classification := ClassifyCustomStdioRunnerFailure(exitCode, trace.StderrTail(), errorText)

	if classification.Category == RunnerUnknownFailure &&
		exitCode == nil &&
		stderrSummary == "" &&
		errorText == "" {
		return nil
	}

	exitCodeText := "unknown"
	if exitCode != nil {
		exitCodeText = fmt.Sprintf("%d", *exitCode)
	}

	prefix := "CustomStdio runner failed"
	switch classification.Category {
	case RunnerStartupFailure:
		prefix = "CustomStdio runner startup failed"
	case RunnerCommandFailure:
		prefix = "CustomStdio runner command failed"
	}

	message := fmt.Sprintf(
		"%s (serverId=%s, originalCommand=%s, runnerImage=%s, exitCode=%s, reason=%s)",
		prefix, serverID, metadata.OriginalCommand, metadata.RunnerImage,
		exitCodeText, classification.Reason,
	)

	if stderrSummary != "" {
		message += " stderr=" + stderrSummary
	}

	return &RunnerFailureDetails{
		Category:      string(classification.Category),
		Reason:        classification.Reason,
		Message:       message,
		StderrSummary: stderrSummary,
	}
}

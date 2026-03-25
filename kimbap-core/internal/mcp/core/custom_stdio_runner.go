package core

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

const CustomStdioRunnerImage = "kimbapio/mcp-runner:latest"
const customStdioRunnerCacheVolume = "kimbap-mcp-runner-cache:/home/runner/.cache"
const defaultStderrTailMaxLength = 4096

var runnerStartupStderrKeywords = []string{
	"cannot connect to the docker daemon",
	"is the docker daemon running",
	"permission denied while trying to connect",
	"error response from daemon",
	"dial unix",
	"docker.sock",
	"pull access denied",
	"no such image",
	"manifest for",
	"failed to solve",
}

var runnerCommandStderrKeywords = []string{
	"executable file not found",
	"command not found",
	"exec format error",
	"permission denied",
	"no such file or directory",
	"failed to exec",
}

type RunnerFailureCategory string

const (
	RunnerStartupFailure RunnerFailureCategory = "runner_startup_failure"
	RunnerCommandFailure RunnerFailureCategory = "runner_command_failure"
	RunnerUnknownFailure RunnerFailureCategory = "runner_unknown_failure"
)

type CustomStdioRunnerMetadata struct {
	RunnerImage     string
	OriginalCommand string
}

type CustomStdioRunnerLaunchPlan struct {
	LaunchConfig map[string]any
	Metadata     CustomStdioRunnerMetadata
}

type RunnerFailureClassification struct {
	Category RunnerFailureCategory
	Reason   string
}

func normalizeCommandForMatch(command string) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return ""
	}
	unquoted := strings.Trim(trimmed, `"'`)
	normalizedPath := strings.ReplaceAll(unquoted, `\`, "/")
	baseName := path.Base(normalizedPath)
	baseName = strings.TrimSuffix(strings.ToLower(baseName), ".exe")
	return baseName
}

func IsExplicitDockerCommand(command string) bool {
	return normalizeCommandForMatch(command) == "docker"
}

func BuildCustomStdioRunnerLaunchPlan(originalLaunchConfig map[string]any, runnerImage string) (CustomStdioRunnerLaunchPlan, error) {
	originalCommand, _ := originalLaunchConfig["command"].(string)
	if strings.TrimSpace(originalCommand) == "" {
		return CustomStdioRunnerLaunchPlan{}, fmt.Errorf("CustomStdio launchConfig.command must be a non-empty string")
	}

	if runnerImage == "" {
		runnerImage = CustomStdioRunnerImage
	}
	originalArgs := toStringSlice(originalLaunchConfig["args"])

	dockerArgs := []string{
		"run", "-i", "--rm", "--init",
		"-v", customStdioRunnerCacheVolume,
	}

	if cwd, ok := originalLaunchConfig["cwd"].(string); ok && strings.TrimSpace(cwd) != "" {
		containerCwd := strings.TrimSpace(cwd)
		if !filepath.IsAbs(containerCwd) {
			if os.Getenv("KIMBAP_CORE_IN_DOCKER") == "true" {
				return CustomStdioRunnerLaunchPlan{}, fmt.Errorf("CustomStdio launchConfig.cwd must be an absolute path when running inside Docker")
			}
			if absCwd, err := filepath.Abs(containerCwd); err == nil {
				containerCwd = absCwd
			}
		}
		stat, err := os.Stat(containerCwd)
		if err != nil {
			return CustomStdioRunnerLaunchPlan{}, fmt.Errorf("CustomStdio launchConfig.cwd is invalid: %w", err)
		}
		if !stat.IsDir() {
			return CustomStdioRunnerLaunchPlan{}, fmt.Errorf("CustomStdio launchConfig.cwd is invalid: not a directory")
		}
		dockerArgs = append(dockerArgs, "-v", containerCwd+":"+containerCwd, "-w", containerCwd)
	}

	for _, entry := range toStringEnvEntries(originalLaunchConfig["env"]) {
		dockerArgs = append(dockerArgs, "-e", entry.Key+"="+entry.Value)
	}

	dockerArgs = append(dockerArgs, runnerImage, originalCommand)
	dockerArgs = append(dockerArgs, originalArgs...)

	launchConfig := make(map[string]any)
	for k, v := range originalLaunchConfig {
		launchConfig[k] = v
	}
	launchConfig["command"] = "docker"
	launchConfig["args"] = toAnySlice(dockerArgs)
	delete(launchConfig, "cwd")

	return CustomStdioRunnerLaunchPlan{
		LaunchConfig: launchConfig,
		Metadata: CustomStdioRunnerMetadata{
			RunnerImage:     runnerImage,
			OriginalCommand: originalCommand,
		},
	}, nil
}

func ClassifyCustomStdioRunnerFailure(exitCode *int, stderrTail string, errorText string) RunnerFailureClassification {
	combinedText := strings.ToLower(stderrTail + "\n" + errorText)

	startupKeyword := findKeyword(combinedText, runnerStartupStderrKeywords)
	if (exitCode != nil && *exitCode == 125) || startupKeyword != "" {
		reason := startupKeyword
		if reason == "" && exitCode != nil {
			reason = fmt.Sprintf("docker exit code %d", *exitCode)
		}
		return RunnerFailureClassification{Category: RunnerStartupFailure, Reason: reason}
	}

	commandKeyword := findKeyword(combinedText, runnerCommandStderrKeywords)
	if (exitCode != nil && (*exitCode == 126 || *exitCode == 127)) || commandKeyword != "" {
		reason := commandKeyword
		if reason == "" && exitCode != nil {
			reason = fmt.Sprintf("docker exit code %d", *exitCode)
		}
		return RunnerFailureClassification{Category: RunnerCommandFailure, Reason: reason}
	}

	if exitCode != nil && *exitCode != 0 {
		return RunnerFailureClassification{
			Category: RunnerCommandFailure,
			Reason:   fmt.Sprintf("docker exit code %d", *exitCode),
		}
	}

	if strings.Contains(combinedText, "spawn") && strings.Contains(combinedText, "enoent") {
		return RunnerFailureClassification{Category: RunnerStartupFailure, Reason: "spawn enoent"}
	}

	return RunnerFailureClassification{Category: RunnerUnknownFailure, Reason: "unable to classify"}
}

func findKeyword(source string, keywords []string) string {
	for _, keyword := range keywords {
		if strings.Contains(source, keyword) {
			return keyword
		}
	}
	return ""
}

type envEntry struct {
	Key   string
	Value string
}

func toStringSlice(value any) []string {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if item == nil {
			continue
		}
		out = append(out, fmt.Sprintf("%v", item))
	}
	return out
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func toStringEnvEntries(envValue any) []envEntry {
	m, ok := envValue.(map[string]any)
	if !ok {
		return nil
	}
	entries := make([]envEntry, 0, len(m))
	for key, value := range m {
		if strings.TrimSpace(key) == "" || value == nil {
			continue
		}
		var valStr string
		if s, ok := value.(string); ok {
			valStr = s
		} else {
			valStr = fmt.Sprintf("%v", value)
		}
		entries = append(entries, envEntry{Key: key, Value: valStr})
	}
	return entries
}

type StderrTailWriter struct {
	mu        sync.Mutex
	buf       []byte
	maxLength int
	passThru  io.Writer
}

func NewStderrTailWriter(maxLength int, passThru io.Writer) *StderrTailWriter {
	if maxLength <= 0 {
		maxLength = defaultStderrTailMaxLength
	}
	if passThru == nil {
		passThru = os.Stderr
	}
	return &StderrTailWriter{
		maxLength: maxLength,
		passThru:  passThru,
	}
}

func (w *StderrTailWriter) Write(p []byte) (int, error) {
	n, err := w.passThru.Write(p)

	w.mu.Lock()
	w.buf = append(w.buf, p[:n]...)
	if len(w.buf) > w.maxLength {
		w.buf = w.buf[len(w.buf)-w.maxLength:]
	}
	w.mu.Unlock()

	return n, err
}

func (w *StderrTailWriter) Tail() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.buf)
}

func (w *StderrTailWriter) SummarizeTail(maxLength int) string {
	tail := w.Tail()
	normalized := strings.Join(strings.Fields(tail), " ")
	if len(normalized) <= maxLength {
		return normalized
	}
	return normalized[len(normalized)-maxLength:]
}

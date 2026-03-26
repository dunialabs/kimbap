package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GlobalAgentConfig defines the user-level paths for a specific AI agent.
type GlobalAgentConfig struct {
	Kind            AgentKind
	AgentSkillsDir  string // absolute path to global skills directory
	InstructionFile string // absolute path to native instruction file (empty = no inject)
	DetectDir       string // absolute path to check for agent presence
}

const (
	markerStart = "<!-- KIMBAP:START -->"
	markerEnd   = "<!-- KIMBAP:END -->"
)

// globalInstructionBlock is the content injected between markers.
// Intentionally short, invariant, and does NOT claim the project uses kimbap.
const globalInstructionBlock = `## Kimbap

Kimbap may be available as the approved path for external service access.
Before using direct third-party credentials or APIs, check whether Kimbap
is configured by running ` + "`kimbap actions list --format json`" + `.
If Kimbap is unavailable or no relevant actions exist, follow the project's
normal conventions.

Never ask for, print, or store raw API keys, passwords, or tokens when
Kimbap is available.
`

// GlobalSetupResult holds the outcome of a global setup operation.
type GlobalSetupResult struct {
	Agent             AgentKind `json:"agent"`
	AgentSkillWritten bool      `json:"agent_skill_written"`
	AgentSkillPath    string    `json:"agent_skill_path"`
	InstructionFile   string    `json:"instruction_file,omitempty"`
	InjectWritten     bool      `json:"inject_written"`
	Skipped           bool      `json:"skipped"`
	Error             string    `json:"error,omitempty"`
}

// GlobalSetupOptions controls global setup behavior.
type GlobalSetupOptions struct {
	Agents []AgentKind
	Force  bool
	DryRun bool
}

// GlobalTeardownResult holds the outcome of a global teardown operation.
type GlobalTeardownResult struct {
	Agent             AgentKind `json:"agent"`
	AgentSkillRemoved bool      `json:"agent_skill_removed"`
	InjectRemoved     bool      `json:"inject_removed"`
	Error             string    `json:"error,omitempty"`
}

// resolveGlobalConfigs builds the global agent config for each known agent,
// resolving home directory and XDG paths for the current platform.
func resolveGlobalConfigs() (map[AgentKind]GlobalAgentConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	xdgConfig := xdgConfigHome(home)

	configs := map[AgentKind]GlobalAgentConfig{
		AgentClaudeCode: {
			Kind:            AgentClaudeCode,
			AgentSkillsDir:  filepath.Join(home, ".claude", "skills"),
			InstructionFile: filepath.Join(home, ".claude", "CLAUDE.md"),
			DetectDir:       filepath.Join(home, ".claude"),
		},
		AgentOpenCode: {
			Kind:            AgentOpenCode,
			AgentSkillsDir:  filepath.Join(xdgConfig, "opencode", "skills"),
			InstructionFile: filepath.Join(xdgConfig, "opencode", "AGENTS.md"),
			DetectDir:       filepath.Join(xdgConfig, "opencode"),
		},
		AgentCodex: {
			Kind:            AgentCodex,
			AgentSkillsDir:  filepath.Join(home, ".agents", "skills"),
			InstructionFile: filepath.Join(home, ".codex", "AGENTS.md"),
			DetectDir:       filepath.Join(home, ".codex"),
		},
		AgentCursor: {
			Kind:           AgentCursor,
			AgentSkillsDir: filepath.Join(home, ".cursor", "skills"),
			DetectDir:      filepath.Join(home, ".cursor"),
		},
	}

	return configs, nil
}

// xdgConfigHome returns XDG_CONFIG_HOME or the platform default.
func xdgConfigHome(home string) string {
	if env := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); env != "" {
		return env
	}
	return filepath.Join(home, ".config")
}

// GlobalDetectAgents discovers which AI agents are installed by checking
// whether their user-level config directories exist.
func GlobalDetectAgents() ([]GlobalAgentConfig, error) {
	configs, err := resolveGlobalConfigs()
	if err != nil {
		return nil, err
	}

	var detected []GlobalAgentConfig
	for _, kind := range []AgentKind{AgentClaudeCode, AgentOpenCode, AgentCodex, AgentCursor} {
		cfg := configs[kind]
		if info, statErr := os.Stat(cfg.DetectDir); statErr == nil && info.IsDir() {
			detected = append(detected, cfg)
		}
	}

	return detected, nil
}

// GlobalSetup writes the kimbap meta-service to global agent skill directories
// and injects a discovery block into native instruction files.
func GlobalSetup(metaSkillContent string, opts GlobalSetupOptions) ([]GlobalSetupResult, error) {
	allConfigs, err := resolveGlobalConfigs()
	if err != nil {
		return nil, err
	}

	targets, err := selectGlobalTargets(allConfigs, opts.Agents)
	if err != nil {
		return nil, err
	}

	results := make([]GlobalSetupResult, 0, len(targets))
	for _, cfg := range targets {
		result := globalSetupOne(cfg, metaSkillContent, opts)
		results = append(results, result)
	}

	return results, nil
}

func globalSetupOne(cfg GlobalAgentConfig, metaSkillContent string, opts GlobalSetupOptions) GlobalSetupResult {
	result := GlobalSetupResult{
		Agent:          cfg.Kind,
		AgentSkillPath: filepath.Join(cfg.AgentSkillsDir, "kimbap", "SKILL.md"),
	}

	skillDir := filepath.Join(cfg.AgentSkillsDir, "kimbap")
	skillPath := filepath.Join(skillDir, "SKILL.md")
	needsWrite, checkErr := fileNeedsWrite(skillPath, metaSkillContent, opts.Force)
	if checkErr != nil {
		result.Error = fmt.Sprintf("check service file: %v", checkErr)
		return result
	}

	if needsWrite {
		if opts.DryRun {
			result.AgentSkillWritten = true
		} else {
			if err := os.MkdirAll(skillDir, 0o755); err != nil {
				result.Error = fmt.Sprintf("create service directory: %v", err)
				return result
			}
			if err := atomicWriteFile(skillPath, metaSkillContent); err != nil {
				result.Error = fmt.Sprintf("write service file: %v", err)
				return result
			}
			result.AgentSkillWritten = true
		}
	}

	if cfg.InstructionFile == "" {
		result.Skipped = !result.AgentSkillWritten
		return result
	}
	result.InstructionFile = cfg.InstructionFile

	injected, err := injectMarkerBlock(cfg.InstructionFile, opts.Force, opts.DryRun)
	if err != nil {
		result.Error = fmt.Sprintf("inject instructions: %v", err)
		return result
	}
	result.InjectWritten = injected
	result.Skipped = !result.AgentSkillWritten && !result.InjectWritten

	return result
}

// GlobalTeardown removes kimbap global service directories and marker blocks
// from instruction files.
func GlobalTeardown(opts GlobalSetupOptions) ([]GlobalTeardownResult, error) {
	allConfigs, err := resolveGlobalConfigs()
	if err != nil {
		return nil, err
	}

	targets, err := selectTeardownTargets(allConfigs, opts.Agents)
	if err != nil {
		return nil, err
	}

	results := make([]GlobalTeardownResult, 0, len(targets))
	for _, cfg := range targets {
		result := globalTeardownOne(cfg, opts.DryRun)
		results = append(results, result)
	}

	return results, nil
}

func globalTeardownOne(cfg GlobalAgentConfig, dryRun bool) GlobalTeardownResult {
	result := GlobalTeardownResult{Agent: cfg.Kind}

	skillDir := filepath.Join(cfg.AgentSkillsDir, "kimbap")
	if _, err := os.Stat(skillDir); err == nil {
		if dryRun {
			result.AgentSkillRemoved = true
		} else {
			if err := os.RemoveAll(skillDir); err != nil {
				result.Error = fmt.Sprintf("remove agent skill dir: %v", err)
				return result
			}
			result.AgentSkillRemoved = true
		}
	} else if !os.IsNotExist(err) {
		result.Error = fmt.Sprintf("stat agent skill dir: %v", err)
		return result
	}

	if cfg.InstructionFile == "" {
		return result
	}

	removed, err := removeMarkerBlock(cfg.InstructionFile, dryRun)
	if err != nil {
		result.Error = fmt.Sprintf("remove instruction block: %v", err)
		return result
	}
	result.InjectRemoved = removed

	return result
}

func resolveRequestedAgents(allConfigs map[AgentKind]GlobalAgentConfig, requested []AgentKind) ([]GlobalAgentConfig, error) {
	out := make([]GlobalAgentConfig, 0, len(requested))
	for _, kind := range requested {
		cfg, ok := allConfigs[kind]
		if !ok {
			return nil, fmt.Errorf("unknown agent kind: %q", kind)
		}
		out = append(out, cfg)
	}
	return out, nil
}

// selectGlobalTargets picks which agents to operate on for setup. If agents is
// nil, auto-detect by checking whether the agent's config directory exists.
func selectGlobalTargets(allConfigs map[AgentKind]GlobalAgentConfig, requested []AgentKind) ([]GlobalAgentConfig, error) {
	if len(requested) > 0 {
		return resolveRequestedAgents(allConfigs, requested)
	}

	var detected []GlobalAgentConfig
	for _, kind := range []AgentKind{AgentClaudeCode, AgentOpenCode, AgentCodex, AgentCursor} {
		cfg := allConfigs[kind]
		if info, err := os.Stat(cfg.DetectDir); err == nil && info.IsDir() {
			detected = append(detected, cfg)
		}
	}
	return detected, nil
}

// selectTeardownTargets picks which agents to operate on for teardown. If
// agents is nil, auto-detect by checking whether the agent's config directory
// OR any kimbap-managed artifact exists — so teardown works even after the
// host agent has been uninstalled.
func selectTeardownTargets(allConfigs map[AgentKind]GlobalAgentConfig, requested []AgentKind) ([]GlobalAgentConfig, error) {
	if len(requested) > 0 {
		return resolveRequestedAgents(allConfigs, requested)
	}

	var targets []GlobalAgentConfig
	for _, kind := range []AgentKind{AgentClaudeCode, AgentOpenCode, AgentCodex, AgentCursor} {
		cfg := allConfigs[kind]
		if hasKimbapArtifacts(cfg) {
			targets = append(targets, cfg)
		}
	}
	return targets, nil
}

// hasKimbapArtifacts reports whether kimbap has left any managed files for cfg.
func hasKimbapArtifacts(cfg GlobalAgentConfig) bool {
	if info, err := os.Stat(cfg.DetectDir); err == nil && info.IsDir() {
		return true
	}
	if _, err := os.Stat(filepath.Join(cfg.AgentSkillsDir, "kimbap")); err == nil {
		return true
	}
	if cfg.InstructionFile != "" {
		if data, err := os.ReadFile(cfg.InstructionFile); err == nil {
			s := string(data)
			si := strings.Index(s, markerStart)
			ei := strings.Index(s, markerEnd)
			if si >= 0 && ei >= 0 && si < ei {
				return true
			}
		}
	}
	return false
}

// injectMarkerBlock inserts or replaces the kimbap block in the given
// instruction file. Returns true if the file was modified.
func injectMarkerBlock(path string, force bool, dryRun bool) (bool, error) {
	block := markerStart + "\n" + globalInstructionBlock + markerEnd + "\n"

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read %q: %w", path, err)
	}

	content := string(existing)
	startIdx := strings.Index(content, markerStart)
	endIdx := strings.Index(content, markerEnd)

	hasStart := startIdx >= 0
	hasEnd := endIdx >= 0

	if hasStart != hasEnd {
		found, missing := "start", "end"
		if !hasStart {
			found, missing = "end", "start"
		}
		return false, fmt.Errorf("malformed kimbap markers in %q: found %s marker but not %s", path, found, missing)
	}

	if hasStart && hasEnd {
		if startIdx >= endIdx {
			return false, fmt.Errorf("malformed kimbap markers in %q: end marker appears before start marker", path)
		}
		endIdx += len(markerEnd)
		if endIdx < len(content) && content[endIdx] == '\n' {
			endIdx++
		}
		existingBlock := content[startIdx:endIdx]
		if existingBlock == block && !force {
			return false, nil
		}
		newContent := content[:startIdx] + block + content[endIdx:]
		if dryRun {
			return true, nil
		}
		return true, atomicWriteFile(path, newContent)
	}

	var newContent string
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		newContent = content + "\n\n" + block
	} else if len(content) > 0 {
		newContent = content + "\n" + block
	} else {
		newContent = block
	}

	if dryRun {
		return true, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create directory for %q: %w", path, err)
	}
	return true, atomicWriteFile(path, newContent)
}

// removeMarkerBlock removes the kimbap marker block from the given file.
// Returns true if the file was modified.
func removeMarkerBlock(path string, dryRun bool) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read %q: %w", path, err)
	}

	content := string(existing)
	startIdx := strings.Index(content, markerStart)
	endIdx := strings.Index(content, markerEnd)

	hasStart := startIdx >= 0
	hasEnd := endIdx >= 0

	if !hasStart && !hasEnd {
		return false, nil
	}
	if hasStart != hasEnd {
		found, missing := "start", "end"
		if !hasStart {
			found, missing = "end", "start"
		}
		return false, fmt.Errorf("malformed kimbap markers in %q: found %s marker but not %s", path, found, missing)
	}
	if startIdx >= endIdx {
		return false, fmt.Errorf("malformed kimbap markers in %q: end marker appears before start marker", path)
	}

	endIdx += len(markerEnd)
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	before := content[:startIdx]
	after := content[endIdx:]

	trimmedBefore := strings.TrimRight(before, "\n")
	trimmedAfter := strings.TrimLeft(after, "\n")

	var newContent string
	switch {
	case trimmedBefore == "" && trimmedAfter == "":
		newContent = ""
	case trimmedBefore == "":
		newContent = trimmedAfter
		if !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
	case trimmedAfter == "":
		newContent = trimmedBefore + "\n"
	default:
		sep := "\n"
		if strings.HasSuffix(before, "\n\n") {
			sep = "\n\n"
		}
		newContent = trimmedBefore + sep + trimmedAfter
		if !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
	}

	if dryRun {
		return true, nil
	}
	return true, atomicWriteFile(path, newContent)
}

// atomicWriteFile writes content to a temp file then renames it into place.
func atomicWriteFile(path string, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".kimbap-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func atomicWriteDir(targetDir string, files map[string]string) error {
	if len(files) == 0 {
		return nil
	}
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	tmpDir, err := os.MkdirTemp(parentDir, ".kimbap-pack-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(tmpDir)
		}
	}()
	names := make([]string, 0, len(files))
	for name := range files {
		if err := validatePackFileName(name); err != nil {
			return err
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(files[name]), 0o644); err != nil {
			return fmt.Errorf("write file %q to temp dir: %w", name, err)
		}
	}
	oldDir := targetDir + ".old"
	hasOld := false
	if _, statErr := os.Stat(targetDir); statErr == nil {
		if err := os.RemoveAll(oldDir); err != nil {
			return fmt.Errorf("remove stale backup dir: %w", err)
		}
		if err := os.Rename(targetDir, oldDir); err != nil {
			return fmt.Errorf("backup existing dir: %w", err)
		}
		hasOld = true
	}
	if err := os.Rename(tmpDir, targetDir); err != nil {
		if hasOld {
			_ = os.Rename(oldDir, targetDir)
		}
		return fmt.Errorf("rename temp to target: %w", err)
	}
	if hasOld {
		os.RemoveAll(oldDir)
	}
	success = true
	return nil
}

func validatePackFileName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("pack filename must be non-empty")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("pack filename %q must be relative", name)
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("pack filename %q must not escape pack directory", name)
	}
	if strings.ContainsAny(clean, `/\`) {
		return fmt.Errorf("pack filename %q must not contain path separators", name)
	}
	return nil
}

// GlobalStatus returns the global setup state for all known agents.
func GlobalStatus() ([]GlobalStatusResult, error) {
	configs, err := resolveGlobalConfigs()
	if err != nil {
		return nil, err
	}

	var results []GlobalStatusResult
	for _, kind := range []AgentKind{AgentClaudeCode, AgentOpenCode, AgentCodex, AgentCursor} {
		cfg := configs[kind]
		result := GlobalStatusResult{
			Agent:          kind,
			AgentSkillsDir: filepath.Join(cfg.AgentSkillsDir, "kimbap"),
		}

		if info, statErr := os.Stat(cfg.DetectDir); statErr == nil && info.IsDir() {
			result.Detected = true
		}

		skillPath := filepath.Join(cfg.AgentSkillsDir, "kimbap", "SKILL.md")
		if _, statErr := os.Stat(skillPath); statErr == nil {
			result.AgentSkillPresent = true
		}

		if cfg.InstructionFile != "" {
			result.InstructionFile = cfg.InstructionFile
			if data, readErr := os.ReadFile(cfg.InstructionFile); readErr == nil {
				s := string(data)
				si := strings.Index(s, markerStart)
				ei := strings.Index(s, markerEnd)
				result.InjectPresent = si >= 0 && ei >= 0 && si < ei
			}
		}

		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Agent < results[j].Agent
	})

	return results, nil
}

// GlobalStatusResult holds the global install state for one agent.
type GlobalStatusResult struct {
	Agent             AgentKind `json:"agent"`
	Detected          bool      `json:"detected"`
	AgentSkillsDir    string    `json:"agent_skills_dir"`
	AgentSkillPresent bool      `json:"agent_skill_present"`
	InstructionFile   string    `json:"instruction_file,omitempty"`
	InjectPresent     bool      `json:"inject_present"`
}

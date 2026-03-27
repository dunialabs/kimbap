package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/services"
)

type AgentKind string

const (
	AgentClaudeCode AgentKind = "claude-code"
	AgentOpenCode   AgentKind = "opencode"
	AgentCodex      AgentKind = "codex"
	AgentCursor     AgentKind = "cursor"
	AgentGeneric    AgentKind = "generic"
)

var KnownAgents = []AgentKind{AgentClaudeCode, AgentOpenCode, AgentCodex, AgentCursor, AgentGeneric}

type AgentConfig struct {
	Kind           AgentKind
	AgentSkillsDir string
	RulesFile      string
	DetectPaths    []string
}

var agentConfigs = map[AgentKind]AgentConfig{
	AgentClaudeCode: {
		Kind:           AgentClaudeCode,
		AgentSkillsDir: ".claude/skills",
		RulesFile:      ".claude/KIMBAP_OPERATING_RULES.md",
		DetectPaths:    []string{".claude", ".claude/settings.json", ".claude/CLAUDE.md"},
	},
	AgentOpenCode: {
		Kind:           AgentOpenCode,
		AgentSkillsDir: ".opencode/skills",
		RulesFile:      ".opencode/KIMBAP_OPERATING_RULES.md",
		DetectPaths:    []string{".opencode", "opencode.json", ".opencode/config.json"},
	},
	AgentCodex: {
		Kind:           AgentCodex,
		AgentSkillsDir: ".codex/skills",
		RulesFile:      ".codex/KIMBAP_OPERATING_RULES.md",
		DetectPaths:    []string{".codex", "AGENTS.md"},
	},
	AgentCursor: {
		Kind:           AgentCursor,
		AgentSkillsDir: ".cursor/skills",
		RulesFile:      ".cursor/KIMBAP_OPERATING_RULES.md",
		DetectPaths:    []string{".cursor", ".cursor/rules"},
	},
	AgentGeneric: {
		Kind:           AgentGeneric,
		AgentSkillsDir: ".agents/skills",
		RulesFile:      ".agents/KIMBAP_OPERATING_RULES.md",
		DetectPaths:    []string{".agents"},
	},
}

func GetAgentConfig(kind AgentKind) (AgentConfig, bool) {
	cfg, ok := agentConfigs[kind]
	return cfg, ok
}

type DetectedAgent struct {
	Kind       AgentKind
	Config     AgentConfig
	ProjectDir string
}

func DetectAgents(projectDir string) []DetectedAgent {
	baseDir := normalizeProjectDir(projectDir)
	detected := make([]DetectedAgent, 0)

	for _, kind := range KnownAgents {
		cfg, ok := agentConfigs[kind]
		if !ok {
			continue
		}
		if !isAgentDetected(baseDir, cfg) {
			continue
		}
		detected = append(detected, DetectedAgent{Kind: kind, Config: cfg, ProjectDir: baseDir})
	}

	return detected
}

type SyncResult struct {
	Agent          AgentKind `json:"agent"`
	AgentSkillsDir string    `json:"agent_skills_dir"`
	Written        []string  `json:"written"`
	Skipped        []string  `json:"skipped"`
	Failed         []string  `json:"failed"`
	Pruned         []string  `json:"pruned,omitempty"`
	RulesWritten   bool      `json:"rules_written"`
	Errors         []string  `json:"errors,omitempty"`
}

type SyncOptions struct {
	ProjectDir string
	Agents     []AgentKind
	Force      bool
	DryRun     bool
	SkipRules  bool
	SkipPrune  bool
}

type ServiceInstaller interface {
	List() ([]InstalledService, error)
}

type InstalledService struct {
	Name    string
	Content string
}

type InstalledServicePack struct {
	Name         string
	AgentSkillMD string
	PackFiles    map[string]string
}

type PackServiceInstaller interface {
	ServiceInstaller
	ListPacks() ([]InstalledServicePack, error)
}

func SyncServices(installer ServiceInstaller, rulesContent string, opts SyncOptions) ([]SyncResult, error) {
	if installer == nil {
		return nil, fmt.Errorf("installer is nil")
	}

	projectDir := normalizeProjectDir(opts.ProjectDir)
	if info, err := os.Stat(projectDir); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("project path %q is not a directory", projectDir)
	}

	installedServices, err := installer.List()
	if err != nil {
		return nil, fmt.Errorf("list installed services: %w", err)
	}

	var installedPacks []InstalledServicePack
	if packInstaller, ok := installer.(PackServiceInstaller); ok {
		if packs, pErr := packInstaller.ListPacks(); pErr == nil {
			if len(packs) > 0 && len(packs) == len(installedServices) {
				packByName := make(map[string]InstalledServicePack, len(packs))
				for _, p := range packs {
					if p.Name == "" {
						packByName = nil
						break
					}
					if _, exists := packByName[p.Name]; exists {
						packByName = nil
						break
					}
					if p.AgentSkillMD == "" && len(p.PackFiles) == 0 {
						packByName = nil
						break
					}
					packByName[p.Name] = p
				}
				if packByName != nil {
					completeCoverage := true
					for _, s := range installedServices {
						if _, ok := packByName[s.Name]; !ok {
							completeCoverage = false
							break
						}
					}
					if completeCoverage {
						installedPacks = packs
					}
				}
			}
		}
	}

	agentsToProcess := selectedAgents(projectDir, opts.Agents)
	results := make([]SyncResult, 0, len(agentsToProcess))

	for _, selected := range agentsToProcess {
		agentSkillsDir := ""
		if selected.cfg.AgentSkillsDir != "" {
			agentSkillsDir = filepath.Join(projectDir, selected.cfg.AgentSkillsDir)
		}
		result := SyncResult{
			Agent:          selected.kind,
			AgentSkillsDir: agentSkillsDir,
			Written:        make([]string, 0),
			Skipped:        make([]string, 0),
			Failed:         make([]string, 0),
			Errors:         make([]string, 0),
		}

		if selected.err != nil {
			result.Errors = append(result.Errors, selected.err.Error())
			results = append(results, result)
			continue
		}

		if installedPacks != nil {
			for _, pack := range installedPacks {
				if err := services.ValidateServiceName(pack.Name); err != nil {
					result.Failed = append(result.Failed, pack.Name)
					result.Errors = append(result.Errors, fmt.Sprintf("service %q: %v", pack.Name, err))
					continue
				}
				packDir := filepath.Join(projectDir, selected.cfg.AgentSkillsDir, pack.Name)
				allFiles := make(map[string]string, len(pack.PackFiles)+1)
				if pack.AgentSkillMD != "" {
					allFiles["SKILL.md"] = pack.AgentSkillMD
				}
				for k, v := range pack.PackFiles {
					allFiles[k] = v
				}
				needsWrite, checkErr := packNeedsWrite(packDir, allFiles, opts.Force)
				if checkErr != nil {
					result.Failed = append(result.Failed, pack.Name)
					result.Errors = append(result.Errors, fmt.Sprintf("service %q: %v", pack.Name, checkErr))
					continue
				}
				if !needsWrite {
					result.Skipped = append(result.Skipped, pack.Name)
					continue
				}
				if opts.DryRun {
					result.Written = append(result.Written, pack.Name)
					continue
				}
				if err := atomicWriteDir(packDir, allFiles); err != nil {
					result.Failed = append(result.Failed, pack.Name)
					result.Errors = append(result.Errors, fmt.Sprintf("service %q: write pack: %v", pack.Name, err))
					continue
				}
				result.Written = append(result.Written, pack.Name)
			}
			packNames := make([]InstalledService, len(installedPacks))
			for i, p := range installedPacks {
				packNames[i] = InstalledService{Name: p.Name}
			}
			if !opts.SkipPrune {
				pruned, pruneErrs := pruneStaleServices(filepath.Join(projectDir, selected.cfg.AgentSkillsDir), packNames, opts.DryRun)
				result.Pruned = pruned
				for _, e := range pruneErrs {
					result.Errors = append(result.Errors, e)
				}
			}
		} else {
			for _, installedService := range installedServices {
				if err := services.ValidateServiceName(installedService.Name); err != nil {
					result.Failed = append(result.Failed, installedService.Name)
					result.Errors = append(result.Errors, fmt.Sprintf("service %q: %v", installedService.Name, err))
					continue
				}

				agentSkillPath := filepath.Join(projectDir, selected.cfg.AgentSkillsDir, installedService.Name, "SKILL.md")
				needsWrite, checkErr := fileNeedsWrite(agentSkillPath, installedService.Content, opts.Force)
				if checkErr != nil {
					result.Failed = append(result.Failed, installedService.Name)
					result.Errors = append(result.Errors, fmt.Sprintf("service %q: %v", installedService.Name, checkErr))
					continue
				}
				if !needsWrite {
					result.Skipped = append(result.Skipped, installedService.Name)
					continue
				}

				if opts.DryRun {
					result.Written = append(result.Written, installedService.Name)
					continue
				}

				if err := os.MkdirAll(filepath.Dir(agentSkillPath), 0o755); err != nil {
					result.Failed = append(result.Failed, installedService.Name)
					result.Errors = append(result.Errors, fmt.Sprintf("service %q: create dir: %v", installedService.Name, err))
					continue
				}
				if err := os.WriteFile(agentSkillPath, []byte(installedService.Content), 0o644); err != nil {
					result.Failed = append(result.Failed, installedService.Name)
					result.Errors = append(result.Errors, fmt.Sprintf("service %q: write file: %v", installedService.Name, err))
					continue
				}

				result.Written = append(result.Written, installedService.Name)
			}

			if !opts.SkipPrune {
				pruned, pruneErrs := pruneStaleServices(
					filepath.Join(projectDir, selected.cfg.AgentSkillsDir),
					installedServices,
					opts.DryRun,
				)
				result.Pruned = pruned
				for _, e := range pruneErrs {
					result.Errors = append(result.Errors, e)
				}
			}
		}

		if !opts.SkipRules {
			rulesPath := filepath.Join(projectDir, selected.cfg.RulesFile)
			needsWrite, checkErr := fileNeedsWrite(rulesPath, rulesContent, opts.Force)
			if checkErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("rules: %v", checkErr))
			} else if needsWrite {
				if opts.DryRun {
					result.RulesWritten = true
				} else {
					if err := os.MkdirAll(filepath.Dir(rulesPath), 0o755); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("rules: create dir: %v", err))
					} else if err := os.WriteFile(rulesPath, []byte(rulesContent), 0o644); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("rules: write file: %v", err))
					} else {
						result.RulesWritten = true
					}
				}
			}
		}

		sort.Strings(result.Written)
		sort.Strings(result.Skipped)
		sort.Strings(result.Failed)
		results = append(results, result)
	}

	return results, nil
}

type StatusResult struct {
	Agent          AgentKind `json:"agent"`
	Detected       bool      `json:"detected"`
	AgentSkillsDir string    `json:"agent_skills_dir"`
	SyncedServices []string  `json:"synced_services"`
	RulesPresent   bool      `json:"rules_present"`
}

func Status(projectDir string) ([]StatusResult, error) {
	baseDir := normalizeProjectDir(projectDir)
	results := make([]StatusResult, 0, len(KnownAgents))

	for _, kind := range KnownAgents {
		cfg, ok := agentConfigs[kind]
		if !ok {
			continue
		}

		rulesPath := filepath.Join(baseDir, cfg.RulesFile)
		rulesPresent, err := pathExists(rulesPath)
		if err != nil {
			return nil, fmt.Errorf("check rules file for %q: %w", kind, err)
		}

		syncedServices, err := listSyncedServices(filepath.Join(baseDir, cfg.AgentSkillsDir))
		if err != nil {
			return nil, fmt.Errorf("list synced services for %q: %w", kind, err)
		}

		detected, detectErr := isAgentDetectedDetailed(baseDir, cfg)
		if detectErr != nil {
			return nil, fmt.Errorf("detect agent %q: %w", kind, detectErr)
		}

		results = append(results, StatusResult{
			Agent:          kind,
			Detected:       detected,
			AgentSkillsDir: filepath.Join(baseDir, cfg.AgentSkillsDir),
			SyncedServices: syncedServices,
			RulesPresent:   rulesPresent,
		})
	}

	return results, nil
}

type selectedAgent struct {
	kind AgentKind
	cfg  AgentConfig
	err  error
}

func selectedAgents(projectDir string, requested []AgentKind) []selectedAgent {
	if len(requested) > 0 {
		out := make([]selectedAgent, 0, len(requested))
		for _, kind := range requested {
			cfg, ok := agentConfigs[kind]
			if !ok {
				out = append(out, selectedAgent{kind: kind, err: fmt.Errorf("unknown agent kind: %q", kind)})
				continue
			}
			out = append(out, selectedAgent{kind: kind, cfg: cfg})
		}
		return out
	}

	detected := DetectAgents(projectDir)
	out := make([]selectedAgent, 0, len(detected))
	for _, agent := range detected {
		out = append(out, selectedAgent{kind: agent.Kind, cfg: agent.Config})
	}
	return out
}

func normalizeProjectDir(projectDir string) string {
	if strings.TrimSpace(projectDir) == "" {
		return "."
	}
	return projectDir
}

func isAgentDetected(projectDir string, cfg AgentConfig) bool {
	detected, _ := isAgentDetectedDetailed(projectDir, cfg)
	return detected
}

func isAgentDetectedDetailed(projectDir string, cfg AgentConfig) (bool, error) {
	var firstErr error
	for _, detectPath := range cfg.DetectPaths {
		exists, err := pathExists(filepath.Join(projectDir, detectPath))
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("check detect path %q: %w", detectPath, err)
			}
			continue
		}
		if exists {
			return true, nil
		}
	}
	return false, firstErr
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func fileNeedsWrite(path string, content string, force bool) (bool, error) {
	if force {
		return true, nil
	}

	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("read existing file %q: %w", path, err)
	}

	if string(existing) == content {
		return false, nil
	}
	return true, nil
}

func packNeedsWrite(packDir string, packFiles map[string]string, force bool) (bool, error) {
	if force {
		return true, nil
	}

	entries, err := os.ReadDir(packDir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("read existing pack dir %q: %w", packDir, err)
	}

	allowed := make(map[string]struct{}, len(packFiles))
	for name := range packFiles {
		allowed[name] = struct{}{}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return true, nil
		}
		if _, ok := allowed[entry.Name()]; !ok {
			return true, nil
		}
	}

	for name, content := range packFiles {
		filePath := filepath.Join(packDir, name)
		existing, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return true, nil
			}
			return false, fmt.Errorf("read existing pack file %q: %w", name, err)
		}
		if string(existing) != content {
			return true, nil
		}
	}
	return false, nil
}

func listSyncedServices(agentSkillsDir string) ([]string, error) {
	entries, err := os.ReadDir(agentSkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	out := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "kimbap" {
			continue
		}
		agentSkillFile := filepath.Join(agentSkillsDir, entry.Name(), "SKILL.md")
		exists, err := pathExists(agentSkillFile)
		if err != nil {
			return nil, err
		}
		if exists {
			out = append(out, entry.Name())
		}
	}

	sort.Strings(out)
	return out, nil
}

func pruneStaleServices(agentSkillsDir string, installed []InstalledService, dryRun bool) ([]string, []string) {
	entries, err := os.ReadDir(agentSkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{fmt.Sprintf("read agent skills dir: %v", err)}
	}

	active := map[string]bool{"kimbap": true}
	for _, s := range installed {
		active[s.Name] = true
	}

	var pruned []string
	var errs []string
	for _, entry := range entries {
		if !entry.IsDir() || active[entry.Name()] {
			continue
		}
		agentSkillFile := filepath.Join(agentSkillsDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(agentSkillFile); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errs = append(errs, fmt.Sprintf("check %q: %v", entry.Name(), err))
			continue
		}
		if dryRun {
			pruned = append(pruned, entry.Name())
			continue
		}
		if err := os.RemoveAll(filepath.Join(agentSkillsDir, entry.Name())); err != nil {
			errs = append(errs, fmt.Sprintf("prune %q: %v", entry.Name(), err))
		} else {
			pruned = append(pruned, entry.Name())
		}
	}
	sort.Strings(pruned)
	return pruned, errs
}

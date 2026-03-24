package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/agents"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/profiles"
	"github.com/dunialabs/kimbap-core/internal/skills"
	"github.com/spf13/cobra"
)

type agentSetupResult struct {
	SyncResults    []agents.SyncResult `json:"sync_results"`
	MetaSkillPaths []string            `json:"meta_skill_paths,omitempty"`
	AgentsFound    int                 `json:"agents_found"`
}

func newAgentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Manage AI agent skill integration",
	}
	cmd.AddCommand(newAgentsSetupCommand())
	cmd.AddCommand(newAgentsSyncCommand())
	cmd.AddCommand(newAgentsStatusCommand())
	return cmd
}

func newAgentsSetupCommand() *cobra.Command {
	var (
		dir      string
		agentRaw string
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up agent skills and operating rules",
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := runAgentsSync(dir, agentRaw, false, dryRun)
			if err != nil {
				return err
			}
			return printOutput(result)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "target project directory")
	cmd.Flags().StringVar(&agentRaw, "agent", "", "comma-separated agent kinds")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned changes without writing files")

	return cmd
}

func newAgentsSyncCommand() *cobra.Command {
	var (
		dir      string
		agentRaw string
		force    bool
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync installed skills to detected agent directories",
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := runAgentsSync(dir, agentRaw, force, dryRun)
			if err != nil {
				return err
			}
			return printOutput(result)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "target project directory")
	cmd.Flags().StringVar(&agentRaw, "agent", "", "comma-separated agent kinds")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite unchanged files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned changes without writing files")

	return cmd
}

func newAgentsStatusCommand() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show sync status for known AI agents",
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := agents.Status(dir)
			if err != nil {
				return err
			}
			return printOutput(result)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "target project directory")
	return cmd
}

func runAgentsSync(projectDir string, rawAgentKinds string, force bool, dryRun bool) (agentSetupResult, error) {
	cfg, err := loadAppConfig()
	if err != nil {
		return agentSetupResult{}, err
	}

	installedSkills, err := buildInstalledSkillsForSync(cfg)
	if err != nil {
		return agentSetupResult{}, err
	}

	rulesContent := buildRulesContent(cfg)
	syncResults, err := agents.SyncSkills(
		staticSkillInstaller{skills: installedSkills},
		rulesContent,
		agents.SyncOptions{
			ProjectDir: projectDir,
			Agents:     parseAgentKinds(rawAgentKinds),
			Force:      force,
			DryRun:     dryRun,
		},
	)
	if err != nil {
		return agentSetupResult{}, err
	}

	metaContent := skills.GenerateMetaSkillMD()
	metaPaths := make([]string, 0, len(syncResults))

	normalizedProjectDir := strings.TrimSpace(projectDir)
	if normalizedProjectDir == "" {
		normalizedProjectDir = "."
	}

	var syncErrs []string
	for _, result := range syncResults {
		if len(result.Errors) > 0 || len(result.Failed) > 0 {
			for _, e := range result.Errors {
				syncErrs = append(syncErrs, fmt.Sprintf("[%s] %s", result.Agent, e))
			}
			continue
		}

		agentCfg, ok := agents.GetAgentConfig(result.Agent)
		if !ok {
			continue
		}

		metaDir := filepath.Join(normalizedProjectDir, agentCfg.SkillsDir, "kimbap")
		metaPath := filepath.Join(metaDir, "SKILL.md")

		needsWrite := force
		if !needsWrite {
			existing, readErr := os.ReadFile(metaPath)
			switch {
			case readErr == nil:
				needsWrite = string(existing) != metaContent
			case os.IsNotExist(readErr):
				needsWrite = true
			default:
				return agentSetupResult{}, fmt.Errorf("read existing meta-skill for %q at %q: %w", result.Agent, metaPath, readErr)
			}
		}
		if !needsWrite {
			continue
		}

		if dryRun {
			metaPaths = append(metaPaths, metaPath)
			continue
		}

		if err := os.MkdirAll(metaDir, 0o755); err != nil {
			return agentSetupResult{}, fmt.Errorf("create meta-skill directory for %q: %w", result.Agent, err)
		}
		if err := os.WriteFile(metaPath, []byte(metaContent), 0o644); err != nil {
			return agentSetupResult{}, fmt.Errorf("write meta-skill for %q: %w", result.Agent, err)
		}
		metaPaths = append(metaPaths, metaPath)
	}

	if len(syncErrs) > 0 {
		return agentSetupResult{}, fmt.Errorf("sync errors: %s", strings.Join(syncErrs, "; "))
	}

	return agentSetupResult{
		SyncResults:    syncResults,
		MetaSkillPaths: metaPaths,
		AgentsFound:    len(syncResults),
	}, nil
}

func parseAgentKinds(raw string) []agents.AgentKind {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	parts := strings.Split(trimmed, ",")
	out := make([]agents.AgentKind, 0, len(parts))
	for _, part := range parts {
		kind := strings.TrimSpace(part)
		if kind == "" {
			continue
		}
		out = append(out, agents.AgentKind(kind))
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func buildInstalledSkillsForSync(cfg *config.KimbapConfig) ([]agents.InstalledSkill, error) {
	installed, err := installerFromConfig(cfg).List()
	if err != nil {
		return nil, err
	}

	out := make([]agents.InstalledSkill, 0, len(installed))
	for _, s := range installed {
		content, genErr := skills.GenerateSkillMD(&s.Manifest)
		if genErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to generate SKILL.md for %q: %v\n", s.Manifest.Name, genErr)
			continue
		}
		out = append(out, agents.InstalledSkill{Name: s.Manifest.Name, Content: content})
	}

	return out, nil
}

func buildRulesContent(cfg *config.KimbapConfig) string {
	services, svcErr := collectInstalledServicesFromConfig(cfg.Skills.Dir)
	if svcErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %v\n", svcErr)
	}

	profile, err := profiles.GenerateDynamicProfile(profiles.ProfileGeneric, services)
	if err == nil {
		return profile.Template
	}

	_, _ = fmt.Fprintf(os.Stderr, "warning: failed to generate dynamic profile: %v\n", err)
	fallback, fallbackErr := profiles.PrintProfile(profiles.ProfileGeneric)
	if fallbackErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: failed to load fallback profile: %v\n", fallbackErr)
		return ""
	}
	return fallback
}

type staticSkillInstaller struct {
	skills []agents.InstalledSkill
}

func (i staticSkillInstaller) List() ([]agents.InstalledSkill, error) {
	return i.skills, nil
}

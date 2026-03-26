package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/agents"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/profiles"
	"github.com/dunialabs/kimbap-core/internal/services"
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
		Short: "Manage AI agent service integration",
	}
	cmd.AddCommand(newAgentsSetupCommand())
	cmd.AddCommand(newAgentsSyncCommand())
	cmd.AddCommand(newAgentsStatusCommand())
	cmd.AddCommand(newAgentsUninstallGlobalCommand())
	return cmd
}

func newAgentsSetupCommand() *cobra.Command {
	var (
		agentRaw string
		force    bool
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install global kimbap discovery hints for detected AI agents",
		RunE: func(_ *cobra.Command, _ []string) error {
			metaContent := services.GenerateMetaSkillMD()
			results, err := agents.GlobalSetup(metaContent, agents.GlobalSetupOptions{
				Agents: parseAgentKinds(agentRaw),
				Force:  force,
				DryRun: dryRun,
			})
			if err != nil {
				return err
			}

			if err := printOutput(results); err != nil {
				return err
			}

			var errs []string
			for _, r := range results {
				if r.Error != "" {
					errs = append(errs, fmt.Sprintf("[%s] %s", r.Agent, r.Error))
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("setup errors: %s", strings.Join(errs, "; "))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&agentRaw, "agent", "", "comma-separated agent kinds")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite unchanged files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned changes without writing files")

	return cmd
}

func newAgentsUninstallGlobalCommand() *cobra.Command {
	var (
		agentRaw string
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "uninstall-global",
		Short: "Remove global kimbap discovery hints from AI agent configs",
		RunE: func(_ *cobra.Command, _ []string) error {
			results, err := agents.GlobalTeardown(agents.GlobalSetupOptions{
				Agents: parseAgentKinds(agentRaw),
				DryRun: dryRun,
			})
			if err != nil {
				return err
			}

			if err := printOutput(results); err != nil {
				return err
			}

			var errs []string
			for _, r := range results {
				if r.Error != "" {
					errs = append(errs, fmt.Sprintf("[%s] %s", r.Agent, r.Error))
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("uninstall errors: %s", strings.Join(errs, "; "))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&agentRaw, "agent", "", "comma-separated agent kinds")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned changes without removing files")

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
		Short: "Sync installed services to detected agent directories",
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

type combinedStatusResult struct {
	Global  []agents.GlobalStatusResult `json:"global"`
	Project []agents.StatusResult       `json:"project"`
}

func newAgentsStatusCommand() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show sync status for known AI agents",
		RunE: func(_ *cobra.Command, _ []string) error {
			globalResults, globalErr := agents.GlobalStatus()
			if globalErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "warning: global status: %v\n", globalErr)
			}

			projectResults, projectErr := agents.Status(dir)
			if projectErr != nil {
				return projectErr
			}

			return printOutput(combinedStatusResult{
				Global:  globalResults,
				Project: projectResults,
			})
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "target project directory")
	return cmd
}

func runAgentsSync(projectDir string, rawAgentKinds string, force bool, dryRun bool) (agentSetupResult, error) {
	cfg, err := loadAppConfigReadOnly()
	if err != nil {
		return agentSetupResult{}, err
	}

	installedServices, err := buildInstalledServicesForSync(cfg)
	if err != nil {
		return agentSetupResult{}, err
	}

	rulesContent := buildRulesContent(cfg)
	syncResults, err := agents.SyncServices(
		staticServiceInstaller{services: installedServices},
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

	metaContent := services.GenerateMetaSkillMD()
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
			for _, f := range result.Failed {
				syncErrs = append(syncErrs, fmt.Sprintf("[%s] failed: %s", result.Agent, f))
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

	if !dryRun && len(syncResults) > 0 {
		recordProjectSyncState(projectSyncScope(projectDir), installedServices)
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

func buildInstalledServicesForSync(cfg *config.KimbapConfig) ([]agents.InstalledService, error) {
	installed, err := installerFromConfig(cfg).List()
	if err != nil {
		return nil, err
	}

	out := make([]agents.InstalledService, 0, len(installed))
	for _, s := range installed {
		content, genErr := services.GenerateSkillMD(&s.Manifest)
		if genErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to generate SKILL.md for %q: %v\n", s.Manifest.Name, genErr)
			continue
		}
		out = append(out, agents.InstalledService{Name: s.Manifest.Name, Content: content})
	}

	return out, nil
}

func buildRulesContent(cfg *config.KimbapConfig) string {
	services, svcErr := collectInstalledServicesFromConfig(cfg.Services.Dir)
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

type staticServiceInstaller struct {
	services []agents.InstalledService
}

func (i staticServiceInstaller) List() ([]agents.InstalledService, error) {
	return i.services, nil
}

func projectSyncScope(projectDir string) string {
	normalizedProjectDir := strings.TrimSpace(projectDir)
	if normalizedProjectDir == "" {
		normalizedProjectDir = "."
	}
	absProjectDir, err := filepath.Abs(normalizedProjectDir)
	if err == nil {
		return absProjectDir
	}
	return normalizedProjectDir
}

func recordProjectSyncState(scope string, installedServices []agents.InstalledService) {
	names := make([]string, 0, len(installedServices))
	contents := make([]string, 0, len(installedServices))
	for _, s := range installedServices {
		names = append(names, s.Name)
		contents = append(contents, s.Content)
	}

	if err := agents.RecordSync(scope, names, contents); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: failed to record sync state: %v\n", err)
	}
}

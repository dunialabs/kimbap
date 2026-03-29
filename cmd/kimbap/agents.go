package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dunialabs/kimbap/internal/agents"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/profiles"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
)

type agentSetupResult struct {
	SyncResults         []agents.SyncResult `json:"sync_results"`
	MetaAgentSkillPaths []string            `json:"meta_agent_skill_paths,omitempty"`
	AgentsFound         int                 `json:"agents_found"`
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
		agentRaw     string
		force        bool
		dryRun       bool
		withProfiles bool
		profileDir   string
		dir          string
		noSync       bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install global kimbap discovery hints for detected AI agents",
		RunE: func(_ *cobra.Command, _ []string) error {
			metaContent := services.GenerateMetaAgentSkillMD()
			results, err := agents.GlobalSetup(metaContent, agents.GlobalSetupOptions{
				Agents: parseAgentKinds(agentRaw),
				Force:  force,
				DryRun: dryRun,
			})
			if err != nil {
				return err
			}

			if len(results) == 0 {
				if outputAsJSON() {
					return printOutput(results)
				}
				fmt.Println("No AI agents detected. Install Claude Code, OpenCode, Cursor, or Codex, then re-run 'kimbap agents setup'.")
				return nil
			}

			if outputAsJSON() {
				if err := printOutput(results); err != nil {
					return err
				}
			} else {
				for _, r := range results {
					if r.Error != "" {
						fmt.Printf("  ✗ %-16s %s\n", r.Agent, r.Error)
					} else if r.Skipped {
						fmt.Printf("  - %-16s skipped\n", r.Agent)
					} else {
						fmt.Printf("  ✓ %-16s skill written\n", r.Agent)
					}
				}
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

			if withProfiles {
				if err := installProfilesForAgents(results, profileDir, dryRun); err != nil {
					return err
				}
			}

			if noSync {
				return nil
			}

			trimmedDir := strings.TrimSpace(dir)
			if trimmedDir == "" {
				trimmedDir = "."
			}
			absDir, err := filepath.Abs(trimmedDir)
			if err != nil {
				return fmt.Errorf("resolve sync directory: %w", err)
			}
			if absDir == "/" {
				return fmt.Errorf("refusing to sync to root directory")
			}
			st, err := os.Stat(absDir)
			if err != nil {
				return err
			}
			if !st.IsDir() {
				return fmt.Errorf("sync target is not a directory: %s", absDir)
			}

			syncResult, err := runAgentsSync(absDir, agentRaw, "", force, dryRun)
			if err != nil {
				return err
			}

			hasWrittenServices := false
			for _, r := range syncResult.SyncResults {
				if len(r.Written) > 0 {
					hasWrittenServices = true
					break
				}
			}

			if !outputAsJSON() && hasWrittenServices && !dryRun {
				fmt.Printf("✓ Services synced to %s\n", absDir)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&agentRaw, "agent", "", "comma-separated agent kinds")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite unchanged files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned changes without writing files")
	cmd.Flags().StringVar(&dir, "dir", ".", "project directory to sync into")
	cmd.Flags().BoolVar(&noSync, "no-sync", false, "skip sync after global setup")
	cmd.Flags().BoolVar(&withProfiles, "with-profiles", false, "also install agent operating profiles into the project directory")
	cmd.Flags().StringVar(&profileDir, "profile-dir", ".", "target directory for profile installation (used with --with-profiles)")

	return cmd
}

func installProfilesForAgents(results []agents.GlobalSetupResult, targetDir string, dryRun bool) error {
	dir := strings.TrimSpace(targetDir)
	if dir == "" {
		dir = "."
	}
	for _, r := range results {
		if r.Error != "" {
			continue
		}
		profileType := profiles.ProfileType(r.Agent)
		profile, err := profiles.GetProfile(profileType)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: no profile for agent %q: %v\n", r.Agent, err)
			continue
		}
		if dryRun {
			_, _ = fmt.Fprintf(os.Stderr, "would install profile %q to %s\n", profile.Name, filepath.Join(dir, profile.InstallPath))
			continue
		}
		if err := profiles.InstallProfile(profile, dir); err != nil {
			return fmt.Errorf("install profile for %q: %w", r.Agent, err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "installed profile %q → %s\n", profile.Name, filepath.Join(dir, profile.InstallPath))
	}
	return nil
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

			if outputAsJSON() {
				if err := printOutput(results); err != nil {
					return err
				}
			} else {
				for _, r := range results {
					if r.Error != "" {
						fmt.Printf("  ✗ %-16s %s\n", r.Agent, r.Error)
					} else if !r.AgentSkillRemoved && !r.InjectRemoved {
						fmt.Printf("  - %-16s nothing to remove\n", r.Agent)
					} else {
						fmt.Printf("  ✓ %-16s removed\n", r.Agent)
					}
				}
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
		services string
		force    bool
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync installed services to detected agent directories",
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := runAgentsSync(dir, agentRaw, services, force, dryRun)
			if err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(result)
			}
			fmt.Printf("Agents found: %d\n", result.AgentsFound)
			for _, r := range result.SyncResults {
				icon := "✓"
				if len(r.Failed) > 0 {
					icon = "✗"
				}
				fmt.Printf("  %s %-16s written=%d skipped=%d failed=%d\n",
					icon, r.Agent, len(r.Written), len(r.Skipped), len(r.Failed))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "target project directory")
	cmd.Flags().StringVar(&agentRaw, "agent", "", "comma-separated agent kinds")
	cmd.Flags().StringVar(&services, "services", "", "comma-separated service names to sync")
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

			combined := combinedStatusResult{
				Global:  globalResults,
				Project: projectResults,
			}

			if outputAsJSON() {
				return printOutput(combined)
			}

			if len(combined.Global) > 0 {
				fmt.Println("Global:")
				for _, r := range combined.Global {
					icon := "✗"
					if r.Detected {
						icon = "✓"
					}
					skill := "absent"
					if r.AgentSkillPresent {
						skill = "present"
					}
					inject := "absent"
					if r.InjectPresent {
						inject = "present"
					}
					fmt.Printf("  %s %-16s skill=%-10s instruction=%s\n", icon, r.Agent, skill, inject)
				}
			}

			if len(combined.Project) > 0 {
				fmt.Printf("Project (%s):\n", dir)
				for _, r := range combined.Project {
					icon := "✗"
					if r.Detected {
						icon = "✓"
					}
					synced := "-"
					if len(r.SyncedServices) > 0 {
						synced = strings.Join(r.SyncedServices, ", ")
					}
					rules := "absent"
					if r.RulesPresent {
						rules = "present"
					}
					fmt.Printf("  %s %-16s synced=%-20s rules=%s\n", icon, r.Agent, synced, rules)
				}
			}

			if len(combined.Global) == 0 && len(combined.Project) == 0 {
				fmt.Println("No agent configurations found.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "target project directory")
	return cmd
}

func runAgentsSync(projectDir string, rawAgentKinds string, rawServices string, force bool, dryRun bool) (agentSetupResult, error) {
	cfg, err := loadAppConfigReadOnly()
	if err != nil {
		return agentSetupResult{}, err
	}

	isPartialSync := strings.TrimSpace(rawServices) != ""

	serviceFilter, err := resolveSyncServiceFilter(cfg, rawServices)
	if err != nil {
		return agentSetupResult{}, err
	}

	installedServices, err := buildInstalledServicesForSync(cfg, serviceFilter)
	if err != nil {
		return agentSetupResult{}, err
	}
	installedPacks, packsErr := buildInstalledPacksForSync(cfg, serviceFilter)
	if packsErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: failed to build agent skill packs, using legacy mode: %v\n", packsErr)
		installedPacks = nil
	}

	syncResults, err := agents.SyncServices(
		staticServiceInstaller{services: installedServices, packs: installedPacks},
		"",
		agents.SyncOptions{
			ProjectDir: projectDir,
			Agents:     parseAgentKinds(rawAgentKinds),
			Force:      force,
			DryRun:     dryRun,
			SkipPrune:  isPartialSync,
			SkipRules:  true,
		},
	)
	if err != nil {
		return agentSetupResult{}, err
	}

	metaContent := services.GenerateMetaAgentSkillMD()
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

		metaDir := filepath.Join(normalizedProjectDir, agentCfg.AgentSkillsDir, "kimbap")
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

	if !dryRun && !isPartialSync && len(syncResults) > 0 {
		recordProjectSyncState(projectSyncScope(projectDir), installedServices, installedPacks)
	}

	return agentSetupResult{
		SyncResults:         syncResults,
		MetaAgentSkillPaths: metaPaths,
		AgentsFound:         len(syncResults),
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

func loadServicesForSync(cfg *config.KimbapConfig, serviceFilter []string) ([]services.InstalledService, error) {
	installer := installerFromConfig(cfg)
	if len(serviceFilter) == 0 {
		return installer.ListEnabled()
	}
	installed, err := installer.ListEnabled()
	if err != nil {
		return nil, err
	}
	return filterInstalledServices(installed, serviceFilter), nil
}

func buildInstalledServicesForSync(cfg *config.KimbapConfig, serviceFilter []string) ([]agents.InstalledService, error) {
	installed, err := loadServicesForSync(cfg, serviceFilter)
	if err != nil {
		return nil, err
	}

	out := make([]agents.InstalledService, 0, len(installed))
	for _, s := range installed {
		content, genErr := services.GenerateAgentSkillMD(&s.Manifest, services.WithSource(s.Source))
		if genErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to generate SKILL.md for %q: %v\n", s.Manifest.Name, genErr)
			continue
		}
		out = append(out, agents.InstalledService{Name: s.Manifest.Name, Content: content})
	}

	return out, nil
}

func buildInstalledPacksForSync(cfg *config.KimbapConfig, serviceFilter []string) ([]agents.InstalledServicePack, error) {
	installed, err := loadServicesForSync(cfg, serviceFilter)
	if err != nil {
		return nil, err
	}
	out := make([]agents.InstalledServicePack, 0, len(installed))
	for _, s := range installed {
		pack, genErr := services.GenerateAgentSkillPack(&s.Manifest, services.WithSource(s.Source))
		if genErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to generate skill pack for %q: %v\n", s.Manifest.Name, genErr)
			continue
		}
		skillMD := pack["SKILL.md"]
		packFiles := make(map[string]string)
		for k, v := range pack {
			if k != "SKILL.md" {
				packFiles[k] = v
			}
		}
		out = append(out, agents.InstalledServicePack{Name: s.Manifest.Name, AgentSkillMD: skillMD, PackFiles: packFiles})
	}
	return out, nil
}

func resolveSyncServiceFilter(cfg *config.KimbapConfig, rawServices string) ([]string, error) {
	requested := parseCSV(rawServices)
	if len(requested) > 0 {
		installed, listErr := installerFromConfig(cfg).List()
		if listErr != nil {
			return nil, listErr
		}
		installedNames := make(map[string]struct{}, len(installed))
		for _, svc := range installed {
			installedNames[svc.Manifest.Name] = struct{}{}
		}
		for _, name := range requested {
			if _, ok := installedNames[name]; !ok {
				_, _ = fmt.Fprintf(os.Stderr, "warning: service %q is not installed and will be skipped\n", name)
			}
		}
		return requested, nil
	}

	enabled, err := installerFromConfig(cfg).ListEnabled()
	if err != nil {
		return nil, err
	}
	selected := make([]string, 0, len(enabled))
	for _, svc := range enabled {
		selected = append(selected, svc.Manifest.Name)
	}
	return selected, nil
}

func filterInstalledServices(installed []services.InstalledService, serviceFilter []string) []services.InstalledService {
	if len(serviceFilter) == 0 {
		return installed
	}
	allowed := make(map[string]struct{}, len(serviceFilter))
	for _, name := range serviceFilter {
		allowed[name] = struct{}{}
	}
	filtered := make([]services.InstalledService, 0, len(installed))
	for _, svc := range installed {
		if _, ok := allowed[svc.Manifest.Name]; ok {
			filtered = append(filtered, svc)
		}
	}
	return filtered
}

type staticServiceInstaller struct {
	services []agents.InstalledService
	packs    []agents.InstalledServicePack
}

func (i staticServiceInstaller) List() ([]agents.InstalledService, error) {
	return i.services, nil
}

func (i staticServiceInstaller) ListPacks() ([]agents.InstalledServicePack, error) {
	return i.packs, nil
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

func recordProjectSyncState(scope string, installedServices []agents.InstalledService, installedPacks []agents.InstalledServicePack) {
	if len(installedPacks) > 0 && len(installedPacks) == len(installedServices) {
		if err := agents.RecordSyncPacks(scope, installedPacks); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to record pack-aware sync state: %v\n", err)
		} else {
			return
		}
	}

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

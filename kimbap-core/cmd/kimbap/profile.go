package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/profiles"
	"github.com/dunialabs/kimbap-core/internal/skills"
	"github.com/spf13/cobra"
)

func newAgentProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-profile",
		Short: "Manage agent operating profiles",
	}

	cmd.AddCommand(newProfileInstallCommand())
	cmd.AddCommand(newProfilePrintCommand())
	cmd.AddCommand(newProfileListCommand())

	return cmd
}

func newProfileInstallCommand() *cobra.Command {
	var (
		targetDir string
		dynamic   bool
	)
	cmd := &cobra.Command{
		Use:   "install <profile-name>",
		Short: "Install agent operating profile into the current project",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			profileType := profiles.ProfileType(strings.TrimSpace(args[0]))

			dir := strings.TrimSpace(targetDir)
			if dir == "" {
				dir = "."
			}

			if dynamic {
				cfg, err := loadAppConfig()
				if err != nil {
					return err
				}
				services, svcErr := collectInstalledServicesFromConfig(cfg.Skills.Dir)
				if svcErr != nil {
					_, _ = fmt.Fprintf(os.Stderr, "warning: %v\n", svcErr)
				}
				profile, err := profiles.GenerateDynamicProfile(profileType, services)
				if err != nil {
					return err
				}
				if err := profiles.InstallProfile(profile, dir); err != nil {
					return err
				}
				return printOutput(map[string]any{
					"installed": true,
					"profile":   string(profile.Name),
					"path":      profile.InstallPath,
					"services":  len(services),
				})
			}

			profile, err := profiles.GetProfile(profileType)
			if err != nil {
				return err
			}
			if err := profiles.InstallProfile(profile, dir); err != nil {
				return err
			}
			return printOutput(map[string]any{
				"installed": true,
				"profile":   string(profile.Name),
				"path":      profile.InstallPath,
			})
		},
	}
	cmd.Flags().StringVar(&targetDir, "dir", ".", "target directory for profile installation")
	cmd.Flags().BoolVar(&dynamic, "dynamic", true, "include installed skills in profile")
	return cmd
}

func newProfilePrintCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "print <profile-name>",
		Short: "Print profile content to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			content, err := profiles.PrintProfile(profiles.ProfileType(strings.TrimSpace(args[0])))
			if err != nil {
				return err
			}
			fmt.Print(content)
			return nil
		},
	}
}

func newProfileListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available agent profiles",
		RunE: func(_ *cobra.Command, _ []string) error {
			available := []map[string]string{
				{"name": "claude-code", "path": ".claude/KIMBAP_OPERATING_RULES.md"},
				{"name": "generic", "path": ".agents/KIMBAP_OPERATING_RULES.md"},
				{"name": "cursor", "path": ".cursor/KIMBAP_OPERATING_RULES.md"},
			}
			return printOutput(available)
		},
	}
}

func collectInstalledServicesFromConfig(skillsDir string) ([]profiles.InstalledService, error) {
	installer := skills.NewLocalInstaller(skillsDir)
	installed, err := installer.List()
	if err != nil {
		return nil, fmt.Errorf("list installed skills: %w", err)
	}

	serviceMap := map[string][]string{}
	for _, it := range installed {
		for actionKey := range it.Manifest.Actions {
			serviceMap[it.Manifest.Name] = append(serviceMap[it.Manifest.Name], actionKey)
		}
	}

	out := make([]profiles.InstalledService, 0, len(serviceMap))
	for name, actions := range serviceMap {
		out = append(out, profiles.InstalledService{Name: name, Actions: actions})
	}
	return out, nil
}

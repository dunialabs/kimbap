package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/agents"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
)

func newServiceExportAgentSkillCommand() *cobra.Command {
	var outputPath string
	var outputDir string
	var exportLegacy bool

	cmd := &cobra.Command{
		Use:   "export-agent-skill <name> [--output file] [--dir directory]",
		Short: "Export installed service as agent SKILL.md",
		Long:  "Generate a SKILL.md file compatible with Claude Code, OpenAI Codex, GitHub Copilot, and other AI agents.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("dir") && cmd.Flags().Changed("output") {
				return fmt.Errorf("--dir and --output are mutually exclusive")
			}
			if exportLegacy && strings.TrimSpace(outputDir) == "" {
				return fmt.Errorf("--legacy requires --dir")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			installed, err := installerFromConfig(cfg).Get(args[0])
			if err != nil {
				return fmt.Errorf("service %q not found: %w", args[0], err)
			}
			if !installed.Enabled {
				return fmt.Errorf("service %q is installed but disabled; enable it before exporting agent skill", args[0])
			}

			skillOpts := []services.SkillMDOption{services.WithSource(installed.Source)}
			if callAlias := configuredServiceCallAlias(cfg, installed.Manifest.Name); callAlias != "" {
				skillOpts = append(skillOpts, services.WithCallAlias(callAlias))
			}

			if strings.TrimSpace(outputDir) != "" {
				serviceDir := filepath.Join(outputDir, installed.Manifest.Name)
				if !exportLegacy {
					pack, packErr := services.GenerateAgentSkillPack(&installed.Manifest, skillOpts...)
					if packErr != nil {
						return packErr
					}
					writtenFiles, writeErr := writeAgentSkillPackDir(serviceDir, pack)
					if writeErr != nil {
						return writeErr
					}
					sort.Strings(writtenFiles)
					if outputAsJSON() {
						return printOutput(map[string]any{"exported": true, "pack": true, "files": writtenFiles})
					}
					return printOutput(fmt.Sprintf(successCheck()+" %s exported (%d files)", installed.Manifest.Name, len(writtenFiles)))
				}
				content, legacyErr := services.GenerateAgentSkillMD(&installed.Manifest, skillOpts...)
				if legacyErr != nil {
					return legacyErr
				}
				if _, writeErr := writeAgentSkillPackDir(serviceDir, map[string]string{"SKILL.md": content}); writeErr != nil {
					return writeErr
				}
				outPath := filepath.Join(serviceDir, "SKILL.md")
				if outputAsJSON() {
					return printOutput(map[string]any{"exported": true, "path": outPath})
				}
				return printOutput(fmt.Sprintf(successCheck()+" %s exported to %s", installed.Manifest.Name, outPath))
			}

			content, err := services.GenerateAgentSkillMD(&installed.Manifest, skillOpts...)
			if err != nil {
				return err
			}

			if strings.TrimSpace(outputPath) != "" {
				if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("write SKILL.md: %w", err)
				}
				if outputAsJSON() {
					return printOutput(map[string]any{"exported": true, "path": outputPath})
				}
				return printOutput(fmt.Sprintf(successCheck()+" %s exported to %s", installed.Manifest.Name, outputPath))
			}

			fmt.Print(content)
			return nil
		},
	}

	cmd.Flags().StringVar(&outputPath, "output", "", "output file path")
	cmd.Flags().StringVar(&outputDir, "dir", "", "output directory (creates name/ folder pack with SKILL.md, GOTCHAS.md, RECIPES.md)")
	cmd.Flags().BoolVar(&exportLegacy, "legacy", false, "export as single SKILL.md file (legacy mode) instead of folder pack")

	return cmd
}

func writeAgentSkillPackDir(serviceDir string, pack map[string]string) ([]string, error) {
	names := make([]string, 0, len(pack))
	for filename := range pack {
		if err := validateExportPackFileName(filename); err != nil {
			return nil, err
		}
		names = append(names, filename)
	}
	sort.Strings(names)

	if err := agents.WritePackDir(serviceDir, pack); err != nil {
		return nil, err
	}

	writtenFiles := make([]string, 0, len(names))
	for _, filename := range names {
		writtenFiles = append(writtenFiles, filepath.Join(serviceDir, filename))
	}
	return writtenFiles, nil
}

func validateExportPackFileName(name string) error {
	return agents.ValidatePackFileName(name)
}

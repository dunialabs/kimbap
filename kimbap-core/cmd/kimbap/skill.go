package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/skills"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newSkillCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage skill manifests",
	}

	cmd.AddCommand(newSkillInstallCommand())
	cmd.AddCommand(newSkillListCommand())
	cmd.AddCommand(newSkillRemoveCommand())
	cmd.AddCommand(newSkillValidateCommand())
	cmd.AddCommand(newSkillGenerateCommand())

	return cmd
}

func newSkillInstallCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "install <path-to-yaml> [--force]",
		Short: "Install a local skill manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			manifest, err := skills.ParseManifestFile(args[0])
			if err != nil {
				return err
			}

			installed, err := installerFromConfig(cfg).InstallWithForce(manifest, args[0], force)
			if err != nil {
				return err
			}
			return printOutput(installed)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing skill if already installed")
	return cmd
}

func newSkillListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed skills",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			installed, err := installerFromConfig(cfg).List()
			if err != nil {
				return err
			}
			return printOutput(installed)
		},
	}
	return cmd
}

func newSkillRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove installed skill by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			if err := installerFromConfig(cfg).Remove(args[0]); err != nil {
				return err
			}
			return printOutput(map[string]any{"removed": true, "name": args[0]})
		},
	}
	return cmd
}

func newSkillValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <path-to-yaml>",
		Short: "Validate a skill manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			manifest, err := skills.ParseManifestFile(args[0])
			if err != nil {
				return err
			}
			if errs := skills.ValidateManifest(manifest); len(errs) > 0 {
				return fmt.Errorf("manifest invalid: %v", errs)
			}
			return printOutput(map[string]any{"valid": true, "name": manifest.Name, "version": manifest.Version})
		},
	}
	return cmd
}

func newSkillGenerateCommand() *cobra.Command {
	var openapiSource string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "generate --openapi <path-or-url> [--output file.yaml]",
		Short: "Generate a skill manifest from OpenAPI 3.x",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				manifest *skills.SkillManifest
				err      error
			)

			if isHTTPURL(openapiSource) {
				manifest, err = skills.GenerateFromOpenAPIURL(openapiSource)
			} else {
				manifest, err = skills.GenerateFromOpenAPIFile(openapiSource)
			}
			if err != nil {
				return err
			}

			encoded, err := yaml.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("marshal generated manifest as YAML: %w", err)
			}

			if strings.TrimSpace(outputPath) == "" {
				fmt.Print(string(encoded))
				return nil
			}

			if err := os.WriteFile(outputPath, encoded, 0o644); err != nil {
				return fmt.Errorf("write generated manifest file: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&openapiSource, "openapi", "", "OpenAPI 3.x spec path or URL")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output file path (defaults to stdout)")
	_ = cmd.MarkFlagRequired("openapi")

	return cmd
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

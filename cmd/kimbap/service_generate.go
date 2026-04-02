package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type generatedActionWarning struct {
	Action   string   `json:"action"`
	Warnings []string `json:"warnings"`
}

func newServiceGenerateCommand() *cobra.Command {
	var openapiSource string
	var outputPath string
	var serviceName string
	var tags []string
	var pathPrefixes []string
	var install bool

	cmd := &cobra.Command{
		Use:   "generate --openapi <path-or-url> [--output file.yaml]",
		Short: "Generate a service manifest from OpenAPI 3.x",
		Example: `  # Generate YAML from a local OpenAPI file
  kimbap service generate --openapi ./openapi.yaml

  # Generate from a localhost URL during API development
  kimbap service generate --openapi http://127.0.0.1:8080/openapi.yaml --name local-api

  # Limit generation to one tag and install immediately
  kimbap service generate --openapi ./openapi.yaml --tag admin --install`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var (
				manifest *services.ServiceManifest
				err      error
			)
			generateOpts := services.OpenAPIGenerateOptions{
				NameOverride: serviceName,
				Tags:         append([]string(nil), tags...),
				PathPrefixes: append([]string(nil), pathPrefixes...),
			}

			if isServiceHTTPURL(openapiSource) {
				manifest, err = services.GenerateFromOpenAPIURLWithOptions(commandContext(cmd), openapiSource, generateOpts)
			} else {
				manifest, err = services.GenerateFromOpenAPIFileWithOptions(openapiSource, generateOpts)
			}
			if err != nil {
				return err
			}
			warnings := collectGeneratedActionWarnings(manifest)
			if !outputAsJSON() {
				printGeneratedActionWarnings(warnings)
			}

			encoded, err := yaml.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("marshal generated manifest as YAML: %w", err)
			}
			resolvedOutputPath, err := writeGeneratedManifest(outputPath, encoded)
			if err != nil {
				return err
			}

			if install {
				cfg, cfgErr := loadAppConfig()
				if cfgErr != nil {
					return cfgErr
				}
				return installGeneratedManifest(cfg, manifest, resolvedOutputPath, warnings)
			}

			if strings.TrimSpace(outputPath) == "" {
				fmt.Print(string(encoded))
				return nil
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&openapiSource, "openapi", "", "OpenAPI 3.x spec path or URL")
	cmd.Flags().StringVar(&serviceName, "name", "", "override the generated service name")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output file path (defaults to stdout)")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "include only operations with a matching OpenAPI tag (repeatable)")
	cmd.Flags().StringSliceVar(&pathPrefixes, "path-prefix", nil, "include only operations whose path starts with this prefix (repeatable)")
	cmd.Flags().BoolVar(&install, "install", false, "install the generated manifest immediately")
	_ = cmd.MarkFlagRequired("openapi")

	return cmd
}

func isServiceHTTPURL(value string) bool {
	_, ok := serviceURLScheme(value)
	return ok
}

func serviceURLScheme(value string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	return scheme, true
}

func parseServiceManifestURL(ctx context.Context, serviceURL string) (*services.ServiceManifest, error) {
	const maxManifestBytes = 1 << 20
	body, _, err := services.FetchHTTPResource(ctx, serviceURL, maxManifestBytes, "service manifest", true)
	if err != nil {
		return nil, err
	}

	manifest, err := services.ParseManifest(body)
	if err != nil {
		return nil, fmt.Errorf("parse service manifest from %q: %w", serviceURL, err)
	}

	return manifest, nil
}

func collectGeneratedActionWarnings(manifest *services.ServiceManifest) []generatedActionWarning {
	if manifest == nil || len(manifest.Actions) == 0 {
		return nil
	}

	actionNames := make([]string, 0, len(manifest.Actions))
	for name := range manifest.Actions {
		actionNames = append(actionNames, name)
	}
	sort.Strings(actionNames)

	warnings := make([]generatedActionWarning, 0)
	for _, actionName := range actionNames {
		action := manifest.Actions[actionName]
		if len(action.Warnings) == 0 {
			continue
		}
		item := generatedActionWarning{
			Action:   actionName,
			Warnings: append([]string(nil), action.Warnings...),
		}
		warnings = append(warnings, item)
	}
	if len(warnings) == 0 {
		return nil
	}
	return warnings
}

func printGeneratedActionWarnings(warnings []generatedActionWarning) {
	for _, item := range warnings {
		for _, warning := range item.Warnings {
			_, _ = fmt.Fprintf(os.Stderr, "warning: generated action %s: %s\n", item.Action, warning)
		}
	}
}

func writeGeneratedManifest(outputPath string, encoded []byte) (string, error) {
	trimmed := strings.TrimSpace(outputPath)
	if trimmed == "" {
		return "", nil
	}
	if err := os.WriteFile(trimmed, encoded, 0o644); err != nil {
		return "", fmt.Errorf("write generated manifest file: %w", err)
	}
	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve generated manifest file path: %w", err)
	}
	return absPath, nil
}

func installGeneratedManifest(cfg *config.KimbapConfig, manifest *services.ServiceManifest, outputPath string, warnings []generatedActionWarning) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	source, err := generatedInstallSource(cfg.Services.Dir, manifest.Name, outputPath)
	if err != nil {
		return err
	}

	installer := installerFromConfig(cfg)
	installed, err := installer.InstallWithForceAndActivation(manifest, source, false, true)
	if err != nil {
		return err
	}

	autoAlias := ""
	autoAliasCreated := false
	actionAliasesCreated := make([]string, 0)
	if installed.Enabled {
		shortcutResult := applyInstalledShortcuts(cfg, installer, &installed.Manifest, "auto")
		autoAlias = shortcutResult.AutoAlias
		autoAliasCreated = shortcutResult.AutoAliasCreated
		actionAliasesCreated = shortcutResult.ActionAliasesCreated
	}

	if outputAsJSON() {
		maybePrintAgentSyncHint(opts.format)
		payload := map[string]any{
			"generated":                 true,
			"installed":                 true,
			"service":                   installed,
			"auto_alias":                autoAlias,
			"auto_alias_created":        autoAliasCreated,
			"action_aliases_created":    actionAliasesCreated,
			"generated_action_warnings": warnings,
		}
		if strings.TrimSpace(outputPath) != "" {
			payload["output_path"] = outputPath
		}
		return printOutput(payload)
	}

	maybePrintAgentSyncHint(opts.format)
	msg := fmt.Sprintf(successCheck()+" %s (%s) generated and installed", installed.Manifest.Name, installed.Manifest.Version)
	if strings.TrimSpace(outputPath) != "" {
		msg += fmt.Sprintf(" (manifest: %s)", outputPath)
	}
	if autoAlias != "" {
		msg += fmt.Sprintf(" (alias: %s)", autoAlias)
	}
	if len(actionAliasesCreated) > 0 {
		msg += fmt.Sprintf(" (action aliases: %s)", strings.Join(actionAliasesCreated, ", "))
	}
	return printOutput(msg)
}

func generatedInstallSource(servicesDir, serviceName, outputPath string) (string, error) {
	trimmedOutputPath := strings.TrimSpace(outputPath)
	if trimmedOutputPath != "" {
		return "local:" + trimmedOutputPath, nil
	}

	installedPath := filepath.Join(strings.TrimSpace(servicesDir), strings.TrimSpace(serviceName)+".yaml")
	absInstalledPath, err := filepath.Abs(installedPath)
	if err != nil {
		return "", fmt.Errorf("resolve installed manifest path: %w", err)
	}
	return "local:" + absInstalledPath, nil
}

package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/dunialabs/kimbap/internal/services"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newServiceGenerateCommand() *cobra.Command {
	var openapiSource string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "generate --openapi <path-or-url> [--output file.yaml]",
		Short: "Generate a service manifest from OpenAPI 3.x",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				manifest *services.ServiceManifest
				err      error
			)

			if strings.HasPrefix(strings.TrimSpace(openapiSource), "http://") {
				return fmt.Errorf("insecure URL %q rejected: use https:// for OpenAPI sources", openapiSource)
			}

			if isServiceHTTPURL(openapiSource) {
				manifest, err = services.GenerateFromOpenAPIURL(openapiSource)
			} else {
				manifest, err = services.GenerateFromOpenAPIFile(openapiSource)
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

func isServiceHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func parseServiceManifestURL(serviceURL string) (*services.ServiceManifest, error) {
	const maxManifestBytes = 1 << 20
	body, _, err := services.FetchHTTPResource(context.Background(), serviceURL, maxManifestBytes, "service manifest", true)
	if err != nil {
		return nil, err
	}

	manifest, err := services.ParseManifest(body)
	if err != nil {
		return nil, fmt.Errorf("parse service manifest from %q: %w", serviceURL, err)
	}

	return manifest, nil
}

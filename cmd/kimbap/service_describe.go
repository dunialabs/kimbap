package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newServiceDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <name>",
		Short: "Describe a catalog service before installation",
		Example: `  kimbap service describe github
  kimbap service describe open-meteo-geocoding`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}

			summaries, err := loadCatalogServiceSummaries(cfg)
			if err != nil {
				return err
			}

			summary, ok := findCatalogServiceSummary(summaries, args[0])
			if !ok {
				return catalogServiceNotFoundError(args[0], summaries)
			}

			payload := buildCatalogDescribePayload(summary)
			if outputAsJSON() {
				return printOutput(payload)
			}

			printCatalogDescribeText(payload)
			return nil
		},
	}

	return cmd
}

func printCatalogDescribeText(payload catalogDescribePayload) {
	_, _ = fmt.Fprintf(os.Stdout, "Service: %s\n", payload.Name)
	_, _ = fmt.Fprintf(os.Stdout, "Description: %s\n", emptyDash(payload.Description))
	_, _ = fmt.Fprintf(os.Stdout, "Adapter: %s\n", emptyDash(payload.Adapter))
	_, _ = fmt.Fprintf(os.Stdout, "Auth: %s\n", formatCatalogAuthSummary(payload))
	_, _ = fmt.Fprintf(os.Stdout, "Status: %s\n", payload.Status)
	if len(payload.Shortcuts) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Shortcuts: %s\n", strings.Join(payload.Shortcuts, ", "))
	}
	if len(payload.Aliases) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Aliases: %s\n", strings.Join(payload.Aliases, ", "))
	}
	if payload.Triggers != nil {
		_, _ = fmt.Fprintln(os.Stdout, "Triggers:")
		if len(payload.Triggers.TaskVerbs) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "  Verbs: %s\n", strings.Join(payload.Triggers.TaskVerbs, ", "))
		}
		if len(payload.Triggers.Objects) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "  Objects: %s\n", strings.Join(payload.Triggers.Objects, ", "))
		}
		if len(payload.Triggers.InsteadOf) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "  Instead Of: %s\n", strings.Join(payload.Triggers.InsteadOf, "; "))
		}
		if len(payload.Triggers.Exclusions) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "  Exclusions: %s\n", strings.Join(payload.Triggers.Exclusions, "; "))
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Actions (%d):\n", payload.ActionCount)
	for _, action := range payload.Actions {
		line := fmt.Sprintf("  - %s: %s", action.Name, emptyDash(action.Description))
		if action.AuthRequired {
			line += " [auth]"
		}
		_, _ = fmt.Fprintln(os.Stdout, line)
	}

	if payload.GotchaCount > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Gotchas (%d):\n", payload.GotchaCount)
		for _, gotcha := range payload.Gotchas {
			if strings.TrimSpace(gotcha.Severity) != "" {
				_, _ = fmt.Fprintf(os.Stdout, "  - [%s] %s\n", gotcha.Severity, emptyDash(gotcha.Symptom))
				continue
			}
			_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", emptyDash(gotcha.Symptom))
		}
	}

	if payload.RecipeCount > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Recipes (%d):\n", payload.RecipeCount)
		for _, recipe := range payload.Recipes {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s: %s\n", recipe.Name, emptyDash(recipe.Description))
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Hint: %s\n", payload.InstallHint)
}

func formatCatalogAuthSummary(payload catalogDescribePayload) string {
	if !payload.AuthRequired {
		return "none"
	}

	authType := strings.TrimSpace(payload.AuthType)
	if authType == "" {
		authType = "required"
	}
	if strings.TrimSpace(payload.CredentialRef) != "" {
		return fmt.Sprintf("%s (%s)", authType, payload.CredentialRef)
	}
	return authType
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

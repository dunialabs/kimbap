package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newServiceSearchCommand() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search catalog services by keyword or capability",
		Example: `  # Find catalog services before installing
  kimbap service search github

  # Search by action or trigger keywords
  kimbap service search "issue management"`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if limit < 0 {
				return fmt.Errorf("--limit must be non-negative")
			}

			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}

			summaries, err := loadCatalogServiceSummaries(cfg)
			if err != nil {
				return err
			}

			results := catalogSearchResults(summaries, splitSearchTerms(args[0]))
			if limit > 0 && len(results) > limit {
				results = results[:limit]
			}

			if len(results) == 0 {
				if outputAsJSON() {
					return printOutput([]catalogSearchResult{})
				}
				return printOutput("No matching catalog services found.")
			}

			if outputAsJSON() {
				return printOutput(results)
			}

			for _, result := range results {
				desc := result.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Printf("%-24s %-14s %s", result.Name, result.Status, desc)
				if summary := formatCatalogSearchMatchSummary(result); summary != "" {
					fmt.Printf(" [%s]", summary)
				}
				fmt.Println()
			}
			fmt.Println()
			fmt.Println("Run 'kimbap service describe <name>' for full catalog details.")
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "maximum number of results to return")
	return cmd
}

func formatCatalogSearchMatchSummary(result catalogSearchResult) string {
	if len(result.MatchedActions) > 0 {
		actions := result.MatchedActions
		extra := ""
		if len(actions) > 3 {
			extra = fmt.Sprintf(", +%d more", len(actions)-3)
			actions = actions[:3]
		}
		return "actions: " + strings.Join(actions, ", ") + extra
	}
	if len(result.MatchedFields) > 0 {
		fields := result.MatchedFields
		if len(fields) > 3 {
			fields = fields[:3]
		}
		return "matched: " + strings.Join(fields, ", ")
	}
	return ""
}

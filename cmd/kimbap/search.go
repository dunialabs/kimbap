package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/spf13/cobra"
)

type searchResult struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Risk        actions.RiskLevel `json:"risk"`
	Score       int               `json:"score"`
	Namespace   string            `json:"namespace"`
}

func newSearchCommand() *cobra.Command {
	var service string
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search actions by keyword or description",
		Example: `  # Find actions related to messages
  kimbap search message

  # Search within a specific service
  kimbap search list --service github`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if limit < 0 {
				return fmt.Errorf("--limit must be non-negative")
			}

			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}

			defs, err := loadInstalledActions(cfg)
			if err != nil {
				return err
			}

			terms := splitSearchTerms(args[0])
			results := make([]searchResult, 0)
			for _, def := range defs {
				if service != "" && !strings.EqualFold(def.Namespace, service) {
					continue
				}

				score := scoreAction(def, terms)
				if score <= 0 {
					continue
				}

				results = append(results, searchResult{
					Name:        def.Name,
					Description: def.Description,
					Risk:        def.Risk,
					Score:       score,
					Namespace:   def.Namespace,
				})
			}

			sort.Slice(results, func(i, j int) bool {
				if results[i].Score == results[j].Score {
					return results[i].Name < results[j].Name
				}
				return results[i].Score > results[j].Score
			})

			if limit > 0 && len(results) > limit {
				results = results[:limit]
			}

			if len(results) == 0 {
				if outputAsJSON() {
					return printOutput([]searchResult{})
				}
				return printOutput("No matching actions found.")
			}

			if outputAsJSON() {
				return printOutput(results)
			}

			for _, result := range results {
				fmt.Printf("%-40s %s\n", result.Name, result.Description)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&service, "service", "", "scope search to a specific service")
	cmd.Flags().IntVar(&limit, "limit", 10, "maximum number of results to return")

	return cmd
}

func splitSearchTerms(query string) []string {
	parts := strings.Fields(strings.ToLower(query))
	seen := map[string]bool{}
	terms := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			terms = append(terms, trimmed)
		}
	}
	return terms
}

func scoreAction(def actions.ActionDefinition, terms []string) int {
	if len(terms) == 0 {
		return 0
	}

	name := strings.ToLower(def.Name)
	description := strings.ToLower(def.Description)
	namespace := strings.ToLower(def.Namespace)

	score := 0
	for _, term := range terms {
		if strings.Contains(name, term) {
			score += 3
		}
		if strings.Contains(description, term) {
			score += 2
		}
		if strings.Contains(namespace, term) {
			score += 1
		}
	}

	return score
}

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
				if service == "" {
					summaries, catalogErr := loadCatalogServiceSummaries(cfg)
					if catalogErr == nil {
						catalogResults := catalogSearchResults(summaries, terms)
						notActive := make([]catalogSearchResult, 0, len(catalogResults))
						for _, r := range catalogResults {
							if !r.Enabled {
								notActive = append(notActive, r)
							}
						}
						if len(notActive) > 0 {
							showLimit := limit
							if showLimit <= 0 || showLimit > len(notActive) {
								showLimit = len(notActive)
							}
							show := notActive[:showLimit]
							fmt.Println("No matching actions found in installed services.")
							fmt.Println()
							fmt.Println("Found in catalog:")
							for _, r := range show {
								desc := r.Description
								if desc == "" {
									desc = "-"
								}
								fmt.Printf("  %-22s %-14s %s\n", r.Name, r.Status, desc)
							}
							if len(notActive) > showLimit {
								fmt.Printf("  ... and %d more\n", len(notActive)-showLimit)
							}
							fmt.Println()
							if len(notActive) == 1 {
								for _, s := range summaries {
									if s.Name == notActive[0].Name {
										fmt.Println(catalogServiceInstallHint(s))
										break
									}
								}
							} else {
								hasDisabled := false
								for _, r := range notActive {
									if r.Installed {
										hasDisabled = true
										break
									}
								}
								if hasDisabled {
									fmt.Println("Run 'kimbap service install <name>' to install, or 'kimbap service enable <name>' to enable.")
								} else {
									fmt.Println("Run 'kimbap service install <name>' to install.")
								}
							}
							return nil
						}
					}
				}
				return printOutput("No matching actions found.")
			}

			if outputAsJSON() {
				return printOutput(results)
			}

			useColor := isColorStdout()

			hasAnyShortcut := false
			for _, result := range results {
				if shortestCommandAlias(result.Name, cfg.CommandAliases) != "" {
					hasAnyShortcut = true
					break
				}
			}

			if hasAnyShortcut {
				fmt.Printf("%-40s %-16s %s\n", "ACTION", "SHORTCUT", "DESCRIPTION")
			}
			for _, result := range results {
				desc := result.Description
				if desc == "" {
					desc = "-"
				}
				badge := searchRiskBadge(string(result.Risk), useColor)
				if hasAnyShortcut {
					shortcut := shortestCommandAlias(result.Name, cfg.CommandAliases)
					if shortcut == "" {
						shortcut = "-"
					}
					if badge != "" {
						fmt.Printf("%-40s %-16s %s  %s\n", result.Name, shortcut, desc, badge)
					} else {
						fmt.Printf("%-40s %-16s %s\n", result.Name, shortcut, desc)
					}
				} else {
					if badge != "" {
						fmt.Printf("%-40s %s  %s\n", result.Name, desc, badge)
					} else {
						fmt.Printf("%-40s %s\n", result.Name, desc)
					}
				}
			}

			fmt.Println()
			if hasAnyShortcut {
				fmt.Println("Run '<shortcut> --help' for usage (or 'kimbap call <service.action> --help').")
			} else {
				fmt.Println("Run 'kimbap call <service.action> --help' for usage details.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&service, "service", "", "scope search to a specific service")
	cmd.Flags().IntVar(&limit, "limit", 10, "maximum number of results to return")

	return cmd
}

func searchRiskBadge(risk string, useColor bool) string {
	switch risk {
	case "read", "write", "admin", "destructive":
	default:
		return ""
	}
	badge := "[" + risk + "]"
	if !useColor {
		return badge
	}
	switch risk {
	case "write", "admin":
		return "\x1b[33m" + badge + "\x1b[0m"
	case "destructive":
		return "\x1b[31m" + badge + "\x1b[0m"
	default:
		return "\x1b[2m" + badge + "\x1b[0m"
	}
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

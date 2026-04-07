package main

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/spf13/cobra"
)

func newActionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "actions",
		Short: "List and inspect available actions",
	}

	var service string
	var brief bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List actions from installed services",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}
			defs, err := loadInstalledActions(cfg)
			if err != nil {
				return err
			}

			out := make([]actions.ActionDefinition, 0)
			for _, def := range defs {
				if service != "" && !strings.EqualFold(def.Namespace, service) {
					continue
				}
				out = append(out, def)
			}

			if brief && outputAsJSON() {
				briefList := make([]map[string]string, 0, len(out))
				for _, def := range out {
					briefList = append(briefList, map[string]string{
						"name":        def.Name,
						"description": def.Description,
						"risk":        string(def.Risk),
					})
				}
				return printOutput(briefList)
			}

			if outputAsJSON() {
				return printOutput(out)
			}

			if len(out) == 0 {
				return printOutput("No actions found.")
			}

			useColor := isColorStdout()

			hasAnyShortcut := false
			for _, def := range out {
				if shortestCommandAlias(def.Name, cfg.CommandAliases) != "" {
					hasAnyShortcut = true
					break
				}
			}

			if hasAnyShortcut {
				fmt.Printf("%-40s %-16s %s\n", "ACTION", "SHORTCUT", "DESCRIPTION")
			}
			for _, def := range out {
				desc := def.Description
				if desc == "" {
					desc = "-"
				}
				badge := searchRiskBadge(string(def.Risk), useColor)
				if hasAnyShortcut {
					shortcut := shortestCommandAlias(def.Name, cfg.CommandAliases)
					if shortcut == "" {
						shortcut = "-"
					}
					if badge != "" {
						fmt.Printf("%-40s %-16s %s  %s\n", def.Name, shortcut, desc, badge)
					} else {
						fmt.Printf("%-40s %-16s %s\n", def.Name, shortcut, desc)
					}
				} else {
					if badge != "" {
						fmt.Printf("%-40s %s  %s\n", def.Name, desc, badge)
					} else {
						fmt.Printf("%-40s %s\n", def.Name, desc)
					}
				}
			}

			fmt.Println()
			if hasAnyShortcut {
				fmt.Println("Run '<shortcut> --help' for usage (or 'kimbap call <service.action> --help' for the full form).")
			} else {
				fmt.Println("Run 'kimbap call <service.action> --help' for usage details.")
			}

			return nil
		},
	}
	listCmd.Flags().StringVar(&service, "service", "", "filter by service name")
	listCmd.Flags().BoolVar(&brief, "brief", false, "show only action names and descriptions (agent-friendly)")

	describeCmd := &cobra.Command{
		Use:   "describe <service.action>",
		Short: "Describe one action",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}
			def, err := resolveActionByName(cfg, args[0])
			if err != nil {
				return err
			}

			credReady := false
			if def.Auth.Type == actions.AuthTypeNone || def.Auth.Optional {
				credReady = true
			} else if strings.TrimSpace(def.Auth.CredentialRef) != "" {
				credCheck := probeCredentialReadOnly(cfg, strings.TrimSpace(def.Auth.CredentialRef))
				if credCheck.Status == "pass" {
					credReady = true
				} else if !strings.Contains(credCheck.Detail, "credential missing") && !strings.Contains(credCheck.Detail, "vault is locked") {
					_, _ = fmt.Fprintf(os.Stderr, "warning: vault unavailable, credential status unknown: %s\n", credCheck.Detail)
				}
			}
			approvalRequired := def.ApprovalHint == actions.ApprovalRequired

			if outputAsJSON() {
				return printOutput(map[string]any{
					"action":            def,
					"credential_ready":  credReady,
					"approval_required": approvalRequired,
				})
			}

			fmt.Printf("Action: %s\n", def.Name)
			if shortcut := shortestCommandAlias(def.Name, cfg.CommandAliases); shortcut != "" {
				fmt.Printf("Shortcut: %s\n", shortcut)
			}
			fmt.Printf("Description: %s\n", def.Description)
			fmt.Printf("Namespace: %s\n", def.Namespace)
			fmt.Printf("HTTP: %s %s\n", strings.ToUpper(def.Verb), def.Resource)
			fmt.Printf("Risk: %s\n", def.Risk)
			fmt.Printf("Auth: %s (%s)\n", def.Auth.Type, def.Auth.CredentialRef)
			credReadyStr := "false"
			if credReady {
				credReadyStr = "true"
				if isColorStdout() {
					credReadyStr = "\x1b[32m" + credReadyStr + "\x1b[0m"
				}
			} else if isColorStdout() {
				credReadyStr = "\x1b[31m" + credReadyStr + "\x1b[0m"
			}
			approvalStr := "false"
			if approvalRequired {
				approvalStr = "true"
				if isColorStdout() {
					approvalStr = "\x1b[33m" + approvalStr + "\x1b[0m"
				}
			}
			fmt.Printf("Credential Ready: %s\n", credReadyStr)
			fmt.Printf("Approval Required: %s\n", approvalStr)

			if def.InputSchema != nil && len(def.InputSchema.Properties) > 0 {
				fmt.Println("Params:")
				keys := make([]string, 0, len(def.InputSchema.Properties))
				for key := range def.InputSchema.Properties {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					s := def.InputSchema.Properties[key]
					required := "optional"
					if slices.Contains(def.InputSchema.Required, key) {
						required = "required"
					}
					enumStr := ""
					if s != nil && len(s.Enum) > 0 && len(s.Enum) <= 8 {
						parts := make([]string, len(s.Enum))
						for i, v := range s.Enum {
							parts[i] = fmt.Sprintf("%v", v)
						}
						enumStr = ", one of: " + strings.Join(parts, ", ")
					}
					fmt.Printf("  - %s: %s (%s%s)\n", key, s.Type, required, enumStr)
				}

				var requiredParts, optionalParts []string
				for _, key := range keys {
					s := def.InputSchema.Properties[key]
					paramType := s.Type
					if paramType == "" {
						paramType = "value"
					}
					if slices.Contains(def.InputSchema.Required, key) {
						requiredParts = append(requiredParts, fmt.Sprintf("--%s <%s>", key, paramType))
					} else {
						optionalParts = append(optionalParts, fmt.Sprintf("[--%s <%s>]", key, paramType))
					}
				}
				invocation := preferredInvocation(def.Name, cfg.CommandAliases)
				fmt.Printf("Usage:\n  %s", invocation)
				for _, p := range requiredParts {
					fmt.Printf(" %s", p)
				}
				for _, p := range optionalParts {
					fmt.Printf(" %s", p)
				}
				fmt.Println()
			} else {
				invocation := preferredInvocation(def.Name, cfg.CommandAliases)
				fmt.Printf("Usage:\n  %s\n", invocation)
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd, describeCmd)
	return cmd
}

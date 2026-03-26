package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/actions"
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
			cfg, err := loadAppConfig()
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

			if brief {
				for _, def := range out {
					fmt.Printf("%-40s %s\n", def.Name, def.Description)
				}
				return nil
			}

			for _, def := range out {
				fmt.Printf("- %s\n  risk=%s auth=%s method=%s path=%s\n", def.Name, def.Risk, def.Auth.Type, strings.ToUpper(def.Verb), def.Resource)
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
			cfg, err := loadAppConfig()
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
				if vs, vErr := initVaultStore(cfg); vErr == nil {
					if raw, gErr := vs.GetValue(contextBackground(), defaultTenantID(), def.Auth.CredentialRef); gErr == nil && len(raw) > 0 {
						credReady = true
					}
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
			fmt.Printf("Description: %s\n", def.Description)
			fmt.Printf("Namespace: %s\n", def.Namespace)
			fmt.Printf("HTTP: %s %s\n", strings.ToUpper(def.Verb), def.Resource)
			fmt.Printf("Risk: %s\n", def.Risk)
			fmt.Printf("Auth: %s (%s)\n", def.Auth.Type, def.Auth.CredentialRef)
			fmt.Printf("Credential Ready: %v\n", credReady)
			fmt.Printf("Approval Required: %v\n", approvalRequired)

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
					fmt.Printf("  - %s: %s (%s)\n", key, s.Type, required)
				}
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd, describeCmd)
	return cmd
}

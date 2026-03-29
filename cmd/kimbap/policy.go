package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dunialabs/kimbap/internal/policy"
	"github.com/spf13/cobra"
)

func newPolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage policy documents",
	}

	cmd.AddCommand(newPolicySetCommand())
	cmd.AddCommand(newPolicyGetCommand())
	cmd.AddCommand(newPolicyEvalCommand())

	return cmd
}

func newPolicySetCommand() *cobra.Command {
	var filePath string
	cmd := &cobra.Command{
		Use:   "set --file <path>",
		Short: "Load active policy from YAML file",
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(filePath) == "" {
				return fmt.Errorf("--file is required")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			raw, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			doc, err := policy.ParseDocument(raw)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(cfg.Policy.Path), 0o700); err != nil {
				return err
			}
			if err := os.WriteFile(cfg.Policy.Path, raw, 0o600); err != nil {
				return err
			}

			if outputAsJSON() {
				return printOutput(map[string]any{
					"policy_path": cfg.Policy.Path,
					"rule_count":  len(doc.Rules),
					"version":     doc.Version,
				})
			}
			return printOutput(fmt.Sprintf(successCheck()+" policy loaded (%d rules, version %s)", len(doc.Rules), doc.Version))
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "policy file path")
	return cmd
}

func newPolicyGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show active policy",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}
			doc, err := policy.ParseDocumentFile(cfg.Policy.Path)
			if err != nil {
				return err
			}
			return printOutput(doc)
		},
	}
	return cmd
}

func newPolicyEvalCommand() *cobra.Command {
	var (
		agent  string
		action string
		params string
	)
	cmd := &cobra.Command{
		Use:   "eval --agent <name> --action <service.action> [--params <json>]",
		Short: "Dry-run policy evaluation",
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(agent) == "" {
				return fmt.Errorf("--agent is required")
			}
			if strings.TrimSpace(action) == "" {
				return fmt.Errorf("--action is required")
			}

			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}
			doc, err := policy.ParseDocumentFile(cfg.Policy.Path)
			if err != nil {
				return err
			}
			evaluator := policy.NewEvaluator(doc)

			argMap, err := parseJSONMap(params)
			if err != nil {
				return fmt.Errorf("parse --params: %w", err)
			}

			service, localAction := splitActionName(action)
			risk := "low"
			mutating := false
			if def, err := resolveActionByName(cfg, action); err == nil {
				risk = def.Risk.DocVocab()
				mutating = !def.Idempotent
			}

			res, err := evaluator.Evaluate(contextBackground(), policy.EvalRequest{
				TenantID:  defaultTenantID(),
				AgentName: agent,
				Service:   service,
				Action:    localAction,
				Risk:      risk,
				Mutating:  mutating,
				Args:      argMap,
			})
			if err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(res)
			}
			decision := strings.ToUpper(string(res.Decision))
			if res.Decision == "require_approval" {
				decision = "REQUIRE APPROVAL"
			}
			if isColorStdout() {
				switch res.Decision {
				case "allow":
					decision = "\x1b[32m" + decision + "\x1b[0m"
				case "deny":
					decision = "\x1b[31m" + decision + "\x1b[0m"
				case "require_approval":
					decision = "\x1b[33m" + decision + "\x1b[0m"
				}
			}
			fmt.Printf("Decision:  %s\n", decision)
			if strings.TrimSpace(res.Reason) != "" {
				fmt.Printf("Reason:    %s\n", res.Reason)
			}
			if res.MatchedRule != nil && strings.TrimSpace(res.MatchedRule.ID) != "" {
				fmt.Printf("Rule:      %s\n", res.MatchedRule.ID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&agent, "agent", "", "agent name")
	cmd.Flags().StringVar(&action, "action", "", "service.action")
	cmd.Flags().StringVar(&params, "params", "", "json object of params")
	return cmd
}

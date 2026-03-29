package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/policy"
	"github.com/spf13/cobra"
)

type preflightCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type preflightReport struct {
	Action   string           `json:"action"`
	Verdict  string           `json:"verdict"`
	Blockers []string         `json:"blockers"`
	Checks   []preflightCheck `json:"checks"`
}

func newCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check <service.action>",
		Short: "Run local preflight checks for an action",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			report, err := runActionPreflight(cfg, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			if outputAsJSON() {
				return printOutput(report)
			}
			return printOutput(renderPreflightReport(report))
		},
	}
}

func runActionPreflight(cfg *config.KimbapConfig, actionName string) (preflightReport, error) {
	def, err := resolveActionByName(cfg, actionName)
	if err != nil {
		return preflightReport{}, err
	}

	report := preflightReport{Action: def.Name}
	report.Checks = append(report.Checks, preflightCheck{
		Name:   "action_resolution",
		Status: "pass",
		Detail: fmt.Sprintf("resolved action %q", def.Name),
	})

	credentialBlocked := false
	if def.Auth.Type == actions.AuthTypeNone || def.Auth.Optional {
		report.Checks = append(report.Checks, preflightCheck{
			Name:   "credential",
			Status: "pass",
			Detail: "no credential required",
		})
	} else {
		vs, vaultErr := initVaultStore(cfg)
		if vaultErr != nil {
			reason := classifyVaultInitError(vaultErr)
			if reason == "vault_locked" {
				report.Blockers = append(report.Blockers, "vault_locked")
				report.Checks = append(report.Checks, preflightCheck{
					Name:   "credential",
					Status: "fail",
					Detail: "vault is locked; set KIMBAP_MASTER_KEY_HEX",
				})
			} else {
				report.Blockers = append(report.Blockers, "vault_error")
				report.Checks = append(report.Checks, preflightCheck{
					Name:   "credential",
					Status: "fail",
					Detail: "vault is unavailable; fix vault/config and retry",
				})
			}
			credentialBlocked = true
		} else {
			closeVaultStoreIfPossible(vs)
			if isCredentialReady(cfg, actions.ExecutionRequest{Action: *def}) {
				report.Checks = append(report.Checks, preflightCheck{
					Name:   "credential",
					Status: "pass",
					Detail: fmt.Sprintf("credential ref %q is available", strings.TrimSpace(def.Auth.CredentialRef)),
				})
			} else {
				report.Blockers = append(report.Blockers, "credential_missing")
				report.Checks = append(report.Checks, preflightCheck{
					Name:   "credential",
					Status: "fail",
					Detail: "credential missing; run 'kimbap link <service>'",
				})
				credentialBlocked = true
			}
		}
	}

	if credentialBlocked {
		report.Checks = append(report.Checks, preflightCheck{
			Name:   "policy",
			Status: "skip",
			Detail: "skipped because credential preflight failed",
		})
		report.Verdict = "not_ready"
		return report, nil
	}

	policyPath := strings.TrimSpace(cfg.Policy.Path)
	if policyPath == "" {
		report.Checks = append(report.Checks, preflightCheck{
			Name:   "policy",
			Status: "pass",
			Detail: "no policy file configured; allow all",
		})
	} else if _, statErr := os.Stat(policyPath); os.IsNotExist(statErr) {
		report.Checks = append(report.Checks, preflightCheck{
			Name:   "policy",
			Status: "pass",
			Detail: "policy file not found; allow all",
		})
	} else {
		doc, parseErr := policy.ParseDocumentFile(policyPath)
		if parseErr != nil {
			return preflightReport{}, parseErr
		}

		evaluator := policy.NewEvaluator(doc)
		service, action := splitActionName(def.Name)
		evalResult, evalErr := evaluator.Evaluate(contextBackground(), policy.EvalRequest{
			TenantID: defaultTenantID(),
			Service:  service,
			Action:   action,
			Risk:     def.Risk.DocVocab(),
			Mutating: def.Risk != actions.RiskRead,
			Args:     map[string]any{},
		})
		if evalErr != nil {
			return preflightReport{}, evalErr
		}

		switch evalResult.Decision {
		case policy.DecisionDeny:
			report.Blockers = append(report.Blockers, "policy_denied")
			report.Checks = append(report.Checks, preflightCheck{
				Name:   "policy",
				Status: "fail",
				Detail: "policy decision: deny",
			})
		case policy.DecisionRequireApproval:
			report.Checks = append(report.Checks, preflightCheck{
				Name:   "policy",
				Status: "pass",
				Detail: "policy decision: require_approval",
			})
		default:
			report.Checks = append(report.Checks, preflightCheck{
				Name:   "policy",
				Status: "pass",
				Detail: "policy decision: allow",
			})
		}
	}

	if len(report.Blockers) == 0 {
		report.Verdict = "ready"
	} else {
		report.Verdict = "not_ready"
	}
	return report, nil
}

func classifyVaultInitError(err error) string {
	lowerErr := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(lowerErr, "vault master key is required") {
		return "vault_locked"
	}
	return "vault_error"
}

func renderPreflightReport(report preflightReport) string {
	var b strings.Builder
	b.WriteString("Action: ")
	b.WriteString(report.Action)
	b.WriteString("\n")
	b.WriteString("Verdict: ")
	if report.Verdict == "ready" {
		b.WriteString("READY")
	} else {
		b.WriteString("NOT READY")
	}
	b.WriteString("\n")

	if len(report.Blockers) > 0 {
		b.WriteString("Blockers:\n")
		for _, blocker := range report.Blockers {
			b.WriteString("  - ")
			b.WriteString(blocker)
			b.WriteString("\n")
		}
	}

	b.WriteString("Checks:\n")
	for _, check := range report.Checks {
		symbol := "-"
		switch strings.ToLower(strings.TrimSpace(check.Status)) {
		case "pass":
			symbol = "✓"
		case "fail":
			symbol = "✗"
		case "skip":
			symbol = "-"
		}
		b.WriteString("  ")
		b.WriteString(symbol)
		b.WriteString(" ")
		b.WriteString(check.Name)
		if strings.TrimSpace(check.Detail) != "" {
			b.WriteString(": ")
			b.WriteString(check.Detail)
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

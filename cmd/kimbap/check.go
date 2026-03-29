package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	corecrypto "github.com/dunialabs/kimbap/internal/crypto"
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
			cfg, err := loadAppConfigReadOnly()
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
		credCheck := probeCredentialReadOnly(cfg, strings.TrimSpace(def.Auth.CredentialRef))
		report.Checks = append(report.Checks, credCheck)
		if credCheck.Status == "fail" {
			credentialBlocked = true
			switch {
			case strings.Contains(credCheck.Detail, "vault is locked"):
				report.Blockers = append(report.Blockers, "vault_locked")
			case strings.Contains(credCheck.Detail, "credential missing"):
				report.Blockers = append(report.Blockers, "credential_missing")
			default:
				report.Blockers = append(report.Blockers, "vault_error")
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
			Mutating: !def.Idempotent,
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

func probeCredentialReadOnly(cfg *config.KimbapConfig, credentialRef string) preflightCheck {
	vaultPath := strings.TrimSpace(cfg.Vault.Path)
	if vaultPath == "" {
		return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is unavailable; fix vault/config and retry"}
	}
	masterKey, err := resolveVaultMasterKeyReadOnly(cfg)
	if err != nil {
		if classifyVaultInitError(err) == "vault_locked" {
			return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is locked; set KIMBAP_MASTER_KEY_HEX"}
		}
		return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is unavailable; fix vault/config and retry"}
	}
	if _, err := os.Stat(vaultPath); err != nil {
		if os.IsNotExist(err) {
			return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is unavailable; fix vault/config and retry"}
		}
		return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is unavailable; fix vault/config and retry"}
	}
	db, err := sql.Open("sqlite", "file:"+vaultPath+"?mode=ro&immutable=1")
	if err != nil {
		return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is unavailable; fix vault/config and retry"}
	}
	defer db.Close()

	var envelope corecrypto.EncryptedEnvelope
	queryErr := db.QueryRowContext(contextBackground(), `
		SELECT sv.ciphertext, sv.nonce, sv.salt, sv.key_id, sv.algorithm, sv.wrapped_dek, sv.dek_nonce
		FROM secrets s
		JOIN secret_versions sv ON sv.secret_id = s.id AND sv.version = s.current_version
		WHERE s.tenant_id = ? AND s.name = ?
		LIMIT 1
	`, defaultTenantID(), credentialRef).Scan(
		&envelope.Ciphertext,
		&envelope.Nonce,
		&envelope.Salt,
		&envelope.KeyID,
		&envelope.Algorithm,
		&envelope.WrappedDEK,
		&envelope.DEKNonce,
	)
	if queryErr == sql.ErrNoRows {
		return preflightCheck{Name: "credential", Status: "fail", Detail: "credential missing; run 'kimbap link <service>'"}
	}
	if queryErr != nil {
		return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is unavailable; fix vault/config and retry"}
	}

	envelopeService, err := corecrypto.NewEnvelopeService(masterKey)
	if err != nil {
		return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is unavailable; fix vault/config and retry"}
	}
	if _, err := envelopeService.Decrypt(&envelope); err != nil {
		return preflightCheck{Name: "credential", Status: "fail", Detail: "vault is unavailable; fix vault/config and retry"}
	}

	return preflightCheck{Name: "credential", Status: "pass", Detail: fmt.Sprintf("credential ref %q is available", credentialRef)}
}

func resolveVaultMasterKeyReadOnly(cfg *config.KimbapConfig) ([]byte, error) {
	if decoded, err, present := decodeMasterKeyHexEnv(); present {
		if err != nil {
			return nil, err
		}
		return decoded, nil
	}

	devEnabled := strings.EqualFold(strings.TrimSpace(cfg.Mode), "dev")
	if !devEnabled {
		if rawDev, ok := os.LookupEnv("KIMBAP_DEV"); ok {
			parsed, err := strconv.ParseBool(strings.TrimSpace(rawDev))
			if err != nil {
				return nil, err
			}
			devEnabled = parsed
		}
	}

	if !devEnabled {
		return nil, fmt.Errorf("vault master key is required: set KIMBAP_MASTER_KEY_HEX or enable dev mode (--mode dev or KIMBAP_DEV=true)")
	}

	devKeyPath := filepath.Join(cfg.DataDir, ".dev-master-key")
	key, err := readPersistedDevMasterKey(devKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read dev master key %s: %w", devKeyPath, err)
	}
	return key, nil
}

func classifyVaultInitError(err error) string {
	lowerErr := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(lowerErr, "vault master key is required") {
		return "vault_locked"
	}
	return "vault_error"
}

func renderPreflightReport(report preflightReport) string {
	useColor := isColorStdout()
	var b strings.Builder
	b.WriteString("Action: ")
	b.WriteString(report.Action)
	b.WriteString("\n")
	b.WriteString("Verdict: ")
	if report.Verdict == "ready" {
		verdict := "READY"
		if useColor {
			verdict = "\x1b[32m" + verdict + "\x1b[0m"
		}
		b.WriteString(verdict)
	} else {
		verdict := "NOT READY"
		if useColor {
			verdict = "\x1b[31m" + verdict + "\x1b[0m"
		}
		b.WriteString(verdict)
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
			if useColor {
				symbol = "\x1b[32m" + symbol + "\x1b[0m"
			}
		case "fail":
			symbol = "✗"
			if useColor {
				symbol = "\x1b[31m" + symbol + "\x1b[0m"
			}
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

package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/dunialabs/kimbap/internal/agents"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/policy"
	"github.com/spf13/cobra"

	_ "modernc.org/sqlite"
)

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func newDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run runtime diagnostics",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadDoctorConfig()
			if err != nil {
				return err
			}

			checks := make([]doctorCheck, 0, 8)
			hasFailure := false

			configCheck := checkConfigFile()
			checks = append(checks, configCheck)
			hasFailure = hasFailure || configCheck.Status == "fail"

			dataDirCheck := checkDataDirWritable(cfg.DataDir)
			checks = append(checks, dataDirCheck)
			hasFailure = hasFailure || dataDirCheck.Status == "fail"

			vaultCheck := checkVaultAccessible(cfg)
			checks = append(checks, vaultCheck)
			hasFailure = hasFailure || vaultCheck.Status == "fail"

			servicesCheck := checkServicesDir(cfg.Services.Dir)
			checks = append(checks, servicesCheck)
			hasFailure = hasFailure || servicesCheck.Status == "fail"

			policyCheck := checkPolicyFile(cfg.Policy.Path)
			checks = append(checks, policyCheck)
			hasFailure = hasFailure || policyCheck.Status == "fail"

			agentIntegrationCheck := checkAgentIntegration(cfg)
			checks = append(checks, agentIntegrationCheck)
			hasFailure = hasFailure || agentIntegrationCheck.Status == "fail"

			credentialHealthCheck := checkCredentialHealth(cfg)
			checks = append(checks, credentialHealthCheck)
			hasFailure = hasFailure || credentialHealthCheck.Status == "fail"

			actionReadinessCheck := checkActionReadiness(cfg)
			checks = append(checks, actionReadinessCheck)
			hasFailure = hasFailure || actionReadinessCheck.Status == "fail"

			if outputAsJSON() {
				if err := printOutput(checks); err != nil {
					return err
				}
			} else {
				if err := printOutput(renderDoctorSummary(checks)); err != nil {
					return err
				}
			}
			if hasFailure {
				return fmt.Errorf("doctor found failing checks")
			}
			return nil
		},
	}
	return cmd
}

func checkConfigFile() doctorCheck {
	path, err := resolveConfigPath()
	if err != nil {
		return doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return doctorCheck{Name: "config file", Status: "warn", Detail: "not found; defaults will be used"}
		}
		return doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	if _, err := config.LoadKimbapConfigWithoutDefault(path); err != nil {
		return doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "config file", Status: "ok", Detail: path}
}

func loadDoctorConfig() (*config.KimbapConfig, error) {
	cfg, err := loadBaseConfigForCLI()
	if err != nil {
		return nil, err
	}

	config.ApplyDataDirOverride(cfg, opts.dataDir)

	return cfg, nil
}

func resolveConfigPath() (string, error) {
	return config.ResolveConfigPathWithDataDir(opts.configPath, opts.dataDir)
}

func checkDataDirWritable(dataDir string) doctorCheck {
	st, err := os.Stat(dataDir)
	if err != nil {
		return doctorCheck{Name: "data directory writable", Status: "fail", Detail: err.Error()}
	}
	if !st.IsDir() {
		return doctorCheck{Name: "data directory writable", Status: "fail", Detail: "path is not a directory"}
	}
	file, err := os.CreateTemp(dataDir, "kimbap-doctor-*.tmp")
	if err != nil {
		return doctorCheck{Name: "data directory writable", Status: "fail", Detail: err.Error()}
	}
	_ = file.Close()
	_ = os.Remove(file.Name())
	return doctorCheck{Name: "data directory writable", Status: "ok", Detail: dataDir}
}

func checkVaultAccessible(cfg *config.KimbapConfig) doctorCheck {
	if strings.TrimSpace(cfg.Vault.Path) == "" {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: "vault path is empty"}
	}
	st, err := os.Stat(cfg.Vault.Path)
	if err != nil {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: err.Error()}
	}
	if st.IsDir() {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: "path is not a file"}
	}
	db, err := sql.Open("sqlite", cfg.Vault.Path)
	if err != nil {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: err.Error()}
	}
	defer db.Close()
	var exists int
	if err := db.QueryRowContext(contextBackground(), "SELECT 1 FROM secrets LIMIT 1").Scan(&exists); err != nil && err != sql.ErrNoRows {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "vault accessible", Status: "ok", Detail: cfg.Vault.Path}
}

func checkServicesDir(servicesDir string) doctorCheck {
	st, err := os.Stat(servicesDir)
	if err != nil {
		return doctorCheck{Name: "services directory exists", Status: "fail", Detail: err.Error()}
	}
	if !st.IsDir() {
		return doctorCheck{Name: "services directory exists", Status: "fail", Detail: "path is not a directory"}
	}
	return doctorCheck{Name: "services directory exists", Status: "ok", Detail: servicesDir}
}

func checkPolicyFile(path string) doctorCheck {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return doctorCheck{Name: "policy file valid", Status: "skip", Detail: "no active policy file"}
	}
	if _, err := policy.ParseDocumentFile(path); err != nil {
		return doctorCheck{Name: "policy file valid", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "policy file valid", Status: "ok", Detail: path}
}

func checkAgentIntegration(_ *config.KimbapConfig) doctorCheck {
	status, err := agents.GlobalStatus()
	if err != nil {
		return doctorCheck{Name: "agent integration", Status: "skip", Detail: err.Error()}
	}
	for _, result := range status {
		if result.AgentSkillPresent || result.InjectPresent {
			return doctorCheck{Name: "agent integration", Status: "ok", Detail: "agent integration detected"}
		}
	}
	return doctorCheck{Name: "agent integration", Status: "warn", Detail: "no agents configured — run 'kimbap agents setup'"}
}

func checkCredentialHealth(cfg *config.KimbapConfig) doctorCheck {
	if strings.TrimSpace(cfg.Vault.Path) == "" {
		return doctorCheck{Name: "credentials stored", Status: "skip", Detail: "vault path is empty"}
	}
	st, err := os.Stat(cfg.Vault.Path)
	if err != nil {
		return doctorCheck{Name: "credentials stored", Status: "skip", Detail: err.Error()}
	}
	if st.IsDir() {
		return doctorCheck{Name: "credentials stored", Status: "skip", Detail: "path is not a file"}
	}

	db, err := sql.Open("sqlite", cfg.Vault.Path)
	if err != nil {
		return doctorCheck{Name: "credentials stored", Status: "skip", Detail: err.Error()}
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(contextBackground(), "SELECT COUNT(*) FROM secrets").Scan(&count); err != nil {
		return doctorCheck{Name: "credentials stored", Status: "skip", Detail: err.Error()}
	}
	if count == 0 {
		return doctorCheck{Name: "credentials stored", Status: "warn", Detail: "no credentials stored — run 'kimbap link list'"}
	}
	return doctorCheck{Name: "credentials stored", Status: "ok", Detail: fmt.Sprintf("%d credential(s) stored", count)}
}

func checkActionReadiness(cfg *config.KimbapConfig) doctorCheck {
	installed, err := loadEnabledInstalledServices(cfg)
	if err != nil {
		return doctorCheck{Name: "action readiness", Status: "skip", Detail: err.Error()}
	}

	for _, service := range installed {
		if len(service.Manifest.Actions) > 0 {
			return doctorCheck{Name: "action readiness", Status: "ok", Detail: "enabled service actions detected"}
		}
	}

	return doctorCheck{Name: "action readiness", Status: "warn", Detail: "no services installed — run 'kimbap service install <name>'"}
}

func renderDoctorSummary(checks []doctorCheck) string {
	passed := 0
	skipped := 0
	warnings := 0
	failed := 0
	for _, c := range checks {
		switch c.Status {
		case "ok":
			passed++
		case "skip":
			skipped++
		case "warn":
			warnings++
		case "fail":
			failed++
		}
	}

	b := strings.Builder{}
	b.WriteString("Kimbap runtime diagnostics\n")
	useColor := isColorStdout()
	warnStr := fmt.Sprintf("%d", warnings)
	failStr := fmt.Sprintf("%d", failed)
	if useColor && warnings > 0 {
		warnStr = "\x1b[33m" + warnStr + "\x1b[0m"
	}
	if useColor && failed > 0 {
		failStr = "\x1b[31m" + failStr + "\x1b[0m"
	}
	_, _ = fmt.Fprintf(&b, "  passed: %d  skipped: %d  warnings: %s  failed: %s\n\n", passed, skipped, warnStr, failStr)
	for _, c := range checks {
		icon := "✓"
		switch c.Status {
		case "skip":
			icon = "-"
		case "warn":
			icon = "!"
		case "fail":
			icon = "✗"
		}
		if useColor {
			switch c.Status {
			case "ok":
				icon = "[32m" + icon + "[0m"
			case "warn":
				icon = "[33m" + icon + "[0m"
			case "fail":
				icon = "[31m" + icon + "[0m"
			}
		}
		_, _ = fmt.Fprintf(&b, "  %s %-25s %s\n", icon, c.Name, c.Detail)
	}

	if warnings > 0 || failed > 0 {
		fixes := make([]string, 0)
		seen := make(map[string]bool)
		for _, c := range checks {
			if c.Status != "warn" && c.Status != "fail" {
				continue
			}
			fix := doctorSuggestedFix(c.Name)
			if fix != "" && !seen[fix] {
				seen[fix] = true
				fixes = append(fixes, fix)
			}
		}
		if len(fixes) > 0 {
			b.WriteString("\n\nSuggested fixes:\n")
			for _, fix := range fixes {
				_, _ = fmt.Fprintf(&b, "  %s\n", fix)
			}
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func doctorSuggestedFix(checkName string) string {
	switch checkName {
	case "agent integration":
		return "kimbap agents setup"
	case "credentials stored":
		return "kimbap link list"
	case "action readiness":
		return "kimbap service install <name>"
	default:
		return ""
	}
}

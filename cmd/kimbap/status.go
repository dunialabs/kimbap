package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dunialabs/kimbap/internal/agents"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/spf13/cobra"
)

type statusSummary struct {
	Mode        string `json:"mode"`
	Vault       string `json:"vault"`
	Services    int    `json:"services"`
	Credentials int    `json:"credentials"`
	Agents      int    `json:"agents"`
	Policy      string `json:"policy"`
}

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show aggregated runtime health",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfigReadOnly()
			if err != nil {
				return err
			}
			summary := collectStatusSummary(cfg)
			if outputAsJSON() {
				return printOutput(summary)
			}
			return printOutput(renderStatusSummary(summary))
		},
	}
	return cmd
}

func collectStatusSummary(cfg *config.KimbapConfig) statusSummary {
	servicesCount, _ := countEnabledServices(cfg)
	credentialCount, _ := countStoredCredentials(cfg)
	agentCount := countConfiguredAgents()
	mode := strings.TrimSpace(cfg.Mode)
	if mode == "" {
		mode = "embedded"
	}
	return statusSummary{
		Mode:        mode,
		Vault:       vaultStatusString(cfg),
		Services:    servicesCount,
		Credentials: credentialCount,
		Agents:      agentCount,
		Policy:      policyStatusString(cfg),
	}
}

func renderStatusSummary(summary statusSummary) string {
	return strings.Join([]string{
		fmt.Sprintf("%-14s%s", "Mode:", summary.Mode),
		fmt.Sprintf("%-14s%s", "Vault:", summary.Vault),
		fmt.Sprintf("%-14s%d enabled", "Services:", summary.Services),
		fmt.Sprintf("%-14s%d stored", "Credentials:", summary.Credentials),
		fmt.Sprintf("%-14s%d configured", "Agents:", summary.Agents),
		fmt.Sprintf("%-14s%s", "Policy:", summary.Policy),
	}, "\n")
}

func countEnabledServices(cfg *config.KimbapConfig) (int, error) {
	installed, err := loadEnabledInstalledServices(cfg)
	if err != nil {
		return 0, err
	}
	return len(installed), nil
}

func countStoredCredentials(cfg *config.KimbapConfig) (int, error) {
	vaultPath := strings.TrimSpace(cfg.Vault.Path)
	if vaultPath == "" {
		return 0, fmt.Errorf("vault path is empty")
	}
	st, err := os.Stat(vaultPath)
	if err != nil || st.IsDir() {
		return 0, fmt.Errorf("vault not accessible")
	}
	db, err := sql.Open("sqlite", "file:"+vaultPath+"?mode=ro&immutable=1")
	if err != nil {
		return 0, err
	}
	defer db.Close()
	count := 0
	if err := db.QueryRowContext(contextBackground(), "SELECT COUNT(*) FROM secrets").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func countConfiguredAgents() int {
	results, err := agents.GlobalStatus()
	if err != nil {
		return 0
	}
	n := 0
	for _, r := range results {
		if r.AgentSkillPresent || r.InjectPresent {
			n++
		}
	}
	return n
}

func vaultStatusString(cfg *config.KimbapConfig) string {
	vaultPath := strings.TrimSpace(cfg.Vault.Path)
	if vaultPath == "" {
		return "error"
	}
	available, err := vaultKeyAvailableReadOnly(cfg)
	if err != nil {
		return "error"
	}
	if available {
		return "ready"
	}
	st, err := os.Stat(vaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "not initialized"
		}
		return "error"
	}
	if st.IsDir() {
		return "error"
	}
	return "locked"
}

func vaultKeyAvailableReadOnly(cfg *config.KimbapConfig) (bool, error) {
	if _, err, present := decodeMasterKeyHexEnv(); present {
		if err != nil {
			return false, err
		}
		return true, nil
	}
	devEnabled := strings.EqualFold(strings.TrimSpace(cfg.Mode), "dev")
	if !devEnabled {
		if rawDev, ok := os.LookupEnv("KIMBAP_DEV"); ok {
			parsed, err := strconv.ParseBool(strings.TrimSpace(rawDev))
			if err != nil {
				return false, err
			}
			devEnabled = parsed
		}
	}
	if !devEnabled {
		return false, nil
	}
	devKeyPath := filepath.Join(cfg.DataDir, ".dev-master-key")
	_, err := os.Stat(devKeyPath)
	return err == nil, nil
}

func policyStatusString(cfg *config.KimbapConfig) string {
	check := checkPolicyFile(cfg.Policy.Path)
	switch check.Status {
	case "ok":
		return "loaded"
	case "skip":
		return "not configured"
	default:
		return "error"
	}
}

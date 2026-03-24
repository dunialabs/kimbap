package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/policy"
	"github.com/dunialabs/kimbap-core/internal/vault"
	"github.com/spf13/cobra"
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
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			checks := make([]doctorCheck, 0, 5)
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

			skillsCheck := checkSkillsDir(cfg.Skills.Dir)
			checks = append(checks, skillsCheck)
			hasFailure = hasFailure || skillsCheck.Status == "fail"

			policyCheck := checkPolicyFile(cfg.Policy.Path)
			checks = append(checks, policyCheck)
			hasFailure = hasFailure || policyCheck.Status == "fail"

			if err := printOutput(checks); err != nil {
				return err
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
		return doctorCheck{Name: "config file", Status: "fail", Detail: fmt.Sprintf("missing: %s", path)}
	}
	if _, err := config.LoadKimbapConfig(path); err != nil {
		return doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "config file", Status: "ok", Detail: path}
}

func resolveConfigPath() (string, error) {
	if opts.configPath != "" {
		return opts.configPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("resolve user home directory")
	}
	return filepath.Join(home, ".kimbap", "config.yaml"), nil
}

func checkDataDirWritable(dataDir string) doctorCheck {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return doctorCheck{Name: "data directory writable", Status: "fail", Detail: err.Error()}
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
	store, err := initVaultStore(cfg)
	if err != nil {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: err.Error()}
	}
	_, err = store.List(contextBackground(), defaultTenantID(), vault.ListOptions{Limit: 1})
	if err != nil {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "vault accessible", Status: "ok", Detail: cfg.Vault.Path}
}

func checkSkillsDir(skillsDir string) doctorCheck {
	st, err := os.Stat(skillsDir)
	if err != nil {
		return doctorCheck{Name: "skills directory exists", Status: "fail", Detail: err.Error()}
	}
	if !st.IsDir() {
		return doctorCheck{Name: "skills directory exists", Status: "fail", Detail: "path is not a directory"}
	}
	return doctorCheck{Name: "skills directory exists", Status: "ok", Detail: skillsDir}
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

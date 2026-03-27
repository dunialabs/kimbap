package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

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

			servicesCheck := checkServicesDir(cfg.Services.Dir)
			checks = append(checks, servicesCheck)
			hasFailure = hasFailure || servicesCheck.Status == "fail"

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
		if os.IsNotExist(err) {
			return doctorCheck{Name: "config file", Status: "fail", Detail: fmt.Sprintf("missing: %s", path)}
		}
		return doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	if _, err := config.LoadKimbapConfigWithoutDefault(path); err != nil {
		return doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "config file", Status: "ok", Detail: path}
}

func loadDoctorConfig() (*config.KimbapConfig, error) {
	var (
		cfg *config.KimbapConfig
		err error
	)
	if strings.TrimSpace(opts.configPath) == "" {
		cfg, err = config.LoadKimbapConfig()
	} else {
		cfg, err = config.LoadKimbapConfigWithoutDefault(opts.configPath)
	}
	if err != nil {
		return nil, err
	}

	config.ApplyDataDirOverride(cfg, opts.dataDir)

	return cfg, nil
}

func resolveConfigPath() (string, error) {
	return config.ResolveConfigPath(opts.configPath)
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

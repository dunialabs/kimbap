package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/policy"
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

			skillsCheck := checkSkillsDir(cfg.Services.Dir)
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

	if strings.TrimSpace(opts.dataDir) != "" {
		prevDataDir := cfg.DataDir
		cfg.DataDir = opts.dataDir
		if cfg.Vault.Path == filepath.Join(prevDataDir, "vault.db") {
			cfg.Vault.Path = filepath.Join(cfg.DataDir, "vault.db")
		}
		if cfg.Services.Dir == filepath.Join(prevDataDir, "services") {
			cfg.Services.Dir = filepath.Join(cfg.DataDir, "services")
		}
		if cfg.Audit.Path == filepath.Join(prevDataDir, "audit.jsonl") {
			cfg.Audit.Path = filepath.Join(cfg.DataDir, "audit.jsonl")
		}
		if cfg.Policy.Path == filepath.Join(prevDataDir, "policy.yaml") {
			cfg.Policy.Path = filepath.Join(cfg.DataDir, "policy.yaml")
		}
		if cfg.Database.DSN == filepath.Join(prevDataDir, "kimbap.db") {
			cfg.Database.DSN = filepath.Join(cfg.DataDir, "kimbap.db")
		}
	}

	return cfg, nil
}

func resolveConfigPath() (string, error) {
	if opts.configPath != "" {
		return opts.configPath, nil
	}
	xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	xdgPath := ""
	xdgIsDir := false
	if xdg != "" {
		xdgPath = filepath.Join(xdg, "kimbap", "config.yaml")
		if st, err := os.Stat(xdgPath); err == nil {
			if !st.IsDir() {
				return xdgPath, nil
			}
			xdgIsDir = true
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat xdg config path %q: %w", xdgPath, err)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		if xdgPath != "" {
			if xdgIsDir {
				return "", fmt.Errorf("config path is a directory: %s", xdgPath)
			}
			return xdgPath, nil
		}
		return "", fmt.Errorf("resolve user home directory")
	}
	legacyPath := filepath.Join(home, ".kimbap", "config.yaml")
	if xdgPath != "" {
		if st, err := os.Stat(legacyPath); err == nil {
			if st.IsDir() {
				return "", fmt.Errorf("legacy config path is a directory: %s", legacyPath)
			}
			return legacyPath, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat legacy config path %q: %w", legacyPath, err)
		}
		if xdgIsDir {
			return "", fmt.Errorf("config path is a directory: %s", xdgPath)
		}
		return xdgPath, nil
	}
	if st, err := os.Stat(legacyPath); err == nil && st.IsDir() {
		return "", fmt.Errorf("legacy config path is a directory: %s", legacyPath)
	}
	return legacyPath, nil
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

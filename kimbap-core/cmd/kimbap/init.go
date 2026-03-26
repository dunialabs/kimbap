package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newInitCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap a fresh Kimbap installation",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := buildInitConfig()
			if err := validateInitMode(cfg.Mode); err != nil {
				return err
			}

			checks := make([]doctorCheck, 0, 8)
			hasFailure := false

			dataDirCheck := ensureDirWithStatus("data directory writable", cfg.DataDir)
			checks = append(checks, dataDirCheck)
			hasFailure = hasFailure || dataDirCheck.Status == "fail"

			configPath, configCheck := writeInitConfig(cfg, force)
			checks = append(checks, configCheck)
			hasFailure = hasFailure || configCheck.Status == "fail"

			skillsCheck := ensureDirWithStatus("services directory exists", cfg.Services.Dir)
			checks = append(checks, skillsCheck)
			hasFailure = hasFailure || skillsCheck.Status == "fail"

			vaultCheck, devKeyCheck := initializeVault(cfg)
			checks = append(checks, vaultCheck, devKeyCheck)
			hasFailure = hasFailure || vaultCheck.Status == "fail" || devKeyCheck.Status == "fail"

			policyCheck := ensurePolicyFile(cfg.Policy.Path, cfg.Mode)
			checks = append(checks, policyCheck)
			hasFailure = hasFailure || policyCheck.Status == "fail"

			auditCheck := ensureEmptyFile("audit file initialized", cfg.Audit.Path)
			checks = append(checks, auditCheck)
			hasFailure = hasFailure || auditCheck.Status == "fail"

			if outputAsJSON() {
				if err := printOutput(checks); err != nil {
					return err
				}
				if hasFailure {
					return fmt.Errorf("init failed to complete")
				}
				return nil
			}

			if err := printOutput(renderInitSummary(configPath, checks)); err != nil {
				return err
			}
			if hasFailure {
				return fmt.Errorf("init failed to complete")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config file")
	return cmd
}

func buildInitConfig() *config.KimbapConfig {
	cfg := config.DefaultConfig()
	if strings.TrimSpace(opts.dataDir) != "" {
		cfg.DataDir = opts.dataDir
		cfg.Vault.Path = filepath.Join(cfg.DataDir, "vault.db")
		cfg.Services.Dir = filepath.Join(cfg.DataDir, "services")
		cfg.Audit.Path = filepath.Join(cfg.DataDir, "audit.jsonl")
		cfg.Database.DSN = filepath.Join(cfg.DataDir, "kimbap.db")
		cfg.Policy.Path = filepath.Join(cfg.DataDir, "policy.yaml")
	}
	if strings.TrimSpace(opts.logLevel) != "" {
		cfg.LogLevel = opts.logLevel
	}
	if strings.TrimSpace(opts.mode) != "" {
		cfg.Mode = opts.mode
	}
	return cfg
}

func validateInitMode(mode string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "embedded", "dev", "connected":
		return nil
	default:
		return fmt.Errorf("unsupported mode %q: expected dev, embedded, or connected", mode)
	}
}

func writeInitConfig(cfg *config.KimbapConfig, force bool) (string, doctorCheck) {
	path, err := resolveConfigPath()
	if err != nil {
		return "", doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}

	if _, statErr := os.Stat(path); statErr == nil && !force {
		return path, doctorCheck{Name: "config file", Status: "skip", Detail: fmt.Sprintf("exists: %s (use --force to overwrite)", path)}
	}

	payload, err := yaml.Marshal(cfg)
	if err != nil {
		return path, doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}

	status := "ok"
	detailPrefix := "created"
	if _, err := os.Stat(path); err == nil {
		detailPrefix = "overwritten"
	}

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return path, doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	return path, doctorCheck{Name: "config file", Status: status, Detail: fmt.Sprintf("%s: %s", detailPrefix, path)}
}

func ensureDirWithStatus(name, dir string) doctorCheck {
	st, err := os.Stat(dir)
	if err == nil {
		if !st.IsDir() {
			return doctorCheck{Name: name, Status: "fail", Detail: "path exists but is not a directory"}
		}
		return doctorCheck{Name: name, Status: "skip", Detail: fmt.Sprintf("exists: %s", dir)}
	}
	if !os.IsNotExist(err) {
		return doctorCheck{Name: name, Status: "fail", Detail: err.Error()}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return doctorCheck{Name: name, Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: name, Status: "ok", Detail: fmt.Sprintf("created: %s", dir)}
}

func initializeVault(cfg *config.KimbapConfig) (doctorCheck, doctorCheck) {
	vaultPath := strings.TrimSpace(cfg.Vault.Path)
	if vaultPath == "" {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: "vault path is empty"}, doctorCheck{Name: "dev master key", Status: "skip", Detail: "mode is not dev"}
	}

	vaultExisted := fileExists(vaultPath)
	devKeyPath := filepath.Join(cfg.DataDir, ".dev-master-key")
	devKeyExisted := fileExists(devKeyPath)

	if _, err := initVaultStore(cfg); err != nil {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: err.Error()}, doctorCheck{Name: "dev master key", Status: ternary(strings.EqualFold(strings.TrimSpace(cfg.Mode), "dev"), "fail", "skip"), Detail: err.Error()}
	}

	vaultDetail := ternary(vaultExisted, "exists", "created")
	vaultCheck := doctorCheck{Name: "vault accessible", Status: ternary(vaultExisted, "skip", "ok"), Detail: fmt.Sprintf("%s: %s", vaultDetail, vaultPath)}

	if !strings.EqualFold(strings.TrimSpace(cfg.Mode), "dev") {
		return vaultCheck, doctorCheck{Name: "dev master key", Status: "skip", Detail: "mode is not dev"}
	}

	if !fileExists(devKeyPath) {
		return vaultCheck, doctorCheck{Name: "dev master key", Status: "fail", Detail: fmt.Sprintf("missing after init: %s", devKeyPath)}
	}

	keyStatus := ternary(devKeyExisted, "skip", "ok")
	keyDetail := ternary(devKeyExisted, "exists", "created")
	return vaultCheck, doctorCheck{Name: "dev master key", Status: keyStatus, Detail: fmt.Sprintf("%s: %s", keyDetail, devKeyPath)}
}

func policyForMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "dev") {
		return `version: "1.0.0"
rules:
  - id: allow-all
    priority: 1
    match:
      actions: ["*"]
    decision: allow
`
	}
	return `version: "1.0.0"
rules:
  - id: deny-unreviewed
    priority: 1
    match:
      actions: ["*"]
    decision: require_approval
`
}

func ensurePolicyFile(path, mode string) doctorCheck {
	if st, err := os.Stat(path); err == nil {
		if st.IsDir() {
			return doctorCheck{Name: "policy file valid", Status: "fail", Detail: "path exists but is a directory"}
		}
		if st.Size() == 0 {
			return doctorCheck{Name: "policy file valid", Status: "fail", Detail: fmt.Sprintf("policy file is empty: %s", path)}
		}
		return doctorCheck{Name: "policy file valid", Status: "skip", Detail: fmt.Sprintf("exists: %s", path)}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return doctorCheck{Name: "policy file valid", Status: "fail", Detail: err.Error()}
	}
	if err := os.WriteFile(path, []byte(policyForMode(mode)), 0o600); err != nil {
		return doctorCheck{Name: "policy file valid", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "policy file valid", Status: "ok", Detail: fmt.Sprintf("created: %s", path)}
}

func ensureEmptyFile(name, path string) doctorCheck {
	if fileExists(path) {
		return doctorCheck{Name: name, Status: "skip", Detail: fmt.Sprintf("exists: %s", path)}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return doctorCheck{Name: name, Status: "fail", Detail: err.Error()}
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		return doctorCheck{Name: name, Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: name, Status: "ok", Detail: fmt.Sprintf("created: %s", path)}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func renderInitSummary(configPath string, checks []doctorCheck) string {
	created := 0
	skipped := 0
	failed := 0
	for _, c := range checks {
		switch c.Status {
		case "ok":
			created++
		case "skip":
			skipped++
		case "fail":
			failed++
		}
	}

	b := strings.Builder{}
	b.WriteString("Kimbap initialization summary\n")
	_, _ = fmt.Fprintf(&b, "  config: %s\n", configPath)
	_, _ = fmt.Fprintf(&b, "  created: %d  skipped: %d  failed: %d\n\n", created, skipped, failed)
	for _, c := range checks {
		icon := "✓"
		switch c.Status {
		case "skip":
			icon = "-"
		case "fail":
			icon = "✗"
		}
		_, _ = fmt.Fprintf(&b, "  %s %-25s %s\n", icon, c.Name, c.Detail)
	}
	return strings.TrimRight(b.String(), "\n")
}

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/services"
	"github.com/dunialabs/kimbap-core/skills"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newInitCommand() *cobra.Command {
	var force bool
	var servicesRaw string
	var noServices bool
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

			serviceSelection, selectionErr := resolveInitServiceSelection(servicesRaw, noServices)
			if selectionErr != nil {
				return selectionErr
			}

			serviceCheck := installInitServices(cfg, serviceSelection, hasFailure)
			checks = append(checks, serviceCheck)
			hasFailure = hasFailure || serviceCheck.Status == "fail"

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
	cmd.Flags().StringVar(&servicesRaw, "services", "", "comma-separated official services to install")
	cmd.Flags().BoolVar(&noServices, "no-services", false, "skip service installation during init")
	return cmd
}

type initServiceSelection struct {
	Names   []string
	Skipped bool
	Reason  string
}

func resolveInitServiceSelection(rawServices string, noServices bool) (initServiceSelection, error) {
	if noServices {
		return initServiceSelection{Skipped: true, Reason: "skipped by --no-services"}, nil
	}

	if selected := parseCSV(rawServices); len(selected) > 0 {
		normalized, err := normalizeSelectedOfficialServices(selected)
		if err != nil {
			return initServiceSelection{}, err
		}
		return initServiceSelection{Names: normalized}, nil
	}

	if !isInteractiveStdin() {
		return initServiceSelection{Skipped: true, Reason: "non-interactive stdin"}, nil
	}

	if err := printOfficialServiceCategories(); err != nil {
		return initServiceSelection{}, err
	}
	_, _ = fmt.Fprint(os.Stdout, "Install services now? (Enter comma-separated names, or 'all' for everything, empty to skip): ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return initServiceSelection{}, fmt.Errorf("read service selection: %w", err)
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return initServiceSelection{Skipped: true, Reason: "empty selection"}, nil
	}

	if strings.EqualFold(trimmed, "all") {
		all, listErr := skills.List()
		if listErr != nil {
			return initServiceSelection{}, fmt.Errorf("list official services: %w", listErr)
		}
		return initServiceSelection{Names: all}, nil
	}

	selected := parseCSV(trimmed)
	normalized, normalizeErr := normalizeSelectedOfficialServices(selected)
	if normalizeErr != nil {
		return initServiceSelection{}, normalizeErr
	}
	if len(normalized) == 0 {
		return initServiceSelection{Skipped: true, Reason: "empty selection"}, nil
	}
	return initServiceSelection{Names: normalized}, nil
}

func isInteractiveStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func installInitServices(cfg *config.KimbapConfig, selection initServiceSelection, hasFailure bool) doctorCheck {
	if hasFailure {
		return doctorCheck{Name: "official services installed", Status: "skip", Detail: "skipped due to previous init failures"}
	}
	if selection.Skipped {
		return doctorCheck{Name: "official services installed", Status: "skip", Detail: selection.Reason}
	}
	if len(selection.Names) == 0 {
		return doctorCheck{Name: "official services installed", Status: "skip", Detail: "no services selected"}
	}

	installer := installerFromConfig(cfg)
	installed := 0
	enabled := 0
	skipped := 0
	failed := make([]string, 0)

	for _, name := range selection.Names {
		data, getErr := skills.Get(name)
		if getErr != nil {
			failed = append(failed, fmt.Sprintf("%s (load: %v)", name, getErr))
			continue
		}

		manifest, parseErr := services.ParseManifest(data)
		if parseErr != nil {
			failed = append(failed, fmt.Sprintf("%s (parse: %v)", name, parseErr))
			continue
		}

		if _, installErr := installer.InstallWithForceAndActivation(manifest, "official:"+name, false, true); installErr != nil {
			if errors.Is(installErr, services.ErrServiceAlreadyInstalled) {
				existing, getErr := installer.Get(name)
				if getErr == nil && existing.Enabled {
					skipped++
					continue
				}
				if enableErr := installer.Enable(name); enableErr != nil {
					failed = append(failed, fmt.Sprintf("%s (enable: %v)", name, enableErr))
				} else {
					enabled++
				}
				continue
			}
			failed = append(failed, fmt.Sprintf("%s (install: %v)", name, installErr))
			continue
		}
		installed++
	}

	detail := fmt.Sprintf("installed: %d, enabled: %d, unchanged: %d", installed, enabled, skipped)
	if len(failed) > 0 {
		return doctorCheck{Name: "official services installed", Status: "fail", Detail: fmt.Sprintf("%s, failed: %s", detail, strings.Join(failed, "; "))}
	}
	if installed == 0 && enabled == 0 {
		return doctorCheck{Name: "official services installed", Status: "skip", Detail: detail}
	}
	return doctorCheck{Name: "official services installed", Status: "ok", Detail: detail}
}

func normalizeSelectedOfficialServices(names []string) ([]string, error) {
	available, err := skills.List()
	if err != nil {
		return nil, fmt.Errorf("list official services: %w", err)
	}
	valid := make(map[string]struct{}, len(available))
	for _, name := range available {
		valid[name] = struct{}{}
	}

	out := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized == "" {
			continue
		}
		if _, ok := valid[normalized]; !ok {
			return nil, fmt.Errorf("unknown official service %q", normalized)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

func printOfficialServiceCategories() error {
	categories := map[string][]string{
		"SaaS & APIs":   {"github", "slack", "stripe", "notion", "linear", "hubspot", "airtable", "pinecone", "todoist", "posthog", "sentry", "sendgrid", "resend", "exa", "brave-search"},
		"Communication": {"telegram", "whatsapp", "wechat", "zoom", "apple-mail", "messages"},
		"Local apps":    {"blender", "comfyui", "ollama", "mermaid", "spotify", "notebooklm"},
		"macOS native":  {"finder", "safari", "contacts", "shortcuts", "apple-notes", "apple-calendar", "apple-reminders", "keynote", "pages", "numbers"},
		"Office":        {"ms-word", "ms-excel", "ms-powerpoint"},
		"Data":          {"wikipedia", "hacker-news", "coingecko", "open-meteo", "open-meteo-air-quality", "open-meteo-historical", "open-meteo-geocoding", "financial-datasets", "rest-countries", "exchange-rate", "public-holidays", "nominatim", "ntfy"},
	}

	order := []string{"SaaS & APIs", "Communication", "Local apps", "macOS native", "Office", "Data"}
	known, err := skills.List()
	if err != nil {
		return fmt.Errorf("list official services: %w", err)
	}
	knownSet := make(map[string]struct{}, len(known))
	for _, name := range known {
		knownSet[name] = struct{}{}
	}

	_, _ = fmt.Fprintln(os.Stdout, "Official services:")
	for _, category := range order {
		names := categories[category]
		filtered := make([]string, 0, len(names))
		for _, name := range names {
			if _, ok := knownSet[name]; ok {
				filtered = append(filtered, name)
			}
		}
		if len(filtered) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(os.Stdout, "  %-16s %s\n", category+":", strings.Join(filtered, ", "))
	}

	leftovers := make([]string, 0)
	for _, name := range known {
		inCategory := false
		for _, category := range order {
			for _, listed := range categories[category] {
				if listed == name {
					inCategory = true
					break
				}
			}
			if inCategory {
				break
			}
		}
		if !inCategory {
			leftovers = append(leftovers, name)
		}
	}
	if len(leftovers) > 0 {
		sort.Strings(leftovers)
		_, _ = fmt.Fprintf(os.Stdout, "  %-16s %s\n", "Other:", strings.Join(leftovers, ", "))
	}
	return nil
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

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/agents"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/policy"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/services/catalog"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newInitCommand() *cobra.Command {
	var force bool
	var servicesRaw string
	var noServices bool
	var noShortcuts bool
	var withConsole bool
	var withAgents bool
	var agentsProjectDir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap a fresh Kimbap installation",
		Example: `  # Quick start (defaults to dev mode) with interactive checklist
	  kimbap init --services select

	  # Production setup (requires KIMBAP_MASTER_KEY_HEX, installs all services)
	  kimbap init --mode embedded --services all

	  # Initialize with recommended services and agent integration
	  kimbap init --services recommended --with-agents`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := buildInitConfig()
			if withConsole {
				cfg.Console.Enabled = true
			}
			if err := validateInitMode(cfg.Mode); err != nil {
				return err
			}

			checks := make([]doctorCheck, 0, 8)
			hasFailure := false

			dataDirCheck := ensureWritableDirWithStatus("data directory writable", cfg.DataDir)
			checks, hasFailure = appendInitChecks(checks, hasFailure, dataDirCheck)

			configPath, configCheck := writeInitConfig(cfg, force)
			checks, hasFailure = appendInitChecks(checks, hasFailure, configCheck)

			skillsCheck := ensureDirWithStatus("services directory exists", cfg.Services.Dir)
			checks, hasFailure = appendInitChecks(checks, hasFailure, skillsCheck)

			vaultCheck, devKeyCheck := initializeVault(cfg)
			checks, hasFailure = appendInitChecks(checks, hasFailure, vaultCheck, devKeyCheck)

			policyCheck := ensurePolicyFile(cfg.Policy.Path, cfg.Mode)
			checks, hasFailure = appendInitChecks(checks, hasFailure, policyCheck)

			auditCheck := ensureEmptyFile("audit file initialized", cfg.Audit.Path)
			checks, hasFailure = appendInitChecks(checks, hasFailure, auditCheck)

			serviceSelection, selectionErr := resolveInitServiceSelection(servicesRaw, noServices)
			if selectionErr != nil {
				return selectionErr
			}

			serviceCheck := installInitServices(cfg, serviceSelection, hasFailure, noShortcuts)
			checks, hasFailure = appendInitChecks(checks, hasFailure, serviceCheck)

			readinessCheck := checkInitLocalAdapterReadiness(serviceSelection, hasFailure)
			checks = append(checks, readinessCheck)

			consoleCheck := ensureConsoleEnabled(configPath, configCheck, withConsole)
			checks, hasFailure = appendInitChecks(checks, hasFailure, consoleCheck)

			agentsCheck := setupInitAgents(withAgents, agentsProjectDir, hasFailure)
			checks, hasFailure = appendInitChecks(checks, hasFailure, agentsCheck)

			kbCheck := ensureKBSymlink()
			checks = append(checks, kbCheck)

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
	cmd.Flags().StringVar(&servicesRaw, "services", "", "comma-separated catalog services to install, all, recommended (legacy alias: starter), or select (interactive checklist)")
	cmd.Flags().BoolVar(&noServices, "no-services", false, "skip service installation during init")
	cmd.Flags().BoolVar(&noShortcuts, "no-shortcuts", false, "skip automatic shortcut alias setup during init service installation")
	cmd.Flags().BoolVar(&withConsole, "with-console", false, "enable embedded console route in config")
	cmd.Flags().BoolVar(&withAgents, "with-agents", false, "run agents setup and sync after init")
	cmd.Flags().StringVar(&agentsProjectDir, "agents-project-dir", "", "project directory for agents sync when used with --with-agents")
	return cmd
}

func ensureConsoleEnabled(configPath string, configCheck doctorCheck, withConsole bool) doctorCheck {
	enabled, exists, err := readConsoleEnabledFromConfig(configPath)
	if err != nil {
		return doctorCheck{Name: "console mode", Status: "fail", Detail: fmt.Sprintf("read config: %v", err)}
	}
	if !exists {
		if withConsole {
			return doctorCheck{Name: "console mode", Status: "fail", Detail: "config file missing after init"}
		}
		return doctorCheck{Name: "console mode", Status: "skip", Detail: "disabled by default"}
	}

	if !withConsole {
		if enabled {
			return doctorCheck{Name: "console mode", Status: "skip", Detail: "already enabled in config"}
		}
		return doctorCheck{Name: "console mode", Status: "skip", Detail: "disabled by default"}
	}
	if enabled {
		return doctorCheck{Name: "console mode", Status: "ok", Detail: "enabled (serve --console also supported)"}
	}
	if configCheck.Status == "skip" {
		return doctorCheck{Name: "console mode", Status: "fail", Detail: "config exists and was not overwritten; rerun with --force to persist --with-console"}
	}
	return doctorCheck{Name: "console mode", Status: "fail", Detail: "requested but not persisted to config"}
}

func setupInitAgents(withAgents bool, projectDir string, hasFailure bool) doctorCheck {
	if !withAgents {
		return doctorCheck{Name: "agent integration", Status: "skip", Detail: "skipped (use --with-agents to enable)"}
	}
	if hasFailure {
		return doctorCheck{Name: "agent integration", Status: "skip", Detail: "skipped due to previous init failures"}
	}

	metaContent := services.GenerateMetaAgentSkillMD()
	globalResults, err := agents.GlobalSetup(metaContent, agents.GlobalSetupOptions{})
	if err != nil {
		return doctorCheck{Name: "agent integration", Status: "fail", Detail: fmt.Sprintf("global setup failed: %v", err)}
	}

	globalFailures := 0
	for _, result := range globalResults {
		if strings.TrimSpace(result.Error) != "" {
			globalFailures++
		}
	}
	trimmedProjectDir := strings.TrimSpace(projectDir)
	if trimmedProjectDir == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil || strings.TrimSpace(cwd) == "" {
			trimmedProjectDir = "."
		} else {
			trimmedProjectDir = cwd
		}
	}

	syncResult, syncErr := runAgentsSync(trimmedProjectDir, "", "", false, false)
	if syncErr != nil {
		return doctorCheck{Name: "agent integration", Status: "fail", Detail: fmt.Sprintf("project sync failed: %v", syncErr)}
	}

	if globalFailures > 0 {
		return doctorCheck{Name: "agent integration", Status: "warn", Detail: fmt.Sprintf("project synced (%s) for %d agents, but %d global setup entries failed", trimmedProjectDir, syncResult.AgentsFound, globalFailures)}
	}

	return doctorCheck{Name: "agent integration", Status: "ok", Detail: fmt.Sprintf("global hints installed, project synced (%s) for %d agents", trimmedProjectDir, syncResult.AgentsFound)}
}

func readConsoleEnabledFromConfig(path string) (bool, bool, error) {
	if strings.TrimSpace(path) == "" {
		return false, false, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	var loaded config.KimbapConfig
	if err := yaml.Unmarshal(payload, &loaded); err != nil {
		return false, true, err
	}
	return loaded.Console.Enabled, true, nil
}

func installInitServices(cfg *config.KimbapConfig, selection initServiceSelection, hasFailure bool, noShortcuts bool) doctorCheck {
	if hasFailure {
		return doctorCheck{Name: "catalog services installed", Status: "skip", Detail: "skipped due to previous init failures"}
	}
	if selection.Skipped {
		return doctorCheck{Name: "catalog services installed", Status: "skip", Detail: selection.Reason}
	}
	if len(selection.Names) == 0 {
		return doctorCheck{Name: "catalog services installed", Status: "skip", Detail: "no services selected"}
	}

	installer := installerFromConfig(cfg)
	setupShortcuts := !noShortcuts
	installed := 0
	enabled := 0
	skipped := 0
	aliased := 0
	actionAliased := 0
	failed := make([]string, 0)
	applyAlias := func(manifest *services.ServiceManifest) {
		if !setupShortcuts {
			return
		}
		if manifest == nil {
			return
		}
		shortcutResult, skippedActionAliases, aliasErr, actionAliasErr := runInstalledShortcutSetup(cfg, installer, manifest)
		if aliasErr != nil {
			failed = append(failed, fmt.Sprintf("%s (alias: %v)", manifest.Name, aliasErr))
			return
		}
		if shortcutResult.AutoAliasCreated {
			aliased++
		}
		if actionAliasErr != nil {
			failed = append(failed, fmt.Sprintf("%s (action-alias: %v)", manifest.Name, actionAliasErr))
		} else {
			actionAliased += len(shortcutResult.ActionAliasesCreated)
			if len(skippedActionAliases) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "warning: skipped action aliases for %s: %s\n", manifest.Name, strings.Join(skippedActionAliases, ", "))
			}
		}
	}

	for _, name := range selection.Names {
		data, getErr := catalog.Get(name)
		if getErr != nil {
			failed = append(failed, fmt.Sprintf("%s (load: %v)", name, getErr))
			continue
		}

		manifest, parseErr := services.ParseManifest(data)
		if parseErr != nil {
			failed = append(failed, fmt.Sprintf("%s (parse: %v)", name, parseErr))
			continue
		}

		if _, installErr := installer.InstallWithForceAndActivation(manifest, "registry:"+name, false, true); installErr != nil {
			if errors.Is(installErr, services.ErrServiceAlreadyInstalled) {
				existing, getErr := installer.Get(name)
				if getErr == nil && existing.Enabled {
					applyAlias(&existing.Manifest)
					skipped++
					continue
				}
				if enableErr := installer.Enable(name); enableErr != nil {
					failed = append(failed, fmt.Sprintf("%s (enable: %v)", name, enableErr))
				} else {
					if existing, getEnabledErr := installer.Get(name); getEnabledErr == nil {
						applyAlias(&existing.Manifest)
					}
					enabled++
				}
				continue
			}
			failed = append(failed, fmt.Sprintf("%s (install: %v)", name, installErr))
			continue
		}
		applyAlias(manifest)
		installed++
	}

	detail := fmt.Sprintf("installed: %d, enabled: %d, unchanged: %d, aliased: %d, action aliases: %d", installed, enabled, skipped, aliased, actionAliased)
	if len(failed) > 0 {
		return doctorCheck{Name: "catalog services installed", Status: "fail", Detail: fmt.Sprintf("%s, failed: %s", detail, strings.Join(failed, "; "))}
	}
	if installed == 0 && enabled == 0 {
		return doctorCheck{Name: "catalog services installed", Status: "skip", Detail: detail}
	}
	return doctorCheck{Name: "catalog services installed", Status: "ok", Detail: detail}
}

func checkInitLocalAdapterReadiness(selection initServiceSelection, hasFailure bool) doctorCheck {
	if hasFailure {
		return doctorCheck{Name: "local adapter readiness", Status: "skip", Detail: "skipped due to previous init failures"}
	}
	if selection.Skipped {
		return doctorCheck{Name: "local adapter readiness", Status: "skip", Detail: "no catalog services selected"}
	}
	if len(selection.Names) == 0 {
		return doctorCheck{Name: "local adapter readiness", Status: "skip", Detail: "no services selected"}
	}

	inspected := 0
	issues := make([]string, 0)

	for _, name := range selection.Names {
		data, getErr := catalog.Get(name)
		if getErr != nil {
			continue
		}
		manifest, parseErr := services.ParseManifest(data)
		if parseErr != nil {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(manifest.Adapter)) {
		case "command":
			inspected++
			executable := ""
			if manifest.CommandSpec != nil {
				executable = strings.TrimSpace(manifest.CommandSpec.Executable)
			}
			if executable == "" {
				issues = append(issues, fmt.Sprintf("%s (missing command_spec.executable)", name))
				continue
			}
			if _, lookErr := exec.LookPath(executable); lookErr != nil {
				issues = append(issues, fmt.Sprintf("%s (executable not found: %s)", name, executable))
			}
		case "applescript":
			inspected++
			target := strings.TrimSpace(manifest.TargetApp)
			if runtime.GOOS != "darwin" {
				issues = append(issues, fmt.Sprintf("%s (AppleScript requires macOS)", name))
				continue
			}
			if target == "" {
				issues = append(issues, fmt.Sprintf("%s (missing target_app)", name))
				continue
			}
			if err := exec.Command("osascript", "-e", fmt.Sprintf("id of application \"%s\"", target)).Run(); err != nil {
				issues = append(issues, fmt.Sprintf("%s (application unavailable: %s)", name, target))
			}
		}
	}

	if inspected == 0 {
		return doctorCheck{Name: "local adapter readiness", Status: "skip", Detail: "no command/applescript services selected"}
	}
	if len(issues) > 0 {
		detail := strings.Join(issues, "; ")
		if len(issues) > 6 {
			detail = strings.Join(issues[:6], "; ") + fmt.Sprintf("; ... and %d more", len(issues)-6)
		}
		return doctorCheck{Name: "local adapter readiness", Status: "warn", Detail: detail}
	}

	return doctorCheck{Name: "local adapter readiness", Status: "ok", Detail: fmt.Sprintf("verified runtime prerequisites for %d local-adapter services", inspected)}
}

func normalizeSelectedCatalogServices(names []string) ([]string, error) {
	available, err := catalog.List()
	if err != nil {
		return nil, fmt.Errorf("list catalog services: %w", err)
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
			return nil, fmt.Errorf("unknown catalog service %q", normalized)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

func printCatalogServiceCategories() error {
	categories := map[string][]string{
		"SaaS & APIs":   {"github", "slack", "stripe", "notion", "linear", "hubspot", "airtable", "pinecone", "todoist", "posthog", "sentry", "sendgrid", "resend", "exa", "brave-search"},
		"Communication": {"telegram", "whatsapp", "zoom", "apple-mail", "messages"},
		"Local apps":    {"blender", "comfyui", "ollama", "mermaid", "kitty", "spotify"},
		"macOS native":  {"finder", "safari", "contacts", "shortcuts", "apple-notes", "apple-calendar", "apple-reminders", "keynote", "pages", "numbers"},
		"Office":        {"ms-word", "ms-excel", "ms-powerpoint"},
		"Data":          {"wikipedia", "hacker-news", "coingecko", "open-meteo", "open-meteo-air-quality", "open-meteo-historical", "open-meteo-geocoding", "financial-datasets", "rest-countries", "exchange-rate", "public-holidays", "nominatim", "ntfy"},
	}

	order := []string{"SaaS & APIs", "Communication", "Local apps", "macOS native", "Office", "Data"}
	known, err := catalog.List()
	if err != nil {
		return fmt.Errorf("list catalog services: %w", err)
	}
	knownSet := make(map[string]struct{}, len(known))
	for _, name := range known {
		knownSet[name] = struct{}{}
	}

	_, _ = fmt.Fprintln(os.Stderr, "Catalog services:")
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
		_, _ = fmt.Fprintf(os.Stderr, "  %-16s %s\n", category+":", strings.Join(filtered, ", "))
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
		_, _ = fmt.Fprintf(os.Stderr, "  %-16s %s\n", "Other:", strings.Join(leftovers, ", "))
	}
	return nil
}

func buildInitConfig() *config.KimbapConfig {
	cfg := config.DefaultConfig()
	if strings.TrimSpace(opts.dataDir) != "" {
		config.ApplyDataDirOverride(cfg, opts.dataDir)
	}
	if strings.TrimSpace(opts.logLevel) != "" {
		cfg.LogLevel = opts.logLevel
	}
	cfg.Mode = resolveInitMode()
	return cfg
}

func resolveInitMode() string {
	if mode := strings.TrimSpace(opts.mode); mode != "" {
		return mode
	}
	if mode := strings.TrimSpace(os.Getenv("KIMBAP_MODE")); mode != "" {
		return mode
	}
	return "dev"
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
	path, err := config.ResolveConfigPathWithDataDir(opts.configPath, opts.dataDir)
	if err != nil {
		return "", doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, doctorCheck{Name: "config file", Status: "fail", Detail: err.Error()}
	}

	if st, statErr := os.Stat(path); statErr == nil {
		if st.IsDir() {
			return path, doctorCheck{Name: "config file", Status: "fail", Detail: fmt.Sprintf("path is a directory: %s", path)}
		}
		if !force {
			return path, doctorCheck{Name: "config file", Status: "skip", Detail: fmt.Sprintf("exists: %s (use --force to overwrite)", path)}
		}
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

	tmp, tmpErr := os.CreateTemp(filepath.Dir(path), ".kimbap-config-*.tmp")
	if tmpErr != nil {
		return path, doctorCheck{Name: "config file", Status: "fail", Detail: tmpErr.Error()}
	}
	tmpPath := tmp.Name()
	if _, wErr := tmp.Write(payload); wErr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return path, doctorCheck{Name: "config file", Status: "fail", Detail: wErr.Error()}
	}
	if cErr := tmp.Close(); cErr != nil {
		_ = os.Remove(tmpPath)
		return path, doctorCheck{Name: "config file", Status: "fail", Detail: cErr.Error()}
	}
	_ = os.Chmod(tmpPath, 0o600)
	if rErr := os.Rename(tmpPath, path); rErr != nil {
		_ = os.Remove(tmpPath)
		return path, doctorCheck{Name: "config file", Status: "fail", Detail: rErr.Error()}
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

	if vs, err := initVaultStore(cfg); err != nil {
		return doctorCheck{Name: "vault accessible", Status: "fail", Detail: err.Error()}, doctorCheck{Name: "dev master key", Status: ternary(strings.EqualFold(strings.TrimSpace(cfg.Mode), "dev"), "fail", "skip"), Detail: err.Error()}
	} else {
		closeVaultStoreIfPossible(vs)
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
		if _, parseErr := policy.ParseDocumentFile(path); parseErr != nil {
			return doctorCheck{Name: "policy file valid", Status: "fail", Detail: parseErr.Error()}
		}
		return doctorCheck{Name: "policy file valid", Status: "skip", Detail: fmt.Sprintf("exists: %s", path)}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return doctorCheck{Name: "policy file valid", Status: "fail", Detail: err.Error()}
	}
	if err := os.WriteFile(path, []byte(policyForMode(mode)), 0o600); err != nil {
		return doctorCheck{Name: "policy file valid", Status: "fail", Detail: err.Error()}
	}
	if _, parseErr := policy.ParseDocumentFile(path); parseErr != nil {
		return doctorCheck{Name: "policy file valid", Status: "fail", Detail: parseErr.Error()}
	}
	return doctorCheck{Name: "policy file valid", Status: "ok", Detail: fmt.Sprintf("created: %s", path)}
}

func ensureWritableDirWithStatus(name, dir string) doctorCheck {
	st, err := os.Stat(dir)
	permissiveMode := false
	if err == nil {
		if !st.IsDir() {
			return doctorCheck{Name: name, Status: "fail", Detail: "path exists but is not a directory"}
		}
		if st.Mode().Perm()&0o077 != 0 {
			permissiveMode = true
		}
	} else {
		if !os.IsNotExist(err) {
			return doctorCheck{Name: name, Status: "fail", Detail: err.Error()}
		}
		if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
			return doctorCheck{Name: name, Status: "fail", Detail: mkErr.Error()}
		}
	}

	tmp, tmpErr := os.CreateTemp(dir, "kimbap-init-*.tmp")
	if tmpErr != nil {
		return doctorCheck{Name: name, Status: "fail", Detail: tmpErr.Error()}
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(tmpPath)

	if err == nil {
		if permissiveMode {
			return doctorCheck{Name: name, Status: "warn", Detail: fmt.Sprintf("permissions are %o; recommended 700: %s", st.Mode().Perm(), dir)}
		}
		return doctorCheck{Name: name, Status: "skip", Detail: fmt.Sprintf("exists: %s", dir)}
	}
	return doctorCheck{Name: name, Status: "ok", Detail: fmt.Sprintf("created: %s", dir)}
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
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func appendInitChecks(checks []doctorCheck, hasFailure bool, newChecks ...doctorCheck) ([]doctorCheck, bool) {
	checks = append(checks, newChecks...)
	for _, check := range newChecks {
		hasFailure = hasFailure || check.Status == "fail"
	}
	return checks, hasFailure
}

func renderInitSummary(configPath string, checks []doctorCheck) string {
	created := 0
	skipped := 0
	warnings := 0
	failed := 0
	for _, c := range checks {
		switch c.Status {
		case "ok":
			created++
		case "skip":
			skipped++
		case "warn":
			warnings++
		case "fail":
			failed++
		}
	}

	b := strings.Builder{}
	b.WriteString("Kimbap initialization summary\n")
	_, _ = fmt.Fprintf(&b, "  config: %s\n", configPath)
	useColor := isColorStdout()
	warnStr := fmt.Sprintf("%d", warnings)
	failStr := fmt.Sprintf("%d", failed)
	if useColor && warnings > 0 {
		warnStr = "\x1b[33m" + warnStr + "\x1b[0m"
	}
	if useColor && failed > 0 {
		failStr = "\x1b[31m" + failStr + "\x1b[0m"
	}
	_, _ = fmt.Fprintf(&b, "  created: %d  skipped: %d  warnings: %s  failed: %s\n\n", created, skipped, warnStr, failStr)
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

	if failed == 0 && created > 0 {
		b.WriteString("\nNext steps:\n")
		_, _ = fmt.Fprintf(&b, "  %-38s %s\n", "kimbap link <service>", "Connect your first service")
		_, _ = fmt.Fprintf(&b, "  %-38s %s\n", "kimbap call <service>.<action>", "Run your first action")
		_, _ = fmt.Fprintf(&b, "  %-38s %s\n", "kimbap agents setup", "Set up AI agent integration (optional)")
	}

	return strings.TrimRight(b.String(), "\n")
}

func ensureKBSymlink() doctorCheck {
	execPath, err := os.Executable()
	if err != nil {
		return doctorCheck{Name: "kb alias", Status: "skip", Detail: "cannot determine executable path"}
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return doctorCheck{Name: "kb alias", Status: "skip", Detail: "cannot resolve executable path"}
	}
	if filepath.Base(execPath) != "kimbap" {
		return doctorCheck{Name: "kb alias", Status: "skip", Detail: "binary is not named kimbap"}
	}
	kbPath := filepath.Join(filepath.Dir(execPath), "kb")
	if existing, statErr := os.Lstat(kbPath); statErr == nil {
		if existing.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(kbPath)
			if readErr == nil {
				resolvedTarget, resolveErr := resolveSymlinkTarget(kbPath, target)
				if resolveErr == nil && resolvedTarget == execPath {
					return doctorCheck{Name: "kb alias", Status: "skip", Detail: fmt.Sprintf("exists: %s", kbPath)}
				}
			}
			tmpPath := kbPath + ".tmp"
			_ = os.Remove(tmpPath)
			if symlinkErr := os.Symlink(execPath, tmpPath); symlinkErr != nil {
				return doctorCheck{Name: "kb alias", Status: "fail", Detail: fmt.Sprintf("stage symlink: %v", symlinkErr)}
			}
			if renameErr := os.Rename(tmpPath, kbPath); renameErr != nil {
				_ = os.Remove(tmpPath)
				return doctorCheck{Name: "kb alias", Status: "fail", Detail: fmt.Sprintf("replace symlink: %v", renameErr)}
			}
			return doctorCheck{Name: "kb alias", Status: "ok", Detail: fmt.Sprintf("updated: %s -> %s", kbPath, execPath)}
		}
		return doctorCheck{Name: "kb alias", Status: "skip", Detail: fmt.Sprintf("exists (not symlink): %s", kbPath)}
	}
	if symlinkErr := os.Symlink(execPath, kbPath); symlinkErr != nil {
		return doctorCheck{Name: "kb alias", Status: "fail", Detail: fmt.Sprintf("create symlink: %v (try: sudo ln -s %s %s)", symlinkErr, execPath, kbPath)}
	}
	return doctorCheck{Name: "kb alias", Status: "ok", Detail: fmt.Sprintf("created: %s -> %s", kbPath, execPath)}
}

func resolveSymlinkTarget(linkPath, target string) (string, error) {
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(linkPath), resolved)
	}
	return filepath.EvalSymlinks(resolved)
}

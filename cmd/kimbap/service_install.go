package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/dunialabs/kimbap/internal/registry"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/dunialabs/kimbap/services/catalog"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newServiceInstallCommand() *cobra.Command {
	var force bool
	var noActivate bool
	var noShortcuts bool
	cmd := &cobra.Command{
		Use:   "install <name|path-to-yaml|url> [--force]",
		Short: "Install a service manifest",
		Example: `  # Install a catalog service by name
  kimbap service install github

  # Install from a local YAML file
  kimbap service install ./my-service.yaml

  # Install from a URL
  kimbap service install https://example.com/service.yaml

  # Reinstall (overwrite existing)
  kimbap service install github --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			installer := installerFromConfig(cfg)

			manifest, source, err := resolveServiceInstallSource(commandContext(cmd), args[0])
			if err != nil {
				return err
			}

			installed, err := installer.InstallWithForceAndActivation(manifest, source, force, !noActivate)
			if err != nil {
				if errors.Is(err, services.ErrServiceAlreadyInstalled) {
					return fmt.Errorf("%w — use --force to reinstall: kimbap service install %s --force", err, args[0])
				}
				return err
			}
			if installed.Enabled && strings.HasPrefix(strings.TrimSpace(source), "registry:") {
				if credErr := ensureInstalledServiceCredentials(commandContext(cmd), cfg, &installed.Manifest); credErr != nil {
					return credErr
				}
			}

			autoAlias := ""
			autoAliasCreated := false
			actionAliasesCreated := make([]string, 0)

			setupShortcuts := !noShortcuts && installed.Enabled

			if setupShortcuts {
				shortcutResult := applyInstalledShortcuts(cfg, installer, &installed.Manifest, "auto")
				autoAlias = shortcutResult.AutoAlias
				autoAliasCreated = shortcutResult.AutoAliasCreated
				actionAliasesCreated = shortcutResult.ActionAliasesCreated
			}

			if outputAsJSON() {
				maybePrintAgentSyncHint(opts.format)
				return printOutput(map[string]any{
					"service":                installed,
					"auto_alias":             autoAlias,
					"auto_alias_created":     autoAliasCreated,
					"action_aliases_created": actionAliasesCreated,
				})
			}
			status := "enabled"
			if !installed.Enabled {
				status = "disabled"
			}
			msg := fmt.Sprintf(successCheck()+" %s (%s) installed [%s]", installed.Manifest.Name, installed.Manifest.Version, status)
			if autoAlias != "" {
				msg += fmt.Sprintf(" (alias: %s)", autoAlias)
			}
			msg += serviceShortcutHint(actionAliasesCreated)
			maybePrintAgentSyncHint(opts.format)
			if err := printOutput(msg); err != nil {
				return err
			}
			if !installed.Enabled && !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Enable: run 'kimbap service enable %s' to activate the service.\n", installed.Manifest.Name)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing service if already installed")
	cmd.Flags().BoolVar(&noActivate, "no-activate", false, "install service as disabled")
	cmd.Flags().BoolVar(&noShortcuts, "no-shortcuts", false, "skip automatic shortcut alias setup during install")
	return cmd
}

func serviceManifestRequiresCredentials(manifest *services.ServiceManifest) bool {
	if manifest == nil {
		return false
	}
	if authTypeRequiresCredential(manifest.Auth.Type) {
		return true
	}
	for _, action := range manifest.Actions {
		if action.Auth != nil && authTypeRequiresCredential(action.Auth.Type) {
			return true
		}
	}
	return false
}

func authTypeRequiresCredential(raw string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	return trimmed != "" && trimmed != string(actions.AuthTypeNone)
}

func ensureInstalledServiceCredentials(ctx context.Context, cfg *config.KimbapConfig, manifest *services.ServiceManifest) error {
	if cfg == nil || manifest == nil || !serviceManifestRequiresCredentials(manifest) {
		return nil
	}
	if !canPromptInTTY() {
		_, _ = fmt.Fprintf(os.Stderr, "Tip: run 'kimbap link %s --stdin' to store credentials for this service.\n", strings.TrimSpace(manifest.Name))
		return nil
	}

	serviceName := strings.TrimSpace(manifest.Name)
	if serviceName == "" {
		return nil
	}

	servicesByName, err := loadLinkServices(cfg)
	if err != nil {
		return fmt.Errorf("load installed services for credential setup: %w", err)
	}

	info, ok := servicesByName[strings.ToLower(serviceName)]
	if !ok {
		info = linkServiceInfo{
			Service:       serviceName,
			AuthType:      actions.AuthType(strings.ToLower(strings.TrimSpace(manifest.Auth.Type))),
			CredentialRef: strings.TrimSpace(manifest.Auth.CredentialRef),
			OAuthProvider: linkResolveOAuthProvider(serviceName),
		}
	}

	tenantID := defaultTenantID()
	oauthStates, oauthErr := listConnectorStates(ctx, cfg, tenantID)
	if oauthErr != nil {
		oauthStates = nil
	}

	if linkIsOAuthService(info, oauthStates) {
		providerID := strings.TrimSpace(info.OAuthProvider)
		if providerID == "" {
			providerID = serviceName
		}
		return runAuthConnect(
			cfg,
			providerID,
			tenantID,
			"auto",
			"",
			"auto",
			false,
			0,
			5*time.Minute,
			"",
			string(connectors.ScopeUser),
			"default",
			false,
			nil,
		)
	}

	credentialRef := strings.TrimSpace(info.CredentialRef)
	if credentialRef == "" {
		return fmt.Errorf("service %q requires authentication but has no credential_ref", serviceName)
	}

	vs, err := initVaultStore(cfg)
	if err != nil {
		return err
	}
	defer closeVaultStoreIfPossible(vs)

	exists, existsErr := vs.Exists(ctx, tenantID, credentialRef)
	if existsErr == nil && exists {
		return nil
	}
	if existsErr != nil && !errors.Is(existsErr, vault.ErrSecretNotFound) {
		return existsErr
	}
	authLabel := credentialPromptLabel(info.AuthType)
	_, _ = fmt.Fprintf(os.Stderr, "Enter %s for %s: ", authLabel, serviceName)
	payload, readErr := term.ReadPassword(int(os.Stdin.Fd()))
	if readErr != nil {
		return fmt.Errorf("read credential for %s: %w", serviceName, readErr)
	}
	_, _ = fmt.Fprintln(os.Stderr)
	payload = []byte(strings.TrimSpace(string(payload)))
	if len(payload) == 0 {
		return fmt.Errorf("credential value is empty")
	}

	secretType := linkAuthTypeToSecretType(info.AuthType)
	if _, upsertErr := vs.Upsert(ctx, tenantID, credentialRef, secretType, payload, nil, "cli"); upsertErr != nil {
		return upsertErr
	}
	return nil
}

func credentialPromptLabel(authType actions.AuthType) string {
	switch authType {
	case actions.AuthTypeBearer:
		return "a bearer token"
	case actions.AuthTypeBasic:
		return "basic auth credentials"
	case actions.AuthTypeHeader:
		return "a custom header credential"
	case actions.AuthTypeQuery:
		return "a query parameter credential"
	case actions.AuthTypeAPIKey:
		return "an API key"
	default:
		return "a credential"
	}
}

type serviceShortcutResult struct {
	AutoAlias            string
	AutoAliasCreated     bool
	ActionAliasesCreated []string
}

func runInstalledShortcutSetup(cfg *config.KimbapConfig, installer *services.LocalInstaller, manifest *services.ServiceManifest) (serviceShortcutResult, []string, error, error) {
	result := serviceShortcutResult{ActionAliasesCreated: make([]string, 0)}
	if cfg == nil || installer == nil || manifest == nil {
		return result, nil, nil, nil
	}

	alias, created, aliasErr := ensureInstalledServiceAlias(cfg, installer, manifest)
	if aliasErr == nil {
		result.AutoAlias = alias
		result.AutoAliasCreated = created
	}

	createdActionAliases, skippedActionAliases, actionAliasErr := ensureInstalledActionAliases(cfg, installer, manifest)
	if actionAliasErr == nil {
		result.ActionAliasesCreated = createdActionAliases
	}

	configPath, resolveErr := resolveConfigPath()
	if resolveErr != nil {
		combinedAliasErr := errors.Join(aliasErr, fmt.Errorf("resolve config path for shortcut verification: %w", resolveErr))
		combinedActionErr := errors.Join(actionAliasErr, fmt.Errorf("resolve config path for shortcut verification: %w", resolveErr))
		return result, skippedActionAliases, combinedAliasErr, combinedActionErr
	}

	persistedCfg, persistedErr := loadPersistedShortcutConfig(configPath)
	if persistedErr != nil {
		combinedAliasErr := errors.Join(aliasErr, persistedErr)
		combinedActionErr := errors.Join(actionAliasErr, persistedErr)
		return result, skippedActionAliases, combinedAliasErr, combinedActionErr
	}

	if aliasValidateErr := validatePersistedServiceAlias(persistedCfg, manifest, result.AutoAlias); aliasValidateErr != nil {
		result.AutoAlias = ""
		result.AutoAliasCreated = false
		aliasErr = errors.Join(aliasErr, aliasValidateErr)
	}

	validatedActionAliases, validationSkipped := filterPersistedActionAliases(persistedCfg, manifest, result.ActionAliasesCreated)
	result.ActionAliasesCreated = validatedActionAliases
	if len(validationSkipped) > 0 {
		skippedActionAliases = append(skippedActionAliases, validationSkipped...)
	}

	rollbackRejectedActionAliasExecutables(createdActionAliases, validatedActionAliases)

	return result, skippedActionAliases, aliasErr, actionAliasErr
}

func rollbackRejectedActionAliasExecutables(created, validated []string) {
	if len(created) == 0 {
		return
	}
	kept := make(map[string]struct{}, len(validated))
	for _, alias := range validated {
		kept[strings.ToLower(strings.TrimSpace(alias))] = struct{}{}
	}
	for _, alias := range created {
		normalized := strings.ToLower(strings.TrimSpace(alias))
		if _, ok := kept[normalized]; ok {
			continue
		}
		_, _ = removeExecutableActionAlias(normalized)
	}
}

func loadPersistedShortcutConfig(configPath string) (*config.KimbapConfig, error) {
	trimmed := strings.TrimSpace(configPath)
	if trimmed == "" {
		return &config.KimbapConfig{Aliases: map[string]string{}, CommandAliases: map[string]string{}}, nil
	}
	if _, err := os.Stat(trimmed); err != nil {
		if os.IsNotExist(err) {
			return &config.KimbapConfig{Aliases: map[string]string{}, CommandAliases: map[string]string{}}, nil
		}
		return nil, fmt.Errorf("stat persisted config %q: %w", trimmed, err)
	}

	persistedCfg, err := config.LoadKimbapConfigWithoutDefault(trimmed)
	if err != nil {
		return nil, fmt.Errorf("load persisted config %q: %w", trimmed, err)
	}
	if persistedCfg.Aliases == nil {
		persistedCfg.Aliases = map[string]string{}
	}
	if persistedCfg.CommandAliases == nil {
		persistedCfg.CommandAliases = map[string]string{}
	}
	return persistedCfg, nil
}

func validatePersistedServiceAlias(persistedCfg *config.KimbapConfig, manifest *services.ServiceManifest, alias string) error {
	alias = strings.ToLower(strings.TrimSpace(alias))
	if alias == "" || persistedCfg == nil || manifest == nil {
		return nil
	}
	target := strings.ToLower(strings.TrimSpace(manifest.Name))
	if target == "" {
		return nil
	}
	if persistedTarget := strings.ToLower(strings.TrimSpace(persistedCfg.Aliases[alias])); persistedTarget != target {
		return fmt.Errorf("auto alias %q was not persisted to config for %s", alias, target)
	}
	return nil
}

func filterPersistedActionAliases(persistedCfg *config.KimbapConfig, manifest *services.ServiceManifest, aliases []string) ([]string, []string) {
	if len(aliases) == 0 {
		return nil, nil
	}
	expectedTargets := expectedActionAliasTargets(manifest)
	validated := make([]string, 0, len(aliases))
	skipped := make([]string, 0)
	for _, rawAlias := range aliases {
		alias := strings.ToLower(strings.TrimSpace(rawAlias))
		if alias == "" {
			continue
		}
		expectedTarget, ok := expectedTargets[alias]
		if !ok {
			skipped = append(skipped, alias+" (no matching manifest action)")
			continue
		}
		persistedTarget := strings.ToLower(strings.TrimSpace(persistedCfg.CommandAliases[alias]))
		if persistedTarget == "" {
			skipped = append(skipped, alias+" (not persisted in config)")
			continue
		}
		if persistedTarget != expectedTarget {
			skipped = append(skipped, alias+" (config maps to "+persistedTarget+")")
			continue
		}
		if _, err := aliasLookPath(alias); err != nil {
			skipped = append(skipped, alias+" (not discoverable on PATH)")
			continue
		}
		validated = append(validated, alias)
	}
	return validated, skipped
}

func expectedActionAliasTargets(manifest *services.ServiceManifest) map[string]string {
	if manifest == nil || len(manifest.Actions) == 0 {
		return map[string]string{}
	}
	targetService := strings.ToLower(strings.TrimSpace(manifest.Name))
	out := make(map[string]string)
	actionKeys := make([]string, 0, len(manifest.Actions))
	for actionKey := range manifest.Actions {
		actionKeys = append(actionKeys, actionKey)
	}
	sort.Strings(actionKeys)
	defaultActionKey := defaultShortcutActionKey(manifest)
	for _, actionKey := range actionKeys {
		action := manifest.Actions[actionKey]
		target := targetService + "." + actionKey
		for _, alias := range action.Aliases {
			normalized := strings.ToLower(strings.TrimSpace(alias))
			if normalized != "" {
				out[normalized] = target
			}
		}
		if len(action.Aliases) == 0 && actionKey == defaultActionKey {
			for _, alias := range generatedDefaultActionAliases(manifest, actionKey) {
				normalized := strings.ToLower(strings.TrimSpace(alias))
				if normalized != "" {
					out[normalized] = target
				}
			}
		}
	}
	return out
}

func applyInstalledShortcuts(cfg *config.KimbapConfig, installer *services.LocalInstaller, manifest *services.ServiceManifest, mode string) serviceShortcutResult {
	result := serviceShortcutResult{ActionAliasesCreated: make([]string, 0)}
	if cfg == nil || installer == nil || manifest == nil {
		return result
	}
	name := strings.TrimSpace(manifest.Name)
	if name == "" {
		return result
	}

	serviceWarn := "auto alias setup skipped"
	actionWarn := "action alias setup skipped"
	if strings.EqualFold(strings.TrimSpace(mode), "enabled") {
		serviceWarn = "enabled service alias setup skipped"
		actionWarn = "enabled action alias setup skipped"
	}

	shortcutResult, skipped, aliasErr, actionAliasErr := runInstalledShortcutSetup(cfg, installer, manifest)
	result = shortcutResult

	if aliasErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %s for %s: %v\n", serviceWarn, name, aliasErr)
	}

	if actionAliasErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %s for %s: %v\n", actionWarn, name, actionAliasErr)
	} else if len(skipped) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "warning: skipped action aliases for %s: %s\n", name, strings.Join(skipped, ", "))
	}

	return result
}

func resolveServiceInstallSource(ctx context.Context, arg string) (*services.ServiceManifest, string, error) {
	if ctx == nil {
		ctx = contextBackground()
	}
	trimmed := strings.TrimSpace(arg)
	if trimmed == "" {
		return nil, "", fmt.Errorf("service source is required — provide a catalog name, file path, or URL")
	}

	if strings.HasPrefix(trimmed, "registry:") {
		registryName := strings.TrimSpace(strings.TrimPrefix(trimmed, "registry:"))
		return resolveRegistryServiceByName(ctx, registryName)
	}

	if scheme := serviceSourceURLScheme(trimmed); scheme == "http" {
		httpsURL := "https://" + strings.TrimPrefix(trimmed, "http://")
		return nil, "", fmt.Errorf("insecure URL %q rejected: use https:// to install service manifests\nTry: kimbap service install %s", trimmed, httpsURL)
	}
	if scheme := serviceSourceURLScheme(trimmed); scheme == "https" {
		manifest, err := parseServiceManifestURL(ctx, trimmed)
		if err != nil {
			return nil, "", fmt.Errorf("fetch manifest from %q: %w\nVerify the URL is accessible and returns a valid YAML manifest.", trimmed, err)
		}
		return manifest, "remote:" + trimmed, nil
	}

	if strings.HasPrefix(trimmed, "github:") {
		owner, repo, serviceName, subdir, parseErr := registry.ParseGitHubRef(trimmed)
		if parseErr != nil {
			return nil, "", parseErr
		}
		reg := registry.NewGitHubRegistry(owner, repo, "", subdir)
		resolveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		manifest, source, resolveErr := reg.Resolve(resolveCtx, serviceName)
		if resolveErr != nil {
			return nil, "", fmt.Errorf("fetch from GitHub %q: %w\nVerify the repository and service name are correct.", trimmed, resolveErr)
		}
		if manifest.Name != serviceName {
			return nil, "", fmt.Errorf("manifest name %q does not match requested service %q — check the service name in the GitHub repository", manifest.Name, serviceName)
		}
		return manifest, source, nil
	}

	if stat, err := os.Stat(trimmed); err == nil {
		if stat.IsDir() {
			return nil, "", fmt.Errorf("service source %q is a directory — pass a YAML file path (e.g. ./service.yaml), URL, or catalog service name (e.g. github)", trimmed)
		}
		absPath, absErr := filepath.Abs(trimmed)
		if absErr != nil {
			absPath = trimmed
		}
		manifest, parseErr := services.ParseManifestFile(absPath)
		if parseErr != nil {
			return nil, "", fmt.Errorf("parse manifest %q: %w\nRun 'kimbap service validate %s' for detailed validation errors.", absPath, parseErr, absPath)
		}
		return manifest, "local:" + absPath, nil
	} else if !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("stat service source %q: %w", trimmed, err)
	} else if strings.HasSuffix(strings.ToLower(trimmed), ".yaml") || strings.HasSuffix(strings.ToLower(trimmed), ".yml") {
		return nil, "", fmt.Errorf("manifest file %q not found — check the file path or use a catalog service name", trimmed)
	}

	return resolveRegistryServiceByName(ctx, trimmed)
}

func resolveRegistryServiceByName(ctx context.Context, name string) (*services.ServiceManifest, string, error) {
	if ctx == nil {
		ctx = contextBackground()
	}
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if err := services.ValidateServiceName(trimmed); err != nil {
		return nil, "", err
	}

	data, err := catalog.Get(trimmed)
	if err == nil {
		manifest, parseErr := services.ParseManifest(data)
		if parseErr != nil {
			return nil, "", fmt.Errorf("parse catalog service %q: %w\nReport this issue at https://github.com/dunialabs/kimbap/issues", trimmed, parseErr)
		}
		return manifest, "registry:" + trimmed, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, "", fmt.Errorf("load catalog service %q: %w", trimmed, err)
	}

	cfg, cfgErr := loadAppConfig()
	if cfgErr != nil {
		return nil, "", fmt.Errorf("load config for registry lookup: %w", cfgErr)
	}
	registryURL := strings.TrimSpace(cfg.Services.RegistryURL)
	if registryURL != "" && registryURL != "https://services.kimbap.ai" {
		remoteReg := registry.NewRemoteRegistry("registry", registryURL)
		resolveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		manifest, _, resolveErr := remoteReg.Resolve(resolveCtx, trimmed)
		if resolveErr == nil {
			return manifest, "registry:" + trimmed, nil
		}
		var notFound *registry.ErrNotFound
		if !errors.As(resolveErr, &notFound) {
			return nil, "", fmt.Errorf("load registry service %q from %s: %w\nCheck your network connection and registry URL configuration.", trimmed, registryURL, resolveErr)
		}
	}

	catalogNames, _ := catalog.List()
	hint := "Run 'kimbap service list --available' to see all catalog services."
	if suggestion := didYouMean(trimmed, catalogNames); suggestion != "" {
		hint = fmt.Sprintf("Did you mean %q? Run 'kimbap service list --available' to see all catalog services.", suggestion)
	}
	return nil, "", fmt.Errorf("%w. %s", &registry.ErrNotFound{Name: trimmed, Registry: "catalog"}, hint)
}

func serviceSourceURLScheme(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return ""
	}
	return scheme
}

func sourceToInstallArg(source string) string {
	trimmed := strings.TrimSpace(source)
	if strings.HasPrefix(trimmed, "registry:") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "remote:") {
		return strings.TrimPrefix(trimmed, "remote:")
	}
	if strings.HasPrefix(trimmed, "local:") {
		return strings.TrimPrefix(trimmed, "local:")
	}
	if strings.HasPrefix(trimmed, "github:") {
		rest := strings.TrimPrefix(trimmed, "github:")
		base, serviceName, ok := strings.Cut(rest, ":")
		if !ok || strings.TrimSpace(serviceName) == "" {
			return trimmed
		}
		return "github:" + strings.Trim(base, "/") + "/" + strings.TrimSpace(serviceName)
	}
	return trimmed
}

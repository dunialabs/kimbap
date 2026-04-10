package app

import (
	"errors"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/adapters"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/dunialabs/kimbap/internal/policy"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
)

type RuntimeDeps struct {
	Config           *config.KimbapConfig
	VaultStore       vault.Store
	ConnectorStore   connectors.ConnectorStore
	ConnectorConfigs []connectors.ConnectorConfig
	PolicyStore      store.PolicyStore
	PolicyPath       string
	ServicesDir      string
	AuditWriter      runtimepkg.AuditWriter
	ApprovalManager  runtimepkg.ApprovalManager
	HeldStore        store.HeldExecutionStore
}

func BuildRuntime(deps RuntimeDeps) (*runtimepkg.Runtime, error) {
	if deps.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if err := services.SetAppleScriptRegistryMode(deps.Config.Services.AppleScriptRegistryMode); err != nil {
		return nil, err
	}

	servicesDir := strings.TrimSpace(deps.ServicesDir)
	policyPath := strings.TrimSpace(deps.PolicyPath)
	if servicesDir == "" {
		servicesDir = strings.TrimSpace(deps.Config.Services.Dir)
	}
	if policyPath == "" {
		policyPath = strings.TrimSpace(deps.Config.Policy.Path)
	}

	actionRegistry := &servicesActionRegistry{
		installer:       services.NewLocalInstaller(servicesDir),
		verifyMode:      strings.ToLower(strings.TrimSpace(deps.Config.Services.Verify)),
		signaturePolicy: strings.ToLower(strings.TrimSpace(deps.Config.Services.SignaturePolicy)),
		servicesDir:     servicesDir,
	}

	var filePolicyEvaluator runtimepkg.PolicyEvaluator
	if policyPath != "" {
		if stat, err := os.Stat(policyPath); err == nil {
			if stat.IsDir() {
				return nil, fmt.Errorf("policy path %q is a directory, not a file", policyPath)
			}
			doc, parseErr := policy.ParseDocumentFile(policyPath)
			if parseErr != nil {
				return nil, parseErr
			}
			filePolicyEvaluator = &policyEvaluatorAdapter{evaluator: policy.NewEvaluator(doc)}
		} else if errors.Is(err, os.ErrNotExist) && deps.PolicyStore != nil {
		} else {
			return nil, fmt.Errorf("stat policy path %q: %w", policyPath, err)
		}
	}
	var policyEvaluator runtimepkg.PolicyEvaluator
	if deps.PolicyStore != nil {
		policyEvaluator = &storePolicyEvaluator{policyStore: deps.PolicyStore, fallback: filePolicyEvaluator}
	} else {
		policyEvaluator = filePolicyEvaluator
	}

	var credentialResolver runtimepkg.CredentialResolver
	var resolvers []runtimepkg.CredentialResolver
	if deps.ConnectorStore != nil && len(deps.ConnectorConfigs) > 0 {
		mgr := connectors.NewManager(deps.ConnectorStore)
		for _, cfg := range deps.ConnectorConfigs {
			mgr.RegisterConfig(cfg)
		}
		resolvers = append(resolvers, &connectorCredentialResolver{mgr: mgr})
	}
	if deps.VaultStore != nil {
		resolvers = append(resolvers, &vaultCredentialResolver{store: deps.VaultStore})
	}
	resolvers = append(resolvers, &envCredentialResolver{})
	if len(resolvers) == 1 {
		credentialResolver = resolvers[0]
	} else if len(resolvers) > 1 {
		credentialResolver = &chainCredentialResolver{resolvers: resolvers}
	}

	var heldStore runtimepkg.HeldExecutionStore
	if deps.ApprovalManager != nil {
		if deps.HeldStore != nil {
			heldStore = NewSQLHeldExecutionStore(deps.HeldStore)
		} else {
			heldStore = NewMemoryHeldExecutionStore()
		}
	}

	commandAllowlist, commandAllowlistErr := collectCommandExecutables(actionRegistry)
	if commandAllowlistErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: command allowlist initialization failed: %v; command actions will remain blocked until services are fixed\n", commandAllowlistErr)
		commandAllowlist = []string{}
	}
	adaptersMap := map[string]adapters.Adapter{
		"http":    adapters.NewHTTPAdapter(nil),
		"command": adapters.NewCommandAdapter(commandAllowlist, 60*time.Second),
	}
	if goruntime.GOOS == "darwin" {
		adaptersMap["applescript"] = adapters.NewAppleScriptAdapter(nil)
	}

	return runtimepkg.NewRuntime(runtimepkg.Runtime{
		ActionRegistry:     actionRegistry,
		PolicyEvaluator:    policyEvaluator,
		CredentialResolver: credentialResolver,
		AuditWriter:        deps.AuditWriter,
		ApprovalManager:    deps.ApprovalManager,
		HeldExecutionStore: heldStore,
		Adapters:           adaptersMap,
	}), nil

}

func collectCommandExecutables(registry *servicesActionRegistry) ([]string, error) {
	executables, err := registry.commandExecutables()
	if err != nil {
		return nil, err
	}
	if len(executables) == 0 {
		return nil, nil
	}
	return executables, nil
}

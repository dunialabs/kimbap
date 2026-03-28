package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/dunialabs/kimbap/internal/policy"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/rs/zerolog/log"
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

	adaptersMap := map[string]adapters.Adapter{
		"http":    adapters.NewHTTPAdapter(nil),
		"command": adapters.NewCommandAdapter(nil, 60*time.Second),
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

type servicesActionRegistry struct {
	installer        *services.LocalInstaller
	verifyMode       string
	signaturePolicy  string
	servicesDir      string
	mu               sync.RWMutex
	cachedDefs       []actions.ActionDefinition
	cacheFingerprint string
}

func (r *servicesActionRegistry) Lookup(_ context.Context, name string) (*actions.ActionDefinition, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("action name is required")
	}
	defs, err := r.loadDefinitions()
	if err != nil {
		return nil, err
	}
	for i := range defs {
		if defs[i].Name == name {
			return &defs[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", actions.ErrLookupNotFound, name)
}

func (r *servicesActionRegistry) List(_ context.Context, opts runtimepkg.ListOptions) ([]actions.ActionDefinition, error) {
	defs, err := r.loadDefinitions()
	if err != nil {
		return nil, err
	}

	namespace := strings.TrimSpace(opts.Namespace)
	resource := strings.TrimSpace(opts.Resource)
	verb := strings.TrimSpace(opts.Verb)
	filtered := make([]actions.ActionDefinition, 0, len(defs))
	for _, def := range defs {
		if namespace != "" && !strings.EqualFold(def.Namespace, namespace) {
			continue
		}
		if resource != "" && !strings.EqualFold(def.Resource, resource) {
			continue
		}
		if verb != "" && !strings.EqualFold(def.Verb, verb) {
			continue
		}
		filtered = append(filtered, def)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})
	if opts.Limit > 0 && len(filtered) > opts.Limit {
		filtered = filtered[:opts.Limit]
	}
	return filtered, nil
}

func (r *servicesActionRegistry) computeFingerprint() string {
	entries, err := os.ReadDir(r.servicesDir)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "%s:%d:%d;", e.Name(), info.Size(), info.ModTime().UnixNano())
	}
	lockPath := filepath.Join(r.servicesDir, "kimbap-services.lock")
	if info, err := os.Stat(lockPath); err == nil {
		fmt.Fprintf(&b, "lock:%d:%d", info.Size(), info.ModTime().UnixNano())
	}
	return b.String()
}

func (r *servicesActionRegistry) loadDefinitions() ([]actions.ActionDefinition, error) {
	if r == nil || r.installer == nil {
		return nil, fmt.Errorf("services installer is not initialized")
	}
	fp := r.computeFingerprint()
	r.mu.RLock()
	if fp != "" && fp == r.cacheFingerprint && r.cachedDefs != nil {
		defs := r.cachedDefs
		r.mu.RUnlock()
		return defs, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if fp != "" && fp == r.cacheFingerprint && r.cachedDefs != nil {
		return r.cachedDefs, nil
	}
	defs, err := r.loadDefinitionsUncached()
	if err != nil {
		return nil, err
	}
	r.cachedDefs = defs
	r.cacheFingerprint = fp
	return defs, nil
}

func (r *servicesActionRegistry) loadDefinitionsUncached() ([]actions.ActionDefinition, error) {
	installed, err := r.installer.ListEnabled()
	if err != nil {
		return nil, err
	}
	out := make([]actions.ActionDefinition, 0)
	for _, it := range installed {
		if ok, verifyErr := r.verifyInstalledService(it.Manifest.Name); !ok {
			if verifyErr != nil {
				return nil, verifyErr
			}
			continue
		}
		defs, convErr := services.ToActionDefinitions(&it.Manifest)
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, defs...)
	}
	return out, nil
}

func (r *servicesActionRegistry) verifyInstalledService(name string) (bool, error) {
	verifyMode := normalizeVerifyMode(r.verifyMode)
	signaturePolicy := normalizeSignaturePolicy(r.signaturePolicy)

	if verifyMode == "off" && signaturePolicy != "required" {
		return true, nil
	}

	result, err := r.installer.Verify(name)
	if err != nil {
		if signaturePolicy == "required" {
			return false, fmt.Errorf("verify installed service %q for required signature policy: %w", name, err)
		}
		if verifyMode == "strict" {
			return false, fmt.Errorf("verify installed service %q: %w", name, err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "warning: verify installed service %q failed: %v\n", name, err)
		return true, nil
	}

	if signaturePolicy == "required" {
		if !result.Locked || !result.Signed || !result.SignatureValid {
			msg := fmt.Sprintf("service %q failed required signature verification (locked=%v signed=%v valid=%v)", name, result.Locked, result.Signed, result.SignatureValid)
			if verifyMode == "strict" {
				return false, fmt.Errorf("%s", msg)
			}
			_, _ = fmt.Fprintln(os.Stderr, "warning:", msg)
			return false, nil
		}
	}

	if verifyMode != "off" {
		if !result.Locked || !result.Verified {
			msg := fmt.Sprintf("service %q failed digest verification (locked=%v verified=%v)", name, result.Locked, result.Verified)
			if verifyMode == "strict" {
				return false, fmt.Errorf("%s", msg)
			}
			_, _ = fmt.Fprintln(os.Stderr, "warning:", msg)
			return true, nil
		}
	}

	return true, nil
}

func normalizeVerifyMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "off", "strict", "warn":
		return normalized
	default:
		return "warn"
	}
}

func normalizeSignaturePolicy(policy string) string {
	normalized := strings.ToLower(strings.TrimSpace(policy))
	switch normalized {
	case "off", "optional", "required":
		return normalized
	default:
		return "optional"
	}
}

type policyEvaluatorAdapter struct {
	evaluator *policy.Evaluator
}

func (a *policyEvaluatorAdapter) Evaluate(ctx context.Context, req runtimepkg.PolicyRequest) (*runtimepkg.PolicyDecision, error) {
	if a == nil || a.evaluator == nil {
		return nil, fmt.Errorf("policy evaluator is not initialized")
	}
	service, actionName := resolveServiceAction(req)

	res, err := a.evaluator.Evaluate(ctx, policy.EvalRequest{
		TenantID:  req.TenantID,
		AgentName: req.Principal.AgentName,
		Service:   service,
		Action:    actionName,
		Risk:      req.Action.Risk.DocVocab(),
		Mutating:  !req.Action.Idempotent,
		Args:      req.Input,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	decision := &runtimepkg.PolicyDecision{
		Decision: string(res.Decision),
		Reason:   res.Reason,
		Meta:     map[string]any{},
	}
	if res.MatchedRule != nil {
		decision.RuleID = res.MatchedRule.ID
	}
	if res.RateStatus != nil {
		decision.Meta["rate_allowed"] = res.RateStatus.Allowed
		decision.Meta["rate_remaining"] = res.RateStatus.Remaining
		decision.Meta["rate_reset_at"] = res.RateStatus.ResetAt
	}
	return decision, nil
}

type cachedPolicyEntry struct {
	eval        *policyEvaluatorAdapter
	fingerprint [32]byte
}

type storePolicyEvaluator struct {
	policyStore store.PolicyStore
	fallback    runtimepkg.PolicyEvaluator

	mu    sync.Mutex
	cache map[string]cachedPolicyEntry
}

func (e *storePolicyEvaluator) Evaluate(ctx context.Context, req runtimepkg.PolicyRequest) (*runtimepkg.PolicyDecision, error) {
	data, err := e.policyStore.GetPolicy(ctx, req.TenantID)
	if err == nil && len(data) > 0 {
		fp := sha256.Sum256(data)

		e.mu.Lock()
		if e.cache == nil {
			e.cache = make(map[string]cachedPolicyEntry)
		}
		if entry, ok := e.cache[req.TenantID]; ok && entry.fingerprint == fp {
			eval := entry.eval
			e.mu.Unlock()
			return eval.Evaluate(ctx, req)
		}
		doc, parseErr := policy.ParseDocument(data)
		if parseErr != nil {
			e.mu.Unlock()
			return nil, fmt.Errorf("parse tenant policy: %w", parseErr)
		}
		newEntry := cachedPolicyEntry{
			eval:        &policyEvaluatorAdapter{evaluator: policy.NewEvaluator(doc)},
			fingerprint: fp,
		}
		e.cache[req.TenantID] = newEntry
		eval := newEntry.eval
		e.mu.Unlock()
		return eval.Evaluate(ctx, req)
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("load tenant policy: %w", err)
	}
	if e.fallback != nil {
		return e.fallback.Evaluate(ctx, req)
	}
	return nil, nil
}

type vaultCredentialResolver struct {
	store vault.Store
}

func (r *vaultCredentialResolver) Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	if r == nil || r.store == nil {
		return nil, fmt.Errorf("vault store is not initialized")
	}
	secretName := strings.TrimSpace(req.CredentialRef)
	if secretName == "" {
		if req.Optional {
			return nil, nil
		}
		return nil, fmt.Errorf("credential_ref is required")
	}

	raw, err := r.store.GetValue(ctx, tenantID, secretName)
	if err != nil {
		if errors.Is(err, vault.ErrSecretNotFound) {
			return nil, nil
		}
		return nil, err
	}
	set := parseCredentialSet(raw, req)
	if set == nil && !req.Optional {
		return nil, fmt.Errorf("credential %q is empty", secretName)
	}
	if set != nil {
		if err := r.store.MarkUsed(ctx, tenantID, secretName); err != nil {
			log.Warn().Err(err).Str("tenantID", tenantID).Str("secret", secretName).Msg("failed to mark credential as used")
		}
	}
	return set, nil
}

func parseCredentialSet(raw []byte, req actions.AuthRequirement) *actions.ResolvedCredentialSet {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil
	}

	var direct actions.ResolvedCredentialSet
	if err := json.Unmarshal(raw, &direct); err == nil {
		if direct.Type == "" {
			direct.Type = string(req.Type)
		}
		return &direct
	}

	set := &actions.ResolvedCredentialSet{Type: string(req.Type)}
	if idx := strings.Index(trimmed, ":"); req.Type == actions.AuthTypeBasic && idx > 0 {
		set.Username = trimmed[:idx]
		set.Password = trimmed[idx+1:]
		return set
	}

	switch req.Type {
	case actions.AuthTypeBearer, actions.AuthTypeOAuth2, actions.AuthTypeSession:
		set.Token = trimmed
	case actions.AuthTypeAPIKey, actions.AuthTypeHeader, actions.AuthTypeQuery, actions.AuthTypeBody:
		set.APIKey = trimmed
		set.Token = trimmed
	case actions.AuthTypeBasic:
		set.Username = trimmed
	default:
		set.Token = trimmed
		set.APIKey = trimmed
	}
	return set
}

// connectorCredentialResolver resolves credentials by looking up stored OAuth
// connections via a long-lived connectors.Manager. This bridges "kimbap auth
// connect" with "kimbap call" — tokens stored by the CLI auth flow become
// usable by the runtime without any manual token export step.
type connectorCredentialResolver struct {
	mgr *connectors.Manager
}

func (r *connectorCredentialResolver) Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	if r == nil || r.mgr == nil {
		return nil, nil
	}
	if req.Type != actions.AuthTypeBearer && req.Type != actions.AuthTypeOAuth2 && req.Type != actions.AuthTypeSession {
		return nil, nil
	}
	connectorName := strings.TrimSpace(req.CredentialRef)
	if connectorName == "" {
		return nil, nil
	}

	connectorName, ok := connectorNameFromRef(connectorName)
	if !ok {
		return nil, nil
	}

	token, err := r.mgr.GetAccessToken(ctx, tenantID, connectorName)
	if err != nil {
		if errors.Is(err, connectors.ErrConnectorNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	return &actions.ResolvedCredentialSet{
		Type:  string(req.Type),
		Token: token,
	}, nil
}

type envCredentialResolver struct{}

func (r *envCredentialResolver) Resolve(_ context.Context, _ string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	if req.Type != actions.AuthTypeBearer && req.Type != actions.AuthTypeOAuth2 && req.Type != actions.AuthTypeSession {
		return nil, nil
	}

	connectorName := strings.TrimSpace(req.CredentialRef)
	if connectorName == "" {
		return nil, nil
	}

	connectorName, ok := connectorNameFromRef(connectorName)
	if !ok {
		return nil, nil
	}

	envKey := "KIMBAP_" + toEnvSegment(connectorName) + "_TOKEN"
	token := strings.TrimSpace(os.Getenv(envKey))
	if token == "" {
		return nil, nil
	}

	return &actions.ResolvedCredentialSet{Type: string(req.Type), Token: token}, nil
}

func connectorNameFromRef(ref string) (string, bool) {
	suffixes := []string{".oauth_token", ".oidc_token", ".token", ".oauth"}
	lower := strings.ToLower(ref)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			name := strings.TrimSpace(ref[:len(ref)-len(suffix)])
			if name != "" {
				return name, true
			}
			return "", false
		}
	}
	return "", false
}

func toEnvSegment(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

type chainCredentialResolver struct {
	resolvers []runtimepkg.CredentialResolver
}

func (c *chainCredentialResolver) Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	for _, r := range c.resolvers {
		creds, err := r.Resolve(ctx, tenantID, req)
		if err != nil {
			return nil, err
		}
		if creds != nil {
			return creds, nil
		}
	}
	return nil, nil
}

type auditWriterAdapter struct {
	writer audit.Writer
}

func NewAuditWriterAdapter(writer audit.Writer) runtimepkg.AuditWriter {
	if writer == nil {
		return nil
	}
	return &auditWriterAdapter{writer: writer}
}

func (a *auditWriterAdapter) Write(ctx context.Context, event runtimepkg.AuditEvent) error {
	if a == nil || a.writer == nil {
		return nil
	}
	service := ""
	action := event.ActionName
	if left, right, ok := strings.Cut(event.ActionName, "."); ok {
		service = left
		action = right
	}

	out := audit.AuditEvent{
		Timestamp:      event.Timestamp,
		RequestID:      event.RequestID,
		TraceID:        event.TraceID,
		TenantID:       event.TenantID,
		PrincipalID:    event.PrincipalID,
		AgentName:      event.AgentName,
		Service:        service,
		Action:         action,
		Mode:           string(event.Mode),
		Status:         mapAuditStatus(event),
		PolicyDecision: event.PolicyDecision,
		DurationMS:     event.DurationMS,
		Meta:           event.Meta,
	}
	if event.ErrorCode != "" || event.ErrorMessage != "" {
		out.Error = &audit.AuditError{Code: event.ErrorCode, Message: event.ErrorMessage}
	}
	return a.writer.Write(ctx, out)
}

func mapAuditStatus(event runtimepkg.AuditEvent) audit.AuditStatus {
	if strings.EqualFold(event.PolicyDecision, "deny") {
		return audit.AuditStatusDenied
	}
	switch event.Status {
	case actions.StatusSuccess:
		return audit.AuditStatusSuccess
	case actions.StatusApprovalRequired:
		return audit.AuditStatusApprovalRequired
	case actions.StatusTimeout:
		return audit.AuditStatusTimeout
	case actions.StatusCancelled:
		return audit.AuditStatusCancelled
	default:
		return audit.AuditStatusError
	}
}

func resolveServiceAction(req runtimepkg.PolicyRequest) (string, string) {
	if req.Classification != nil {
		service := strings.TrimSpace(req.Classification.Service)
		action := strings.TrimSpace(req.Classification.ActionName)
		if service != "" && strings.HasPrefix(action, service+".") {
			action = strings.TrimPrefix(action, service+".")
		}
		if service != "" || action != "" {
			return service, action
		}
	}
	name := strings.TrimSpace(req.Action.Name)
	if left, right, ok := strings.Cut(name, "."); ok {
		return strings.TrimSpace(left), strings.TrimSpace(right)
	}
	return strings.TrimSpace(req.Action.Namespace), name
}

type ApprovalManagerAdapter struct {
	mgr *approvals.ApprovalManager
}

func NewApprovalManagerAdapter(mgr *approvals.ApprovalManager) runtimepkg.ApprovalManager {
	if mgr == nil {
		return nil
	}
	return &ApprovalManagerAdapter{mgr: mgr}
}

func (a *ApprovalManagerAdapter) CreateRequest(ctx context.Context, req runtimepkg.ApprovalRequest) (*runtimepkg.ApprovalResult, error) {
	if a == nil || a.mgr == nil {
		return nil, fmt.Errorf("approval manager unavailable")
	}
	approvalReq := &approvals.ApprovalRequest{
		TenantID:  req.TenantID,
		RequestID: req.RequestID,
		AgentName: req.Principal.AgentName,
		Risk:      req.Action.Risk.DocVocab(),
		Input:     req.Input,
	}
	actionName := strings.TrimSpace(req.Action.Name)
	if actionName != "" {
		if svc, action, ok := strings.Cut(actionName, "."); ok {
			approvalReq.Service = strings.TrimSpace(svc)
			actionName = strings.TrimSpace(action)
		}
	}
	if approvalReq.Service == "" {
		approvalReq.Service = strings.TrimSpace(req.Action.Namespace)
	}
	approvalReq.Action = actionName
	if err := a.mgr.Submit(ctx, approvalReq); err != nil {
		return nil, err
	}

	return &runtimepkg.ApprovalResult{
		Approved:  false,
		RequestID: approvalReq.ID,
		Reason:    "approval pending",
		Meta: map[string]any{
			"approval_id": approvalReq.ID,
			"status":      "pending",
		},
	}, nil
}

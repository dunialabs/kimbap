package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/adapters"
	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/audit"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/policy"
	"github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/dunialabs/kimbap-core/internal/skills"
	"github.com/dunialabs/kimbap-core/internal/vault"
)

type RuntimeDeps struct {
	Config          *config.KimbapConfig
	VaultStore      vault.Store
	PolicyPath      string
	SkillsDir       string
	AuditWriter     runtime.AuditWriter
	ApprovalManager runtime.ApprovalManager
}

func BuildRuntime(deps RuntimeDeps) (*runtime.Runtime, error) {
	if deps.Config == nil {
		return nil, fmt.Errorf("config is required")
	}

	skillsDir := strings.TrimSpace(deps.SkillsDir)
	policyPath := strings.TrimSpace(deps.PolicyPath)
	if skillsDir == "" {
		skillsDir = strings.TrimSpace(deps.Config.Skills.Dir)
	}
	if policyPath == "" {
		policyPath = strings.TrimSpace(deps.Config.Policy.Path)
	}

	actionRegistry := &skillsActionRegistry{installer: skills.NewLocalInstaller(skillsDir)}

	var policyEvaluator runtime.PolicyEvaluator
	if policyPath != "" {
		if stat, err := os.Stat(policyPath); err == nil && !stat.IsDir() {
			doc, parseErr := policy.ParseDocumentFile(policyPath)
			if parseErr != nil {
				return nil, parseErr
			}
			policyEvaluator = &policyEvaluatorAdapter{evaluator: policy.NewEvaluator(doc)}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat policy path: %w", err)
		}
	}

	var credentialResolver runtime.CredentialResolver
	if deps.VaultStore != nil {
		credentialResolver = &vaultCredentialResolver{store: deps.VaultStore}
	}

	return runtime.NewRuntime(runtime.Runtime{
		ActionRegistry:     actionRegistry,
		PolicyEvaluator:    policyEvaluator,
		CredentialResolver: credentialResolver,
		AuditWriter:        deps.AuditWriter,
		ApprovalManager:    deps.ApprovalManager,
		Adapters: map[string]adapters.Adapter{
			"http": adapters.NewHTTPAdapter(nil),
		},
	}), nil
}

type skillsActionRegistry struct {
	installer *skills.LocalInstaller
}

func (r *skillsActionRegistry) Lookup(_ context.Context, name string) (*actions.ActionDefinition, error) {
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
	return nil, fmt.Errorf("action %q not found", name)
}

func (r *skillsActionRegistry) List(_ context.Context, opts runtime.ListOptions) ([]actions.ActionDefinition, error) {
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

func (r *skillsActionRegistry) loadDefinitions() ([]actions.ActionDefinition, error) {
	if r == nil || r.installer == nil {
		return nil, fmt.Errorf("skills installer is not initialized")
	}
	installed, err := r.installer.List()
	if err != nil {
		return nil, err
	}
	out := make([]actions.ActionDefinition, 0)
	for _, it := range installed {
		defs, convErr := skills.ToActionDefinitions(&it.Manifest)
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, defs...)
	}
	return out, nil
}

type policyEvaluatorAdapter struct {
	evaluator *policy.Evaluator
}

func (a *policyEvaluatorAdapter) Evaluate(ctx context.Context, req runtime.PolicyRequest) (*runtime.PolicyDecision, error) {
	if a == nil || a.evaluator == nil {
		return nil, fmt.Errorf("policy evaluator is not initialized")
	}
	service, actionName := resolveServiceAction(req)

	res, err := a.evaluator.Evaluate(ctx, policy.EvalRequest{
		TenantID:  req.TenantID,
		AgentName: req.Principal.AgentName,
		Service:   service,
		Action:    actionName,
		Risk:      string(req.Action.Risk),
		Mutating:  !req.Action.Idempotent,
		Args:      req.Input,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	decision := &runtime.PolicyDecision{
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
		return nil, err
	}
	_ = r.store.MarkUsed(ctx, tenantID, secretName)

	set := parseCredentialSet(raw, req)
	if set == nil && !req.Optional {
		return nil, fmt.Errorf("credential %q is empty", secretName)
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

type auditWriterAdapter struct {
	writer audit.Writer
}

func NewAuditWriterAdapter(writer audit.Writer) runtime.AuditWriter {
	if writer == nil {
		return nil
	}
	return &auditWriterAdapter{writer: writer}
}

func (a *auditWriterAdapter) Write(ctx context.Context, event runtime.AuditEvent) error {
	if a == nil || a.writer == nil {
		return nil
	}
	service := ""
	if left, _, ok := strings.Cut(event.ActionName, "."); ok {
		service = left
	}

	out := audit.AuditEvent{
		Timestamp:      event.Timestamp,
		RequestID:      event.RequestID,
		TraceID:        event.TraceID,
		TenantID:       event.TenantID,
		PrincipalID:    event.PrincipalID,
		Service:        service,
		Action:         event.ActionName,
		Mode:           string(event.Mode),
		Status:         mapAuditStatus(event),
		PolicyDecision: event.PolicyDecision,
		DurationMS:     event.DurationMS,
		Meta:           event.Meta,
	}
	if event.ErrorCode != "" {
		out.Error = &audit.AuditError{Code: event.ErrorCode}
	}
	return a.writer.Write(ctx, out)
}

func mapAuditStatus(event runtime.AuditEvent) audit.AuditStatus {
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
	default:
		return audit.AuditStatusError
	}
}

func resolveServiceAction(req runtime.PolicyRequest) (string, string) {
	if req.Classification != nil {
		service := strings.TrimSpace(req.Classification.Service)
		action := strings.TrimSpace(req.Classification.ActionName)
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

func NewApprovalManagerAdapter(mgr *approvals.ApprovalManager) runtime.ApprovalManager {
	if mgr == nil {
		return nil
	}
	return &ApprovalManagerAdapter{mgr: mgr}
}

func (a *ApprovalManagerAdapter) CreateRequest(ctx context.Context, req runtime.ApprovalRequest) (*runtime.ApprovalResult, error) {
	if a == nil || a.mgr == nil {
		return nil, fmt.Errorf("approval manager unavailable")
	}
	approvalReq := &approvals.ApprovalRequest{
		TenantID:  req.TenantID,
		RequestID: req.RequestID,
		AgentName: req.Principal.AgentName,
		Action:    req.Action.Name,
		Risk:      string(req.Action.Risk),
		Input:     req.Input,
	}
	if req.Action.Name != "" {
		if svc, _, ok := strings.Cut(req.Action.Name, "."); ok {
			approvalReq.Service = svc
		}
	}
	if err := a.mgr.Submit(ctx, approvalReq); err != nil {
		return nil, err
	}

	return &runtime.ApprovalResult{
		Approved:  false,
		RequestID: approvalReq.ID,
		Reason:    "approval pending",
		Meta: map[string]any{
			"approval_id": approvalReq.ID,
			"status":      "pending",
		},
	}, nil
}

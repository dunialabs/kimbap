package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/auth"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/policy"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/storeconv"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/dunialabs/kimbap/internal/webhooks"
	"github.com/go-chi/chi/v5"
)

type createTokenRequest struct {
	TenantID   string   `json:"tenant_id"`
	AgentName  string   `json:"agent_name"`
	Scopes     []string `json:"scopes"`
	TTLSeconds int64    `json:"ttl_seconds"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, r, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": config.CLIVersion(),
	})
}

func (s *Server) handleListActions(w http.ResponseWriter, r *http.Request) {
	limit, err := parseNonNegativeIntParam(r.URL.Query().Get("limit"), "limit")
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	if s.runtime == nil || s.runtime.ActionRegistry == nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "action registry unavailable", http.StatusInternalServerError, false, nil))
		return
	}
	defs, err := s.runtime.ActionRegistry.List(r.Context(), runtimepkg.ListOptions{
		Namespace: r.URL.Query().Get("namespace"),
		Resource:  r.URL.Query().Get("resource"),
		Verb:      r.URL.Query().Get("verb"),
		Limit:     limit,
	})
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeSuccess(w, r, http.StatusOK, toPublicActionDefinitions(defs))
}

func (s *Server) handleDescribeAction(w http.ResponseWriter, r *http.Request) {
	if s.runtime == nil || s.runtime.ActionRegistry == nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "action registry unavailable", http.StatusInternalServerError, false, nil))
		return
	}
	name := chi.URLParam(r, "service") + "." + chi.URLParam(r, "action")
	def, err := s.runtime.ActionRegistry.Lookup(r.Context(), name)
	if err != nil {
		if errors.Is(err, actions.ErrLookupNotFound) {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, "action not found", http.StatusNotFound, false, map[string]any{"action": name}))
		} else {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "action lookup failed", http.StatusInternalServerError, true, map[string]any{"action": name}))
		}
		return
	}
	if def == nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, "action not found", http.StatusNotFound, false, map[string]any{"action": name}))
		return
	}
	writeSuccess(w, r, http.StatusOK, toPublicActionDefinition(*def))
}

type publicAuthRequirement struct {
	Type string `json:"Type"`
}

type publicActionDefinition struct {
	Name         string                `json:"Name"`
	Version      int                   `json:"Version"`
	DisplayName  string                `json:"DisplayName"`
	Namespace    string                `json:"Namespace"`
	Verb         string                `json:"Verb"`
	Resource     string                `json:"Resource"`
	Description  string                `json:"Description"`
	Risk         actions.RiskLevel     `json:"Risk"`
	Idempotent   bool                  `json:"Idempotent"`
	ApprovalHint actions.ApprovalHint  `json:"ApprovalHint"`
	Auth         publicAuthRequirement `json:"Auth"`
	InputSchema  *actions.Schema       `json:"InputSchema,omitempty"`
}

func toPublicActionDefinitions(defs []actions.ActionDefinition) []publicActionDefinition {
	out := make([]publicActionDefinition, 0, len(defs))
	for _, def := range defs {
		out = append(out, toPublicActionDefinition(def))
	}
	return out
}

func toPublicActionDefinition(def actions.ActionDefinition) publicActionDefinition {
	authType := string(def.Auth.Type)
	if authType == "" {
		authType = string(actions.AuthTypeNone)
	}
	return publicActionDefinition{
		Name:         def.Name,
		Version:      def.Version,
		DisplayName:  def.DisplayName,
		Namespace:    def.Namespace,
		Verb:         def.Verb,
		Resource:     def.Resource,
		Description:  def.Description,
		Risk:         def.Risk,
		Idempotent:   def.Idempotent,
		ApprovalHint: def.ApprovalHint,
		Auth: publicAuthRequirement{
			Type: authType,
		},
		InputSchema: def.InputSchema,
	}
}

func (s *Server) handleExecuteAction(w http.ResponseWriter, r *http.Request) {
	if s.runtime == nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "runtime pipeline unavailable", http.StatusInternalServerError, false, nil))
		return
	}
	var body struct {
		Input map[string]any `json:"input"`
	}
	if !decodeJSONOrWriteError(w, r, &body) {
		return
	}
	principal, tenantID, ok := requirePrincipalWithTenantContext(w, r)
	if !ok {
		return
	}
	requestID := requestIDFromContext(r.Context())
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	req := actions.ExecutionRequest{
		RequestID:      requestID,
		IdempotencyKey: idempotencyKey,
		TenantID:       tenantID,
		Principal: actions.Principal{
			ID:        principal.ID,
			TenantID:  tenantID,
			AgentName: principal.AgentName,
			Type:      string(principal.Type),
			Scopes:    append([]string(nil), principal.Scopes...),
		},
		Action: actions.ActionDefinition{Name: chi.URLParam(r, "service") + "." + chi.URLParam(r, "action")},
		Input:  body.Input,
		Mode:   actions.ModeServe,
	}
	result := s.runtime.Execute(r.Context(), req)
	if result.Error != nil {
		s.emitApprovalRequestedWebhook(req, result)
		writeEnvelopeError(w, r, result.Error)
		return
	}
	status := result.HTTPStatus
	if status <= 0 || status == http.StatusNoContent {
		status = http.StatusOK
	}
	writeSuccess(w, r, status, result)
}

func (s *Server) handleValidateAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Schema *actions.Schema `json:"schema"`
		Input  map[string]any  `json:"input"`
	}
	if !decodeJSONOrWriteError(w, r, &body) {
		return
	}
	if verr := actions.ValidateInput(body.Schema, body.Input); verr != nil {
		writeEnvelopeError(w, r, verr)
		return
	}
	writeSuccess(w, r, http.StatusOK, map[string]any{"valid": true})
}

func (s *Server) handleListVaultKeys(w http.ResponseWriter, r *http.Request) {
	if s.vaultStore == nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "vault store unavailable", http.StatusInternalServerError, false, nil))
		return
	}
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	limit, err := parseNonNegativeIntParam(r.URL.Query().Get("limit"), "limit")
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	offset, err := parseNonNegativeIntParam(r.URL.Query().Get("offset"), "offset")
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	listOpts := vault.ListOptions{Limit: limit, Offset: offset}
	if secretType := strings.TrimSpace(r.URL.Query().Get("type")); secretType != "" {
		t := vault.SecretType(secretType)
		switch t {
		case vault.SecretTypeAPIKey, vault.SecretTypeBearerToken, vault.SecretTypeOAuthClient,
			vault.SecretTypePassword, vault.SecretTypeRefreshToken, vault.SecretTypeCertificate:
			listOpts.Type = &t
		default:
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "unknown secret type", http.StatusBadRequest, false, nil))
			return
		}
	}
	records, listErr := s.vaultStore.List(r.Context(), tenantID, listOpts)
	if listErr != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeSuccess(w, r, http.StatusOK, map[string]any{"items": records})
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	if s.tokenService == nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "token service unavailable", http.StatusInternalServerError, false, nil))
		return
	}
	var req createTokenRequest
	if !decodeJSONOrWriteError(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.AgentName) == "" {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "agent_name is required", http.StatusBadRequest, false, nil))
		return
	}
	principal, tenantID, ok := requirePrincipalWithTenantContext(w, r)
	if !ok {
		return
	}
	requestedTenantID := strings.TrimSpace(req.TenantID)
	if requestedTenantID != "" && requestedTenantID != tenantID {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthorized, "tenant_id does not match authenticated tenant context", http.StatusForbidden, false, nil))
		return
	}
	// Prevent scope escalation: if the caller has restricted scopes and requests
	// none, inherit the caller's scopes rather than issuing an unrestricted token.
	issuedScopes := req.Scopes
	if len(issuedScopes) == 0 && len(principal.Scopes) > 0 {
		issuedScopes = append([]string(nil), principal.Scopes...)
	}
	for _, requested := range issuedScopes {
		if !principalHasScope(principal, requested) {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthorized,
				"cannot mint token with scope not held by caller: "+requested,
				http.StatusForbidden, false, nil))
			return
		}
	}
	ttl := 30 * 24 * time.Hour
	if req.TTLSeconds != 0 {
		if req.TTLSeconds < 0 || req.TTLSeconds > maxCreateTokenTTLSeconds {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "ttl_seconds must be between 1 and 31536000", http.StatusBadRequest, false, nil))
			return
		}
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}
	raw, issued, err := s.tokenService.Issue(r.Context(), tenantID, req.AgentName, principal.ID, issuedScopes, ttl)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidTTL) {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
			return
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	if s.webhookDispatcher != nil {
		s.webhookDispatcher.EmitForTenant(issued.TenantID, webhooks.EventTokenCreated, map[string]any{
			"token_id":   issued.ID,
			"tenant_id":  issued.TenantID,
			"agent_name": issued.AgentName,
			"expires_at": issued.ExpiresAt,
		})
	}
	writeSuccess(w, r, http.StatusCreated, map[string]any{"token": raw, "metadata": issued})
}

type tokenView struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	AgentName   string     `json:"agent_name"`
	DisplayHint string     `json:"display_hint"`
	Scopes      []string   `json:"scopes"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedBy   string     `json:"created_by"`
}

func toTokenView(t store.TokenRecord) tokenView {
	return tokenView{
		ID:          t.ID,
		TenantID:    t.TenantID,
		AgentName:   t.AgentName,
		DisplayHint: t.DisplayHint,
		Scopes:      storeconv.ParseScopes(t.Scopes),
		CreatedAt:   t.CreatedAt,
		ExpiresAt:   t.ExpiresAt,
		LastUsedAt:  t.LastUsedAt,
		RevokedAt:   t.RevokedAt,
		CreatedBy:   t.CreatedBy,
	}
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListTokens(r.Context(), tenantID)
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	views := make([]tokenView, len(items))
	for i, it := range items {
		views[i] = toTokenView(it)
	}
	writeSuccess(w, r, http.StatusOK, views)
}

func (s *Server) handleInspectToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	tok, ok := s.requireTokenForTenant(w, r, id, tenantID)
	if !ok {
		return
	}
	writeSuccess(w, r, http.StatusOK, toTokenView(*tok))
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	tok, ok := s.requireTokenForTenant(w, r, id, tenantID)
	if !ok {
		return
	}
	if err := s.store.RevokeToken(r.Context(), tok.ID); err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	if s.webhookDispatcher != nil {
		s.webhookDispatcher.EmitForTenant(tok.TenantID, webhooks.EventTokenDeleted, map[string]any{
			"token_id":   tok.ID,
			"tenant_id":  tok.TenantID,
			"agent_name": tok.AgentName,
		})
	}
	writeSuccess(w, r, http.StatusOK, map[string]any{"revoked": true})
}

func (s *Server) requireTokenForTenant(w http.ResponseWriter, r *http.Request, tokenID, tenantID string) (*store.TokenRecord, bool) {
	if strings.TrimSpace(tenantID) == "" {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "tenant context required", http.StatusUnauthorized, false, nil))
		return nil, false
	}
	tok, err := s.store.GetToken(r.Context(), tokenID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, "token not found", http.StatusNotFound, false, nil))
			return nil, false
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return nil, false
	}
	if strings.TrimSpace(tok.TenantID) != strings.TrimSpace(tenantID) {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, "token not found", http.StatusNotFound, false, nil))
		return nil, false
	}
	return tok, true
}

func principalTenantMatches(principal *auth.Principal, tenantID string) bool {
	if principal == nil {
		return false
	}
	principalTenant := strings.TrimSpace(principal.TenantID)
	if principalTenant == "" {
		return true
	}
	return principalTenant == strings.TrimSpace(tenantID)
}

func requirePrincipalWithTenantContext(w http.ResponseWriter, r *http.Request) (*auth.Principal, string, bool) {
	principal := principalFromContext(r.Context())
	if principal == nil {
		setBearerAuthHeader(w, "invalid_token", "authentication required", "")
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "authentication required", http.StatusUnauthorized, false, nil))
		return nil, "", false
	}
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return nil, "", false
	}
	if !principalTenantMatches(principal, tenantID) {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthorized, "principal tenant does not match tenant context", http.StatusForbidden, false, nil))
		return nil, "", false
	}
	return principal, tenantID, true
}

func requireTenantContext(w http.ResponseWriter, r *http.Request) (string, bool) {
	tenantID := strings.TrimSpace(tenantFromContext(r.Context()))
	if tenantID == "" {
		setBearerAuthHeader(w, "invalid_token", "tenant context required", "")
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "tenant context required", http.StatusUnauthorized, false, nil))
		return "", false
	}
	return tenantID, true
}

type tenantPolicyCacheInvalidator interface {
	InvalidateTenantPolicyCache(tenantID string)
}

func (s *Server) invalidateTenantPolicyCache(tenantID string) {
	if s == nil || s.runtime == nil || s.runtime.PolicyEvaluator == nil {
		return
	}
	if invalidator, ok := s.runtime.PolicyEvaluator.(tenantPolicyCacheInvalidator); ok {
		invalidator.InvalidateTenantPolicyCache(tenantID)
	}
}

func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	doc, err := s.store.GetPolicy(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, sanitizeErrMsg(err, http.StatusNotFound), http.StatusNotFound, false, nil))
		} else {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		}
		return
	}
	writeSuccess(w, r, http.StatusOK, map[string]any{"tenant_id": tenantID, "document": string(doc)})
}

func (s *Server) handleSetPolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Document string `json:"document"`
	}
	if !decodeJSONOrWriteError(w, r, &body) {
		return
	}
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	currentPolicy, getErr := s.store.GetPolicy(r.Context(), tenantID)
	policyEvent := webhooks.EventPolicyUpdated
	if getErr != nil {
		if errors.Is(getErr, store.ErrNotFound) {
			policyEvent = webhooks.EventPolicyCreated
		} else {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return
		}
	} else if len(currentPolicy) == 0 {
		policyEvent = webhooks.EventPolicyCreated
	}
	if _, parseErr := policy.ParseDocument([]byte(body.Document)); parseErr != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, fmt.Sprintf("invalid policy document: %v", parseErr), http.StatusBadRequest, false, nil))
		return
	}
	if err := s.store.SetPolicy(r.Context(), tenantID, []byte(body.Document)); err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	s.invalidateTenantPolicyCache(tenantID)
	if s.webhookDispatcher != nil {
		s.webhookDispatcher.EmitForTenant(tenantID, policyEvent, map[string]any{
			"tenant_id": tenantID,
		})
	}
	writeSuccess(w, r, http.StatusOK, map[string]any{"updated": true})
}

func (s *Server) handleEvalPolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AgentName string         `json:"agent_name"`
		Service   string         `json:"service"`
		Action    string         `json:"action"`
		Risk      string         `json:"risk"`
		Mutating  bool           `json:"mutating"`
		Args      map[string]any `json:"args"`
	}
	if !decodeJSONOrWriteError(w, r, &body) {
		return
	}
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	docBytes, err := s.store.GetPolicy(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, sanitizeErrMsg(err, http.StatusNotFound), http.StatusNotFound, false, nil))
		} else {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		}
		return
	}
	doc, err := policy.ParseDocument(docBytes)
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	evaluator := policy.NewEvaluator(doc)
	result, err := evaluator.Evaluate(r.Context(), policy.EvalRequest{TenantID: tenantID, AgentName: body.AgentName, Service: body.Service, Action: body.Action, Risk: body.Risk, Mutating: body.Mutating, Args: body.Args})
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeSuccess(w, r, http.StatusOK, result)
}

func (s *Server) emitApprovalRequestedWebhook(req actions.ExecutionRequest, result actions.ExecutionResult) {
	if s.webhookDispatcher == nil || result.Status != actions.StatusApprovalRequired {
		return
	}
	approvalID, ok := result.Meta["approval_request_id"].(string)
	if !ok || strings.TrimSpace(approvalID) == "" {
		return
	}
	service, action, hasDot := strings.Cut(req.Action.Name, ".")
	service = strings.TrimSpace(service)
	action = strings.TrimSpace(action)
	if !hasDot {
		service = strings.TrimSpace(req.Action.Namespace)
		action = strings.TrimSpace(req.Action.Name)
	}
	s.webhookDispatcher.EmitForTenant(req.TenantID, webhooks.EventApprovalRequested, map[string]any{
		"approval_id": approvalID,
		"tenant_id":   req.TenantID,
		"request_id":  req.RequestID,
		"agent_name":  req.Principal.AgentName,
		"service":     service,
		"action":      action,
		"status":      "pending",
	})
}

func (s *Server) handleQueryAudit(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	limit, err := parseNonNegativeIntParam(r.URL.Query().Get("limit"), "limit")
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	offset, err := parseNonNegativeIntParam(r.URL.Query().Get("offset"), "offset")
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	from, fromErr := parseRFC3339Ptr(r.URL.Query().Get("from"))
	if fromErr != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "invalid 'from' timestamp: expected RFC3339 format", http.StatusBadRequest, false, nil))
		return
	}
	to, toErr := parseRFC3339Ptr(r.URL.Query().Get("to"))
	if toErr != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "invalid 'to' timestamp: expected RFC3339 format", http.StatusBadRequest, false, nil))
		return
	}
	items, err := s.store.QueryAuditEvents(r.Context(), store.AuditFilter{
		TenantID:  tenantID,
		AgentName: r.URL.Query().Get("agent_name"),
		Service:   r.URL.Query().Get("service"),
		Action:    r.URL.Query().Get("action"),
		Status:    r.URL.Query().Get("status"),
		From:      from,
		To:        to,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeSuccess(w, r, http.StatusOK, items)
}

func (s *Server) handleExportAudit(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	format := strings.TrimSpace(r.URL.Query().Get("format"))
	var contentType string
	switch strings.ToLower(format) {
	case "", "jsonl", "ndjson":
		contentType = "application/x-ndjson"
		format = "jsonl"
	case "csv":
		contentType = "text/csv"
	default:
		writeEnvelopeError(w, r, actions.NewExecutionError(
			actions.ErrValidationFailed,
			fmt.Sprintf("unsupported export format %q; accepted values: jsonl, ndjson, csv", format),
			http.StatusBadRequest, false, nil,
		))
		return
	}
	from, fromErr := parseRFC3339Ptr(r.URL.Query().Get("from"))
	if fromErr != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "invalid 'from' timestamp: expected RFC3339 format", http.StatusBadRequest, false, nil))
		return
	}
	to, toErr := parseRFC3339Ptr(r.URL.Query().Get("to"))
	if toErr != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "invalid 'to' timestamp: expected RFC3339 format", http.StatusBadRequest, false, nil))
		return
	}
	if from == nil && to == nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "audit export requires at least one of 'from' or 'to' to bound the result set", http.StatusBadRequest, false, nil))
		return
	}
	exportFilter := store.AuditFilter{
		TenantID:  tenantID,
		AgentName: r.URL.Query().Get("agent_name"),
		Service:   r.URL.Query().Get("service"),
		Action:    r.URL.Query().Get("action"),
		Status:    r.URL.Query().Get("status"),
		From:      from,
		To:        to,
	}
	w.Header().Set("Content-Type", contentType)
	stream := &lazyStatusWriter{w: w, statusCode: http.StatusOK}
	if err := s.store.ExportAuditEvents(r.Context(), exportFilter, format, stream); err != nil {
		if stream.wrote {
			return
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "export failed", http.StatusInternalServerError, false, nil))
		return
	}
	if !stream.wrote {
		w.WriteHeader(http.StatusOK)
	}
}

type lazyStatusWriter struct {
	w          http.ResponseWriter
	statusCode int
	wrote      bool
}

func (w *lazyStatusWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.w.WriteHeader(w.statusCode)
		w.wrote = true
	}
	return w.w.Write(p)
}

const maxAPIRequestBodyBytes int64 = 4 << 20
const maxCreateTokenTTLSeconds int64 = int64((365 * 24 * time.Hour) / time.Second)

var errRequestBodyTooLarge = errors.New("request body too large")

func decodeJSON(r *http.Request, out any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxAPIRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			return errRequestBodyTooLarge
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("invalid JSON at offset %d", syntaxErr.Offset)
		case errors.As(err, &typeErr):
			return fmt.Errorf("invalid type for field %q", typeErr.Field)
		case strings.Contains(err.Error(), "unknown field"):
			return err
		default:
			return errors.New("invalid request body")
		}
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		var maxErr2 *http.MaxBytesError
		if errors.As(err, &maxErr2) {
			return errRequestBodyTooLarge
		}
		return errors.New("unexpected trailing content after JSON body")
	}
	return nil
}

func decodeJSONOrWriteError(w http.ResponseWriter, r *http.Request, out any) bool {
	if err := decodeJSON(r, out); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), status, false, nil))
		return false
	}
	return true
}

func parseRFC3339Ptr(v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &ts, nil
}

func parseNonNegativeIntParam(raw string, field string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", field)
	}
	return value, nil
}

func mapApprovalError(err error) *actions.ExecutionError {
	switch {
	case errors.Is(err, approvals.ErrAlreadyResolved), errors.Is(err, store.ErrApprovalAlreadyResolved):
		return actions.NewExecutionError(actions.ErrValidationFailed, "approval already resolved", http.StatusConflict, false, nil)
	case errors.Is(err, approvals.ErrExpired), errors.Is(err, store.ErrApprovalExpired):
		return actions.NewExecutionError(actions.ErrApprovalTimeout, "approval has expired", http.StatusGone, false, nil)
	case errors.Is(err, approvals.ErrDuplicateVote), errors.Is(err, store.ErrApprovalDuplicateVote):
		return actions.NewExecutionError(actions.ErrValidationFailed, "approver has already voted", http.StatusConflict, false, nil)
	case errors.Is(err, approvals.ErrNotFound), errors.Is(err, store.ErrNotFound):
		return actions.NewExecutionError(actions.ErrResourceNotFound, "approval not found", http.StatusNotFound, false, nil)
	default:
		return actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil)
	}
}

func sanitizeErrMsg(err error, status int) string {
	if status == http.StatusInternalServerError {
		return "internal server error"
	}
	return err.Error()
}

func (s *Server) removeHeldExecution(ctx context.Context, approvalRequestID string) {
	removeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if s.runtime != nil && s.runtime.HeldExecutionStore != nil {
		_ = s.runtime.HeldExecutionStore.Remove(removeCtx, approvalRequestID)
		return
	}
	if executionStore, ok := s.store.(interface {
		RemoveExecution(context.Context, string) error
	}); ok {
		_ = executionStore.RemoveExecution(removeCtx, approvalRequestID)
	}
}

//go:build ignore

package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/auth"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/policy"
	runtimepkg "github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/dunialabs/kimbap-core/internal/store"
	"github.com/dunialabs/kimbap-core/internal/vault"
	"github.com/dunialabs/kimbap-core/internal/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
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
		"version": strings.TrimSpace(config.AppInfo.Version),
	})
}

func (s *Server) handleListActions(w http.ResponseWriter, r *http.Request) {
	limit, err := parseNonNegativeIntParam(r.URL.Query().Get("limit"), "limit")
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	defs := make([]actions.ActionDefinition, 0)
	if s.runtime != nil && s.runtime.ActionRegistry != nil {
		items, err := s.runtime.ActionRegistry.List(r.Context(), runtimepkg.ListOptions{
			Namespace: r.URL.Query().Get("namespace"),
			Resource:  r.URL.Query().Get("resource"),
			Verb:      r.URL.Query().Get("verb"),
			Limit:     limit,
		})
		if err != nil {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return
		}
		defs = items
	}
	writeSuccess(w, r, http.StatusOK, defs)
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
	writeSuccess(w, r, http.StatusOK, def)
}

func (s *Server) handleExecuteAction(w http.ResponseWriter, r *http.Request) {
	if s.runtime == nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "runtime pipeline unavailable", http.StatusInternalServerError, false, nil))
		return
	}
	var body struct {
		Input map[string]any `json:"input"`
	}
	if err := decodeJSON(r, &body); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), status, false, nil))
		return
	}
	principal, tenantID, ok := requirePrincipalWithTenantContext(w, r)
	if !ok {
		return
	}
	requestID := requestIDFromContext(r.Context())
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = requestID
	}
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
		Mode:   actions.ModeCall,
	}
	result := s.runtime.Execute(r.Context(), req)
	if result.Error != nil {
		s.emitApprovalRequestedWebhook(req, result)
		writeEnvelopeError(w, r, result.Error)
		return
	}
	writeSuccess(w, r, http.StatusOK, result)
}

func (s *Server) handleValidateAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Schema *actions.Schema `json:"schema"`
		Input  map[string]any  `json:"input"`
	}
	if err := decodeJSON(r, &body); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), status, false, nil))
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
	if err := decodeJSON(r, &req); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), status, false, nil))
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
	if len(req.Scopes) > 0 {
		for _, requested := range req.Scopes {
			if !principalHasScope(principal, requested) {
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthorized,
					"cannot mint token with scope not held by caller: "+requested,
					http.StatusForbidden, false, nil))
				return
			}
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
	raw, issued, err := s.tokenService.Issue(r.Context(), tenantID, req.AgentName, req.Scopes, ttl)
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
	writeSuccess(w, r, http.StatusOK, items)
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
	writeSuccess(w, r, http.StatusOK, tok)
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
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, "token not found", http.StatusNotFound, false, nil))
			return nil, false
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return nil, false
	}
	if strings.TrimSpace(tok.TenantID) != strings.TrimSpace(tenantID) {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, "token not found", http.StatusNotFound, false, nil))
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

func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	doc, err := s.store.GetPolicy(r.Context(), tenantID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
		return
	}
	writeSuccess(w, r, http.StatusOK, map[string]any{"tenant_id": tenantID, "document": string(doc)})
}

func (s *Server) handleSetPolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Document string `json:"document"`
	}
	if err := decodeJSON(r, &body); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), status, false, nil))
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
	if err := s.store.SetPolicy(r.Context(), tenantID, []byte(body.Document)); err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
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
	if err := decodeJSON(r, &body); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), status, false, nil))
		return
	}
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	docBytes, err := s.store.GetPolicy(r.Context(), tenantID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
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

func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" || status == "pending" {
		if expirer, ok := s.store.(interface {
			ExpirePendingApprovals(ctx context.Context) (int, error)
		}); ok {
			if _, err := expirer.ExpirePendingApprovals(r.Context()); err != nil {
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
				return
			}
		}
	}
	items, err := s.store.ListApprovals(r.Context(), tenantID, status)
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeSuccess(w, r, http.StatusOK, items)
}

func (s *Server) requirePendingApproval(w http.ResponseWriter, r *http.Request, id, tenantID string) (*store.ApprovalRecord, bool) {
	existing, err := s.store.GetApproval(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, sanitizeErrMsg(err, status), status, false, nil))
		return nil, false
	}
	if existing.TenantID != tenantID {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, "approval not found", http.StatusNotFound, false, nil))
		return nil, false
	}
	now := time.Now().UTC()
	if existing.Status == "expired" {
		s.removeHeldExecution(r.Context(), id)
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrApprovalTimeout, "approval expired", http.StatusGone, false, nil))
		return nil, false
	}
	if !existing.ExpiresAt.After(now) {
		expirer, ok := s.store.(interface {
			ExpireApproval(ctx context.Context, id string) (bool, error)
		})
		if !ok {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return nil, false
		}
		expired, expireErr := expirer.ExpireApproval(r.Context(), id)
		if expireErr != nil {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return nil, false
		}
		if !expired {
			refreshed, getErr := s.store.GetApproval(r.Context(), id)
			if getErr != nil {
				if errors.Is(getErr, store.ErrNotFound) {
					writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, "approval not found", http.StatusNotFound, false, nil))
				} else {
					writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
				}
				return nil, false
			}
			if refreshed == nil {
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrActionNotFound, "approval not found", http.StatusNotFound, false, nil))
				return nil, false
			}
			if refreshed.Status != "expired" {
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "approval already resolved", http.StatusConflict, false, nil))
				return nil, false
			}
			existing = refreshed
		}
		if s.webhookDispatcher != nil && expired {
			s.webhookDispatcher.EmitForTenant(tenantID, webhooks.EventApprovalExpired, map[string]any{
				"approval_id": id,
				"tenant_id":   tenantID,
				"request_id":  existing.RequestID,
				"agent_name":  existing.AgentName,
				"service":     existing.Service,
				"action":      existing.Action,
				"status":      "expired",
			})
		}
		s.removeHeldExecution(r.Context(), id)
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrApprovalTimeout, "approval expired", http.StatusGone, false, nil))
		return nil, false
	}
	if existing.Status != "pending" {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "approval already resolved", http.StatusConflict, false, nil))
		return nil, false
	}
	return existing, true
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	principal, tenantID, ok := requirePrincipalWithTenantContext(w, r)
	if !ok {
		return
	}

	existing, pending := s.requirePendingApproval(w, r, id, tenantID)
	if !pending {
		return
	}

	if err := s.store.UpdateApprovalStatus(r.Context(), id, "approved", principal.ID, "approved via api"); err != nil {
		writeEnvelopeError(w, r, mapApprovalError(err))
		return
	}
	if s.webhookDispatcher != nil {
		s.webhookDispatcher.EmitForTenant(tenantID, webhooks.EventApprovalApproved, map[string]any{
			"approval_id": id,
			"tenant_id":   tenantID,
			"request_id":  existing.RequestID,
			"agent_name":  existing.AgentName,
			"service":     existing.Service,
			"action":      existing.Action,
			"resolved_by": principal.ID,
			"status":      "approved",
		})
	}

	var execResult *actions.ExecutionResult
	if s.runtime != nil {
		result := s.runtime.ResumeApproved(r.Context(), id)
		execResult = &result
	}

	resp := map[string]any{"approved": true}
	if execResult != nil {
		execMap := map[string]any{
			"status":      execResult.Status,
			"http_status": execResult.HTTPStatus,
		}
		if execResult.Output != nil {
			execMap["output"] = execResult.Output
		}
		if execResult.Error != nil {
			msg := execResult.Error.Message
			if execResult.HTTPStatus >= 500 {
				msg = "internal server error"
			}
			execMap["error"] = map[string]any{
				"code":      execResult.Error.Code,
				"message":   msg,
				"retryable": execResult.Error.Retryable,
			}
		}
		resp["execution"] = execMap
	}
	writeSuccess(w, r, http.StatusOK, resp)
}

func (s *Server) handleDeny(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	principal, tenantID, ok := requirePrincipalWithTenantContext(w, r)
	if !ok {
		return
	}

	existing, pending := s.requirePendingApproval(w, r, id, tenantID)
	if !pending {
		return
	}

	if err := s.store.UpdateApprovalStatus(r.Context(), id, "denied", principal.ID, "denied via api"); err != nil {
		writeEnvelopeError(w, r, mapApprovalError(err))
		return
	}
	s.removeHeldExecution(r.Context(), id)
	if s.webhookDispatcher != nil {
		s.webhookDispatcher.EmitForTenant(tenantID, webhooks.EventApprovalDenied, map[string]any{
			"approval_id": id,
			"tenant_id":   tenantID,
			"request_id":  existing.RequestID,
			"agent_name":  existing.AgentName,
			"service":     existing.Service,
			"action":      existing.Action,
			"resolved_by": principal.ID,
			"status":      "denied",
		})
	}
	writeSuccess(w, r, http.StatusOK, map[string]any{"denied": true})
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
	case "", "json", "jsonl", "ndjson":
		contentType = "application/x-ndjson"
		format = "jsonl"
	case "csv":
		contentType = "text/csv"
	default:
		writeEnvelopeError(w, r, actions.NewExecutionError(
			actions.ErrValidationFailed,
			fmt.Sprintf("unsupported export format %q; accepted values: json, jsonl, csv", format),
			http.StatusBadRequest, false, nil,
		))
		return
	}
	var buf bytes.Buffer
	err := s.store.ExportAuditEvents(r.Context(), store.AuditFilter{TenantID: tenantID}, format, &buf)
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	if written, copyErr := io.Copy(w, &buf); copyErr != nil {
		log.Warn().Err(copyErr).Str("requestId", requestIDFromContext(r.Context())).Str("tenantId", tenantID).Str("format", format).Int64("bytesWritten", written).Msg("failed to write audit export response body")
	}
}

const maxAPIRequestBodyBytes int64 = 4 << 20
const maxCreateTokenTTLSeconds int64 = int64((365 * 24 * time.Hour) / time.Second)

var errRequestBodyTooLarge = errors.New("request body too large")

func decodeJSON(r *http.Request, out any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	limited := io.LimitReader(r.Body, maxAPIRequestBodyBytes+1)
	raw, readErr := io.ReadAll(limited)
	if readErr != nil {
		return errors.New("failed to read request body")
	}
	if int64(len(raw)) > maxAPIRequestBodyBytes {
		return errRequestBodyTooLarge
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		switch {
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
	if dec.Decode(&extra) != io.EOF {
		return errors.New("unexpected trailing content after JSON body")
	}
	return nil
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

type storeTokenAdapter struct {
	st store.Store
}

func (a *storeTokenAdapter) Create(ctx context.Context, token *auth.ServiceToken) error {
	if token == nil {
		return errors.New("token is required")
	}
	return a.st.CreateToken(ctx, &store.TokenRecord{
		ID:          token.ID,
		TenantID:    token.TenantID,
		AgentName:   token.AgentName,
		TokenHash:   token.TokenHash,
		DisplayHint: token.DisplayHint,
		Scopes:      marshalScopes(token.Scopes),
		CreatedAt:   token.CreatedAt,
		ExpiresAt:   token.ExpiresAt,
		LastUsedAt:  token.LastUsedAt,
		RevokedAt:   token.RevokedAt,
		CreatedBy:   token.CreatedBy,
	})
}

func (a *storeTokenAdapter) ValidateAndResolve(ctx context.Context, rawToken string) (*auth.Principal, error) {
	hash := sha256.Sum256([]byte(rawToken))
	token, err := a.st.GetTokenByHash(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}
	now := time.Now().UTC()
	if token.RevokedAt != nil {
		return nil, auth.ErrRevokedToken
	}
	if now.After(token.ExpiresAt) {
		return nil, auth.ErrExpiredToken
	}
	return &auth.Principal{
		ID:        token.AgentName,
		Type:      auth.PrincipalTypeService,
		TenantID:  token.TenantID,
		AgentName: token.AgentName,
		Scopes:    parseScopes(token.Scopes),
		TokenID:   token.ID,
		IssuedAt:  token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
	}, nil
}

func (a *storeTokenAdapter) Revoke(ctx context.Context, tokenID string) error {
	if strings.TrimSpace(tokenID) == "" {
		return auth.ErrInvalidToken
	}
	err := a.st.RevokeToken(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
}

func (a *storeTokenAdapter) List(ctx context.Context, tenantID string) ([]auth.ServiceToken, error) {
	items, err := a.st.ListTokens(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]auth.ServiceToken, 0, len(items))
	for i := range items {
		it := items[i]
		out = append(out, auth.ServiceToken{
			ID:          it.ID,
			TenantID:    it.TenantID,
			AgentName:   it.AgentName,
			TokenHash:   it.TokenHash,
			DisplayHint: it.DisplayHint,
			Scopes:      parseScopes(it.Scopes),
			CreatedAt:   it.CreatedAt,
			ExpiresAt:   it.ExpiresAt,
			LastUsedAt:  it.LastUsedAt,
			RevokedAt:   it.RevokedAt,
			CreatedBy:   it.CreatedBy,
		})
	}
	return out, nil
}

func (a *storeTokenAdapter) Inspect(ctx context.Context, tokenID string) (*auth.ServiceToken, error) {
	it, err := a.st.GetToken(ctx, tokenID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, auth.ErrInvalidToken
		}
		return nil, err
	}
	return &auth.ServiceToken{
		ID:          it.ID,
		TenantID:    it.TenantID,
		AgentName:   it.AgentName,
		TokenHash:   it.TokenHash,
		DisplayHint: it.DisplayHint,
		Scopes:      parseScopes(it.Scopes),
		CreatedAt:   it.CreatedAt,
		ExpiresAt:   it.ExpiresAt,
		LastUsedAt:  it.LastUsedAt,
		RevokedAt:   it.RevokedAt,
		CreatedBy:   it.CreatedBy,
	}, nil
}

func (a *storeTokenAdapter) MarkUsed(ctx context.Context, tokenID string) error {
	err := a.st.UpdateTokenLastUsed(ctx, tokenID)
	if errors.Is(err, store.ErrNotFound) {
		return auth.ErrInvalidToken
	}
	return err
}

func mapApprovalError(err error) *actions.ExecutionError {
	switch {
	case errors.Is(err, approvals.ErrAlreadyResolved), errors.Is(err, store.ErrApprovalAlreadyResolved):
		return actions.NewExecutionError(actions.ErrValidationFailed, "approval already resolved", http.StatusConflict, false, nil)
	case errors.Is(err, approvals.ErrExpired), errors.Is(err, store.ErrApprovalExpired):
		return actions.NewExecutionError(actions.ErrApprovalTimeout, "approval has expired", http.StatusGone, false, nil)
	case errors.Is(err, approvals.ErrDuplicateVote):
		return actions.NewExecutionError(actions.ErrValidationFailed, "approver has already voted", http.StatusConflict, false, nil)
	case errors.Is(err, approvals.ErrNotFound), errors.Is(err, store.ErrNotFound):
		return actions.NewExecutionError(actions.ErrActionNotFound, "approval not found", http.StatusNotFound, false, nil)
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

func marshalScopes(scopes []string) string {
	if len(scopes) == 0 {
		return "[]"
	}
	b, err := json.Marshal(scopes)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func parseScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var scopes []string
	if err := json.Unmarshal([]byte(raw), &scopes); err != nil {
		return nil
	}
	return scopes
}

func (s *Server) removeHeldExecution(ctx context.Context, approvalRequestID string) {
	if s.runtime == nil || s.runtime.HeldExecutionStore == nil {
		return
	}
	removeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	_ = s.runtime.HeldExecutionStore.Remove(removeCtx, approvalRequestID)
}

package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/webhooks"
	"github.com/go-chi/chi/v5"
)

func (s *Server) requirePendingApproval(w http.ResponseWriter, r *http.Request, id, tenantID string, allowApproved bool) (*store.ApprovalRecord, bool) {
	existing, err := s.store.GetApproval(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, sanitizeErrMsg(err, http.StatusNotFound), http.StatusNotFound, false, nil))
		} else {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		}
		return nil, false
	}
	if existing.TenantID != tenantID {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, "approval not found", http.StatusNotFound, false, nil))
		return nil, false
	}
	now := time.Now().UTC()
	if existing.Status == "expired" {
		s.removeHeldExecution(r.Context(), id)
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrApprovalTimeout, "approval expired", http.StatusGone, false, nil))
		return nil, false
	}
	if !existing.ExpiresAt.After(now) {
		expired, expireErr := s.store.ExpireApproval(r.Context(), id)
		if expireErr != nil {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return nil, false
		}
		if !expired {
			refreshed, getErr := s.store.GetApproval(r.Context(), id)
			if getErr != nil {
				if errors.Is(getErr, store.ErrNotFound) {
					writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, "approval not found", http.StatusNotFound, false, nil))
				} else {
					writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
				}
				return nil, false
			}
			if refreshed == nil {
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, "approval not found", http.StatusNotFound, false, nil))
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
		if allowApproved && existing.Status == "approved" {
			return existing, true
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "approval already resolved", http.StatusConflict, false, nil))
		return existing, false
	}
	return existing, true
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	principal, tenantID, ok := requirePrincipalWithTenantContext(w, r)
	if !ok {
		return
	}

	existing, approvable := s.requirePendingApproval(w, r, id, tenantID, true)
	if !approvable {
		return
	}
	alreadyApproved := existing.Status == "approved"
	manager, canUseManager := newApprovalManager(s.store)
	if !alreadyApproved && !canUseManager {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "approval manager unavailable", http.StatusInternalServerError, false, nil))
		return
	}

	if !alreadyApproved {
		if err := manager.Approve(r.Context(), id, principal.ID); err != nil {
			writeEnvelopeError(w, r, mapApprovalError(err))
			return
		}
		updated, err := manager.Get(r.Context(), id)
		if err != nil {
			writeEnvelopeError(w, r, mapApprovalError(err))
			return
		}
		if updated == nil {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrResourceNotFound, "approval not found", http.StatusNotFound, false, nil))
			return
		}
		existing.Status = string(updated.Status)
		existing.ResolvedBy = updated.ResolvedBy
		existing.ResolvedAt = updated.ResolvedAt
		existing.Reason = updated.DenyReason
		if existing.Status == "approved" && s.webhookDispatcher != nil {
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
	}

	var execResult *actions.ExecutionResult
	fullyApproved := existing.Status == "approved"
	if s.runtime != nil && fullyApproved {
		result := s.runtime.ResumeApproved(context.WithoutCancel(r.Context()), id)
		if alreadyApproved && result.Error != nil && result.Error.Code == actions.ErrActionNotFound {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "approval already resolved", http.StatusConflict, false, nil))
			return
		}
		execResult = &result
	}

	resp := map[string]any{"approved": fullyApproved}
	if !fullyApproved {
		resp["pending"] = true
	}
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

	existing, pending := s.requirePendingApproval(w, r, id, tenantID, false)
	if !pending {
		return
	}

	manager, canUseManager := newApprovalManager(s.store)
	if !canUseManager {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "approval manager unavailable", http.StatusInternalServerError, false, nil))
		return
	}
	if err := manager.Deny(r.Context(), id, principal.ID, "denied via api"); err != nil {
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

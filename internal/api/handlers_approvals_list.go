package api

import (
	"net/http"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/webhooks"
)

func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	expiryStore, ok := s.store.(approvals.ExpiryStore)
	if !ok {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	if _, err := approvals.ExpirePendingWithSideEffects(r.Context(), expiryStore, tenantID, func(approval store.ApprovalRecord) {
		s.removeHeldExecution(r.Context(), approval.ID)
		if s.webhookDispatcher != nil {
			s.webhookDispatcher.EmitForTenant(tenantID, webhooks.EventApprovalExpired, map[string]any{
				"approval_id": approval.ID,
				"tenant_id":   tenantID,
				"request_id":  approval.RequestID,
				"agent_name":  approval.AgentName,
				"service":     approval.Service,
				"action":      approval.Action,
				"status":      "expired",
			})
		}
	}); err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	items, err := s.store.ListApprovals(r.Context(), tenantID, status)
	if err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
		return
	}
	writeSuccess(w, r, http.StatusOK, items)
}

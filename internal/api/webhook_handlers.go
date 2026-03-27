package api

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) registerWebhookRoutes(r chi.Router) {
	if s.webhookDispatcher == nil {
		return
	}
	r.With(RequireScope("webhooks:read")).Get("/webhooks", s.handleListWebhooks)
	r.With(RequireScope("webhooks:write")).Post("/webhooks", s.handleCreateWebhook)
	r.With(RequireScope("webhooks:write")).Delete("/webhooks/{id}", s.handleDeleteWebhook)
	r.With(RequireScope("webhooks:read")).Get("/webhooks/events", s.handleListRecentEvents)
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	subs := s.webhookDispatcher.ListSubscriptionsByTenant(tenantID)
	writeSuccess(w, r, http.StatusOK, map[string]any{"webhooks": subs})
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}

	var sub webhooks.Subscription
	if err := decodeJSON(r, &sub); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), status, false, nil))
		return
	}
	if sub.URL == "" {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "url is required", http.StatusBadRequest, false, nil))
		return
	}
	for _, event := range sub.Events {
		if !webhooks.IsKnownEventType(event) {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "events contains unknown or inactive event type", http.StatusBadRequest, false, map[string]any{"event": event}))
			return
		}
	}
	if err := validateWebhookURL(sub.URL); err != nil {
		writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil))
		return
	}
	if sub.ID == "" {
		sub.ID = "wh_" + uuid.NewString()
	}
	sub.TenantID = tenantID
	s.webhookDispatcher.Subscribe(sub)
	writeSuccess(w, r, http.StatusCreated, map[string]any{"webhook": map[string]string{"id": sub.ID}})
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	s.webhookDispatcher.UnsubscribeByTenant(id, tenantID)
	writeSuccess(w, r, http.StatusOK, map[string]any{"deleted": true})
}

const maxWebhookEventsLimit = 1000

func (s *Server) handleListRecentEvents(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "limit must be a positive integer", http.StatusBadRequest, false, nil))
			return
		}
		if parsed > maxWebhookEventsLimit {
			parsed = maxWebhookEventsLimit
		}
		limit = parsed
	}
	events := s.webhookDispatcher.RecentEventsByTenant(tenantID, limit)
	writeSuccess(w, r, http.StatusOK, map[string]any{"events": events})
}

// validateWebhookURL validates the URL format and rejects known private/loopback
// addresses at registration time as an early safety check. Delivery-time SSRF
// mitigation is enforced separately by the webhook dispatcher, which resolves
// the hostname and dials the resolved IP via a custom DialContext, rejecting
// private or loopback addresses before connecting.
func validateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("url scheme must be http or https, got %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url must have a host")
	}
	if isPrivateHost(host) {
		return fmt.Errorf("webhook url must not target private/loopback addresses")
	}
	return nil
}

func isPrivateHost(host string) bool {
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "127.0.0.1" || lower == "::1" || lower == "0.0.0.0" || lower == "::" {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return isPrivateIP(ip)
	}
	resolved, err := net.LookupIP(host)
	if err != nil || len(resolved) == 0 {
		return true
	}
	for _, addr := range resolved {
		if isPrivateIP(addr) {
			return true
		}
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

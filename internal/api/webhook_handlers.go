package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type webhookPersistenceStore interface {
	UpsertWebhookSubscription(ctx context.Context, sub *store.WebhookSubscriptionRecord) error
	DeleteWebhookSubscription(ctx context.Context, id string, tenantID string) error
	ListWebhookEvents(ctx context.Context, tenantID string, limit int) ([]store.WebhookEventRecord, error)
}

func (s *Server) webhookPersistenceStore() webhookPersistenceStore {
	if s == nil || s.store == nil {
		return nil
	}
	persist, _ := s.store.(webhookPersistenceStore)
	return persist
}

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
	if !decodeJSONOrWriteError(w, r, &sub) {
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
	if persist := s.webhookPersistenceStore(); persist != nil {
		eventsJSON := "[]"
		if len(sub.Events) > 0 {
			if b, err := json.Marshal(sub.Events); err == nil {
				eventsJSON = string(b)
			}
		}
		if err := persist.UpsertWebhookSubscription(r.Context(), &store.WebhookSubscriptionRecord{
			ID:         sub.ID,
			TenantID:   sub.TenantID,
			URL:        sub.URL,
			Secret:     sub.Secret,
			EventsJSON: eventsJSON,
			Active:     true,
		}); err != nil {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return
		}
	}
	s.webhookDispatcher.Subscribe(sub)
	writeSuccess(w, r, http.StatusCreated, map[string]any{"webhook": map[string]string{"id": sub.ID}})
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenantContext(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if persist := s.webhookPersistenceStore(); persist != nil {
		if err := persist.DeleteWebhookSubscription(r.Context(), id, tenantID); err != nil {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return
		}
	}
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
	if persist := s.webhookPersistenceStore(); persist != nil {
		recs, err := persist.ListWebhookEvents(r.Context(), tenantID, limit)
		if err != nil {
			writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "internal server error", http.StatusInternalServerError, false, nil))
			return
		}
		events = make([]webhooks.Event, 0, len(recs))
		for _, rec := range recs {
			payload := map[string]any{}
			if strings.TrimSpace(rec.DataJSON) != "" {
				_ = json.Unmarshal([]byte(rec.DataJSON), &payload)
			}
			events = append(events, webhooks.Event{
				ID:        rec.ID,
				Type:      webhooks.EventType(rec.Type),
				TenantID:  rec.TenantID,
				Timestamp: rec.Timestamp,
				Data:      payload,
			})
		}
	}
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
	if u.Scheme != "https" {
		return fmt.Errorf("url scheme must be https, got %q", u.Scheme)
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
		return false
	}
	for _, addr := range resolved {
		if isPrivateIP(addr) {
			return true
		}
	}
	return false
}

var registrationPrivateCIDRs = func() []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10", "169.254.0.0/16", "fc00::/7"} {
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}()

func isPrivateIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range registrationPrivateCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

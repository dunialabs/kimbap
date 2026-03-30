package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
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
	if err := validateWebhookURL(r.Context(), sub.URL); err != nil {
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
		merged := make([]webhooks.Event, 0, len(events)+len(recs))
		merged = append(merged, events...)
		for _, rec := range recs {
			payload := map[string]any{}
			if strings.TrimSpace(rec.DataJSON) != "" {
				if err := json.Unmarshal([]byte(rec.DataJSON), &payload); err != nil {
					continue
				}
			}
			merged = append(merged, webhooks.Event{
				ID:        rec.ID,
				Type:      webhooks.EventType(rec.Type),
				TenantID:  rec.TenantID,
				Timestamp: rec.Timestamp,
				Data:      payload,
			})
		}
		events = dedupeAndTrimWebhookEvents(merged, limit)
	}
	writeSuccess(w, r, http.StatusOK, map[string]any{"events": events})
}

func dedupeAndTrimWebhookEvents(items []webhooks.Event, limit int) []webhooks.Event {
	if len(items) == 0 {
		return nil
	}
	byID := make(map[string]webhooks.Event, len(items))
	for _, item := range items {
		existing, exists := byID[item.ID]
		if !exists || item.Timestamp.After(existing.Timestamp) {
			byID[item.ID] = item
		}
	}
	combined := make([]webhooks.Event, 0, len(byID))
	for _, item := range byID {
		combined = append(combined, item)
	}
	sort.SliceStable(combined, func(i, j int) bool {
		if combined[i].Timestamp.Equal(combined[j].Timestamp) {
			return combined[i].ID < combined[j].ID
		}
		return combined[i].Timestamp.Before(combined[j].Timestamp)
	})
	if limit <= 0 || len(combined) <= limit {
		return combined
	}
	return combined[len(combined)-limit:]
}

// validateWebhookURL validates the URL format and rejects known private/loopback
// addresses at registration time as an early safety check. Delivery-time SSRF
// mitigation is enforced separately by the webhook dispatcher, which resolves
// the hostname and dials the resolved IP via a custom DialContext, rejecting
// private or loopback addresses before connecting.
func validateWebhookURL(ctx context.Context, rawURL string) error {
	if ctx == nil {
		ctx = context.Background()
	}
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
	if isPrivateHost(ctx, host) {
		return fmt.Errorf("webhook url must not target private/loopback addresses")
	}
	return nil
}

func isPrivateHost(ctx context.Context, host string) bool {
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "127.0.0.1" || lower == "::1" || lower == "0.0.0.0" || lower == "::" {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return webhooks.IsPrivateIP(ip)
	}
	resolver := net.Resolver{}
	resolved, err := resolver.LookupIPAddr(ctx, host)
	if err != nil || len(resolved) == 0 {
		return false
	}
	for _, addr := range resolved {
		if webhooks.IsPrivateIP(addr.IP) {
			return true
		}
	}
	return false
}

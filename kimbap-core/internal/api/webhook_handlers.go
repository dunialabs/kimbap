package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/webhooks"
	"github.com/go-chi/chi/v5"
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
	tenantID := tenantFromContext(r.Context())
	subs := s.webhookDispatcher.ListSubscriptionsByTenant(tenantID)
	respondJSON(w, http.StatusOK, map[string]any{"webhooks": subs})
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromContext(r.Context())

	var sub webhooks.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if sub.URL == "" {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "url is required"})
		return
	}
	if err := validateWebhookURL(sub.URL); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if sub.ID == "" {
		sub.ID = fmt.Sprintf("wh_%d", time.Now().UnixNano())
	}
	sub.TenantID = tenantID
	s.webhookDispatcher.Subscribe(sub)
	respondJSON(w, http.StatusCreated, map[string]any{"webhook": map[string]string{"id": sub.ID}})
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromContext(r.Context())
	id := chi.URLParam(r, "id")
	s.webhookDispatcher.UnsubscribeByTenant(id, tenantID)
	respondJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) handleListRecentEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromContext(r.Context())
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	events := s.webhookDispatcher.RecentEventsByTenant(tenantID, limit)
	respondJSON(w, http.StatusOK, map[string]any{"events": events})
}

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
		return false
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

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

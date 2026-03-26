package controller

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/oauth/service"
)

type OAuthMetadataController struct {
	oauthService *service.OAuthService
}

func NewOAuthMetadataController(oauthService *service.OAuthService) *OAuthMetadataController {
	if oauthService == nil {
		oauthService = service.NewOAuthService(nil)
	}
	return &OAuthMetadataController{oauthService: oauthService}
}

func (c *OAuthMetadataController) AuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	issuer := publicURL()
	if issuer == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Internal server error"})
		return
	}
	meta := c.oauthService.GenerateAuthorizationServerMetadata(issuer)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Vary", "X-Forwarded-Host, X-Forwarded-Proto")
	writeJSON(w, http.StatusOK, meta)
}

func (c *OAuthMetadataController) ProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	resource := publicURL()
	if resource == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Internal server error"})
		return
	}
	authURL := resource
	meta := c.oauthService.GenerateProtectedResourceMetadata(resource, authURL)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Vary", "X-Forwarded-Host, X-Forwarded-Proto")
	writeJSON(w, http.StatusOK, meta)
}

func (c *OAuthMetadataController) HandleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusOK)
}

func publicURL() string {
	raw := strings.TrimSpace(config.Env("KIMBAP_PUBLIC_BASE_URL"))
	if raw != "" {
		parsed, err := url.Parse(raw)
		if err == nil && parsed.Host != "" && parsed.User == nil {
			scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
			if scheme == "http" || scheme == "https" {
				parsed.RawQuery = ""
				parsed.Fragment = ""
				return strings.TrimRight(parsed.String(), "/")
			}
		}
	}
	return ""
}

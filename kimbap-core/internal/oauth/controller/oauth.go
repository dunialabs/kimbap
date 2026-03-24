package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/dunialabs/kimbap-core/internal/oauth/service"
	oauthtypes "github.com/dunialabs/kimbap-core/internal/oauth/types"
	oauthviews "github.com/dunialabs/kimbap-core/internal/oauth/views"
	"github.com/dunialabs/kimbap-core/internal/repository"
)

type UserTokenValidator interface {
	ValidateUserToken(token string) (string, error)
}

type OAuthController struct {
	oauthService  *service.OAuthService
	clientService *service.OAuthClientService
	proxyRepo     *repository.ProxyRepository
	userValidator UserTokenValidator
}

const maxOAuthControllerRequestBytes int64 = 1 << 20

func setOAuthNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func NewOAuthController(oauthService *service.OAuthService, clientService *service.OAuthClientService, userValidator UserTokenValidator) *OAuthController {
	if oauthService == nil {
		oauthService = service.NewOAuthService(nil)
	}
	if clientService == nil {
		clientService = service.NewOAuthClientService(nil, oauthService)
	}
	return &OAuthController{oauthService: oauthService, clientService: clientService, proxyRepo: repository.NewProxyRepository(nil), userValidator: userValidator}
}

func (c *OAuthController) Register(w http.ResponseWriter, r *http.Request) {
	setOAuthNoStoreHeaders(w)
	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthControllerRequestBytes)
	var metadata oauthtypes.OAuthClientMetadata
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "Invalid request body")
		return
	}
	if !strings.HasPrefix(metadata.ClientID, "https://") {
		if strings.TrimSpace(metadata.ClientName) == "" || metadata.RedirectURIs == nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "client_name is required and must be a string, and redirect_uris is required and must be an array")
			return
		}
	}
	info, err := c.clientService.RegisterClient(r.Context(), metadata, nil)
	if err != nil {
		parts := strings.SplitN(err.Error(), ":", 2)
		desc := "Internal server error"
		code := "server_error"
		status := http.StatusInternalServerError
		errMsg := err.Error()
		trimmedErrMsg := strings.TrimSpace(errMsg)
		if strings.HasPrefix(trimmedErrMsg, "invalid_client_metadata") || strings.HasPrefix(trimmedErrMsg, "invalid_redirect_uri") {
			code = strings.TrimSpace(parts[0])
			status = http.StatusBadRequest
			if len(parts) == 2 {
				desc = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(errMsg, "unsupported ") {
			code = "invalid_client_metadata"
			status = http.StatusBadRequest
		}
		writeOAuthError(w, status, code, desc)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (c *OAuthController) GetClientInfo(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientId")
	client, err := c.clientService.GetClient(r.Context(), clientID)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal server error")
		return
	}
	if client == nil {
		writeOAuthError(w, http.StatusNotFound, "not_found", "Client not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":     client.ClientID,
		"client_name":   client.ClientName,
		"redirect_uris": client.RedirectURIs,
		"scopes":        client.Scopes,
	})
}

func (c *OAuthController) ShowAuthorizePage(w http.ResponseWriter, r *http.Request) {
	setOAuthNoStoreHeaders(w)
	q := r.URL.Query()
	responseType := q.Get("response_type")
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	if responseType == "" || clientID == "" || redirectURI == "" {
		writePlainText(w, http.StatusBadRequest, "Missing required parameters")
		return
	}
	if responseType != "code" {
		writePlainText(w, http.StatusBadRequest, "Unsupported response_type")
		return
	}
	client, err := c.clientService.GetClient(r.Context(), clientID)
	if err != nil || client == nil {
		writePlainText(w, http.StatusBadRequest, "Invalid client_id")
		return
	}
	if !c.oauthService.ValidateRedirectURI(redirectURI, client.RedirectURIs) {
		writePlainText(w, http.StatusBadRequest, "Invalid redirect_uri")
		return
	}

	body := oauthviews.ConsentHTML
	if strings.TrimSpace(body) == "" {
		writePlainText(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	html := body
	proxyKey := ""
	if c.proxyRepo != nil {
		if proxy, proxyErr := c.proxyRepo.FindFirst(); proxyErr == nil && proxy != nil {
			proxyKey = proxy.ProxyKey
		}
	}
	requestedScopes := c.oauthService.ParseScope(q.Get("scope"))
	clientName := client.ClientName
	if strings.TrimSpace(clientName) == "" {
		clientName = "Unknown Application"
	}
	scopeDescriptions := map[string]string{
		"mcp:tools":     "Execute MCP tools and functions",
		"mcp:resources": "Access MCP resources and data",
		"mcp:prompts":   "Use MCP prompt templates",
	}
	scopeItems := make([]string, 0, len(requestedScopes))
	for _, scope := range requestedScopes {
		description := scopeDescriptions[scope]
		if strings.TrimSpace(description) == "" {
			description = scope
		}
		scopeItems = append(scopeItems, `<li><span class="scope-icon">✓</span>`+htmlEscape(description)+`</li>`)
	}
	scopeListHTML := strings.Join(scopeItems, "")
	html = strings.NewReplacer(
		"__CLIENT_NAME__", htmlEscape(clientName),
		"{{SCOPE_LIST}}", scopeListHTML,
		"{{CLIENT_ID}}", jsEscape(clientID),
		"{{REDIRECT_URI}}", jsEscape(redirectURI),
		"{{SCOPE}}", jsEscape(q.Get("scope")),
		"{{STATE}}", jsEscape(q.Get("state")),
		"{{CODE_CHALLENGE}}", jsEscape(q.Get("code_challenge")),
		"{{CODE_CHALLENGE_METHOD}}", jsEscape(q.Get("code_challenge_method")),
		"{{RESOURCE}}", jsEscape(q.Get("resource")),
		"{{PROXY_KEY}}", jsEscape(proxyKey),
	).Replace(html)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (c *OAuthController) Authorize(w http.ResponseWriter, r *http.Request) {
	setOAuthNoStoreHeaders(w)
	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthControllerRequestBytes)
	var req oauthtypes.AuthorizationApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	redirect, oauthErr, status := c.oauthService.HandleAuthorize(r.Context(), req, func(ctx context.Context, token string) (string, error) {
		if c.userValidator == nil {
			return "", http.ErrNoCookie
		}
		return c.userValidator.ValidateUserToken(token)
	})
	if oauthErr != nil {
		writeJSON(w, status, oauthErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirect": redirect})
}

func (c *OAuthController) Token(w http.ResponseWriter, r *http.Request) {
	setOAuthNoStoreHeaders(w)
	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthControllerRequestBytes)
	var request oauthtypes.TokenRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}
		request = oauthtypes.TokenRequest{
			GrantType:    r.Form.Get("grant_type"),
			Code:         r.Form.Get("code"),
			RedirectURI:  r.Form.Get("redirect_uri"),
			ClientID:     r.Form.Get("client_id"),
			ClientSecret: r.Form.Get("client_secret"),
			CodeVerifier: r.Form.Get("code_verifier"),
			RefreshToken: r.Form.Get("refresh_token"),
			Scope:        r.Form.Get("scope"),
		}
	} else if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	response, oauthErr, status := c.oauthService.HandleToken(r.Context(), request, r.Header.Get("Authorization"), c.clientService)
	if oauthErr != nil {
		writeJSON(w, status, oauthErr)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (c *OAuthController) Introspect(w http.ResponseWriter, r *http.Request) {
	setOAuthNoStoreHeaders(w)
	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthControllerRequestBytes)
	var req oauthtypes.TokenIntrospectionRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}
		req.Token = r.FormValue("token")
		req.TokenTypeHint = r.FormValue("token_type_hint")
		req.ClientID = r.FormValue("client_id")
		req.ClientSecret = r.FormValue("client_secret")
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	response, oauthErr, status := c.oauthService.HandleIntrospect(r.Context(), req, r.Header.Get("Authorization"), c.clientService)
	if oauthErr != nil {
		writeJSON(w, status, oauthErr)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (c *OAuthController) Revoke(w http.ResponseWriter, r *http.Request) {
	setOAuthNoStoreHeaders(w)
	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthControllerRequestBytes)
	var req oauthtypes.TokenRevocationRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}
		req.Token = r.FormValue("token")
		req.TokenTypeHint = r.FormValue("token_type_hint")
		req.ClientID = r.FormValue("client_id")
		req.ClientSecret = r.FormValue("client_secret")
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}
	oauthErr, status := c.oauthService.HandleRevoke(r.Context(), req, r.Header.Get("Authorization"), c.clientService)
	if oauthErr != nil {
		writeJSON(w, status, oauthErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (c *OAuthController) HandleOptions(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeOAuthError(w http.ResponseWriter, status int, errorCode string, description string) {
	writeJSON(w, status, oauthtypes.OAuthErrorResponse{Error: errorCode, ErrorDescription: description})
}

func writePlainText(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

func htmlEscape(v string) string {
	v = strings.ReplaceAll(v, "&", "&amp;")
	v = strings.ReplaceAll(v, "<", "&lt;")
	v = strings.ReplaceAll(v, ">", "&gt;")
	v = strings.ReplaceAll(v, "\"", "&quot;")
	v = strings.ReplaceAll(v, "'", "&#39;")
	return v
}

func jsEscape(v string) string {
	b, _ := json.Marshal(v)
	if len(b) >= 2 {
		v = string(b[1 : len(b)-1])
	}
	v = strings.ReplaceAll(v, "<", "\\u003C")
	v = strings.ReplaceAll(v, ">", "\\u003E")
	v = strings.ReplaceAll(v, "&", "\\u0026")
	return v
}

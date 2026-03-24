package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/dunialabs/kimbap-core/internal/oauth/service"
	oauthtypes "github.com/dunialabs/kimbap-core/internal/oauth/types"
)

const maxOAuthClientRequestBytes int64 = 1 << 20

type OAuthClientController struct {
	clientService *service.OAuthClientService
}

func NewOAuthClientController(clientService *service.OAuthClientService) *OAuthClientController {
	if clientService == nil {
		clientService = service.NewOAuthClientService(nil, nil)
	}
	return &OAuthClientController{clientService: clientService}
}

func (c *OAuthClientController) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthClientRequestBytes)
	var metadata oauthtypes.OAuthClientMetadata
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeOAuthError(w, http.StatusRequestEntityTooLarge, "invalid_request", "request entity too large")
			return
		}
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "Invalid request body")
		return
	}
	client, err := c.clientService.RegisterClient(r.Context(), metadata, nil)
	if err != nil {
		errMsg := err.Error()
		status := http.StatusInternalServerError
		code := "server_error"
		desc := "Internal server error"
		if strings.HasPrefix(errMsg, "invalid_client_metadata") || strings.HasPrefix(errMsg, "invalid_redirect_uri") {
			parts := strings.SplitN(errMsg, ":", 2)
			status = http.StatusBadRequest
			parsedCode := strings.TrimSpace(parts[0])
			if parsedCode == "invalid_client_metadata" || parsedCode == "invalid_redirect_uri" {
				code = parsedCode
			}
			if code == "invalid_redirect_uri" {
				desc = "Invalid redirect URI"
			} else {
				desc = "Invalid client metadata"
			}
		} else if strings.HasPrefix(errMsg, "unsupported ") {
			status = http.StatusBadRequest
			code = "invalid_client_metadata"
			desc = "Invalid client metadata"
		}
		writeOAuthError(w, status, code, desc)
		return
	}
	writeJSON(w, http.StatusOK, client)
}

func (c *OAuthClientController) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := c.clientService.ListClients(r.Context())
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"clients": clients})
}

func (c *OAuthClientController) GetClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientId")
	if clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "clientId is required")
		return
	}

	client, err := c.clientService.GetClient(r.Context(), clientID)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal server error")
		return
	}
	if client == nil {
		writeOAuthError(w, http.StatusNotFound, "invalid_client", "Client not found")
		return
	}

	writeJSON(w, http.StatusOK, client)
}

func (c *OAuthClientController) UpdateClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientId")
	if clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "clientId is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthClientRequestBytes)
	var updates oauthtypes.OAuthClientMetadata
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeOAuthError(w, http.StatusRequestEntityTooLarge, "invalid_request", "request entity too large")
			return
		}
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	client, err := c.clientService.UpdateClient(r.Context(), clientID, updates)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal server error")
		return
	}
	if client == nil {
		writeOAuthError(w, http.StatusNotFound, "invalid_client", "Client not found")
		return
	}

	writeJSON(w, http.StatusOK, client)
}

func (c *OAuthClientController) DeleteClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		clientID = chi.URLParam(r, "clientId")
	}
	if strings.TrimSpace(clientID) == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "clientId is required")
		return
	}
	success, err := c.clientService.DeleteClient(r.Context(), clientID)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Internal server error")
		return
	}
	if !success {
		writeOAuthError(w, http.StatusNotFound, "invalid_client", "Client not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Client deleted successfully"})
}

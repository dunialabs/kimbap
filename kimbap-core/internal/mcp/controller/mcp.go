package controller

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	logservice "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/middleware"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/dunialabs/kimbap-core/internal/utils"
)

type MCPController struct{}

const maxInitializeDetectBodyBytes int64 = 1 << 20
const maxSessionPostBodyBytes int64 = 10 << 20

func NewMCPController() *MCPController {
	return &MCPController{}
}

func (c *MCPController) HandleMCP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.handleGet(w, r)
	case http.MethodHead:
		c.handleGet(w, r)
	case http.MethodPost:
		c.handlePost(w, r)
	case http.MethodDelete:
		c.handleDelete(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSONRPCErrorWithCode(w, http.StatusMethodNotAllowed, -32000, "Method not allowed.")
	}
}

func (c *MCPController) handleGet(w http.ResponseWriter, r *http.Request) {
	store := core.SessionStoreInstance()
	sessionID := mcpSessionIDFromHeader(r)
	if sessionID == "" {
		writeJSONRPCError(w, http.StatusBadRequest, "Invalid or missing session ID")
		return
	}

	session := store.GetSession(sessionID)
	proxySession := store.GetProxySession(sessionID)
	if session == nil || proxySession == nil {
		writeJSONRPCError(w, http.StatusBadRequest, "Invalid or missing session ID")
		return
	}

	session.Touch()

	var sseMu sync.Mutex
	sseConnected := false
	markSSEConnected := func() {
		sseMu.Lock()
		defer sseMu.Unlock()
		if sseConnected {
			return
		}
		sseConnected = true
		session.MarkSSEConnected()
	}
	markSSEDisconnected := func() {
		sseMu.Lock()
		defer sseMu.Unlock()
		if !sseConnected {
			return
		}
		sseConnected = false
		session.MarkSSEDisconnected()
	}

	trackingWriter := newSSETrackingResponseWriter(w, func(statusCode int, header http.Header) {
		if statusCode >= 200 && statusCode < 300 {
			for _, value := range header.Values("Content-Type") {
				if strings.Contains(strings.ToLower(value), "text/event-stream") {
					markSSEConnected()
					break
				}
			}
		}
	})
	defer markSSEDisconnected()
	go func() {
		<-r.Context().Done()
		markSSEDisconnected()
	}()

	if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
		if err := proxySession.HandleReconnection(r.Context(), trackingWriter, lastEventID); err != nil {
			if !trackingWriter.headersSent {
				writeJSONRPCError(w, http.StatusInternalServerError, "Failed to handle reconnection")
			}
			// If headers already sent (SSE stream started), just return — stream ends on handler exit
		}
		return
	}

	proxySession.HandleRequest(trackingWriter, r)
}

func (c *MCPController) handlePost(w http.ResponseWriter, r *http.Request) {
	store := core.SessionStoreInstance()
	sessionID := mcpSessionIDFromHeader(r)
	if sessionID != "" {
		r.Body = http.MaxBytesReader(w, r.Body, maxSessionPostBodyBytes)
	}
	if sessionID == "" {
		r.Body = http.MaxBytesReader(w, r.Body, maxInitializeDetectBodyBytes)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				middleware.WriteRequestEntityTooLargeLikeExpress(w)
				return
			}
			writeJSONRPCError(w, http.StatusBadRequest, "Bad Request: Invalid request body")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		isInitRequest, validJSON := isInitializeRequest(bodyBytes)
		if !isInitRequest {
			if !validJSON {
				writeJSONRPCError(w, http.StatusBadRequest, "Bad Request: Invalid JSON")
				return
			}
			writeJSONRPCError(w, http.StatusBadRequest, "Bad Request: Server not initialized")
			return
		}

		authCtx, ok := middleware.GetAuthContext(r.Context())
		if !ok || authCtx == nil {
			w.Header().Set("WWW-Authenticate", middleware.BuildWWWAuthenticateHeader(r, "invalid_token", "Missing auth context"))
			writeJSONRPCError(w, http.StatusUnauthorized, "Missing auth context")
			return
		}
		token, err := middleware.ExtractAuthToken(r)
		if err != nil {
			w.Header().Set("WWW-Authenticate", middleware.BuildWWWAuthenticateHeader(r, "invalid_token", "invalid or missing token"))
			writeJSONRPCError(w, http.StatusUnauthorized, "invalid or missing token")
			return
		}
		sessionID, err = utils.GenerateSessionID()
		if err != nil {
			writeJSONRPCError(w, http.StatusInternalServerError, "Failed to create session")
			return
		}
		mcpAuth := convertAuthContext(authCtx)
		ip := middleware.ClientIPFromRequest(r)
		ua := strings.TrimSpace(r.Header.Get("User-Agent"))
		sl := logservice.NewSessionLogger(authCtx.UserID, sessionID, authCtx.Token, ip, ua)
		_, err = store.CreateSession(r.Context(), sessionID, authCtx.UserID, token, mcpAuth, sl)
		if err != nil {
			writeJSONRPCError(w, http.StatusInternalServerError, "Failed to create session")
			return
		}
		sl.LogAuth(types.MCPEventLogTypeAuthTokenValidation, "", nil)

		w.Header().Set("Mcp-Session-Id", sessionID)
		w.Header().Set("mcp-session-id", sessionID)
	}

	proxySession := store.GetProxySession(sessionID)
	if proxySession == nil {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"jsonrpc": "2.0",
			"error": map[string]any{
				"code":    -32603,
				"message": "Invalid or missing session ID",
			},
			"id": nil,
		})
		return
	}
	if session := store.GetSession(sessionID); session != nil {
		session.Touch()
	}

	proxySession.HandleRequest(w, r)
}

func (c *MCPController) handleDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := mcpSessionIDFromHeader(r)
	if sessionID == "" {
		writeJSONRPCErrorWithCode(w, http.StatusBadRequest, -32000, "Bad Request: No valid session ID provided")
		return
	}

	store := core.SessionStoreInstance()
	session := store.GetSession(sessionID)
	proxySession := store.GetProxySession(sessionID)
	if session != nil {
		session.Touch()
	}

	if proxySession == nil {
		if session != nil {
			store.RemoveSession(sessionID, mcptypes.DisconnectReasonClientDisconnect, false)
		}
		writeJSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "result": map[string]any{"message": "Session terminated or not found"}, "id": nil})
		return
	}

	proxySession.HandleRequest(w, r)
	store.RemoveSession(sessionID, mcptypes.DisconnectReasonClientDisconnect, false)
}

type sseTrackingResponseWriter struct {
	http.ResponseWriter
	onHeaders   func(statusCode int, header http.Header)
	headersSent bool
}

func newSSETrackingResponseWriter(w http.ResponseWriter, onHeaders func(statusCode int, header http.Header)) *sseTrackingResponseWriter {
	return &sseTrackingResponseWriter{ResponseWriter: w, onHeaders: onHeaders}
}

func (w *sseTrackingResponseWriter) WriteHeader(statusCode int) {
	if !w.headersSent {
		w.headersSent = true
		if w.onHeaders != nil {
			w.onHeaders(statusCode, w.Header())
		}
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *sseTrackingResponseWriter) Write(p []byte) (int, error) {
	if !w.headersSent {
		w.headersSent = true
		if w.onHeaders != nil {
			w.onHeaders(http.StatusOK, w.Header())
		}
	}
	return w.ResponseWriter.Write(p)
}

func (w *sseTrackingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *sseTrackingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http hijacker is unavailable")
	}
	return hj.Hijack()
}

func mcpSessionIDFromHeader(r *http.Request) string {
	return r.Header.Get("Mcp-Session-Id")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func convertAuthContext(auth *types.AuthContext) mcptypes.AuthContext {
	ctx := mcptypes.AuthContext{
		Kind:            auth.Kind,
		UserID:          auth.UserID,
		Role:            auth.Role,
		Status:          auth.Status,
		Permissions:     auth.Permissions,
		UserPreferences: auth.UserPreferences,
		LaunchConfigs:   auth.LaunchConfigs,
		OAuthClientID:   auth.OAuthClientID,
		OAuthScopes:     auth.OAuthScopes,
		UserAgent:       auth.UserAgent,
		AuthenticatedAt: auth.AuthenticatedAt,
		RateLimit:       auth.RateLimit,
	}
	if auth.ExpiresAt != nil {
		ctx.ExpiresAt = *auth.ExpiresAt
	}
	return ctx
}

func isInitializeRequest(body []byte) (bool, bool) {
	if len(body) == 0 {
		return false, false
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, false
	}
	switch root := payload.(type) {
	case map[string]any:
		method, _ := root["method"].(string)
		return method == "initialize", true
	case []any:
		if len(root) == 0 {
			return false, true
		}
		first, ok := root[0].(map[string]any)
		if !ok {
			return false, true
		}
		method, _ := first["method"].(string)
		return method == "initialize", true
	default:
		return false, true
	}
}

func writeJSONRPCError(w http.ResponseWriter, status int, message string) {
	code := -32603
	switch {
	case status == http.StatusBadRequest:
		code = -32600
	case status == http.StatusNotFound || status == http.StatusMethodNotAllowed:
		code = -32601
	case status == http.StatusUnprocessableEntity:
		code = -32602
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		code = -32000
	}
	writeJSONRPCErrorWithCode(w, status, code, message)
}

func writeJSONRPCErrorWithCode(w http.ResponseWriter, status int, code int, message string) {
	writeJSON(w, status, map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"id": nil,
	})
}

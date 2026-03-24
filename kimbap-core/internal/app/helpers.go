package app

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/middleware"
)

// latestProtocolVersion is hardcoded because the Go SDK (go-sdk v1.3.0) keeps this
// value unexported and uses "2025-06-18" as latest, while "2025-11-25" is marked
// "not yet released". We intentionally use "2025-11-25" as the current protocol
// version. Update this when the Go SDK formally exports it.
const latestProtocolVersion = "2025-11-25"

var StartTime = time.Now()

func ParseIntDefault(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func FileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func FormParserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			_ = r.ParseForm()
		}
		next.ServeHTTP(w, r)
	})
}

func MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"Method not allowed."},"id":null}`))
	}
}

func HeadMCPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.Fields(authHeader)
		hasToken := len(parts) > 1 && strings.EqualFold(parts[0], "Bearer") && strings.TrimSpace(strings.Join(parts[1:], " ")) != ""
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id,mcp-session-id,www-authenticate")
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("mcp-protocol-version", latestProtocolVersion)
		if !hasToken {
			w.Header().Set("WWW-Authenticate", middleware.BuildWWWAuthenticateHeader(r, "invalid_token", "Missing Authorization header"))
			WriteJSON(w, http.StatusUnauthorized, map[string]any{
				"jsonrpc": "2.0",
				"error": map[string]any{
					"code":    -32000,
					"message": "Method not allowed.",
				},
				"id": nil,
			})
			return
		}
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"jsonrpc": "2.0",
			"error": map[string]any{
				"code":    -32000,
				"message": "Method not allowed.",
			},
			"id": nil,
		})
	}
}

func OptionsMCPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// CORS origin is handled by the CORS middleware for normal requests.
		// For OPTIONS preflight, the CORS middleware also handles it since
		// OptionsPassthrough is false. This handler is a fallback for
		// MCP-specific preflight details.
		h.Set("Access-Control-Allow-Methods", "GET, POST, DELETE")
		reqHeaders := r.Header.Get("Access-Control-Request-Headers")
		if reqHeaders == "" {
			reqHeaders = "Content-Type, Authorization, Mcp-Session-Id, mcp-session-id, mcp-protocol-version, Accept, last-event-id"
		}
		h.Set("Access-Control-Allow-Headers", reqHeaders)
		h.Set("Access-Control-Expose-Headers", "Mcp-Session-Id,mcp-session-id,www-authenticate")
		h.Set("Access-Control-Max-Age", "86400")
		h.Set("Vary", "Access-Control-Request-Headers")
		w.WriteHeader(http.StatusNoContent)
	}
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

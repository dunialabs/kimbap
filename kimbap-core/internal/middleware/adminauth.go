package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	types "github.com/dunialabs/kimbap-core/internal/types"
)

type AdminAuthMiddleware struct {
	auth *AuthMiddleware
}

func NewAdminAuthMiddleware(auth *AuthMiddleware) *AdminAuthMiddleware {
	return &AdminAuthMiddleware{auth: auth}
}

func (m *AdminAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresAdminAuthPath(r) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if m == nil || m.auth == nil {
			writeAdminUnauthorized(w, r, "invalid_request", "admin authentication is not configured")
			return
		}

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeAdminUnauthorized(w, r, "invalid_request", "Authorization header with Bearer token is required")
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			writeAdminUnauthorized(w, r, "invalid_request", "Authorization header with Bearer token is required")
			return
		}

		authContext, err := m.auth.AuthenticateRequest(r)
		if err != nil {
			status := authStatusCodeForError(err)
			if status == http.StatusForbidden {
				writeAdminForbidden(w, r, "access_denied", sanitizeAuthError(err))
				return
			}
			if status == http.StatusUnauthorized {
				writeAdminUnauthorized(w, r, "invalid_token", sanitizeAuthError(err))
				return
			}
			writeJSONError(w, status, "authentication service unavailable")
			return
		}
		if authContext == nil {
			writeAdminUnauthorized(w, r, "invalid_request", "invalid authorization header format")
			return
		}
		if authContext.Role != types.UserRoleOwner && authContext.Role != types.UserRoleAdmin {
			writeAdminForbidden(w, r, "access_denied", "admin access required")
			return
		}

		ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requiresAdminAuthPath(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	path := strings.TrimSpace(r.URL.Path)
	return path == "/admin" || strings.HasPrefix(path, "/admin/") || strings.HasPrefix(path, "/oauth/admin/")
}

func writeAdminUnauthorized(w http.ResponseWriter, r *http.Request, errCode, message string) {
	setWWWAuthenticateIfUnauthorized(w, r, http.StatusUnauthorized, errCode, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	if strings.HasPrefix(r.URL.Path, "/oauth/admin/") {
		if errCode == "" {
			errCode = "invalid_token"
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             errCode,
			"error_description": message,
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":   false,
		"message":   message,
		"timestamp": time.Now().Unix(),
	})
}

func writeAdminForbidden(w http.ResponseWriter, r *http.Request, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	if strings.HasPrefix(r.URL.Path, "/oauth/admin/") {
		if errCode == "" {
			errCode = "access_denied"
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             errCode,
			"error_description": message,
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":   false,
		"message":   message,
		"timestamp": time.Now().Unix(),
	})
}

package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/auth"
	"github.com/google/uuid"
)

type contextKey string

const (
	contextKeyPrincipal contextKey = "principal"
	contextKeyTenant    contextKey = "tenant"
	contextKeyRequestID contextKey = "request_id"
)

func BearerAuth(tokenService *auth.TokenService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokenService == nil {
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "token service unavailable", http.StatusInternalServerError, false, nil))
				return
			}
			authz := strings.TrimSpace(r.Header.Get("Authorization"))
			if authz == "" {
				setBearerAuthHeader(w, "", "", "")
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "bearer token required", http.StatusUnauthorized, false, nil))
				return
			}
			parts := strings.Fields(authz)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				setBearerAuthHeader(w, "invalid_request", "bearer token required", "")
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrValidationFailed, "malformed authorization header", http.StatusBadRequest, false, nil))
				return
			}
			principal, err := tokenService.Validate(r.Context(), strings.TrimSpace(parts[1]))
			if err != nil {
				switch {
				case errors.Is(err, auth.ErrExpiredToken):
					setBearerAuthHeader(w, "invalid_token", "token expired", "")
					writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "token expired", http.StatusUnauthorized, false, nil))
				case errors.Is(err, auth.ErrRevokedToken):
					setBearerAuthHeader(w, "invalid_token", "token revoked", "")
					writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "token revoked", http.StatusUnauthorized, false, nil))
				case errors.Is(err, auth.ErrInvalidToken):
					setBearerAuthHeader(w, "invalid_token", "invalid token", "")
					writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "invalid token", http.StatusUnauthorized, false, nil))
				default:
					writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrDownstreamUnavailable, "authentication service error", http.StatusInternalServerError, false, nil))
				}
				return
			}
			ctx := context.WithValue(r.Context(), contextKeyPrincipal, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TenantContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := principalFromContext(r.Context())
			if principal == nil {
				setBearerAuthHeader(w, "invalid_token", "tenant context unavailable", "")
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "tenant context unavailable", http.StatusUnauthorized, false, nil))
				return
			}
			if principal.Type == auth.PrincipalTypeService && strings.TrimSpace(principal.TenantID) == "" {
				setBearerAuthHeader(w, "invalid_token", "service principal missing tenant context", "")
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "service principal missing tenant context", http.StatusUnauthorized, false, nil))
				return
			}
			tenantID := effectiveTenantID(principal)
			if tenantID == "" {
				setBearerAuthHeader(w, "invalid_token", "tenant context unavailable", "")
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "tenant context unavailable", http.StatusUnauthorized, false, nil))
				return
			}
			ctx := context.WithValue(r.Context(), contextKeyTenant, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func effectiveTenantID(principal *auth.Principal) string {
	if principal != nil {
		if tenantID := strings.TrimSpace(principal.TenantID); tenantID != "" {
			return tenantID
		}
	}
	if tenantID := strings.TrimSpace(os.Getenv("KIMBAP_API_DEFAULT_TENANT_ID")); tenantID != "" {
		return tenantID
	}
	if tenantID := strings.TrimSpace(os.Getenv("KIMBAP_TENANT_ID")); tenantID != "" {
		return tenantID
	}
	return ""
}

func RequestID() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := sanitizeRequestID(r.Header.Get("X-Request-ID"))
			if id == "" {
				id = uuid.NewString()
			}
			w.Header().Set("X-Request-ID", id)
			ctx := context.WithValue(r.Context(), contextKeyRequestID, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

const maxRequestIDLen = 128

func sanitizeRequestID(raw string) string {
	id := strings.TrimSpace(raw)
	if id == "" || len(id) > maxRequestIDLen {
		return ""
	}
	for _, c := range id {
		if c < 0x20 || c == 0x7f {
			return ""
		}
		isAlnum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		isSafe := c == '-' || c == '_' || c == '.'
		if !isAlnum && !isSafe {
			return ""
		}
	}
	return id
}

func JSONContentType() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	}
}

func principalFromContext(ctx context.Context) *auth.Principal {
	v := ctx.Value(contextKeyPrincipal)
	p, _ := v.(*auth.Principal)
	return p
}

func tenantFromContext(ctx context.Context) string {
	v := ctx.Value(contextKeyTenant)
	ten, _ := v.(string)
	return ten
}

func requestIDFromContext(ctx context.Context) string {
	v := ctx.Value(contextKeyRequestID)
	id, _ := v.(string)
	return id
}

func RequireScope(scope string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := principalFromContext(r.Context())
			if principal == nil {
				setBearerAuthHeader(w, "invalid_token", "authentication required", "")
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthenticated, "authentication required", http.StatusUnauthorized, false, nil))
				return
			}
			if !principalHasScope(principal, scope) {
				setBearerAuthHeader(w, "insufficient_scope", "insufficient scope", scope)
				writeEnvelopeError(w, r, actions.NewExecutionError(actions.ErrUnauthorized, "insufficient scope: "+scope, http.StatusForbidden, false, nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setBearerAuthHeader(w http.ResponseWriter, errorCode, errorDescription, scope string) {
	parts := []string{`Bearer realm="kimbap"`}
	if v := strings.TrimSpace(errorCode); v != "" {
		parts = append(parts, fmt.Sprintf(`error="%s"`, sanitizeAuthHeaderValue(v)))
	}
	if v := strings.TrimSpace(errorDescription); v != "" {
		parts = append(parts, fmt.Sprintf(`error_description="%s"`, sanitizeAuthHeaderValue(v)))
	}
	if v := strings.TrimSpace(scope); v != "" {
		parts = append(parts, fmt.Sprintf(`scope="%s"`, sanitizeAuthHeaderValue(v)))
	}
	w.Header().Set("WWW-Authenticate", strings.Join(parts, ", "))
}

func sanitizeAuthHeaderValue(v string) string {
	v = strings.ReplaceAll(v, `\`, ``)
	v = strings.ReplaceAll(v, `"`, ``)
	return v
}

func principalHasScope(principal *auth.Principal, scope string) bool {
	if principal == nil {
		return false
	}
	for _, s := range principal.Scopes {
		if s == scope || s == "*" || s == "admin" {
			return true
		}
	}
	return false
}

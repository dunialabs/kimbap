package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/auth"
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
				writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthenticated, "token service unavailable", http.StatusUnauthorized, false, nil))
				return
			}
			authz := strings.TrimSpace(r.Header.Get("Authorization"))
			parts := strings.Fields(authz)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthenticated, "bearer token required", http.StatusUnauthorized, false, nil))
				return
			}
			principal, err := tokenService.Validate(r.Context(), strings.TrimSpace(parts[1]))
			if err != nil {
				status := http.StatusUnauthorized
				if errors.Is(err, auth.ErrExpiredToken) || errors.Is(err, auth.ErrRevokedToken) {
					status = http.StatusForbidden
				}
				msg := "authentication failed"
				if errors.Is(err, auth.ErrExpiredToken) {
					msg = "token expired"
				} else if errors.Is(err, auth.ErrRevokedToken) {
					msg = "token revoked"
				} else if errors.Is(err, auth.ErrInvalidToken) {
					msg = "invalid token"
				}
				writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthenticated, msg, status, false, nil))
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
			if principal == nil || strings.TrimSpace(principal.TenantID) == "" {
				writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthorized, "tenant context unavailable", http.StatusForbidden, false, nil))
				return
			}
			ctx := context.WithValue(r.Context(), contextKeyTenant, principal.TenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestID() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if id == "" {
				id = uuid.NewString()
			}
			w.Header().Set("X-Request-ID", id)
			ctx := context.WithValue(r.Context(), contextKeyRequestID, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
				writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthenticated, "authentication required", http.StatusUnauthorized, false, nil))
				return
			}
			if !principalHasScope(principal, scope) {
				writeExecutionError(w, actions.NewExecutionError(actions.ErrUnauthorized, "insufficient scope: "+scope, http.StatusForbidden, false, nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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

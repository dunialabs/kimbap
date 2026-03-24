package user

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/middleware"
	types "github.com/dunialabs/kimbap-core/internal/types"
)

type TokenValidator interface {
	ValidateToken(token string) (*types.AuthContext, error)
}

type UserAuthMiddleware struct {
	tokenValidator TokenValidator
}

func NewUserAuthMiddleware(tokenValidator TokenValidator) *UserAuthMiddleware {
	return &UserAuthMiddleware{tokenValidator: tokenValidator}
}

func (m *UserAuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeUserAuthError(w, http.StatusUnauthorized, UserErrorUnauthorized, "Missing or invalid authorization header")
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" {
			writeUserAuthError(w, http.StatusUnauthorized, UserErrorUnauthorized, "Missing or invalid authorization header")
			return
		}
		if m.tokenValidator == nil {
			writeUserAuthError(w, http.StatusInternalServerError, UserErrorInternal, "internal server error")
			return
		}
		authCtx, err := m.tokenValidator.ValidateToken(token)
		if err != nil || authCtx == nil {
			writeUserAuthError(w, http.StatusUnauthorized, UserErrorUnauthorized, "authentication failed")
			return
		}
		ctx := context.WithValue(r.Context(), middleware.AuthContextKey, authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeUserAuthError(w http.ResponseWriter, status int, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(UserResponse{Success: false, Error: &UserRespErr{Code: code, Message: message}})
}

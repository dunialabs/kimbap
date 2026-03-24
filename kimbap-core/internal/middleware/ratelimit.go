package middleware

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	logservice "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/types"
)

type rateLimiter interface {
	CheckRateLimit(userID string, limit int) (allowed bool, remaining int, resetTime time.Time)
}

type RateLimitMiddleware struct {
	service rateLimiter
}

func NewRateLimitMiddleware(service rateLimiter, _ int) *RateLimitMiddleware {
	return &RateLimitMiddleware{service: service}
}

func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.service == nil {
			next.ServeHTTP(w, r)
			return
		}
		authContext, ok := GetAuthContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		limit := authContext.RateLimit
		if limit <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		allowed, remaining, resetTime := m.service.CheckRateLimit(authContext.UserID, limit)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(remaining, 0)))
		w.Header().Set("X-RateLimit-Reset", resetTime.UTC().Format("2006-01-02T15:04:05.000Z"))

		if !allowed {
			retryAfter := int(math.Ceil(time.Until(resetTime).Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))

			sessionID := mcpSessionIDFromHeader(r)
			ip := ClientIPFromRequest(r)
			logservice.GetLogService().EnqueueLog(database.Log{
				Action:    types.MCPEventLogTypeAuthRateLimit,
				UserID:    authContext.UserID,
				SessionID: sessionID,
				IP:        ip,
				UA:        r.Header.Get("User-Agent"),
				TokenMask: authContext.Token,
				Error:     fmt.Sprintf("Rate limit exceeded: %d requests/min, currentCount: %d", limit, limit-remaining),
			})

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"error": map[string]any{
					"code":    -32603,
					"message": "Rate limit exceeded",
					"details": map[string]any{
						"rateLimit":  limit,
						"retryAfter": retryAfter,
					},
				},
				"id": nil,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

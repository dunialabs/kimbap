package types

import (
	"time"

	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
)

type AuthContext struct {
	Kind            string               `json:"kind,omitempty"`
	UserID          string               `json:"userId"`
	Token           string               `json:"token"`
	Role            int                  `json:"role"`
	Status          int                  `json:"status"`
	Permissions     mcptypes.Permissions `json:"permissions"`
	UserPreferences mcptypes.Permissions `json:"userPreferences"`
	LaunchConfigs   string               `json:"launchConfigs"`
	AuthenticatedAt time.Time            `json:"authenticatedAt"`
	ExpiresAt       *int64               `json:"expiresAt"`
	RateLimit       int                  `json:"rateLimit"`
	OAuthClientID   string               `json:"oauthClientId,omitempty"`
	OAuthScopes     []string             `json:"oauthScopes,omitempty"`
	UserAgent       string               `json:"userAgent,omitempty"`
}

const (
	AuthErrorTypeInvalidToken      = "INVALID_TOKEN"
	AuthErrorTypeUserNotFound      = "USER_NOT_FOUND"
	AuthErrorTypeUserDisabled      = "USER_DISABLED"
	AuthErrorTypeUserExpired       = "USER_EXPIRED"
	AuthErrorTypeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

type AuthError struct {
	Type    string
	Message string
	UserID  string
	Details interface{}
}

func (e *AuthError) Error() string {
	return e.Message
}

func NewAuthError(errType, message, userID string, details interface{}) *AuthError {
	return &AuthError{Type: errType, Message: message, UserID: userID, Details: details}
}

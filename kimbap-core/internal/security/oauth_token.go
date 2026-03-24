package security

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	coretypes "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/golang-jwt/jwt/v5"
)

type OAuthTokenRecord struct {
	AccessToken string
	UserID      string
	ClientID    string
	Scopes      []string
	ExpiresAt   time.Time
	Revoked     bool
}

type OAuthTokenRepository interface {
	FindByAccessToken(ctx context.Context, accessToken string) (*OAuthTokenRecord, error)
}

type UserRecord struct {
	UserID    string
	Status    int
	ExpiresAt int64
}

type UserRepository interface {
	FindByUserID(ctx context.Context, userID string) (*UserRecord, error)
}

type OAuthTokenValidator struct {
	repository     OAuthTokenRepository
	userRepository UserRepository
}

func NewOAuthTokenValidator(repository OAuthTokenRepository, userRepository UserRepository) *OAuthTokenValidator {
	return &OAuthTokenValidator{repository: repository, userRepository: userRepository}
}

func (v *OAuthTokenValidator) ValidateOAuthToken(ctx context.Context, accessToken string) (userID, clientID string, scopes []string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if accessToken == "" {
		return "", "", nil, errors.New("access token is required")
	}
	if v.repository == nil {
		return "", "", nil, errors.New("oauth token repository is not configured")
	}
	if v.userRepository == nil {
		return "", "", nil, errors.New("user repository is not configured")
	}

	claims, parseErr := parseAndVerifyOAuthJWT(accessToken)
	if parseErr != nil {
		return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeInvalidToken, fmt.Sprintf("invalid oauth token: %v", parseErr), "", parseErr)
	}
	if expectedAud := expectedOAuthAudience(); expectedAud != "" {
		audiences := claimAudiences(claims)
		if !containsAudience(audiences, expectedAud) {
			return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeInvalidToken, "invalid oauth token audience", "", nil)
		}
	}

	record, dbErr := v.repository.FindByAccessToken(ctx, accessToken)
	if dbErr != nil {
		return "", "", nil, fmt.Errorf("failed to validate oauth token: %w", dbErr)
	}
	if record == nil {
		return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeInvalidToken, "token not found", "", nil)
	}
	if record.Revoked {
		return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeInvalidToken, "token revoked", "", nil)
	}
	if !record.ExpiresAt.IsZero() && time.Now().After(record.ExpiresAt) {
		return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeInvalidToken, "token expired", "", nil)
	}

	claimUserID := claimString(claims, "user_id")
	user, userErr := v.userRepository.FindByUserID(ctx, claimUserID)
	if userErr != nil {
		return "", "", nil, fmt.Errorf("failed to validate oauth token: %w", userErr)
	}
	if user == nil {
		return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeUserNotFound, "user not found", claimUserID, nil)
	}
	if user.Status != coretypes.UserStatusEnabled {
		return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeUserDisabled, "user is disabled", user.UserID, map[string]any{"status": user.Status})
	}
	if user.ExpiresAt > 0 && time.Now().Unix() > user.ExpiresAt {
		return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeUserExpired, "user authorization has expired", user.UserID, map[string]any{"expiresAt": user.ExpiresAt})
	}

	userID = claimString(claims, "user_id")
	clientID = claimString(claims, "client_id")
	scopes = claimScopes(claims)

	if userID == "" || clientID == "" {
		return "", "", nil, coretypes.NewAuthError(coretypes.AuthErrorTypeInvalidToken, "invalid oauth token metadata", claimUserID, nil)
	}

	return userID, clientID, scopes, nil
}

func parseAndVerifyOAuthJWT(token string) (jwt.MapClaims, error) {
	if strings.Count(token, ".") != 2 {
		return nil, errors.New("token is not jwt format")
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET environment variable is required")
	}

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token signature")
	}
	if typ, _ := claims["type"].(string); typ != "access_token" {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}

func claimString(claims jwt.MapClaims, keys ...string) string {
	for _, k := range keys {
		if v, ok := claims[k]; ok {
			switch raw := v.(type) {
			case string:
				if raw != "" {
					return raw
				}
			case []string:
				if len(raw) > 0 {
					return raw[0]
				}
			case []interface{}:
				if len(raw) > 0 {
					if s, ok := raw[0].(string); ok {
						return s
					}
				}
			}
		}
	}
	return ""
}

func claimScopes(claims jwt.MapClaims) []string {
	if raw, ok := claims["scopes"]; ok {
		switch v := raw.(type) {
		case []string:
			return append([]string(nil), v...)
		case []interface{}:
			out := make([]string, 0, len(v))
			for _, scope := range v {
				if s, ok := scope.(string); ok {
					out = append(out, s)
				}
			}
			return out
		case string:
			if v == "" {
				return nil
			}
			return strings.Fields(v)
		}
	}
	return nil
}

func claimAudiences(claims jwt.MapClaims) []string {
	if raw, ok := claims["aud"]; ok {
		switch v := raw.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return []string{strings.TrimSpace(v)}
			}
		case []string:
			out := make([]string, 0, len(v))
			for _, item := range v {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			return out
		case []interface{}:
			out := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						out = append(out, s)
					}
				}
			}
			return out
		}
	}
	return nil
}

func containsAudience(audiences []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	for _, aud := range audiences {
		if strings.TrimSpace(aud) == expected {
			return true
		}
	}
	return false
}

func expectedOAuthAudience() string {
	base := strings.TrimSpace(config.Env("KIMBAP_PUBLIC_BASE_URL"))
	if base == "" {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" {
		return ""
	}
	if parsed.User != nil {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	normalized := strings.TrimRight(parsed.String(), "/")
	return normalized
}

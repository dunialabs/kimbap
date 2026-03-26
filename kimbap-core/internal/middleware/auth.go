package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/security"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

const (
	UserStatusEnabled        = 1
	defaultUserInfoRefresh   = 5 * time.Minute
	maxInitProbeBodyBytes    = 1 << 20
	userAPIErrorInvalidReq   = 1001
	userAPIErrorUnauthorized = 1002
	userAPIErrorUserDisabled = 1003
	userAPIErrorInternal     = 5001
)

var (
	errAuthorizationTokenRequired  = errors.New("authorization token is required")
	errUserRepositoryNotConfigured = errors.New("user repository is not configured")
	errInvalidPermissionsFormat    = errors.New("invalid permissions format")
	errInvalidPermissionsStructure = errors.New("invalid permissions structure")
)

type User struct {
	UserID           string
	Role             int
	Status           int
	ExpiresAtUnix    int64
	RateLimit        int
	Permissions      json.RawMessage
	UserPreferences  json.RawMessage
	LaunchConfigs    json.RawMessage
	EncryptedToken   string
	OAuthClientID    string
	OAuthScopes      []string
	AdditionalFields map[string]any
}

type UserRepository interface {
	FindByUserID(ctx context.Context, userID string) (*User, error)
}

type tokenValidator interface {
	ValidateToken(token string) (string, error)
}

type AuthMiddleware struct {
	tokenValidator  tokenValidator
	oauthValidator  *security.OAuthTokenValidator
	userRepository  UserRepository
	refreshInterval time.Duration
	nowFn           func() time.Time
	db              *gorm.DB
}

func NewAuthMiddleware(tv tokenValidator, oauth *security.OAuthTokenValidator, repo UserRepository, db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator:  tv,
		oauthValidator:  oauth,
		userRepository:  repo,
		refreshInterval: defaultUserInfoRefresh,
		nowFn:           time.Now,
		db:              db,
	}
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authContext, err := m.AuthenticateRequest(r)
		if err != nil {
			safeMsg := sanitizeAuthError(err)
			status := authStatusCodeForError(err)
			if strings.HasPrefix(r.URL.Path, "/user") {
				writeUserAuthError(w, status, safeMsg)
				return
			}
			writeJSONError(w, status, safeMsg)
			return
		}
		if authContext == nil {
			if strings.HasPrefix(r.URL.Path, "/user") {
				r.Body = http.MaxBytesReader(w, r.Body, maxInitProbeBodyBytes)
				bodyBytes, readErr := io.ReadAll(r.Body)
				if readErr == nil {
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					if len(strings.TrimSpace(string(bodyBytes))) > 0 {
						var payload any
						if unmarshalErr := json.Unmarshal(bodyBytes, &payload); unmarshalErr != nil {
							writeUserInvalidJSONLikeExpress(w, unmarshalErr)
							return
						}
					}
				} else {
					var maxBytesErr *http.MaxBytesError
					if errors.As(readErr, &maxBytesErr) {
						WriteRequestEntityTooLargeLikeExpress(w)
						return
					}
				}
				writeUserAuthError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		authContext.UserAgent = strings.TrimSpace(r.Header.Get("User-Agent"))

		ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func mcpSessionIDFromHeader(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("Mcp-Session-Id")); v != "" {
		return v
	}
	return strings.TrimSpace(r.Header.Get("mcp-session-id"))
}

func WriteRequestEntityTooLargeLikeExpress(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	_, _ = w.Write([]byte("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Error</title></head><body><pre>PayloadTooLargeError: request entity too large</pre></body></html>"))
}

func (m *AuthMiddleware) AuthenticateRequest(r *http.Request) (*types.AuthContext, error) {
	if m.userRepository == nil {
		return nil, errUserRepositoryNotConfigured
	}

	token, err := ExtractAuthToken(r)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, nil
	}

	userID, oauthClientID, oauthScopes, err := m.validateToken(r.Context(), token)
	if err != nil {
		internallog.GetLogService().EnqueueLog(database.Log{
			Action: types.MCPEventLogTypeAuthError,
			Error:  "Token validation failed",
		})
		return nil, err
	}

	user, err := m.userRepository.FindByUserID(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, types.NewAuthError(types.AuthErrorTypeUserNotFound, "user not found", userID, nil)
	}
	if user.Status != UserStatusEnabled {
		return nil, types.NewAuthError(types.AuthErrorTypeUserDisabled, "user is disabled", user.UserID, map[string]any{"status": user.Status})
	}

	if user.EncryptedToken != "" {
		if !security.VerifyTokenAgainstEncrypted(token, user.EncryptedToken) {
			return nil, types.NewAuthError(types.AuthErrorTypeInvalidToken, "token has been revoked or rotated", user.UserID, nil)
		}
	}

	now := m.nowFn().Unix()
	if user.ExpiresAtUnix > 0 && now > user.ExpiresAtUnix {
		return nil, types.NewAuthError(types.AuthErrorTypeUserExpired, "user authorization has expired", user.UserID, map[string]any{"expiresAt": user.ExpiresAtUnix})
	}

	var expiresAt *int64
	if user.ExpiresAtUnix > 0 {
		expiresAt = &user.ExpiresAtUnix
	}

	permissions, permErr := decodePermissions(user.Permissions)
	if permErr != nil {
		return nil, permErr
	}
	userPreferences, prefErr := decodePermissions(defaultJSON(user.UserPreferences))
	if prefErr != nil {
		return nil, prefErr
	}

	ctx := &types.AuthContext{
		UserID:          user.UserID,
		Token:           maskToken(token),
		Role:            user.Role,
		Status:          user.Status,
		Permissions:     permissions,
		UserPreferences: userPreferences,
		LaunchConfigs:   string(user.LaunchConfigs),
		AuthenticatedAt: m.nowFn(),
		ExpiresAt:       expiresAt,
		RateLimit:       user.RateLimit,
		OAuthClientID:   oauthClientID,
		OAuthScopes:     oauthScopes,
	}

	return ctx, nil
}

func GetAuthContext(ctx context.Context) (*types.AuthContext, bool) {
	value := ctx.Value(AuthContextKey)
	authContext, ok := value.(*types.AuthContext)
	return authContext, ok
}

func (m *AuthMiddleware) validateToken(ctx context.Context, token string) (userID, clientID string, scopes []string, err error) {
	isJWTFormat := strings.Count(token, ".") == 2
	isHexFormat := security.IsTraditionalTokenFormat(token)

	if isJWTFormat {
		if m.oauthValidator != nil {
			userID, clientID, scopes, err = m.oauthValidator.ValidateOAuthToken(ctx, token)
			return userID, clientID, scopes, err
		}

		if m.tokenValidator == nil {
			return "", "", nil, errors.New("token validator is not configured")
		}
		userID, err = m.tokenValidator.ValidateToken(token)
		if err == nil {
			return userID, "", nil, nil
		}
		return "", "", nil, types.NewAuthError(types.AuthErrorTypeInvalidToken, "invalid or expired token", "", nil)
	}

	if isHexFormat {
		if m.tokenValidator == nil {
			return "", "", nil, errors.New("token validator is not configured")
		}
		userID, err = m.tokenValidator.ValidateToken(token)
		if err == nil {
			return userID, "", nil, nil
		}
		return "", "", nil, types.NewAuthError(types.AuthErrorTypeInvalidToken, "invalid or expired token", "", nil)
	}

	if m.oauthValidator != nil {
		userID, clientID, scopes, err = m.oauthValidator.ValidateOAuthToken(ctx, token)
		if err == nil {
			return userID, clientID, scopes, nil
		}
	}

	if m.tokenValidator == nil {
		return "", "", nil, errors.New("token validator is not configured")
	}
	userID, err = m.tokenValidator.ValidateToken(token)
	if err == nil {
		return userID, "", nil, nil
	}

	return "", "", nil, types.NewAuthError(types.AuthErrorTypeInvalidToken, "invalid or expired token", "", nil)
}

func ExtractAuthToken(r *http.Request) (string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.Fields(authHeader)
	if len(parts) > 0 && strings.EqualFold(parts[0], "Bearer") {
		token := ""
		if len(parts) > 1 {
			token = strings.TrimSpace(strings.Join(parts[1:], " "))
		}
		if token != "" {
			return token, nil
		}

		return "", errAuthorizationTokenRequired
	}

	return "", nil
}

func setWWWAuthenticateIfUnauthorized(w http.ResponseWriter, r *http.Request, status int, errCode string, errDescription string) {
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", BuildWWWAuthenticateHeader(r, errCode, errDescription))
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    status,
			"message": message,
		},
	})
}

func writeUserAuthError(w http.ResponseWriter, status int, message string) {
	code := userAPIErrorCode(status, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeUserInvalidJSONLikeExpress(w http.ResponseWriter, _ error) {
	for _, header := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Expose-Headers",
		"Access-Control-Max-Age",
		"Vary",
	} {
		w.Header().Del(header)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Error</title></head><body><pre>SyntaxError: invalid JSON in request body</pre></body></html>"))
}

func userAPIErrorCode(status int, message string) int {
	switch status {
	case http.StatusBadRequest:
		return userAPIErrorInvalidReq
	case http.StatusUnauthorized:
		return userAPIErrorUnauthorized
	case http.StatusForbidden:
		msg := strings.ToLower(message)
		if strings.Contains(msg, "disabled") {
			return userAPIErrorUserDisabled
		}
		return userAPIErrorUnauthorized
	default:
		if status >= http.StatusInternalServerError {
			return userAPIErrorInternal
		}
		return userAPIErrorInternal
	}
}

func maskToken(token string) string {
	if len(token) <= 16 {
		return token
	}
	return token[:8] + "..." + token[len(token)-8:]
}

func defaultJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}

func decodePermissions(raw json.RawMessage) (mcptypes.Permissions, error) {
	permissions := mcptypes.Permissions{}
	if len(raw) == 0 {
		return permissions, nil
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return mcptypes.Permissions{}, fmt.Errorf("%w: null", errInvalidPermissionsFormat)
	}
	if err := json.Unmarshal(raw, &permissions); err != nil {
		return mcptypes.Permissions{}, fmt.Errorf("%w: %w", errInvalidPermissionsFormat, err)
	}
	if permissions == nil {
		return mcptypes.Permissions{}, fmt.Errorf("%w: null", errInvalidPermissionsFormat)
	}
	if err := validatePermissions(permissions); err != nil {
		return mcptypes.Permissions{}, fmt.Errorf("%w: %w", errInvalidPermissionsStructure, err)
	}
	return permissions, nil
}

// validatePermissions checks the structural integrity of decoded permissions.
// Mirrors TS isValidPermissions(): each server entry must have an 'enabled' boolean,
// and optional tools/resources/prompts must be maps if present.
func validatePermissions(perms mcptypes.Permissions) error {
	for serverID := range perms {
		if strings.TrimSpace(serverID) == "" {
			return fmt.Errorf("permissions contains empty server id")
		}
	}
	return nil
}

func authStatusCodeForError(err error) int {
	if err == nil {
		return http.StatusUnauthorized
	}
	var authErr *types.AuthError
	if errors.As(err, &authErr) {
		switch authErr.Type {
		case types.AuthErrorTypeUserDisabled, types.AuthErrorTypeUserExpired:
			return http.StatusForbidden
		case types.AuthErrorTypeRateLimitExceeded:
			return http.StatusTooManyRequests
		default:
			return http.StatusUnauthorized
		}
	}
	switch {
	case errors.Is(err, errAuthorizationTokenRequired):
		return http.StatusUnauthorized
	case errors.Is(err, errUserRepositoryNotConfigured),
		errors.Is(err, errInvalidPermissionsFormat),
		errors.Is(err, errInvalidPermissionsStructure):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func BuildWWWAuthenticateHeader(r *http.Request, errCode string, errDescription string) string {
	metadataURL := ""
	if canonical := canonicalAuthPublicURL(); canonical != "" {
		metadataURL = canonical + "/.well-known/oauth-protected-resource"
	}
	escapedDesc := strings.ReplaceAll(strings.ReplaceAll(errDescription, `\`, `\\`), `"`, `\"`)
	header := `Bearer realm="kimbap-core", error="` + errCode + `", error_description="` + escapedDesc + `"`
	if metadataURL != "" {
		header += `, resource_metadata="` + metadataURL + `"`
	}
	return header
}

func canonicalAuthPublicURL() string {
	canonical := strings.TrimSpace(config.Env("KIMBAP_PUBLIC_BASE_URL"))
	if canonical == "" {
		return ""
	}
	parsed, err := url.Parse(canonical)
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
	return strings.TrimRight(parsed.String(), "/")
}

func sanitizeAuthError(err error) string {
	if err == nil {
		return "authentication failed"
	}
	var authErr *types.AuthError
	if errors.As(err, &authErr) {
		switch authErr.Type {
		case types.AuthErrorTypeInvalidToken:
			return "invalid or expired token"
		case types.AuthErrorTypeUserNotFound:
			return "user not found"
		case types.AuthErrorTypeUserDisabled:
			return "user is disabled"
		case types.AuthErrorTypeUserExpired:
			return "user authorization has expired"
		case types.AuthErrorTypeRateLimitExceeded:
			return "rate limit exceeded"
		default:
			return "authentication failed"
		}
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "authorization token is required"):
		return "authorization token is required"
	case strings.Contains(msg, "missing or invalid authorization"):
		return "missing or invalid authorization header"
	default:
		return "authentication failed"
	}
}

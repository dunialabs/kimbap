package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/security"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

var traditionalTokenPattern = regexp.MustCompile(`^[a-f0-9]{128}$`)
var authMiddlewareLogger = logger.CreateLogger("AuthMiddleware")

const (
	UserStatusEnabled        = 1
	defaultUserInfoRefresh   = 5 * time.Minute
	maxInitProbeBodyBytes    = 1 << 20
	userAPIErrorInvalidReq   = 1001
	userAPIErrorUnauthorized = 1002
	userAPIErrorUserDisabled = 1003
	userAPIErrorInternal     = 5001
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
		if strings.HasPrefix(r.URL.Path, "/mcp") {
			sessionID := mcpSessionIDFromHeader(r)
			if sessionID != "" {
				session := core.SessionStoreInstance().GetSession(sessionID)
				if session == nil {
					if r.Method == http.MethodDelete {
						next.ServeHTTP(w, r)
						return
					}
					writeJSONRPCConnectionClosed(w, http.StatusBadRequest, "Bad Request: No valid session ID provided")
					return
				}
				token := strings.TrimSpace(session.Token)
				if token != "" && strings.Count(token, ".") == 2 && m.oauthValidator != nil {
					if _, _, _, err := m.oauthValidator.ValidateOAuthToken(r.Context(), token); err != nil {
						core.SessionStoreInstance().RemoveAllUserSessions(session.UserID, mcptypes.DisconnectReasonSessionRemoved)
						status := http.StatusUnauthorized
						safeMsg := sanitizeAuthError(err)
						setWWWAuthenticateIfUnauthorized(w, r, status, "invalid_token", safeMsg)
						writeJSONRPCConnectionClosed(w, status, safeMsg)
						return
					}
				}

				m.refreshUserInfoIfNeeded(r.Context(), session)

				sessionAuthContext := session.AuthContextSnapshot()
				if sessionAuthContext.Status != UserStatusEnabled {
					respondUserDisabled(w, r, session.UserID)
					return
				}

				now := m.nowFn()
				if session.IsExpired(now, 0) {
					if m.userRepository != nil {
						if user, err := m.userRepository.FindByUserID(r.Context(), session.UserID); err == nil && user != nil {
							if user.Status != UserStatusEnabled {
								respondUserDisabled(w, r, session.UserID)
								return
							}
							if user.ExpiresAtUnix <= 0 || now.Unix() <= user.ExpiresAtUnix {
								session.UpdateExpiresAt(user.ExpiresAtUnix)
							} else {
								respondUserExpired(w, r, session.UserID)
								return
							}
						} else {
							respondUserExpired(w, r, session.UserID)
							return
						}
					}

					if session.IsExpired(m.nowFn(), 0) {
						respondUserExpired(w, r, session.UserID)
						return
					}
				}

				authContext := &types.AuthContext{
					Kind:            sessionAuthContext.Kind,
					UserID:          sessionAuthContext.UserID,
					Role:            sessionAuthContext.Role,
					Status:          sessionAuthContext.Status,
					Permissions:     sessionAuthContext.Permissions,
					UserPreferences: sessionAuthContext.UserPreferences,
					LaunchConfigs:   sessionAuthContext.LaunchConfigs,
					UserAgent:       sessionAuthContext.UserAgent,
					AuthenticatedAt: sessionAuthContext.AuthenticatedAt,
					RateLimit:       sessionAuthContext.RateLimit,
					OAuthClientID:   sessionAuthContext.OAuthClientID,
					OAuthScopes:     sessionAuthContext.OAuthScopes,
				}
				if sessionAuthContext.ExpiresAt > 0 {
					expires := sessionAuthContext.ExpiresAt
					authContext.ExpiresAt = &expires
				}

				session.Touch()
				w.Header().Set("Mcp-Session-Id", sessionID)
				w.Header().Set("mcp-session-id", sessionID)
				ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
		}

		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/mcp") {
			sessionID := mcpSessionIDFromHeader(r)
			if sessionID == "" {
				r.Body = http.MaxBytesReader(w, r.Body, maxInitProbeBodyBytes)
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					var maxBytesErr *http.MaxBytesError
					if errors.As(err, &maxBytesErr) {
						WriteRequestEntityTooLargeLikeExpress(w)
						return
					}
					writeJSONRPCConnectionClosed(w, http.StatusBadRequest, "Bad Request: Invalid request body")
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

				isInitialize, validJSON := isInitializeRequestBody(bodyBytes)
				if !isInitialize {
					if !validJSON {
						writeJSONRPCConnectionClosed(w, http.StatusBadRequest, "Bad Request: Invalid JSON")
						return
					}
					writeJSONRPCConnectionClosed(w, http.StatusBadRequest, "Bad Request: Server not initialized")
					return
				}
			}
		}

		if r.Method == http.MethodHead && strings.HasPrefix(r.URL.Path, "/mcp") {
			next.ServeHTTP(w, r)
			return
		}

		authContext, err := m.AuthenticateRequest(r)
		if err != nil {
			safeMsg := sanitizeAuthError(err)
			if strings.HasPrefix(r.URL.Path, "/mcp") {
				status := authStatusCodeForError(err)
				if status == http.StatusUnauthorized {
					w.Header().Set("WWW-Authenticate", BuildWWWAuthenticateHeader(r, "invalid_token", safeMsg))
				}
				writeJSONRPCConnectionClosed(w, status, safeMsg)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/user") {
				status := authStatusCodeForError(err)
				writeUserAuthError(w, status, safeMsg)
				return
			}
			writeJSONError(w, http.StatusUnauthorized, safeMsg)
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
			if strings.HasPrefix(r.URL.Path, "/mcp") {
				// If there was an actual auth attempt (Authorization header present), reject
				if hasAuthorizationHeader(r) {
					message := "Authorization header with Bearer token is required"
					w.Header().Set("WWW-Authenticate", BuildWWWAuthenticateHeader(r, "invalid_request", message))
					writeJSONRPCAuthError(w, http.StatusUnauthorized, message)
					return
				}
				// No auth header at all — try anonymous access
				anonCtx := m.handleAnonymousAccess(r)
				if anonCtx != nil {
					anonCtx.UserAgent = strings.TrimSpace(r.Header.Get("User-Agent"))
					ctx := context.WithValue(r.Context(), AuthContextKey, anonCtx)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				message := "Authorization header with Bearer token is required"
				w.Header().Set("WWW-Authenticate", BuildWWWAuthenticateHeader(r, "invalid_request", message))
				writeJSONRPCAuthError(w, http.StatusUnauthorized, message)
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

func (m *AuthMiddleware) refreshUserInfoIfNeeded(ctx context.Context, session *core.ClientSession) {
	if m.userRepository == nil || session == nil {
		return
	}
	// Skip refresh for anonymous sessions — no DB-backed user to refresh
	authCtx := session.AuthContextSnapshot()
	if authCtx.Kind == "anonymous" {
		return
	}

	nowMillis := m.nowFn().UnixMilli()
	lastRefresh := session.GetLastUserInfoRefresh()
	refreshIntervalMillis := m.refreshInterval.Milliseconds()

	if lastRefresh > 0 && refreshIntervalMillis > 0 && (nowMillis-lastRefresh) < refreshIntervalMillis {
		return
	}

	if err := m.refreshUserInfo(ctx, session); err != nil {
		authMiddlewareLogger.Warn().Err(err).Str("sessionId", session.SessionID).Msg("failed to refresh user info for session")
		var authErr *types.AuthError
		if errors.As(err, &authErr) {
			switch authErr.Type {
			case types.AuthErrorTypeUserNotFound, types.AuthErrorTypeUserDisabled, types.AuthErrorTypeUserExpired:
				updatedAuthContext := session.AuthContextSnapshot()
				updatedAuthContext.Status = 0
				session.UpdateAuthContext(updatedAuthContext)
			}
		}
		session.UpdateLastUserInfoRefresh(nowMillis)
		return
	}

	session.UpdateLastUserInfoRefresh(nowMillis)
}

func (m *AuthMiddleware) refreshUserInfo(ctx context.Context, session *core.ClientSession) error {
	if m.userRepository == nil || session == nil {
		return nil
	}

	user, err := m.userRepository.FindByUserID(ctx, session.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return types.NewAuthError(types.AuthErrorTypeUserNotFound, "user not found", session.UserID, nil)
	}

	updatedAuthContext := session.AuthContextSnapshot()
	updatedAuthContext.UserID = user.UserID
	updatedAuthContext.Role = user.Role
	updatedAuthContext.Status = user.Status
	permissions, permErr := decodePermissions(user.Permissions)
	if permErr != nil {
		return fmt.Errorf("invalid permissions format during refresh: %w", permErr)
	}
	userPreferences, prefErr := decodePermissions(defaultJSON(user.UserPreferences))
	if prefErr != nil {
		return fmt.Errorf("invalid userPreferences format during refresh: %w", prefErr)
	}
	updatedAuthContext.Permissions = permissions
	updatedAuthContext.UserPreferences = userPreferences
	updatedAuthContext.LaunchConfigs = string(user.LaunchConfigs)
	if user.ExpiresAtUnix > 0 {
		updatedAuthContext.ExpiresAt = user.ExpiresAtUnix
	} else {
		updatedAuthContext.ExpiresAt = 0
	}
	updatedAuthContext.RateLimit = user.RateLimit

	session.UpdateAuthContext(updatedAuthContext)
	return nil
}

func (m *AuthMiddleware) handleAnonymousAccess(r *http.Request) *types.AuthContext {
	if m.db == nil {
		return nil
	}

	// Origin validation (MCP spec compliance — block non-HTTP(S) origins)
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil
		}
	}

	// Query all anonymously-accessible servers
	var servers []database.Server
	result := m.db.Where("anonymous_access = ? AND enabled = ? AND allow_user_input = ?", true, true, false).
		Select("server_id", "server_name", "auth_type", "anonymous_rate_limit").
		Find(&servers)
	if result.Error != nil || len(servers) == 0 {
		return nil
	}

	// Build permissions for all qualifying servers
	permissions := mcptypes.Permissions{}
	minRateLimit := math.MaxInt32
	for _, server := range servers {
		permissions[server.ServerID] = mcptypes.ServerConfigWithEnabled{
			Enabled:        true,
			ServerName:     server.ServerName,
			AllowUserInput: false,
			AuthType:       server.AuthType,
			Tools:          map[string]mcptypes.ToolCapabilityConfig{},
			Resources:      map[string]mcptypes.ResourceCapabilityConfig{},
			Prompts:        map[string]mcptypes.PromptCapabilityConfig{},
			Configured:     false,
			ConfigTemplate: "{}",
		}
		if server.AnonymousRateLimit < minRateLimit {
			minRateLimit = server.AnonymousRateLimit
		}
	}
	if minRateLimit == math.MaxInt32 {
		minRateLimit = 10
	}

	// Build synthetic anonymous user identity
	clientIP := ClientIPFromRequest(r)
	ipHash := sha256Hex(clientIP)[:12]

	return &types.AuthContext{
		Kind:            "anonymous",
		UserID:          "anon:" + ipHash,
		Role:            types.UserRoleGuest,
		Status:          types.UserStatusEnabled,
		Permissions:     permissions,
		UserPreferences: mcptypes.Permissions{},
		LaunchConfigs:   "{}",
		AuthenticatedAt: time.Now(),
		RateLimit:       minRateLimit,
	}
}

func hasAuthorizationHeader(r *http.Request) bool {
	return strings.TrimSpace(r.Header.Get("Authorization")) != ""
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func WriteRequestEntityTooLargeLikeExpress(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	_, _ = w.Write([]byte("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Error</title></head><body><pre>PayloadTooLargeError: request entity too large</pre></body></html>"))
}

func (m *AuthMiddleware) AuthenticateRequest(r *http.Request) (*types.AuthContext, error) {
	if m.userRepository == nil {
		return nil, errors.New("user repository is not configured")
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
	isHexFormat := traditionalTokenPattern.MatchString(token)

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
		return "", "", nil, err
	}

	if isHexFormat {
		if m.tokenValidator == nil {
			return "", "", nil, errors.New("token validator is not configured")
		}
		userID, err = m.tokenValidator.ValidateToken(token)
		if err == nil {
			return userID, "", nil, nil
		}
		return "", "", nil, err
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

	return "", "", nil, err
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

		return "", errors.New("authorization token is required")
	}

	return "", nil
}

func respondUserExpired(w http.ResponseWriter, r *http.Request, userID string) {
	notifier := core.ServerManagerInstance().Notifier()
	if notifier != nil {
		notifier.NotifyUserExpired(userID)
	}
	core.SessionStoreInstance().RemoveAllUserSessions(userID, mcptypes.DisconnectReasonUserExpired)
	status := http.StatusForbidden
	msg := "User authorization has expired"
	setWWWAuthenticateIfUnauthorized(w, r, status, "invalid_token", msg)
	writeJSONRPCConnectionClosed(w, status, msg)
}

func respondUserDisabled(w http.ResponseWriter, r *http.Request, userID string) {
	notifier := core.ServerManagerInstance().Notifier()
	if notifier != nil {
		notifier.NotifyUserDisabled(userID, "User is disabled")
	}
	core.SessionStoreInstance().RemoveAllUserSessions(userID, mcptypes.DisconnectReasonUserDisabled)
	status := http.StatusForbidden
	msg := "User is disabled"
	setWWWAuthenticateIfUnauthorized(w, r, status, "invalid_token", msg)
	writeJSONRPCConnectionClosed(w, status, msg)
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

func writeUserInvalidJSONLikeExpress(w http.ResponseWriter, err error) {
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
		return mcptypes.Permissions{}, fmt.Errorf("invalid permissions format: null")
	}
	if err := json.Unmarshal(raw, &permissions); err != nil {
		return mcptypes.Permissions{}, fmt.Errorf("invalid permissions format: %w", err)
	}
	if permissions == nil {
		return mcptypes.Permissions{}, fmt.Errorf("invalid permissions format: null")
	}
	if err := validatePermissions(permissions); err != nil {
		return mcptypes.Permissions{}, fmt.Errorf("invalid permissions structure: %w", err)
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

func mcpSessionIDFromHeader(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("Mcp-Session-Id")); v != "" {
		return v
	}
	return strings.TrimSpace(r.Header.Get("mcp-session-id"))
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
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "disabled") || strings.Contains(msg, "expired") {
		return http.StatusForbidden
	}
	return http.StatusUnauthorized
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

func authPublicURL(r *http.Request) string {
	if canonical := canonicalAuthPublicURL(); canonical != "" {
		return canonical
	}

	protocol := "http"
	if r != nil && r.TLS != nil {
		protocol = "https"
	}
	host := ""
	if r != nil {
		host = r.Host
		if isTrustedForwardedAuthSource(r) {
			if xf := strings.TrimSpace(strings.SplitN(r.Header.Get("X-Forwarded-Host"), ",", 2)[0]); xf != "" {
				host = xf
			}
			xp := strings.ToLower(strings.TrimSpace(strings.SplitN(r.Header.Get("X-Forwarded-Proto"), ",", 2)[0]))
			if xp == "http" || xp == "https" {
				protocol = xp
			}
		}
	}
	return protocol + "://" + host
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

func isTrustedForwardedAuthSource(r *http.Request) bool {
	if r == nil {
		return false
	}
	addr := strings.TrimSpace(r.RemoteAddr)
	if addr == "" {
		return false
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func isInitializeRequestBody(body []byte) (bool, bool) {
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
		for _, item := range root {
			obj, ok := item.(map[string]any)
			if !ok {
				return false, true
			}
			method, _ := obj["method"].(string)
			if method != "initialize" {
				return false, true
			}
		}
		return true, true
	default:
		return false, true
	}
}

func writeJSONRPCConnectionClosed(w http.ResponseWriter, status int, message string) {
	writeJSONRPCError(w, status, -32000, message, nil)
}

func writeJSONRPCAuthError(w http.ResponseWriter, status int, message string) {
	writeJSONRPCError(w, status, -32000, message, nil)
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

func writeJSONRPCError(w http.ResponseWriter, status int, code int, message string, details any) {
	errorPayload := map[string]any{
		"code":    code,
		"message": message,
	}
	if details != nil {
		errorPayload["details"] = details
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"error":   errorPayload,
		"id":      nil,
	})
}

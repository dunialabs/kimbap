package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	oauthtypes "github.com/dunialabs/kimbap-core/internal/oauth/types"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type OAuthService struct {
	db        *gorm.DB
	jwtSecret string
}

type userTokenResolver func(ctx context.Context, token string) (string, error)

func NewOAuthService(db *gorm.DB) *OAuthService {
	if db == nil {
		db = database.DB
	}
	return &OAuthService{db: db, jwtSecret: os.Getenv("JWT_SECRET")}
}

func (s *OAuthService) HandleAuthorize(ctx context.Context, req oauthtypes.AuthorizationApprovalRequest, resolveUser userTokenResolver) (string, *oauthtypes.OAuthErrorResponse, int) {
	clientService := NewOAuthClientService(s.db, s)
	client, err := clientService.GetClient(ctx, req.ClientID)
	if err != nil {
		return "", &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if client == nil {
		return "", &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "Client not found"}, http.StatusBadRequest
	}
	if !s.ValidateRedirectURI(req.RedirectURI, client.RedirectURIs) {
		return "", &oauthtypes.OAuthErrorResponse{Error: "invalid_request", ErrorDescription: "Invalid redirect_uri"}, http.StatusBadRequest
	}
	if client.TokenEndpointAuthMethod == "none" {
		if strings.TrimSpace(req.CodeChallenge) == "" {
			return "", &oauthtypes.OAuthErrorResponse{Error: "invalid_request", ErrorDescription: "code_challenge is required for public clients"}, http.StatusBadRequest
		}
		if method := strings.TrimSpace(req.CodeChallengeMethod); method != "S256" {
			return "", &oauthtypes.OAuthErrorResponse{Error: "invalid_request", ErrorDescription: "code_challenge_method must be S256 for public clients"}, http.StatusBadRequest
		}
	}

	if !req.Approved {
		return s.buildErrorRedirectURL(req.RedirectURI, "access_denied", "User denied authorization", req.State), nil, http.StatusOK
	}
	if strings.TrimSpace(req.UserToken) == "" {
		return s.buildErrorRedirectURL(req.RedirectURI, "invalid_request", "User token is required", req.State), nil, http.StatusOK
	}
	if resolveUser == nil {
		return "", &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	userID, err := resolveUser(ctx, req.UserToken)
	if err != nil || userID == "" {
		return s.buildErrorRedirectURL(req.RedirectURI, "invalid_request", "Invalid user token", req.State), nil, http.StatusOK
	}

	scopes := s.ParseScope(req.Scope)
	if !s.isScopeSubset(scopes, client.Scopes) {
		return s.buildErrorRedirectURL(req.RedirectURI, "invalid_scope", "Requested scope exceeds client allowed scopes", req.State), nil, http.StatusOK
	}
	code, err := s.generateAuthorizationCode()
	if err != nil {
		return "", &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	expiresAt := time.Now().Add(oauthtypes.AuthorizationCodeLifetime * time.Second)
	scopeJSON, _ := json.Marshal(scopes)

	var codeChallenge *string
	if req.CodeChallenge != "" {
		codeChallenge = &req.CodeChallenge
	}
	var challengeMethod *string
	if req.CodeChallengeMethod != "" {
		challengeMethod = &req.CodeChallengeMethod
	}
	var resource *string
	if req.Resource != "" {
		resource = &req.Resource
	}

	record := database.OAuthAuthorizationCode{
		Code:            code,
		ClientID:        req.ClientID,
		UserID:          userID,
		RedirectURI:     req.RedirectURI,
		Scopes:          scopeJSON,
		CodeChallenge:   codeChallenge,
		ChallengeMethod: challengeMethod,
		Resource:        resource,
		ExpiresAt:       expiresAt,
		CreatedAt:       time.Now(),
		Used:            false,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return "", &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	return s.buildSuccessRedirectURL(req.RedirectURI, code, req.State), nil, http.StatusOK
}

func (s *OAuthService) HandleToken(ctx context.Context, req oauthtypes.TokenRequest, authHeader string, clientService *OAuthClientService) (*oauthtypes.TokenResponse, *oauthtypes.OAuthErrorResponse, int) {
	if clientService == nil {
		clientService = NewOAuthClientService(s.db, s)
	}
	if req.GrantType == "" {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "unsupported_grant_type", ErrorDescription: "grant_type is required"}, http.StatusBadRequest
	}

	clientID := req.ClientID
	clientSecret := req.ClientSecret
	if basic := s.parseBasicAuth(authHeader); basic != nil {
		clientID = basic.ClientID
		clientSecret = basic.ClientSecret
	}

	switch req.GrantType {
	case "authorization_code":
		return s.handleAuthorizationCodeGrant(ctx, req, clientID, clientSecret, clientService)
	case "refresh_token":
		return s.handleRefreshTokenGrant(ctx, req, clientID, clientSecret, clientService)
	default:
		return nil, &oauthtypes.OAuthErrorResponse{Error: "unsupported_grant_type", ErrorDescription: "Grant type is not supported"}, http.StatusBadRequest
	}
}

func (s *OAuthService) authenticateClient(ctx context.Context, reqClientID, reqClientSecret, authHeader string, clientService *OAuthClientService) (string, *oauthtypes.OAuthErrorResponse, int) {
	clientID := reqClientID
	clientSecret := reqClientSecret
	if basic := s.parseBasicAuth(authHeader); basic != nil {
		clientID = basic.ClientID
		clientSecret = basic.ClientSecret
	}
	if clientID != "" {
		client, err := clientService.GetClient(ctx, clientID)
		if err != nil {
			return "", &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
		}
		if client == nil {
			return "", &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "Client not found"}, http.StatusUnauthorized
		}
		if client.TokenEndpointAuthMethod != "none" && clientSecret == "" {
			return "", &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "client_secret is required for confidential clients"}, http.StatusUnauthorized
		}
		if client.TokenEndpointAuthMethod != "none" {
			valid, err := clientService.VerifyClientCredentials(ctx, clientID, clientSecret)
			if err != nil {
				return "", &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
			}
			if !valid {
				return "", &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "Invalid client credentials"}, http.StatusUnauthorized
			}
		}
	}
	return clientID, nil, 0
}

func (s *OAuthService) HandleIntrospect(ctx context.Context, req oauthtypes.TokenIntrospectionRequest, authHeader string, clientService *OAuthClientService) (map[string]any, *oauthtypes.OAuthErrorResponse, int) {
	if req.Token == "" {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_request", ErrorDescription: "token is required"}, http.StatusBadRequest
	}
	if req.TokenTypeHint != "" && req.TokenTypeHint != "access_token" {
		return map[string]any{"active": false}, nil, http.StatusOK
	}
	if clientService == nil {
		clientService = NewOAuthClientService(s.db, s)
	}

	clientID, oauthErr, statusCode := s.authenticateClient(ctx, req.ClientID, req.ClientSecret, authHeader, clientService)
	if oauthErr != nil {
		return nil, oauthErr, statusCode
	}
	if strings.TrimSpace(clientID) == "" {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "client authentication is required"}, http.StatusUnauthorized
	}
	client, err := clientService.GetClient(ctx, clientID)
	if err != nil {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if client == nil || client.TokenEndpointAuthMethod == "none" {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "client authentication is required"}, http.StatusUnauthorized
	}

	verified := s.verifyAccessToken(req.Token)
	if !verified.Valid || verified.Payload == nil {
		return map[string]any{"active": false}, nil, http.StatusOK
	}
	payload := verified.Payload
	if clientID != "" {
		if cid, _ := payload["client_id"].(string); cid != clientID {
			return map[string]any{"active": false}, nil, http.StatusOK
		}
	}

	var tokenRecord database.OAuthToken
	if err := s.db.WithContext(ctx).Where("access_token = ?", hashRefreshToken(req.Token)).First(&tokenRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]any{"active": false}, nil, http.StatusOK
		}
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if tokenRecord.Revoked {
		return map[string]any{"active": false}, nil, http.StatusOK
	}
	if time.Now().After(tokenRecord.AccessTokenExpiresAt) {
		return map[string]any{"active": false}, nil, http.StatusOK
	}

	if tokenRecord.UserID == "" {
		return map[string]any{"active": false}, nil, http.StatusOK
	}

	var user database.User
	if err := s.db.WithContext(ctx).Where("user_id = ?", tokenRecord.UserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]any{"active": false}, nil, http.StatusOK
		}
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if user.Status != types.UserStatusEnabled {
		return map[string]any{"active": false}, nil, http.StatusOK
	}
	if user.ExpiresAt > 0 && time.Now().Unix() > int64(user.ExpiresAt) {
		return map[string]any{"active": false}, nil, http.StatusOK
	}

	response := map[string]any{
		"active":    true,
		"client_id": payload["client_id"],
		"sub":       payload["user_id"],
		"exp":       payload["exp"],
		"iat":       payload["iat"],
	}
	if scopes, ok := payload["scopes"].([]string); ok && len(scopes) > 0 {
		response["scope"] = strings.Join(scopes, " ")
	} else if scopesAny, ok := payload["scopes"].([]any); ok {
		list := make([]string, 0, len(scopesAny))
		for _, v := range scopesAny {
			if s, ok := v.(string); ok {
				list = append(list, s)
			}
		}
		if len(list) > 0 {
			response["scope"] = strings.Join(list, " ")
		}
	}
	if aud, ok := payload["aud"]; ok {
		response["aud"] = aud
	}
	return response, nil, http.StatusOK
}

func (s *OAuthService) HandleRevoke(ctx context.Context, req oauthtypes.TokenRevocationRequest, authHeader string, clientService *OAuthClientService) (*oauthtypes.OAuthErrorResponse, int) {
	if req.Token == "" {
		return &oauthtypes.OAuthErrorResponse{Error: "invalid_request", ErrorDescription: "token is required"}, http.StatusBadRequest
	}
	if clientService == nil {
		clientService = NewOAuthClientService(s.db, s)
	}

	clientID, oauthErr, statusCode := s.authenticateClient(ctx, req.ClientID, req.ClientSecret, authHeader, clientService)
	if oauthErr != nil {
		return oauthErr, statusCode
	}
	if strings.TrimSpace(clientID) == "" {
		return &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "client authentication is required"}, http.StatusUnauthorized
	}
	client, err := clientService.GetClient(ctx, clientID)
	if err != nil {
		return &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if client == nil || client.TokenEndpointAuthMethod == "none" {
		return &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "client authentication is required"}, http.StatusUnauthorized
	}

	var token database.OAuthToken
	query := s.db.WithContext(ctx)
	if req.TokenTypeHint == "refresh_token" || req.TokenTypeHint == "" {
		hashedRefreshToken := hashRefreshToken(req.Token)
		if err := query.Where("refresh_token = ? OR refresh_token = ?", req.Token, hashedRefreshToken).First(&token).Error; err == nil {
			if clientID == "" || token.ClientID == clientID {
				if err := s.db.WithContext(ctx).Model(&database.OAuthToken{}).Where("token_id = ?", token.TokenID).Update("revoked", true).Error; err != nil {
					return &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
				}
			}
			return nil, http.StatusOK
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
		}
	}
	if req.TokenTypeHint == "access_token" || req.TokenTypeHint == "" {
		if err := query.Where("access_token = ?", hashRefreshToken(req.Token)).First(&token).Error; err == nil {
			if clientID == "" || token.ClientID == clientID {
				if err := s.db.WithContext(ctx).Model(&database.OAuthToken{}).Where("token_id = ?", token.TokenID).Update("revoked", true).Error; err != nil {
					return &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
				}
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
		}
	}
	return nil, http.StatusOK
}

func (s *OAuthService) generateAuthorizationCode() (string, error) {
	return randomHex(32)
}

func (s *OAuthService) validateAuthorizationCode(ctx context.Context, code, clientID, redirectURI string) (*database.OAuthAuthorizationCode, *oauthtypes.OAuthErrorResponse, int) {
	var authCode database.OAuthAuthorizationCode
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&authCode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Invalid authorization code"}, http.StatusBadRequest
		}
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if authCode.Used {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Authorization code has been used"}, http.StatusBadRequest
	}
	if authCode.ExpiresAt.Before(time.Now()) {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Authorization code has expired"}, http.StatusBadRequest
	}
	if authCode.ClientID != clientID {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Authorization code was issued to another client"}, http.StatusBadRequest
	}
	if authCode.RedirectURI != redirectURI {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "redirect_uri mismatch"}, http.StatusBadRequest
	}
	return &authCode, nil, http.StatusOK
}

func (s *OAuthService) generateTokenPair(ctx context.Context, clientID, userID string, scopes []string, resource *string) (*database.OAuthToken, error) {
	accessToken, err := s.generateAccessToken(clientID, userID, scopes, resource)
	if err != nil {
		return nil, err
	}
	rawRefreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}
	hashedRefreshToken := hashRefreshToken(rawRefreshToken)
	now := time.Now()
	accessExp := now.Add(oauthtypes.AccessTokenLifetime * time.Second)
	refreshExp := now.Add(oauthtypes.RefreshTokenLifetime * time.Second)

	scopeJSON, _ := json.Marshal(scopes)
	token := database.OAuthToken{
		TokenID:               generateCUIDLikeID(),
		AccessToken:           hashRefreshToken(accessToken),
		RefreshToken:          &hashedRefreshToken,
		ClientID:              clientID,
		UserID:                userID,
		Scopes:                scopeJSON,
		Resource:              resource,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: &refreshExp,
		CreatedAt:             now,
		UpdatedAt:             now,
		Revoked:               false,
	}
	if err := s.db.WithContext(ctx).Create(&token).Error; err != nil {
		return nil, err
	}
	token.AccessToken = accessToken
	token.RefreshToken = &rawRefreshToken
	return &token, nil
}

func (s *OAuthService) ValidateRedirectURI(redirectURI string, allowed []string) bool {
	for _, u := range allowed {
		if u == redirectURI {
			return true
		}
	}
	return false
}

func (s *OAuthService) validatePKCE(codeVerifier, codeChallenge, method string) bool {
	if method == "" {
		method = "S256"
	}
	switch method {
	case "plain":
		return codeVerifier == codeChallenge
	case "S256":
		h := sha256.Sum256([]byte(codeVerifier))
		encoded := base64.RawURLEncoding.EncodeToString(h[:])
		return encoded == codeChallenge
	default:
		return false
	}
}

func (s *OAuthService) ParseScope(scope string) []string {
	if scope == "" {
		return []string{}
	}
	parts := strings.Fields(scope)
	return append([]string{}, parts...)
}

func (s *OAuthService) isScopeSubset(requested, original []string) bool {
	set := map[string]bool{}
	for _, sc := range original {
		set[sc] = true
	}
	for _, sc := range requested {
		if !set[sc] {
			return false
		}
	}
	return true
}

func (s *OAuthService) generateAccessToken(clientID, userID string, scopes []string, resource *string) (string, error) {
	secret, err := s.jwtSecretOrError()
	if err != nil {
		return "", err
	}

	now := time.Now().Unix()
	payload := jwt.MapClaims{
		"type":      "access_token",
		"client_id": clientID,
		"user_id":   userID,
		"scopes":    scopes,
		"iat":       now,
		"exp":       now + int64(oauthtypes.AccessTokenLifetime),
	}
	if resource != nil && *resource != "" {
		payload["aud"] = *resource
	} else if pub := strings.TrimSpace(os.Getenv("KIMBAP_PUBLIC_BASE_URL")); pub != "" {
		payload["aud"] = strings.TrimRight(pub, "/")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}
	return signed, nil
}

type verifyTokenResult struct {
	Valid   bool
	Payload jwt.MapClaims
	Error   string
}

func (s *OAuthService) verifyAccessToken(token string) verifyTokenResult {
	secret := strings.TrimSpace(s.jwtSecret)
	if secret == "" {
		return verifyTokenResult{Valid: false, Error: "JWT_SECRET environment variable is required"}
	}
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return verifyTokenResult{Valid: false, Error: err.Error()}
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return verifyTokenResult{Valid: false, Error: "invalid token"}
	}
	return verifyTokenResult{Valid: true, Payload: claims}
}

func (s *OAuthService) generateRefreshToken() (string, error) {
	return randomHex(64)
}

func (s *OAuthService) generateClientSecret() (string, error) {
	return randomHex(32)
}

func (s *OAuthService) generateClientID() (string, error) {
	return randomHex(16)
}

type basicAuthCredentials struct {
	ClientID     string
	ClientSecret string
}

func (s *OAuthService) parseBasicAuth(header string) *basicAuthCredentials {
	if !strings.HasPrefix(header, "Basic ") {
		return nil
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Basic "))
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil
	}
	return &basicAuthCredentials{ClientID: parts[0], ClientSecret: parts[1]}
}

func (s *OAuthService) buildErrorRedirectURL(redirectURI, errCode, description, state string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("error", errCode)
	if description != "" {
		q.Set("error_description", description)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *OAuthService) buildSuccessRedirectURL(redirectURI, code, state string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *OAuthService) GenerateAuthorizationServerMetadata(issuer string) oauthtypes.AuthorizationServerMetadata {
	return oauthtypes.AuthorizationServerMetadata{
		Issuer:                                 issuer,
		AuthorizationEndpoint:                  issuer + "/authorize",
		TokenEndpoint:                          issuer + "/token",
		RegistrationEndpoint:                   issuer + "/register",
		RevocationEndpoint:                     issuer + "/revoke",
		IntrospectionEndpoint:                  issuer + "/introspect",
		ScopesSupported:                        []string{"mcp:tools", "mcp:resources", "mcp:prompts"},
		ResponseTypesSupported:                 []string{"code"},
		GrantTypesSupported:                    []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported:      []string{"client_secret_basic", "client_secret_post", "none"},
		RevocationEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},
		TokenEndpointAuthSigningAlgsSupported:  []string{"HS256"},
		CodeChallengeMethodsSupported:          []string{"S256", "plain"},
		ClientIDMetadataDocumentSupported:      true,
		ServiceDocumentation:                   issuer + "/docs/oauth",
	}
}

func (s *OAuthService) GenerateProtectedResourceMetadata(resourceURL, authServerURL string) oauthtypes.ProtectedResourceMetadata {
	return oauthtypes.ProtectedResourceMetadata{
		Resource:                        resourceURL,
		AuthorizationServers:            []string{authServerURL},
		BearerMethodsSupported:          []string{"header"},
		ResourceDocumentation:           authServerURL + "/docs/oauth",
		ResourceSigningAlgValuesSupport: []string{"HS256"},
		ScopesSupported:                 []string{"mcp:tools", "mcp:resources", "mcp:prompts"},
	}
}

func (s *OAuthService) validateClient(ctx context.Context, clientID, clientSecret string, clientService *OAuthClientService) (*oauthtypes.OAuthErrorResponse, int) {
	if clientID == "" {
		return &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "client_id is required"}, http.StatusUnauthorized
	}
	client, err := clientService.GetClient(ctx, clientID)
	if err != nil {
		return &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if client == nil {
		return &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "Client not found"}, http.StatusUnauthorized
	}
	if client.TokenEndpointAuthMethod != "none" {
		if clientSecret == "" {
			return &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "client_secret is required for confidential clients"}, http.StatusUnauthorized
		}
		valid, err := clientService.VerifyClientCredentials(ctx, clientID, clientSecret)
		if err != nil {
			return &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
		}
		if !valid {
			return &oauthtypes.OAuthErrorResponse{Error: "invalid_client", ErrorDescription: "Invalid client credentials"}, http.StatusUnauthorized
		}
	}
	return nil, 0
}

func (s *OAuthService) handleAuthorizationCodeGrant(ctx context.Context, req oauthtypes.TokenRequest, clientID, clientSecret string, clientService *OAuthClientService) (*oauthtypes.TokenResponse, *oauthtypes.OAuthErrorResponse, int) {
	if req.Code == "" || req.RedirectURI == "" {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_request", ErrorDescription: "code and redirect_uri are required"}, http.StatusBadRequest
	}
	if oauthErr, status := s.validateClient(ctx, clientID, clientSecret, clientService); oauthErr != nil {
		return nil, oauthErr, status
	}

	authCode, oauthErr, status := s.validateAuthorizationCode(ctx, req.Code, clientID, req.RedirectURI)
	if oauthErr != nil {
		return nil, oauthErr, status
	}
	if authCode.CodeChallenge != nil && *authCode.CodeChallenge != "" {
		if req.CodeVerifier == "" {
			return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "code_verifier is required"}, http.StatusBadRequest
		}
		method := "S256"
		if authCode.ChallengeMethod != nil && *authCode.ChallengeMethod != "" {
			method = *authCode.ChallengeMethod
		}
		if !s.validatePKCE(req.CodeVerifier, *authCode.CodeChallenge, method) {
			return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Invalid code_verifier"}, http.StatusBadRequest
		}
	}

	var scopes []string
	if err := json.Unmarshal(authCode.Scopes, &scopes); err != nil {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}

	result := s.db.WithContext(ctx).Model(&database.OAuthAuthorizationCode{}).Where("code = ? AND used = ?", req.Code, false).Update("used", true)
	if result.Error != nil {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if result.RowsAffected != 1 {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Authorization code has been used"}, http.StatusBadRequest
	}

	token, err := s.generateTokenPair(ctx, authCode.ClientID, authCode.UserID, scopes, authCode.Resource)
	if err != nil {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	response := &oauthtypes.TokenResponse{
		AccessToken: token.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   oauthtypes.AccessTokenLifetime,
		Scope:       strings.Join(scopes, " "),
	}
	if token.RefreshToken != nil {
		response.RefreshToken = *token.RefreshToken
	}
	if token.Resource != nil {
		response.Resource = *token.Resource
	}
	return response, nil, http.StatusOK
}

func (s *OAuthService) handleRefreshTokenGrant(ctx context.Context, req oauthtypes.TokenRequest, clientID, clientSecret string, clientService *OAuthClientService) (*oauthtypes.TokenResponse, *oauthtypes.OAuthErrorResponse, int) {
	if req.RefreshToken == "" {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_request", ErrorDescription: "refresh_token is required"}, http.StatusBadRequest
	}
	if oauthErr, status := s.validateClient(ctx, clientID, clientSecret, clientService); oauthErr != nil {
		return nil, oauthErr, status
	}
	if !isRawRefreshTokenFormat(req.RefreshToken) {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Invalid refresh token"}, http.StatusBadRequest
	}

	var token database.OAuthToken
	hashedRefreshToken := hashRefreshToken(req.RefreshToken)
	if err := s.db.WithContext(ctx).Where("refresh_token = ?", hashedRefreshToken).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Invalid refresh token"}, http.StatusBadRequest
		}
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	if token.Revoked {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Token has been revoked"}, http.StatusBadRequest
	}
	if token.RefreshTokenExpiresAt != nil && token.RefreshTokenExpiresAt.Before(time.Now()) {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Refresh token has expired"}, http.StatusBadRequest
	}
	if token.ClientID != clientID {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_grant", ErrorDescription: "Refresh token was issued to another client"}, http.StatusBadRequest
	}

	var originalScopes []string
	if err := json.Unmarshal(token.Scopes, &originalScopes); err != nil {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	newScopes := originalScopes
	if req.Scope != "" {
		reqScopes := s.ParseScope(req.Scope)
		if !s.isScopeSubset(reqScopes, originalScopes) {
			return nil, &oauthtypes.OAuthErrorResponse{Error: "invalid_scope", ErrorDescription: "Requested scope exceeds original grant"}, http.StatusBadRequest
		}
		newScopes = reqScopes
	}

	access, err := s.generateAccessToken(token.ClientID, token.UserID, newScopes, token.Resource)
	if err != nil {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}
	accessExp := time.Now().Add(oauthtypes.AccessTokenLifetime * time.Second)
	scopeJSON, _ := json.Marshal(newScopes)
	if err := s.db.WithContext(ctx).Model(&database.OAuthToken{}).Where("token_id = ?", token.TokenID).Updates(map[string]any{
		"access_token":            hashRefreshToken(access),
		"access_token_expires_at": accessExp,
		"scopes":                  scopeJSON,
	}).Error; err != nil {
		return nil, &oauthtypes.OAuthErrorResponse{Error: "server_error", ErrorDescription: "Internal server error"}, http.StatusInternalServerError
	}

	response := &oauthtypes.TokenResponse{
		AccessToken:  hashRefreshToken(access),
		TokenType:    "Bearer",
		ExpiresIn:    oauthtypes.AccessTokenLifetime,
		RefreshToken: req.RefreshToken,
		Scope:        strings.Join(newScopes, " "),
	}
	response.AccessToken = access
	if token.Resource != nil {
		response.Resource = *token.Resource
	}
	return response, nil, http.StatusOK
}

func randomHex(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func isRawRefreshTokenFormat(token string) bool {
	if len(token) != 128 {
		return false
	}
	for _, c := range token {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func generateCUIDLikeID() string {
	const totalLength = 25
	timestampPart := strings.ToLower(strconv.FormatInt(time.Now().UnixMilli(), 36))
	maxTimestampLen := totalLength - 1
	if len(timestampPart) > maxTimestampLen {
		timestampPart = timestampPart[len(timestampPart)-maxTimestampLen:]
	}
	randomPart := randomBase36String(maxTimestampLen - len(timestampPart))
	return "c" + timestampPart + randomPart
}

func randomBase36String(length int) string {
	if length <= 0 {
		return ""
	}
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		fallback := strings.ToLower(strconv.FormatInt(time.Now().UnixNano(), 36))
		if len(fallback) >= length {
			return fallback[:length]
		}
		return fallback + strings.Repeat("0", length-len(fallback))
	}
	out := make([]byte, length)
	for i := range randomBytes {
		out[i] = alphabet[int(randomBytes[i])%len(alphabet)]
	}
	return string(out)
}

func (s *OAuthService) jwtSecretOrError() (string, error) {
	secret := strings.TrimSpace(s.jwtSecret)
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable is required")
	}
	return secret, nil
}

package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	oauthtypes "github.com/dunialabs/kimbap-core/internal/oauth/types"
	"gorm.io/gorm"
)

type OAuthClientService struct {
	db    *gorm.DB
	oauth *OAuthService
}

func NewOAuthClientService(db *gorm.DB, oauth *OAuthService) *OAuthClientService {
	if db == nil {
		db = database.DB
	}
	if oauth == nil {
		oauth = NewOAuthService(db)
	}
	return &OAuthClientService{db: db, oauth: oauth}
}

func (s *OAuthClientService) RegisterClient(ctx context.Context, metadata oauthtypes.OAuthClientMetadata, userID *string) (*oauthtypes.OAuthClientInformation, error) {
	if strings.HasPrefix(metadata.ClientID, "https://") {
		fetcher := NewClientMetadataFetcher()
		fetched, err := fetcher.FetchClientMetadata(metadata.ClientID)
		if err != nil {
			return nil, err
		}
		fetched.ClientID = metadata.ClientID
		return s.registerURLClient(ctx, *fetched, userID)
	}
	authMethod := metadata.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "client_secret_post"
	}
	grantTypes := metadata.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	responseTypes := metadata.ResponseTypes
	if len(responseTypes) == 0 && slices.Contains(grantTypes, "authorization_code") {
		responseTypes = []string{"code"}
	}
	requiresRedirectURIs := slices.Contains(grantTypes, "authorization_code") || slices.Contains(responseTypes, "code")
	if requiresRedirectURIs && len(metadata.RedirectURIs) == 0 {
		return nil, fmt.Errorf("invalid_client_metadata: redirect_uris is required and must be a non-empty array")
	}
	for _, uri := range metadata.RedirectURIs {
		parsed, err := url.Parse(uri)
		if err != nil || parsed == nil || strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
			return nil, fmt.Errorf("invalid_redirect_uri: Invalid redirect_uri: %s", uri)
		}
	}

	validGrantTypes := map[string]bool{"authorization_code": true, "refresh_token": true, "client_credentials": true}
	for _, gt := range grantTypes {
		if !validGrantTypes[gt] {
			return nil, fmt.Errorf("unsupported grant_type: %s", gt)
		}
	}

	validResponseTypes := map[string]bool{"code": true}
	for _, rt := range responseTypes {
		if !validResponseTypes[rt] {
			return nil, fmt.Errorf("unsupported response_type: %s", rt)
		}
	}

	validAuthMethods := map[string]bool{"client_secret_basic": true, "client_secret_post": true, "none": true}
	if !validAuthMethods[authMethod] {
		return nil, fmt.Errorf("unsupported token_endpoint_auth_method: %s", authMethod)
	}
	if authMethod == "none" && slices.Contains(grantTypes, "client_credentials") {
		return nil, fmt.Errorf("invalid_client_metadata: token_endpoint_auth_method 'none' cannot be used with client_credentials grant")
	}

	scopes := s.oauth.ParseScope(metadata.Scope)

	// Only perform duplicate check when client_name is explicitly provided
	if strings.TrimSpace(metadata.ClientName) != "" {
		dup, err := s.findDuplicate(ctx, metadata.ClientName, metadata.RedirectURIs, authMethod, grantTypes, responseTypes)
		if err != nil {
			return nil, err
		}
		if dup != nil {
			return toClientInfo(*dup, false)
		}
	}

	clientID, err := s.oauth.generateClientID()
	if err != nil {
		return nil, err
	}
	var clientSecret *string
	var rawSecret string
	if authMethod != "none" {
		secret, err := s.oauth.generateClientSecret()
		if err != nil {
			return nil, err
		}
		rawSecret = secret
		hashed := hashClientSecret(secret)
		clientSecret = &hashed
	}
	redirectJSON, _ := json.Marshal(metadata.RedirectURIs)
	grantJSON, _ := json.Marshal(grantTypes)
	responseJSON, _ := json.Marshal(responseTypes)
	scopeJSON, _ := json.Marshal(scopes)
	clientName := metadata.ClientName
	if strings.TrimSpace(clientName) == "" {
		clientName = "Client " + clientID
	}

	client := database.OAuthClient{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		TokenEndpointAuthMethod: authMethod,
		Name:                    clientName,
		RedirectUris:            redirectJSON,
		GrantTypes:              grantJSON,
		ResponseTypes:           responseJSON,
		Scopes:                  scopeJSON,
		UserID:                  userID,
		Trusted:                 false,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&client).Error; err != nil {
		return nil, err
	}
	info, err := toClientInfo(client, false)
	if err != nil {
		return nil, err
	}
	if rawSecret != "" {
		info.ClientSecret = rawSecret
	}
	return info, nil
}

func (s *OAuthClientService) registerURLClient(ctx context.Context, metadata oauthtypes.OAuthClientMetadata, userID *string) (*oauthtypes.OAuthClientInformation, error) {
	var existing database.OAuthClient
	err := s.db.WithContext(ctx).Where("client_id = ?", metadata.ClientID).First(&existing).Error
	if err == nil {
		return toClientInfo(existing, false)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	authMethod := metadata.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "none"
	}
	if authMethod != "none" {
		return nil, fmt.Errorf("invalid_client_metadata: URL-based clients must use token_endpoint_auth_method 'none'")
	}
	grantTypes := metadata.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	if authMethod == "none" && slices.Contains(grantTypes, "client_credentials") {
		return nil, fmt.Errorf("invalid_client_metadata: token_endpoint_auth_method 'none' cannot be used with client_credentials grant")
	}
	responseTypes := metadata.ResponseTypes
	if len(responseTypes) == 0 && slices.Contains(grantTypes, "authorization_code") {
		responseTypes = []string{"code"}
	}
	requiresRedirectURIs := slices.Contains(grantTypes, "authorization_code") || slices.Contains(responseTypes, "code")
	if requiresRedirectURIs && len(metadata.RedirectURIs) == 0 {
		return nil, fmt.Errorf("invalid_client_metadata: redirect_uris is required and must be a non-empty array")
	}
	for _, uri := range metadata.RedirectURIs {
		parsed, err := url.Parse(uri)
		if err != nil || parsed == nil || strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
			return nil, fmt.Errorf("invalid_redirect_uri: Invalid redirect_uri: %s", uri)
		}
	}
	scopes := s.oauth.ParseScope(metadata.Scope)

	redirectJSON, _ := json.Marshal(metadata.RedirectURIs)
	grantJSON, _ := json.Marshal(grantTypes)
	responseJSON, _ := json.Marshal(responseTypes)
	scopeJSON, _ := json.Marshal(scopes)
	urlClientName := metadata.ClientName
	if strings.TrimSpace(urlClientName) == "" {
		urlClientName = "URL Client " + metadata.ClientID
	}

	client := database.OAuthClient{
		ClientID:                metadata.ClientID,
		ClientSecret:            nil,
		TokenEndpointAuthMethod: authMethod,
		Name:                    urlClientName,
		RedirectUris:            redirectJSON,
		GrantTypes:              grantJSON,
		ResponseTypes:           responseJSON,
		Scopes:                  scopeJSON,
		UserID:                  userID,
		Trusted:                 false,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&client).Error; err != nil {
		return nil, err
	}
	return toClientInfo(client, true)
}

func (s *OAuthClientService) GetClient(ctx context.Context, clientID string) (*oauthtypes.OAuthClientInformation, error) {
	var client database.OAuthClient
	if err := s.db.WithContext(ctx).Where("client_id = ?", clientID).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toClientInfo(client, true)
}

func (s *OAuthClientService) UpdateClient(ctx context.Context, clientID string, updates oauthtypes.OAuthClientMetadata) (*oauthtypes.OAuthClientInformation, error) {
	data := map[string]any{}
	if updates.ClientName != "" {
		data["name"] = updates.ClientName
	}
	if len(updates.RedirectURIs) > 0 {
		for _, uri := range updates.RedirectURIs {
			parsed, err := url.Parse(uri)
			if err != nil || parsed == nil || strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
				return nil, fmt.Errorf("invalid_redirect_uri: Invalid redirect_uri: %s", uri)
			}
		}
		b, _ := json.Marshal(updates.RedirectURIs)
		data["redirect_uris"] = b
	}
	if updates.Scope != "" {
		b, _ := json.Marshal(s.oauth.ParseScope(updates.Scope))
		data["scopes"] = b
	}
	if len(updates.GrantTypes) > 0 {
		if slices.Contains(updates.GrantTypes, "client_credentials") {
			var existing database.OAuthClient
			err := s.db.WithContext(ctx).Select("token_endpoint_auth_method").Where("client_id = ?", clientID).First(&existing).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}
			if existing.TokenEndpointAuthMethod == "none" {
				return nil, fmt.Errorf("invalid_client_metadata: token_endpoint_auth_method 'none' cannot be used with client_credentials grant")
			}
		}
		b, _ := json.Marshal(updates.GrantTypes)
		data["grant_types"] = b
	}
	if len(updates.ResponseTypes) > 0 {
		b, _ := json.Marshal(updates.ResponseTypes)
		data["response_types"] = b
	}
	if len(data) == 0 {
		return s.GetClient(ctx, clientID)
	}

	result := s.db.WithContext(ctx).Model(&database.OAuthClient{}).Where("client_id = ?", clientID).Updates(data)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return s.GetClient(ctx, clientID)
}

func (s *OAuthClientService) DeleteClient(ctx context.Context, clientID string) (bool, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("client_id = ?", clientID).Delete(&database.OAuthAuthorizationCode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("client_id = ?", clientID).Delete(&database.OAuthToken{}).Error; err != nil {
			return err
		}
		result := tx.Where("client_id = ?", clientID).Delete(&database.OAuthClient{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *OAuthClientService) ListClients(ctx context.Context) ([]oauthtypes.OAuthClientInformation, error) {
	var clients []database.OAuthClient
	if err := s.db.WithContext(ctx).Order("created_at DESC").Find(&clients).Error; err != nil {
		return nil, err
	}
	out := make([]oauthtypes.OAuthClientInformation, 0, len(clients))
	for _, c := range clients {
		info, err := toClientInfo(c, false)
		if err != nil {
			return nil, err
		}
		out = append(out, *info)
	}
	return out, nil
}

func (s *OAuthClientService) VerifyClientCredentials(ctx context.Context, clientID, clientSecret string) (bool, error) {
	info, err := s.GetClient(ctx, clientID)
	if err != nil || info == nil {
		return false, err
	}
	if info.TokenEndpointAuthMethod == "none" {
		return true, nil
	}
	if strings.TrimSpace(info.ClientSecret) == "" {
		return false, nil
	}
	hashedInput := hashClientSecret(clientSecret)
	return subtle.ConstantTimeCompare([]byte(info.ClientSecret), []byte(hashedInput)) == 1, nil
}

func hashClientSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

func (s *OAuthClientService) findDuplicate(ctx context.Context, name string, redirectURIs []string, authMethod string, grantTypes []string, responseTypes []string) (*database.OAuthClient, error) {
	var clients []database.OAuthClient
	if err := s.db.WithContext(ctx).Where("name = ? AND token_endpoint_auth_method = ?", name, authMethod).Find(&clients).Error; err != nil {
		return nil, err
	}
	for _, c := range clients {
		var redirects []string
		var grants []string
		var responses []string
		if err := json.Unmarshal(c.RedirectUris, &redirects); err != nil {
			return nil, fmt.Errorf("invalid oauth_clients.redirect_uris JSON for client %s: %w", c.ClientID, err)
		}
		if err := json.Unmarshal(c.GrantTypes, &grants); err != nil {
			return nil, fmt.Errorf("invalid oauth_clients.grant_types JSON for client %s: %w", c.ClientID, err)
		}
		if err := json.Unmarshal(c.ResponseTypes, &responses); err != nil {
			return nil, fmt.Errorf("invalid oauth_clients.response_types JSON for client %s: %w", c.ClientID, err)
		}
		if sameStringSlice(redirects, redirectURIs) && sameStringSlice(grants, grantTypes) && sameStringSlice(responses, responseTypes) {
			dup := c
			return &dup, nil
		}
	}
	return nil, nil
}

func sameStringSlice(a, b []string) bool {
	return slices.Equal(a, b)
}

func toClientInfo(c database.OAuthClient, includeSecret bool) (*oauthtypes.OAuthClientInformation, error) {
	var redirectURIs []string
	var grantTypes []string
	var scopes []string
	if err := json.Unmarshal(c.RedirectUris, &redirectURIs); err != nil {
		return nil, fmt.Errorf("invalid oauth_clients.redirect_uris JSON for client %s: %w", c.ClientID, err)
	}
	if err := json.Unmarshal(c.GrantTypes, &grantTypes); err != nil {
		return nil, fmt.Errorf("invalid oauth_clients.grant_types JSON for client %s: %w", c.ClientID, err)
	}
	if err := json.Unmarshal(c.Scopes, &scopes); err != nil {
		return nil, fmt.Errorf("invalid oauth_clients.scopes JSON for client %s: %w", c.ClientID, err)
	}
	info := &oauthtypes.OAuthClientInformation{
		ClientID:                c.ClientID,
		ClientName:              c.Name,
		RedirectURIs:            redirectURIs,
		GrantTypes:              grantTypes,
		Scopes:                  scopes,
		TokenEndpointAuthMethod: c.TokenEndpointAuthMethod,
		Trusted:                 c.Trusted,
		CreatedAt:               c.CreatedAt,
		UpdatedAt:               c.UpdatedAt,
	}
	if includeSecret && c.ClientSecret != nil {
		info.ClientSecret = *c.ClientSecret
	}
	return info, nil
}

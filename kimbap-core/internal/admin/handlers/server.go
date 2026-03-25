package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcpoauth "github.com/dunialabs/kimbap-core/internal/mcp/oauth"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/dunialabs/kimbap-core/internal/security"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/dunialabs/kimbap-core/internal/utils"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type ServerHandler struct {
	db             *gorm.DB
	serverManager  serverRuntimeManager
	sessionStore   *core.SessionStore
	socketNotifier core.SocketNotifier
}

type serverListResponse struct {
	ServerID             string  `json:"serverId"`
	ServerName           string  `json:"serverName"`
	Enabled              bool    `json:"enabled"`
	LaunchConfig         string  `json:"launchConfig"`
	Capabilities         string  `json:"capabilities"`
	CreatedAt            int     `json:"createdAt"`
	UpdatedAt            int     `json:"updatedAt"`
	AllowUserInput       bool    `json:"allowUserInput"`
	ConfigTemplate       *string `json:"configTemplate"`
	ProxyID              int     `json:"proxyId"`
	ToolTmplID           *string `json:"toolTmplId"`
	AuthType             int     `json:"authType"`
	Category             int     `json:"category"`
	LazyStartEnabled     bool    `json:"lazyStartEnabled"`
	PublicAccess         bool    `json:"publicAccess"`
	UseKimbapOauthConfig bool    `json:"useKimbapOauthConfig"`
	TransportType        *string `json:"transportType"`
	AnonymousAccess      bool    `json:"anonymousAccess"`
	AnonymousRateLimit   int     `json:"anonymousRateLimit"`
}

type serverRuntimeManager interface {
	AddServer(ctx context.Context, server database.Server, token string) (*core.ServerContext, error)
	RemoveServer(ctx context.Context, serverID string) (*core.ServerContext, error)
	ReconnectServer(ctx context.Context, server database.Server, token string) (*core.ServerContext, error)
	UpdateServerCapabilitiesConfig(ctx context.Context, serverID string, raw string) (bool, bool, bool, error)
	ConnectAllServers(ctx context.Context, token string) ([]core.ServerConnectResult, []core.ServerConnectResult, error)
	GetServerContext(serverID, userID string) *core.ServerContext
	GetTemporaryServers() []*core.ServerContext
	UpdateTemporaryServersByTemplate(server database.Server)
	CloseAllTemporaryServersByTemplate(serverID string)
}

func NewServerHandler(db *gorm.DB, serverManager serverRuntimeManager, sessionStore *core.SessionStore, socketNotifier core.SocketNotifier) *ServerHandler {
	if db == nil {
		db = database.DB
	}
	if serverManager == nil {
		serverManager = core.ServerManagerInstance()
	}
	if sessionStore == nil {
		sessionStore = core.SessionStoreInstance()
	}
	if socketNotifier == nil {
		socketNotifier = core.NewNoopSocketNotifier()
	}
	return &ServerHandler{db: db, serverManager: serverManager, sessionStore: sessionStore, socketNotifier: socketNotifier}
}

func (h *ServerHandler) StartServer(data map[string]any, ownerToken string) (any, error) {
	id := toString(data["targetId"])
	if id == "" {
		return nil, &types.AdminError{Message: "missing targetId", Code: types.AdminErrorCodeInvalidRequest}
	}
	server, err := h.findServer(id)
	if err != nil {
		return nil, err
	}
	if !server.AllowUserInput && strings.TrimSpace(server.LaunchConfig) == "" {
		return nil, &types.AdminError{
			Message: fmt.Sprintf("Cannot start template server %s. Users must configure it first through client.", id),
			Code:    types.AdminErrorCodeInvalidRequest,
		}
	}
	if err := h.db.Model(&database.Server{}).Where("server_id = ?", id).Update("enabled", true).Error; err != nil {
		return nil, err
	}
	server.Enabled = true

	if server.AllowUserInput {
		h.serverManager.UpdateTemporaryServersByTemplate(server)
		affectedSessions := h.sessionStore.GetSessionsUsingServer(id)
		h.notifyCapabilityChangedSessionsByFlags(affectedSessions, true, true, true)
		// Notify all affected sessions — Go sends notifications unconditionally on operations
		// to ensure clients stay in sync. TS is similar but sometimes skips when no changes detected.
		h.socketNotifier.NotifyUserPermissionChangedByServer(id)
		return nil, nil
	}
	if _, err := h.serverManager.AddServer(context.Background(), server, ownerToken); err != nil {
		if dbErr := h.db.Model(&database.Server{}).Where("server_id = ?", id).Update("enabled", false).Error; dbErr != nil {
			log.Error().Err(dbErr).Str("serverID", id).Msg("failed to disable server after AddServer failure")
		}
		return nil, err
	}
	affectedSessions := h.sessionStore.GetSessionsUsingServer(id)
	h.notifyCapabilityChangedSessionsByFlags(affectedSessions, true, true, true)
	h.socketNotifier.NotifyUserPermissionChangedByServer(id)
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminServerEdit,
		RequestParams: toJSONString(map[string]any{
			"targetId": id,
		}, "{}"),
	})
	return nil, nil
}

func (h *ServerHandler) StopServer(data map[string]any) (any, error) {
	id := toString(data["targetId"])
	if id == "" {
		return nil, &types.AdminError{Message: "missing targetId", Code: types.AdminErrorCodeInvalidRequest}
	}
	server, err := h.findServer(id)
	if err != nil {
		return nil, err
	}
	affectedSessions := h.sessionStore.GetSessionsUsingServer(id)
	if server.AllowUserInput {
		h.serverManager.CloseAllTemporaryServersByTemplate(id)
	} else {
		if _, err := h.serverManager.RemoveServer(context.Background(), id); err != nil {
			return nil, err
		}
	}
	if err := h.db.Model(&database.Server{}).Where("server_id = ?", id).Update("enabled", false).Error; err != nil {
		return nil, err
	}
	h.notifyCapabilityChangedSessionsByFlags(affectedSessions, true, true, true)
	h.socketNotifier.NotifyUserPermissionChangedByServer(id)
	return nil, nil
}

func (h *ServerHandler) UpdateServerCapabilities(data map[string]any) (any, error) {
	id := toString(data["targetId"])
	if id == "" {
		return nil, &types.AdminError{Message: "missing targetId", Code: types.AdminErrorCodeInvalidRequest}
	}
	server, err := h.findServer(id)
	if err != nil {
		return nil, err
	}
	mergedCapabilities, mergeErr := mergeCapabilitiesJSON(server.Capabilities, toJSONString(data["capabilities"], "{}"))
	if mergeErr != nil {
		return nil, &types.AdminError{Message: "invalid capabilities", Code: types.AdminErrorCodeInvalidRequest}
	}
	if mergedCapabilities == server.Capabilities {
		return nil, nil
	}
	if err := h.db.Model(&database.Server{}).Where("server_id = ?", id).Update("capabilities", mergedCapabilities).Error; err != nil {
		return nil, err
	}
	server.Capabilities = mergedCapabilities
	h.serverManager.UpdateTemporaryServersByTemplate(server)
	toolsChanged, resourcesChanged, promptsChanged, err := h.serverManager.UpdateServerCapabilitiesConfig(context.Background(), id, mergedCapabilities)
	if err != nil {
		return nil, err
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminServerEdit,
		RequestParams: toJSONString(map[string]any{
			"targetId":     id,
			"capabilities": data["capabilities"],
		}, "{}"),
	})
	affectedSessions := h.sessionStore.GetSessionsUsingServer(id)
	h.notifyCapabilityChangedSessionsByFlags(affectedSessions, toolsChanged, resourcesChanged, promptsChanged)
	h.socketNotifier.NotifyUserPermissionChangedByServer(id)
	return nil, nil
}

func (h *ServerHandler) UpdateServerLaunchCmd(data map[string]any, ownerToken string) (any, error) {
	id := toString(data["targetId"])
	launch := toJSONString(data["launchConfig"], "")
	if id == "" || isEmptyConfigString(launch) {
		return nil, &types.AdminError{Message: "missing targetId or launchConfig", Code: types.AdminErrorCodeInvalidRequest}
	}
	server, err := h.findServer(id)
	if err != nil {
		return nil, err
	}
	if server.AllowUserInput && !(server.Category == types.ServerCategoryRestAPI || server.Category == types.ServerCategoryCustomRemote || server.Category == types.ServerCategoryCustomStdio) {
		return nil, &types.AdminError{Message: fmt.Sprintf("Server %s is a template server and cannot be updated", id), Code: types.AdminErrorCodeInvalidRequest}
	}
	affectedSessions := h.sessionStore.GetSessionsUsingServer(id)
	if err := h.db.Model(&database.Server{}).Where("server_id = ?", id).Update("launch_config", launch).Error; err != nil {
		return nil, err
	}
	server.LaunchConfig = launch

	if server.Enabled && !server.AllowUserInput {
		if _, err := h.serverManager.ReconnectServer(context.Background(), server, ownerToken); err != nil {
			return nil, err
		}
		h.notifyCapabilityChangedSessionsByFlags(affectedSessions, true, true, true)
		h.socketNotifier.NotifyUserPermissionChangedByServer(id)
	}

	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminServerEdit,
		RequestParams: toJSONString(map[string]any{
			"targetId": id,
		}, "{}"),
	})
	return nil, nil
}

func (h *ServerHandler) ConnectAllServers(ownerToken string) (any, error) {
	successServers, failedServers, err := h.serverManager.ConnectAllServers(context.Background(), ownerToken)
	if err != nil {
		return nil, err
	}
	for _, server := range successServers {
		serverContext := h.serverManager.GetServerContext(server.ServerID, "")
		if serverContext == nil {
			continue
		}
		toolsChanged, resourcesChanged, promptsChanged := serverContext.CapabilityChanged()
		affectedSessions := h.sessionStore.GetSessionsUsingServer(server.ServerID)
		h.notifyCapabilityChangedSessionsByFlags(affectedSessions, toolsChanged, resourcesChanged, promptsChanged)
		h.socketNotifier.NotifyUserPermissionChangedByServer(server.ServerID)
	}

	// Return summary objects {serverId, serverName, proxyId}, not full DB records
	return map[string]any{"successServers": successServers, "failedServers": failedServers}, nil
}

func (h *ServerHandler) CreateServer(data map[string]any, ownerToken string) (any, error) {
	serverID := toString(data["serverId"])
	if serverID == "" {
		return nil, &types.AdminError{Message: "missing serverId", Code: types.AdminErrorCodeInvalidRequest}
	}
	var existing database.Server
	err := h.db.Where("server_id = ?", serverID).First(&existing).Error
	if err == nil {
		return nil, &types.AdminError{Message: "server already exists", Code: types.AdminErrorCodeServerAlreadyExists}
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	launchConfig, ok := data["launchConfig"].(string)
	if !ok || isEmptyConfigString(launchConfig) {
		return nil, &types.AdminError{Message: "launchConfig must be a string and cannot be empty", Code: types.AdminErrorCodeInvalidRequest}
	}

	authTypeValue := types.ServerAuthTypeApiKey
	if rawAuthType, exists := data["authType"]; exists {
		authTypeParsed, valid := asIntMaybe(rawAuthType)
		if !valid || !isValidServerAuthType(authTypeParsed) {
			return nil, &types.AdminError{Message: "Invalid authType", Code: types.AdminErrorCodeInvalidRequest}
		}
		authTypeValue = authTypeParsed
	}

	rawCategory, categoryExists := data["category"]
	if !categoryExists {
		return nil, &types.AdminError{Message: "Invalid category", Code: types.AdminErrorCodeInvalidRequest}
	}
	categoryValue, validCategory := asIntMaybe(rawCategory)
	if !validCategory || !isValidServerCategory(categoryValue) {
		return nil, &types.AdminError{Message: "Invalid category", Code: types.AdminErrorCodeInvalidRequest}
	}

	allowUserInputValue, err := parseBoolLike(data["allowUserInput"], false)
	if err != nil {
		return nil, &types.AdminError{Message: "Invalid allowUserInput", Code: types.AdminErrorCodeInvalidRequest}
	}

	configTemplateStr, configTemplateProvided, err := optionalStringValue(data["configTemplate"])
	if err != nil {
		return nil, &types.AdminError{Message: "Invalid configTemplate", Code: types.AdminErrorCodeInvalidRequest}
	}
	configTemplateInvalid := !configTemplateProvided || isEmptyConfigString(configTemplateStr)
	if (allowUserInputValue || categoryValue == types.ServerCategoryTemplate || categoryValue == types.ServerCategoryRestAPI || categoryValue == types.ServerCategorySkills) && configTemplateInvalid {
		return nil, &types.AdminError{Message: "configTemplate is required for this server", Code: types.AdminErrorCodeInvalidRequest}
	}

	useKimbapOauthConfigValue := true
	launchConfigStr := launchConfig

	if categoryValue == types.ServerCategoryTemplate {
		configTemplateValue := map[string]any{}
		if strings.TrimSpace(configTemplateStr) != "" {
			if err := json.Unmarshal([]byte(configTemplateStr), &configTemplateValue); err != nil {
				return nil, &types.AdminError{Message: "Invalid configTemplate", Code: types.AdminErrorCodeInvalidRequest}
			}
		}

		oauthConfig, _ := configTemplateValue["oAuthConfig"].(map[string]any)
		oauthConfigClientID := strings.TrimSpace(toString(oauthConfig["clientId"]))
		if oauthConfigClientID != "" {
			if strings.TrimSpace(ownerToken) == "" {
				return nil, &types.AdminError{Message: "invalid token", Code: types.AdminErrorCodeInvalidRequest}
			}

			decryptedLaunchConfig, err := security.DecryptDataFromString(launchConfig, ownerToken)
			if err != nil {
				return nil, &types.AdminError{Message: "Failed to decrypt launchConfig", Code: types.AdminErrorCodeInvalidRequest}
			}

			decryptedLaunchConfigValue := map[string]any{}
			if err := json.Unmarshal([]byte(decryptedLaunchConfig), &decryptedLaunchConfigValue); err != nil {
				return nil, &types.AdminError{Message: "Invalid decrypted launchConfig", Code: types.AdminErrorCodeInvalidRequest}
			}

			oauth, _ := decryptedLaunchConfigValue["oauth"].(map[string]any)
			oauthClientID := strings.TrimSpace(toString(oauth["clientId"]))
			if oauthClientID == "" {
				return nil, &types.AdminError{Message: "Missing required field: clientId", Code: types.AdminErrorCodeInvalidRequest}
			}

			provider := utils.OAuthProviderFromAuthType(authTypeValue)
			if provider == "" {
				return nil, &types.AdminError{Message: "Invalid OAuth provider", Code: types.AdminErrorCodeInvalidRequest}
			}

			if !allowUserInputValue {
				oauthCode := strings.TrimSpace(toString(oauth["code"]))
				if oauthCode == "" {
					return nil, &types.AdminError{Message: "Invalid OAuth code", Code: types.AdminErrorCodeInvalidRequest}
				}

				redirectURI := strings.TrimSpace(toString(oauth["redirectUri"]))
				if oauthClientID == oauthConfigClientID {
					useKimbapOauthConfigValue = true
					keyLength := int(math.Ceil(float64(len(ownerToken)) * 0.5))
					seed := ownerToken[keyLength:] + serverID + fmt.Sprintf("%t", allowUserInputValue)
					hashKey := sha256Hash(seed)

					codeVerifier := strings.TrimSpace(toString(oauth["codeVerifier"]))
					requestBody := map[string]any{
						"clientId":    oauthClientID,
						"provider":    provider,
						"key":         hashKey,
						"code":        oauthCode,
						"redirectUri": redirectURI,
					}
					if codeVerifier != "" {
						requestBody["codeVerifier"] = codeVerifier
					}

					if authTypeValue == types.ServerAuthTypeZendeskAuth {
						oauthScope := strings.TrimSpace(toString(oauthConfig["scope"]))
						if oauthScope != "" {
							requestBody["scope"] = oauthScope
						}
					}

					tokenURL := strings.TrimSpace(toString(oauthConfig["tokenUrl"]))
					if provider == "zendesk" || provider == "canvas" {
						if tokenURL == "" {
							return nil, &types.AdminError{Message: "Missing OAuth tokenUrl", Code: types.AdminErrorCodeInvalidRequest}
						}
						requestBody["tokenUrl"] = tokenURL
					}

					response, err := utils.PostJSON(config.GetKimbapAuthConfig().BaseURL+"/v1/oauth/exchange", requestBody)
					if err != nil {
						return nil, &types.AdminError{Message: "Failed to exchange OAuth code", Code: types.AdminErrorCodeInvalidRequest}
					}

					accessToken := strings.TrimSpace(toString(response["accessToken"]))
					expiresAt := utils.ValueAsInt64(response["expiresAt"])
					if accessToken == "" || expiresAt == 0 {
						return nil, &types.AdminError{Message: "Failed to exchange OAuth code", Code: types.AdminErrorCodeInvalidRequest}
					}

					decryptedLaunchConfigValue["oauth"] = map[string]any{
						"clientId":    oauthClientID,
						"key":         hashKey,
						"accessToken": accessToken,
						"expiresAt":   expiresAt,
					}

					encryptedData, err := security.EncryptData(toJSONString(decryptedLaunchConfigValue, "{}"), ownerToken)
					if err != nil {
						return nil, &types.AdminError{Message: "Failed to encrypt launchConfig", Code: types.AdminErrorCodeInvalidRequest}
					}
					launchConfigStr = encryptedData
				} else {
					useKimbapOauthConfigValue = false
					clientSecret := strings.TrimSpace(toString(oauth["clientSecret"]))
					if clientSecret == "" {
						return nil, &types.AdminError{Message: "Invalid OAuth client secret", Code: types.AdminErrorCodeInvalidRequest}
					}

					codeVerifier := strings.TrimSpace(toString(oauth["codeVerifier"]))
					oauthScope := ""
					if authTypeValue == types.ServerAuthTypeZendeskAuth {
						oauthScope = strings.TrimSpace(toString(oauthConfig["scope"]))
					}

					exchangeResult, err := mcpoauth.ExchangeAuthorizationCode(mcpoauth.ExchangeContext{
						Provider:     provider,
						TokenURL:     strings.TrimSpace(toString(oauthConfig["tokenUrl"])),
						ClientID:     oauthClientID,
						ClientSecret: clientSecret,
						Code:         oauthCode,
						RedirectURI:  redirectURI,
						CodeVerifier: codeVerifier,
						Scope:        oauthScope,
					})
					if err != nil || exchangeResult == nil || strings.TrimSpace(exchangeResult.AccessToken) == "" || strings.TrimSpace(exchangeResult.RefreshToken) == "" {
						return nil, &types.AdminError{Message: "Failed to exchange OAuth code", Code: types.AdminErrorCodeInvalidRequest}
					}

					expiresAt := time.Now().Add(30 * 24 * time.Hour).UnixMilli()
					if exchangeResult.ExpiresAt != nil && *exchangeResult.ExpiresAt > 0 {
						expiresAt = *exchangeResult.ExpiresAt
					}

					oauthData := map[string]any{
						"clientId":     oauthClientID,
						"clientSecret": clientSecret,
						"accessToken":  exchangeResult.AccessToken,
						"refreshToken": exchangeResult.RefreshToken,
						"expiresAt":    expiresAt,
					}
					if authTypeValue == types.ServerAuthTypeZendeskAuth || authTypeValue == types.ServerAuthTypeCanvasAuth {
						oauthData["tokenUrl"] = strings.TrimSpace(toString(oauthConfig["tokenUrl"]))
					}
					if authTypeValue == types.ServerAuthTypeZendeskAuth {
						oauthData["scope"] = strings.TrimSpace(toString(oauthConfig["scope"]))
					}
					if authTypeValue == types.ServerAuthTypeZendeskAuth {
						if rtei, ok := exchangeResult.Raw["refresh_token_expires_in"]; ok {
							if parsed, ok := mcpoauth.NumberToInt64(rtei); ok {
								oauthData["refreshTokenExpiresAt"] = time.Now().UnixMilli() + parsed*1000
							}
						}
					}
					decryptedLaunchConfigValue["oauth"] = oauthData

					encryptedData, err := security.EncryptData(toJSONString(decryptedLaunchConfigValue, "{}"), ownerToken)
					if err != nil {
						return nil, &types.AdminError{Message: "Failed to encrypt launchConfig", Code: types.AdminErrorCodeInvalidRequest}
					}
					launchConfigStr = encryptedData
				}
			} else {
				oauthUserClientID := strings.TrimSpace(toString(oauthConfig["userClientId"]))
				if oauthClientID != oauthUserClientID {
					useKimbapOauthConfigValue = false
					if strings.TrimSpace(toString(oauth["clientSecret"])) == "" {
						return nil, &types.AdminError{Message: "Invalid OAuth client secret", Code: types.AdminErrorCodeInvalidRequest}
					}
				}

				jwtSecret := strings.TrimSpace(config.Env("JWT_SECRET"))
				if jwtSecret == "" {
					return nil, &types.AdminError{Message: "JWT_SECRET environment variable is required", Code: types.AdminErrorCodeInvalidRequest}
				}

				encryptedData, err := security.EncryptData(decryptedLaunchConfig, jwtSecret)
				if err != nil {
					return nil, &types.AdminError{Message: "Failed to encrypt launchConfig", Code: types.AdminErrorCodeInvalidRequest}
				}
				launchConfigStr = encryptedData

				oauthConfig["deskClientId"] = oauthClientID
				configTemplateValue["oAuthConfig"] = oauthConfig
				configTemplateStr = toJSONString(configTemplateValue, "{}")
			}
		}
	}
	if categoryValue == types.ServerCategoryRestAPI && allowUserInputValue {
		jwtSecret := strings.TrimSpace(config.Env("JWT_SECRET"))
		if jwtSecret == "" {
			return nil, &types.AdminError{Message: "JWT_SECRET environment variable is required", Code: types.AdminErrorCodeInvalidRequest}
		}
		decrypted, err := security.DecryptDataFromString(launchConfigStr, ownerToken)
		if err != nil {
			return nil, &types.AdminError{Message: "Failed to decrypt launchConfig", Code: types.AdminErrorCodeInvalidRequest}
		}
		encryptedData, err := security.EncryptData(decrypted, jwtSecret)
		if err != nil {
			return nil, &types.AdminError{Message: "Failed to encrypt launchConfig", Code: types.AdminErrorCodeInvalidRequest}
		}
		launchConfigStr = encryptedData
	}

	configTemplatePtr := ptrIfNotEmpty(configTemplateStr)
	now := int(time.Now().Unix())
	enabledValue, _, err := parseBoolField(data, "enabled", true)
	if err != nil {
		return nil, err
	}
	publicAccessValue, _, err := parseBoolField(data, "publicAccess", false)
	if err != nil {
		return nil, err
	}
	anonymousAccessValue, _, err := parseBoolField(data, "anonymousAccess", false)
	if err != nil {
		return nil, err
	}
	anonymousRateLimitValue := 10
	if rawAnonRL, ok := data["anonymousRateLimit"]; ok {
		anonRL, valid := asIntMaybe(rawAnonRL)
		if !valid || anonRL < 1 || anonRL > 1000 {
			return nil, &types.AdminError{Message: "Invalid anonymousRateLimit: must be an integer between 1 and 1000", Code: types.AdminErrorCodeInvalidRequest}
		}
		anonymousRateLimitValue = anonRL
	}
	lazyStartEnabledValue, lazyStartProvided, err := parseBoolField(data, "lazyStartEnabled", false)
	if err != nil {
		return nil, err
	}
	server := database.Server{
		ServerID:             serverID,
		ServerName:           toString(data["serverName"]),
		Enabled:              enabledValue,
		LaunchConfig:         launchConfigStr,
		Capabilities:         "{\"tools\":{},\"resources\":{},\"prompts\":{}}",
		CreatedAt:            now,
		UpdatedAt:            now,
		AllowUserInput:       allowUserInputValue,
		ProxyID:              toInt(data["proxyId"], 0),
		ToolTmplID:           ptrIfNotEmpty(toString(data["toolTmplId"])),
		AuthType:             authTypeValue,
		ConfigTemplate:       configTemplatePtr,
		Category:             categoryValue,
		PublicAccess:         publicAccessValue,
		UseKimbapOauthConfig: useKimbapOauthConfigValue,
		AnonymousAccess:      anonymousAccessValue,
		AnonymousRateLimit:   anonymousRateLimitValue,
		LazyStartEnabled:     lazyStartEnabledValue,
	}
	createQuery := h.db
	if !lazyStartProvided {
		createQuery = createQuery.Omit("lazy_start_enabled")
	}
	if err := createQuery.Create(&server).Error; err != nil {
		return nil, err
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminServerCreate,
		RequestParams: toJSONString(map[string]any{
			"serverId": serverID,
			"authType": data["authType"],
		}, "{}"),
	})
	return map[string]any{"server": server}, nil
}

func (h *ServerHandler) GetServers(data map[string]any) (any, error) {
	query := h.db.Model(&database.Server{}).Omit("cached_tools", "cached_resources", "cached_resource_templates", "cached_prompts")
	if serverID := toString(data["serverId"]); serverID != "" {
		query = query.Where("server_id = ?", serverID)
	}
	if proxyRaw, ok := data["proxyId"]; ok {
		query = query.Where("proxy_id = ?", toInt(proxyRaw, 0))
	}
	if enabledRaw, ok := data["enabled"]; ok {
		enabled, err := parseBoolLike(enabledRaw, false)
		if err != nil {
			return nil, &types.AdminError{Message: "enabled must be a boolean", Code: types.AdminErrorCodeInvalidRequest}
		}
		query = query.Where("enabled = ?", enabled)
	}
	var servers []database.Server
	if err := query.Find(&servers).Error; err != nil {
		return nil, err
	}
	response := make([]serverListResponse, 0, len(servers))
	for _, server := range servers {
		configTemplate := server.ConfigTemplate
		if server.Category == types.ServerCategoryTemplate {
			configTemplate = nil
		}
		response = append(response, serverListResponse{
			ServerID:             server.ServerID,
			ServerName:           server.ServerName,
			Enabled:              server.Enabled,
			LaunchConfig:         redactLaunchConfig(server.LaunchConfig),
			Capabilities:         server.Capabilities,
			CreatedAt:            server.CreatedAt,
			UpdatedAt:            server.UpdatedAt,
			AllowUserInput:       server.AllowUserInput,
			ConfigTemplate:       configTemplate,
			ProxyID:              server.ProxyID,
			ToolTmplID:           server.ToolTmplID,
			AuthType:             server.AuthType,
			Category:             server.Category,
			LazyStartEnabled:     server.LazyStartEnabled,
			PublicAccess:         server.PublicAccess,
			UseKimbapOauthConfig: server.UseKimbapOauthConfig,
			TransportType:        server.TransportType,
			AnonymousAccess:      server.AnonymousAccess,
			AnonymousRateLimit:   server.AnonymousRateLimit,
		})
	}
	return map[string]any{"servers": response}, nil
}

func (h *ServerHandler) UpdateServer(data map[string]any, ownerToken string) (any, error) {
	serverID := toString(data["serverId"])
	if serverID == "" {
		return nil, &types.AdminError{Message: "missing serverId", Code: types.AdminErrorCodeInvalidRequest}
	}
	existing, err := h.findServer(serverID)
	if err != nil {
		return nil, err
	}
	affectedSessions := h.sessionStore.GetSessionsUsingServer(serverID)
	updates := map[string]any{"updated_at": int(time.Now().Unix())}
	nameChanged := false
	configChanged := false
	capabilitiesChanged := false
	if rawAllowUserInput, ok := data["allowUserInput"]; ok {
		allowUserInput, err := parseBoolLike(rawAllowUserInput, existing.AllowUserInput)
		if err != nil {
			return nil, &types.AdminError{Message: "Invalid allowUserInput", Code: types.AdminErrorCodeInvalidRequest}
		}
		if allowUserInput != existing.AllowUserInput {
			return nil, &types.AdminError{Message: "allowUserInput field is immutable after server creation", Code: types.AdminErrorCodeInvalidRequest}
		}
	}
	hasEditableConfigTemplate := existing.Category == types.ServerCategoryRestAPI || existing.Category == types.ServerCategoryCustomRemote || existing.Category == types.ServerCategoryCustomStdio
	if _, ok := data["serverName"]; ok {
		newServerName := toString(data["serverName"])
		updates["server_name"] = newServerName
		if newServerName != existing.ServerName {
			nameChanged = true
		}
	}
	if _, ok := data["launchConfig"]; ok {
		newLaunchConfig := toJSONString(data["launchConfig"], "{}")
		if isEmptyConfigString(newLaunchConfig) {
			return nil, &types.AdminError{Message: "launchConfig must be a string and cannot be empty", Code: types.AdminErrorCodeInvalidRequest}
		}
		if newLaunchConfig != existing.LaunchConfig {
			configChanged = true
			if existing.AllowUserInput && !(existing.Category == types.ServerCategoryRestAPI || existing.Category == types.ServerCategoryCustomRemote || existing.Category == types.ServerCategoryCustomStdio) {
				return nil, &types.AdminError{Message: "This type of server does not allow modification of the launch configuration.", Code: types.AdminErrorCodeInvalidRequest}
			}
		}
		updates["launch_config"] = newLaunchConfig
	}
	if _, ok := data["capabilities"]; ok {
		if mergedCapabilities, changed, mergeErr := mergeCapabilities(existing.Capabilities, data["capabilities"]); mergeErr != nil {
			return nil, &types.AdminError{Message: "invalid capabilities", Code: types.AdminErrorCodeInvalidRequest}
		} else if changed {
			updates["capabilities"] = mergedCapabilities
			capabilitiesChanged = true
		}
	}
	if newEnabled, provided, err := parseBoolField(data, "enabled", existing.Enabled); err != nil {
		return nil, err
	} else if provided {
		updates["enabled"] = newEnabled
		if newEnabled != existing.Enabled {
			configChanged = true
		}
	}
	if newLazyStart, provided, err := parseBoolField(data, "lazyStartEnabled", existing.LazyStartEnabled); err != nil {
		return nil, err
	} else if provided {
		updates["lazy_start_enabled"] = newLazyStart
		if newLazyStart != existing.LazyStartEnabled {
			configChanged = true
		}
	}
	if newPublicAccess, provided, err := parseBoolField(data, "publicAccess", existing.PublicAccess); err != nil {
		return nil, err
	} else if provided {
		updates["public_access"] = newPublicAccess
		if newPublicAccess != existing.PublicAccess {
			configChanged = true
		}
	}
	if newAnonymousAccess, provided, err := parseBoolField(data, "anonymousAccess", existing.AnonymousAccess); err != nil {
		return nil, err
	} else if provided {
		updates["anonymous_access"] = newAnonymousAccess
		if newAnonymousAccess != existing.AnonymousAccess {
			configChanged = true
		}
	}
	if rawAnonRL, ok := data["anonymousRateLimit"]; ok {
		anonRL, valid := asIntMaybe(rawAnonRL)
		if !valid || anonRL < 1 || anonRL > 1000 {
			return nil, &types.AdminError{Message: "Invalid anonymousRateLimit: must be an integer between 1 and 1000", Code: types.AdminErrorCodeInvalidRequest}
		}
		if anonRL != existing.AnonymousRateLimit {
			updates["anonymous_rate_limit"] = anonRL
			configChanged = true
		}
	}
	if _, ok := data["configTemplate"]; ok {
		if !hasEditableConfigTemplate {
			return nil, &types.AdminError{Message: "configTemplate field is immutable after server creation", Code: types.AdminErrorCodeInvalidRequest}
		}
		rawConfigTemplate := data["configTemplate"]
		if rawConfigTemplate != nil {
			v := toString(rawConfigTemplate)
			if strings.TrimSpace(v) != "" {
				newConfigTemplate := &v
				if !stringPtrEqual(newConfigTemplate, existing.ConfigTemplate) {
					updates["config_template"] = newConfigTemplate
					configChanged = true
				}
			}
		}
	}
	res := h.db.Model(&database.Server{}).Where("server_id = ?", serverID).Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, &types.AdminError{Message: "server not found", Code: types.AdminErrorCodeServerNotFound}
	}
	var server database.Server
	if err := h.db.Where("server_id = ?", serverID).First(&server).Error; err != nil {
		return nil, err
	}

	notifyServerChange := nameChanged || configChanged || capabilitiesChanged
	forceNotifyAllCapabilities := false
	runtimeConfigChanged := existing.LaunchConfig != server.LaunchConfig ||
		existing.LazyStartEnabled != server.LazyStartEnabled ||
		!stringPtrEqual(existing.ConfigTemplate, server.ConfigTemplate)
	var notificationContext *core.ServerContext
	if existing.Enabled && !server.Enabled {
		if server.AllowUserInput {
			forceNotifyAllCapabilities = true
			notificationContext = h.getLastTemporaryServerContextByTemplate(serverID)
			h.serverManager.CloseAllTemporaryServersByTemplate(serverID)
		} else {
			serverContext, err := h.serverManager.RemoveServer(context.Background(), serverID)
			if err != nil {
				return nil, err
			}
			notificationContext = serverContext
		}
	} else if !existing.Enabled && server.Enabled {
		if server.AllowUserInput {
			forceNotifyAllCapabilities = true
			h.serverManager.UpdateTemporaryServersByTemplate(server)
			notificationContext = h.getLastTemporaryServerContextByTemplate(serverID)
		} else {
			serverContext, err := h.serverManager.AddServer(context.Background(), server, ownerToken)
			if err != nil {
				return nil, err
			}
			notificationContext = serverContext
		}
	} else if existing.Enabled && server.Enabled {
		if server.AllowUserInput {
			h.serverManager.UpdateTemporaryServersByTemplate(server)
			notificationContext = h.getLastTemporaryServerContextByTemplate(serverID)
		} else {
			runtimeContext := h.serverManager.GetServerContext(serverID, "")
			switch {
			case runtimeContext != nil && runtimeConfigChanged:
				serverContext, err := h.serverManager.ReconnectServer(context.Background(), server, ownerToken)
				if err != nil {
					return nil, err
				}
				notificationContext = serverContext
			case runtimeContext == nil:
				serverContext, err := h.serverManager.AddServer(context.Background(), server, ownerToken)
				if err != nil {
					return nil, err
				}
				notificationContext = serverContext
			default:
				runtimeContext.UpdateServerEntity(server)
				notificationContext = runtimeContext
			}
		}
	}

	if capabilitiesChanged {
		_, _, _, err := h.serverManager.UpdateServerCapabilitiesConfig(context.Background(), serverID, server.Capabilities)
		if err != nil {
			return nil, err
		}
		if server.Enabled {
			if server.AllowUserInput {
				notificationContext = h.getLastTemporaryServerContextByTemplate(serverID)
			} else {
				notificationContext = h.serverManager.GetServerContext(serverID, "")
			}
		}
	}

	if notifyServerChange && notificationContext != nil {
		toolsChanged, resourcesChanged, promptsChanged := notificationContext.CapabilityChanged()
		h.notifyCapabilityChangedSessionsByFlags(affectedSessions, toolsChanged, resourcesChanged, promptsChanged)
		h.socketNotifier.NotifyUserPermissionChangedByServer(serverID)
	} else if notifyServerChange && forceNotifyAllCapabilities {
		h.notifyCapabilityChangedSessionsByFlags(affectedSessions, true, true, true)
		h.socketNotifier.NotifyUserPermissionChangedByServer(serverID)
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminServerEdit,
		RequestParams: toJSONString(map[string]any{
			"serverId": serverID,
		}, "{}"),
	})
	if server.Category == types.ServerCategoryTemplate {
		server.ConfigTemplate = nil
	}
	server.TransportType = nil
	server.CachedTools = nil
	server.CachedResources = nil
	server.CachedResourceTemplates = nil
	server.CachedPrompts = nil
	return map[string]any{"server": server}, nil
}

func (h *ServerHandler) DeleteServer(data map[string]any) (any, error) {
	serverID := toString(data["serverId"])
	if serverID == "" {
		return nil, &types.AdminError{Message: "missing serverId", Code: types.AdminErrorCodeInvalidRequest}
	}
	server, err := h.findServer(serverID)
	if err != nil {
		return nil, err
	}
	if server.AllowUserInput {
		h.serverManager.CloseAllTemporaryServersByTemplate(serverID)
		if err := repository.NewUserRepository(h.db).RemoveServerFromAllUsers(serverID); err != nil {
			return nil, err
		}
		log.Info().Str("serverId", serverID).Msg("cleaned up all user configurations for template server")
	}

	affectedSessions := h.sessionStore.GetSessionsUsingServer(serverID)

	toolsChanged := false
	resourcesChanged := false
	promptsChanged := false
	if server.AllowUserInput {
		toolsChanged = true
		resourcesChanged = true
		promptsChanged = true
	} else {
		serverContext, err := h.serverManager.RemoveServer(context.Background(), serverID)
		if err != nil {
			return nil, err
		}
		if serverContext != nil {
			toolsChanged, resourcesChanged, promptsChanged = serverContext.CapabilityChanged()
		}
	}

	if err := h.db.Where("server_id = ?", serverID).Delete(&database.Server{}).Error; err != nil {
		return nil, err
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminServerDelete,
		RequestParams: toJSONString(map[string]any{
			"serverId": serverID,
		}, "{}"),
	})

	h.notifyCapabilityChangedSessionsByFlags(affectedSessions, toolsChanged, resourcesChanged, promptsChanged)
	h.socketNotifier.NotifyUserPermissionChangedByServer(serverID)
	return map[string]any{"message": "Server deleted successfully"}, nil
}

func (h *ServerHandler) DeleteServersByProxy(data map[string]any) (any, error) {
	proxyID, ok := asIntMaybe(data["proxyId"])
	if !ok {
		return nil, &types.AdminError{Message: "missing proxyId", Code: types.AdminErrorCodeInvalidRequest}
	}
	var servers []database.Server
	if err := h.db.Where("proxy_id = ?", proxyID).Find(&servers).Error; err != nil {
		return nil, err
	}
	for _, server := range servers {
		affectedSessions := h.sessionStore.GetSessionsUsingServer(server.ServerID)

		if server.AllowUserInput {
			h.serverManager.CloseAllTemporaryServersByTemplate(server.ServerID)
			if err := repository.NewUserRepository(h.db).RemoveServerFromAllUsers(server.ServerID); err != nil {
				return nil, err
			}
			h.notifyCapabilityChangedSessionsByFlags(affectedSessions, true, true, true)
		} else {
			serverContext, err := h.serverManager.RemoveServer(context.Background(), server.ServerID)
			if err != nil {
				return nil, err
			}
			toolsChanged, resourcesChanged, promptsChanged := false, false, false
			if serverContext != nil {
				toolsChanged, resourcesChanged, promptsChanged = serverContext.CapabilityChanged()
			}
			h.notifyCapabilityChangedSessionsByFlags(affectedSessions, toolsChanged, resourcesChanged, promptsChanged)
		}
		h.socketNotifier.NotifyUserPermissionChangedByServer(server.ServerID)
	}
	res := h.db.Where("proxy_id = ?", proxyID).Delete(&database.Server{})
	if res.Error != nil {
		return nil, res.Error
	}
	return map[string]any{"deletedCount": res.RowsAffected}, nil
}

func (h *ServerHandler) CountServers() (any, error) {
	var count int64
	if err := h.db.Model(&database.Server{}).Count(&count).Error; err != nil {
		return nil, err
	}
	return map[string]any{"count": count}, nil
}

func (h *ServerHandler) findServer(serverID string) (database.Server, error) {
	var server database.Server
	if err := h.db.Where("server_id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return server, &types.AdminError{Message: "server not found", Code: types.AdminErrorCodeServerNotFound}
		}
		return server, err
	}
	return server, nil
}

func parseBoolField(data map[string]any, key string, fallback bool) (bool, bool, error) {
	raw, ok := data[key]
	if !ok {
		return fallback, false, nil
	}
	if raw == nil {
		return false, false, &types.AdminError{Message: key + " must be a boolean", Code: types.AdminErrorCodeInvalidRequest}
	}
	val, err := parseBoolLike(raw, fallback)
	if err != nil {
		return false, false, &types.AdminError{Message: key + " must be a boolean", Code: types.AdminErrorCodeInvalidRequest}
	}
	return val, true, nil
}

func parseBoolLike(v any, fallback bool) (bool, error) {
	if v == nil {
		return fallback, nil
	}
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		switch strings.TrimSpace(strings.ToLower(x)) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		}
	case float64:
		if x == 0 || x == 1 {
			return x == 1, nil
		}
	case float32:
		if x == 0 || x == 1 {
			return x == 1, nil
		}
	case int:
		if x == 0 || x == 1 {
			return x == 1, nil
		}
	case int64:
		if x == 0 || x == 1 {
			return x == 1, nil
		}
	}
	return false, errors.New("invalid boolean-like value")
}

func optionalStringValue(v any) (string, bool, error) {
	if v == nil {
		return "", false, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", true, errors.New("invalid string")
	}
	return s, true, nil
}

func isEmptyConfigString(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed == "" || trimmed == "{}"
}

func isValidServerAuthType(value int) bool {
	switch value {
	case types.ServerAuthTypeApiKey,
		types.ServerAuthTypeGoogleAuth,
		types.ServerAuthTypeNotionAuth,
		types.ServerAuthTypeFigmaAuth,
		types.ServerAuthTypeGoogleCalendarAuth,
		types.ServerAuthTypeGithubAuth,
		types.ServerAuthTypeZendeskAuth,
		types.ServerAuthTypeCanvasAuth,
		types.ServerAuthTypeCanvaAuth:
		return true
	default:
		return false
	}
}

func isValidServerCategory(value int) bool {
	switch value {
	case types.ServerCategoryTemplate,
		types.ServerCategoryCustomRemote,
		types.ServerCategoryRestAPI,
		types.ServerCategorySkills,
		types.ServerCategoryCustomStdio:
		return true
	default:
		return false
	}
}

func sha256Hash(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash)
}

func (h *ServerHandler) notifyCapabilityChangedSessionsByFlags(sessions []*core.ClientSession, toolsChanged, resourcesChanged, promptsChanged bool) {
	if !toolsChanged && !resourcesChanged && !promptsChanged {
		return
	}
	for _, session := range sessions {
		if session == nil {
			continue
		}
		proxySession := session.GetProxySession()
		if proxySession == nil {
			continue
		}
		if toolsChanged {
			proxySession.SendToolsListChangedToClient()
		}
		if resourcesChanged {
			proxySession.SendResourcesListChangedToClient()
		}
		if promptsChanged {
			proxySession.SendPromptsListChangedToClient()
		}
	}
}

func (h *ServerHandler) getLastTemporaryServerContextByTemplate(serverID string) *core.ServerContext {
	temporaryServers := h.serverManager.GetTemporaryServers()
	var matched *core.ServerContext
	for _, temporaryServer := range temporaryServers {
		if temporaryServer == nil || temporaryServer.ServerID != serverID {
			continue
		}
		matched = temporaryServer
	}
	return matched
}

func stringPtrEqual(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func mergeCapabilitiesJSON(oldRaw string, overlayRaw string) (string, error) {
	merged, changed, err := mergeCapabilities(oldRaw, overlayRaw)
	if err != nil {
		return "", err
	}
	if !changed {
		return oldRaw, nil
	}
	return merged, nil
}

func mergeCapabilities(existingRaw string, incoming any) (string, bool, error) {
	if incoming == nil {
		return "", false, nil
	}
	incomingRaw := toJSONString(incoming, "")
	if strings.TrimSpace(incomingRaw) == "" {
		return "", false, nil
	}

	overlay, err := parseCapabilities(incomingRaw)
	if err != nil {
		return "", false, err
	}
	if incomingRaw == existingRaw {
		return "", false, nil
	}
	base, err := parseCapabilities(existingRaw)
	if err != nil {
		overlayRaw, marshalErr := json.Marshal(overlay)
		if marshalErr != nil {
			return "", false, marshalErr
		}
		normalizedOverlay := string(overlayRaw)
		if normalizedOverlay == existingRaw {
			return "", false, nil
		}
		return normalizedOverlay, true, nil
	}
	changed := false

	if len(overlay.Tools) > 0 {
		for name, cfg := range overlay.Tools {
			base.Tools[name] = cfg
		}
		changed = true
	}
	if len(overlay.Resources) > 0 {
		for name, cfg := range overlay.Resources {
			base.Resources[name] = cfg
		}
		changed = true
	}
	if len(overlay.Prompts) > 0 {
		for name, cfg := range overlay.Prompts {
			base.Prompts[name] = cfg
		}
		changed = true
	}
	if !changed {
		return "", false, nil
	}

	mergedRaw, err := json.Marshal(base)
	if err != nil {
		return "", false, err
	}
	merged := string(mergedRaw)
	if merged == existingRaw {
		return "", false, nil
	}
	return merged, true, nil
}

func parseCapabilities(raw string) (mcptypes.ServerConfigCapabilities, error) {
	out := mcptypes.ServerConfigCapabilities{
		Tools:     map[string]mcptypes.ToolCapabilityConfig{},
		Resources: map[string]mcptypes.ResourceCapabilityConfig{},
		Prompts:   map[string]mcptypes.PromptCapabilityConfig{},
	}
	if strings.TrimSpace(raw) == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return out, err
	}
	if out.Tools == nil {
		out.Tools = map[string]mcptypes.ToolCapabilityConfig{}
	}
	if out.Resources == nil {
		out.Resources = map[string]mcptypes.ResourceCapabilityConfig{}
	}
	if out.Prompts == nil {
		out.Prompts = map[string]mcptypes.PromptCapabilityConfig{}
	}
	return out, nil
}

var launchConfigSensitiveKeys = []string{
	"clientsecret", "client_secret", "accesstoken", "access_token",
	"refreshtoken", "refresh_token", "apikey", "api_key",
	"password", "secret", "token", "authorization", "credential",
	"privatekey", "private_key",
}

func redactLaunchConfig(raw string) string {
	if raw == "" || raw == "{}" {
		return raw
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "{}"
	}
	redactMapSensitive(parsed)
	b, err := json.Marshal(parsed)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func redactMapSensitive(m map[string]any) {
	for k, v := range m {
		lower := strings.ToLower(k)
		sensitive := false
		for _, sk := range launchConfigSensitiveKeys {
			if strings.Contains(lower, sk) {
				sensitive = true
				break
			}
		}
		if sensitive {
			m[k] = "[REDACTED]"
			continue
		}
		switch typed := v.(type) {
		case map[string]any:
			redactMapSensitive(typed)
		case []any:
			for _, item := range typed {
				if sub, ok := item.(map[string]any); ok {
					redactMapSensitive(sub)
				}
			}
		}
	}
}

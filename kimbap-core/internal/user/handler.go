package user

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcpoauth "github.com/dunialabs/kimbap-core/internal/mcp/oauth"
	mcpservice "github.com/dunialabs/kimbap-core/internal/mcp/service"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/security"
	coretypes "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/dunialabs/kimbap-core/internal/utils"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RequestHandler struct {
	db *gorm.DB
}

func NewRequestHandler(db *gorm.DB) *RequestHandler {
	if db == nil {
		db = database.DB
	}
	return &RequestHandler{db: db}
}

func (h *RequestHandler) GetCapabilities(userID string) (map[string]any, error) {
	capabilities, err := mcpservice.CapabilitiesServiceInstance().GetUserCapabilities(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(capabilities))
	for k, v := range capabilities {
		out[k] = v
	}
	return out, nil
}

func (h *RequestHandler) SetCapabilities(userID string, submitted map[string]any) error {
	if submitted == nil {
		submitted = map[string]any{}
	}

	current, err := mcpservice.CapabilitiesServiceInstance().GetUserCapabilities(context.Background(), userID)
	if err != nil {
		return err
	}

	deltas := extractEnabledFieldDeltas(submitted, current)

	var mergedJSON string
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var freshUser database.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&freshUser).Error; err != nil {
			return err
		}
		existing := map[string]any{}
		if freshUser.UserPreferences != "" {
			_ = json.Unmarshal([]byte(freshUser.UserPreferences), &existing)
		}
		for serverID, serverDeltas := range deltas {
			serverPrefs, ok := existing[serverID].(map[string]any)
			if !ok {
				serverPrefs = map[string]any{}
			}
			deltaMap, ok := serverDeltas.(map[string]any)
			if !ok {
				continue
			}
			for k, v := range deltaMap {
				serverPrefs[k] = v
			}
			existing[serverID] = serverPrefs
		}
		b, err := json.Marshal(existing)
		if err != nil {
			return err
		}
		mergedJSON = string(b)
		return tx.Model(&database.User{}).Where("user_id = ?", userID).Updates(map[string]any{"user_preferences": mergedJSON, "updated_at": int(time.Now().Unix())}).Error
	}); err != nil {
		return err
	}

	if mergedJSON != "" {
		var mergedMap map[string]any
		_ = json.Unmarshal([]byte(mergedJSON), &mergedMap)
		syncUserPreferencesToSessions(userID, mustPermissionsFromAny(mergedMap))
	}
	core.ServerManagerInstance().NotifyUserPermissionChanged(userID)
	return nil
}

func (h *RequestHandler) ConfigureServer(userID, userToken string, data map[string]any) (map[string]any, error) {
	serverID, _ := data["serverId"].(string)
	if serverID == "" {
		return nil, &UserError{Message: "serverId is required", Code: UserErrorInvalidRequest}
	}
	var server database.Server
	if err := h.db.Where("server_id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &UserError{Message: "Server not found", Code: UserErrorServerNotFound}
		}
		return nil, err
	}
	user, err := h.getUser(userID)
	if err != nil {
		return nil, err
	}
	if !isServerEnabledForUser(user, server) {
		return nil, &UserError{Message: "Server not found", Code: UserErrorServerNotFound}
	}
	if !server.AllowUserInput {
		return nil, &UserError{Message: "Server does not allow user input", Code: UserErrorServerNoUserInput}
	}
	if !server.Enabled {
		return nil, &UserError{Message: "Server is disabled", Code: UserErrorServerDisabled}
	}
	if server.ConfigTemplate == nil || strings.TrimSpace(*server.ConfigTemplate) == "" {
		return nil, &UserError{Message: "Server does not have a configuration template", Code: UserErrorServerNoTemplate}
	}

	launchConfig := map[string]any{}
	switch server.Category {
	case coretypes.ServerCategoryTemplate:
		template := map[string]any{}
		if err := json.Unmarshal([]byte(*server.ConfigTemplate), &template); err != nil {
			return nil, &UserError{Message: "Invalid configTemplate", Code: UserErrorServerConfigInvalid}
		}
		authConf := data["authConf"]
		if authConf == nil {
			return nil, &UserError{Message: "authConf is required", Code: UserErrorServerConfigInvalid}
		}
		launchConfig, err = h.buildTemplateLaunchConfig(server, userToken, serverID, template, authConf)
		if err != nil {
			return nil, err
		}
	case coretypes.ServerCategoryCustomRemote:
		if server.ConfigTemplate == nil || *server.ConfigTemplate == "" || *server.ConfigTemplate == "{}" {
			return nil, &UserError{Message: fmt.Sprintf("Server %s does not have a configuration template", serverID), Code: UserErrorServerNoTemplate}
		}
		var configTemplateObj map[string]any
		if err := json.Unmarshal([]byte(*server.ConfigTemplate), &configTemplateObj); err != nil {
			return nil, &UserError{Message: "Invalid configTemplate JSON", Code: UserErrorServerConfigInvalid}
		}

		if command, ok := configTemplateObj["command"].(string); ok && strings.TrimSpace(command) != "" {
			args, _ := configTemplateObj["args"].([]any)
			argsStr := make([]string, 0, len(args))
			for _, a := range args {
				if s, ok := a.(string); ok {
					argsStr = append(argsStr, s)
				}
			}
			envMap := map[string]any{}
			if e, ok := configTemplateObj["env"].(map[string]any); ok {
				envMap = e
			}
			launchConfig = map[string]any{
				"command": command,
				"args":    argsStr,
				"env":     envMap,
			}
		} else {
			remote, _ := data["remoteAuth"].(map[string]any)
			if remote == nil {
				return nil, &UserError{Message: "remoteAuth is required and cannot be empty and must contain either params or headers", Code: UserErrorServerConfigInvalid}
			}
			params, _ := remote["params"].(map[string]any)
			headers, _ := remote["headers"].(map[string]any)
			if len(params) == 0 && len(headers) == 0 {
				return nil, &UserError{Message: "remoteAuth is required and cannot be empty and must contain either params or headers", Code: UserErrorServerConfigInvalid}
			}
			baseURL := strings.TrimSpace(toString(configTemplateObj["url"]))
			if baseURL == "" {
				return nil, &UserError{Message: "configTemplate.url is required", Code: UserErrorServerConfigInvalid}
			}
			if headers == nil {
				headers = map[string]any{}
			}
			finalURL := baseURL
			if len(params) > 0 {
				queryString := encodeURLParams(params)
				if strings.Contains(finalURL, "?") {
					parts := strings.SplitN(finalURL, "?", 2)
					finalURL = parts[0]
				}
				finalURL = finalURL + "?" + queryString
			}
			launchConfig = map[string]any{"url": finalURL, "headers": headers}
		}
	case coretypes.ServerCategoryCustomStdio:
		if server.ConfigTemplate == nil || *server.ConfigTemplate == "" || *server.ConfigTemplate == "{}" {
			return nil, &UserError{Message: fmt.Sprintf("Server %s does not have a configuration template", serverID), Code: UserErrorServerNoTemplate}
		}
		var stdioTemplate map[string]any
		if err := json.Unmarshal([]byte(*server.ConfigTemplate), &stdioTemplate); err != nil {
			return nil, &UserError{Message: "Invalid configTemplate JSON", Code: UserErrorServerConfigInvalid}
		}
		command := strings.TrimSpace(toString(stdioTemplate["command"]))
		if command == "" {
			return nil, &UserError{Message: fmt.Sprintf("Server %s has invalid stdio configuration: missing command", serverID), Code: UserErrorServerConfigInvalid}
		}

		args := []string{}
		if rawArgs, ok := stdioTemplate["args"].([]any); ok {
			args = make([]string, 0, len(rawArgs))
			for _, arg := range rawArgs {
				if arg == nil {
					continue
				}
				args = append(args, toString(arg))
			}
		}
		cwd := strings.TrimSpace(toString(stdioTemplate["cwd"]))

		adminEnv := map[string]any{}
		if envRaw, ok := stdioTemplate["env"].(map[string]any); ok && envRaw != nil {
			for key, value := range envRaw {
				adminEnv[key] = value
			}
		}
		if userOverrides, ok := data["stdioEnv"].(map[string]any); ok && userOverrides != nil {
			for key, value := range userOverrides {
				if strings.TrimSpace(key) == "" || value == nil {
					continue
				}
				if isDangerousEnvKey(key) {
					continue
				}
				adminEnv[key] = toString(value)
			}
		}

		launchConfig = map[string]any{
			"command": command,
			"args":    args,
			"env":     adminEnv,
		}
		if cwd != "" {
			launchConfig["cwd"] = cwd
		}
	case coretypes.ServerCategoryRestAPI:
		if isEmptyRestfulAPIAuth(data["restfulApiAuth"]) {
			return nil, &UserError{Message: "restfulApiAuth is required", Code: UserErrorServerConfigInvalid}
		}
		jwtSecret := strings.TrimSpace(config.Env("JWT_SECRET"))
		if jwtSecret == "" {
			return nil, &UserError{Message: "JWT_SECRET environment variable is not configured", Code: UserErrorInternal}
		}
		decrypted, err := security.DecryptDataFromString(server.LaunchConfig, jwtSecret)
		if err != nil {
			return nil, &UserError{Message: "Failed to decrypt server launch config", Code: UserErrorServerConfigInvalid}
		}
		if err := json.Unmarshal([]byte(decrypted), &launchConfig); err != nil {
			return nil, &UserError{Message: "Invalid server launch config", Code: UserErrorServerConfigInvalid}
		}
		if launchConfig == nil {
			launchConfig = map[string]any{}
		}
		launchConfig["auth"] = data["restfulApiAuth"]
	default:
		return nil, &UserError{Message: "Invalid server category", Code: UserErrorServerConfigInvalid}
	}

	encryptedObj, err := security.EncryptDataToObject(string(mustJSON(launchConfig)), userToken)
	if err != nil {
		return nil, err
	}

	var launchConfigsJSON string
	var launchConfigs map[string]any
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var freshUser database.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&freshUser).Error; err != nil {
			return err
		}
		var parseErr error
		launchConfigs, parseErr = parseJSONField(freshUser.LaunchConfigs, "launch configs")
		if parseErr != nil {
			return parseErr
		}
		launchConfigs[serverID] = encryptedObj
		launchConfigsJSON = string(mustJSON(launchConfigs))
		return tx.Model(&database.User{}).Where("user_id = ?", userID).Updates(map[string]any{"launch_configs": launchConfigsJSON, "updated_at": int(time.Now().Unix())}).Error
	}); err != nil {
		return nil, err
	}

	syncLaunchConfigsToSessions(userID, launchConfigsJSON)

	encryptedStr, _ := security.EncryptedAnyToString(encryptedObj)
	serverCopy := server
	serverCopy.LaunchConfig = encryptedStr
	serverContext, err := core.ServerManagerInstance().CreateTemporaryServer(context.Background(), userID, serverCopy, userToken, false)
	if err != nil {
		if rbErr := h.rollbackLaunchConfig(userID, serverID); rbErr != nil {
			log.Error().Err(rbErr).Str("userID", userID).Str("serverID", serverID).Msg("failed to rollback launch config after server creation failure")
		}
		return nil, err
	}

	var prefsJSON string
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var freshUser database.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&freshUser).Error; err != nil {
			return err
		}
		freshLC, lcErr := parseJSONField(freshUser.LaunchConfigs, "launch configs")
		if lcErr != nil {
			return lcErr
		}
		if _, stillExists := freshLC[serverID]; !stillExists {
			return fmt.Errorf("server %s was unconfigured concurrently", serverID)
		}
		prefs, parseErr := parseJSONField(freshUser.UserPreferences, "user preferences")
		if parseErr != nil {
			return parseErr
		}
		prefs[serverID] = buildServerPreference(serverContext)
		prefsJSON = string(mustJSON(prefs))
		return tx.Model(&database.User{}).Where("user_id = ?", userID).Updates(map[string]any{"user_preferences": prefsJSON, "updated_at": int(time.Now().Unix())}).Error
	}); err != nil {
		_, _ = core.ServerManagerInstance().CloseTemporaryServer(context.Background(), serverID, userID)
		if rbErr := h.rollbackLaunchConfig(userID, serverID); rbErr != nil {
			log.Error().Err(rbErr).Str("userID", userID).Str("serverID", serverID).Msg("failed to rollback launch config after preferences update failure")
		}
		return nil, err
	}

	if prefsJSON != "" {
		var prefsMap map[string]any
		_ = json.Unmarshal([]byte(prefsJSON), &prefsMap)
		syncUserPreferencesToSessions(userID, mustPermissionsFromAny(prefsMap))
	}
	core.ServerManagerInstance().NotifyUserPermissionChanged(userID)

	return map[string]any{"serverId": serverID, "message": "Server configured and started successfully"}, nil
}

func (h *RequestHandler) buildTemplateLaunchConfig(server database.Server, userToken string, serverID string, template map[string]any, authConf any) (map[string]any, error) {
	oauthCfg, _ := template["oAuthConfig"].(map[string]any)
	deskClientID := strings.TrimSpace(toString(oauthCfg["deskClientId"]))
	if deskClientID == "" {
		return buildLaunchConfig(template, authConf)
	}

	jwtSecret := strings.TrimSpace(config.Env("JWT_SECRET"))
	if jwtSecret == "" {
		return nil, &UserError{Message: "JWT_SECRET environment variable is not configured", Code: UserErrorInternal}
	}

	decrypted, err := security.DecryptDataFromString(server.LaunchConfig, jwtSecret)
	if err != nil {
		return nil, &UserError{Message: "Failed to decrypt server launch config", Code: UserErrorServerConfigInvalid}
	}

	launchConfig := map[string]any{}
	if err := json.Unmarshal([]byte(decrypted), &launchConfig); err != nil {
		return nil, &UserError{Message: "Invalid decrypted launch config", Code: UserErrorServerConfigInvalid}
	}
	if launchConfig == nil {
		return nil, &UserError{Message: "Invalid decrypted launch config", Code: UserErrorServerConfigInvalid}
	}
	oauthMap, _ := launchConfig["oauth"].(map[string]any)
	clientID := strings.TrimSpace(toString(oauthMap["clientId"]))
	clientSecret := strings.TrimSpace(toString(oauthMap["clientSecret"]))
	if clientID == "" {
		return nil, &UserError{Message: "Invalid OAuth launch config", Code: UserErrorServerConfigInvalid}
	}

	authValues, err := authConfToMap(authConf)
	if err != nil {
		return nil, &UserError{Message: "invalid server configuration", Code: UserErrorServerConfigInvalid}
	}
	oauthCode := strings.TrimSpace(authValues["YOUR_OAUTH_CODE"].Value)
	redirectURI := strings.TrimSpace(authValues["YOUR_OAUTH_REDIRECT_URL"].Value)
	oauthCodeVerifier := ""
	if v, ok := authValues["YOUR_OAUTH_PKCE_VERIFIER"]; ok {
		oauthCodeVerifier = strings.TrimSpace(v.Value)
	}
	if oauthCode == "" {
		return nil, &UserError{Message: "code is required and cannot be empty", Code: UserErrorServerConfigInvalid}
	}
	if redirectURI == "" {
		return nil, &UserError{Message: "redirectUri is required and cannot be empty", Code: UserErrorServerConfigInvalid}
	}

	provider := utils.OAuthProviderFromAuthType(server.AuthType)
	if provider == "" {
		return nil, &UserError{Message: "Invalid OAuth provider", Code: UserErrorServerConfigInvalid}
	}

	userClientID := strings.TrimSpace(toString(oauthCfg["userClientId"]))
	if clientID == userClientID {
		keyLen := int(math.Ceil(float64(len(userToken)) * 0.5))
		seed := userToken[keyLen:] + serverID + "true"
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(seed)))

		requestBody := map[string]any{
			"clientId":    clientID,
			"provider":    provider,
			"key":         hash,
			"code":        oauthCode,
			"redirectUri": redirectURI,
		}
		if oauthCodeVerifier != "" {
			requestBody["codeVerifier"] = oauthCodeVerifier
		}

		if server.AuthType == coretypes.ServerAuthTypeZendeskAuth {
			oauthScope := strings.TrimSpace(toString(oauthCfg["scope"]))
			if oauthScope != "" {
				requestBody["scope"] = oauthScope
			}
		}

		tokenURL := strings.TrimSpace(toString(oauthCfg["tokenUrl"]))
		if provider == "zendesk" || provider == "canvas" {
			if tokenURL == "" {
				return nil, &UserError{Message: "Missing OAuth tokenUrl", Code: UserErrorServerConfigInvalid}
			}
			requestBody["tokenUrl"] = tokenURL
		}

		res, err := utils.PostJSON(config.GetKimbapAuthConfig().BaseURL+"/v1/oauth/exchange", requestBody)
		if err != nil {
			return nil, &UserError{Message: "Failed to exchange OAuth code", Code: UserErrorServerConfigInvalid}
		}
		accessToken := strings.TrimSpace(toString(res["accessToken"]))
		expiresAt := utils.ValueAsInt64(res["expiresAt"])
		if accessToken == "" || expiresAt == 0 {
			return nil, &UserError{Message: "Failed to exchange OAuth code", Code: UserErrorServerConfigInvalid}
		}

		launchConfig["oauth"] = map[string]any{
			"clientId":    clientID,
			"key":         hash,
			"accessToken": accessToken,
			"expiresAt":   expiresAt,
		}
		return launchConfig, nil
	}

	oauthScope := ""
	if server.AuthType == coretypes.ServerAuthTypeZendeskAuth {
		oauthScope = strings.TrimSpace(toString(oauthCfg["scope"]))
	}

	exchangeResult, err := mcpoauth.ExchangeAuthorizationCode(mcpoauth.ExchangeContext{
		Provider:     provider,
		TokenURL:     strings.TrimSpace(toString(oauthCfg["tokenUrl"])),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Code:         oauthCode,
		RedirectURI:  redirectURI,
		CodeVerifier: oauthCodeVerifier,
		Scope:        oauthScope,
	})
	if err != nil || exchangeResult == nil || strings.TrimSpace(exchangeResult.AccessToken) == "" || strings.TrimSpace(exchangeResult.RefreshToken) == "" {
		return nil, &UserError{Message: "Failed to exchange OAuth code", Code: UserErrorServerConfigInvalid}
	}

	expiresAt := int64(0)
	if exchangeResult.ExpiresAt != nil {
		expiresAt = *exchangeResult.ExpiresAt
	}
	if expiresAt == 0 {
		expiresAt = time.Now().Add(30 * 24 * time.Hour).UnixMilli()
	}

	oauthData := map[string]any{
		"clientId":     clientID,
		"clientSecret": clientSecret,
		"accessToken":  exchangeResult.AccessToken,
		"refreshToken": exchangeResult.RefreshToken,
		"expiresAt":    expiresAt,
	}
	if server.AuthType == coretypes.ServerAuthTypeZendeskAuth || server.AuthType == coretypes.ServerAuthTypeCanvasAuth {
		oauthData["tokenUrl"] = strings.TrimSpace(toString(oauthCfg["tokenUrl"]))
	}
	if server.AuthType == coretypes.ServerAuthTypeZendeskAuth {
		oauthData["scope"] = strings.TrimSpace(toString(oauthCfg["scope"]))
	}
	if server.AuthType == coretypes.ServerAuthTypeZendeskAuth {
		if rtei, ok := exchangeResult.Raw["refresh_token_expires_in"]; ok {
			if parsed, ok := mcpoauth.NumberToInt64(rtei); ok {
				oauthData["refreshTokenExpiresAt"] = time.Now().UnixMilli() + parsed*1000
			}
		}
	}
	launchConfig["oauth"] = oauthData

	return launchConfig, nil
}

func (h *RequestHandler) UnconfigureServer(userID string, data map[string]any) (map[string]any, error) {
	serverID, _ := data["serverId"].(string)
	if serverID == "" {
		return nil, &UserError{Message: "serverId is required", Code: UserErrorInvalidRequest}
	}
	_, _ = core.ServerManagerInstance().CloseTemporaryServer(context.Background(), serverID, userID)

	var launchConfigsJSON, prefsJSON string
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var freshUser database.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&freshUser).Error; err != nil {
			return err
		}
		launchConfigs, err := parseJSONField(freshUser.LaunchConfigs, "launch configs")
		if err != nil {
			return err
		}
		if _, exists := launchConfigs[serverID]; !exists {
			return nil
		}
		delete(launchConfigs, serverID)
		launchConfigsJSON = string(mustJSON(launchConfigs))

		prefs, err := parseJSONField(freshUser.UserPreferences, "user preferences")
		if err != nil {
			return err
		}
		delete(prefs, serverID)
		prefsJSON = string(mustJSON(prefs))

		return tx.Model(&database.User{}).Where("user_id = ?", userID).Updates(map[string]any{"launch_configs": launchConfigsJSON, "user_preferences": prefsJSON, "updated_at": int(time.Now().Unix())}).Error
	}); err != nil {
		return nil, err
	}

	core.ServerManagerInstance().NotifyUserPermissionChangedByServer(serverID)
	if launchConfigsJSON != "" {
		syncLaunchConfigsToSessions(userID, launchConfigsJSON)
	}
	if prefsJSON != "" {
		var prefsMap map[string]any
		_ = json.Unmarshal([]byte(prefsJSON), &prefsMap)
		syncUserPreferencesToSessions(userID, mustPermissionsFromAny(prefsMap))
	}

	return map[string]any{"serverId": serverID, "message": "Server unconfigured successfully"}, nil
}

func (h *RequestHandler) GetOnlineSessions(userID string) ([]map[string]any, error) {
	sessions := core.SessionStoreInstance().GetUserSessions(userID)
	out := make([]map[string]any, 0, len(sessions))

	for _, session := range sessions {
		if session == nil {
			continue
		}
		authContext := session.AuthContextSnapshot()
		userAgent := strings.TrimSpace(authContext.UserAgent)
		if userAgent == "" {
			userAgent = "Unknown"
		}
		clientName := "Unknown Client"
		if info := session.GetClientInfo(); info != nil && info.Name != "" {
			clientName = info.Name
		}
		out = append(out, map[string]any{
			"sessionId":  session.SessionID,
			"clientName": clientName,
			"userAgent":  userAgent,
			"lastActive": session.LastActiveSnapshot(),
		})
	}

	return out, nil
}

func (h *RequestHandler) getUser(userID string) (*database.User, error) {
	var user database.User
	if err := h.db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &UserError{Message: "User not found", Code: UserErrorInternal}
		}
		return nil, err
	}
	return &user, nil
}

func isServerEnabledForUser(user *database.User, server database.Server) bool {
	enabled := server.PublicAccess
	if strings.TrimSpace(user.Permissions) == "" {
		return enabled
	}
	permissions := mcptypes.Permissions{}
	if err := json.Unmarshal([]byte(user.Permissions), &permissions); err != nil {
		return false
	}
	if permissions == nil {
		return false
	}
	if permission, ok := permissions[server.ServerID]; ok {
		return permission.Enabled
	}
	return enabled
}

func buildLaunchConfig(template map[string]any, authConf any) (map[string]any, error) {
	mcpConf := map[string]any{}
	raw, ok := template["mcpJsonConf"]
	if !ok {
		return nil, &UserError{Message: "configTemplate must contain mcpJsonConf field", Code: UserErrorServerConfigInvalid}
	}
	b, _ := json.Marshal(raw)
	if err := json.Unmarshal(b, &mcpConf); err != nil {
		return nil, &UserError{Message: "Invalid mcpJsonConf", Code: UserErrorServerConfigInvalid}
	}
	if mcpConf == nil {
		return nil, &UserError{Message: "Invalid mcpJsonConf", Code: UserErrorServerConfigInvalid}
	}
	authItems, err := parseAuthConfItems(authConf)
	if err != nil {
		return nil, &UserError{Message: "invalid server configuration", Code: UserErrorServerConfigInvalid}
	}
	replacements := map[string]string{}
	for _, authItem := range authItems {
		if authItem.DataType != 1 {
			continue
		}
		replacements[authItem.Key] = authItem.Value
	}
	replaced := replaceConfigPlaceholders(mcpConf, replacements)
	b2, _ := json.Marshal(replaced)
	out := map[string]any{}
	if err := json.Unmarshal(b2, &out); err != nil {
		return nil, &UserError{Message: "Configuration became invalid after credential replacement", Code: UserErrorServerConfigInvalid}
	}
	if out == nil {
		return nil, &UserError{Message: "Invalid mcpJsonConf", Code: UserErrorServerConfigInvalid}
	}
	applyOAuthExpirationDefaults(template, out)
	return out, nil
}

func replaceConfigPlaceholders(value any, replacements map[string]string) any {
	if len(replacements) == 0 {
		return value
	}
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, val := range v {
			out[key] = replaceConfigPlaceholders(val, replacements)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = replaceConfigPlaceholders(val, replacements)
		}
		return out
	case string:
		if replacement, ok := replacements[v]; ok {
			return replacement
		}
		return v
	default:
		return v
	}
}

func parseJSONField(raw string, fieldName string) (map[string]any, error) {
	result := map[string]any{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			return nil, &UserError{Message: "Failed to parse stored " + fieldName, Code: UserErrorInternal}
		}
		if result == nil {
			result = map[string]any{}
		}
	}
	return result, nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func toPermissions(v any) (mcptypes.Permissions, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := mcptypes.Permissions{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func mustPermissionsFromAny(v any) mcptypes.Permissions {
	p, err := toPermissions(v)
	if err != nil {
		return mcptypes.Permissions{}
	}
	return p
}

func extractEnabledFieldDeltas(submitted map[string]any, current mcptypes.Permissions) map[string]any {
	deltas := map[string]any{}
	for serverID, currentServer := range current {
		rawServer, ok := submitted[serverID]
		if !ok {
			continue
		}
		submittedServer, ok := rawServer.(map[string]any)
		if !ok {
			continue
		}
		serverDelta := map[string]any{}
		if enabled, ok := submittedServer["enabled"].(bool); ok {
			serverDelta["enabled"] = enabled
		}
		submittedTools, _ := submittedServer["tools"].(map[string]any)
		submittedResources, _ := submittedServer["resources"].(map[string]any)
		submittedPrompts, _ := submittedServer["prompts"].(map[string]any)
		if len(submittedTools) > 0 {
			toolDeltas := map[string]any{}
			for toolName := range currentServer.Tools {
				if rawTool, exists := submittedTools[toolName]; exists {
					toolDeltas[toolName] = rawTool
				}
			}
			if len(toolDeltas) > 0 {
				serverDelta["tools"] = toolDeltas
			}
		}
		if len(submittedResources) > 0 {
			resDeltas := map[string]any{}
			for resName := range currentServer.Resources {
				if rawRes, exists := submittedResources[resName]; exists {
					resDeltas[resName] = rawRes
				}
			}
			if len(resDeltas) > 0 {
				serverDelta["resources"] = resDeltas
			}
		}
		if len(submittedPrompts) > 0 {
			promptDeltas := map[string]any{}
			for promptName := range currentServer.Prompts {
				if rawPrompt, exists := submittedPrompts[promptName]; exists {
					promptDeltas[promptName] = rawPrompt
				}
			}
			if len(promptDeltas) > 0 {
				serverDelta["prompts"] = promptDeltas
			}
		}
		if len(serverDelta) > 0 {
			deltas[serverID] = serverDelta
		}
	}
	return deltas
}

func extractEnabledFields(submitted map[string]any, current mcptypes.Permissions) mcptypes.Permissions {
	merged := clonePermissions(current)

	for serverID, currentServer := range current {
		rawServer, ok := submitted[serverID]
		if !ok {
			continue
		}
		submittedServer, ok := rawServer.(map[string]any)
		if !ok {
			continue
		}

		server := merged[serverID]
		if enabled, ok := submittedServer["enabled"].(bool); ok {
			server.Enabled = enabled
		}

		submittedTools, _ := submittedServer["tools"].(map[string]any)
		submittedResources, _ := submittedServer["resources"].(map[string]any)
		submittedPrompts, _ := submittedServer["prompts"].(map[string]any)

		for toolName := range currentServer.Tools {
			rawTool, exists := submittedTools[toolName]
			if !exists {
				continue
			}
			submittedTool, ok := rawTool.(map[string]any)
			if !ok {
				continue
			}
			tool := server.Tools[toolName]
			if enabled, ok := submittedTool["enabled"].(bool); ok {
				tool.Enabled = enabled
			}
			if rawDL, ok := submittedTool["dangerLevel"]; ok {
				dl, ok := toInt(rawDL)
				if ok && dl >= coretypes.DangerLevelSilent && dl <= coretypes.DangerLevelApproval {
					tool.DangerLevel = &dl
				}
			}
			server.Tools[toolName] = tool
		}

		for resourceName, currentResource := range currentServer.Resources {
			rawResource, exists := submittedResources[resourceName]
			if !exists {
				continue
			}
			submittedResource, ok := rawResource.(map[string]any)
			if !ok {
				continue
			}
			resource := currentResource
			if enabled, ok := submittedResource["enabled"].(bool); ok {
				resource.Enabled = enabled
			}
			server.Resources[resourceName] = resource
		}

		for promptName, currentPrompt := range currentServer.Prompts {
			rawPrompt, exists := submittedPrompts[promptName]
			if !exists {
				continue
			}
			submittedPrompt, ok := rawPrompt.(map[string]any)
			if !ok {
				continue
			}
			prompt := currentPrompt
			if enabled, ok := submittedPrompt["enabled"].(bool); ok {
				prompt.Enabled = enabled
			}
			server.Prompts[promptName] = prompt
		}

		merged[serverID] = server
	}

	return merged
}

func clonePermissions(in mcptypes.Permissions) mcptypes.Permissions {
	out := make(mcptypes.Permissions, len(in))
	for serverID, cfg := range in {
		copied := cfg

		copied.Tools = make(map[string]mcptypes.ToolCapabilityConfig, len(cfg.Tools))
		for name, tool := range cfg.Tools {
			toolCopy := tool
			if tool.DangerLevel != nil {
				danger := *tool.DangerLevel
				toolCopy.DangerLevel = &danger
			}
			if tool.Metadata != nil {
				metadata := make(map[string]any, len(tool.Metadata))
				for k, v := range tool.Metadata {
					metadata[k] = v
				}
				toolCopy.Metadata = metadata
			}
			copied.Tools[name] = toolCopy
		}

		copied.Resources = make(map[string]mcptypes.ResourceCapabilityConfig, len(cfg.Resources))
		for name, resource := range cfg.Resources {
			copied.Resources[name] = resource
		}

		copied.Prompts = make(map[string]mcptypes.PromptCapabilityConfig, len(cfg.Prompts))
		for name, prompt := range cfg.Prompts {
			copied.Prompts[name] = prompt
		}

		out[serverID] = copied
	}
	return out
}

func syncUserPreferencesToSessions(userID string, prefs mcptypes.Permissions) {
	core.SessionStoreInstance().UpdateUserPreferences(userID, prefs, mcpservice.CapabilitiesServiceInstance())
}

func syncLaunchConfigsToSessions(userID, launchConfigs string) {
	for _, session := range core.SessionStoreInstance().GetUserSessions(userID) {
		if session == nil {
			continue
		}
		session.UpdateLaunchConfigs(launchConfigs)
	}
}

func buildServerPreference(serverContext *core.ServerContext) map[string]any {
	if serverContext == nil {
		return map[string]any{
			"enabled":        true,
			"serverName":     "",
			"allowUserInput": false,
			"authType":       0,
			"category":       nil,
			"configTemplate": "",
			"configured":     true,
			"tools":          map[string]any{},
			"resources":      map[string]any{},
			"prompts":        map[string]any{},
		}
	}

	capabilities := serverContext.GetMCPCapabilities()

	return map[string]any{
		"enabled":        capabilities.Enabled,
		"serverName":     capabilities.ServerName,
		"allowUserInput": capabilities.AllowUserInput,
		"authType":       capabilities.AuthType,
		"category":       capabilities.Category,
		"configTemplate": capabilities.ConfigTemplate,
		"configured":     capabilities.Configured,
		"tools":          capabilities.Tools,
		"resources":      capabilities.Resources,
		"prompts":        capabilities.Prompts,
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

type authConfItem struct {
	Key      string
	Value    string
	DataType int
}

type authConfMapItem struct {
	Value    string `json:"value"`
	DataType int    `json:"dataType"`
}

func parseAuthConfItems(authConf any) ([]authConfItem, error) {
	arr, ok := authConf.([]any)
	if !ok || len(arr) == 0 {
		return nil, errors.New("authConf is required and cannot be empty")
	}

	items := make([]authConfItem, 0, len(arr))
	for _, item := range arr {
		auth, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("Invalid auth config item")
		}

		keyRaw, keyExists := auth["key"]
		key, keyIsString := keyRaw.(string)
		if !keyExists || !keyIsString || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("Invalid auth.key: %v", keyRaw)
		}

		valueRaw, valueExists := auth["value"]
		value, valueIsString := valueRaw.(string)
		if !valueExists || !valueIsString {
			return nil, fmt.Errorf("Invalid auth.value for key: %s", key)
		}

		dataTypeRaw, dataTypeExists := auth["dataType"]
		dataType, dataTypeIsNumber := toInt(dataTypeRaw)
		if !dataTypeExists || !dataTypeIsNumber {
			return nil, fmt.Errorf("Invalid auth.dataType for key: %s", key)
		}

		items = append(items, authConfItem{Key: key, Value: value, DataType: dataType})
	}

	return items, nil
}

func authConfToMap(authConf any) (map[string]authConfMapItem, error) {
	items, err := parseAuthConfItems(authConf)
	if err != nil {
		return nil, err
	}

	out := make(map[string]authConfMapItem, len(items))
	for _, item := range items {
		out[item.Key] = authConfMapItem{Value: item.Value, DataType: item.DataType}
	}

	return out, nil
}

func encodeURLParams(v any) string {
	params, ok := v.(map[string]any)
	if !ok || len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for key, value := range params {
		if key == "" || value == nil {
			continue
		}
		values.Set(key, toString(value))
	}

	return values.Encode()
}

func isEmptyRestfulAPIAuth(v any) bool {
	if v == nil {
		return true
	}
	if m, ok := v.(map[string]any); ok {
		if len(m) == 0 {
			return true
		}
		if size, hasSize := toInt(m["size"]); hasSize {
			return size == 0
		}
		for key, value := range m {
			if key == "size" {
				continue
			}
			if value == nil {
				continue
			}
			strValue, ok := value.(string)
			if !ok {
				return false
			}
			if strings.TrimSpace(strValue) != "" {
				return false
			}
		}
		return true
	}
	return true
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	if string(b) == "null" {
		return ""
	}
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		return string(b[1 : len(b)-1])
	}
	return string(b)
}

func applyOAuthExpirationDefaults(template map[string]any, launchConfig map[string]any) {
	authType, ok := toInt(template["authType"])
	if !ok {
		return
	}
	oauthCfg, _ := launchConfig["oauth"].(map[string]any)
	if oauthCfg == nil {
		oauthCfg = map[string]any{}
	}
	if expiresAt := utils.ValueAsInt64(oauthCfg["expiresAt"]); expiresAt > 0 {
		launchConfig["oauth"] = oauthCfg
		return
	}
	now := time.Now()
	switch authType {
	case coretypes.ServerAuthTypeNotionAuth:
		oauthCfg["expiresAt"] = now.Add(30 * 24 * time.Hour).UnixMilli()
	case coretypes.ServerAuthTypeFigmaAuth:
		oauthCfg["expiresAt"] = now.Add(90 * 24 * time.Hour).UnixMilli()
	default:
		return
	}
	launchConfig["oauth"] = oauthCfg
}

func (h *RequestHandler) rollbackLaunchConfig(userID, serverID string) error {
	var lcJSON string
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var freshUser database.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&freshUser).Error; err != nil {
			return err
		}
		lc, err := parseJSONField(freshUser.LaunchConfigs, "launch configs")
		if err != nil {
			return err
		}
		delete(lc, serverID)
		lcJSON = string(mustJSON(lc))
		return tx.Model(&database.User{}).Where("user_id = ?", userID).Updates(map[string]any{"launch_configs": lcJSON, "updated_at": int(time.Now().Unix())}).Error
	}); err != nil {
		return err
	}
	syncLaunchConfigsToSessions(userID, lcJSON)
	return nil
}

func isDangerousEnvKey(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	for _, prefix := range []string{"LD_", "DYLD_", "PYTHON", "NODE_OPTIONS", "RUBYOPT", "BASH_ENV", "PERL", "CLASSPATH", "JAVA_TOOL_OPTIONS", "_JAVA_OPTIONS", "COMP_", "ENV", "BASH_FUNC_"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	for _, exact := range []string{"PATH", "HOME", "USER", "SHELL", "IFS", "CDPATH", "GLOBIGNORE", "SHELLOPTS", "BASHOPTS", "PS1", "PS2", "PS4", "PROMPT_COMMAND"} {
		if upper == exact {
			return true
		}
	}
	return false
}

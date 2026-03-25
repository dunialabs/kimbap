package handlers

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcpservice "github.com/dunialabs/kimbap-core/internal/mcp/service"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/security"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

type UserHandler struct {
	db             *gorm.DB
	sessionStore   *core.SessionStore
	socketNotifier core.SocketNotifier
	serverManager  userRuntimeManager
}

type userRuntimeManager interface {
	NotifyUserPermissionChanged(userID string)
}

func NewUserHandler(db *gorm.DB, sessionStore *core.SessionStore, socketNotifier core.SocketNotifier, serverManager userRuntimeManager) *UserHandler {
	if db == nil {
		db = database.DB
	}
	if sessionStore == nil {
		sessionStore = core.SessionStoreInstance()
	}
	if socketNotifier == nil {
		socketNotifier = core.NewNoopSocketNotifier()
	}
	if serverManager == nil {
		serverManager = core.ServerManagerInstance()
	}
	return &UserHandler{db: db, sessionStore: sessionStore, socketNotifier: socketNotifier, serverManager: serverManager}
}

func (h *UserHandler) DisableUser(data map[string]any) (any, error) {
	targetID, _ := data["targetId"].(string)
	if targetID == "" {
		return nil, &types.AdminError{Message: "missing targetId", Code: types.AdminErrorCodeInvalidRequest}
	}
	result := h.db.Model(&database.User{}).Where("user_id = ?", targetID).Update("status", types.UserStatusDisabled)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, &types.AdminError{Message: "user not found", Code: types.AdminErrorCodeUserNotFound}
	}
	if h.socketNotifier != nil {
		h.socketNotifier.NotifyUserDisabled(targetID, "")
	}
	h.sessionStore.RemoveAllUserSessions(targetID, mcptypes.DisconnectReasonUserDisabled)
	return nil, nil
}

func (h *UserHandler) UpdateUserPermissions(data map[string]any) (any, error) {
	targetID, _ := data["targetId"].(string)
	if targetID == "" {
		return nil, &types.AdminError{Message: "missing targetId", Code: types.AdminErrorCodeInvalidRequest}
	}
	var user database.User
	err := h.db.Select("user_id", "permissions").Where("user_id = ?", targetID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "user not found", Code: types.AdminErrorCodeUserNotFound}
		}
		return nil, err
	}

	permissions, ok := data["permissions"]
	if !ok || permissions == nil {
		return nil, &types.AdminError{Message: "invalid permissions", Code: types.AdminErrorCodeInvalidRequest}
	}
	permString, updatedPermissions, normalizeErr := normalizePermissionsJSON(permissions)
	if normalizeErr != nil {
		return nil, &types.AdminError{Message: "invalid permissions", Code: types.AdminErrorCodeInvalidRequest}
	}
	if permString == user.Permissions {
		return nil, nil
	}
	oldPermissions := parsePermissionsJSON(user.Permissions)
	toolsChanged, resourcesChanged, promptsChanged := mcpservice.CapabilitiesServiceInstance().ComparePermissions(oldPermissions, updatedPermissions)

	result := h.db.Model(&database.User{}).Where("user_id = ?", targetID).Update("permissions", permString)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, &types.AdminError{Message: "user not found", Code: types.AdminErrorCodeUserNotFound}
	}
	for _, session := range h.sessionStore.GetUserSessions(targetID) {
		if session == nil {
			continue
		}
		session.UpdatePermissions(updatedPermissions)
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
	if h.socketNotifier != nil && (toolsChanged || resourcesChanged || promptsChanged) {
		h.socketNotifier.NotifyUserPermissionChanged(targetID)
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminUserEdit,
		RequestParams: toJSONString(map[string]any{
			"userId":      targetID,
			"permissions": permissions,
		}, "{}"),
	})
	return nil, nil
}

func (h *UserHandler) CreateUser(data map[string]any, adminToken string) (any, error) {
	userID := toString(data["userId"])
	if userID == "" {
		return nil, &types.AdminError{Message: "missing userId", Code: types.AdminErrorCodeInvalidRequest}
	}
	encryptedToken := toString(data["encryptedToken"])
	if encryptedToken == "" {
		return nil, &types.AdminError{Message: "missing encryptedToken", Code: types.AdminErrorCodeInvalidRequest}
	}

	role := toInt(data["role"], types.UserRoleUser)
	if !isValidUserRole(role) {
		return nil, &types.AdminError{Message: "invalid role", Code: types.AdminErrorCodeInvalidRequest}
	}
	if role == types.UserRoleOwner {
		var ownerCount int64
		if err := h.db.Model(&database.User{}).Where("role = ?", types.UserRoleOwner).Count(&ownerCount).Error; err != nil {
			return nil, err
		}
		if ownerCount > 0 {
			return nil, &types.AdminError{Message: "There can only be one owner", Code: types.AdminErrorCodeUserAlreadyExists}
		}
		var totalCount int64
		if err := h.db.Model(&database.User{}).Count(&totalCount).Error; err != nil {
			return nil, err
		}
		if totalCount > 0 {
			return nil, &types.AdminError{Message: "The owner must be the first user created", Code: types.AdminErrorCodeInvalidRequest}
		}
		if adminToken == "" {
			return nil, &types.AdminError{Message: "admin token required", Code: types.AdminErrorCodeForbidden}
		}
		decrypted, err := security.DecryptDataFromString(encryptedToken, adminToken)
		if err != nil {
			return nil, &types.AdminError{Message: "invalid token: decryption failed", Code: types.AdminErrorCodeInvalidRequest}
		}
		if security.CalculateUserID(decrypted) != userID {
			return nil, &types.AdminError{Message: "invalid token", Code: types.AdminErrorCodeInvalidRequest}
		}
	}

	var existing database.User
	err := h.db.Where("user_id = ?", userID).First(&existing).Error
	if err == nil {
		return nil, &types.AdminError{Message: "user already exists", Code: types.AdminErrorCodeUserAlreadyExists}
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if role != types.UserRoleOwner {
		if adminToken == "" {
			return nil, &types.AdminError{Message: "admin token required", Code: types.AdminErrorCodeForbidden}
		}
		adminUserID := security.CalculateUserID(adminToken)
		var adminUser database.User
		if err := h.db.Select("user_id", "role").Where("user_id = ?", adminUserID).First(&adminUser).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &types.AdminError{Message: "Only Owner and Admin role can perform admin operations.", Code: types.AdminErrorCodeForbidden}
			}
			return nil, err
		}
		if adminUser.Role != types.UserRoleOwner && adminUser.Role != types.UserRoleAdmin {
			return nil, &types.AdminError{Message: "Only Owner and Admin role can perform admin operations.", Code: types.AdminErrorCodeForbidden}
		}
		decrypted, err := security.DecryptDataFromString(encryptedToken, adminToken)
		if err != nil {
			return nil, &types.AdminError{Message: "invalid token: decryption failed", Code: types.AdminErrorCodeInvalidRequest}
		}
		if security.CalculateUserID(decrypted) != userID {
			return nil, &types.AdminError{Message: "invalid token", Code: types.AdminErrorCodeInvalidRequest}
		}
	}

	if rawExpiresAt, ok := data["expiresAt"]; ok && rawExpiresAt != nil {
		if _, valid := asIntMaybe(rawExpiresAt); !valid {
			return nil, &types.AdminError{Message: "Invalid expiresAt", Code: types.AdminErrorCodeInvalidRequest}
		}
	}

	permissions := "{}"
	if rawPermissions, ok := data["permissions"]; ok {
		normalizedPermissions, _, normalizeErr := normalizePermissionsJSON(rawPermissions)
		if normalizeErr != nil {
			return nil, &types.AdminError{Message: "invalid permissions", Code: types.AdminErrorCodeInvalidRequest}
		}
		permissions = normalizedPermissions
	}
	now := int(time.Now().Unix())
	status := toInt(data["status"], types.UserStatusEnabled)
	if !isValidUserStatus(status) {
		return nil, &types.AdminError{Message: "invalid status", Code: types.AdminErrorCodeInvalidRequest}
	}
	user := database.User{
		UserID:          userID,
		Status:          status,
		Role:            role,
		Permissions:     permissions,
		UserPreferences: "{}",
		LaunchConfigs:   "{}",
		ExpiresAt:       normalizeUnix(data["expiresAt"]),
		CreatedAt:       toInt(data["createdAt"], now),
		UpdatedAt:       toInt(data["updatedAt"], now),
		Ratelimit:       toInt(data["ratelimit"], 100),
		Name:            toString(data["name"]),
		ProxyID:         toInt(data["proxyId"], 0),
		Notes:           ptrIfNotEmpty(toString(data["notes"])),
		EncryptedToken:  &encryptedToken,
	}
	if err := h.db.Create(&user).Error; err != nil {
		return nil, err
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminUserCreate,
		RequestParams: toJSONString(map[string]any{
			"userId": userID,
		}, "{}"),
	})
	redacted := usersToResponse([]database.User{user})
	return map[string]any{"user": redacted[0]}, nil
}

var userListColumns = []string{
	"user_id", "status", "role", "permissions",
	"expires_at", "created_at", "updated_at",
	"ratelimit", "name", "encrypted_token",
	"proxy_id", "notes",
}

func (h *UserHandler) GetUsers(data map[string]any) (any, error) {
	if userID := toString(data["userId"]); userID != "" {
		var users []database.User
		if err := h.db.Model(&database.User{}).Where("user_id = ?", userID).Select(userListColumns).Find(&users).Error; err != nil {
			return nil, err
		}
		return map[string]any{"users": usersToResponse(users)}, nil
	}

	query := h.db.Model(&database.User{})
	if proxyRaw, ok := data["proxyId"]; ok {
		query = query.Where("proxy_id = ?", toInt(proxyRaw, 0))
	}
	if exRaw, ok := data["excludeRole"]; ok {
		query = query.Where("role <> ?", toInt(exRaw, 0))
	} else if roleRaw, ok := data["role"]; ok {
		query = query.Where("role = ?", toInt(roleRaw, 0))
	}
	var users []database.User
	if err := query.Select(userListColumns).Find(&users).Error; err != nil {
		return nil, err
	}
	return map[string]any{"users": usersToResponse(users)}, nil
}

func usersToResponse(users []database.User) []map[string]any {
	result := make([]map[string]any, 0, len(users))
	for _, user := range users {
		result = append(result, map[string]any{
			"userId":      user.UserID,
			"status":      user.Status,
			"role":        user.Role,
			"permissions": user.Permissions,
			"expiresAt":   user.ExpiresAt,
			"createdAt":   user.CreatedAt,
			"updatedAt":   user.UpdatedAt,
			"ratelimit":   user.Ratelimit,
			"name":        user.Name,
			"proxyId":     user.ProxyID,
			"notes":       user.Notes,
		})
	}
	return result
}

func (h *UserHandler) UpdateUser(data map[string]any) (any, error) {
	userID := toString(data["userId"])
	if userID == "" {
		return nil, &types.AdminError{Message: "missing userId", Code: types.AdminErrorCodeInvalidRequest}
	}
	var user database.User
	err := h.db.Select("user_id", "status").Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "user not found", Code: types.AdminErrorCodeUserNotFound}
		}
		return nil, err
	}

	updates := map[string]any{}
	permissionsProvided := false
	normalizedPermissions := ""
	if _, ok := data["name"]; ok {
		updates["name"] = toString(data["name"])
	}
	if _, ok := data["notes"]; ok {
		if data["notes"] == nil {
			updates["notes"] = nil
		} else {
			updates["notes"] = toString(data["notes"])
		}
	}
	if _, ok := data["permissions"]; ok {
		parsedPermissions, _, normalizeErr := normalizePermissionsJSON(data["permissions"])
		if normalizeErr != nil {
			return nil, &types.AdminError{Message: "invalid permissions", Code: types.AdminErrorCodeInvalidRequest}
		}
		permissionsProvided = true
		normalizedPermissions = parsedPermissions
	}
	if _, ok := data["status"]; ok {
		status := toInt(data["status"], types.UserStatusEnabled)
		if !isValidUserStatus(status) {
			return nil, &types.AdminError{Message: "invalid status", Code: types.AdminErrorCodeInvalidRequest}
		}
		updates["status"] = status
	}
	if _, ok := data["encryptedToken"]; ok {
		v := toString(data["encryptedToken"])
		updates["encrypted_token"] = &v
	}
	if len(updates) > 0 {
		updates["updated_at"] = int(time.Now().Unix())
	}
	if len(updates) == 0 && !permissionsProvided {
		var currentUser database.User
		if err := h.db.Where("user_id = ?", userID).First(&currentUser).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &types.AdminError{Message: "user not found", Code: types.AdminErrorCodeUserNotFound}
			}
			return nil, err
		}
		redacted := usersToResponse([]database.User{currentUser})
		return map[string]any{"user": redacted[0]}, nil
	}
	isDisabling := false
	if nextStatusRaw, ok := updates["status"]; ok {
		nextStatus, _ := nextStatusRaw.(int)
		isDisabling = user.Status != types.UserStatusDisabled && nextStatus == types.UserStatusDisabled
	}

	if len(updates) > 0 {
		result := h.db.Model(&database.User{}).Where("user_id = ?", userID).Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			return nil, &types.AdminError{Message: "user not found", Code: types.AdminErrorCodeUserNotFound}
		}
		if isDisabling {
			if h.socketNotifier != nil {
				h.socketNotifier.NotifyUserDisabled(userID, "")
			}
			h.sessionStore.RemoveAllUserSessions(userID, mcptypes.DisconnectReasonUserDisabled)
		}
		if _, tokenRotated := updates["encrypted_token"]; tokenRotated && !isDisabling {
			h.sessionStore.RemoveAllUserSessions(userID, mcptypes.DisconnectReasonTokenRevoked)
		}
	}
	if permissionsProvided {
		if _, err := h.UpdateUserPermissions(map[string]any{"targetId": userID, "permissions": normalizedPermissions}); err != nil {
			return nil, err
		}
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminUserEdit,
		RequestParams: toJSONString(map[string]any{
			"userId": userID,
		}, "{}"),
	})
	var updatedUser database.User
	if err := h.db.Where("user_id = ?", userID).First(&updatedUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "user not found", Code: types.AdminErrorCodeUserNotFound}
		}
		return nil, err
	}
	redacted := usersToResponse([]database.User{updatedUser})
	return map[string]any{"user": redacted[0]}, nil
}

func (h *UserHandler) DeleteUser(data map[string]any) (any, error) {
	userID := toString(data["userId"])
	if userID == "" {
		return nil, &types.AdminError{Message: "missing userId", Code: types.AdminErrorCodeInvalidRequest}
	}
	// Match TS: disableUser first (update status + disconnect with USER_DISABLED), then delete
	_, err := h.DisableUser(map[string]any{"targetId": userID})
	if err != nil {
		return nil, err
	}
	result := h.db.Where("user_id = ?", userID).Delete(&database.User{})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, &types.AdminError{Message: "user not found", Code: types.AdminErrorCodeUserNotFound}
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminUserDelete,
		RequestParams: toJSONString(map[string]any{
			"userId": userID,
		}, "{}"),
	})
	return map[string]any{"message": "User deleted successfully"}, nil
}

func (h *UserHandler) DeleteUsersByProxy(data map[string]any) (any, error) {
	proxyID, ok := asIntMaybe(data["proxyId"])
	if !ok {
		return nil, &types.AdminError{Message: "missing proxyId", Code: types.AdminErrorCodeInvalidRequest}
	}
	var users []database.User
	if err := h.db.Where("proxy_id = ?", proxyID).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		if _, err := h.DisableUser(map[string]any{"targetId": user.UserID}); err != nil {
			return nil, err
		}
	}
	result := h.db.Where("proxy_id = ?", proxyID).Delete(&database.User{})
	if result.Error != nil {
		return nil, result.Error
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminUserDelete,
		RequestParams: toJSONString(map[string]any{
			"proxyId": proxyID,
		}, "{}"),
	})
	return map[string]any{"deletedCount": result.RowsAffected}, nil
}

func (h *UserHandler) CountUsers(data map[string]any) (any, error) {
	query := h.db.Model(&database.User{})
	if exRaw, ok := data["excludeRole"]; ok {
		query = query.Where("role <> ?", toInt(exRaw, 0))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}
	return map[string]any{"count": count}, nil
}

func (h *UserHandler) GetOwner() (any, error) {
	var user database.User
	if err := h.db.Where("role = ?", types.UserRoleOwner).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "owner user not found", Code: types.AdminErrorCodeUserNotFound}
		}
		return nil, err
	}
	return map[string]any{"owner": map[string]any{
		"userId":      user.UserID,
		"status":      user.Status,
		"role":        user.Role,
		"permissions": user.Permissions,
		"expiresAt":   user.ExpiresAt,
		"createdAt":   user.CreatedAt,
		"updatedAt":   user.UpdatedAt,
		"ratelimit":   user.Ratelimit,
		"name":        user.Name,
		"proxyId":     user.ProxyID,
		"notes":       user.Notes,
	}}, nil
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toInt(v any, fallback int) int {
	if v == nil {
		return fallback
	}
	if i, ok := asIntMaybe(v); ok {
		return i
	}
	return fallback
}

func asIntMaybe(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	case int32:
		return int(x), true
	case string:
		i, err := strconv.Atoi(stringsTrim(x))
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func toJSONString(v any, fallback string) string {
	if v == nil {
		return fallback
	}
	if s, ok := v.(string); ok {
		trimmed := stringsTrim(s)
		if trimmed == "" || trimmed == "null" {
			return fallback
		}
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fallback
	}
	return string(b)
}

func normalizePermissionsJSON(v any) (string, mcptypes.Permissions, error) {
	if v == nil {
		return "", nil, errors.New("invalid permissions")
	}
	trimmed := ""
	if s, ok := v.(string); ok {
		trimmed = stringsTrim(s)
		if trimmed == "" || trimmed == "null" {
			return "", nil, errors.New("invalid permissions")
		}
	} else {
		payload, err := json.Marshal(v)
		if err != nil {
			return "", nil, errors.New("invalid permissions")
		}
		trimmed = stringsTrim(string(payload))
	}
	updatedPermissions := mcptypes.Permissions{}
	if err := json.Unmarshal([]byte(trimmed), &updatedPermissions); err != nil || updatedPermissions == nil {
		return "", nil, errors.New("invalid permissions")
	}
	for serverID, serverPerm := range updatedPermissions {
		if serverPerm.Tools == nil {
			serverPerm.Tools = map[string]mcptypes.ToolCapabilityConfig{}
		}
		if serverPerm.Resources == nil {
			serverPerm.Resources = map[string]mcptypes.ResourceCapabilityConfig{}
		}
		if serverPerm.Prompts == nil {
			serverPerm.Prompts = map[string]mcptypes.PromptCapabilityConfig{}
		}
		updatedPermissions[serverID] = serverPerm
	}
	normalizedPermissions, err := json.Marshal(updatedPermissions)
	if err != nil {
		return "", nil, err
	}
	return string(normalizedPermissions), updatedPermissions, nil
}

func normalizeUnix(v any) int {
	i, ok := asIntMaybe(v)
	if !ok {
		return 0
	}
	if i >= 1_000_000_000_000 {
		return i / 1000
	}
	return i
}

func isValidUserRole(role int) bool {
	switch role {
	case types.UserRoleOwner, types.UserRoleAdmin, types.UserRoleUser, types.UserRoleGuest:
		return true
	default:
		return false
	}
}

func isValidUserStatus(status int) bool {
	switch status {
	case types.UserStatusEnabled, types.UserStatusDisabled:
		return true
	default:
		return false
	}
}

func ptrIfNotEmpty(v string) *string {
	if stringsTrim(v) == "" {
		return nil
	}
	return &v
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}

func parsePermissionsJSON(raw string) mcptypes.Permissions {
	out := mcptypes.Permissions{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return mcptypes.Permissions{}
	}
	if out == nil {
		return mcptypes.Permissions{}
	}
	return out
}

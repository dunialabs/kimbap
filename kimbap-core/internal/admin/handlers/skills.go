package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	"github.com/dunialabs/kimbap-core/internal/service"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

var skillsLog = logger.CreateLogger("SkillsHandler")

// skillsServerReconnector is a narrow interface for reconnecting a skills server.
type skillsServerReconnector interface {
	ReconnectServer(ctx context.Context, server database.Server, token string) (*core.ServerContext, error)
	GetServerContext(serverID, userID string) *core.ServerContext
}

type SkillsHandler struct {
	skillsService   *service.SkillsService
	db              *gorm.DB
	serverReconnect skillsServerReconnector
}

func NewSkillsHandler(svc *service.SkillsService, db *gorm.DB, serverReconnect skillsServerReconnector) *SkillsHandler {
	if svc == nil {
		svc = service.NewSkillsService()
	}
	if db == nil {
		db = database.DB
	}
	if serverReconnect == nil {
		serverReconnect = core.ServerManagerInstance()
	}
	return &SkillsHandler{skillsService: svc, db: db, serverReconnect: serverReconnect}
}

func (h *SkillsHandler) ListSkills(data map[string]any) (any, error) {
	serverID, err := requireStringField(data, "serverId")
	if err != nil {
		return nil, err
	}

	skills, err := h.skillsService.ListSkills(serverID)
	if err != nil {
		return nil, &types.AdminError{Message: sanitizeAdminErr(err, mapSkillsErrorCode(err, "list")), Code: mapSkillsErrorCode(err, "list")}
	}

	return map[string]any{"skills": skills}, nil
}

func (h *SkillsHandler) UploadSkill(data map[string]any, token string) (any, error) {
	serverID, err := requireStringField(data, "serverId")
	if err != nil {
		return nil, err
	}
	zipBytes, err := extractBytes(data["data"])
	if err != nil {
		return nil, &types.AdminError{Message: "invalid data format: expected []byte, base64 string, or byte array", Code: types.AdminErrorCodeInvalidRequest}
	}
	uploaded, err := h.skillsService.UploadSkill(serverID, zipBytes)
	if err != nil {
		code := mapSkillsErrorCode(err, "upload")
		return nil, &types.AdminError{Message: sanitizeAdminErr(err, code), Code: code}
	}

	h.reloadSkillsServer(serverID, token)

	return map[string]any{
		"success":   true,
		"skillName": strings.Join(uploaded, ", "),
		"message":   fmt.Sprintf("Successfully uploaded %d skill(s)", len(uploaded)),
	}, nil
}

func (h *SkillsHandler) DeleteSkill(data map[string]any, token string) (any, error) {
	serverID, err := requireStringField(data, "serverId")
	if err != nil {
		return nil, err
	}
	skillName, err := requireStringField(data, "skillName")
	if err != nil {
		return nil, err
	}

	err = h.skillsService.DeleteSkill(serverID, skillName)
	if err != nil {
		code := mapSkillsErrorCode(err, "delete")
		return nil, &types.AdminError{Message: sanitizeAdminErr(err, code), Code: code}
	}

	h.reloadSkillsServer(serverID, token)

	return map[string]any{
		"success": true,
		"message": "Skill deleted successfully",
	}, nil
}

func (h *SkillsHandler) DeleteServerSkills(data map[string]any, token string) (any, error) {
	serverID, err := requireStringField(data, "serverId")
	if err != nil {
		return nil, err
	}

	err = h.skillsService.DeleteServerSkills(serverID)
	if err != nil {
		if errors.Is(err, service.ErrNoSkillsDirectoryFound) {
			return map[string]any{
				"success": true,
				"message": "No skills directory found",
			}, nil
		}
		code := mapSkillsErrorCode(err, "delete")
		return nil, &types.AdminError{Message: sanitizeAdminErr(err, code), Code: code}
	}

	h.reloadSkillsServer(serverID, token)

	return map[string]any{
		"success": true,
		"message": "Server skills deleted successfully",
	}, nil
}

// reloadSkillsServer reconnects the skills server so that newly uploaded/deleted
// skills are picked up immediately without requiring a manual restart.
func (h *SkillsHandler) reloadSkillsServer(serverID, token string) {
	if h.serverReconnect == nil || h.db == nil {
		return
	}

	var server database.Server
	if err := h.db.Where("server_id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			skillsLog.Debug().Str("serverId", serverID).Msg("Skip skills server reload: server not found")
		} else {
			skillsLog.Warn().Err(err).Str("serverId", serverID).Msg("Skip skills server reload: database error")
		}
		return
	}
	if !server.Enabled {
		skillsLog.Debug().Str("serverId", serverID).Msg("Skip skills server reload: server disabled")
		return
	}
	if server.AllowUserInput {
		skillsLog.Debug().Str("serverId", serverID).Msg("Skip skills server reload: template server")
		return
	}
	if h.serverReconnect.GetServerContext(serverID, "") == nil {
		skillsLog.Debug().Str("serverId", serverID).Msg("Skip skills server reload: server not currently running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := h.serverReconnect.ReconnectServer(ctx, server, token); err != nil {
		skillsLog.Warn().Err(err).Str("serverId", serverID).Msg("Failed to reload skills server")
		return
	}
	skillsLog.Info().Str("serverId", serverID).Msg("Skills server reloaded after skill change")
}

func requireStringField(data map[string]any, field string) (string, error) {
	v, ok := data[field]
	if !ok {
		return "", &types.AdminError{Message: fmt.Sprintf("Missing required field: %s", field), Code: types.AdminErrorCodeInvalidRequest}
	}
	name, ok := v.(string)
	if !ok || name == "" {
		return "", &types.AdminError{Message: fmt.Sprintf("Missing required field: %s", field), Code: types.AdminErrorCodeInvalidRequest}
	}
	return name, nil
}

func extractBytes(v any) ([]byte, error) {
	switch b := v.(type) {
	case []byte:
		return b, nil
	case string:
		decoded, err := base64.StdEncoding.DecodeString(b)
		if err != nil {
			return []byte(b), nil
		}
		return decoded, nil
	case []any:
		out := make([]byte, 0, len(b))
		for _, item := range b {
			if n, ok := asIntMaybe(item); ok {
				out = append(out, byte(n))
			}
		}
		return out, nil
	default:
		return nil, os.ErrInvalid
	}
}

func sanitizeAdminErr(err error, code int) string {
	if code >= 5000 {
		return "internal server error"
	}
	return err.Error()
}

func mapSkillsErrorCode(err error, operation string) int {
	message := strings.ToLower(err.Error())

	if strings.Contains(message, "invalid serverid") ||
		strings.Contains(message, "invalid skill name") ||
		strings.Contains(message, "name is required") ||
		strings.Contains(message, "invalid characters") {
		return types.AdminErrorCodeInvalidRequest
	}

	if strings.Contains(message, "not found") {
		return types.AdminErrorCodeSkillNotFound
	}

	if strings.Contains(message, "invalid zip") ||
		strings.Contains(message, "no valid skills") ||
		strings.Contains(message, "zip file exceeds") ||
		strings.Contains(message, "zip file contains too many") ||
		strings.Contains(message, "zip has too many") ||
		strings.Contains(message, "zip file uncompressed") ||
		strings.Contains(message, "zip uncompressed") ||
		strings.Contains(message, "zip bomb") ||
		strings.Contains(message, "path traversal") ||
		strings.Contains(message, "absolute path") {
		return types.AdminErrorCodeInvalidSkillFormat
	}

	if errors.Is(err, os.ErrPermission) ||
		strings.Contains(message, "permission") ||
		strings.Contains(message, "eacces") ||
		strings.Contains(message, "eperm") ||
		strings.Contains(message, "enoent") ||
		strings.Contains(message, "eio") {
		return types.AdminErrorCodeDatabaseOpFailed
	}

	switch operation {
	case "upload":
		return types.AdminErrorCodeSkillUploadFailed
	case "delete":
		return types.AdminErrorCodeSkillDeleteFailed
	default:
		return types.AdminErrorCodeDatabaseOpFailed
	}
}

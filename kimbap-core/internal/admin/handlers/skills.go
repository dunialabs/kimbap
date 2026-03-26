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

var servicesLog = logger.CreateLogger("ServicesHandler")

// servicesServerReconnector is a narrow interface for reconnecting a services server.
type servicesServerReconnector interface {
	ReconnectServer(ctx context.Context, server database.Server, token string) (*core.ServerContext, error)
	GetServerContext(serverID, userID string) *core.ServerContext
}

type ServicesHandler struct {
	servicesService *service.ServicesService
	db              *gorm.DB
	serverReconnect servicesServerReconnector
}

func NewServicesHandler(svc *service.ServicesService, db *gorm.DB, serverReconnect servicesServerReconnector) *ServicesHandler {
	if svc == nil {
		svc = service.NewServicesService()
	}
	if db == nil {
		db = database.DB
	}
	if serverReconnect == nil {
		serverReconnect = core.ServerManagerInstance()
	}
	return &ServicesHandler{servicesService: svc, db: db, serverReconnect: serverReconnect}
}

func (h *ServicesHandler) ListServices(data map[string]any) (any, error) {
	serverID, err := requireStringField(data, "serverId")
	if err != nil {
		return nil, err
	}

	skills, err := h.servicesService.ListServices(serverID)
	if err != nil {
		return nil, &types.AdminError{Message: sanitizeAdminErr(err, mapServicesErrorCode(err, "list")), Code: mapServicesErrorCode(err, "list")}
	}

	return map[string]any{"skills": skills}, nil
}

func (h *ServicesHandler) UploadService(data map[string]any, token string) (any, error) {
	serverID, err := requireStringField(data, "serverId")
	if err != nil {
		return nil, err
	}
	zipBytes, err := extractBytes(data["data"])
	if err != nil {
		return nil, &types.AdminError{Message: "invalid data format: expected []byte, base64 string, or byte array", Code: types.AdminErrorCodeInvalidRequest}
	}
	uploaded, err := h.servicesService.UploadService(serverID, zipBytes)
	if err != nil {
		code := mapServicesErrorCode(err, "upload")
		return nil, &types.AdminError{Message: sanitizeAdminErr(err, code), Code: code}
	}

	h.reloadServicesServer(serverID, token)

	return map[string]any{
		"success":     true,
		"serviceName": strings.Join(uploaded, ", "),
		"message":     fmt.Sprintf("Successfully uploaded %d service(s)", len(uploaded)),
	}, nil
}

func (h *ServicesHandler) DeleteService(data map[string]any, token string) (any, error) {
	serverID, err := requireStringField(data, "serverId")
	if err != nil {
		return nil, err
	}
	serviceName, err := requireStringField(data, "serviceName")
	if err != nil {
		return nil, err
	}

	err = h.servicesService.DeleteService(serverID, serviceName)
	if err != nil {
		code := mapServicesErrorCode(err, "delete")
		return nil, &types.AdminError{Message: sanitizeAdminErr(err, code), Code: code}
	}

	h.reloadServicesServer(serverID, token)

	return map[string]any{
		"success": true,
		"message": "Service deleted successfully",
	}, nil
}

func (h *ServicesHandler) DeleteServerServices(data map[string]any, token string) (any, error) {
	serverID, err := requireStringField(data, "serverId")
	if err != nil {
		return nil, err
	}

	err = h.servicesService.DeleteServerServices(serverID)
	if err != nil {
		if errors.Is(err, service.ErrNoServicesDirectoryFound) {
			return map[string]any{
				"success": true,
				"message": "No services directory found",
			}, nil
		}
		code := mapServicesErrorCode(err, "delete")
		return nil, &types.AdminError{Message: sanitizeAdminErr(err, code), Code: code}
	}

	h.reloadServicesServer(serverID, token)

	return map[string]any{
		"success": true,
		"message": "Server services deleted successfully",
	}, nil
}

// reloadServicesServer reconnects the skills server so that newly uploaded/deleted
// skills are picked up immediately without requiring a manual restart.
func (h *ServicesHandler) reloadServicesServer(serverID, token string) {
	if h.serverReconnect == nil || h.db == nil {
		return
	}

	var server database.Server
	if err := h.db.Where("server_id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			servicesLog.Debug().Str("serverId", serverID).Msg("Skip services server reload: server not found")
		} else {
			servicesLog.Warn().Err(err).Str("serverId", serverID).Msg("Skip services server reload: database error")
		}
		return
	}
	if !server.Enabled {
		servicesLog.Debug().Str("serverId", serverID).Msg("Skip services server reload: server disabled")
		return
	}
	if server.AllowUserInput {
		servicesLog.Debug().Str("serverId", serverID).Msg("Skip services server reload: template server")
		return
	}
	if h.serverReconnect.GetServerContext(serverID, "") == nil {
		servicesLog.Debug().Str("serverId", serverID).Msg("Skip services server reload: server not currently running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := h.serverReconnect.ReconnectServer(ctx, server, token); err != nil {
		servicesLog.Warn().Err(err).Str("serverId", serverID).Msg("Failed to reload services server")
		return
	}
	servicesLog.Info().Str("serverId", serverID).Msg("Services server reloaded after service change")
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

func mapServicesErrorCode(err error, operation string) int {
	message := strings.ToLower(err.Error())

	if strings.Contains(message, "invalid serverid") ||
		strings.Contains(message, "invalid service name") ||
		strings.Contains(message, "name is required") ||
		strings.Contains(message, "invalid characters") {
		return types.AdminErrorCodeInvalidRequest
	}

	if strings.Contains(message, "not found") {
		return types.AdminErrorCodeServiceNotFound
	}

	if strings.Contains(message, "invalid zip") ||
		strings.Contains(message, "no valid services") ||
		strings.Contains(message, "zip file exceeds") ||
		strings.Contains(message, "zip file contains too many") ||
		strings.Contains(message, "zip has too many") ||
		strings.Contains(message, "zip file uncompressed") ||
		strings.Contains(message, "zip uncompressed") ||
		strings.Contains(message, "zip bomb") ||
		strings.Contains(message, "path traversal") ||
		strings.Contains(message, "absolute path") {
		return types.AdminErrorCodeInvalidServiceFormat
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
		return types.AdminErrorCodeServiceUploadFailed
	case "delete":
		return types.AdminErrorCodeServiceDeleteFailed
	default:
		return types.AdminErrorCodeDatabaseOpFailed
	}
}

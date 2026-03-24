package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/security"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

var backupLog = logger.CreateLogger("BackupHandler")

type BackupHandler struct {
	db                 *gorm.DB
	ipWhitelistService *security.IPWhitelistService
}

type backupData struct {
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Tables    struct {
		Users       []database.User        `json:"users"`
		Servers     []database.Server      `json:"servers"`
		Proxies     []database.Proxy       `json:"proxies"`
		IPWhitelist []database.IPWhitelist `json:"ipWhitelist"`
	} `json:"tables"`
}

func NewBackupHandler(db *gorm.DB, ipWhitelistService *security.IPWhitelistService) *BackupHandler {
	if db == nil {
		db = database.DB
	}
	return &BackupHandler{db: db, ipWhitelistService: ipWhitelistService}
}

func (h *BackupHandler) BackupDatabase() (any, error) {
	var users []database.User
	var servers []database.Server
	var proxies []database.Proxy
	var whitelist []database.IPWhitelist
	if err := h.db.Find(&users).Error; err != nil {
		return nil, backupFailedError(err)
	}
	if err := h.db.Find(&servers).Error; err != nil {
		return nil, backupFailedError(err)
	}
	if err := h.db.Find(&proxies).Error; err != nil {
		return nil, backupFailedError(err)
	}
	if err := h.db.Find(&whitelist).Error; err != nil {
		return nil, backupFailedError(err)
	}
	b := backupData{Version: "1.0", Timestamp: time.Now().Unix()}
	b.Tables.Users = users
	b.Tables.Servers = servers
	b.Tables.Proxies = proxies
	b.Tables.IPWhitelist = whitelist
	proxyID := 0
	if len(proxies) > 0 {
		proxyID = proxies[0].ID
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminBackupDatabase,
		RequestParams: toJSONString(map[string]any{
			"proxyId": proxyID,
		}, "{}"),
	})
	return map[string]any{
		"backup": b,
		"stats":  map[string]any{"usersCount": len(users), "serversCount": len(servers), "proxiesCount": len(proxies), "ipWhitelistCount": len(whitelist)},
	}, nil
}

func backupFailedError(_ error) error {
	return &types.AdminError{
		Message: "backup operation failed",
		Code:    types.AdminErrorCodeBackupFailed,
	}
}

func (h *BackupHandler) RestoreDatabase(data map[string]any, token string) (result any, err error) {
	backupRaw, ok := data["backup"].(map[string]any)
	if !ok {
		return nil, &types.AdminError{Message: "invalid backup data", Code: types.AdminErrorCodeInvalidRequest}
	}
	tables, ok := backupRaw["tables"].(map[string]any)
	if !ok {
		return nil, &types.AdminError{Message: "invalid backup tables", Code: types.AdminErrorCodeInvalidRequest}
	}
	users, err := decodeSlice[database.User](tables["users"])
	if err != nil {
		return nil, &types.AdminError{Message: fmt.Sprintf("invalid users backup data: %v", err), Code: types.AdminErrorCodeInvalidRequest}
	}
	servers, err := decodeSlice[database.Server](tables["servers"])
	if err != nil {
		return nil, &types.AdminError{Message: fmt.Sprintf("invalid servers backup data: %v", err), Code: types.AdminErrorCodeInvalidRequest}
	}
	proxies, err := decodeSlice[database.Proxy](tables["proxies"])
	if err != nil {
		return nil, &types.AdminError{Message: fmt.Sprintf("invalid proxies backup data: %v", err), Code: types.AdminErrorCodeInvalidRequest}
	}
	if len(proxies) > 1 {
		return nil, &types.AdminError{Message: "only one proxy is allowed", Code: types.AdminErrorCodeInvalidRequest}
	}
	whitelist, err := decodeSlice[database.IPWhitelist](tables["ipWhitelist"])
	if err != nil {
		return nil, &types.AdminError{Message: fmt.Sprintf("invalid ipWhitelist backup data: %v", err), Code: types.AdminErrorCodeInvalidRequest}
	}

	var existingProxy database.Proxy
	if err := h.db.Limit(1).Find(&existingProxy).Error; err != nil {
		return nil, &types.AdminError{Message: "backup restore failed", Code: types.AdminErrorCodeRestoreFailed}
	}
	if existingProxy.ID != 0 {
		return nil, &types.AdminError{Message: "proxy is not empty", Code: types.AdminErrorCodeInvalidRequest}
	}

	var usersCount int64
	if err := h.db.Model(&database.User{}).Count(&usersCount).Error; err != nil {
		return nil, &types.AdminError{Message: "backup restore failed", Code: types.AdminErrorCodeRestoreFailed}
	}
	if usersCount > 0 {
		return nil, &types.AdminError{Message: "users are not empty", Code: types.AdminErrorCodeInvalidRequest}
	}

	var serversCount int64
	if err := h.db.Model(&database.Server{}).Count(&serversCount).Error; err != nil {
		return nil, &types.AdminError{Message: "backup restore failed", Code: types.AdminErrorCodeRestoreFailed}
	}
	if serversCount > 0 {
		return nil, &types.AdminError{Message: "servers are not empty", Code: types.AdminErrorCodeInvalidRequest}
	}

	var whitelistCount int64
	if err := h.db.Model(&database.IPWhitelist{}).Count(&whitelistCount).Error; err != nil {
		return nil, &types.AdminError{Message: "backup restore failed", Code: types.AdminErrorCodeRestoreFailed}
	}
	if whitelistCount > 0 {
		return nil, &types.AdminError{Message: "ip whitelist is not empty", Code: types.AdminErrorCodeInvalidRequest}
	}

	ctx := context.Background()
	shutdownCompleted := false
	defer func() {
		if !shutdownCompleted || err == nil {
			return
		}
		recoveryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, _, reconnectErr := core.ServerManagerInstance().ConnectAllServers(recoveryCtx, token); reconnectErr != nil {
			backupLog.Warn().Err(reconnectErr).Msg("failed to reconnect servers during restore rollback")
		}
	}()
	core.SessionStoreInstance().RemoveAllSessions(mcptypes.DisconnectReasonServerShutdown)
	core.ServerManagerInstance().Shutdown(ctx)
	shutdownCompleted = true

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1=1").Delete(&database.User{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1=1").Delete(&database.Server{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1=1").Delete(&database.Proxy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1=1").Delete(&database.IPWhitelist{}).Error; err != nil {
			return err
		}
		if len(proxies) > 0 {
			if err := tx.Create(&proxies).Error; err != nil {
				return err
			}
		}
		if len(users) > 0 {
			if err := tx.Create(&users).Error; err != nil {
				return err
			}
		}
		if len(servers) > 0 {
			if err := tx.Create(&servers).Error; err != nil {
				return err
			}
		}
		if len(whitelist) > 0 {
			if err := tx.Create(&whitelist).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, &types.AdminError{Message: "backup restore failed", Code: types.AdminErrorCodeRestoreFailed}
	}
	if h.db.Dialector.Name() == "postgres" {
		if err := h.db.Exec("SELECT setval(pg_get_serial_sequence('proxy','id'), COALESCE((SELECT MAX(id) FROM proxy), 0), true)").Error; err != nil {
			return nil, &types.AdminError{Message: "backup restore failed", Code: types.AdminErrorCodeRestoreFailed}
		}
		if err := h.db.Exec("SELECT setval(pg_get_serial_sequence('ip_whitelist','id'), COALESCE((SELECT MAX(id) FROM ip_whitelist), 0), true)").Error; err != nil {
			return nil, &types.AdminError{Message: "backup restore failed", Code: types.AdminErrorCodeRestoreFailed}
		}
	}
	var ipWhitelistReloadErr error
	if h.ipWhitelistService != nil {
		if err := h.ipWhitelistService.LoadFromDB(); err != nil {
			ipWhitelistReloadErr = err
			backupLog.Warn().Err(err).Msg("IP whitelist reload failed after restore")
		}
	}

	connectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	successServers, failedServers, err := core.ServerManagerInstance().ConnectAllServers(connectCtx, token)
	if err != nil {
		return nil, &types.AdminError{Message: "backup restore failed", Code: types.AdminErrorCodeRestoreFailed}
	}
	proxyID := 0
	if len(proxies) > 0 {
		proxyID = proxies[0].ID
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminRestoreDatabase,
		RequestParams: toJSONString(map[string]any{
			"proxyId": proxyID,
		}, "{}"),
	})

	resultMap := map[string]any{
		"message": "Database restored successfully",
		"stats": map[string]any{
			"usersRestored":       len(users),
			"serversRestored":     len(servers),
			"proxiesRestored":     len(proxies),
			"ipWhitelistRestored": len(whitelist),
			"serversStarted":      len(successServers),
			"serversFailed":       len(failedServers),
		},
	}
	if ipWhitelistReloadErr != nil {
		resultMap["warnings"] = []string{"IP whitelist reload failed, will retry on next request"}
	}

	return resultMap, nil
}

func decodeSlice[T any](v any) ([]T, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out []T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

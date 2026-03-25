package handlers

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type ProxyHandler struct {
	db             *gorm.DB
	sessionStore   *core.SessionStore
	serverManager  proxyRuntimeManager
	socketNotifier core.SocketNotifier
}

type proxyRuntimeManager interface {
	Shutdown(ctx context.Context)
}

func NewProxyHandler(db *gorm.DB, sessionStore *core.SessionStore, serverManager proxyRuntimeManager, socketNotifier core.SocketNotifier) *ProxyHandler {
	if db == nil {
		db = database.DB
	}
	if sessionStore == nil {
		sessionStore = core.SessionStoreInstance()
	}
	if serverManager == nil {
		serverManager = core.ServerManagerInstance()
	}
	if socketNotifier == nil {
		socketNotifier = core.NewNoopSocketNotifier()
	}
	return &ProxyHandler{
		db:             db,
		sessionStore:   sessionStore,
		serverManager:  serverManager,
		socketNotifier: socketNotifier,
	}
}

func (h *ProxyHandler) GetProxy() (any, error) {
	var proxy database.Proxy
	err := h.db.Order("id asc").First(&proxy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]any{"proxy": nil}, nil
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"proxy": proxy}, nil
}

func (h *ProxyHandler) CreateProxy(data map[string]any) (any, error) {
	name := toString(data["name"])
	proxyKey := toString(data["proxyKey"])
	if name == "" {
		return nil, &types.AdminError{Message: "Missing required field: name", Code: types.AdminErrorCodeInvalidRequest}
	}
	if proxyKey == "" {
		return nil, &types.AdminError{Message: "Missing required field: proxyKey", Code: types.AdminErrorCodeInvalidRequest}
	}
	proxy := database.Proxy{
		Name:      name,
		ProxyKey:  proxyKey,
		StartPort: backendStartPort(),
		Addtime:   int(time.Now().Unix()),
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE proxy IN EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&database.Proxy{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return &types.AdminError{Message: "only one proxy is allowed", Code: types.AdminErrorCodeProxyAlreadyExists}
		}
		return tx.Create(&proxy).Error
	}); err != nil {
		return nil, err
	}
	if h.socketNotifier != nil {
		h.socketNotifier.UpdateServerInfo()
	}
	return map[string]any{"proxy": proxy}, nil
}

func (h *ProxyHandler) UpdateProxy(data map[string]any) (any, error) {
	proxyID, ok := asIntMaybe(data["proxyId"])
	if !ok {
		return nil, &types.AdminError{Message: "missing proxyId", Code: types.AdminErrorCodeInvalidRequest}
	}
	rawName, hasName := data["name"]
	if !hasName {
		return nil, &types.AdminError{Message: "missing name", Code: types.AdminErrorCodeInvalidRequest}
	}
	name := toString(rawName)
	res := h.db.Model(&database.Proxy{}).Where("id = ?", proxyID).Update("name", name)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, &types.AdminError{Message: "proxy not found", Code: types.AdminErrorCodeProxyNotFound}
	}
	var proxy database.Proxy
	if err := h.db.Where("id = ?", proxyID).First(&proxy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "proxy not found", Code: types.AdminErrorCodeProxyNotFound}
		}
		return nil, err
	}
	if h.socketNotifier != nil {
		h.socketNotifier.UpdateServerInfo()
	}
	return map[string]any{"proxy": proxy}, nil
}

func backendStartPort() int {
	raw := strings.TrimSpace(os.Getenv("BACKEND_PORT"))
	if raw == "" {
		return 3002
	}
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 3002
	}
	return port
}

func (h *ProxyHandler) DeleteProxy(data map[string]any) (any, error) {
	proxyID, ok := asIntMaybe(data["proxyId"])
	if !ok {
		return nil, &types.AdminError{Message: "missing proxyId", Code: types.AdminErrorCodeInvalidRequest}
	}

	var proxy database.Proxy
	if err := h.db.Where("id = ?", proxyID).First(&proxy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "proxy not found", Code: types.AdminErrorCodeProxyNotFound}
		}
		return nil, err
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ?", proxyID).Delete(&database.Proxy{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return &types.AdminError{Message: "proxy not found", Code: types.AdminErrorCodeProxyNotFound}
		}
		var userIDs []string
		if err := tx.Model(&database.User{}).Where("proxy_id = ?", proxyID).Pluck("user_id", &userIDs).Error; err != nil {
			return err
		}
		if len(userIDs) > 0 {
			if err := tx.Where("user_id IN ?", userIDs).Delete(&database.OAuthAuthorizationCode{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&database.OAuthClient{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&database.OAuthToken{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("proxy_id = ?", proxyID).Delete(&database.User{}).Error; err != nil {
			return err
		}
		if err := tx.Where("proxy_id = ?", proxyID).Delete(&database.Server{}).Error; err != nil {
			return err
		}
		if err := deleteByProxyIDIfColumnExists(tx, &database.License{}, proxyID); err != nil {
			return err
		}
		if err := deleteByProxyIDIfColumnExists(tx, &database.Event{}, proxyID); err != nil {
			return err
		}
		if err := deleteByProxyIDIfColumnExists(tx, &database.Log{}, proxyID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	h.sessionStore.RemoveAllSessions(mcptypes.DisconnectReasonServerShutdown)
	h.serverManager.Shutdown(context.Background())
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminProxyReset,
		RequestParams: toJSONString(map[string]any{
			"proxyId": proxyID,
		}, "{}"),
	})
	return map[string]any{"message": "Proxy deleted successfully"}, nil
}

func deleteByProxyIDIfColumnExists(tx *gorm.DB, model any, proxyID int) error {
	stmt := &gorm.Statement{DB: tx}
	if err := stmt.Parse(model); err != nil {
		return err
	}

	query := tx
	if stmt.Schema != nil && stmt.Schema.LookUpField("ProxyID") != nil {
		query = query.Where("proxy_id = ?", proxyID)
	} else {
		return nil
	}

	return query.Delete(model).Error
}

func (h *ProxyHandler) StopProxy() (any, error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("recovered panic during proxy shutdown, sending SIGTERM")
				_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
			}
		}()
		time.Sleep(100 * time.Millisecond)
		h.sessionStore.RemoveAllSessions(mcptypes.DisconnectReasonServerShutdown)
		shutdownDone := make(chan struct{})
		go func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer shutdownCancel()
			h.serverManager.Shutdown(shutdownCtx)
			close(shutdownDone)
		}()
		select {
		case <-shutdownDone:
		case <-time.After(3 * time.Second):
		}
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}()
	return map[string]any{"message": "Proxy shutdown initiated successfully"}, nil
}

package handlers

import (
	"errors"
	"net/url"

	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

type LogHandler struct {
	db *gorm.DB
}

func NewLogHandler(db *gorm.DB) *LogHandler {
	if db == nil {
		db = database.DB
	}
	return &LogHandler{db: db}
}

func (h *LogHandler) SetLogWebhookURL(data map[string]any) (any, error) {
	proxyKey := toString(data["proxyKey"])
	if proxyKey == "" {
		return nil, &types.AdminError{Message: "proxyKey is required", Code: types.AdminErrorCodeInvalidRequest}
	}
	if rawWebhookURL, exists := data["webhookUrl"]; exists && rawWebhookURL != nil {
		if _, ok := rawWebhookURL.(string); !ok {
			return nil, &types.AdminError{Message: "webhookUrl must be a string or null", Code: types.AdminErrorCodeInvalidRequest}
		}
	}
	webhookURL := toString(data["webhookUrl"])
	if webhookURL != "" {
		u, err := url.Parse(webhookURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, &types.AdminError{Message: "invalid webhookUrl format", Code: types.AdminErrorCodeInvalidRequest}
		}
	}
	var proxy database.Proxy
	if err := h.db.Where("proxy_key = ?", proxyKey).First(&proxy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "proxy not found", Code: types.AdminErrorCodeProxyNotFound}
		}
		return nil, err
	}
	if err := h.db.Model(&database.Proxy{}).Where("id = ?", proxy.ID).Update("log_webhook_url", ptrIfNotEmpty(webhookURL)).Error; err != nil {
		return nil, err
	}
	if err := internallog.GetLogSyncService().ReloadWebhookURL(); err != nil {
		return nil, err
	}
	message := "Log webhook URL cleared (sync disabled)"
	if webhookURL != "" {
		message = "Log webhook URL set successfully"
	}
	return map[string]any{"proxyId": proxy.ID, "proxyName": proxy.Name, "webhookUrl": ptrIfNotEmpty(webhookURL), "message": message}, nil
}

func (h *LogHandler) GetLogs(data map[string]any) (any, error) {
	startID := toInt(data["id"], 0)
	if startID < 0 {
		startID = 0
	}
	if startID == 0 {
		startID = 1
	}
	limit := toInt(data["limit"], 1000)
	if limit < 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}
	var logs []database.Log
	if err := h.db.Where("id >= ?", startID).Order("id asc").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return map[string]any{"logs": logs, "count": len(logs), "startId": startID, "limit": limit}, nil
}

package handlers

import (
	"encoding/json"
	"errors"

	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/service"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

type CloudflaredHandler struct {
	db      *gorm.DB
	service *service.CloudflaredService
}

func NewCloudflaredHandler(db *gorm.DB) *CloudflaredHandler {
	if db == nil {
		db = database.DB
	}
	return &CloudflaredHandler{db: db, service: service.NewCloudflaredService()}
}

func (h *CloudflaredHandler) UpdateConfig(data map[string]any) (any, error) {
	proxyKey := toString(data["proxyKey"])
	tunnelID := toString(data["tunnelId"])
	subdomain := toString(data["subdomain"])
	credentials := data["credentials"]
	publicIP := toString(data["publicIp"])
	if proxyKey == "" || tunnelID == "" || subdomain == "" || credentials == nil {
		return nil, &types.AdminError{Message: "missing required fields", Code: types.AdminErrorCodeInvalidRequest}
	}

	var creds service.TunnelCredentials
	switch value := credentials.(type) {
	case string:
		if err := json.Unmarshal([]byte(value), &creds); err != nil {
			return nil, &types.AdminError{Message: "invalid credentials format", Code: types.AdminErrorCodeInvalidCredentialsFormat}
		}
	case map[string]any:
		credJSON, err := json.Marshal(value)
		if err != nil {
			return nil, &types.AdminError{Message: "invalid credentials format", Code: types.AdminErrorCodeInvalidCredentialsFormat}
		}
		if err := json.Unmarshal(credJSON, &creds); err != nil {
			return nil, &types.AdminError{Message: "invalid credentials format", Code: types.AdminErrorCodeInvalidCredentialsFormat}
		}
	default:
		return nil, &types.AdminError{Message: "invalid credentials format", Code: types.AdminErrorCodeInvalidCredentialsFormat}
	}
	if creds.TunnelSecret == "" {
		return nil, &types.AdminError{Message: "credentials must contain TunnelSecret field", Code: types.AdminErrorCodeInvalidCredentialsFormat}
	}

	var proxy database.Proxy
	if err := h.db.Where("proxy_key = ?", proxyKey).First(&proxy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "proxy not found", Code: types.AdminErrorCodeProxyNotFound}
		}
		return nil, err
	}

	conf, restarted, restartErr, err := h.service.UpdateConfig(proxy.ID, tunnelID, subdomain, creds, publicIP)
	if err != nil {
		return nil, err
	}
	if conf == nil {
		return nil, &types.AdminError{Message: "cloudflared config not found", Code: types.AdminErrorCodeCloudflaredConfigNotFound}
	}

	message := "Cloudflared config updated and restarted successfully"
	if !restarted {
		message = "Config updated but restart failed"
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminDNSCreate,
		RequestParams: toJSONString(map[string]any{
			"proxyKey":  proxyKey,
			"subdomain": subdomain,
		}, "{}"),
	})

	result := map[string]any{
		"dnsConf": func() database.DnsConf {
			safe := *conf
			safe.Credentials = ""
			return safe
		}(),
		"restarted": restarted,
		"message":   message,
		"publicUrl": h.service.PublicURL(subdomain),
	}
	if restartErr != "" {
		result["restartError"] = restartErr
	}
	return result, nil
}

func (h *CloudflaredHandler) GetConfigs(data map[string]any) (any, error) {
	query := h.db.Model(&database.DnsConf{})
	if proxyKey := toString(data["proxyKey"]); proxyKey != "" {
		var proxy database.Proxy
		if err := h.db.Where("proxy_key = ?", proxyKey).First(&proxy).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &types.AdminError{Message: "proxy not found", Code: types.AdminErrorCodeProxyNotFound}
			}
			return nil, err
		}
		query = query.Where("proxy_id = ?", proxy.ID)
	}
	if tunnelID := toString(data["tunnelId"]); tunnelID != "" {
		query = query.Where("tunnel_id = ?", tunnelID)
	}
	if subdomain := toString(data["subdomain"]); subdomain != "" {
		query = query.Where("subdomain = ?", subdomain)
	}
	if _, ok := data["type"]; ok {
		query = query.Where("type = ?", toInt(data["type"], 0))
	}
	var confs []database.DnsConf
	if err := query.Order("id desc").Find(&confs).Error; err != nil {
		return nil, err
	}

	containerStatus := h.service.GetContainerStatus()
	dnsConfs := make([]map[string]any, len(confs))
	for i, conf := range confs {
		b, _ := json.Marshal(conf)
		m := map[string]any{}
		_ = json.Unmarshal(b, &m)
		delete(m, "credentials")
		m["status"] = containerStatus
		dnsConfs[i] = m
	}

	return map[string]any{"dnsConfs": dnsConfs}, nil
}

func (h *CloudflaredHandler) DeleteConfig(data map[string]any) (any, error) {
	id, hasID := asIntMaybe(data["id"])
	tunnelID := toString(data["tunnelId"])
	if !hasID && tunnelID == "" {
		return nil, &types.AdminError{Message: "either id or tunnelId must be provided", Code: types.AdminErrorCodeInvalidRequest}
	}
	query := h.db.Model(&database.DnsConf{})
	if hasID {
		query = query.Where("id = ?", id)
	}
	if tunnelID != "" {
		query = query.Where("tunnel_id = ?", tunnelID)
	}
	var conf database.DnsConf
	if err := query.First(&conf).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "cloudflared config not found", Code: types.AdminErrorCodeCloudflaredConfigNotFound}
		}
		return nil, err
	}

	deletedTunnelID := conf.TunnelID

	var deleteID *int
	if hasID {
		deleteID = &id
	}
	var deleteTunnelID *string
	if tunnelID != "" {
		deleteTunnelID = &tunnelID
	}
	if err := h.service.DeleteConfig(deleteID, deleteTunnelID); err != nil {
		return nil, err
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminDNSDelete,
		RequestParams: toJSONString(map[string]any{
			"id":       conf.ID,
			"tunnelId": deletedTunnelID,
		}, "{}"),
	})

	return map[string]any{
		"success": true,
		"message": "Cloudflared configuration deleted successfully",
		"deletedConfig": map[string]any{
			"id":        conf.ID,
			"tunnelId":  deletedTunnelID,
			"subdomain": conf.Subdomain,
		},
	}, nil
}

func (h *CloudflaredHandler) Restart() (any, error) {
	result, err := h.service.RestartCloudflared()
	if err != nil {
		return nil, err
	}
	requestParams := map[string]any{"action": "restart"}
	if result.Config != nil && result.Config.TunnelID != "" {
		requestParams["tunnelId"] = result.Config.TunnelID
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action:        types.MCPEventLogTypeAdminDNSCreate,
		RequestParams: toJSONString(requestParams, "{}"),
	})
	return result, nil
}

func (h *CloudflaredHandler) Stop() (any, error) {
	result, err := h.service.StopCloudflared()
	if err != nil {
		return nil, err
	}
	internallog.GetLogService().EnqueueLog(database.Log{
		Action: types.MCPEventLogTypeAdminDNSCreate,
		RequestParams: toJSONString(map[string]any{
			"action": "stop",
		}, "{}"),
	})
	return result, nil
}

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcpservice "github.com/dunialabs/kimbap-core/internal/mcp/service"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

type QueryHandler struct {
	db *gorm.DB
}

func NewQueryHandler(db *gorm.DB) *QueryHandler {
	if db == nil {
		db = database.DB
	}
	return &QueryHandler{db: db}
}

func (h *QueryHandler) GetAvailableServersCapabilities() (any, error) {
	var servers []database.Server
	if err := h.db.Where("enabled = ?", true).Find(&servers).Error; err != nil {
		return nil, err
	}

	serverManager := core.ServerManagerInstance()
	runtimeCaps := serverManager.GetAvailableServersCapabilities()

	capabilities := map[string]any{}
	serverIDsSeen := map[string]bool{}

	for serverID, cap := range runtimeCaps {
		serverIDsSeen[serverID] = true
		for _, server := range servers {
			if server.ServerID == serverID {
				cap.Enabled = server.PublicAccess
				break
			}
		}
		capabilities[serverID] = cap
	}

	for _, server := range servers {
		if serverIDsSeen[server.ServerID] {
			continue
		}
		serverCap := map[string]any{}
		parseFailed := false
		if server.Capabilities != "" {
			if err := json.Unmarshal([]byte(server.Capabilities), &serverCap); err != nil {
				serverCap = map[string]any{}
				parseFailed = true
			}
			if serverCap == nil {
				serverCap = map[string]any{}
				parseFailed = true
			}
		}
		serverEntry := map[string]any{
			"enabled":        server.PublicAccess,
			"serverName":     server.ServerName,
			"allowUserInput": server.AllowUserInput,
			"authType":       server.AuthType,
			"configTemplate": func() string {
				if server.AllowUserInput && server.ConfigTemplate != nil && strings.TrimSpace(*server.ConfigTemplate) != "" {
					return *server.ConfigTemplate
				}
				return "{}"
			}(),
			"configured": true,
			"tools":      mapOrDefault(serverCap["tools"]),
			"resources":  mapOrDefault(serverCap["resources"]),
			"prompts":    mapOrDefault(serverCap["prompts"]),
		}
		if parseFailed {
			serverEntry["capabilitiesParseError"] = true
		}
		capabilities[server.ServerID] = serverEntry
	}
	return map[string]any{"capabilities": capabilities}, nil
}

func (h *QueryHandler) GetUserAvailableServersCapabilities(data map[string]any) (any, error) {
	targetID := toString(data["targetId"])
	capabilities, err := mcpservice.CapabilitiesServiceInstance().GetCapabilitiesFromDatabase(context.Background(), targetID)
	if err != nil {
		return nil, err
	}
	for serverID, serverConfig := range capabilities {
		if serverConfig.AllowUserInput {
			serverConfig.Tools = map[string]mcptypes.ToolCapabilityConfig{}
			serverConfig.Resources = map[string]mcptypes.ResourceCapabilityConfig{}
			serverConfig.Prompts = map[string]mcptypes.PromptCapabilityConfig{}
			capabilities[serverID] = serverConfig
		}
	}
	return map[string]any{"capabilities": capabilities}, nil
}

func (h *QueryHandler) GetServersStatus() (any, error) {
	var servers []database.Server
	if err := h.db.Find(&servers).Error; err != nil {
		return nil, err
	}
	serverManager := core.ServerManagerInstance()
	statuses := map[string]int{}
	for _, server := range servers {
		if contextObj := serverManager.GetServerContext(server.ServerID, ""); contextObj != nil {
			statuses[server.ServerID] = contextObj.StatusSnapshot()
			continue
		}
		statuses[server.ServerID] = types.ServerStatusOffline
	}
	return map[string]any{"serversStatus": statuses}, nil
}

func (h *QueryHandler) GetServerCapabilities(data map[string]any) (any, error) {
	targetID := toString(data["targetId"])
	serverManager := core.ServerManagerInstance()
	if serverContext := serverManager.GetServerContext(targetID, ""); serverContext != nil {
		serverCapabilities := serverContext.GetMCPCapabilities()
		capabilities := map[string]any{
			"tools":     serverCapabilities.Tools,
			"resources": serverCapabilities.Resources,
			"prompts":   serverCapabilities.Prompts,
		}
		return map[string]any{"capabilities": capabilities}, nil
	}

	var server database.Server
	if err := h.db.Where("server_id = ?", targetID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{
				Message: fmt.Sprintf("Server %s not found", targetID),
				Code:    types.AdminErrorCodeServerNotFound,
			}
		}
		return nil, err
	}
	capabilities := map[string]any{}
	parseFailed := false
	if server.Capabilities != "" {
		if err := json.Unmarshal([]byte(server.Capabilities), &capabilities); err != nil {
			capabilities = map[string]any{}
			parseFailed = true
		}
		if capabilities == nil {
			capabilities = map[string]any{}
			parseFailed = true
		}
	}
	capabilities["tools"] = mapOrDefault(capabilities["tools"])
	capabilities["resources"] = mapOrDefault(capabilities["resources"])
	capabilities["prompts"] = mapOrDefault(capabilities["prompts"])
	if parseFailed {
		capabilities["capabilitiesParseError"] = true
	}
	return map[string]any{"capabilities": capabilities}, nil
}

func mapOrDefault(v any) any {
	if v == nil {
		return map[string]any{}
	}
	if _, ok := v.(map[string]any); !ok {
		return map[string]any{}
	}
	return v
}

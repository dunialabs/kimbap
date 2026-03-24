package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/types"
)

var capLog = logger.CreateLogger("CapabilitiesService")

type CapabilitiesService struct{}

var (
	capabilitiesInstance *CapabilitiesService
	capabilitiesOnce     sync.Once
)

func CapabilitiesServiceInstance() *CapabilitiesService {
	capabilitiesOnce.Do(func() {
		capabilitiesInstance = &CapabilitiesService{}
	})
	return capabilitiesInstance
}

func (s *CapabilitiesService) ComparePermissions(oldPermissions, newPermissions mcptypes.Permissions) (toolsChanged bool, resourcesChanged bool, promptsChanged bool) {
	oldPerms := oldPermissions
	if oldPerms == nil {
		oldPerms = mcptypes.Permissions{}
	}
	newPerms := newPermissions
	if newPerms == nil {
		newPerms = mcptypes.Permissions{}
	}

	seenServer := map[string]struct{}{}
	for serverID := range oldPerms {
		seenServer[serverID] = struct{}{}
	}
	for serverID := range newPerms {
		seenServer[serverID] = struct{}{}
	}

	for serverID := range seenServer {
		oldServer, oldOK := oldPerms[serverID]
		newServer, newOK := newPerms[serverID]

		if !oldOK && newOK {
			if newServer.Enabled {
				if !toolsChanged && hasAnyToolEnabled(newServer.Tools) {
					toolsChanged = true
				}
				if !resourcesChanged && hasAnyResourceEnabled(newServer.Resources) {
					resourcesChanged = true
				}
				if !promptsChanged && hasAnyPromptEnabled(newServer.Prompts) {
					promptsChanged = true
				}
			}
			continue
		}

		if oldOK && !newOK {
			if oldServer.Enabled {
				if !toolsChanged && hasAnyToolEnabled(oldServer.Tools) {
					toolsChanged = true
				}
				if !resourcesChanged && hasAnyResourceEnabled(oldServer.Resources) {
					resourcesChanged = true
				}
				if !promptsChanged && hasAnyPromptEnabled(oldServer.Prompts) {
					promptsChanged = true
				}
			}
			continue
		}

		if oldServer.Enabled != newServer.Enabled {
			source := oldServer
			if newServer.Enabled {
				source = newServer
			}
			if !toolsChanged && hasAnyToolEnabled(source.Tools) {
				toolsChanged = true
			}
			if !resourcesChanged && hasAnyResourceEnabled(source.Resources) {
				resourcesChanged = true
			}
			if !promptsChanged && hasAnyPromptEnabled(source.Prompts) {
				promptsChanged = true
			}
			continue
		}

		if !toolsChanged && isToolCapabilityListChanged(oldServer.Tools, newServer.Tools) {
			toolsChanged = true
		}
		if !resourcesChanged && isResourceCapabilityListChanged(oldServer.Resources, newServer.Resources) {
			resourcesChanged = true
		}
		if !promptsChanged && isPromptCapabilityListChanged(oldServer.Prompts, newServer.Prompts) {
			promptsChanged = true
		}
	}

	return toolsChanged, resourcesChanged, promptsChanged
}

func (s *CapabilitiesService) GetCapabilitiesFromDatabase(ctx context.Context, userID string) (mcptypes.MCPServerCapabilities, error) {
	userRepo := core.ServerManagerInstance().UserRepository()
	if userRepo == nil {
		return mcptypes.MCPServerCapabilities{}, nil
	}
	user, err := userRepo.FindByUserID(ctx, userID)
	if err != nil {
		return mcptypes.MCPServerCapabilities{}, err
	}
	if user == nil {
		return mcptypes.MCPServerCapabilities{}, fmt.Errorf("User %s not found", userID)
	}

	permissions := mcptypes.Permissions{}
	trimmedPermissions := strings.TrimSpace(user.Permissions)
	if trimmedPermissions != "" {
		if err := json.Unmarshal([]byte(user.Permissions), &permissions); err != nil {
			return mcptypes.MCPServerCapabilities{}, fmt.Errorf("invalid user permissions for user %s: %w", userID, err)
		}
		if permissions == nil {
			return mcptypes.MCPServerCapabilities{}, fmt.Errorf("invalid user permissions for user %s: permissions cannot be null", userID)
		}
	}
	rawPermissions := map[string]any{}
	if trimmedPermissions != "" {
		if err := json.Unmarshal([]byte(user.Permissions), &rawPermissions); err != nil {
			return mcptypes.MCPServerCapabilities{}, fmt.Errorf("invalid raw user permissions for user %s: %w", userID, err)
		}
		if rawPermissions == nil {
			return mcptypes.MCPServerCapabilities{}, fmt.Errorf("invalid raw user permissions for user %s: permissions cannot be null", userID)
		}
	}
	launchConfigs := map[string]any{}
	if err := json.Unmarshal([]byte(user.LaunchConfigs), &launchConfigs); err != nil && user.LaunchConfigs != "" {
		capLog.Warn().Err(err).Str("userId", userID).Msg("failed to unmarshal launch configs")
	}

	result := mcptypes.MCPServerCapabilities{}
	allEnabledServers := []database.Server{}
	repo := core.ServerManagerInstance().Repository()
	if repo != nil {
		allEnabledServers, err = repo.FindAllEnabled(ctx)
		if err != nil {
			return mcptypes.MCPServerCapabilities{}, err
		}
	}

	for _, serverEntity := range allEnabledServers {
		serverID := serverEntity.ServerID

		var serverCtx *core.ServerContext
		if serverEntity.AllowUserInput {
			serverCtx = core.ServerManagerInstance().GetTemporaryServer(serverID, userID)
		} else {
			serverCtx = core.ServerManagerInstance().GetServerContext(serverID, "")
		}

		// If admin has explicitly set a permission, use it; otherwise fall back to publicAccess.
		perm, hasAdminPerm := permissions[serverID]
		rawServerPerm, hasRawServerPerm := rawPermissions[serverID]
		serverEnabled := serverEntity.PublicAccess
		if hasAdminPerm && hasRawServerPerm {
			if enabled, ok := boolFieldFromAnyMap(rawServerPerm, "enabled"); ok {
				serverEnabled = enabled
			}
		}

		serverStatus := types.ServerStatusOffline
		var tools map[string]mcptypes.ToolCapabilityConfig
		var resources map[string]mcptypes.ResourceCapabilityConfig
		var prompts map[string]mcptypes.PromptCapabilityConfig
		if serverCtx != nil {
			serverStatus = serverCtx.StatusSnapshot()
			// Use live capabilities which merge discovered tools/resources/prompts with config
			liveCaps := serverCtx.GetMCPCapabilities()
			tools = cloneToolCapabilities(liveCaps.Tools)
			resources = cloneResourceCapabilities(liveCaps.Resources)
			prompts = clonePromptCapabilities(liveCaps.Prompts)
		} else {
			baseCapabilities := parseServerCapabilitiesConfig(serverEntity.Capabilities)
			tools = cloneToolCapabilities(baseCapabilities.Tools)
			resources = cloneResourceCapabilities(baseCapabilities.Resources)
			prompts = clonePromptCapabilities(baseCapabilities.Prompts)
		}

		filterDisabledTools(tools)
		filterDisabledResources(resources)
		filterDisabledPrompts(prompts)

		rawToolPerms := nestedAnyMap(rawServerPerm, "tools")
		for toolName, toolPerm := range perm.Tools {
			enabled, hasEnabled := boolFieldFromNamedMap(rawToolPerms, toolName, "enabled")
			if toolCfg, ok := tools[toolName]; ok {
				if hasEnabled {
					toolCfg.Enabled = enabled
				}
				if toolPerm.DangerLevel != nil {
					toolCfg.DangerLevel = intPtr(*toolPerm.DangerLevel)
				}
				tools[toolName] = toolCfg
			}
		}

		rawResourcePerms := nestedAnyMap(rawServerPerm, "resources")
		for resourceName := range perm.Resources {
			enabled, ok := boolFieldFromNamedMap(rawResourcePerms, resourceName, "enabled")
			if !ok {
				continue
			}
			if resourceCfg, ok := resources[resourceName]; ok {
				resourceCfg.Enabled = enabled
				resources[resourceName] = resourceCfg
			}
		}

		rawPromptPerms := nestedAnyMap(rawServerPerm, "prompts")
		for promptName := range perm.Prompts {
			enabled, ok := boolFieldFromNamedMap(rawPromptPerms, promptName, "enabled")
			if !ok {
				continue
			}
			if promptCfg, ok := prompts[promptName]; ok {
				promptCfg.Enabled = enabled
				prompts[promptName] = promptCfg
			}
		}

		configTemplate := "{}"
		if serverEntity.AllowUserInput && serverEntity.ConfigTemplate != nil {
			configTemplate = *serverEntity.ConfigTemplate
		}

		config := mcptypes.ServerConfigWithEnabled{
			Tools:          tools,
			Resources:      resources,
			Prompts:        prompts,
			Enabled:        serverEnabled,
			ServerName:     serverEntity.ServerName,
			AllowUserInput: serverEntity.AllowUserInput,
			AuthType:       serverEntity.AuthType,
			Category:       intPtr(serverEntity.Category),
			ConfigTemplate: configTemplate,
			Configured:     !serverEntity.AllowUserInput,
			Status:         intPtr(serverStatus),
		}
		if serverEntity.AllowUserInput {
			_, userConfigured := launchConfigs[serverID]
			config.Configured = userConfigured
		}
		result[serverID] = config
	}

	return result, nil
}

func (s *CapabilitiesService) GetUserCapabilities(ctx context.Context, userID string) (mcptypes.MCPServerCapabilities, error) {
	result, err := s.GetCapabilitiesFromDatabase(ctx, userID)
	if err != nil {
		return mcptypes.MCPServerCapabilities{}, err
	}

	for serverID, serverConfig := range result {
		if !serverConfig.Enabled {
			delete(result, serverID)
			continue
		}
		filterDisabledTools(serverConfig.Tools)
		filterDisabledResources(serverConfig.Resources)
		filterDisabledPrompts(serverConfig.Prompts)
		result[serverID] = serverConfig
	}

	userRepo := core.ServerManagerInstance().UserRepository()
	if userRepo == nil {
		return result, nil
	}
	user, err := userRepo.FindByUserID(ctx, userID)
	if err != nil {
		return mcptypes.MCPServerCapabilities{}, err
	}
	if user == nil {
		return mcptypes.MCPServerCapabilities{}, fmt.Errorf("User %s not found", userID)
	}

	preferences := mcptypes.Permissions{}
	trimmedUserPreferences := strings.TrimSpace(user.UserPreferences)
	if trimmedUserPreferences != "" {
		if err := json.Unmarshal([]byte(user.UserPreferences), &preferences); err != nil {
			capLog.Warn().Err(err).Str("userId", userID).Msg("failed to unmarshal user preferences")
			return mcptypes.MCPServerCapabilities{}, fmt.Errorf("invalid user preferences")
		}
	}
	if preferences == nil {
		capLog.Warn().Str("userId", userID).Msg("invalid user preferences json: null")
		return mcptypes.MCPServerCapabilities{}, fmt.Errorf("invalid user preferences")
	}

	for serverID, prefConfig := range preferences {
		cfg, ok := result[serverID]
		if !ok {
			continue
		}

		cfg.Enabled = prefConfig.Enabled

		for toolName, toolPref := range prefConfig.Tools {
			if toolCfg, exists := cfg.Tools[toolName]; exists {
				toolCfg.Enabled = toolPref.Enabled
				if toolPref.DangerLevel != nil {
					toolCfg.DangerLevel = intPtr(*toolPref.DangerLevel)
				}
				cfg.Tools[toolName] = toolCfg
			}
		}

		for resourceName, resourcePref := range prefConfig.Resources {
			if resourceCfg, exists := cfg.Resources[resourceName]; exists {
				resourceCfg.Enabled = resourcePref.Enabled
				cfg.Resources[resourceName] = resourceCfg
			}
		}

		for promptName, promptPref := range prefConfig.Prompts {
			if promptCfg, exists := cfg.Prompts[promptName]; exists {
				promptCfg.Enabled = promptPref.Enabled
				cfg.Prompts[promptName] = promptCfg
			}
		}

		result[serverID] = cfg
	}

	return result, nil
}

func cloneToolCapabilities(src map[string]mcptypes.ToolCapabilityConfig) map[string]mcptypes.ToolCapabilityConfig {
	dst := make(map[string]mcptypes.ToolCapabilityConfig, len(src))
	for key, value := range src {
		copied := value
		if value.DangerLevel != nil {
			copied.DangerLevel = intPtr(*value.DangerLevel)
		}
		dst[key] = copied
	}
	return dst
}

func cloneResourceCapabilities(src map[string]mcptypes.ResourceCapabilityConfig) map[string]mcptypes.ResourceCapabilityConfig {
	dst := make(map[string]mcptypes.ResourceCapabilityConfig, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func clonePromptCapabilities(src map[string]mcptypes.PromptCapabilityConfig) map[string]mcptypes.PromptCapabilityConfig {
	dst := make(map[string]mcptypes.PromptCapabilityConfig, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func filterDisabledTools(caps map[string]mcptypes.ToolCapabilityConfig) {
	for name, cfg := range caps {
		if !cfg.Enabled {
			delete(caps, name)
		}
	}
}

func filterDisabledResources(caps map[string]mcptypes.ResourceCapabilityConfig) {
	for name, cfg := range caps {
		if !cfg.Enabled {
			delete(caps, name)
		}
	}
}

func filterDisabledPrompts(caps map[string]mcptypes.PromptCapabilityConfig) {
	for name, cfg := range caps {
		if !cfg.Enabled {
			delete(caps, name)
		}
	}
}

func hasAnyToolEnabled(caps map[string]mcptypes.ToolCapabilityConfig) bool {
	for _, item := range caps {
		if item.Enabled {
			return true
		}
	}
	return false
}

func hasAnyResourceEnabled(caps map[string]mcptypes.ResourceCapabilityConfig) bool {
	for _, item := range caps {
		if item.Enabled {
			return true
		}
	}
	return false
}

func hasAnyPromptEnabled(caps map[string]mcptypes.PromptCapabilityConfig) bool {
	for _, item := range caps {
		if item.Enabled {
			return true
		}
	}
	return false
}

func isToolCapabilityListChanged(oldCaps, newCaps map[string]mcptypes.ToolCapabilityConfig) bool {
	seen := map[string]struct{}{}
	for key := range oldCaps {
		seen[key] = struct{}{}
	}
	for key := range newCaps {
		seen[key] = struct{}{}
	}
	for key := range seen {
		oldVal, oldOK := oldCaps[key]
		newVal, newOK := newCaps[key]
		if !oldOK || !newOK {
			return true
		}
		if oldVal.Enabled != newVal.Enabled {
			return true
		}
		if (oldVal.DangerLevel == nil) != (newVal.DangerLevel == nil) {
			return true
		}
		if oldVal.DangerLevel != nil && newVal.DangerLevel != nil && *oldVal.DangerLevel != *newVal.DangerLevel {
			return true
		}
	}
	return false
}

func isResourceCapabilityListChanged(oldCaps, newCaps map[string]mcptypes.ResourceCapabilityConfig) bool {
	seen := map[string]struct{}{}
	for key := range oldCaps {
		seen[key] = struct{}{}
	}
	for key := range newCaps {
		seen[key] = struct{}{}
	}
	for key := range seen {
		oldVal, oldOK := oldCaps[key]
		newVal, newOK := newCaps[key]
		if !oldOK || !newOK {
			return true
		}
		if oldVal.Enabled != newVal.Enabled {
			return true
		}
	}
	return false
}

func isPromptCapabilityListChanged(oldCaps, newCaps map[string]mcptypes.PromptCapabilityConfig) bool {
	seen := map[string]struct{}{}
	for key := range oldCaps {
		seen[key] = struct{}{}
	}
	for key := range newCaps {
		seen[key] = struct{}{}
	}
	for key := range seen {
		oldVal, oldOK := oldCaps[key]
		newVal, newOK := newCaps[key]
		if !oldOK || !newOK {
			return true
		}
		if oldVal.Enabled != newVal.Enabled {
			return true
		}
	}
	return false
}

func intPtr(v int) *int {
	return &v
}

func nestedAnyMap(raw any, key string) map[string]any {
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return nil
	}
	child, ok := m[key].(map[string]any)
	if !ok {
		return nil
	}
	return child
}

func boolFieldFromAnyMap(raw any, field string) (bool, bool) {
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return false, false
	}
	v, exists := m[field]
	if !exists {
		return false, false
	}
	b, ok := v.(bool)
	if !ok {
		return false, false
	}
	return b, true
}

func boolFieldFromNamedMap(raw map[string]any, name string, field string) (bool, bool) {
	if raw == nil {
		return false, false
	}
	entry, exists := raw[name]
	if !exists {
		return false, false
	}
	return boolFieldFromAnyMap(entry, field)
}

func parseServerCapabilitiesConfig(raw string) mcptypes.ServerConfigCapabilities {
	parsed := mcptypes.ServerConfigCapabilities{
		Tools:     map[string]mcptypes.ToolCapabilityConfig{},
		Resources: map[string]mcptypes.ResourceCapabilityConfig{},
		Prompts:   map[string]mcptypes.PromptCapabilityConfig{},
	}
	if strings.TrimSpace(raw) == "" {
		return parsed
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return mcptypes.ServerConfigCapabilities{
			Tools:     map[string]mcptypes.ToolCapabilityConfig{},
			Resources: map[string]mcptypes.ResourceCapabilityConfig{},
			Prompts:   map[string]mcptypes.PromptCapabilityConfig{},
		}
	}
	if parsed.Tools == nil {
		parsed.Tools = map[string]mcptypes.ToolCapabilityConfig{}
	}
	if parsed.Resources == nil {
		parsed.Resources = map[string]mcptypes.ResourceCapabilityConfig{}
	}
	if parsed.Prompts == nil {
		parsed.Prompts = map[string]mcptypes.PromptCapabilityConfig{}
	}
	return parsed
}

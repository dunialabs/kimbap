package core

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// ClientInfo represents the MCP client implementation info (name + version).
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ClientSession struct {
	mu sync.RWMutex

	SessionID string
	UserID    string
	Token     string

	LastActive time.Time
	Status     int

	Connection *mcp.Server

	AuthContext    mcptypes.AuthContext
	Capabilities   map[string]any
	ClientInfoData *ClientInfo

	SSEConnected          bool
	LastSSEDisconnectedAt *time.Time
	LastUserInfoRefresh   int64

	proxySession *ProxySession
	closeOnce    sync.Once
}

func NewClientSession(sessionID, userID, token string, authContext mcptypes.AuthContext) *ClientSession {
	return &ClientSession{
		SessionID:    sessionID,
		UserID:       userID,
		Token:        token,
		LastActive:   time.Now(),
		Status:       types.ClientSessionStatusActive,
		AuthContext:  authContext,
		Capabilities: map[string]any{},
	}
}

func (s *ClientSession) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActive = time.Now()
}

func (s *ClientSession) LastActiveSnapshot() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastActive
}

func (s *ClientSession) MarkSSEConnected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SSEConnected = true
	s.LastSSEDisconnectedAt = nil
	s.LastActive = time.Now()
}

func (s *ClientSession) MarkSSEDisconnected() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SSEConnected = false
	s.LastSSEDisconnectedAt = &now
}

func (s *ClientSession) IsSSEConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SSEConnected
}

func (s *ClientSession) IsExpired(now time.Time, timeoutMinutes int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.AuthContext.ExpiresAt > 0 && now.Unix() > s.AuthContext.ExpiresAt {
		return true
	}
	if timeoutMinutes <= 0 {
		return false
	}
	return now.Sub(s.LastActive) > time.Duration(timeoutMinutes)*time.Minute
}

func (s *ClientSession) IsInactive(now time.Time, timeout time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.SSEConnected {
		return false
	}
	effectiveLast := s.LastActive
	if s.LastSSEDisconnectedAt != nil && s.LastSSEDisconnectedAt.After(effectiveLast) {
		effectiveLast = *s.LastSSEDisconnectedAt
	}
	return now.Sub(effectiveLast) > timeout
}

func (s *ClientSession) SetProxySession(ps *ProxySession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proxySession = ps
}

func (s *ClientSession) GetProxySession() *ProxySession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.proxySession
}

func (s *ClientSession) UpdatePermissions(perms mcptypes.Permissions) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuthContext.Permissions = perms
}

func (s *ClientSession) GetUserPreferences() mcptypes.Permissions {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AuthContext.UserPreferences
}

func (s *ClientSession) UpdateUserPreferences(prefs mcptypes.Permissions) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuthContext.UserPreferences = prefs
}

func (s *ClientSession) UpdateLaunchConfigs(launchConfigs string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuthContext.LaunchConfigs = launchConfigs
}

func (s *ClientSession) UpdateExpiresAt(expiresAt int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuthContext.ExpiresAt = expiresAt
}

func (s *ClientSession) AuthContextSnapshot() mcptypes.AuthContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AuthContext
}

func (s *ClientSession) UpdateAuthContext(authContext mcptypes.AuthContext) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuthContext = authContext
}

func (s *ClientSession) GetLastUserInfoRefresh() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastUserInfoRefresh
}

func (s *ClientSession) UpdateLastUserInfoRefresh(lastRefresh int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastUserInfoRefresh = lastRefresh
}

func (s *ClientSession) SetClientInfo(info *ClientInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ClientInfoData = info
}

func (s *ClientSession) GetClientInfo() *ClientInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ClientInfoData
}

func (s *ClientSession) CanRequestSampling() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Capabilities["sampling"]
	return ok
}

func (s *ClientSession) CanRequestElicitation() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Capabilities["elicitation"]
	return ok
}

func (s *ClientSession) CanRequestRoots() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Capabilities["roots"]
	return ok
}

func (s *ClientSession) CanAccessServer(serverID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := ServerManagerInstance().GetServerContext(serverID, s.UserID)
	if ctx == nil {
		return false
	}

	serverEntity := ctx.ServerEntitySnapshot()
	serverStatus := ctx.StatusSnapshot()
	ctxUserID := ctx.UserIDSnapshot()
	serverEnabled := serverEntity.Enabled
	serverPublicAccess := serverEntity.PublicAccess
	allowUserInput := serverEntity.AllowUserInput

	if !serverEnabled {
		return false
	}
	if serverStatus != types.ServerStatusOnline && serverStatus != types.ServerStatusSleeping {
		return false
	}

	isAnonymous := s.AuthContext.Kind == "anonymous"
	if isAnonymous && !serverEntity.AnonymousAccess {
		return false
	}

	perm, hasPerm := s.AuthContext.Permissions[serverID]
	pref, hasPref := s.AuthContext.UserPreferences[serverID]

	serverPermEnabled := false
	if !isAnonymous {
		serverPermEnabled = serverPublicAccess
	}
	if hasPerm {
		serverPermEnabled = perm.Enabled
	}
	userPrefEnabled := true
	if hasPref {
		userPrefEnabled = pref.Enabled
	}

	if allowUserInput && ctxUserID != s.UserID {
		return false
	}

	return serverPermEnabled && userPrefEnabled
}

func (s *ClientSession) CanUseTool(serverID, toolName string) bool {
	return s.canAccessCapability(serverID, "tool", toolName)
}

func (s *ClientSession) CanAccessResource(serverID, resourceName string) bool {
	return s.canAccessCapability(serverID, "resource", resourceName)
}

func (s *ClientSession) CanUsePrompt(serverID, promptName string) bool {
	return s.canAccessCapability(serverID, "prompt", promptName)
}

func (s *ClientSession) canAccessCapability(serverID, kind, name string) bool {
	ctx := ServerManagerInstance().GetServerContext(serverID, s.UserID)
	if ctx == nil {
		return false
	}
	if !s.CanAccessServer(serverID) {
		return false
	}
	if !isCapabilityEnabledByServerConfig(ctx, kind, name) {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	serverEntity := ctx.ServerEntitySnapshot()
	allowUserInput := serverEntity.AllowUserInput
	ctxUserID := ctx.UserIDSnapshot()
	pref := s.AuthContext.UserPreferences[serverID]

	if allowUserInput {
		if ctxUserID != s.UserID {
			return false
		}
		switch kind {
		case "tool":
			if cfg, ok := pref.Tools[name]; ok && !cfg.Enabled {
				return false
			}
		case "resource":
			if cfg, ok := pref.Resources[name]; ok && !cfg.Enabled {
				return false
			}
		case "prompt":
			if cfg, ok := pref.Prompts[name]; ok && !cfg.Enabled {
				return false
			}
		}
		return true
	}

	perm := s.AuthContext.Permissions[serverID]

	switch kind {
	case "tool":
		if cfg, ok := perm.Tools[name]; ok && !cfg.Enabled {
			return false
		}
		if cfg, ok := pref.Tools[name]; ok && !cfg.Enabled {
			return false
		}
	case "resource":
		if cfg, ok := perm.Resources[name]; ok && !cfg.Enabled {
			return false
		}
		if cfg, ok := pref.Resources[name]; ok && !cfg.Enabled {
			return false
		}
	case "prompt":
		if cfg, ok := perm.Prompts[name]; ok && !cfg.Enabled {
			return false
		}
		if cfg, ok := pref.Prompts[name]; ok && !cfg.Enabled {
			return false
		}
	}

	return true
}

func isCapabilityEnabledByServerConfig(ctx *ServerContext, kind, name string) bool {
	if ctx == nil {
		return false
	}
	capabilities := ctx.CapabilitiesConfigSnapshot()
	switch kind {
	case "tool":
		if cfg, ok := capabilities.Tools[name]; ok {
			return cfg.Enabled
		}
	case "resource":
		if cfg, ok := capabilities.Resources[name]; ok {
			return cfg.Enabled
		}
	case "prompt":
		if cfg, ok := capabilities.Prompts[name]; ok {
			return cfg.Enabled
		}
	}
	return true
}

func (s *ClientSession) GetAvailableServers() []*ServerContext {
	servers := ServerManagerInstance().GetAvailableServers()
	result := make([]*ServerContext, 0, len(servers))
	for _, server := range servers {
		if s.CanAccessServer(server.ServerID) {
			result = append(result, server)
		}
	}
	return result
}

func (s *ClientSession) GetServerCapabilities() mcp.ServerCapabilities {
	merged := mcp.ServerCapabilities{
		Tools: &mcp.ToolCapabilities{ListChanged: true},
	}

	for _, serverContext := range s.GetAvailableServers() {
		if serverContext == nil {
			continue
		}
		caps, hasPromptData, hasResourceData := snapshotServerCapabilities(serverContext)

		if caps.Prompts == nil && hasPromptData {
			caps.Prompts = &mcp.PromptCapabilities{}
		}
		if caps.Resources == nil && hasResourceData {
			caps.Resources = &mcp.ResourceCapabilities{}
		}

		if merged.Prompts == nil && caps.Prompts != nil {
			merged.Prompts = &mcp.PromptCapabilities{}
		}
		if caps.Prompts != nil && caps.Prompts.ListChanged {
			merged.Prompts.ListChanged = true
		}

		if merged.Resources == nil && caps.Resources != nil {
			merged.Resources = &mcp.ResourceCapabilities{}
		}
		if caps.Resources != nil && caps.Resources.ListChanged {
			merged.Resources.ListChanged = true
		}
		if merged.Resources != nil && !merged.Resources.Subscribe && caps.Resources != nil && caps.Resources.Subscribe {
			merged.Resources.Subscribe = true
		}

		if merged.Completions == nil && caps.Completions != nil {
			merged.Completions = &mcp.CompletionCapabilities{}
		}
		if merged.Logging == nil && caps.Logging != nil {
			merged.Logging = &mcp.LoggingCapabilities{}
		}
	}

	if merged.Resources != nil {
		merged.Resources.ListChanged = true
	}

	return merged
}

func (s *ClientSession) EffectiveDangerLevel(serverID, toolName string, serverDangerLevel *int) *int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var userDangerLevel *int
	if pref, ok := s.AuthContext.UserPreferences[serverID]; ok {
		if tc, ok := pref.Tools[toolName]; ok && tc.DangerLevel != nil {
			userDangerLevel = tc.DangerLevel
		}
	}

	// User preference overrides admin-set level (userDangerLevel ?? serverDangerLevel)
	if userDangerLevel != nil {
		return userDangerLevel
	}
	return serverDangerLevel
}

func (s *ClientSession) UserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AuthContext.UserAgent
}

// annotatedTool wraps mcp.Tool with root-level readonly and destructiveHint
// fields. The tool object is spread and these custom root-level fields are added.
type annotatedTool struct {
	*mcp.Tool
	Readonly        bool `json:"readonly"`
	DestructiveHint bool `json:"destructiveHint"`
}

// annotatedListToolsResult wraps mcp.ListToolsResult to implement mcp.Result
// while providing custom JSON marshaling that includes per-tool root-level annotations.
type annotatedListToolsResult struct {
	mcp.ListToolsResult // embeds isResult() to satisfy mcp.Result interface
	annotatedTools      []*annotatedTool
}

func (r *annotatedListToolsResult) MarshalJSON() ([]byte, error) {
	type result struct {
		Meta       mcp.Meta         `json:"_meta,omitempty"`
		NextCursor string           `json:"nextCursor,omitempty"`
		Tools      []*annotatedTool `json:"tools"`
	}
	return json.Marshal(&result{
		Meta:       r.ListToolsResult.Meta,
		NextCursor: r.ListToolsResult.NextCursor,
		Tools:      r.annotatedTools,
	})
}

func (s *ClientSession) ListTools() *annotatedListToolsResult {
	tools := make([]*annotatedTool, 0)
	for _, server := range s.GetAvailableServers() {
		server.mu.RLock()
		items := server.Tools
		if items == nil {
			items = server.CachedTools
		}
		serverID := server.ServerID
		internalID := server.ID
		server.mu.RUnlock()
		if items == nil {
			continue
		}
		for _, tool := range items.Tools {
			if tool == nil {
				continue
			}
			if !s.CanUseTool(serverID, tool.Name) {
				continue
			}
			toolCopy := *tool
			toolCopy.Name = s.GenerateNewName(internalID, tool.Name)

			// Compute readonly and destructiveHint from annotations
			readOnly := false
			destructive := false
			if toolCopy.Annotations != nil {
				readOnly = toolCopy.Annotations.ReadOnlyHint
				if toolCopy.Annotations.DestructiveHint != nil {
					destructive = *toolCopy.Annotations.DestructiveHint
				}
			}

			// Apply DangerLevel overrides — user-level takes precedence, then server-level
			// User-level danger level takes precedence, then server-level
			dl := s.EffectiveDangerLevel(serverID, tool.Name, server.GetDangerLevel(tool.Name))
			if dl != nil {
				if toolCopy.Annotations != nil {
					annotCopy := *toolCopy.Annotations
					toolCopy.Annotations = &annotCopy
				} else {
					toolCopy.Annotations = &mcp.ToolAnnotations{}
				}

				switch *dl {
				case types.DangerLevelNotification:
					if !destructive {
						readOnly = false
						destructive = true
					}
				case types.DangerLevelSilent:
					readOnly = true
					destructive = false
				}

				toolCopy.Annotations.ReadOnlyHint = readOnly
				toolCopy.Annotations.DestructiveHint = &destructive
			}

			tools = append(tools, &annotatedTool{
				Tool:            &toolCopy,
				Readonly:        readOnly,
				DestructiveHint: destructive,
			})
		}
	}
	mcpTools := make([]*mcp.Tool, len(tools))
	for i, t := range tools {
		mcpTools[i] = t.Tool
	}
	return &annotatedListToolsResult{
		ListToolsResult: mcp.ListToolsResult{
			Tools: mcpTools,
			Meta:  mcp.Meta{"totalCount": len(tools)},
		},
		annotatedTools: tools,
	}
}

func (s *ClientSession) ListResources() *mcp.ListResourcesResult {
	resources := make([]*mcp.Resource, 0)
	for _, server := range s.GetAvailableServers() {
		server.mu.RLock()
		items := server.Resources
		if items == nil {
			items = server.CachedResources
		}
		serverID := server.ServerID
		internalID := server.ID
		server.mu.RUnlock()
		if items == nil {
			continue
		}
		for _, resource := range items.Resources {
			if resource == nil {
				continue
			}
			if !s.CanAccessResource(serverID, resource.Name) {
				continue
			}
			copy := *resource
			copy.URI = s.GenerateNewName(internalID, resource.URI)
			resources = append(resources, &copy)
		}
	}
	return &mcp.ListResourcesResult{
		Resources: resources,
		Meta:      mcp.Meta{"totalCount": len(resources)},
	}
}

func (s *ClientSession) ListResourceTemplates() *mcp.ListResourceTemplatesResult {
	templates := make([]*mcp.ResourceTemplate, 0)
	for _, server := range s.GetAvailableServers() {
		server.mu.RLock()
		items := server.ResourceTemplates
		if items == nil {
			items = server.CachedResourceTemplates
		}
		serverID := server.ServerID
		internalID := server.ID
		server.mu.RUnlock()
		if items == nil {
			continue
		}
		for _, resourceTemplate := range items.ResourceTemplates {
			if resourceTemplate == nil {
				continue
			}
			if !s.CanAccessResource(serverID, resourceTemplate.Name) {
				continue
			}
			copy := *resourceTemplate
			copy.URITemplate = s.GenerateNewName(internalID, resourceTemplate.URITemplate)
			templates = append(templates, &copy)
		}
	}
	return &mcp.ListResourceTemplatesResult{
		ResourceTemplates: templates,
		Meta:              mcp.Meta{"totalCount": len(templates)},
	}
}

func (s *ClientSession) ListPrompts() *mcp.ListPromptsResult {
	prompts := make([]*mcp.Prompt, 0)
	for _, server := range s.GetAvailableServers() {
		server.mu.RLock()
		items := server.Prompts
		if items == nil {
			items = server.CachedPrompts
		}
		serverID := server.ServerID
		internalID := server.ID
		server.mu.RUnlock()
		if items == nil {
			continue
		}
		for _, prompt := range items.Prompts {
			if prompt == nil {
				continue
			}
			if !s.CanUsePrompt(serverID, prompt.Name) {
				continue
			}
			copy := *prompt
			copy.Name = s.GenerateNewName(internalID, prompt.Name)
			prompts = append(prompts, &copy)
		}
	}
	return &mcp.ListPromptsResult{
		Prompts: prompts,
		Meta:    mcp.Meta{"totalCount": len(prompts)},
	}
}

func (s *ClientSession) GenerateNewName(serverID, name string) string {
	return name + "_-_" + serverID
}

func (s *ClientSession) ParseName(name string) (serverID string, originalName string, ok bool) {
	idx := strings.LastIndex(name, "_-_")
	if idx == -1 {
		return "", "", false
	}
	internalID := name[idx+3:]
	ctx := ServerManagerInstance().GetServerContextByInternalID(internalID, s.UserID)
	if ctx == nil {
		return "", "", false
	}
	return ctx.ServerID, name[:idx], true
}

func (s *ClientSession) ConnectionInitialized(conn *mcp.Server) {
	s.mu.Lock()
	s.Connection = conn
	userID := s.UserID
	s.mu.Unlock()

	notifier := ServerManagerInstance().Notifier()
	if notifier != nil {
		if ok := notifier.NotifyOnlineSessions(userID); !ok {
			log.Debug().Str("userId", userID).Msg("notify online sessions after session connection initialization returned false")
		}
	}

	ServerManagerInstance().StartUserTemporaryServersForSession(s)
}

func (s *ClientSession) SendToolListChanged() {
	if !s.CanSendToolListChanged() {
		return
	}
	s.SendNotification("notifications/tools/list_changed", nil)
}

func (s *ClientSession) SendResourceListChanged() {
	if !s.CanSendResourceListChanged() {
		return
	}
	s.SendNotification("notifications/resources/list_changed", nil)
}

func (s *ClientSession) SendPromptListChanged() {
	if !s.CanSendPromptListChanged() {
		return
	}
	s.SendNotification("notifications/prompts/list_changed", nil)
}

func (s *ClientSession) CanSendToolListChanged() bool {
	caps := s.GetServerCapabilities()
	return caps.Tools != nil && caps.Tools.ListChanged
}

func (s *ClientSession) CanSendResourceListChanged() bool {
	caps := s.GetServerCapabilities()
	return caps.Resources != nil && caps.Resources.ListChanged
}

func (s *ClientSession) CanSendPromptListChanged() bool {
	caps := s.GetServerCapabilities()
	return caps.Prompts != nil && caps.Prompts.ListChanged
}

func (s *ClientSession) SendResourceUpdated(uri string) {
	if strings.TrimSpace(uri) == "" {
		return
	}
	s.SendNotification("notifications/resources/updated", &mcp.ResourceUpdatedNotificationParams{URI: uri})
}

func (s *ClientSession) SendNotification(method string, params any) {
	s.mu.RLock()
	conn := s.Connection
	sessionID := s.SessionID
	s.mu.RUnlock()

	if conn == nil {
		return
	}

	switch method {
	case "notifications/tools/list_changed":
		emitToolsListChanged(conn)
	case "notifications/resources/list_changed":
		emitResourcesListChanged(conn)
	case "notifications/prompts/list_changed":
		emitPromptsListChanged(conn)
	case "notifications/resources/updated":
		resourceURI := resourceURIFromNotificationParams(params)
		if resourceURI == "" {
			log.Warn().Str("sessionId", sessionID).Msg("failed to send resources/updated notification: missing uri")
			return
		}
		if err := conn.ResourceUpdated(context.Background(), &mcp.ResourceUpdatedNotificationParams{URI: resourceURI}); err != nil {
			log.Warn().Err(err).Str("sessionId", sessionID).Str("uri", resourceURI).Msg("failed to send resources/updated notification")
		}
	default:
		log.Warn().Str("sessionId", sessionID).Str("method", method).Msg("unsupported notification method")
	}
}

func (s *ClientSession) Close(reason mcptypes.DisconnectReason) {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.Status = types.ClientSessionStatusClosed
		conn := s.Connection
		s.Connection = nil
		s.proxySession = nil
		userID := s.UserID
		sessionID := s.SessionID
		s.mu.Unlock()

		if conn != nil {
			for session := range conn.Sessions() {
				if session == nil {
					continue
				}
				if err := session.Close(); err != nil {
					log.Warn().Err(err).Str("sessionId", sessionID).Msg("failed to close upstream MCP server session")
				}
			}
		}

		notifier := ServerManagerInstance().Notifier()
		if notifier != nil {
			if ok := notifier.NotifyOnlineSessions(userID); !ok {
				log.Debug().Str("userId", userID).Msg("notify online sessions after session close returned false")
			}
		}

		internallog.GetLogService().EnqueueLog(database.Log{
			Action:    types.MCPEventLogTypeSessionClose,
			UserID:    userID,
			SessionID: sessionID,
			Error:     string(reason),
		})
	})
}

func snapshotServerCapabilities(serverContext *ServerContext) (mcp.ServerCapabilities, bool, bool) {
	if serverContext == nil {
		return mcp.ServerCapabilities{}, false, false
	}

	serverContext.mu.RLock()
	raw := map[string]any{}
	for k, v := range serverContext.Capabilities {
		raw[k] = v
	}
	hasPromptData := serverContext.Prompts != nil || serverContext.CachedPrompts != nil
	hasResourceData := serverContext.Resources != nil || serverContext.CachedResources != nil || serverContext.ResourceTemplates != nil || serverContext.CachedResourceTemplates != nil
	serverContext.mu.RUnlock()

	return parseServerCapabilities(raw), hasPromptData, hasResourceData
}

func parseServerCapabilities(raw map[string]any) mcp.ServerCapabilities {
	result := mcp.ServerCapabilities{}
	if raw == nil {
		return result
	}

	if promptsRaw, ok := raw["prompts"]; ok && promptsRaw != nil {
		promptCaps := &mcp.PromptCapabilities{}
		if prompts, ok := toAnyMap(promptsRaw); ok {
			promptCaps.ListChanged = toBool(prompts["listChanged"])
		}
		result.Prompts = promptCaps
	}

	if resourcesRaw, ok := raw["resources"]; ok && resourcesRaw != nil {
		resourceCaps := &mcp.ResourceCapabilities{}
		if resources, ok := toAnyMap(resourcesRaw); ok {
			resourceCaps.ListChanged = toBool(resources["listChanged"])
			resourceCaps.Subscribe = toBool(resources["subscribe"])
		}
		result.Resources = resourceCaps
	}

	if completionsRaw, ok := raw["completions"]; ok && completionsRaw != nil {
		result.Completions = &mcp.CompletionCapabilities{}
	}

	if loggingRaw, ok := raw["logging"]; ok && loggingRaw != nil {
		result.Logging = &mcp.LoggingCapabilities{}
	}

	if toolsRaw, ok := raw["tools"]; ok && toolsRaw != nil {
		toolCaps := &mcp.ToolCapabilities{}
		if tools, ok := toAnyMap(toolsRaw); ok {
			toolCaps.ListChanged = toBool(tools["listChanged"])
		}
		result.Tools = toolCaps
	}

	if experimentalRaw, ok := raw["experimental"]; ok && experimentalRaw != nil {
		if expMap, ok := toAnyMap(experimentalRaw); ok {
			result.Experimental = expMap
		}
	}

	return result
}

func toAnyMap(raw any) (map[string]any, bool) {
	if raw == nil {
		return nil, false
	}
	if typed, ok := raw.(map[string]any); ok {
		return typed, true
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, false
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

func toBool(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func emitToolsListChanged(conn *mcp.Server) {
	if conn == nil {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			log.Warn().Interface("error", rec).Msg("failed to emit tools/list_changed notification")
		}
	}()
	tmpName := "__kimbap_internal_notify_tool_" + strings.ReplaceAll(time.Now().Format("20060102150405.000000000"), ".", "")
	conn.AddTool(&mcp.Tool{
		Name:        tmpName,
		Description: "internal notification helper",
		InputSchema: map[string]any{"type": "object"},
	}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	})
	conn.RemoveTools(tmpName)
}

func emitResourcesListChanged(conn *mcp.Server) {
	if conn == nil {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			log.Warn().Interface("error", rec).Msg("failed to emit resources/list_changed notification")
		}
	}()
	tmpURI := "kimbap://internal/notify-resource-" + strings.ReplaceAll(time.Now().Format("20060102150405.000000000"), ".", "")
	tmpName := "__kimbap_internal_notify_resource_" + strings.ReplaceAll(time.Now().Format("20060102150405.000000000"), ".", "")
	conn.AddResource(&mcp.Resource{
		URI:      tmpURI,
		Name:     tmpName,
		MIMEType: "text/plain",
	}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{}, nil
	})
	conn.RemoveResources(tmpURI)
}

func emitPromptsListChanged(conn *mcp.Server) {
	if conn == nil {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			log.Warn().Interface("error", rec).Msg("failed to emit prompts/list_changed notification")
		}
	}()
	tmpName := "__kimbap_internal_notify_prompt_" + strings.ReplaceAll(time.Now().Format("20060102150405.000000000"), ".", "")
	conn.AddPrompt(&mcp.Prompt{
		Name:        tmpName,
		Description: "internal notification helper",
	}, func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{}, nil
	})
	conn.RemovePrompts(tmpName)
}

func resourceURIFromNotificationParams(params any) string {
	switch p := params.(type) {
	case *mcp.ResourceUpdatedNotificationParams:
		if p == nil {
			return ""
		}
		return p.URI
	case mcp.ResourceUpdatedNotificationParams:
		return p.URI
	case *mcp.ResourceUpdatedNotificationRequest:
		if p == nil || p.Params == nil {
			return ""
		}
		return p.Params.URI
	case mcp.ResourceUpdatedNotificationRequest:
		if p.Params == nil {
			return ""
		}
		return p.Params.URI
	case map[string]any:
		if v, ok := p["uri"].(string); ok {
			return v
		}
	}
	return ""
}

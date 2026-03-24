package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/dunialabs/kimbap-core/internal/security"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var nextServerContextID int64

type ServerContext struct {
	mu sync.RWMutex

	ID           string
	ServerID     string
	ServerEntity database.Server
	Status       int

	Capabilities       map[string]any
	CapabilitiesConfig mcptypes.ServerConfigCapabilities

	Tools             *mcp.ListToolsResult
	Resources         *mcp.ListResourcesResult
	ResourceTemplates *mcp.ListResourceTemplatesResult
	Prompts           *mcp.ListPromptsResult

	CachedTools             *mcp.ListToolsResult
	CachedResources         *mcp.ListResourcesResult
	CachedResourceTemplates *mcp.ListResourceTemplatesResult
	CachedPrompts           *mcp.ListPromptsResult

	LastSync           time.Time
	LastActive         int64
	activeRequests     atomic.Int64
	recoveryInProgress atomic.Bool

	Connection DownstreamClient
	Transport  TransportCloser
	mcpConn    mcp.Connection

	connectionMonitorCancel context.CancelFunc
	connectionMonitorSeq    uint64

	MaxTimeoutCount int
	TimeoutCount    int
	ErrorCount      int
	LastError       string

	UserID    string
	UserToken string

	RunnerMetadata *CustomStdioRunnerMetadata
	RunnerTrace    *RunnerExecutionTrace

	authStrategy       AuthStrategy
	tokenRefreshStop   chan struct{}
	tokenRefreshTimer  *time.Timer
	refreshCancel      context.CancelFunc
	currentAccessToken string
	currentTokenExpiry int64
	tokenRefreshSeq    uint64

	persistCapabilities func(ctx context.Context, serverID string, data map[string]any) error
}

type oauthConfigPersistenceStrategy interface {
	GetCurrentOAuthConfig() map[string]interface{}
	MarkConfigAsPersisted()
}

func NewServerContext(server database.Server) *ServerContext {
	id := atomic.AddInt64(&nextServerContextID, 1)

	sc := &ServerContext{
		ID:                 fmt.Sprintf("%d", id),
		ServerID:           server.ServerID,
		ServerEntity:       server,
		Status:             types.ServerStatusOffline,
		Capabilities:       map[string]any{},
		CapabilitiesConfig: mcptypes.ServerConfigCapabilities{Tools: map[string]mcptypes.ToolCapabilityConfig{}, Resources: map[string]mcptypes.ResourceCapabilityConfig{}, Prompts: map[string]mcptypes.PromptCapabilityConfig{}},
		LastSync:           time.Now(),
		LastActive:         time.Now().UnixMilli(),
		MaxTimeoutCount:    3,
		tokenRefreshStop:   make(chan struct{}),
	}

	sc.loadCapabilitiesConfig()
	sc.loadCachedCapabilities()

	return sc
}

func (s *ServerContext) loadCapabilitiesConfig() {
	if s.ServerEntity.Capabilities == "" {
		return
	}

	var config mcptypes.ServerConfigCapabilities
	if err := json.Unmarshal([]byte(s.ServerEntity.Capabilities), &config); err != nil {
		log.Error().Err(err).Str("serverId", s.ServerID).Msg("failed to parse server capability config")
		return
	}
	config.EnsureInitialized()
	s.CapabilitiesConfig = config
}

func (s *ServerContext) loadCachedCapabilities() {
	decode := func(raw *string, target any) {
		if raw == nil || *raw == "" {
			return
		}
		if err := json.Unmarshal([]byte(*raw), target); err != nil {
			log.Warn().Err(err).Str("serverId", s.ServerID).Msg("failed to parse cached capability payload")
		}
	}

	decode(s.ServerEntity.CachedTools, &s.CachedTools)
	decode(s.ServerEntity.CachedResources, &s.CachedResources)
	decode(s.ServerEntity.CachedResourceTemplates, &s.CachedResourceTemplates)
	decode(s.ServerEntity.CachedPrompts, &s.CachedPrompts)
}

func (s *ServerContext) Touch() {
	atomic.StoreInt64(&s.LastActive, time.Now().UnixMilli())
}

func (s *ServerContext) IsIdle(timeout time.Duration) bool {
	if s.activeRequests.Load() > 0 {
		return false
	}
	last := atomic.LoadInt64(&s.LastActive)
	return time.Since(time.UnixMilli(last)) > timeout
}

func (s *ServerContext) IncrementActiveRequests() {
	s.activeRequests.Add(1)
	s.Touch()
}

func (s *ServerContext) DecrementActiveRequests() {
	s.activeRequests.Add(-1)
	s.Touch()
}

func (s *ServerContext) UpdateStatus(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
}

func (s *ServerContext) UpdateCapabilities(caps map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Capabilities = caps
	s.LastSync = time.Now()
}

func (s *ServerContext) UpdateCapabilitiesConfig(raw string) error {
	var cfg mcptypes.ServerConfigCapabilities
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return err
	}
	cfg.EnsureInitialized()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ServerEntity.Capabilities = raw
	s.CapabilitiesConfig = cfg
	s.LastSync = time.Now()
	return nil
}

func (s *ServerContext) updateAndPersist(ctx context.Context, assignFn func(), key string, value any) error {
	s.mu.Lock()
	assignFn()
	s.LastSync = time.Now()
	hook := s.persistCapabilities
	s.mu.Unlock()

	if hook != nil && value != nil {
		return hook(ctx, s.ServerID, map[string]any{key: value})
	}
	return nil
}

func (s *ServerContext) UpdateTools(ctx context.Context, tools *mcp.ListToolsResult) error {
	return s.updateAndPersist(ctx, func() { s.Tools = tools }, "tools", tools)
}

func (s *ServerContext) UpdateResources(ctx context.Context, resources *mcp.ListResourcesResult) error {
	return s.updateAndPersist(ctx, func() { s.Resources = resources }, "resources", resources)
}

func (s *ServerContext) UpdateResourceTemplates(ctx context.Context, templates *mcp.ListResourceTemplatesResult) error {
	return s.updateAndPersist(ctx, func() { s.ResourceTemplates = templates }, "resourceTemplates", templates)
}

func (s *ServerContext) UpdatePrompts(ctx context.Context, prompts *mcp.ListPromptsResult) error {
	return s.updateAndPersist(ctx, func() { s.Prompts = prompts }, "prompts", prompts)
}

func (s *ServerContext) GetMCPCapabilities() mcptypes.ServerConfigWithEnabled {
	s.mu.RLock()
	serverEntity := s.ServerEntity
	capabilities := s.CapabilitiesConfig
	toolsResult := s.Tools
	resourcesResult := s.Resources
	resourceTemplatesResult := s.ResourceTemplates
	promptsResult := s.Prompts
	s.mu.RUnlock()

	tools := map[string]mcptypes.ToolCapabilityConfig{}
	resources := map[string]mcptypes.ResourceCapabilityConfig{}
	prompts := map[string]mcptypes.PromptCapabilityConfig{}

	if toolsResult != nil && toolsResult.Tools != nil {
		for _, tool := range toolsResult.Tools {
			if tool == nil {
				continue
			}
			cfg, exists := capabilities.Tools[tool.Name]
			enabled := true
			if exists {
				enabled = cfg.Enabled
			}
			dangerLevel := copyIntPtr(cfg.DangerLevel)
			if dangerLevel == nil {
				if tool.Annotations != nil && tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint {
					dangerLevel = intPtr(types.DangerLevelNotification)
				} else {
					dangerLevel = intPtr(types.DangerLevelSilent)
				}
			}
			tools[tool.Name] = mcptypes.ToolCapabilityConfig{
				Enabled:     enabled,
				Description: tool.Description,
				DangerLevel: dangerLevel,
			}
		}
	} else {
		for name, cfg := range capabilities.Tools {
			tools[name] = mcptypes.ToolCapabilityConfig{
				Enabled:     cfg.Enabled,
				Description: cfg.Description,
				DangerLevel: copyIntPtr(cfg.DangerLevel),
			}
		}
	}

	if resourcesResult != nil && resourcesResult.Resources != nil {
		for _, resource := range resourcesResult.Resources {
			if resource == nil {
				continue
			}
			cfg, exists := capabilities.Resources[resource.Name]
			enabled := true
			if exists {
				enabled = cfg.Enabled
			}
			resources[resource.Name] = mcptypes.ResourceCapabilityConfig{
				Enabled:     enabled,
				Description: resource.Description,
			}
		}

		if resourceTemplatesResult != nil && resourceTemplatesResult.ResourceTemplates != nil {
			for _, resourceTemplate := range resourceTemplatesResult.ResourceTemplates {
				if resourceTemplate == nil {
					continue
				}
				cfg, exists := capabilities.Resources[resourceTemplate.Name]
				enabled := true
				if exists {
					enabled = cfg.Enabled
				}
				resources[resourceTemplate.Name] = mcptypes.ResourceCapabilityConfig{
					Enabled:     enabled,
					Description: resourceTemplate.Description,
				}
			}
		}
	} else {
		for name, cfg := range capabilities.Resources {
			resources[name] = cfg
		}
	}

	if promptsResult != nil && promptsResult.Prompts != nil {
		for _, prompt := range promptsResult.Prompts {
			if prompt == nil {
				continue
			}
			cfg, exists := capabilities.Prompts[prompt.Name]
			enabled := true
			if exists {
				enabled = cfg.Enabled
			}
			prompts[prompt.Name] = mcptypes.PromptCapabilityConfig{
				Enabled:     enabled,
				Description: prompt.Description,
			}
		}
	} else {
		for name, cfg := range capabilities.Prompts {
			prompts[name] = cfg
		}
	}

	configTemplate := ""
	if serverEntity.ConfigTemplate != nil {
		configTemplate = *serverEntity.ConfigTemplate
	}

	result := mcptypes.ServerConfigWithEnabled{
		Enabled:        serverEntity.Enabled,
		ServerName:     serverEntity.ServerName,
		AllowUserInput: serverEntity.AllowUserInput,
		AuthType:       serverEntity.AuthType,
		ConfigTemplate: configTemplate,
		Configured:     true,
		Tools:          tools,
		Resources:      resources,
		Prompts:        prompts,
	}
	result.Category = intPtr(serverEntity.Category)
	return result
}

func (s *ServerContext) IsCapabilityChanged(newConfig mcptypes.ServerConfigCapabilities) (toolsChanged, resourcesChanged, promptsChanged bool) {
	s.mu.RLock()
	current := s.CapabilitiesConfig
	s.mu.RUnlock()

	oldTools := current.Tools
	if oldTools == nil {
		oldTools = map[string]mcptypes.ToolCapabilityConfig{}
	}
	oldResources := current.Resources
	if oldResources == nil {
		oldResources = map[string]mcptypes.ResourceCapabilityConfig{}
	}
	oldPrompts := current.Prompts
	if oldPrompts == nil {
		oldPrompts = map[string]mcptypes.PromptCapabilityConfig{}
	}

	newTools := newConfig.Tools
	if newTools == nil {
		newTools = map[string]mcptypes.ToolCapabilityConfig{}
	}
	newResources := newConfig.Resources
	if newResources == nil {
		newResources = map[string]mcptypes.ResourceCapabilityConfig{}
	}
	newPrompts := newConfig.Prompts
	if newPrompts == nil {
		newPrompts = map[string]mcptypes.PromptCapabilityConfig{}
	}

	return isToolsChanged(oldTools, newTools), isResourcesChanged(oldResources, newResources), isPromptsChanged(oldPrompts, newPrompts)
}

func isToolsChanged(old, new map[string]mcptypes.ToolCapabilityConfig) bool {
	for name, newCfg := range new {
		oldCfg, existed := old[name]
		if newCfg.Enabled {
			if !existed || !oldCfg.Enabled || !dangerLevelEqual(oldCfg.DangerLevel, newCfg.DangerLevel) {
				return true
			}
		} else if existed && oldCfg.Enabled {
			// Item flipped from enabled → disabled
			return true
		} else if !dangerLevelEqual(oldCfg.DangerLevel, newCfg.DangerLevel) {
			// Both disabled (or new+disabled), but dangerLevel changed
			return true
		}
	}
	for name, oldCfg := range old {
		if oldCfg.Enabled {
			if _, exists := new[name]; !exists {
				return true
			}
		}
	}
	return false
}

func dangerLevelEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func isResourcesChanged(old, new map[string]mcptypes.ResourceCapabilityConfig) bool {
	for name, newCfg := range new {
		oldCfg, existed := old[name]
		if newCfg.Enabled {
			if !existed || !oldCfg.Enabled {
				return true
			}
		} else if existed && oldCfg.Enabled {
			return true
		}
	}
	for name, oldCfg := range old {
		if oldCfg.Enabled {
			if _, exists := new[name]; !exists {
				return true
			}
		}
	}
	return false
}

func isPromptsChanged(old, new map[string]mcptypes.PromptCapabilityConfig) bool {
	for name, newCfg := range new {
		oldCfg, existed := old[name]
		if newCfg.Enabled {
			if !existed || !oldCfg.Enabled {
				return true
			}
		} else if existed && oldCfg.Enabled {
			return true
		}
	}
	for name, oldCfg := range old {
		if oldCfg.Enabled {
			if _, exists := new[name]; !exists {
				return true
			}
		}
	}
	return false
}

func (s *ServerContext) RecordTimeout(err error) bool {
	result := s.RecordTimeoutWithRecovery(context.Background(), err)
	return result != nil && *result
}

func (s *ServerContext) RecordTimeoutWithRecovery(ctx context.Context, err error) *bool {
	if !isTimeoutError(err) {
		return nil
	}

	s.mu.Lock()
	s.TimeoutCount++
	timeoutCount := s.TimeoutCount
	maxTimeoutCount := s.MaxTimeoutCount
	s.mu.Unlock()

	if maxTimeoutCount <= 0 {
		maxTimeoutCount = 1
	}
	if timeoutCount < maxTimeoutCount {
		return boolPtr(false)
	}

	if !s.recoveryInProgress.CompareAndSwap(false, true) {
		return boolPtr(false)
	}
	defer s.recoveryInProgress.Store(false)

	conn := s.ConnectionSnapshot()
	if conn == nil {
		return boolPtr(false)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	pingCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
	pingErr := conn.Ping(pingCtx, &mcp.PingParams{})
	cancel()
	if pingErr == nil {
		s.ClearTimeout()
		return boolPtr(false)
	}
	if !isTimeoutError(pingErr) {
		return boolPtr(false)
	}

	s.ClearTimeout()
	ServerManagerInstance().reconnectServerAsync(s)
	return boolPtr(true)
}

func (s *ServerContext) ClearTimeout() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TimeoutCount = 0
}

func (s *ServerContext) RecordError(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount++
	s.LastError = err
}

func (s *ServerContext) StartTokenRefresh(ctx context.Context, strategy AuthStrategy) (string, error) {
	token, expiresAt, err := strategy.GetInitialToken(ctx)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	select {
	case <-s.tokenRefreshStop:
		s.tokenRefreshStop = make(chan struct{})
	default:
	}
	if s.refreshCancel != nil {
		s.refreshCancel()
	}
	refreshCtx, cancel := context.WithCancel(context.Background())
	s.refreshCancel = cancel
	s.authStrategy = strategy
	s.currentAccessToken = token
	s.currentTokenExpiry = expiresAt
	s.tokenRefreshSeq++
	s.scheduleNextRefreshLocked(refreshCtx)
	s.mu.Unlock()

	// Persist OAuth config after initial token fetch
	if persistenceStrategy, ok := strategy.(oauthConfigPersistenceStrategy); ok {
		oauthConfig := persistenceStrategy.GetCurrentOAuthConfig()
		if oauthConfig != nil {
			if err := s.updateRefreshTokenToDatabase(oauthConfig); err != nil {
				log.Warn().Err(err).Str("serverId", s.ServerID).Msg("failed to persist initial OAuth config")
			} else {
				persistenceStrategy.MarkConfigAsPersisted()
			}
		}
	}

	return token, nil
}

func (s *ServerContext) scheduleNextRefreshLocked(refreshCtx context.Context) {
	if s.currentTokenExpiry == 0 || s.authStrategy == nil {
		return
	}

	refreshAt := time.UnixMilli(s.currentTokenExpiry).Add(-5 * time.Minute)
	delay := time.Until(refreshAt)
	if delay < 10*time.Second {
		delay = 10 * time.Second
	}

	if s.tokenRefreshTimer != nil {
		s.tokenRefreshTimer.Stop()
	}
	s.tokenRefreshTimer = time.AfterFunc(delay, func() {
		if refreshCtx.Err() != nil {
			return
		}
		s.performTokenRefresh(refreshCtx)
	})
}

func (s *ServerContext) performTokenRefresh(ctx context.Context) {
	if ctx != nil && ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	strategy := s.authStrategy
	seq := s.tokenRefreshSeq
	s.mu.Unlock()
	if strategy == nil {
		return
	}
	if ctx != nil && ctx.Err() != nil {
		return
	}

	token, expiresAt, err := strategy.RefreshToken(ctx)
	if err != nil {
		errorMessage := err.Error()
		if isOAuthRefreshAuthError(err) {
			log.Error().Err(err).Str("serverId", s.ServerID).Msg("oauth authentication failed, stopping token refresh")
			s.RecordError(errorMessage)
			s.StopTokenRefresh()
			return
		}

		log.Error().Err(err).Str("serverId", s.ServerID).Msg("token refresh failed, scheduling retry")
		s.RecordError(errorMessage)

		s.mu.Lock()
		if s.authStrategy != nil {
			s.scheduleRefreshRetryLocked(ctx, 3*time.Minute)
		}
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	if s.authStrategy == nil || s.tokenRefreshSeq != seq {
		s.mu.Unlock()
		return
	}
	s.currentAccessToken = token
	s.currentTokenExpiry = expiresAt
	s.mu.Unlock()

	if persistenceStrategy, ok := strategy.(oauthConfigPersistenceStrategy); ok {
		oauthConfig := persistenceStrategy.GetCurrentOAuthConfig()
		if oauthConfig != nil {
			if err := s.updateRefreshTokenToDatabase(oauthConfig); err != nil {
				log.Error().Err(err).Str("serverId", s.ServerID).Str("userId", s.UserIDSnapshot()).Msg("failed to persist refreshed oauth config")
			} else {
				persistenceStrategy.MarkConfigAsPersisted()
			}
		}
	}

	s.notifyTokenUpdate(ctx, token)

	s.mu.Lock()
	if s.authStrategy != nil && s.tokenRefreshSeq == seq {
		s.scheduleNextRefreshLocked(ctx)
	}
	s.mu.Unlock()
}

func (s *ServerContext) scheduleRefreshRetryLocked(refreshCtx context.Context, delay time.Duration) {
	if s.authStrategy == nil {
		return
	}
	if delay <= 0 {
		delay = 10 * time.Second
	}
	if s.tokenRefreshTimer != nil {
		s.tokenRefreshTimer.Stop()
	}
	s.tokenRefreshTimer = time.AfterFunc(delay, func() {
		if refreshCtx != nil && refreshCtx.Err() != nil {
			return
		}
		s.performTokenRefresh(refreshCtx)
	})
}

func (s *ServerContext) notifyTokenUpdate(ctx context.Context, token string) {
	if token == "" {
		return
	}

	params := map[string]any{
		"token":     token,
		"timestamp": time.Now().UnixMilli(),
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Error().Err(err).Str("serverId", s.ServerID).Msg("failed to marshal token update params")
		return
	}

	s.mu.RLock()
	conn := s.mcpConn
	s.mu.RUnlock()

	if conn == nil {
		log.Debug().Str("serverId", s.ServerID).Msg("no mcp connection available for token update notification")
		return
	}

	notification := &jsonrpc.Request{
		Method: "notifications/token/update",
		Params: paramsJSON,
	}

	if err := conn.Write(ctx, notification); err != nil {
		log.Warn().Err(err).Str("serverId", s.ServerID).Msg("failed to send token update notification")
		return
	}

	log.Debug().Str("serverId", s.ServerID).Msg("sent token update notification to downstream")
}

func (s *ServerContext) updateRefreshTokenToDatabase(oauthConfig map[string]interface{}) error {
	if oauthConfig == nil {
		return nil
	}

	s.mu.RLock()
	serverEntity := s.ServerEntity
	userID := s.UserID
	userToken := s.UserToken
	s.mu.RUnlock()

	if serverEntity.UseKimbapOauthConfig {
		return nil
	}
	if userToken == "" {
		return nil
	}

	if userID != "" {
		return s.updateUserLaunchConfigOAuth(serverEntity.ServerID, userID, userToken, oauthConfig)
	}
	return s.updateServerLaunchConfigOAuth(serverEntity, userToken, oauthConfig)
}

func (s *ServerContext) updateServerLaunchConfigOAuth(serverEntity database.Server, userToken string, oauthConfig map[string]interface{}) error {
	launchConfig, encryptedLaunchConfig, err := extractLaunchConfig(serverEntity.LaunchConfig, userToken)
	if err != nil {
		return err
	}
	launchConfig["oauth"] = mergeOAuthConfig(launchConfig["oauth"], oauthConfig)

	updated, err := security.EncryptData(string(mustJSON(launchConfig)), userToken)
	if err != nil {
		return err
	}
	if updated == encryptedLaunchConfig {
		return nil
	}

	if _, err := repository.NewServerRepository(nil).UpdateLaunchConfig(serverEntity.ServerID, updated); err != nil {
		return err
	}

	s.mu.Lock()
	s.ServerEntity.LaunchConfig = updated
	s.mu.Unlock()

	return nil
}

func (s *ServerContext) updateUserLaunchConfigOAuth(serverID string, userID string, userToken string, oauthConfig map[string]interface{}) error {
	db := database.DB
	if db == nil {
		return errors.New("database unavailable")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var user database.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&user).Error; err != nil {
			return err
		}

		launchConfigs := map[string]any{}
		if user.LaunchConfigs != "" {
			if err := json.Unmarshal([]byte(user.LaunchConfigs), &launchConfigs); err != nil {
				return err
			}
		}

		encryptedVal, ok := launchConfigs[serverID]
		if !ok {
			return nil
		}
		encryptedStr, err := security.EncryptedAnyToString(encryptedVal)
		if err != nil || encryptedStr == "" {
			return nil
		}

		launchConfig, originalEncrypted, err := extractLaunchConfig(encryptedStr, userToken)
		if err != nil {
			return err
		}
		launchConfig["oauth"] = mergeOAuthConfig(launchConfig["oauth"], oauthConfig)

		updatedObj, err := security.EncryptDataToObject(string(mustJSON(launchConfig)), userToken)
		if err != nil {
			return err
		}
		updatedStr, _ := security.EncryptedAnyToString(updatedObj)
		if updatedStr == originalEncrypted {
			return nil
		}

		launchConfigs[serverID] = updatedObj
		launchConfigsJSON, jsonErr := json.Marshal(launchConfigs)
		if jsonErr != nil {
			return jsonErr
		}
		if err := tx.Model(&database.User{}).Where("user_id = ?", userID).Updates(map[string]any{"launch_configs": string(launchConfigsJSON), "updated_at": int(time.Now().Unix())}).Error; err != nil {
			return err
		}

		if store := SessionStoreInstance(); store != nil {
			userSessions := store.GetUserSessions(userID)
			for _, session := range userSessions {
				session.UpdateLaunchConfigs(string(launchConfigsJSON))
			}
			if len(userSessions) > 0 {
				log.Debug().
					Str("serverID", serverID).
					Str("userID", userID).
					Int("sessionCount", len(userSessions)).
					Msg("synced launchConfigs to user sessions")
			}
		}

		s.mu.Lock()
		s.ServerEntity.LaunchConfig = updatedStr
		s.mu.Unlock()

		return nil
	})
}

func extractLaunchConfig(encryptedLaunchConfig string, userToken string) (map[string]interface{}, string, error) {
	decrypted, err := security.DecryptDataFromString(encryptedLaunchConfig, userToken)
	if err != nil {
		return nil, "", err
	}
	launchConfig := map[string]interface{}{}
	if err := json.Unmarshal([]byte(decrypted), &launchConfig); err != nil {
		return nil, "", err
	}
	if launchConfig == nil {
		launchConfig = map[string]interface{}{}
	}
	return launchConfig, encryptedLaunchConfig, nil
}

func mergeOAuthConfig(existingOAuth any, updatedOAuth map[string]interface{}) map[string]interface{} {
	merged := map[string]interface{}{}
	if current, ok := existingOAuth.(map[string]interface{}); ok {
		for key, value := range current {
			merged[key] = value
		}
	}
	for key, value := range updatedOAuth {
		merged[key] = value
	}
	return merged
}

func isOAuthRefreshAuthError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "401") ||
		strings.Contains(message, "400") ||
		strings.Contains(message, "unauthorized") ||
		strings.Contains(message, "bad request")
}

func (s *ServerContext) StopTokenRefresh() {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.tokenRefreshStop:
	default:
		close(s.tokenRefreshStop)
	}

	if s.tokenRefreshTimer != nil {
		s.tokenRefreshTimer.Stop()
		s.tokenRefreshTimer = nil
	}
	if s.refreshCancel != nil {
		s.refreshCancel()
		s.refreshCancel = nil
	}

	s.authStrategy = nil
	s.currentAccessToken = ""
	s.currentTokenExpiry = 0
}

func (s *ServerContext) CloseConnection(status int) error {
	s.mu.Lock()
	if s.connectionMonitorCancel != nil {
		s.connectionMonitorCancel()
		s.connectionMonitorCancel = nil
	}
	conn := s.Connection
	transport := s.Transport
	s.Connection = nil
	s.Transport = nil
	s.mcpConn = nil
	s.Status = status
	s.mu.Unlock()

	if transport != nil {
		if sessionTerminator, ok := transport.(interface{ TerminateSession() error }); ok {
			if err := sessionTerminator.TerminateSession(); err != nil {
				_ = transport.Close()
				if conn != nil {
					_ = conn.Close()
				}
				return err
			}
		}
		_ = transport.Close()
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (s *ServerContext) SetCapabilitiesPersistHook(hook func(ctx context.Context, serverID string, data map[string]any) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistCapabilities = hook
}

func (s *ServerContext) GetDangerLevel(toolName string) *int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if cfg, ok := s.CapabilitiesConfig.Tools[toolName]; ok {
		return cfg.DangerLevel
	}
	return nil
}

func (s *ServerContext) ConnectionSnapshot() DownstreamClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Connection
}

func (s *ServerContext) SetConnection(conn DownstreamClient, transport TransportCloser, mcpConn mcp.Connection) (context.Context, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connectionMonitorCancel != nil {
		s.connectionMonitorCancel()
		s.connectionMonitorCancel = nil
	}

	monitorCtx, cancel := context.WithCancel(context.Background())
	s.connectionMonitorCancel = cancel
	s.connectionMonitorSeq++

	s.Connection = conn
	s.Transport = transport
	s.mcpConn = mcpConn

	return monitorCtx, s.connectionMonitorSeq
}

func (s *ServerContext) StatusSnapshot() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

func (s *ServerContext) IsCurrentConnection(conn DownstreamClient, seq uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Connection == conn && s.connectionMonitorSeq == seq
}

func (s *ServerContext) UserTokenSnapshot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.UserToken
}

func (s *ServerContext) UserIDSnapshot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.UserID
}

func (s *ServerContext) CapabilitiesConfigSnapshot() mcptypes.ServerConfigCapabilities {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := map[string]mcptypes.ToolCapabilityConfig{}
	resources := map[string]mcptypes.ResourceCapabilityConfig{}
	prompts := map[string]mcptypes.PromptCapabilityConfig{}

	for name, cfg := range s.CapabilitiesConfig.Tools {
		tools[name] = cfg
	}
	for name, cfg := range s.CapabilitiesConfig.Resources {
		resources[name] = cfg
	}
	for name, cfg := range s.CapabilitiesConfig.Prompts {
		prompts[name] = cfg
	}

	return mcptypes.ServerConfigCapabilities{
		Tools:     tools,
		Resources: resources,
		Prompts:   prompts,
	}
}

func (s *ServerContext) CapabilityChanged() (toolsChanged, resourcesChanged, promptsChanged bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	toolsChanged = s.Tools != nil && len(s.Tools.Tools) > 0
	resourcesChanged = s.Resources != nil && len(s.Resources.Resources) > 0
	promptsChanged = s.Prompts != nil && len(s.Prompts.Prompts) > 0
	return
}

func (s *ServerContext) GetToolDescription(toolName string) string {
	s.mu.RLock()
	tools := s.Tools
	cfg, cfgExists := s.CapabilitiesConfig.Tools[toolName]
	s.mu.RUnlock()

	if description, ok := findToolDescription(tools, toolName); ok {
		return description
	}

	if cfgExists {
		return cfg.Description
	}
	return ""
}

func (s *ServerContext) ResolveResourceNameByURI(uri string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Resources == nil {
		return uri
	}
	for _, r := range s.Resources.Resources {
		if r != nil && string(r.URI) == uri {
			return r.Name
		}
	}
	return uri
}

func findToolDescription(tools *mcp.ListToolsResult, toolName string) (string, bool) {
	if tools == nil {
		return "", false
	}
	for _, tool := range tools.Tools {
		if tool == nil {
			continue
		}
		if tool.Name == toolName {
			return tool.Description, true
		}
	}
	return "", false
}

func (s *ServerContext) ServerEntitySnapshot() database.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ServerEntity
}

func (s *ServerContext) UpdateServerEntity(entity database.Server) {
	s.mu.Lock()
	s.ServerEntity = entity
	s.mu.Unlock()
}

func copyIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	value := *v
	return &value
}

func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

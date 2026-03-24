package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	serverlog "github.com/dunialabs/kimbap-core/internal/log"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/security"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type subscriptionState struct {
	SubscribedSessions           map[string]struct{}
	DownstreamSubscribed         bool
	DownstreamSubscribing        bool
	DownstreamUnsubscribePending bool
}

type capabilityRefreshResult struct {
	context          *ServerContext
	toolsChanged     bool
	resourcesChanged bool
	promptsChanged   bool
}

type serverManager struct {
	mu sync.RWMutex

	serverContexts   map[string]*ServerContext
	temporaryServers map[string]*ServerContext
	serverLoggers    map[string]*serverlog.ServerLogger

	resourceSubscriptions map[string]*subscriptionState

	repo             ServerRepository
	users            UserRepository
	authFactory      AuthStrategyFactory
	transportFactory *DownstreamTransportFactory
	notifier         SocketNotifier

	ownerToken string

	startupGroup     singleflight.Group
	serverConnecting sync.Map

	idleTicker   *time.Ticker
	idleStop     chan struct{}
	shutdownOnce sync.Once
	shuttingDown atomic.Bool

	// Reconciliation loop state
	reconcileTicker *time.Ticker
	reconcileStop   chan struct{}
	reconcileDone   chan struct{} // closed when first reconcile completes
	reconcileOnce   *sync.Once    // ensures reconcileDone is closed exactly once
}

var (
	serverManagerInst *serverManager
	serverManagerOnce sync.Once
)

func ServerManagerInstance() *serverManager {
	serverManagerOnce.Do(func() {
		serverManagerInst = &serverManager{
			serverContexts:        map[string]*ServerContext{},
			temporaryServers:      map[string]*ServerContext{},
			serverLoggers:         map[string]*serverlog.ServerLogger{},
			resourceSubscriptions: map[string]*subscriptionState{},
			transportFactory:      NewDownstreamTransportFactory(),
		}
		serverManagerInst.startIdleChecker()
	})
	return serverManagerInst
}

func (m *serverManager) Configure(repo ServerRepository, users UserRepository, authFactory AuthStrategyFactory, notifier SocketNotifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.repo = repo
	m.users = users
	m.authFactory = authFactory
	m.notifier = notifier
}

func (m *serverManager) Notifier() SocketNotifier {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.notifier
}

func (m *serverManager) UserRepository() UserRepository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.users
}

func (m *serverManager) Repository() ServerRepository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.repo
}

func (m *serverManager) NotifyUserPermissionChangedByServer(serverID string) {
	m.mu.RLock()
	notifier := m.notifier
	m.mu.RUnlock()
	if notifier == nil {
		return
	}
	notifier.NotifyUserPermissionChangedByServer(serverID)
}

func (m *serverManager) NotifyUserPermissionChanged(userID string) {
	m.mu.RLock()
	notifier := m.notifier
	m.mu.RUnlock()
	if notifier == nil {
		return
	}
	notifier.NotifyUserPermissionChanged(userID)
}

func (m *serverManager) updateServerStatus(sc *ServerContext, newStatus int) {
	if sc == nil {
		return
	}
	oldStatus := sc.StatusSnapshot()
	if oldStatus == newStatus {
		return
	}
	sc.UpdateStatus(newStatus)

	m.mu.RLock()
	notifier := m.notifier
	m.mu.RUnlock()
	if notifier == nil {
		return
	}
	serverEntity := sc.ServerEntitySnapshot()
	go notifier.NotifyServerStatusChanged(sc.ServerID, serverEntity.ServerName, oldStatus, newStatus)
}

func (m *serverManager) SetOwnerToken(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ownerToken = token
}

func (m *serverManager) GetOwnerToken() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if strings.TrimSpace(m.ownerToken) == "" {
		return "", errors.New("Owner token not available. Please ensure Owner has accessed the API at least once.")
	}
	return m.ownerToken, nil
}

func (m *serverManager) AddServer(ctx context.Context, server database.Server, token string) (*ServerContext, error) {
	result, err, _ := m.startupGroup.Do(server.ServerID, func() (interface{}, error) {
		var contextObj *ServerContext

		m.mu.RLock()
		existing := m.serverContexts[server.ServerID]
		m.mu.RUnlock()

		if existing != nil {
			existingEntity := existing.ServerEntitySnapshot()
			status := existing.StatusSnapshot()

			switch {
			case existingEntity.LaunchConfig != server.LaunchConfig:
				if _, err := m.RemoveServer(ctx, server.ServerID); err != nil {
					return nil, fmt.Errorf("failed to remove server before re-add: %w", err)
				}
			case status == types.ServerStatusOnline || status == types.ServerStatusConnecting:
				return existing, nil
			case status == types.ServerStatusSleeping:
				existing.mu.Lock()
				existing.ServerEntity = server
				existing.mu.Unlock()
				m.attachCapabilitiesPersistHook(existing)
				contextObj = existing
			default:
				if _, err := m.RemoveServer(ctx, server.ServerID); err != nil {
					return nil, fmt.Errorf("failed to remove server before re-add: %w", err)
				}
			}
		}

		connectDone := make(chan struct{})
		m.serverConnecting.Store(server.ServerID, connectDone)

		if contextObj == nil {
			contextObj = NewServerContext(server)
			m.attachCapabilitiesPersistHook(contextObj)
			m.mu.Lock()
			m.serverContexts[server.ServerID] = contextObj
			m.mu.Unlock()
		}

		m.mu.Lock()
		if _, ok := m.serverLoggers[server.ServerID]; !ok {
			m.serverLoggers[server.ServerID] = serverlog.NewServerLogger(server.ServerID)
		}
		m.mu.Unlock()

		defer func() {
			close(connectDone)
			m.serverConnecting.Delete(server.ServerID)
		}()
		err := m.createServerConnection(ctx, contextObj, token)
		if err != nil {
			return nil, err
		}
		return contextObj, nil
	})
	if err != nil {
		return nil, err
	}
	contextObj, ok := result.(*ServerContext)
	if !ok {
		return nil, errors.New("unexpected result type from server init")
	}
	m.logServerLifecycle(server.ServerID, types.MCPEventLogTypeServerInit, "")
	return contextObj, nil
}

func (m *serverManager) RemoveServer(ctx context.Context, serverID string) (*ServerContext, error) {
	if ch, loaded := m.serverConnecting.Load(serverID); loaded {
		waitCh, ok := ch.(chan struct{})
		if !ok {
			return nil, errors.New("invalid server connecting channel type")
		}
		select {
		case <-waitCh:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(30 * time.Second):
			return nil, errors.New("timed out waiting for server connection to complete")
		}
	}

	m.mu.Lock()
	contextObj := m.serverContexts[serverID]
	serverLogger := m.serverLoggers[serverID]
	delete(m.serverContexts, serverID)
	delete(m.serverLoggers, serverID)
	m.mu.Unlock()

	if contextObj == nil {
		return nil, nil
	}
	if serverLogger != nil {
		serverLogger.LogServerLifecycle(types.MCPEventLogTypeServerClose, "")
	}
	oldStatus := contextObj.StatusSnapshot()
	contextObj.StopTokenRefresh()
	if err := contextObj.CloseConnection(types.ServerStatusOffline); err != nil {
		// Close errors are caught so cleanup always completes
		log.Error().Err(err).Str("serverID", serverID).Msg("error closing server connection during removal")
	}
	if oldStatus != types.ServerStatusOffline {
		m.mu.RLock()
		notifier := m.notifier
		m.mu.RUnlock()
		if notifier != nil {
			serverEntity := contextObj.ServerEntitySnapshot()
			go notifier.NotifyServerStatusChanged(contextObj.ServerID, serverEntity.ServerName, oldStatus, types.ServerStatusOffline)
		}
	}
	return contextObj, nil
}

func (m *serverManager) ReconnectServer(ctx context.Context, server database.Server, token string) (*ServerContext, error) {
	if _, err := m.RemoveServer(ctx, server.ServerID); err != nil {
		return nil, fmt.Errorf("failed to remove server before reconnect: %w", err)
	}
	return m.AddServer(ctx, server, token)
}

func (m *serverManager) EnsureServerAvailable(ctx context.Context, serverID string, userID string) (*ServerContext, error) {
	contextObj := m.GetServerContext(serverID, userID)
	if contextObj == nil {
		return nil, fmt.Errorf("server %s not found", serverID)
	}

	status := contextObj.StatusSnapshot()

	if status == types.ServerStatusOnline {
		contextObj.Touch()
		return contextObj, nil
	}
	if status == types.ServerStatusConnecting {
		if err := m.waitForServerReady(ctx, contextObj, 50*time.Second); err != nil {
			return nil, err
		}
		contextObj.Touch()
		return contextObj, nil
	}
	if status == types.ServerStatusError {
		contextObj.mu.RLock()
		lastError := strings.TrimSpace(contextObj.LastError)
		contextObj.mu.RUnlock()
		if lastError == "" {
			lastError = "Unknown error"
		}
		return nil, fmt.Errorf("server %s is in error state: %s", serverID, lastError)
	}

	allowUserInput := contextObj.ServerEntitySnapshot().AllowUserInput

	key := serverID
	if allowUserInput {
		key = tempServerKey(serverID, userID)
	}

	_, err, _ := m.startupGroup.Do(key, func() (any, error) {
		return nil, m.wakeupServer(ctx, contextObj, userID)
	})
	if err != nil {
		return nil, err
	}
	if contextObj.StatusSnapshot() == types.ServerStatusConnecting {
		if err := m.waitForServerReady(ctx, contextObj, 50*time.Second); err != nil {
			return nil, err
		}
	}

	contextObj.Touch()
	return contextObj, nil
}

func (m *serverManager) waitForServerReady(ctx context.Context, contextObj *ServerContext, timeout time.Duration) error {
	if contextObj == nil {
		return errors.New("server context is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = 50 * time.Second
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		switch contextObj.StatusSnapshot() {
		case types.ServerStatusOnline:
			return nil
		case types.ServerStatusError:
			contextObj.mu.RLock()
			lastError := contextObj.LastError
			contextObj.mu.RUnlock()
			if strings.TrimSpace(lastError) == "" {
				lastError = "unknown error"
			}
			return fmt.Errorf("server %s failed to start: %s", contextObj.ServerID, lastError)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutTimer.C:
			return fmt.Errorf("server %s startup timeout after %s", contextObj.ServerID, timeout)
		case <-ticker.C:
		}
	}
}

func (m *serverManager) wakeupServer(ctx context.Context, contextObj *ServerContext, userID string) error {
	contextObj.mu.Lock()
	serverEntity := contextObj.ServerEntity
	allowUserInput := serverEntity.AllowUserInput
	serverID := contextObj.ServerID
	contextObj.mu.Unlock()

	if !m.isLazyStartApplicable(serverEntity) {
		return errors.New("lazy start not supported")
	}
	token, err := m.GetOwnerToken()
	if err != nil {
		return err
	}
	if allowUserInput {
		session := SessionStoreInstance().GetUserFirstSession(userID)
		if session == nil {
			return errors.New("user has no active session")
		}
		token = session.Token
	}

	connectDone := make(chan struct{})
	m.serverConnecting.Store(serverID, connectDone)
	connectErr := m.createServerConnection(ctx, contextObj, token)
	close(connectDone)
	m.serverConnecting.Delete(serverID)

	if connectErr != nil {
		return connectErr
	}
	m.logServerLifecycle(m.loggerKeyForContext(contextObj), types.MCPEventLogTypeServerInit, "")
	return nil
}

func (m *serverManager) createServerConnection(ctx context.Context, contextObj *ServerContext, token string) error {
	status := contextObj.StatusSnapshot()
	if status == types.ServerStatusOnline || status == types.ServerStatusConnecting {
		return nil
	}
	contextObj.UpdateStatus(types.ServerStatusConnecting)

	contextObj.mu.Lock()
	contextObj.UserToken = token
	serverEntity := contextObj.ServerEntity
	serverUserID := contextObj.UserID
	contextObj.mu.Unlock()

	fail := func(retErr error, record string) error {
		m.updateServerStatus(contextObj, types.ServerStatusError)
		contextObj.RecordError(record)
		return retErr
	}
	failStop := func(retErr error, record string) error {
		contextObj.StopTokenRefresh()
		return fail(retErr, record)
	}

	launchConfigRaw, err := m.decryptLaunchConfig(token, serverEntity)
	if err != nil {
		return fail(err, err.Error())
	}

	launchConfig := map[string]any{}
	if err := json.Unmarshal([]byte(launchConfigRaw), &launchConfig); err != nil {
		return fail(err, err.Error())
	}
	if launchConfig == nil {
		launchConfig = map[string]any{}
	}

	if err := m.initializeAuthentication(ctx, contextObj, launchConfig, token); err != nil {
		return fail(err, err.Error())
	}

	switch serverEntity.Category {
	case types.ServerCategoryRestAPI:
		cfgRaw := ""
		if serverEntity.ConfigTemplate != nil {
			cfgRaw = *serverEntity.ConfigTemplate
		}
		trimmedConfigTemplate := strings.TrimSpace(cfgRaw)
		if trimmedConfigTemplate == "" || trimmedConfigTemplate == "{}" || trimmedConfigTemplate == "null" {
			e := fmt.Errorf("[ServerManager] Missing configTemplate for server %s", serverEntity.ServerID)
			return failStop(e, e.Error())
		}

		cfg := map[string]any{}
		if err := json.Unmarshal([]byte(trimmedConfigTemplate), &cfg); err != nil {
			return failStop(fmt.Errorf("invalid configTemplate for server %s: %w", serverEntity.ServerID, err), err.Error())
		}
		if cfg == nil {
			e := fmt.Errorf("[ServerManager] Missing configTemplate for server %s", serverEntity.ServerID)
			return failStop(e, e.Error())
		}
		apis, ok := cfg["apis"].([]any)
		if !ok || len(apis) == 0 {
			e := fmt.Errorf("invalid configTemplate for server %s: missing api entry", serverEntity.ServerID)
			return failStop(e, e.Error())
		}
		first, ok := apis[0].(map[string]any)
		if !ok {
			e := fmt.Errorf("invalid configTemplate for server %s: invalid first api entry", serverEntity.ServerID)
			return failStop(e, e.Error())
		}
		authConfig, hasAuth := launchConfig["auth"]
		if !hasAuth || authConfig == nil {
			e := fmt.Errorf("invalid launchConfig for server %s: missing auth", serverEntity.ServerID)
			return failStop(e, e.Error())
		}
		authMap, ok := authConfig.(map[string]any)
		if !ok || len(authMap) == 0 {
			e := fmt.Errorf("invalid launchConfig for server %s: invalid auth", serverEntity.ServerID)
			return failStop(e, e.Error())
		}
		authTypeRaw, ok := authMap["type"].(string)
		if !ok || strings.TrimSpace(authTypeRaw) == "" {
			e := fmt.Errorf("invalid launchConfig for server %s: invalid auth type", serverEntity.ServerID)
			return failStop(e, e.Error())
		}
		first["auth"] = authConfig
		apis[0] = first
		cfg["apis"] = apis
		delete(launchConfig, "auth")
		env, _ := launchConfig["env"].(map[string]any)
		if env == nil {
			env = map[string]any{"type": "none"}
		}
		configRaw, marshalErr := json.Marshal(cfg)
		if marshalErr != nil {
			return failStop(fmt.Errorf("invalid configTemplate for server %s: %w", serverEntity.ServerID, marshalErr), marshalErr.Error())
		}
		env["GATEWAY_CONFIG"] = string(configRaw)
		launchConfig["env"] = env
	case types.ServerCategorySkills:
		m.resolveSkillsVolumeMounts(launchConfig)
	default:
	}

	resolvedPlan, err := CustomStdioRunnerServiceInstance.ResolveLaunchPlan(serverEntity, launchConfig)
	if err != nil {
		return failStop(err, err.Error())
	}
	runnerMetadata := resolvedPlan.RunnerMetadata
	if runnerMetadata != nil {
		log.Info().
			Str("serverId", serverEntity.ServerID).
			Str("originalCommand", runnerMetadata.OriginalCommand).
			Str("runnerImage", runnerMetadata.RunnerImage).
			Msg("CustomStdio command wrapped to runner container")
	}

	createdTransport, err := m.transportFactory.Create(resolvedPlan.LaunchConfig)
	if err != nil {
		return failStop(err, err.Error())
	}

	var runnerTrace *RunnerExecutionTrace
	var stderrCapture *StderrTailWriter
	if runnerMetadata != nil && createdTransport.Cmd != nil {
		runnerTrace = CustomStdioRunnerServiceInstance.AttachExecutionTrace(createdTransport.Cmd)
		stderrCapture = runnerTrace.StderrWriter
	} else if createdTransport.Cmd != nil {
		stderrCapture = NewStderrTailWriter(defaultStderrTailMaxLength, os.Stderr)
		createdTransport.Cmd.Stderr = stderrCapture
	}

	contextObj.mu.Lock()
	contextObj.RunnerMetadata = runnerMetadata
	contextObj.RunnerTrace = runnerTrace
	contextObj.mu.Unlock()

	diagnostics := newStartupDiagnostics(stderrCapture)

	m.mu.RLock()
	repo := m.repo
	m.mu.RUnlock()
	if repo != nil {
		if err := repo.UpdateTransportType(ctx, contextObj.ServerID, string(createdTransport.Type)); err != nil {
			log.Warn().Err(err).Str("serverId", contextObj.ServerID).Msg("failed to persist transport type")
		}
	}

	client, mcpConn, err := m.connectClient(ctx, createdTransport, contextObj.ServerID, serverUserID)
	if err != nil {
		if runnerMetadata != nil && runnerTrace != nil {
			runnerFailure := CustomStdioRunnerServiceInstance.BuildFailureDetails(
				serverEntity.ServerID, runnerMetadata, runnerTrace, err,
			)
			if runnerFailure != nil {
				log.Error().
					Str("serverId", serverEntity.ServerID).
					Str("originalCommand", runnerMetadata.OriginalCommand).
					Str("runnerImage", runnerMetadata.RunnerImage).
					Str("category", runnerFailure.Category).
					Str("reason", runnerFailure.Reason).
					Str("stderrTail", runnerFailure.StderrSummary).
					Msg("CustomStdio runner failed during startup")
				diagnostics.SetRunnerStartupFailureMessage(runnerFailure.Message)
			}
		}

		preferredMsg := err.Error()
		if preferred := diagnostics.GetPreferredMessage(err); preferred != "" {
			preferredMsg = preferred
		}

		logEvent := log.Warn().Err(err).Str("serverName", serverEntity.ServerName).Str("preferredMessage", preferredMsg)
		if stderrSummary := diagnostics.GetStderrSummary(); stderrSummary != "" {
			logEvent = logEvent.Str("stderrSummary", stderrSummary)
		}
		logEvent.Msg("Server startup failed")

		m.updateServerStatus(contextObj, types.ServerStatusError)
		recordServerStartupError(contextObj, preferredMsg, "")
		contextObj.StopTokenRefresh()
		return fmt.Errorf("%s", preferredMsg)
	}

	var transportCloser TransportCloser
	if closer, ok := createdTransport.Transport.(TransportCloser); ok {
		transportCloser = closer
	}
	monitorCtx, monitorSeq := contextObj.SetConnection(client, transportCloser, mcpConn)
	go m.monitorServerConnection(monitorCtx, contextObj, client, monitorSeq)

	if cs, ok := client.(*mcp.ClientSession); ok {
		if initResult := cs.InitializeResult(); initResult != nil {
			if initResult.Capabilities != nil {
				capsJSON, merr := json.Marshal(initResult.Capabilities)
				if merr == nil {
					capsMap := make(map[string]any)
					if uerr := json.Unmarshal(capsJSON, &capsMap); uerr == nil {
						contextObj.UpdateCapabilities(capsMap)
					} else {
						log.Warn().Err(uerr).Str("serverId", contextObj.ServerID).Msg("failed to decode downstream init capabilities")
					}
				} else {
					log.Warn().Err(merr).Str("serverId", contextObj.ServerID).Msg("failed to marshal downstream init capabilities")
				}
			}

			if (serverEntity.Category == types.ServerCategoryCustomRemote || serverEntity.Category == types.ServerCategoryCustomStdio) && initResult.ServerInfo != nil {
				name := strings.TrimSpace(initResult.ServerInfo.Name)
				if name != "" && name != serverEntity.ServerName {
					dbName := name
					if serverEntity.AllowUserInput {
						dbName += " Personal"
					}
					m.mu.RLock()
					repo := m.repo
					m.mu.RUnlock()
					if repo != nil {
						if err := repo.UpdateServerName(ctx, serverEntity.ServerID, dbName); err != nil {
							log.Warn().Err(err).Str("serverId", serverEntity.ServerID).Msg("failed to persist server name")
						}
					}
					contextObj.mu.Lock()
					contextObj.ServerEntity.ServerName = dbName
					contextObj.mu.Unlock()
				}
			}
		}
	}

	m.updateServerStatus(contextObj, types.ServerStatusOnline)
	contextObj.Touch()

	if err := m.UpdateServerCapabilities(ctx, contextObj); err != nil {
		log.Warn().Err(err).Str("serverId", contextObj.ServerID).Msg("capability refresh failed during startup")
	}

	return nil
}

func (m *serverManager) connectClient(ctx context.Context, created *CreatedTransport, serverID string, userID string) (DownstreamClient, mcp.Connection, error) {
	if created == nil {
		return nil, nil, errors.New("created transport is required")
	}

	client := mcp.NewClient(&mcp.Implementation{Name: config.AppInfo.Name, Version: config.AppInfo.Version}, &mcp.ClientOptions{
		CreateMessageHandler: func(c context.Context, req *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
			if req == nil || req.Params == nil {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "invalid sampling request"}
			}
			proxyRequestID := proxyRequestIDFromMeta(req.GetParams().GetMeta())
			if proxyRequestID == "" {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "missing proxyContext.proxyRequestId in sampling request"}
			}
			if !reverseRequestAllowed(proxyRequestID, reverseRequestSampling) {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeMethodNotFound, Message: "sampling not supported by client"}
			}
			return GlobalRequestRouterInstance().HandleSamplingRequest(c, serverID, req, proxyRequestID)
		},
		ElicitationHandler: func(c context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			if req == nil || req.Params == nil {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "invalid elicitation request"}
			}
			proxyRequestID := proxyRequestIDFromMeta(req.GetParams().GetMeta())
			if proxyRequestID == "" {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "missing proxyContext.proxyRequestId in elicitation request"}
			}
			if !reverseRequestAllowed(proxyRequestID, reverseRequestElicitation) {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeMethodNotFound, Message: "elicitation not supported by client"}
			}
			return GlobalRequestRouterInstance().HandleElicitationRequest(c, serverID, req, proxyRequestID)
		},
		ToolListChangedHandler: func(ctx context.Context, _ *mcp.ToolListChangedRequest) {
			changes := m.refreshCapabilitiesForServerID(ctx, serverID)
			m.logCapabilityNotificationUpdates(changes, "tools/listChanged")
			GlobalRequestRouterInstance().HandleToolsListChanged(serverID)
		},
		PromptListChangedHandler: func(ctx context.Context, _ *mcp.PromptListChangedRequest) {
			changes := m.refreshCapabilitiesForServerID(ctx, serverID)
			m.logCapabilityNotificationUpdates(changes, "prompts/listChanged")
			GlobalRequestRouterInstance().HandlePromptsListChanged(serverID)
		},
		ResourceListChangedHandler: func(ctx context.Context, _ *mcp.ResourceListChangedRequest) {
			changes := m.refreshCapabilitiesForServerID(ctx, serverID)
			m.logCapabilityNotificationUpdates(changes, "resources/listChanged")
			GlobalRequestRouterInstance().HandleResourcesListChanged(serverID)
		},
		ResourceUpdatedHandler: func(_ context.Context, req *mcp.ResourceUpdatedNotificationRequest) {
			GlobalRequestRouterInstance().HandleResourceUpdated(serverID, req)
		},
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			if req == nil || req.Params == nil {
				return
			}
			proxyRequestID := fmt.Sprintf("%v", req.Params.ProgressToken)
			if proxyRequestID == "" {
				return
			}
			GlobalRequestRouterInstance().HandleProgressNotification(proxyRequestID, req.Params)
		},
	})

	client.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(c context.Context, method string, req mcp.Request) (mcp.Result, error) {
			switch method {
			case "roots/list":
				rootsReq, ok := req.(*mcp.ListRootsRequest)
				if !ok || rootsReq == nil || rootsReq.Params == nil {
					return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "invalid roots/list request"}
				}
				proxyRequestID := proxyRequestIDFromMeta(rootsReq.GetParams().GetMeta())
				if proxyRequestID == "" {
					return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "missing proxyContext.proxyRequestId in roots request"}
				}
				if !reverseRequestAllowed(proxyRequestID, reverseRequestRoots) {
					return nil, &jsonrpc.Error{Code: jsonrpc.CodeMethodNotFound, Message: "roots not supported by client"}
				}
				return GlobalRequestRouterInstance().HandleRootsListRequest(c, serverID, rootsReq, proxyRequestID)
			case "notifications/cancelled":
				if req != nil && req.GetParams() != nil {
					if params, ok := req.GetParams().(*mcp.CancelledParams); ok {
						if params.RequestID != nil {
							handled := GlobalRequestRouterInstance().HandleCancelledNotification(params.RequestID, serverID)
							if !handled {
								if progressToken := params.GetProgressToken(); progressToken != nil {
									_ = GlobalRequestRouterInstance().HandleCancelledNotification(progressToken, serverID)
								}
							}
						}
					}
				}
			}
			return next(c, method, req)
		}
	})

	rawTransport, err := toMCPTransport(created)
	if err != nil {
		return nil, nil, err
	}
	capturing := &capturingTransport{inner: rawTransport, serverID: serverID}

	session, err := client.Connect(ctx, capturing, nil)
	if err != nil {
		return nil, nil, err
	}

	if err := session.Ping(ctx, &mcp.PingParams{}); err != nil {
		if cerr := session.Close(); cerr != nil {
			log.Warn().Err(cerr).Str("serverId", serverID).Str("userId", userID).Msg("failed to close downstream session after ping failure")
		}
		return nil, nil, err
	}

	return session, capturing.conn, nil
}

// resolveSkillsVolumeMounts rewrites relative skills volume mount paths in Docker
// launch configs to use absolute host paths. When kimbap-core runs inside Docker,
// relative paths like ./skills/{serverId} resolve to container paths, but the
// Docker daemon needs host-absolute paths for bind mounts.
func (m *serverManager) resolveSkillsVolumeMounts(launchConfig map[string]any) {
	command, _ := launchConfig["command"].(string)
	if command != "docker" {
		return
	}
	rawArgs, ok := launchConfig["args"].([]any)
	if !ok || len(rawArgs) == 0 {
		return
	}

	hostSkillsDir := strings.TrimRight(config.SKILLS_CONFIG.HostSkillsDir, "/")
	isInDocker := config.Env("KIMBAP_CORE_IN_DOCKER") == "true"

	if isInDocker && hostSkillsDir == "" {
		log.Warn().Msg("HOST_SKILLS_DIR not set while running in Docker. Skills volume mounts may fail on Linux.")
		return
	}
	if hostSkillsDir == "" {
		return
	}

	if isInDocker && !strings.HasPrefix(hostSkillsDir, "/") {
		log.Warn().Str("hostSkillsDir", hostSkillsDir).Msg("HOST_SKILLS_DIR must be an absolute host path when KIMBAP_CORE_IN_DOCKER=true. Skills volume mounts may fail on Linux.")
		return
	}

	for i, raw := range rawArgs {
		arg, ok := raw.(string)
		if !ok {
			continue
		}
		if (arg == "-v" || arg == "--volume") && i+1 < len(rawArgs) {
			if nextArg, ok := rawArgs[i+1].(string); ok {
				rewritten := rewriteSkillsVolumeArg(nextArg, hostSkillsDir)
				if rewritten != nextArg {
					rawArgs[i+1] = rewritten
					log.Info().Str("original", nextArg).Str("resolved", rewritten).Msg("Rewrote skills volume mount for Docker-in-Docker compatibility")
				}
			}
		}
		if strings.HasPrefix(arg, "-v=") || strings.HasPrefix(arg, "--volume=") {
			eqIdx := strings.Index(arg, "=")
			volSpec := arg[eqIdx+1:]
			rewritten := rewriteSkillsVolumeArg(volSpec, hostSkillsDir)
			if rewritten != volSpec {
				rawArgs[i] = arg[:eqIdx+1] + rewritten
				log.Info().Str("original", volSpec).Str("resolved", rewritten).Msg("Rewrote skills volume mount for Docker-in-Docker compatibility")
			}
		}
	}
	launchConfig["args"] = rawArgs
}

// rewriteSkillsVolumeArg rewrites a single Docker volume spec if it contains
// a relative skills path. e.g., "./skills/abc123:/app/skills:ro" becomes
// "/host/path/skills/abc123:/app/skills:ro"
func rewriteSkillsVolumeArg(volumeSpec string, hostSkillsDir string) string {
	parts := strings.SplitN(volumeSpec, ":", 3)
	if len(parts) < 2 {
		return volumeSpec
	}
	source := parts[0]
	if strings.HasPrefix(source, "./skills/") || strings.HasPrefix(source, "skills/") {
		relativePart := source
		if strings.HasPrefix(relativePart, "./") {
			relativePart = relativePart[2:]
		}
		afterSkills := strings.TrimPrefix(relativePart, "skills/")
		if afterSkills == "" || strings.HasPrefix(afterSkills, "/") {
			log.Warn().Str("volumeSpec", volumeSpec).Msg("Rejecting skills volume mount rewrite: path traversal detected")
			return volumeSpec
		}
		for _, seg := range strings.Split(afterSkills, "/") {
			if seg == ".." {
				log.Warn().Str("volumeSpec", volumeSpec).Msg("Rejecting skills volume mount rewrite: path traversal detected")
				return volumeSpec
			}
		}
		parts[0] = hostSkillsDir + "/" + afterSkills
		return strings.Join(parts, ":")
	}
	return volumeSpec
}

func (m *serverManager) decryptLaunchConfig(key string, server database.Server) (string, error) {
	decrypted, err := security.DecryptDataFromString(server.LaunchConfig, key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt launch config for server %s: %w", server.ServerID, err)
	}
	return decrypted, nil
}

func (m *serverManager) initializeAuthentication(ctx context.Context, sc *ServerContext, launchConfig map[string]any, token string) error {
	sc.mu.RLock()
	category := sc.ServerEntity.Category
	sc.mu.RUnlock()

	// Only Template servers use the OAuth token refresh strategy.
	// Other categories either embed credentials directly in the launch config
	// or use no authentication at all.
	if category != types.ServerCategoryTemplate {
		return nil
	}

	m.mu.RLock()
	authFactory := m.authFactory
	m.mu.RUnlock()
	if authFactory == nil {
		return nil
	}
	sc.mu.Lock()
	serverEntity := sc.ServerEntity
	sc.mu.Unlock()

	strategy, err := authFactory.Build(ctx, serverEntity, launchConfig, token)
	if err != nil {
		return err
	}
	if strategy == nil {
		return nil
	}
	accessToken, err := sc.StartTokenRefresh(ctx, strategy)
	if err != nil {
		return err
	}
	m.injectOAuthTokenEnv(serverEntity.AuthType, launchConfig, accessToken)
	delete(launchConfig, "oauth")
	return nil
}

func (m *serverManager) injectOAuthTokenEnv(authType int, launchConfig map[string]any, accessToken string) {
	if accessToken == "" {
		return
	}
	env, _ := launchConfig["env"].(map[string]any)
	if env == nil {
		env = map[string]any{}
	}
	switch authType {
	case types.ServerAuthTypeGoogleAuth, types.ServerAuthTypeGoogleCalendarAuth, types.ServerAuthTypeFigmaAuth, types.ServerAuthTypeGithubAuth, types.ServerAuthTypeCanvaAuth, types.ServerAuthTypeCanvasAuth:
		env["accessToken"] = accessToken
	case types.ServerAuthTypeZendeskAuth:
		zendeskSubdomain, _ := env["zendeskSubdomain"].(string)
		zendeskSubdomain = strings.TrimSpace(zendeskSubdomain)
		if zendeskSubdomain == "" {
			log.Error().Msg("[ServerManager] Missing zendeskSubdomain for server auth type ZendeskAuth")
			return
		}
		zendeskSubdomain = strings.TrimPrefix(zendeskSubdomain, "https://")
		zendeskSubdomain = strings.TrimPrefix(zendeskSubdomain, "http://")
		if idx := strings.Index(zendeskSubdomain, "/"); idx >= 0 {
			zendeskSubdomain = zendeskSubdomain[:idx]
		}
		zendeskSubdomain = strings.TrimSuffix(strings.ToLower(zendeskSubdomain), ".zendesk.com")
		env["zendeskSubdomain"] = zendeskSubdomain
		env["accessToken"] = accessToken
	case types.ServerAuthTypeNotionAuth:
		env["notionToken"] = accessToken
	default:
		return
	}
	launchConfig["env"] = env
}

func (m *serverManager) UpdateServerCapabilities(ctx context.Context, sc *ServerContext) error {
	conn := sc.ConnectionSnapshot()
	if conn == nil {
		return nil
	}

	m.applyToolDefaultConfigFallback(ctx, sc)

	before := m.snapshotCapabilitiesState(sc)

	var firstErr error
	hadTimeout := false

	tools, err := conn.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		if firstErr == nil {
			firstErr = err
		}
		if isTimeoutError(err) {
			hadTimeout = true
		}
		m.handleCapabilitiesRefreshError(ctx, sc, "tools", err)
	} else {
		if persistErr := sc.UpdateTools(ctx, tools); persistErr != nil {
			log.Warn().Err(persistErr).Str("serverId", sc.ServerID).Msg("failed to persist tools")
		}
	}

	resources, err := conn.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		if firstErr == nil {
			firstErr = err
		}
		if isTimeoutError(err) {
			hadTimeout = true
		}
		m.handleCapabilitiesRefreshError(ctx, sc, "resources", err)
	} else {
		if persistErr := sc.UpdateResources(ctx, resources); persistErr != nil {
			log.Warn().Err(persistErr).Str("serverId", sc.ServerID).Msg("failed to persist resources")
		}
	}

	templates, err := conn.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{})
	if err != nil {
		if firstErr == nil {
			firstErr = err
		}
		if isTimeoutError(err) {
			hadTimeout = true
		}
		m.handleCapabilitiesRefreshError(ctx, sc, "resourceTemplates", err)
	} else {
		if persistErr := sc.UpdateResourceTemplates(ctx, templates); persistErr != nil {
			log.Warn().Err(persistErr).Str("serverId", sc.ServerID).Msg("failed to persist resource templates")
		}
	}

	prompts, err := conn.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		if firstErr == nil {
			firstErr = err
		}
		if isTimeoutError(err) {
			hadTimeout = true
		}
		m.handleCapabilitiesRefreshError(ctx, sc, "prompts", err)
	} else {
		if persistErr := sc.UpdatePrompts(ctx, prompts); persistErr != nil {
			log.Warn().Err(persistErr).Str("serverId", sc.ServerID).Msg("failed to persist prompts")
		}
	}

	if firstErr == nil && !hadTimeout {
		sc.ClearTimeout()
	}

	after := m.snapshotCapabilitiesState(sc)
	toolsChanged := before.Tools != after.Tools
	resourcesChanged := before.Resources != after.Resources || before.ResourceTemplates != after.ResourceTemplates
	promptsChanged := before.Prompts != after.Prompts
	if toolsChanged || resourcesChanged || promptsChanged {
		m.logServerCapabilityUpdate(m.loggerKeyForContext(sc), map[string]any{
			"type":             "capabilities/refresh",
			"toolsChanged":     toolsChanged,
			"resourcesChanged": resourcesChanged,
			"promptsChanged":   promptsChanged,
		})
	}

	return firstErr
}

func (m *serverManager) refreshCapabilitiesForServerID(ctx context.Context, serverID string) []capabilityRefreshResult {
	contexts := m.serverContextsForServerID(serverID)
	results := make([]capabilityRefreshResult, 0, len(contexts))

	for _, sc := range contexts {
		before := m.snapshotCapabilitiesState(sc)
		if err := m.UpdateServerCapabilities(ctx, sc); err != nil {
			log.Warn().Err(err).Str("serverId", sc.ServerID).Msg("failed to refresh server capabilities")
			continue
		}
		after := m.snapshotCapabilitiesState(sc)
		results = append(results, capabilityRefreshResult{
			context:          sc,
			toolsChanged:     before.Tools != after.Tools,
			resourcesChanged: before.Resources != after.Resources || before.ResourceTemplates != after.ResourceTemplates,
			promptsChanged:   before.Prompts != after.Prompts,
		})
	}

	return results
}

func (m *serverManager) serverContextsForServerID(serverID string) []*ServerContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	contexts := make([]*ServerContext, 0)
	if sc := m.serverContexts[serverID]; sc != nil {
		contexts = append(contexts, sc)
	}
	for _, sc := range m.temporaryServers {
		if sc != nil && sc.ServerID == serverID {
			contexts = append(contexts, sc)
		}
	}
	return contexts
}

func (m *serverManager) logCapabilityNotificationUpdates(results []capabilityRefreshResult, notificationType string) {
	for _, result := range results {
		sc := result.context
		if sc == nil {
			continue
		}

		shouldLog := false
		params := map[string]any{"type": notificationType}

		sc.mu.RLock()
		switch notificationType {
		case "tools/listChanged":
			shouldLog = result.toolsChanged
			if sc.Tools != nil {
				params["toolsCount"] = len(sc.Tools.Tools)
			} else {
				params["toolsCount"] = 0
			}
		case "resources/listChanged":
			shouldLog = result.resourcesChanged
			if sc.Resources != nil {
				params["resourcesCount"] = len(sc.Resources.Resources)
			} else {
				params["resourcesCount"] = 0
			}
		case "prompts/listChanged":
			shouldLog = result.promptsChanged
			if sc.Prompts != nil {
				params["promptsCount"] = len(sc.Prompts.Prompts)
			} else {
				params["promptsCount"] = 0
			}
		}
		sc.mu.RUnlock()

		if !shouldLog {
			continue
		}

		m.logServerCapabilityUpdate(m.loggerKeyForContext(sc), params)
	}
}

func (m *serverManager) UpdateServerCapabilitiesConfig(_ context.Context, serverID string, raw string) (toolsChanged bool, resourcesChanged bool, promptsChanged bool, err error) {
	contextObj := m.GetServerContext(serverID, "")
	if contextObj == nil {
		return false, false, false, nil
	}

	newCapabilitiesConfig := mcptypes.ServerConfigCapabilities{}
	if err := json.Unmarshal([]byte(raw), &newCapabilitiesConfig); err != nil {
		log.Warn().Err(err).Str("serverId", serverID).Msg("invalid capabilities config json, marking all capabilities as changed")
		return true, true, true, nil
	}
	newCapabilitiesConfig.EnsureInitialized()

	toolsChanged, resourcesChanged, promptsChanged = contextObj.IsCapabilityChanged(newCapabilitiesConfig)
	if err := contextObj.UpdateCapabilitiesConfig(raw); err != nil {
		return false, false, false, err
	}

	if toolsChanged || resourcesChanged || promptsChanged {
		m.logServerCapabilityUpdate(serverID, map[string]any{
			"type":             "capabilities/config",
			"toolsChanged":     toolsChanged,
			"resourcesChanged": resourcesChanged,
			"promptsChanged":   promptsChanged,
		})
	}

	return toolsChanged, resourcesChanged, promptsChanged, nil
}

func (m *serverManager) GetServerContext(serverID, userID string) *ServerContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s := m.serverContexts[serverID]; s != nil {
		return s
	}
	if userID == "" {
		return nil
	}
	return m.temporaryServers[serverID+":"+userID]
}

func (m *serverManager) GetServerContextByInternalID(internalID, userID string) *ServerContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ctx := range m.serverContexts {
		if ctx.ID == internalID {
			return ctx
		}
	}
	for _, ctx := range m.temporaryServers {
		if ctx.ID == internalID && (userID == "" || ctx.UserID == userID) {
			return ctx
		}
	}
	return nil
}

func (m *serverManager) GetAvailableServers() []*ServerContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*ServerContext, 0, len(m.serverContexts)+len(m.temporaryServers))
	for _, s := range m.serverContexts {
		status := s.StatusSnapshot()
		if status == types.ServerStatusOnline || status == types.ServerStatusSleeping {
			out = append(out, s)
		}
	}
	for _, s := range m.temporaryServers {
		status := s.StatusSnapshot()
		if status == types.ServerStatusOnline || status == types.ServerStatusSleeping {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i]
		right := out[j]
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.ServerID != right.ServerID {
			return left.ServerID < right.ServerID
		}
		leftUserID := left.UserIDSnapshot()
		rightUserID := right.UserIDSnapshot()
		if leftUserID != rightUserID {
			return leftUserID < rightUserID
		}
		return left.ID < right.ID
	})
	return out
}

func (m *serverManager) GetAvailableServersCapabilities() map[string]mcptypes.MCPServerCapability {
	capabilities := map[string]mcptypes.MCPServerCapability{}
	for _, contextObj := range m.GetAvailableServers() {
		mcpCaps := contextObj.GetMCPCapabilities()
		if contextObj.ServerEntitySnapshot().AllowUserInput {
			mcpCaps.Tools = map[string]mcptypes.ToolCapabilityConfig{}
			mcpCaps.Resources = map[string]mcptypes.ResourceCapabilityConfig{}
			mcpCaps.Prompts = map[string]mcptypes.PromptCapabilityConfig{}
			mcpCaps.Configured = true
		}
		capabilities[contextObj.ServerID] = mcpCaps
	}
	return capabilities
}

func (m *serverManager) GetUserAvailableServers(user database.User) []*ServerContext {
	permissions := mcptypes.Permissions{}
	if strings.TrimSpace(user.Permissions) != "" {
		if err := json.Unmarshal([]byte(user.Permissions), &permissions); err != nil {
			log.Warn().Err(err).Str("userId", user.UserID).Msg("invalid user permissions JSON; returning no available servers")
			return []*ServerContext{}
		}
		if permissions == nil {
			log.Warn().Str("userId", user.UserID).Msg("null user permissions; returning no available servers")
			return []*ServerContext{}
		}
	}

	availableServers := m.GetAvailableServers()
	result := make([]*ServerContext, 0, len(availableServers))
	for _, serverCtx := range availableServers {
		if serverCtx == nil {
			continue
		}
		serverEntity := serverCtx.ServerEntitySnapshot()
		if !serverEntity.Enabled {
			continue
		}
		enabled := serverEntity.PublicAccess
		if permission, ok := permissions[serverEntity.ServerID]; ok {
			enabled = permission.Enabled
		}
		if !enabled {
			continue
		}
		if serverEntity.AllowUserInput && serverCtx.UserIDSnapshot() != user.UserID {
			continue
		}
		result = append(result, serverCtx)
	}
	return result
}

func (m *serverManager) CreateTemporaryServer(ctx context.Context, userID string, server database.Server, token string, sleep bool) (*ServerContext, error) {
	internalKey := tempServerKey(server.ServerID, userID)

	m.mu.Lock()
	if existing := m.temporaryServers[internalKey]; existing != nil {
		m.attachCapabilitiesPersistHook(existing)
		m.mu.Unlock()
		status := existing.StatusSnapshot()
		if status == types.ServerStatusOnline {
			return existing, nil
		}
		_, _ = m.CloseTemporaryServer(ctx, server.ServerID, userID)
		m.mu.Lock()
	}

	contextObj := NewServerContext(server)
	m.attachCapabilitiesPersistHook(contextObj)
	contextObj.mu.Lock()
	contextObj.UserID = userID
	contextObj.UserToken = token
	contextObj.mu.Unlock()
	m.temporaryServers[internalKey] = contextObj
	if _, ok := m.serverLoggers[internalKey]; !ok {
		m.serverLoggers[internalKey] = serverlog.NewServerLogger(internalKey)
	}
	m.mu.Unlock()

	if sleep && m.isLazyStartApplicable(server) {
		m.updateServerStatus(contextObj, types.ServerStatusSleeping)
		return contextObj, nil
	}

	connectDone := make(chan struct{})
	m.serverConnecting.Store(server.ServerID, connectDone)
	connectErr := m.createServerConnection(ctx, contextObj, token)
	close(connectDone)
	m.serverConnecting.Delete(server.ServerID)

	if connectErr != nil {
		m.mu.Lock()
		delete(m.temporaryServers, internalKey)
		delete(m.serverLoggers, internalKey)
		m.mu.Unlock()
		return nil, connectErr
	}
	return contextObj, nil
}

func (m *serverManager) GetTemporaryServer(serverID, userID string) *ServerContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.temporaryServers[tempServerKey(serverID, userID)]
}

func (m *serverManager) GetTemporaryServers() []*ServerContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*ServerContext, 0, len(m.temporaryServers))
	for _, contextObj := range m.temporaryServers {
		out = append(out, contextObj)
	}
	return out
}

func (m *serverManager) CloseTemporaryServer(ctx context.Context, serverID, userID string) (*ServerContext, error) {
	_ = ctx
	key := tempServerKey(serverID, userID)
	m.mu.Lock()
	contextObj := m.temporaryServers[key]
	delete(m.temporaryServers, key)
	serverLogger := m.serverLoggers[key]
	delete(m.serverLoggers, key)
	m.mu.Unlock()
	if contextObj == nil {
		return nil, nil
	}
	if serverLogger != nil {
		serverLogger.LogServerLifecycle(types.MCPEventLogTypeServerClose, "")
	}
	contextObj.StopTokenRefresh()
	return contextObj, contextObj.CloseConnection(types.ServerStatusOffline)
}

func (m *serverManager) CloseUserTemporaryServers(ctx context.Context, userID string) {
	m.closeTemporaryServersByFilter(ctx, func(key string) bool {
		return strings.HasSuffix(key, ":"+userID)
	})
}

func (m *serverManager) CloseAllTemporaryServersByTemplate(serverID string) {
	if strings.TrimSpace(serverID) == "" {
		return
	}
	prefix := serverID + ":"
	m.closeTemporaryServersByFilter(context.Background(), func(key string) bool {
		return strings.HasPrefix(key, prefix)
	})
}

func (m *serverManager) closeTemporaryServersByFilter(ctx context.Context, matchFn func(key string) bool) {
	m.mu.RLock()
	keys := make([]string, 0)
	for key := range m.temporaryServers {
		if matchFn(key) {
			keys = append(keys, key)
		}
	}
	m.mu.RUnlock()
	for _, key := range keys {
		parts := splitServerUserKey(key)
		_, _ = m.CloseTemporaryServer(ctx, parts.serverID, parts.userID)
	}
}

func (m *serverManager) UpdateTemporaryServersByTemplate(server database.Server) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, temporary := range m.temporaryServers {
		if temporary == nil || temporary.ServerID != server.ServerID {
			continue
		}
		temporary.mu.Lock()
		launchConfig := temporary.ServerEntity.LaunchConfig
		next := server
		next.LaunchConfig = launchConfig
		temporary.ServerEntity = next
		temporary.mu.Unlock()
	}
}

func (m *serverManager) StartUserTemporaryServersForSession(session *ClientSession) {
	if session == nil {
		return
	}
	authContext := session.AuthContextSnapshot()
	launchConfigsRaw := authContext.LaunchConfigs
	if launchConfigsRaw == "" {
		return
	}
	configs := map[string]any{}
	if err := json.Unmarshal([]byte(launchConfigsRaw), &configs); err != nil {
		log.Warn().Err(err).Msg("invalid launch configs JSON, skipping temporary server startup")
		return
	}
	m.mu.RLock()
	repo := m.repo
	m.mu.RUnlock()
	if repo == nil {
		return
	}
	for serverID, launchConfigVal := range configs {
		launchConfig, err := security.EncryptedAnyToString(launchConfigVal)
		if err != nil {
			log.Error().Err(err).Str("serverID", serverID).Msg("failed to parse launchConfig value")
			continue
		}
		server, err := repo.FindByServerID(context.Background(), serverID)
		if err != nil {
			log.Warn().Err(err).Str("serverId", serverID).Str("userId", session.UserID).Msg("failed to load server for temporary startup")
			continue
		}
		if server == nil || !server.AllowUserInput {
			continue
		}
		serverCopy := *server
		serverCopy.LaunchConfig = launchConfig
		if _, err := m.CreateTemporaryServer(context.Background(), session.UserID, serverCopy, session.Token, true); err != nil {
			log.Warn().Err(err).Str("serverId", serverID).Str("userId", session.UserID).Msg("failed to start temporary server for session")
		}
	}
}

func (m *serverManager) SubscribeResource(ctx context.Context, serverID, resourceURI, sessionID, userID string) error {
	ctxObj := m.GetServerContext(serverID, userID)
	if ctxObj == nil || ctxObj.StatusSnapshot() != types.ServerStatusOnline {
		return errors.New("server unavailable for subscription")
	}

	ctxObj.mu.RLock()
	rawCaps := ctxObj.Capabilities
	ctxObj.mu.RUnlock()
	parsedCaps := parseServerCapabilities(rawCaps)
	if parsedCaps.Resources == nil || !parsedCaps.Resources.Subscribe {
		return errors.New("server does not support resource subscriptions")
	}

	subKey := makeSubscriptionKey(serverID, resourceURI)

	m.mu.Lock()
	state := m.resourceSubscriptions[subKey]
	if state == nil {
		state = &subscriptionState{SubscribedSessions: map[string]struct{}{}}
		m.resourceSubscriptions[subKey] = state
	}
	_, already := state.SubscribedSessions[sessionID]
	if !already {
		state.SubscribedSessions[sessionID] = struct{}{}
	}
	needDownstream := !state.DownstreamSubscribed && !state.DownstreamSubscribing
	if needDownstream {
		state.DownstreamSubscribing = true
	}
	m.mu.Unlock()

	if !needDownstream {
		return nil
	}
	conn := ctxObj.ConnectionSnapshot()
	if conn == nil {
		m.mu.Lock()
		if state := m.resourceSubscriptions[subKey]; state != nil {
			state.DownstreamSubscribing = false
		}
		m.mu.Unlock()
		return errors.New("server unavailable for subscription")
	}
	if err := conn.Subscribe(ctx, &mcp.SubscribeParams{URI: resourceURI}); err != nil {
		m.mu.Lock()
		if state := m.resourceSubscriptions[subKey]; state != nil {
			state.DownstreamSubscribing = false
		}
		m.mu.Unlock()
		return err
	}

	needDownstreamUnsubscribe := false
	m.mu.Lock()
	if state := m.resourceSubscriptions[subKey]; state != nil {
		state.DownstreamSubscribed = true
		state.DownstreamSubscribing = false
		if state.DownstreamUnsubscribePending || len(state.SubscribedSessions) == 0 {
			needDownstreamUnsubscribe = true
			delete(m.resourceSubscriptions, subKey)
		}
	} else {
		needDownstreamUnsubscribe = true
	}
	m.mu.Unlock()

	if needDownstreamUnsubscribe {
		unsubscribeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := conn.Unsubscribe(unsubscribeCtx, &mcp.UnsubscribeParams{URI: resourceURI}); err != nil {
			log.Warn().Err(err).Str("serverId", serverID).Str("resourceURI", resourceURI).Msg("failed to cleanup downstream subscription")
		}
	}
	return nil
}

func (m *serverManager) UnsubscribeResource(ctx context.Context, serverID, resourceURI, sessionID, userID string) error {
	ctxObj := m.GetServerContext(serverID, userID)
	subKey := makeSubscriptionKey(serverID, resourceURI)

	m.mu.Lock()
	state := m.resourceSubscriptions[subKey]
	if state == nil {
		m.mu.Unlock()
		return nil
	}
	delete(state.SubscribedSessions, sessionID)
	remaining := len(state.SubscribedSessions)
	downstream := state.DownstreamSubscribed
	downstreamSubscribing := state.DownstreamSubscribing
	if remaining == 0 {
		if downstreamSubscribing {
			state.DownstreamUnsubscribePending = true
		} else {
			delete(m.resourceSubscriptions, subKey)
		}
	}
	m.mu.Unlock()

	if remaining > 0 || !downstream || downstreamSubscribing {
		return nil
	}

	if ctxObj == nil {
		return nil
	}
	conn := ctxObj.ConnectionSnapshot()
	if conn == nil {
		return nil
	}
	return conn.Unsubscribe(ctx, &mcp.UnsubscribeParams{URI: resourceURI})
}

func (m *serverManager) CleanupSessionSubscriptions(ctx context.Context, sessionID, userID string) {
	m.mu.RLock()
	keys := make([]string, 0)
	for key, state := range m.resourceSubscriptions {
		if _, ok := state.SubscribedSessions[sessionID]; ok {
			keys = append(keys, key)
		}
	}
	m.mu.RUnlock()

	for _, key := range keys {
		parts := splitSubscriptionKey(key)
		unsubscribeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := m.UnsubscribeResource(unsubscribeCtx, parts.serverID, parts.resourceURI, sessionID, userID); err != nil {
			log.Warn().Err(err).Str("serverId", parts.serverID).Str("resourceURI", parts.resourceURI).Str("sessionId", sessionID).Str("userId", userID).Msg("failed to cleanup session subscription")
		}
		cancel()
	}
}

func (m *serverManager) GetResourceSubscribers(subscriptionKey string) map[string]struct{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]struct{}{}
	state := m.resourceSubscriptions[subscriptionKey]
	if state == nil {
		return out
	}
	for k := range state.SubscribedSessions {
		out[k] = struct{}{}
	}
	return out
}

type reverseRequestKind int

const (
	reverseRequestSampling reverseRequestKind = iota
	reverseRequestRoots
	reverseRequestElicitation
)

func reverseRequestAllowed(proxyRequestID string, kind reverseRequestKind) bool {
	if strings.TrimSpace(proxyRequestID) == "" {
		return false
	}
	sessionID := sessionIDFromProxyRequestID(proxyRequestID)
	if strings.TrimSpace(sessionID) == "" {
		return false
	}
	proxySession := SessionStoreInstance().GetProxySession(sessionID)
	if proxySession == nil {
		return false
	}
	switch kind {
	case reverseRequestSampling:
		return proxySession.clientSession.CanRequestSampling()
	case reverseRequestRoots:
		return proxySession.clientSession.CanRequestRoots()
	case reverseRequestElicitation:
		return proxySession.clientSession.CanRequestElicitation()
	default:
		return false
	}
}

func toMCPTransport(created *CreatedTransport) (mcp.Transport, error) {
	switch transport := created.Transport.(type) {
	case mcp.Transport:
		return transport, nil
	case *mcp.StreamableClientTransport:
		return transport, nil
	case *exec.Cmd:
		return &mcp.CommandTransport{Command: transport}, nil
	default:
		return nil, fmt.Errorf("unsupported transport instance: %T", created.Transport)
	}
}

type capturingTransport struct {
	inner    mcp.Transport
	serverID string
	conn     mcp.Connection
}

func (t *capturingTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	wrapped := &requestTrackingConnection{inner: conn, serverID: t.serverID}
	t.conn = wrapped
	return wrapped, nil
}

type requestTrackingConnection struct {
	inner    mcp.Connection
	serverID string
}

func (c *requestTrackingConnection) SessionID() string {
	if c == nil || c.inner == nil {
		return ""
	}
	return c.inner.SessionID()
}

func (c *requestTrackingConnection) Read(ctx context.Context) (jsonrpc.Message, error) {
	if c == nil || c.inner == nil {
		return nil, errors.New("connection unavailable")
	}
	return c.inner.Read(ctx)
}

func (c *requestTrackingConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	if c == nil || c.inner == nil {
		return errors.New("connection unavailable")
	}
	c.registerRequestMapping(msg)
	return c.inner.Write(ctx, msg)
}

func (c *requestTrackingConnection) Close() error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Close()
}

func (c *requestTrackingConnection) registerRequestMapping(msg jsonrpc.Message) {
	request, ok := msg.(*jsonrpc.Request)
	if !ok || request == nil || !request.IsCall() {
		return
	}
	if !isDownstreamForwardedMethod(request.Method) {
		return
	}
	proxyRequestID := proxyRequestIDFromRequestParams(request.Params)
	if strings.TrimSpace(proxyRequestID) == "" {
		return
	}
	GlobalRequestRouterInstance().RegisterDownstreamRequestMapping(c.serverID, proxyRequestID, request.ID.Raw())
}

func isDownstreamForwardedMethod(method string) bool {
	switch method {
	case "tools/call", "resources/read", "prompts/get", "completion/complete":
		return true
	default:
		return false
	}
}

func proxyRequestIDFromRequestParams(params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}
	var body map[string]any
	if err := json.Unmarshal(params, &body); err != nil {
		return ""
	}
	metaRaw, ok := body["_meta"]
	if !ok || metaRaw == nil {
		metaRaw, ok = body["meta"]
		if !ok || metaRaw == nil {
			return ""
		}
	}
	meta, ok := metaRaw.(map[string]any)
	if !ok {
		return ""
	}
	token, ok := meta["progressToken"]
	if !ok || token == nil {
		return proxyRequestIDFromMeta(meta)
	}
	return fmt.Sprintf("%v", token)
}

func proxyRequestIDFromMeta(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	proxyContextRaw, ok := meta["proxyContext"]
	if !ok || proxyContextRaw == nil {
		return ""
	}
	proxyContext, ok := proxyContextRaw.(map[string]any)
	if !ok {
		return ""
	}
	value, ok := proxyContext["proxyRequestId"]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func (m *serverManager) startIdleChecker() {
	m.idleTicker = time.NewTicker(5 * time.Minute)
	m.idleStop = make(chan struct{})
	go func() {
		for {
			select {
			case <-m.idleTicker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Error().Interface("panic", r).Msg("recovered panic in idle checker")
						}
					}()
					m.checkIdleServers()
				}()
			case <-m.idleStop:
				return
			}
		}
	}()
}

func (m *serverManager) checkIdleServers() {
	for _, sc := range m.GetAvailableServers() {
		status := sc.StatusSnapshot()
		serverEntity := sc.ServerEntitySnapshot()
		if status != types.ServerStatusOnline {
			continue
		}
		if !m.isLazyStartApplicable(serverEntity) {
			continue
		}
		if serverEntity.TransportType == nil || *serverEntity.TransportType != "stdio" {
			continue
		}
		if !sc.IsIdle(5 * time.Minute) {
			continue
		}
		if err := m.sleepServer(sc); err != nil {
			log.Warn().Err(err).Str("serverId", sc.ServerID).Msg("failed to sleep idle server")
		}
	}
}

func (m *serverManager) sleepServer(sc *ServerContext) error {
	oldStatus := sc.StatusSnapshot()
	sc.StopTokenRefresh()
	if err := sc.CloseConnection(types.ServerStatusSleeping); err != nil {
		return err
	}
	if oldStatus != types.ServerStatusSleeping {
		m.mu.RLock()
		notifier := m.notifier
		m.mu.RUnlock()
		if notifier != nil {
			serverEntity := sc.ServerEntitySnapshot()
			go notifier.NotifyServerStatusChanged(sc.ServerID, serverEntity.ServerName, oldStatus, types.ServerStatusSleeping)
		}
	}
	return nil
}

type ServerConnectResult struct {
	ServerID   string `json:"serverId"`
	ServerName string `json:"serverName"`
	ProxyID    int    `json:"proxyId"`
}

func (m *serverManager) ConnectAllServers(ctx context.Context, token string) (success []ServerConnectResult, failed []ServerConnectResult, err error) {
	m.mu.RLock()
	repo := m.repo
	m.mu.RUnlock()
	if repo == nil {
		return nil, nil, errors.New("server repository not configured")
	}
	servers, err := repo.FindAllEnabled(ctx)
	if err != nil {
		return nil, nil, err
	}

	type connectJob struct {
		ctx    *ServerContext
		server database.Server
	}
	var jobs []connectJob

	for _, server := range servers {
		if server.AllowUserInput {
			continue
		}

		m.mu.Lock()
		existing := m.serverContexts[server.ServerID]
		existingStatus := 0
		if existing != nil {
			existingStatus = existing.StatusSnapshot()
		}
		if existing != nil && (existingStatus == types.ServerStatusOnline || existingStatus == types.ServerStatusConnecting || existingStatus == types.ServerStatusSleeping) {
			m.mu.Unlock()
			continue
		}
		sc := NewServerContext(server)
		m.attachCapabilitiesPersistHook(sc)
		m.serverContexts[server.ServerID] = sc
		m.mu.Unlock()

		if m.isLazyStartApplicable(server) {
			m.updateServerStatus(sc, types.ServerStatusSleeping)
			continue
		}
		jobs = append(jobs, connectJob{ctx: sc, server: server})
	}

	type connectOutcome struct {
		info ServerConnectResult
		err  error
	}

	ch := make(chan connectOutcome, len(jobs))
	for _, j := range jobs {
		go func(j connectJob) {
			info := ServerConnectResult{ServerID: j.server.ServerID, ServerName: j.server.ServerName, ProxyID: j.server.ProxyID}
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("serverId", j.server.ServerID).Msg("recovered panic during server connection")
					m.updateServerStatus(j.ctx, types.ServerStatusError)
					j.ctx.RecordError(fmt.Sprintf("panic: %v", r))
					ch <- connectOutcome{info: info, err: fmt.Errorf("panic: %v", r)}
				}
			}()
			err := m.createServerConnection(ctx, j.ctx, token)
			ch <- connectOutcome{info: info, err: err}
		}(j)
	}

	for range jobs {
		outcome := <-ch
		if outcome.err != nil {
			failed = append(failed, outcome.info)
			continue
		}
		success = append(success, outcome.info)
	}
	return success, failed, nil
}

func (m *serverManager) isLazyStartApplicable(server database.Server) bool {
	if os.Getenv("LAZY_START_ENABLED") == "false" {
		return false
	}
	return server.LazyStartEnabled
}

func (m *serverManager) Shutdown(ctx context.Context) {
	m.shutdownOnce.Do(func() {
		m.shuttingDown.Store(true)
		const hardShutdownTimeout = 30 * time.Second

		// Stop reconciliation loop if running
		m.StopReconcileLoop()

		if m.idleTicker != nil {
			m.idleTicker.Stop()
		}
		if m.idleStop != nil {
			close(m.idleStop)
		}

		m.mu.Lock()
		servers := make([]*ServerContext, 0, len(m.serverContexts)+len(m.temporaryServers))
		for _, sc := range m.serverContexts {
			servers = append(servers, sc)
		}
		for _, sc := range m.temporaryServers {
			servers = append(servers, sc)
		}
		m.mu.Unlock()

		done := make(chan struct{})
		go func() {
			defer close(done)
			for _, sc := range servers {
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Error().Interface("panic", r).Str("serverId", sc.ServerID).Msg("recovered panic while closing server during shutdown")
						}
					}()
					sc.StopTokenRefresh()
					_ = sc.CloseConnection(types.ServerStatusOffline)
				}()
			}
		}()

		hardTimeout := time.NewTimer(hardShutdownTimeout)
		defer hardTimeout.Stop()
		select {
		case <-done:
		case <-ctx.Done():
		case <-hardTimeout.C:
			log.Warn().Dur("timeout", hardShutdownTimeout).Msg("server manager shutdown timed out while closing server connections")
		}

		m.mu.Lock()
		m.serverContexts = map[string]*ServerContext{}
		m.temporaryServers = map[string]*ServerContext{}
		m.serverLoggers = map[string]*serverlog.ServerLogger{}
		m.serverConnecting.Clear()
		m.ownerToken = ""
		m.resourceSubscriptions = map[string]*subscriptionState{}
		m.mu.Unlock()
	})
}

func (m *serverManager) PreloadServers(servers []database.Server) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, server := range servers {
		if server.AllowUserInput {
			continue
		}
		if _, exists := m.serverContexts[server.ServerID]; exists {
			continue
		}

		sc := NewServerContext(server)
		m.attachCapabilitiesPersistHook(sc)
		if m.isLazyStartApplicable(server) {
			sc.UpdateStatus(types.ServerStatusSleeping)
		} else {
			sc.UpdateStatus(types.ServerStatusOffline)
		}

		m.serverContexts[server.ServerID] = sc
		count++
	}

	log.Info().Int("count", count).Msg("preloaded servers")
}

// ReconcileServers compares enabled servers in DB with local serverContexts,
// connects missing or offline/error ones, and removes contexts for disabled/deleted servers.
// This is the core mechanism for Track A load balancing: each replica independently
// polls the DB and converges its local state.
func (m *serverManager) ReconcileServers(ctx context.Context) error {
	m.mu.RLock()
	done := m.reconcileDone
	once := m.reconcileOnce
	m.mu.RUnlock()
	return m.reconcileWith(ctx, done, once)
}

// reconcileWith is the internal implementation. The done channel and once guard
// are passed explicitly so the goroutine in StartReconcileLoop can use locally-
// captured copies that are safe from concurrent field resets.
func (m *serverManager) reconcileWith(ctx context.Context, reconcileDone chan struct{}, reconcileOnce *sync.Once) error {
	m.mu.RLock()
	repo := m.repo
	m.mu.RUnlock()
	if repo == nil {
		return errors.New("server repository not configured")
	}

	enabledServers, err := repo.FindAllEnabled(ctx)
	if err != nil {
		return fmt.Errorf("reconcile: failed to query enabled servers: %w", err)
	}

	// Build a set of enabled server IDs from DB (non-template servers only)
	enabledSet := make(map[string]database.Server, len(enabledServers))
	for _, s := range enabledServers {
		if s.AllowUserInput {
			continue // skip template/user-input servers
		}
		enabledSet[s.ServerID] = s
	}

	// Get current local server IDs and their statuses
	m.mu.RLock()
	type localEntry struct {
		status         int
		allowUserInput bool
	}
	localMap := make(map[string]localEntry, len(m.serverContexts))
	for id, sc := range m.serverContexts {
		snap := sc.ServerEntitySnapshot()
		localMap[id] = localEntry{
			status:         sc.StatusSnapshot(),
			allowUserInput: snap.AllowUserInput,
		}
	}
	m.mu.RUnlock()

	// Remove local contexts for servers no longer enabled in DB
	for id, entry := range localMap {
		if entry.allowUserInput {
			continue // don't touch template servers
		}
		if _, enabled := enabledSet[id]; !enabled {
			if _, err := m.RemoveServer(ctx, id); err != nil {
				log.Warn().Err(err).Str("serverId", id).Msg("reconcile: failed to remove disabled server")
			} else {
				log.Info().Str("serverId", id).Msg("reconcile: removed disabled/deleted server")
			}
		}
	}

	// Connect missing or offline/error enabled servers.
	// AddServer handles all cases: missing, offline, error, config changes.
	ownerToken, tokenErr := m.GetOwnerToken()
	if tokenErr != nil {
		log.Warn().Err(tokenErr).Msg("reconcile: owner token not available, skipping server connections")
		// Still mark reconcile as done on first run — the replica is alive and will
		// retry on the next tick. Holding /ready at 503 forever blocks the LB from
		// ever routing the owner's first request that would set the token.
		if reconcileDone != nil && reconcileOnce != nil {
			reconcileOnce.Do(func() {
				close(reconcileDone)
			})
		}
		return fmt.Errorf("reconcile: owner token not available: %w", tokenErr)
	}

	var connectErrors int
	for id, server := range enabledSet {
		entry, existsLocally := localMap[id]

		// Skip servers that are already online or connecting
		if existsLocally {
			switch entry.status {
			case types.ServerStatusOnline, types.ServerStatusConnecting, types.ServerStatusSleeping:
				continue
				// Offline or Error: AddServer will handle reconnection
			}
		}

		if _, err := m.AddServer(ctx, server, ownerToken); err != nil {
			log.Warn().Err(err).Str("serverId", id).Msg("reconcile: failed to connect server")
			connectErrors++
		} else {
			if existsLocally {
				log.Info().Str("serverId", id).Msg("reconcile: reconnected offline/error server")
			} else {
				log.Info().Str("serverId", id).Msg("reconcile: connected new server")
			}
		}
	}

	// Mark first reconcile as complete (ready for traffic).
	// Even if some servers failed, the replica ran reconciliation and can
	// serve traffic for the servers that did connect successfully.
	if reconcileDone != nil && reconcileOnce != nil {
		reconcileOnce.Do(func() {
			close(reconcileDone)
		})
	}

	if connectErrors > 0 {
		return fmt.Errorf("reconcile: %d server(s) failed to connect", connectErrors)
	}
	return nil
}

// StartReconcileLoop begins a periodic reconciliation loop.
// interval: how often to reconcile. Stop via StopReconcileLoop() or Shutdown().
// This method is idempotent: calling it while a loop is already running stops
// the previous loop first.
func (m *serverManager) StartReconcileLoop(interval time.Duration) {
	// Stop any existing loop first (idempotent)
	m.StopReconcileLoop()

	// Reset state for the new loop
	done := make(chan struct{})
	stop := make(chan struct{})
	once := &sync.Once{}
	ticker := time.NewTicker(interval)

	m.mu.Lock()
	m.reconcileDone = done
	m.reconcileOnce = once
	m.reconcileStop = stop
	m.reconcileTicker = ticker
	m.mu.Unlock()

	const reconcileTimeout = 60 * time.Second

	// Capture locals for the goroutine — these must not be read from struct
	// fields inside the goroutine because StopReconcileLoop may nil them.
	go func(ticker *time.Ticker, stop <-chan struct{}, done chan struct{}, once *sync.Once) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("recovered panic in reconcile loop")
			}
		}()

		// Initial reconcile with timeout
		initCtx, initCancel := context.WithTimeout(context.Background(), reconcileTimeout)
		if err := m.reconcileWith(initCtx, done, once); err != nil {
			log.Warn().Err(err).Msg("initial server reconciliation had errors")
		} else {
			log.Info().Msg("initial server reconciliation completed")
		}
		initCancel()

		// Periodic reconcile
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				tickCtx, tickCancel := context.WithTimeout(context.Background(), reconcileTimeout)
				if err := m.reconcileWith(tickCtx, done, once); err != nil {
					log.Warn().Err(err).Msg("periodic server reconciliation had errors")
				}
				tickCancel()
			}
		}
	}(ticker, stop, done, once)
}

// StopReconcileLoop stops the reconciliation loop.
func (m *serverManager) StopReconcileLoop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	done := m.reconcileDone
	once := m.reconcileOnce
	if m.reconcileTicker != nil {
		m.reconcileTicker.Stop()
		m.reconcileTicker = nil
	}
	if m.reconcileStop != nil {
		select {
		case <-m.reconcileStop:
			// already closed
		default:
			close(m.reconcileStop)
		}
		m.reconcileStop = nil
	}
	if done != nil && once != nil {
		once.Do(func() {
			close(done)
		})
	}
	m.reconcileDone = nil
	m.reconcileOnce = nil
}

// ReconcileDone returns a channel that is closed when the first reconcile completes.
// Used by readiness probes to gate traffic until the node has connected its servers.
func (m *serverManager) ReconcileDone() <-chan struct{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.reconcileDone == nil {
		// If no reconcile loop is running, return an already-closed channel
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return m.reconcileDone
}

func (m *serverManager) attachCapabilitiesPersistHook(sc *ServerContext) {
	if sc == nil {
		return
	}
	sc.SetCapabilitiesPersistHook(func(ctx context.Context, serverID string, data map[string]any) error {
		m.mu.RLock()
		repo := m.repo
		m.mu.RUnlock()
		if repo == nil {
			return nil
		}
		return repo.UpdateCapabilitiesCache(ctx, serverID, data)
	})
}

func (m *serverManager) reconnectServerAsync(sc *ServerContext) {
	if sc == nil || m.shuttingDown.Load() {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Str("serverId", sc.ServerID).Msg("recovered panic during async server reconnect")
			}
		}()
		if m.shuttingDown.Load() {
			return
		}
		server := sc.ServerEntitySnapshot()
		token := sc.UserTokenSnapshot()
		userID := sc.UserIDSnapshot()
		ctx := context.Background()
		if server.AllowUserInput && userID != "" {
			if _, err := m.CloseTemporaryServer(ctx, server.ServerID, userID); err != nil {
				log.Warn().Err(err).Str("serverId", server.ServerID).Msg("failed to close temp server before reconnect")
			}
			if _, err := m.CreateTemporaryServer(ctx, userID, server, token, false); err != nil {
				log.Warn().Err(err).Str("serverId", server.ServerID).Msg("failed to recreate temp server after timeout threshold")
			}
		} else {
			if _, err := m.ReconnectServer(ctx, server, token); err != nil {
				log.Warn().Err(err).Str("serverId", server.ServerID).Msg("failed to reconnect server after timeout threshold")
			}
		}
	}()
}

func (m *serverManager) handleCapabilitiesRefreshError(ctx context.Context, sc *ServerContext, section string, err error) {
	if sc == nil || err == nil {
		return
	}

	if isTimeoutError(err) {
		log.Warn().Err(err).Str("serverId", sc.ServerID).Str("section", section).Msg("capability refresh timeout")
		timeoutRecovery := sc.RecordTimeoutWithRecovery(ctx, err)
		if timeoutRecovery == nil {
			log.Warn().Err(err).Str("serverId", sc.ServerID).Str("section", section).Msg("capability refresh timeout recovery failed")
		}
		return
	}

	log.Warn().Err(err).Str("serverId", sc.ServerID).Str("section", section).Msg("capability refresh failed")
}

type waitableDownstreamClient interface {
	Wait() error
}

func (m *serverManager) monitorServerConnection(ctx context.Context, sc *ServerContext, conn DownstreamClient, seq uint64) {
	if ctx == nil || sc == nil || conn == nil {
		return
	}

	if waitable, ok := conn.(waitableDownstreamClient); ok {
		errCh := make(chan error, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("serverId", sc.ServerID).Msg("recovered panic in connection monitor")
					errCh <- fmt.Errorf("panic in connection monitor: %v", r)
				}
			}()
			errCh <- waitable.Wait()
		}()

		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			if ctx.Err() != nil || !sc.IsCurrentConnection(conn, seq) {
				return
			}
			m.handleUnexpectedConnectionClose(sc, err)
			return
		}
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := conn.Ping(pingCtx, &mcp.PingParams{})
			cancel()
			if err == nil {
				continue
			}
			if ctx.Err() != nil || !sc.IsCurrentConnection(conn, seq) {
				return
			}
			m.handleUnexpectedConnectionClose(sc, err)
			return
		}
	}
}

func (m *serverManager) handleUnexpectedConnectionClose(sc *ServerContext, waitErr error) {
	if sc == nil {
		return
	}

	status := sc.StatusSnapshot()
	if status == types.ServerStatusSleeping {
		return
	}

	server := sc.ServerEntitySnapshot()
	affectedSessions := m.getSessionsUsingServer(server.ServerID)
	changes := detectServerCapabilityChange(sc)

	closeErrorMessage := "Transport closed by server"
	if waitErr != nil {
		closeErrorMessage = closeErrorMessage + ": " + waitErr.Error()
	}

	sc.mu.RLock()
	runnerMeta := sc.RunnerMetadata
	runnerTrace := sc.RunnerTrace
	sc.mu.RUnlock()

	if runnerMeta != nil && runnerTrace != nil {
		runnerFailure := CustomStdioRunnerServiceInstance.BuildFailureDetails(
			server.ServerID, runnerMeta, runnerTrace, waitErr,
		)
		if runnerFailure != nil {
			closeErrorMessage = runnerFailure.Message
			log.Error().
				Str("serverId", server.ServerID).
				Str("originalCommand", runnerMeta.OriginalCommand).
				Str("runnerImage", runnerMeta.RunnerImage).
				Str("category", runnerFailure.Category).
				Str("reason", runnerFailure.Reason).
				Str("stderrTail", runnerFailure.StderrSummary).
				Msg("CustomStdio runner process exited")
		}
	}

	sc.mu.Lock()
	sc.RunnerMetadata = nil
	sc.RunnerTrace = nil
	sc.mu.Unlock()

	recordServerStartupError(sc, closeErrorMessage, "")

	if server.AllowUserInput {
		ctx := context.Background()
		_, _ = m.CloseTemporaryServer(ctx, server.ServerID, sc.UserIDSnapshot())
	} else {
		ctx := context.Background()
		_, _ = m.RemoveServer(ctx, server.ServerID)
	}

	m.notifyUsersOfServerChange(server.ServerID, affectedSessions, "server_error", changes)
}

func (m *serverManager) getSessionsUsingServer(serverID string) []*ClientSession {
	all := SessionStoreInstance().GetAllSessions()
	out := make([]*ClientSession, 0)
	for _, session := range all {
		if session == nil {
			continue
		}
		if session.CanAccessServer(serverID) {
			out = append(out, session)
		}
	}
	return out
}

type serverCapabilityChanges struct {
	toolsChanged     bool
	resourcesChanged bool
	promptsChanged   bool
}

func detectServerCapabilityChange(sc *ServerContext) serverCapabilityChanges {
	if sc == nil {
		return serverCapabilityChanges{}
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	toolsChanged := sc.Tools != nil && len(sc.Tools.Tools) > 0
	resourcesChanged := (sc.Resources != nil && len(sc.Resources.Resources) > 0) || (sc.ResourceTemplates != nil && len(sc.ResourceTemplates.ResourceTemplates) > 0)
	promptsChanged := sc.Prompts != nil && len(sc.Prompts.Prompts) > 0

	return serverCapabilityChanges{
		toolsChanged:     toolsChanged,
		resourcesChanged: resourcesChanged,
		promptsChanged:   promptsChanged,
	}
}

func (m *serverManager) notifyUsersOfServerChange(serverID string, sessions []*ClientSession, changeType string, changed serverCapabilityChanges) {
	if strings.TrimSpace(serverID) != "" {
		m.NotifyUserPermissionChangedByServer(serverID)
	}

	if len(sessions) == 0 {
		return
	}

	if !changed.toolsChanged && !changed.resourcesChanged && !changed.promptsChanged {
		return
	}

	users := map[string]struct{}{}
	for _, session := range sessions {
		if session == nil {
			continue
		}
		users[session.UserID] = struct{}{}
		if ps := session.GetProxySession(); ps != nil {
			if changed.toolsChanged {
				ps.SendToolsListChangedToClient()
			}
			if changed.resourcesChanged {
				ps.SendResourcesListChangedToClient()
			}
			if changed.promptsChanged {
				ps.SendPromptsListChangedToClient()
			}
		}
	}

	log.Info().
		Str("serverId", serverID).
		Str("changeType", changeType).
		Bool("toolsChanged", changed.toolsChanged).
		Bool("resourcesChanged", changed.resourcesChanged).
		Bool("promptsChanged", changed.promptsChanged).
		Int("sessionCount", len(sessions)).
		Msg("notifying sessions of server change")

	m.mu.RLock()
	notifier := m.notifier
	m.mu.RUnlock()
	if notifier == nil {
		return
	}

	for userID := range users {
		go notifier.NotifyOnlineSessions(userID)
	}
}

func (m *serverManager) applyToolDefaultConfigFallback(ctx context.Context, sc *ServerContext) {
	if sc == nil {
		return
	}

	current := sc.CapabilitiesConfigSnapshot()
	if len(current.Tools) > 0 || len(current.Resources) > 0 || len(current.Prompts) > 0 {
		return
	}

	serverEntity := sc.ServerEntitySnapshot()
	if serverEntity.ConfigTemplate == nil {
		return
	}
	configTemplate := strings.TrimSpace(*serverEntity.ConfigTemplate)
	if configTemplate == "" || configTemplate == "{}" {
		return
	}

	template := map[string]any{}
	if err := json.Unmarshal([]byte(configTemplate), &template); err != nil {
		log.Warn().Err(err).Str("serverId", sc.ServerID).Msg("invalid configTemplate JSON")
		return
	}

	toolDefaultConfigRaw, exists := template["toolDefaultConfig"]
	if !exists || toolDefaultConfigRaw == nil {
		return
	}

	toolDefaultConfig, ok := parseToolDefaultConfig(toolDefaultConfigRaw)
	if !ok || len(toolDefaultConfig) == 0 {
		return
	}

	raw, err := json.Marshal(mcptypes.ServerConfigCapabilities{
		Tools:     toolDefaultConfig,
		Resources: map[string]mcptypes.ResourceCapabilityConfig{},
		Prompts:   map[string]mcptypes.PromptCapabilityConfig{},
	})
	if err != nil {
		return
	}

	if err := sc.UpdateCapabilitiesConfig(string(raw)); err != nil {
		log.Warn().Err(err).Str("serverId", sc.ServerID).Msg("failed to apply toolDefaultConfig fallback")
		return
	}
	m.persistToolDefaultConfigUserPreferences(ctx, sc)

	m.mu.RLock()
	repo := m.repo
	m.mu.RUnlock()
	if repo != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		if err := repo.UpdateCapabilities(ctx, sc.ServerID, string(raw)); err != nil {
			log.Warn().Err(err).Str("serverId", sc.ServerID).Msg("failed to persist toolDefaultConfig fallback")
		}
	}
}

func (m *serverManager) persistToolDefaultConfigUserPreferences(ctx context.Context, sc *ServerContext) {
	if sc == nil {
		return
	}

	serverEntity := sc.ServerEntitySnapshot()
	if !serverEntity.AllowUserInput {
		return
	}

	userID := strings.TrimSpace(sc.UserIDSnapshot())
	if userID == "" {
		return
	}

	m.mu.RLock()
	users := m.users
	m.mu.RUnlock()
	if users == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	db := database.DB
	if db == nil {
		return
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		var freshUser database.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&freshUser).Error; err != nil {
			return err
		}
		preferences := mcptypes.Permissions{}
		if strings.TrimSpace(freshUser.UserPreferences) != "" {
			if err := json.Unmarshal([]byte(freshUser.UserPreferences), &preferences); err != nil {
				preferences = mcptypes.Permissions{}
			}
		}
		if preferences == nil {
			preferences = mcptypes.Permissions{}
		}
		preferences[sc.ServerID] = sc.GetMCPCapabilities()
		encoded, err := json.Marshal(preferences)
		if err != nil {
			return err
		}
		return tx.Model(&database.User{}).Where("user_id = ?", userID).Updates(map[string]any{"user_preferences": string(encoded), "updated_at": int(time.Now().Unix())}).Error
	}); err != nil {
		log.Warn().Err(err).Str("serverId", sc.ServerID).Str("userId", userID).Msg("failed to persist toolDefaultConfig user preferences")
	}
}

func parseToolDefaultConfig(raw any) (map[string]mcptypes.ToolCapabilityConfig, bool) {
	if raw == nil {
		return nil, false
	}

	var payload []byte
	switch value := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, false
		}
		payload = []byte(trimmed)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, false
		}
		payload = encoded
	}

	parsed := map[string]mcptypes.ToolCapabilityConfig{}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, false
	}
	return parsed, true
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "timeout") || strings.Contains(errText, "deadline exceeded")
}

type subscriptionParts struct {
	serverID    string
	resourceURI string
}

const subscriptionKeySep = "::"

func makeSubscriptionKey(serverID, resourceURI string) string {
	return serverID + subscriptionKeySep + resourceURI
}

func splitSubscriptionKey(key string) subscriptionParts {
	parts := strings.SplitN(key, subscriptionKeySep, 2)
	if len(parts) == 1 {
		return subscriptionParts{serverID: parts[0]}
	}
	return subscriptionParts{serverID: parts[0], resourceURI: parts[1]}
}

type serverUserKeyParts struct {
	serverID string
	userID   string
}

func splitServerUserKey(key string) serverUserKeyParts {
	idx := -1
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return serverUserKeyParts{serverID: key}
	}
	return serverUserKeyParts{serverID: key[:idx], userID: key[idx+1:]}
}

func tempServerKey(serverID, userID string) string {
	return serverID + ":" + userID
}

type capabilitiesState struct {
	Tools             string
	Resources         string
	ResourceTemplates string
	Prompts           string
}

func (m *serverManager) snapshotCapabilitiesState(sc *ServerContext) capabilitiesState {
	sc.mu.RLock()
	tools := sc.Tools
	resources := sc.Resources
	resourceTemplates := sc.ResourceTemplates
	prompts := sc.Prompts
	sc.mu.RUnlock()

	return capabilitiesState{
		Tools:             marshalString(tools),
		Resources:         marshalString(resources),
		ResourceTemplates: marshalString(resourceTemplates),
		Prompts:           marshalString(prompts),
	}
}

func marshalString(v any) string {
	if v == nil {
		return ""
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (m *serverManager) contextMapKey(sc *ServerContext) string {
	if sc == nil {
		return ""
	}
	serverEntity := sc.ServerEntitySnapshot()
	if !serverEntity.AllowUserInput {
		return sc.ServerID
	}
	userID := sc.UserIDSnapshot()
	if userID == "" {
		return sc.ServerID
	}
	return sc.ServerID + ":" + userID
}

func (m *serverManager) loggerKeyForContext(sc *ServerContext) string {
	key := m.contextMapKey(sc)
	if key != "" {
		return key
	}
	if sc == nil {
		return ""
	}
	return sc.ServerID
}

func (m *serverManager) getServerLogger(serverKey string) *serverlog.ServerLogger {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.serverLoggers[serverKey]
}

func (m *serverManager) logServerLifecycle(serverKey string, action int, errMsg string) {
	logger := m.getServerLogger(serverKey)
	if logger == nil {
		return
	}
	logger.LogServerLifecycle(action, errMsg)
}

func (m *serverManager) logServerCapabilityUpdate(serverKey string, params any) {
	logger := m.getServerLogger(serverKey)
	if logger == nil {
		return
	}
	logger.LogServerCapabilityUpdate(params)
}

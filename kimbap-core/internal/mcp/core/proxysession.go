package core

import (
	"bufio"
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	internallog "github.com/dunialabs/kimbap-core/internal/log"
	mcpservices "github.com/dunialabs/kimbap-core/internal/mcp/services"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
	"gorm.io/datatypes"
)

const (
	upstreamRequestIDMetaKey = "kimbapUpstreamRequestId"
	approvalPollIntervalMs   = 3_000
	approvalWaitTimeoutMs    = 55_000
)

type ProxySession struct {
	sessionID string
	userID    string

	clientSession *ClientSession
	sessionLogger SessionLogger
	eventStore    *PersistentEventStore

	requestIDMapper *RequestIDMapper

	onClose func(sessionID string, reason mcptypes.DisconnectReason)

	upstreamServer *mcp.Server
	upstreamHTTP   *mcp.StreamableHTTPHandler

	closeOnce              sync.Once
	closeCallbackOnce      sync.Once
	upstreamCloseWatchOnce sync.Once

	// downstreamMu and reverseMu protect independent maps (downstreamCancels and
	// reverseCancels respectively). They must NEVER be held simultaneously to avoid
	// deadlock. Each function locks only one of the two; Cleanup() locks them sequentially.
	downstreamMu      sync.Mutex
	downstreamCancels map[string]context.CancelFunc

	reverseMu      sync.Mutex
	reverseCancels map[string]context.CancelFunc

	upstreamFallbackCounter uint64
	initialized             atomic.Bool
}

func NewProxySession(sessionID, userID string, clientSession *ClientSession, logger SessionLogger, eventStore *PersistentEventStore, onClose func(string, mcptypes.DisconnectReason)) *ProxySession {
	ps := &ProxySession{
		sessionID:         sessionID,
		userID:            userID,
		clientSession:     clientSession,
		sessionLogger:     logger,
		eventStore:        eventStore,
		requestIDMapper:   NewRequestIDMapper(sessionID),
		onClose:           onClose,
		downstreamCancels: map[string]context.CancelFunc{},
		reverseCancels:    map[string]context.CancelFunc{},
	}
	ps.setupRequestHandlers()
	return ps
}

func (p *ProxySession) setupRequestHandlers() {
	serverOptions := &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools:       &mcp.ToolCapabilities{ListChanged: true},
			Resources:   &mcp.ResourceCapabilities{ListChanged: true, Subscribe: true},
			Prompts:     &mcp.PromptCapabilities{ListChanged: true},
			Completions: &mcp.CompletionCapabilities{},
			Logging:     &mcp.LoggingCapabilities{},
		},
		InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
			_ = ctx
			if req != nil {
				p.watchUpstreamSessionClose(req.Session)
				p.captureClientInfoFromSession(req.Session)
			}
			p.initialized.Store(true)
			p.clientSession.ConnectionInitialized(p.upstreamServer)
		},
		CompletionHandler: func(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
			if req == nil {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "missing completion request"}
			}
			return p.handleComplete(ctx, req.Params, p.upstreamRequestID(req, "completion/complete"), 0)
		},
		SubscribeHandler: func(ctx context.Context, req *mcp.SubscribeRequest) error {
			if req == nil {
				return &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "missing subscribe request"}
			}
			return p.HandleSubscribe(ctx, req.Params, p.upstreamRequestID(req, "resources/subscribe"))
		},
		UnsubscribeHandler: func(ctx context.Context, req *mcp.UnsubscribeRequest) error {
			if req == nil {
				return &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "missing unsubscribe request"}
			}
			return p.HandleUnsubscribe(ctx, req.Params, p.upstreamRequestID(req, "resources/unsubscribe"))
		},
		GetSessionID: func() string {
			return p.sessionID
		},
	}
	p.setupNotificationHandlers(serverOptions)

	p.upstreamServer = mcp.NewServer(
		&mcp.Implementation{Name: config.AppInfo.Name, Version: config.AppInfo.Version},
		serverOptions,
	)

	p.upstreamServer.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			switch method {
			case "initialize":
				if session, ok := req.GetSession().(*mcp.ServerSession); ok {
					p.watchUpstreamSessionClose(session)
				}
				return next(ctx, method, req)
			case "ping":
				if err := p.handlePing(ctx, req, p.upstreamRequestID(req, "ping")); err != nil {
					return nil, err
				}
				return next(ctx, method, req)
			case "logging/setLevel":
				p.handleLoggingSetLevel(ctx, req, p.upstreamRequestID(req, "logging/setLevel"))
				return next(ctx, method, req)
			case "tools/list":
				return p.handleToolsList(ctx, req, p.upstreamRequestID(req, "tools/list")), nil
			case "resources/list":
				return p.handleResourcesList(ctx, req, p.upstreamRequestID(req, "resources/list")), nil
			case "resources/templates/list":
				return p.handleResourcesTemplatesList(ctx, req, p.upstreamRequestID(req, "resources/templates/list")), nil
			case "prompts/list":
				return p.handlePromptsList(ctx, req, p.upstreamRequestID(req, "prompts/list")), nil
			case "tools/call":
				toolReq, ok := req.(*mcp.CallToolRequest)
				if !ok || toolReq == nil {
					return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "invalid tools/call request"}
				}
				params, err := p.normalizeCallToolParams(toolReq.Params)
				if err != nil {
					return nil, err
				}
				return p.handleToolCall(ctx, params, p.upstreamRequestID(req, "tools/call"), 0)
			case "resources/read":
				resourceReq, ok := req.(*mcp.ReadResourceRequest)
				if !ok || resourceReq == nil {
					return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "invalid resources/read request"}
				}
				return p.handleResourceRead(ctx, resourceReq.Params, p.upstreamRequestID(req, "resources/read"), 0)
			case "prompts/get":
				promptReq, ok := req.(*mcp.GetPromptRequest)
				if !ok || promptReq == nil {
					return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "invalid prompts/get request"}
				}
				return p.handlePromptGet(ctx, promptReq.Params, p.upstreamRequestID(req, "prompts/get"), 0)
			case "notifications/cancelled":
				if req != nil && req.GetParams() != nil {
					if params, ok := req.GetParams().(*mcp.CancelledParams); ok {
						p.cancelDownstreamByOriginalRequestID(params.RequestID)
					}
				}
				return next(ctx, method, req)
			default:
				return next(ctx, method, req)
			}
		}
	})

	p.upstreamHTTP = mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return p.upstreamServer
	}, &mcp.StreamableHTTPOptions{})
}

func (p *ProxySession) setupNotificationHandlers(options *mcp.ServerOptions) {
	if options == nil {
		return
	}

	options.RootsListChangedHandler = p.handleRootsListChanged
}

func (p *ProxySession) handleRootsListChanged(ctx context.Context, _ *mcp.RootsListChangedRequest) {
	if !p.clientSession.CanRequestRoots() {
		return
	}

	servers := p.clientSession.GetAvailableServers()
	if len(servers) == 0 {
		return
	}

	for _, server := range servers {
		if server == nil {
			continue
		}

		server.mu.RLock()
		conn := server.mcpConn
		serverID := server.ServerID
		server.mu.RUnlock()

		if conn == nil {
			continue
		}

		if err := conn.Write(ctx, &jsonrpc.Request{Method: "notifications/roots/list_changed"}); err != nil {
			log.Warn().Err(err).Str("sessionId", p.sessionID).Str("serverId", serverID).Msg("failed to forward roots list changed notification to downstream")
		}
	}
}

func (p *ProxySession) handleToolsList(ctx context.Context, req mcp.Request, upstreamRequestID any) *annotatedListToolsResult {
	startedAt := time.Now()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	uniformRequestID := p.generateUniformRequestID()
	requestParams := paramsFromRequest(req)
	allTools := p.clientSession.ListTools()
	filteredTools := make([]*mcp.Tool, 0, len(allTools.Tools))
	filteredAnnotatedTools := make([]*annotatedTool, 0, len(allTools.Tools))
	for i, tool := range allTools.Tools {
		if tool == nil || !p.serverSupportsToolsList(tool.Name) {
			continue
		}
		filteredTools = append(filteredTools, tool)
		if i < len(allTools.annotatedTools) && allTools.annotatedTools[i] != nil {
			filteredAnnotatedTools = append(filteredAnnotatedTools, allTools.annotatedTools[i])
			continue
		}
		filteredAnnotatedTools = append(filteredAnnotatedTools, &annotatedTool{Tool: tool})
	}
	allTools = &annotatedListToolsResult{
		ListToolsResult: mcp.ListToolsResult{
			Tools:      filteredTools,
			Meta:       mcp.Meta{"totalCount": len(filteredTools)},
			NextCursor: allTools.NextCursor,
		},
		annotatedTools: filteredAnnotatedTools,
	}
	toolNames := make([]string, 0, len(allTools.Tools))
	for _, tool := range allTools.Tools {
		if tool == nil {
			continue
		}
		toolNames = append(toolNames, tool.Name)
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeResponseToolList, map[string]any{
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     requestParams,
		"responseResult":    map[string]any{"tools": toolNames},
		"duration":          durationMillis(startedAt),
		"statusCode":        http.StatusOK,
	})
	return allTools
}

func (p *ProxySession) handleResourcesList(ctx context.Context, req mcp.Request, upstreamRequestID any) *mcp.ListResourcesResult {
	startedAt := time.Now()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	uniformRequestID := p.generateUniformRequestID()
	requestParams := paramsFromRequest(req)
	allResources := p.clientSession.ListResources()
	filteredResources := make([]*mcp.Resource, 0, len(allResources.Resources))
	for _, resource := range allResources.Resources {
		if resource == nil || !p.serverSupportsResourcesList(resource.URI) {
			continue
		}
		filteredResources = append(filteredResources, resource)
	}
	allResources = &mcp.ListResourcesResult{
		Resources:  filteredResources,
		Meta:       mcp.Meta{"totalCount": len(filteredResources)},
		NextCursor: allResources.NextCursor,
	}
	resourceURIs := make([]string, 0, len(allResources.Resources))
	for _, resource := range allResources.Resources {
		if resource == nil {
			continue
		}
		resourceURIs = append(resourceURIs, resource.URI)
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeResponseResourceList, map[string]any{
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     requestParams,
		"responseResult":    map[string]any{"resources": resourceURIs},
		"duration":          durationMillis(startedAt),
		"statusCode":        http.StatusOK,
	})
	return allResources
}

func (p *ProxySession) serverSupportsToolsList(prefixedToolName string) bool {
	serverID, _, ok := p.clientSession.ParseName(prefixedToolName)
	if !ok {
		return false
	}
	serverContext := ServerManagerInstance().GetServerContext(serverID, p.userID)
	if serverContext == nil {
		return false
	}
	caps, _, _ := snapshotServerCapabilities(serverContext)
	return caps.Tools != nil
}

func (p *ProxySession) serverSupportsResourcesList(prefixedResourceURI string) bool {
	serverID, _, ok := p.clientSession.ParseName(prefixedResourceURI)
	if !ok {
		return false
	}
	serverContext := ServerManagerInstance().GetServerContext(serverID, p.userID)
	if serverContext == nil {
		return false
	}
	caps, _, _ := snapshotServerCapabilities(serverContext)
	return caps.Resources != nil
}

func (p *ProxySession) handleResourcesTemplatesList(ctx context.Context, req mcp.Request, upstreamRequestID any) *mcp.ListResourceTemplatesResult {
	startedAt := time.Now()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	uniformRequestID := p.generateUniformRequestID()
	requestParams := paramsFromRequest(req)
	allTemplates := p.clientSession.ListResourceTemplates()
	templateNames := make([]string, 0, len(allTemplates.ResourceTemplates))
	for _, template := range allTemplates.ResourceTemplates {
		if template == nil {
			continue
		}
		if strings.TrimSpace(template.Name) != "" {
			templateNames = append(templateNames, template.Name)
			continue
		}
		templateNames = append(templateNames, template.URITemplate)
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeResponseResourceList, map[string]any{
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     requestParams,
		"responseResult":    map[string]any{"resourceTemplates": templateNames},
		"duration":          durationMillis(startedAt),
		"statusCode":        http.StatusOK,
	})
	return allTemplates
}

func (p *ProxySession) handlePromptsList(ctx context.Context, req mcp.Request, upstreamRequestID any) *mcp.ListPromptsResult {
	startedAt := time.Now()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	uniformRequestID := p.generateUniformRequestID()
	requestParams := paramsFromRequest(req)
	allPrompts := p.clientSession.ListPrompts()
	promptNames := make([]string, 0, len(allPrompts.Prompts))
	for _, prompt := range allPrompts.Prompts {
		if prompt == nil {
			continue
		}
		promptNames = append(promptNames, prompt.Name)
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeResponsePromptList, map[string]any{
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     requestParams,
		"responseResult":    map[string]any{"prompts": promptNames},
		"duration":          durationMillis(startedAt),
		"statusCode":        http.StatusOK,
	})
	return allPrompts
}

func (p *ProxySession) handlePing(ctx context.Context, req mcp.Request, upstreamRequestID any) error {
	startedAt := time.Now()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	uniformRequestID := p.generateUniformRequestID()
	requestParams := paramsFromRequest(req)
	session := p.firstUpstreamSession()
	if session == nil {
		p.logClientAudit(ctx, types.MCPEventLogTypeServerNotification, map[string]any{
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     requestParams,
			"error":             "upstream server not initialized",
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusServiceUnavailable,
		})
		return errors.New("upstream server not initialized")
	}
	if err := session.Ping(ctx, &mcp.PingParams{}); err != nil {
		p.logClientAudit(ctx, types.MCPEventLogTypeServerNotification, map[string]any{
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     requestParams,
			"error":             err.Error(),
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusBadGateway,
		})
		return err
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeServerNotification, map[string]any{
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     requestParams,
		"responseResult":    map[string]any{},
		"duration":          durationMillis(startedAt),
		"statusCode":        http.StatusOK,
	})
	return nil
}

func (p *ProxySession) handleLoggingSetLevel(ctx context.Context, req mcp.Request, upstreamRequestID any) {
	startedAt := time.Now()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	uniformRequestID := p.generateUniformRequestID()
	requestParams := paramsFromRequest(req)
	level := ""
	if req != nil && req.GetParams() != nil {
		if params, ok := req.GetParams().(*mcp.SetLoggingLevelParams); ok && params != nil {
			level = string(params.Level)
		}
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeServerNotification, map[string]any{
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     requestParams,
		"responseResult":    map[string]any{},
		"duration":          durationMillis(startedAt),
		"statusCode":        http.StatusOK,
	})
	log.Debug().Str("sessionId", p.sessionID).Str("level", level).Msg("accepted logging level change without forwarding")
}

func (p *ProxySession) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if p.upstreamHTTP == nil {
		http.Error(w, "proxy session transport is not initialized", http.StatusInternalServerError)
		return
	}
	// Inject the JSON-RPC request ID into params._meta so that upstreamRequestID()
	// can correlate upstream↔downstream requests (needed for cancellation mapping).
	// The go-sdk doesn't expose the JSON-RPC ID at the middleware level, so we
	// read it from the raw body before the SDK parses it.
	if r.Method == http.MethodPost {
		if err := p.injectUpstreamRequestIDMeta(r); err != nil {
			log.Warn().Err(err).Str("sessionId", p.sessionID).Msg("failed to inject upstream request id meta")
		}
	}
	if p.eventStore == nil || r.Method != http.MethodGet {
		p.upstreamHTTP.ServeHTTP(w, r)
		return
	}
	p.upstreamHTTP.ServeHTTP(newSSEPersistingResponseWriter(w, r.Context(), p.eventStore, p.sessionID), r)
}

func (p *ProxySession) captureClientInfoFromSession(session *mcp.ServerSession) {
	if session == nil {
		return
	}

	params := session.InitializeParams()
	if params == nil {
		return
	}

	if params.ClientInfo != nil {
		p.clientSession.SetClientInfo(&ClientInfo{
			Name:    params.ClientInfo.Name,
			Version: params.ClientInfo.Version,
		})
	}

	if params.Capabilities != nil {
		caps := map[string]any{}
		if params.Capabilities.Sampling != nil {
			caps["sampling"] = params.Capabilities.Sampling
		}
		if params.Capabilities.Elicitation != nil {
			caps["elicitation"] = params.Capabilities.Elicitation
		}
		if params.Capabilities.RootsV2 != nil {
			caps["roots"] = params.Capabilities.RootsV2
		}
		p.clientSession.mu.Lock()
		p.clientSession.Capabilities = caps
		p.clientSession.mu.Unlock()
	}
}

func (p *ProxySession) watchUpstreamSessionClose(session *mcp.ServerSession) {
	if session == nil {
		return
	}

	p.upstreamCloseWatchOnce.Do(func() {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("sessionId", p.sessionID).Msg("recovered panic in upstream session watcher")
				}
			}()
			if err := session.Wait(); err != nil && !errors.Is(err, net.ErrClosed) {
				log.Debug().Err(err).Str("sessionId", p.sessionID).Msg("upstream session wait ended with error")
			}
			p.triggerOnClose(mcptypes.DisconnectReasonClientDisconnect)
		}()
	})
}

func (p *ProxySession) triggerOnClose(reason mcptypes.DisconnectReason) {
	p.closeCallbackOnce.Do(func() {
		if p.onClose != nil {
			p.onClose(p.sessionID, reason)
		}
	})
}

func (p *ProxySession) HandleReconnection(ctx context.Context, w http.ResponseWriter, lastEventID string) error {
	replay := NewEventReplayService(p.eventStore)
	if err := replay.ReplayAfter(ctx, w, lastEventID, p.sessionID); err != nil {
		log.Error().Err(err).Str("sessionId", p.sessionID).Str("lastEventId", lastEventID).Msg("failed to handle reconnection")
		return err
	}
	return nil
}

// auditFields builds a fresh audit entry map with the common fields shared across all
// downstream-request handler audit log calls. Extra fields (error, duration, statusCode,
// responseResult, etc.) are merged in; the caller must supply them per log site.
func auditFields(serverID, upstreamRequestIDStr, uniformRequestID string, params any, extra map[string]any) map[string]any {
	entry := map[string]any{
		"serverId":          serverID,
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     params,
	}
	for k, v := range extra {
		entry[k] = v
	}
	return entry
}

func (p *ProxySession) handleToolCall(ctx context.Context, params *mcp.CallToolParams, upstreamRequestID any, retryCount int) (*mcp.CallToolResult, error) {
	startedAt := time.Now()
	uniformRequestID := p.generateUniformRequestID()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	if params == nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "tool call params are required"}
	}
	serverID, originalToolName, ok := p.clientSession.ParseName(params.Name)
	if !ok {
		errMsg := fmt.Sprintf("Tool %s not found", params.Name)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields("unknown", upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusNotFound,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeMethodNotFound, Message: errMsg}
	}
	if !p.clientSession.CanUseTool(serverID, originalToolName) {
		errMsg := fmt.Sprintf("Permission denied for tool: %s", params.Name)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorPermission", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusForbidden,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}

	serverCtx, err := ServerManagerInstance().EnsureServerAvailable(ctx, serverID, p.clientSession.UserID)
	if err != nil {
		errMsg := fmt.Sprintf("No server available for tool: %s", params.Name)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusServiceUnavailable,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
	}

	serverDangerLevel := serverCtx.GetDangerLevel(originalToolName)
	userDangerLevel := p.clientSession.EffectiveDangerLevel(serverID, originalToolName, serverDangerLevel)
	dangerLevel := types.DangerLevelSilent
	if userDangerLevel != nil {
		dangerLevel = *userDangerLevel
	} else if serverDangerLevel != nil {
		dangerLevel = *serverDangerLevel
	}

	toolArgs := toolArgsFromAny(params.Arguments)
	policyResult, err := mcpservices.PolicyEngineInstance().Evaluate(mcpservices.PolicyEvaluateParams{
		UserID:      p.userID,
		ServerID:    &serverID,
		ToolName:    originalToolName,
		Args:        toolArgs,
		DangerLevel: dangerLevel,
	})
	if err != nil {
		errMsg := "Policy evaluation failed, please retry"
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorPolicy", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusInternalServerError,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
	}

	if policyResult.Decision == types.PolicyDecisionDeny {
		errMsg := fmt.Sprintf("Tool execution denied by policy: %s", stringOrDefault(policyResult.Reason, "policy rule"))
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("PolicyDenied", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusForbidden,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: errMsg}
	}

	approvalRequestID := ""
	if policyResult.Decision == types.PolicyDecisionRequireApproval {
		approvalCheck, approvalErr := mcpservices.ApprovalServiceInstance().CheckOrCreateApproval(mcpservices.CreateApprovalInput{
			UserID:           p.userID,
			ServerID:         &serverID,
			ToolName:         originalToolName,
			Args:             toolArgs,
			PolicyVersion:    policyResult.PolicyVersion,
			UniformRequestID: &uniformRequestID,
		})
		if approvalErr != nil {
			var rateLimitErr *mcpservices.ApprovalRateLimitError
			if errors.As(approvalErr, &rateLimitErr) {
				errMsg := rateLimitErr.Error()
				p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
					"error":      formatCategorizedError("ApprovalRateLimited", errMsg),
					"duration":   durationMillis(startedAt),
					"statusCode": http.StatusTooManyRequests,
				}))
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("TOOL NOT EXECUTED: %s", errMsg)}},
				}, nil
			}
			errMsg := "failed to create approval request"
			p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
				"error":      formatCategorizedError("ErrorApproval", errMsg),
				"duration":   durationMillis(startedAt),
				"statusCode": http.StatusInternalServerError,
			}))
			return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
		}

		if approvalCheck.Request == nil || strings.TrimSpace(approvalCheck.Request.ID) == "" {
			errMsg := "approval request missing identifier"
			p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
				"error":      formatCategorizedError("ErrorApproval", errMsg),
				"duration":   durationMillis(startedAt),
				"statusCode": http.StatusInternalServerError,
			}))
			return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
		}

		if approvalCheck.NeedsApproval {
			if notifier := ServerManagerInstance().Notifier(); notifier != nil && approvalCheck.Request != nil && approvalCheck.Created {
				notifier.NotifyApprovalCreated(
					p.userID,
					approvalCheck.Request.ID,
					originalToolName,
					&serverID,
					approvalCheck.Request.RedactedArgs,
					approvalCheck.Request.ExpiresAt,
					approvalCheck.Request.CreatedAt,
					approvalCheck.Request.Status,
					&uniformRequestID,
					policyResult.PolicyVersion,
					policyResult.MatchedRuleID,
					policyResult.Reason,
				)
			}

			waitStartedAt := time.Now()
			keepaliveDone := make(chan struct{})
			go func() {
				ticker := time.NewTicker(10 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						p.emitApprovalWaitKeepalive(upstreamRequestID, approvalCheck.Request.ID, originalToolName, time.Since(waitStartedAt))
					case <-keepaliveDone:
						return
					}
				}
			}()

			p.emitApprovalWaitKeepalive(upstreamRequestID, approvalCheck.Request.ID, originalToolName, 0)
			waitStatus, decisionReason := p.waitForApproval(ctx, approvalCheck.Request.ID)
			close(keepaliveDone)

			switch waitStatus {
			case "approved":
			case "rejected":
				reason := stringOrDefault(decisionReason, "Approval request was rejected")
				rejectedText := fmt.Sprintf("TOOL REJECTED: %s\n\nApproval Request ID: %s\nTool: %s", reason, approvalCheck.Request.ID, originalToolName)
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: rejectedText}}}, nil
			case "executing":
				reason := stringOrDefault(policyResult.Reason, "Execution is already in progress in another session")
				if replayResult := p.tryBuildReplayToolResult(approvalCheck.Request.ID); replayResult != nil {
					p.logClientAudit(ctx, types.MCPEventLogTypeResponseTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
						"responseResult": replayResult,
						"duration":       durationMillis(startedAt),
						"statusCode":     http.StatusOK,
					}))
					return replayResult, nil
				}
				return p.buildApprovalOutcomeResult(approvalCheck.Request.ID, originalToolName, reason, policyResult.PolicyVersion, approvalCheck.RequestHash, waitStatus, "Execution is currently in progress elsewhere."), nil
			case "timeout":
				reason := stringOrDefault(policyResult.Reason, "Approval decision not received within wait window")
				return p.buildApprovalOutcomeResult(approvalCheck.Request.ID, originalToolName, reason, policyResult.PolicyVersion, approvalCheck.RequestHash, waitStatus, "Approval is still pending after waiting 55 seconds."), nil
			case "expired":
				reason := stringOrDefault(policyResult.Reason, "Approval request expired before a decision")
				return p.buildApprovalOutcomeResult(approvalCheck.Request.ID, originalToolName, reason, policyResult.PolicyVersion, approvalCheck.RequestHash, waitStatus, "Approval request expired before execution."), nil
			case "aborted":
				reason := stringOrDefault(policyResult.Reason, "Request was canceled while waiting for approval")
				return p.buildApprovalOutcomeResult(approvalCheck.Request.ID, originalToolName, reason, policyResult.PolicyVersion, approvalCheck.RequestHash, waitStatus, "Approval wait was aborted by cancellation."), nil
			case "executed":
				reason := stringOrDefault(policyResult.Reason, "Execution already completed by another session")
				if replayResult := p.tryBuildReplayToolResult(approvalCheck.Request.ID); replayResult != nil {
					p.logClientAudit(ctx, types.MCPEventLogTypeResponseTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
						"responseResult": replayResult,
						"duration":       durationMillis(startedAt),
						"statusCode":     http.StatusOK,
					}))
					return replayResult, nil
				}
				return p.buildApprovalOutcomeResult(approvalCheck.Request.ID, originalToolName, reason, policyResult.PolicyVersion, approvalCheck.RequestHash, waitStatus, "Execution was already completed elsewhere."), nil
			case "failed":
				reason := stringOrDefault(decisionReason, "Approval request failed before execution")
				if replayResult := p.tryBuildReplayToolResult(approvalCheck.Request.ID); replayResult != nil {
					statusCode := http.StatusOK
					if replayResult.IsError {
						statusCode = http.StatusInternalServerError
					}
					p.logClientAudit(ctx, types.MCPEventLogTypeResponseTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
						"responseResult": replayResult,
						"duration":       durationMillis(startedAt),
						"statusCode":     statusCode,
					}))
					return replayResult, nil
				}
				summary := "Approval request is in failed state."
				if reason == "Approval status check failed" {
					summary = "Approval status is temporarily unavailable."
				}
				return p.buildApprovalOutcomeResult(approvalCheck.Request.ID, originalToolName, reason, policyResult.PolicyVersion, approvalCheck.RequestHash, waitStatus, summary), nil
			default:
				reason := stringOrDefault(policyResult.Reason, "Approval wait ended without executable approval")
				return p.buildApprovalOutcomeResult(approvalCheck.Request.ID, originalToolName, reason, policyResult.PolicyVersion, approvalCheck.RequestHash, waitStatus, "Approval is not ready for execution."), nil
			}
		}

		claimResult, claimErr := mcpservices.ApprovalServiceInstance().ClaimForExecutionById(approvalCheck.Request.ID)
		if claimErr != nil {
			errMsg := "Approval request could not be claimed for execution"
			p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
				"error":      formatCategorizedError("ApprovalClaimFailed", errMsg),
				"duration":   durationMillis(startedAt),
				"statusCode": http.StatusConflict,
			}))
			return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: errMsg}
		}
		if !claimResult.Claimed {
			if replayResult := p.tryBuildReplayToolResult(approvalCheck.Request.ID); replayResult != nil {
				statusCode := http.StatusOK
				if replayResult.IsError {
					statusCode = http.StatusInternalServerError
				}
				p.logClientAudit(ctx, types.MCPEventLogTypeResponseTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
					"responseResult": replayResult,
					"duration":       durationMillis(startedAt),
					"statusCode":     statusCode,
				}))
				return replayResult, nil
			}
			return &mcp.CallToolResult{
				IsError: false,
				Content: []mcp.Content{&mcp.TextContent{
					Text: "TOOL NOT EXECUTED: Execution is already in progress or completed elsewhere.",
				}},
			}, nil
		}

		if claimResult.Request != nil && strings.TrimSpace(claimResult.Request.ID) != "" {
			approvalRequestID = claimResult.Request.ID
		} else {
			approvalRequestID = approvalCheck.Request.ID
		}
	}

	markApprovalFailed := func(message string, replayResult *mcp.CallToolResult) {
		if strings.TrimSpace(approvalRequestID) == "" {
			return
		}
		failedReq, _ := mcpservices.ApprovalServiceInstance().MarkFailed(approvalRequestID, message, replayResult)
		if failedReq != nil {
			if notifier := ServerManagerInstance().Notifier(); notifier != nil {
				replayPayload := p.extractReplayCallToolResult(failedReq.ExecutionResult)
				notifier.NotifyApprovalFailed(p.userID, failedReq.ID, originalToolName, message, replayPayload != nil, p.buildExecutionResultPreview(replayPayload))
			}
		}
		approvalRequestID = ""
	}
	markApprovalExecuted := func(result *mcp.CallToolResult) {
		if strings.TrimSpace(approvalRequestID) == "" {
			return
		}
		executedReq, _ := mcpservices.ApprovalServiceInstance().MarkExecuted(approvalRequestID, result)
		if executedReq != nil {
			if notifier := ServerManagerInstance().Notifier(); notifier != nil {
				replayPayload := p.extractReplayCallToolResult(executedReq.ExecutionResult)
				notifier.NotifyApprovalExecuted(p.userID, executedReq.ID, originalToolName, replayPayload != nil, p.buildExecutionResultPreview(replayPayload))
			}
		}
		approvalRequestID = ""
	}

	conn := serverCtx.ConnectionSnapshot()
	if conn == nil {
		errMsg := fmt.Sprintf("No server available for tool: %s", params.Name)
		markApprovalFailed(errMsg, nil)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusServiceUnavailable,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
	}

	proxyRequestID := p.requestIDMapper.RegisterClientRequest(upstreamRequestID, "tools/call", serverID)
	defer p.requestIDMapper.RemoveMapping(proxyRequestID)

	// TODO(runtime-adapter): Check if RuntimeAdapter.CanHandle(toolName) before
	// proxying to downstream. If handled, use RuntimeAdapter.ExecuteToolCall()
	// instead. This enables policy/vault/audit for MCP-initiated actions.
	copyParams := *params
	copyParams.Name = originalToolName
	copyParams.Meta = p.withProxyContext(copyParams.Meta, proxyRequestID, uniformRequestID)
	copyParams.Meta["relatedRequestId"] = proxyRequestID
	p.logClientAudit(ctx, types.MCPEventLogTypeRequestTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, nil))
	log.Debug().Str("sessionId", p.sessionID).Str("serverId", serverID).Str("tool", originalToolName).Interface("upstreamRequestId", upstreamRequestID).Msg("forwarding tool call to downstream")
	callCtx, cancel := context.WithCancel(ctx)
	p.trackDownstreamCancel(proxyRequestID, cancel)
	defer cancel()
	defer p.untrackDownstreamCancel(proxyRequestID)
	serverCtx.IncrementActiveRequests()
	defer serverCtx.DecrementActiveRequests()
	if approvalRequestID != "" {
		heartbeatApprovalID := approvalRequestID
		heartbeatDone := make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_, _ = mcpservices.ApprovalServiceInstance().TouchExecuting(heartbeatApprovalID)
				case <-heartbeatDone:
					return
				}
			}
		}()
		defer close(heartbeatDone)
	}
	result, err := conn.CallTool(callCtx, &copyParams)
	if err != nil {
		hadApproval := strings.TrimSpace(approvalRequestID) != ""
		p.logClientAudit(ctx, types.MCPEventLogTypeResponseTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      err.Error(),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusInternalServerError,
		}))
		markApprovalFailed(err.Error(), nil)
		timeoutRecovery := serverCtx.RecordTimeoutWithRecovery(ctx, err)
		if !hadApproval && timeoutRecovery != nil && !*timeoutRecovery && retryCount < 2 {
			return p.handleToolCall(ctx, params, upstreamRequestID, retryCount+1)
		}
		log.Error().Err(err).Str("sessionId", p.sessionID).Str("serverId", serverID).Str("tool", originalToolName).Msg("tool call failed")
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: sanitizeMCPError(err)}}}, nil
	}
	if result == nil {
		errMsg := "tool call returned empty result"
		markApprovalFailed(errMsg, nil)
		p.logClientAudit(ctx, types.MCPEventLogTypeResponseTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      errMsg,
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusInternalServerError,
		}))
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: errMsg}}}, nil
	}
	serverCtx.ClearTimeout()
	statusCode := http.StatusOK
	if result != nil && result.IsError {
		statusCode = http.StatusInternalServerError
		markApprovalFailed("tool execution returned error", result)
	} else {
		markApprovalExecuted(result)
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeResponseTool, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
		"responseResult": result,
		"duration":       durationMillis(startedAt),
		"statusCode":     statusCode,
	}))
	log.Debug().Str("sessionId", p.sessionID).Str("serverId", serverID).Str("tool", originalToolName).Msg("tool call succeeded")
	return result, nil
}

func (p *ProxySession) handleResourceRead(ctx context.Context, params *mcp.ReadResourceParams, upstreamRequestID any, retryCount int) (*mcp.ReadResourceResult, error) {
	startedAt := time.Now()
	uniformRequestID := p.generateUniformRequestID()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	if params == nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "resource read params are required"}
	}
	serverID, originalURI, ok := p.clientSession.ParseName(params.URI)
	if !ok {
		errMsg := fmt.Sprintf("Invalid resource URI: %s", params.URI)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, auditFields("unknown", upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusNotFound,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}
	resourceName := originalURI
	if sc := ServerManagerInstance().GetServerContext(serverID, p.clientSession.UserID); sc != nil {
		resourceName = sc.ResolveResourceNameByURI(originalURI)
	}
	if !p.clientSession.CanAccessResource(serverID, resourceName) {
		errMsg := fmt.Sprintf("Permission denied for resource: %s", params.URI)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorPermission", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusForbidden,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}
	serverCtx, err := ServerManagerInstance().EnsureServerAvailable(ctx, serverID, p.clientSession.UserID)
	if err != nil {
		errMsg := fmt.Sprintf("No server available for resource: %s", params.URI)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusServiceUnavailable,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
	}
	conn := serverCtx.ConnectionSnapshot()
	if conn == nil {
		errMsg := fmt.Sprintf("No client available for resource: %s", params.URI)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusServiceUnavailable,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
	}

	proxyRequestID := p.requestIDMapper.RegisterClientRequest(upstreamRequestID, "resources/read", serverID)
	defer p.requestIDMapper.RemoveMapping(proxyRequestID)

	copyParams := *params
	copyParams.URI = originalURI
	copyParams.Meta = p.withProxyContext(copyParams.Meta, proxyRequestID, uniformRequestID)
	p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, nil))
	log.Debug().Str("sessionId", p.sessionID).Str("serverId", serverID).Str("resource", originalURI).Interface("upstreamRequestId", upstreamRequestID).Msg("forwarding resource read to downstream")
	callCtx, cancel := context.WithCancel(ctx)
	p.trackDownstreamCancel(proxyRequestID, cancel)
	defer cancel()
	defer p.untrackDownstreamCancel(proxyRequestID)
	serverCtx.IncrementActiveRequests()
	defer serverCtx.DecrementActiveRequests()
	result, err := conn.ReadResource(callCtx, &copyParams)
	if err != nil {
		p.logClientAudit(ctx, types.MCPEventLogTypeResponseResource, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      err.Error(),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusInternalServerError,
		}))
		timeoutRecovery := serverCtx.RecordTimeoutWithRecovery(ctx, err)
		if timeoutRecovery != nil && !*timeoutRecovery && retryCount < 2 {
			return p.handleResourceRead(ctx, params, upstreamRequestID, retryCount+1)
		}
		return nil, fmt.Errorf("%s", sanitizeMCPError(err))
	}
	serverCtx.ClearTimeout()
	p.logClientAudit(ctx, types.MCPEventLogTypeResponseResource, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
		"responseResult": result,
		"duration":       durationMillis(startedAt),
		"statusCode":     http.StatusOK,
	}))
	return result, nil
}

func (p *ProxySession) handlePromptGet(ctx context.Context, params *mcp.GetPromptParams, upstreamRequestID any, retryCount int) (*mcp.GetPromptResult, error) {
	startedAt := time.Now()
	uniformRequestID := p.generateUniformRequestID()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	if params == nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "prompt get params are required"}
	}
	serverID, originalPromptName, ok := p.clientSession.ParseName(params.Name)
	if !ok {
		errMsg := fmt.Sprintf("Invalid prompt name: %s", params.Name)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestPrompt, auditFields("unknown", upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusNotFound,
		}))
		return nil, &jsonrpc.Error{
			Code:    jsonrpc.CodeInvalidParams,
			Message: errMsg,
		}
	}
	if !p.clientSession.CanUsePrompt(serverID, originalPromptName) {
		errMsg := fmt.Sprintf("Permission denied for prompt: %s", params.Name)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestPrompt, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorPermission", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusForbidden,
		}))
		return nil, &jsonrpc.Error{
			Code:    jsonrpc.CodeInvalidParams,
			Message: errMsg,
		}
	}
	serverCtx, err := ServerManagerInstance().EnsureServerAvailable(ctx, serverID, p.clientSession.UserID)
	if err != nil {
		errMsg := fmt.Sprintf("No server available for prompt: %s", params.Name)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestPrompt, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusServiceUnavailable,
		}))
		return nil, &jsonrpc.Error{
			Code:    jsonrpc.CodeInternalError,
			Message: errMsg,
		}
	}
	conn := serverCtx.ConnectionSnapshot()
	if conn == nil {
		errMsg := fmt.Sprintf("No client available for prompt: %s", params.Name)
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestPrompt, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusServiceUnavailable,
		}))
		return nil, &jsonrpc.Error{
			Code:    jsonrpc.CodeInternalError,
			Message: errMsg,
		}
	}

	proxyRequestID := p.requestIDMapper.RegisterClientRequest(upstreamRequestID, "prompts/get", serverID)
	defer p.requestIDMapper.RemoveMapping(proxyRequestID)

	copyParams := *params
	copyParams.Name = originalPromptName
	copyParams.Meta = p.withProxyContext(copyParams.Meta, proxyRequestID, uniformRequestID)
	p.logClientAudit(ctx, types.MCPEventLogTypeRequestPrompt, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, nil))
	log.Debug().Str("sessionId", p.sessionID).Str("serverId", serverID).Str("prompt", originalPromptName).Interface("upstreamRequestId", upstreamRequestID).Msg("forwarding prompt get to downstream")
	callCtx, cancel := context.WithCancel(ctx)
	p.trackDownstreamCancel(proxyRequestID, cancel)
	defer cancel()
	defer p.untrackDownstreamCancel(proxyRequestID)
	serverCtx.IncrementActiveRequests()
	defer serverCtx.DecrementActiveRequests()
	result, err := conn.GetPrompt(callCtx, &copyParams)
	if err != nil {
		p.logClientAudit(ctx, types.MCPEventLogTypeResponsePrompt, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      err.Error(),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusInternalServerError,
		}))
		timeoutRecovery := serverCtx.RecordTimeoutWithRecovery(ctx, err)
		if timeoutRecovery != nil && !*timeoutRecovery && retryCount < 2 {
			return p.handlePromptGet(ctx, params, upstreamRequestID, retryCount+1)
		}
		return nil, fmt.Errorf("%s", sanitizeMCPError(err))
	}
	serverCtx.ClearTimeout()
	p.logClientAudit(ctx, types.MCPEventLogTypeResponsePrompt, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
		"responseResult": result,
		"duration":       durationMillis(startedAt),
		"statusCode":     http.StatusOK,
	}))
	return result, nil
}

func (p *ProxySession) handleComplete(ctx context.Context, params *mcp.CompleteParams, upstreamRequestID any, retryCount int) (*mcp.CompleteResult, error) {
	startedAt := time.Now()
	uniformRequestID := p.generateUniformRequestID()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	action := types.MCPEventLogTypeRequestPrompt
	if params == nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "completion params are required"}
	}
	if params.Ref == nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "completion reference is required"}
	}
	referenceName := params.Ref.Name
	referenceType := params.Ref.Type
	switch referenceType {
	case "ref/prompt":
	case "ref/resource":
		action = types.MCPEventLogTypeRequestResource
		referenceName = params.Ref.URI
	default:
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: fmt.Sprintf("Invalid completion reference: %v", params.Ref)}
	}
	responseAction := types.MCPEventLogTypeResponsePrompt
	if action == types.MCPEventLogTypeRequestResource {
		responseAction = types.MCPEventLogTypeResponseResource
	}
	serverID, originalName, ok := p.clientSession.ParseName(referenceName)
	if !ok {
		errMsg := fmt.Sprintf("Completion %s not found", referenceName)
		p.logClientAudit(ctx, action, auditFields("unknown", upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusNotFound,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: fmt.Sprintf("Invalid completion reference: %v", params.Ref)}
	}
	var permDenied bool
	switch referenceType {
	case "ref/prompt":
		permDenied = !p.clientSession.CanUsePrompt(serverID, originalName)
	case "ref/resource":
		resourceName := originalName
		if sc := ServerManagerInstance().GetServerContext(serverID, p.clientSession.UserID); sc != nil {
			resourceName = sc.ResolveResourceNameByURI(originalName)
		}
		permDenied = !p.clientSession.CanAccessResource(serverID, resourceName)
	}
	if permDenied {
		errMsg := fmt.Sprintf("Permission denied for completion: %s", referenceName)
		p.logClientAudit(ctx, action, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorPermission", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusForbidden,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}
	serverCtx, err := ServerManagerInstance().EnsureServerAvailable(ctx, serverID, p.clientSession.UserID)
	if err != nil {
		errMsg := fmt.Sprintf("No server available for completion: %s", referenceName)
		p.logClientAudit(ctx, action, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusServiceUnavailable,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
	}
	conn := serverCtx.ConnectionSnapshot()
	if conn == nil {
		errMsg := fmt.Sprintf("No client available for completion: %s", referenceName)
		p.logClientAudit(ctx, action, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      formatCategorizedError("ErrorRouting", errMsg),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusServiceUnavailable,
		}))
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: errMsg}
	}

	proxyRequestID := p.requestIDMapper.RegisterClientRequest(upstreamRequestID, "completion/complete", serverID)
	defer p.requestIDMapper.RemoveMapping(proxyRequestID)

	copyParams := *params
	refCopy := *params.Ref
	if refCopy.Type == "ref/resource" {
		refCopy.URI = originalName
	} else {
		refCopy.Name = originalName
	}
	copyParams.Ref = &refCopy
	copyParams.Meta = p.withProxyContext(copyParams.Meta, proxyRequestID, uniformRequestID)
	p.logClientAudit(ctx, action, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, nil))
	log.Debug().Str("sessionId", p.sessionID).Str("serverId", serverID).Str("referenceType", refCopy.Type).Str("reference", originalName).Interface("upstreamRequestId", upstreamRequestID).Msg("forwarding completion request to downstream")
	callCtx, cancel := context.WithCancel(ctx)
	p.trackDownstreamCancel(proxyRequestID, cancel)
	defer cancel()
	defer p.untrackDownstreamCancel(proxyRequestID)
	serverCtx.IncrementActiveRequests()
	result, err := conn.Complete(callCtx, &copyParams)
	serverCtx.DecrementActiveRequests()
	if err != nil {
		p.logClientAudit(ctx, responseAction, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
			"error":      err.Error(),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusInternalServerError,
		}))
		timeoutRecovery := serverCtx.RecordTimeoutWithRecovery(ctx, err)
		if timeoutRecovery != nil && !*timeoutRecovery && retryCount < 2 {
			return p.handleComplete(ctx, params, upstreamRequestID, retryCount+1)
		}
		return nil, fmt.Errorf("%s", sanitizeMCPError(err))
	}
	serverCtx.ClearTimeout()
	p.logClientAudit(ctx, responseAction, auditFields(serverID, upstreamRequestIDStr, uniformRequestID, params, map[string]any{
		"responseResult": result,
		"duration":       durationMillis(startedAt),
		"statusCode":     http.StatusOK,
	}))
	return result, nil
}

func (p *ProxySession) HandleSubscribe(ctx context.Context, params *mcp.SubscribeParams, upstreamRequestID any) error {
	startedAt := time.Now()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	uniformRequestID := p.generateUniformRequestID()
	if params == nil {
		errMsg := "subscribe params are required"
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     params,
			"error":             "ErrorRouting: " + errMsg,
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusBadRequest,
		})
		return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}
	serverID, originalURI, ok := p.clientSession.ParseName(params.URI)
	if !ok {
		errMsg := "Invalid resource URI: " + params.URI
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     params,
			"error":             "ErrorRouting: " + errMsg,
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusNotFound,
		})
		return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}
	resourceName := originalURI
	if sc := ServerManagerInstance().GetServerContext(serverID, p.clientSession.UserID); sc != nil {
		resourceName = sc.ResolveResourceNameByURI(originalURI)
	}
	if !p.clientSession.CanAccessResource(serverID, resourceName) {
		errMsg := "Permission denied for resource: " + params.URI
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
			"serverId":          serverID,
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     params,
			"error":             "ErrorPermission: " + errMsg,
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusForbidden,
		})
		return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}
	serverContext := ServerManagerInstance().GetServerContext(serverID, p.userID)
	if serverContext != nil {
		caps, _, _ := snapshotServerCapabilities(serverContext)
		if caps.Resources == nil || !caps.Resources.Subscribe {
			errMsg := "Server does not support resource subscription"
			p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
				"serverId":          serverID,
				"upstreamRequestId": upstreamRequestIDStr,
				"uniformRequestId":  uniformRequestID,
				"requestParams":     params,
				"error":             errMsg,
				"duration":          durationMillis(startedAt),
				"statusCode":        http.StatusForbidden,
			})
			return &jsonrpc.Error{Code: jsonrpc.CodeMethodNotFound, Message: errMsg}
		}
	}
	err := ServerManagerInstance().SubscribeResource(ctx, serverID, originalURI, p.sessionID, p.userID)
	if err != nil {
		errMsg := err.Error()
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
			"serverId":          serverID,
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     params,
			"error":             "ErrorSubscribe: " + errMsg,
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusInternalServerError,
		})
		return fmt.Errorf("%s", sanitizeMCPError(err))
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
		"serverId":          serverID,
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     params,
		"responseResult":    map[string]any{"subscribed": true},
		"duration":          durationMillis(startedAt),
		"statusCode":        http.StatusOK,
	})
	return nil
}

func (p *ProxySession) HandleUnsubscribe(ctx context.Context, params *mcp.UnsubscribeParams, upstreamRequestID any) error {
	startedAt := time.Now()
	upstreamRequestIDStr := fmt.Sprintf("%v", upstreamRequestID)
	uniformRequestID := p.generateUniformRequestID()
	if params == nil {
		errMsg := "unsubscribe params are required"
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     params,
			"error":             "ErrorRouting: " + errMsg,
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusBadRequest,
		})
		return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}
	serverID, originalURI, ok := p.clientSession.ParseName(params.URI)
	if !ok {
		errMsg := "Invalid resource URI: " + params.URI
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     params,
			"error":             "ErrorRouting: " + errMsg,
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusNotFound,
		})
		return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: errMsg}
	}
	err := ServerManagerInstance().UnsubscribeResource(ctx, serverID, originalURI, p.sessionID, p.userID)
	if err != nil {
		errMsg := err.Error()
		p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
			"serverId":          serverID,
			"upstreamRequestId": upstreamRequestIDStr,
			"uniformRequestId":  uniformRequestID,
			"requestParams":     params,
			"error":             "ErrorUnsubscribe: " + errMsg,
			"duration":          durationMillis(startedAt),
			"statusCode":        http.StatusInternalServerError,
		})
		return fmt.Errorf("%s", sanitizeMCPError(err))
	}
	p.logClientAudit(ctx, types.MCPEventLogTypeRequestResource, map[string]any{
		"serverId":          serverID,
		"upstreamRequestId": upstreamRequestIDStr,
		"uniformRequestId":  uniformRequestID,
		"requestParams":     params,
		"responseResult":    map[string]any{"unsubscribed": true},
		"duration":          durationMillis(startedAt),
		"statusCode":        http.StatusOK,
	})
	return nil
}

func (p *ProxySession) ForwardSamplingToClient(ctx context.Context, request *mcp.CreateMessageRequest, proxyRequestID string) (*mcp.CreateMessageResult, error) {
	timeoutMS := config.GetReverseRequestTimeout("sampling")
	if timeoutMS <= 0 {
		timeoutMS = 60000
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	p.trackReverseCancel(proxyRequestID, cancel)
	defer cancel()
	defer p.untrackReverseCancel(proxyRequestID)
	originalID, ok := p.requestIDMapper.GetOriginalRequestID(proxyRequestID)
	if ok && request != nil && request.Params != nil {
		meta := request.Params.GetMeta()
		if meta == nil {
			meta = map[string]any{}
		}
		meta["relatedRequestId"] = originalID
		request.Params.SetMeta(meta)
	}
	session := p.firstUpstreamSession()
	if session == nil {
		return nil, errors.New("upstream server not initialized")
	}
	if request == nil {
		return nil, errors.New("sampling request is nil")
	}
	result, err := session.CreateMessage(ctxWithTimeout, request.Params)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("sampling request timed out")
		}
		return nil, err
	}
	return result, nil
}

func (p *ProxySession) ForwardRootsListToClient(ctx context.Context, request *mcp.ListRootsRequest, proxyRequestID string) (*mcp.ListRootsResult, error) {
	timeoutMS := config.GetReverseRequestTimeout("roots")
	if timeoutMS <= 0 {
		timeoutMS = 10000
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	p.trackReverseCancel(proxyRequestID, cancel)
	defer cancel()
	defer p.untrackReverseCancel(proxyRequestID)
	if originalID, ok := p.requestIDMapper.GetOriginalRequestID(proxyRequestID); ok && request != nil && request.Params != nil {
		meta := request.Params.GetMeta()
		if meta == nil {
			meta = map[string]any{}
		}
		meta["relatedRequestId"] = originalID
		request.Params.SetMeta(meta)
	}
	session := p.firstUpstreamSession()
	if session == nil {
		return nil, errors.New("upstream server not initialized")
	}
	if request == nil {
		return nil, errors.New("roots list request is nil")
	}
	result, err := session.ListRoots(ctxWithTimeout, request.Params)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("roots request timed out")
		}
		return nil, err
	}
	return result, nil
}

func (p *ProxySession) ForwardElicitationToClient(ctx context.Context, request *mcp.ElicitRequest, proxyRequestID string) (*mcp.ElicitResult, error) {
	timeoutMS := config.GetReverseRequestTimeout("elicitation")
	if timeoutMS <= 0 {
		timeoutMS = 300000
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	p.trackReverseCancel(proxyRequestID, cancel)
	defer cancel()
	defer p.untrackReverseCancel(proxyRequestID)
	if originalID, ok := p.requestIDMapper.GetOriginalRequestID(proxyRequestID); ok && request != nil && request.Params != nil {
		meta := request.Params.GetMeta()
		if meta == nil {
			meta = map[string]any{}
		}
		meta["relatedRequestId"] = originalID
		request.Params.SetMeta(meta)
	}
	session := p.firstUpstreamSession()
	if session == nil {
		return nil, errors.New("upstream server not initialized")
	}
	if request == nil {
		return nil, errors.New("elicitation request is nil")
	}
	result, err := session.Elicit(ctxWithTimeout, request.Params)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("elicitation request timed out")
		}
		return nil, err
	}
	return result, nil
}

func (p *ProxySession) SendToolsListChangedToClient() {
	if p.upstreamServer == nil {
		return
	}
	if p.clientSession != nil && !p.clientSession.CanSendToolListChanged() {
		return
	}
	emitToolsListChanged(p.upstreamServer)
}

func (p *ProxySession) normalizeCallToolParams(params *mcp.CallToolParamsRaw) (*mcp.CallToolParams, error) {
	if params == nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "tool call params are required"}
	}
	var arguments any
	if len(params.Arguments) > 0 {
		if err := decodeJSONUseNumber(params.Arguments, &arguments); err != nil {
			return nil, err
		}
	}
	return &mcp.CallToolParams{
		Meta:      params.Meta,
		Name:      params.Name,
		Arguments: arguments,
	}, nil
}

func (p *ProxySession) upstreamRequestID(req mcp.Request, fallback any) any {
	if req != nil {
		params := req.GetParams()
		if params != nil {
			meta := params.GetMeta()
			if meta != nil {
				if v, ok := meta[upstreamRequestIDMetaKey]; ok && v != nil {
					return v
				}
			}
		}
	}
	seq := atomic.AddUint64(&p.upstreamFallbackCounter, 1)
	entropy := make([]byte, 8)
	if _, err := crand.Read(entropy); err != nil {
		return fmt.Sprintf("%v:%s:%d:%d", fallback, p.sessionID, time.Now().UnixNano(), seq)
	}
	return fmt.Sprintf("%v:%s:%d:%x", fallback, p.sessionID, seq, entropy)
}

func (p *ProxySession) withProxyContext(meta map[string]any, proxyRequestID string, uniformRequestID string) map[string]any {
	out := map[string]any{}
	for k, v := range meta {
		out[k] = v
	}
	if strings.TrimSpace(uniformRequestID) == "" {
		uniformRequestID = p.generateUniformRequestID()
	}
	out["proxyContext"] = map[string]any{
		"proxyRequestId":   proxyRequestID,
		"uniformRequestId": uniformRequestID,
	}
	return out
}

func (p *ProxySession) generateUniformRequestID() string {
	return internallog.GetLogService().GenerateUniformRequestID(p.sessionID)
}

func paramsFromRequest(req mcp.Request) any {
	if req == nil {
		return nil
	}
	return req.GetParams()
}

func (p *ProxySession) logClientAudit(ctx context.Context, action int, entry map[string]any) {
	if p.sessionLogger == nil {
		return
	}
	if entry == nil {
		entry = map[string]any{}
	}
	entry["action"] = action
	if err := p.sessionLogger.LogClientRequest(ctx, entry); err != nil {
		log.Warn().Err(err).Str("sessionId", p.sessionID).Int("action", action).Msg("failed to emit client audit log")
	}
}

func durationMillis(startedAt time.Time) int {
	return int(time.Since(startedAt).Milliseconds())
}

func formatCategorizedError(category string, message string) string {
	return fmt.Sprintf("Error: %s: %s", category, message)
}

func toolArgsFromAny(raw any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	if direct, ok := raw.(map[string]any); ok {
		return direct
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := decodeJSONUseNumber(payload, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func stringOrDefault(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}

func (p *ProxySession) waitForApproval(ctx context.Context, approvalRequestID string) (string, *string) {
	deadline := time.Now().Add(approvalWaitTimeoutMs * time.Millisecond)

	status, reason, pollErr := p.pollApprovalStatus(approvalRequestID)
	if pollErr != nil {
		errMsg := "Approval status check failed"
		log.Warn().Err(pollErr).Str("sessionId", p.sessionID).Str("approvalRequestId", approvalRequestID).Msg("approval status lookup failed")
		return "failed", &errMsg
	}
	if status != "" {
		return status, reason
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "aborted", nil
		case <-time.After(min(time.Duration(approvalPollIntervalMs)*time.Millisecond, time.Until(deadline))):
		}

		if ctx.Err() != nil {
			return "aborted", nil
		}

		status, reason, pollErr = p.pollApprovalStatus(approvalRequestID)
		if pollErr != nil {
			errMsg := "Approval status check failed"
			log.Warn().Err(pollErr).Str("sessionId", p.sessionID).Str("approvalRequestId", approvalRequestID).Msg("approval status lookup failed")
			return "failed", &errMsg
		}
		if status != "" {
			return status, reason
		}
	}

	return "timeout", nil
}

func (p *ProxySession) pollApprovalStatus(approvalRequestID string) (string, *string, error) {
	request, err := mcpservices.ApprovalServiceInstance().GetByID(approvalRequestID)
	if err != nil {
		return "", nil, err
	}
	if request == nil {
		return "", nil, nil
	}
	if (request.Status == types.ApprovalStatusPending || request.Status == types.ApprovalStatusApproved) && time.Now().After(request.ExpiresAt) {
		return "expired", nil, nil
	}

	switch request.Status {
	case types.ApprovalStatusApproved:
		return "approved", nil, nil
	case types.ApprovalStatusRejected:
		return "rejected", request.DecisionReason, nil
	case types.ApprovalStatusExpired:
		return "expired", nil, nil
	case types.ApprovalStatusExecuted:
		return "executed", nil, nil
	case types.ApprovalStatusFailed:
		return "failed", request.ExecutionError, nil
	case types.ApprovalStatusPending:
		return "", nil, nil
	case types.ApprovalStatusExecuting:
		return "executing", nil, nil
	default:
		return "", nil, nil
	}
}

func (p *ProxySession) emitApprovalWaitKeepalive(upstreamRequestID any, approvalRequestID string, toolName string, waited time.Duration) {
	session := p.firstUpstreamSession()
	if session == nil || upstreamRequestID == nil {
		return
	}
	message := fmt.Sprintf("Waiting for approval for tool %s (request %s)", toolName, approvalRequestID)
	_ = session.NotifyProgress(context.Background(), &mcp.ProgressNotificationParams{
		ProgressToken: upstreamRequestID,
		Progress:      float64(waited.Milliseconds()),
		Message:       message,
	})
}

func (p *ProxySession) buildApprovalOutcomeResult(approvalRequestID string, toolName string, reason string, policyVersion int, requestHash string, status string, summary string) *mcp.CallToolResult {
	text := fmt.Sprintf("TOOL NOT EXECUTED: %s\n\nApproval Request ID: %s\nResume Token: %s\nTool: %s\nReason: %s", summary, approvalRequestID, approvalRequestID, toolName, reason)
	return &mcp.CallToolResult{
		IsError: false,
		Meta: mcp.Meta{
			"approval": map[string]any{
				"requestId":     approvalRequestID,
				"resumeToken":   approvalRequestID,
				"toolName":      toolName,
				"reason":        reason,
				"policyVersion": policyVersion,
				"requestHash":   requestHash,
				"status":        status,
			},
		},
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func (p *ProxySession) tryBuildReplayToolResult(approvalRequestID string) *mcp.CallToolResult {
	resultRequest, err := mcpservices.ApprovalServiceInstance().GetResultByID(approvalRequestID)
	if err != nil || resultRequest == nil {
		return nil
	}

	replayPayload := p.extractReplayCallToolResult(resultRequest.ExecutionResult)
	if resultRequest.Status == types.ApprovalStatusExecuted && replayPayload != nil {
		return replayPayload
	}

	if resultRequest.Status == types.ApprovalStatusFailed {
		if replayPayload != nil && replayPayload.IsError {
			return replayPayload
		}
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: stringOrDefault(resultRequest.ExecutionError, "Execution failed in another request.")}},
		}
	}

	return nil
}

func (p *ProxySession) extractReplayCallToolResult(executionResult datatypes.JSON) *mcp.CallToolResult {
	if len(executionResult) == 0 {
		return nil
	}
	var result mcp.CallToolResult
	if err := json.Unmarshal(executionResult, &result); err != nil {
		return nil
	}
	if len(result.Content) == 0 {
		return nil
	}
	return &result
}

func (p *ProxySession) buildExecutionResultPreview(result *mcp.CallToolResult) *string {
	if result == nil {
		return nil
	}
	for _, content := range result.Content {
		textContent, ok := content.(*mcp.TextContent)
		if !ok || strings.TrimSpace(textContent.Text) == "" {
			continue
		}
		preview := textContent.Text
		if len(preview) > 180 {
			preview = preview[:180]
		}
		return &preview
	}
	return nil
}

func (p *ProxySession) injectUpstreamRequestIDMeta(r *http.Request) error {
	if r == nil || r.Body == nil {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return err
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(payload))

	payloadWithMeta, changed, err := injectJSONRPCRequestIDsToMeta(payload)
	if err != nil || !changed {
		return nil
	}
	r.Body = io.NopCloser(bytes.NewReader(payloadWithMeta))
	r.ContentLength = int64(len(payloadWithMeta))
	return nil
}

func injectJSONRPCRequestIDsToMeta(payload []byte) ([]byte, bool, error) {
	if len(payload) == 0 {
		return payload, false, nil
	}

	var root any
	if err := decodeJSONUseNumber(payload, &root); err != nil {
		return nil, false, err
	}

	changed := false
	switch messages := root.(type) {
	case map[string]any:
		changed = injectRequestIDToMeta(messages)
	case []any:
		for _, msg := range messages {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			if injectRequestIDToMeta(m) {
				changed = true
			}
		}
	default:
		return payload, false, nil
	}

	if !changed {
		return payload, false, nil
	}

	updated, err := json.Marshal(root)
	if err != nil {
		return nil, false, err
	}
	return updated, true, nil
}

func decodeJSONUseNumber(payload []byte, target any) error {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	if err := dec.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("invalid JSON payload")
		}
		return err
	}
	return nil
}

func injectRequestIDToMeta(message map[string]any) bool {
	if message == nil {
		return false
	}
	requestID, hasID := message["id"]
	if !hasID || requestID == nil {
		return false
	}

	params, ok := message["params"].(map[string]any)
	if !ok || params == nil {
		params = map[string]any{}
		message["params"] = params
	}
	meta, ok := params["_meta"].(map[string]any)
	if !ok || meta == nil {
		meta = map[string]any{}
		params["_meta"] = meta
	}
	meta[upstreamRequestIDMetaKey] = requestID
	return true
}

func (p *ProxySession) firstUpstreamSession() *mcp.ServerSession {
	if p.upstreamServer == nil {
		return nil
	}
	for session := range p.upstreamServer.Sessions() {
		if session != nil {
			return session
		}
	}
	return nil
}

func (p *ProxySession) SendResourcesListChangedToClient() {
	if p.upstreamServer == nil {
		return
	}
	if p.clientSession != nil && !p.clientSession.CanSendResourceListChanged() {
		return
	}
	emitResourcesListChanged(p.upstreamServer)
}

func (p *ProxySession) SendPromptsListChangedToClient() {
	if p.upstreamServer == nil {
		return
	}
	if p.clientSession != nil && !p.clientSession.CanSendPromptListChanged() {
		return
	}
	emitPromptsListChanged(p.upstreamServer)
}

func (p *ProxySession) SendResourceUpdatedToClient(serverID string, notification *mcp.ResourceUpdatedNotificationRequest) {
	if p.upstreamServer == nil {
		return
	}
	if notification == nil || notification.Params == nil {
		return
	}
	ctx := ServerManagerInstance().GetServerContext(serverID, p.userID)
	if ctx == nil {
		return
	}
	uri := p.clientSession.GenerateNewName(ctx.ID, notification.Params.URI)
	_ = p.upstreamServer.ResourceUpdated(context.Background(), &mcp.ResourceUpdatedNotificationParams{URI: uri})
}

func (p *ProxySession) ForwardProgressToClient(params *mcp.ProgressNotificationParams) {
	if params == nil {
		return
	}
	session := p.firstUpstreamSession()
	if session == nil {
		return
	}
	forward := *params
	if token := fmt.Sprintf("%v", params.ProgressToken); token != "" {
		original, ok := p.requestIDMapper.GetOriginalRequestID(token)
		if !ok {
			log.Warn().Str("sessionId", p.sessionID).Str("proxyRequestId", token).Msg("dropping progress notification due to missing request mapping")
			return
		}
		forward.ProgressToken = original
	}
	_ = session.NotifyProgress(context.Background(), &forward)
}

func (p *ProxySession) HandleDownstreamCancellation(proxyRequestID string) {
	if strings.TrimSpace(proxyRequestID) == "" {
		return
	}
	originalRequestID, hasOriginal := p.requestIDMapper.GetOriginalRequestID(proxyRequestID)
	p.downstreamMu.Lock()
	downstreamCancel := p.downstreamCancels[proxyRequestID]
	delete(p.downstreamCancels, proxyRequestID)
	p.downstreamMu.Unlock()
	if downstreamCancel != nil {
		downstreamCancel()
	}

	p.cancelReverseRequest(proxyRequestID)
	if hasOriginal {
		if err := p.notifyUpstreamCancellation(originalRequestID); err != nil {
			log.Warn().Err(err).Str("sessionId", p.sessionID).Interface("originalRequestId", originalRequestID).Msg("failed to forward cancellation to upstream client")
		}
	}
	p.requestIDMapper.RemoveMapping(proxyRequestID)
}

func (p *ProxySession) cancelDownstreamByOriginalRequestID(originalRequestID any) {
	proxyRequestID, ok := p.requestIDMapper.GetProxyRequestID(originalRequestID)
	if !ok {
		proxyRequestID, ok = p.requestIDMapper.GetProxyRequestID(fmt.Sprintf("%v", originalRequestID))
	}
	if !ok || strings.TrimSpace(proxyRequestID) == "" {
		return
	}
	mappingEntry, hasMapping := p.requestIDMapper.GetMappingEntry(proxyRequestID)
	p.downstreamMu.Lock()
	cancel := p.downstreamCancels[proxyRequestID]
	delete(p.downstreamCancels, proxyRequestID)
	p.downstreamMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if hasMapping && strings.TrimSpace(mappingEntry.ServerID) != "" && mappingEntry.DownstreamRequestID != nil {
		if err := p.notifyDownstreamCancellation(mappingEntry.ServerID, mappingEntry.DownstreamRequestID); err != nil {
			log.Warn().Err(err).Str("sessionId", p.sessionID).Str("serverId", mappingEntry.ServerID).Interface("downstreamRequestId", mappingEntry.DownstreamRequestID).Msg("failed to forward cancellation to downstream server")
		}
	}
	p.requestIDMapper.RemoveMapping(proxyRequestID)
}

func (p *ProxySession) notifyDownstreamCancellation(serverID string, downstreamRequestID any) error {
	if strings.TrimSpace(serverID) == "" || downstreamRequestID == nil {
		return nil
	}
	serverCtx := ServerManagerInstance().GetServerContext(serverID, p.clientSession.UserID)
	if serverCtx == nil {
		return errors.New("server context unavailable")
	}
	conn := serverCtx.ConnectionSnapshot()
	if conn == nil {
		return errors.New("downstream connection unavailable")
	}
	notifier, ok := any(conn).(interface {
		NotifyCancelled(context.Context, *mcp.CancelledParams) error
	})
	if !ok {
		return errors.New("downstream connection does not support cancelled notifications")
	}
	params := &mcp.CancelledParams{RequestID: downstreamRequestID}
	return notifier.NotifyCancelled(context.Background(), params)
}

func (p *ProxySession) notifyUpstreamCancellation(originalRequestID any) error {
	if originalRequestID == nil {
		return nil
	}
	session := p.firstUpstreamSession()
	if session == nil {
		return errors.New("upstream session unavailable")
	}
	notifier, ok := any(session).(interface {
		NotifyCancelled(context.Context, *mcp.CancelledParams) error
	})
	if !ok {
		return errors.New("upstream session does not support cancelled notifications")
	}
	params := &mcp.CancelledParams{RequestID: originalRequestID}
	return notifier.NotifyCancelled(context.Background(), params)
}

func (p *ProxySession) ResolveProxyRequestID(originalRequestID any) (string, bool) {
	proxyRequestID, ok := p.requestIDMapper.GetProxyRequestID(originalRequestID)
	if ok {
		return proxyRequestID, true
	}
	return p.requestIDMapper.GetProxyRequestID(fmt.Sprintf("%v", originalRequestID))
}

func (p *ProxySession) ResolveProxyRequestIDForServer(originalRequestID any, serverID string) (string, bool) {
	proxyRequestID, ok := p.ResolveProxyRequestID(originalRequestID)
	if !ok || proxyRequestID == "" {
		return "", false
	}
	if strings.TrimSpace(serverID) == "" {
		return proxyRequestID, true
	}
	if p.IsProxyRequestBoundToServer(proxyRequestID, serverID) {
		return proxyRequestID, true
	}
	return "", false
}

func (p *ProxySession) RegisterDownstreamRequestMapping(proxyRequestID string, downstreamRequestID any, serverID string) {
	if strings.TrimSpace(proxyRequestID) == "" || downstreamRequestID == nil || strings.TrimSpace(serverID) == "" {
		return
	}
	p.requestIDMapper.RegisterDownstreamMapping(proxyRequestID, downstreamRequestID, serverID)
}

func (p *ProxySession) IsProxyRequestBoundToServer(proxyRequestID string, serverID string) bool {
	if strings.TrimSpace(proxyRequestID) == "" || strings.TrimSpace(serverID) == "" {
		return false
	}
	entry, ok := p.requestIDMapper.GetMappingEntry(proxyRequestID)
	if !ok || entry == nil {
		return false
	}
	return strings.TrimSpace(entry.ServerID) == strings.TrimSpace(serverID)
}

func (p *ProxySession) trackDownstreamCancel(proxyRequestID string, cancel context.CancelFunc) {
	if strings.TrimSpace(proxyRequestID) == "" || cancel == nil {
		return
	}
	p.downstreamMu.Lock()
	defer p.downstreamMu.Unlock()
	p.downstreamCancels[proxyRequestID] = cancel
}

func (p *ProxySession) untrackDownstreamCancel(proxyRequestID string) {
	if strings.TrimSpace(proxyRequestID) == "" {
		return
	}
	p.downstreamMu.Lock()
	defer p.downstreamMu.Unlock()
	delete(p.downstreamCancels, proxyRequestID)
}

func (p *ProxySession) trackReverseCancel(proxyRequestID string, cancel context.CancelFunc) {
	if strings.TrimSpace(proxyRequestID) == "" || cancel == nil {
		return
	}
	p.reverseMu.Lock()
	defer p.reverseMu.Unlock()
	p.reverseCancels[proxyRequestID] = cancel
}

func (p *ProxySession) untrackReverseCancel(proxyRequestID string) {
	if strings.TrimSpace(proxyRequestID) == "" {
		return
	}
	p.reverseMu.Lock()
	defer p.reverseMu.Unlock()
	delete(p.reverseCancels, proxyRequestID)
}

func (p *ProxySession) cancelReverseRequest(proxyRequestID string) {
	if strings.TrimSpace(proxyRequestID) == "" {
		return
	}
	p.reverseMu.Lock()
	cancel := p.reverseCancels[proxyRequestID]
	delete(p.reverseCancels, proxyRequestID)
	p.reverseMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (p *ProxySession) Cleanup(ctx context.Context) {
	p.closeOnce.Do(func() {
		p.downstreamMu.Lock()
		for key, cancel := range p.downstreamCancels {
			if cancel != nil {
				cancel()
			}
			delete(p.downstreamCancels, key)
		}
		p.downstreamMu.Unlock()

		p.reverseMu.Lock()
		for key, cancel := range p.reverseCancels {
			if cancel != nil {
				cancel()
			}
			delete(p.reverseCancels, key)
		}
		p.reverseMu.Unlock()

		p.requestIDMapper.Destroy()
		ServerManagerInstance().CleanupSessionSubscriptions(ctx, p.sessionID, p.userID)
		if p.upstreamServer != nil {
			if closer, ok := any(p.upstreamServer).(interface{ Close() error }); ok {
				_ = closer.Close()
			}
		}
	})
}

type ssePersistingResponseWriter struct {
	http.ResponseWriter
	ctx             context.Context
	eventStore      *PersistentEventStore
	defaultStreamID mcptypes.StreamID
	pending         []byte
}

func newSSEPersistingResponseWriter(w http.ResponseWriter, ctx context.Context, eventStore *PersistentEventStore, defaultStreamID mcptypes.StreamID) *ssePersistingResponseWriter {
	return &ssePersistingResponseWriter{ResponseWriter: w, ctx: ctx, eventStore: eventStore, defaultStreamID: defaultStreamID}
}

func (w *ssePersistingResponseWriter) Write(p []byte) (int, error) {
	if w.eventStore == nil {
		return w.ResponseWriter.Write(p)
	}

	w.pending = append(w.pending, p...)
	total := len(p)
	for {
		idx := bytes.Index(w.pending, []byte("\n\n"))
		if idx < 0 {
			break
		}
		chunk := w.pending[:idx]
		w.pending = w.pending[idx+2:]
		out := w.rewriteAndPersistChunk(chunk)
		if _, err := w.ResponseWriter.Write(out); err != nil {
			return 0, err
		}
	}

	return total, nil
}

func (w *ssePersistingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *ssePersistingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http hijacker is unavailable")
	}
	return hj.Hijack()
}

func (w *ssePersistingResponseWriter) rewriteAndPersistChunk(chunk []byte) []byte {
	if len(chunk) == 0 {
		return []byte("\n\n")
	}

	lines := strings.Split(string(chunk), "\n")
	eventName := ""
	eventID := ""
	dataLines := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		switch {
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "id:"):
			eventID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}

	if eventName != "message" || len(dataLines) == 0 {
		return append(chunk, []byte("\n\n")...)
	}

	payload := strings.Join(dataLines, "\n")
	message := mcptypes.JSONRPCMessage{}
	if err := json.Unmarshal([]byte(payload), &message); err != nil {
		return append(chunk, []byte("\n\n")...)
	}

	streamID := w.defaultStreamID
	if eventID != "" {
		if parsed := extractStreamID(eventID); parsed != "" {
			streamID = parsed
		}
	}

	storedID, err := w.eventStore.StoreEvent(w.ctx, streamID, message)
	if err != nil {
		log.Warn().Err(err).Str("sessionId", w.eventStore.sessionID).Str("streamId", streamID).Msg("failed to persist outbound SSE event")
		return append(chunk, []byte("\n\n")...)
	}

	return []byte(fmt.Sprintf("event: message\nid: %s\ndata: %s\n\n", storedID, mustJSON(message)))
}

func sanitizeMCPError(err error) string {
	if err == nil {
		return "unknown error"
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	for _, keyword := range []string{"dial tcp", "connection refused", "no such host", "tls:", "x509:", "i/o timeout", "broken pipe", "sql:", "database", "gorm"} {
		if strings.Contains(lower, keyword) {
			return "internal server error"
		}
	}
	return msg
}

package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	internallog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

type notificationTracker struct {
	order []string
	set   map[string]struct{}
}

type GlobalRequestRouter struct {
	mu                sync.RWMutex
	sentNotifications map[string]*notificationTracker
}

var (
	globalRequestRouter     *GlobalRequestRouter
	globalRequestRouterOnce sync.Once
)

func GlobalRequestRouterInstance() *GlobalRequestRouter {
	globalRequestRouterOnce.Do(func() {
		globalRequestRouter = &GlobalRequestRouter{sentNotifications: map[string]*notificationTracker{}}
	})
	return globalRequestRouter
}

type reverseHandlerConfig struct {
	requestAction  int
	responseAction int
	capCheck       func(*ProxySession) bool
	capDeniedMsg   string
	capDeniedCode  int64
	forward        func(context.Context, *ProxySession) (any, error)
}

func (r *GlobalRequestRouter) handleReverse(ctx context.Context, serverID, proxyContext string, requestParams any, meta map[string]any, cfg reverseHandlerConfig) (any, error) {
	if strings.TrimSpace(proxyContext) == "" {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "missing proxyContext.proxyRequestId"}
	}
	sessionID := sessionIDFromProxyRequestID(proxyContext)
	proxySession := SessionStoreInstance().GetProxySession(sessionID)
	if proxySession == nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: fmt.Sprintf("No ProxySession found for sessionId: %s", sessionID)}
	}
	if !proxySession.IsProxyRequestBoundToServer(proxyContext, serverID) {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "invalid reverse request context"}
	}
	if !proxySession.clientSession.CanAccessServer(serverID) {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidRequest, Message: "unauthorized server for reverse request"}
	}
	sessionLogger := SessionStoreInstance().GetSessionLogger(sessionID)
	uniformRequestID := internallog.GetLogService().GenerateUniformRequestID(sessionID)
	parentUniformRequestID := ""
	if meta != nil {
		if proxyContextRaw, ok := meta["proxyContext"]; ok && proxyContextRaw != nil {
			if proxyContext, ok := proxyContextRaw.(map[string]any); ok {
				if parentID, ok := proxyContext["uniformRequestId"]; ok && parentID != nil {
					parentUniformRequestID = fmt.Sprintf("%v", parentID)
				}
			}
		}
	}
	startedAt := time.Now()

	makeEntry := func(action int, extra map[string]any) map[string]any {
		entry := map[string]any{
			"action":                 action,
			"serverId":               serverID,
			"upstreamRequestId":      "",
			"uniformRequestId":       uniformRequestID,
			"parentUniformRequestId": parentUniformRequestID,
			"proxyRequestId":         proxyContext,
			"requestParams":          requestParams,
		}
		for k, v := range extra {
			entry[k] = v
		}
		return entry
	}

	if !cfg.capCheck(proxySession) {
		logReverseAudit(ctx, sessionID, sessionLogger, makeEntry(cfg.requestAction, map[string]any{
			"error":      cfg.capDeniedMsg,
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusForbidden,
		}))
		return nil, &jsonrpc.Error{Code: cfg.capDeniedCode, Message: cfg.capDeniedMsg}
	}

	logReverseAudit(ctx, sessionID, sessionLogger, makeEntry(cfg.requestAction, nil))

	result, err := cfg.forward(ctx, proxySession)
	if err != nil {
		logReverseAudit(ctx, sessionID, sessionLogger, makeEntry(cfg.responseAction, map[string]any{
			"error":      err.Error(),
			"duration":   durationMillis(startedAt),
			"statusCode": http.StatusInternalServerError,
		}))
		return nil, err
	}

	logReverseAudit(ctx, sessionID, sessionLogger, makeEntry(cfg.responseAction, map[string]any{
		"responseResult": result,
		"duration":       durationMillis(startedAt),
		"statusCode":     http.StatusOK,
	}))
	return result, nil
}

func (r *GlobalRequestRouter) HandleSamplingRequest(ctx context.Context, serverID string, request *mcp.CreateMessageRequest, proxyContext string) (*mcp.CreateMessageResult, error) {
	var params any
	var meta map[string]any
	if request != nil {
		params = request.Params
		if request.Params != nil {
			meta = request.Params.GetMeta()
		}
	}
	result, err := r.handleReverse(ctx, serverID, proxyContext, params, meta, reverseHandlerConfig{
		requestAction:  types.MCPEventLogTypeReverseSamplingRequest,
		responseAction: types.MCPEventLogTypeReverseSamplingResponse,
		capCheck:       func(ps *ProxySession) bool { return ps.clientSession.CanRequestSampling() },
		capDeniedMsg:   "Client is not allowed to request sampling",
		capDeniedCode:  jsonrpc.CodeInvalidParams,
		forward: func(ctx context.Context, ps *ProxySession) (any, error) {
			return ps.ForwardSamplingToClient(ctx, request, proxyContext)
		},
	})
	if err != nil {
		return nil, err
	}
	res, ok := result.(*mcp.CreateMessageResult)
	if !ok {
		return nil, errors.New("unexpected result type from sampling forward")
	}
	return res, nil
}

func (r *GlobalRequestRouter) HandleRootsListRequest(ctx context.Context, serverID string, request *mcp.ListRootsRequest, proxyContext string) (*mcp.ListRootsResult, error) {
	var params any
	var meta map[string]any
	if request != nil {
		params = request.Params
		if request.Params != nil {
			meta = request.Params.GetMeta()
		}
	}
	result, err := r.handleReverse(ctx, serverID, proxyContext, params, meta, reverseHandlerConfig{
		requestAction:  types.MCPEventLogTypeReverseRootsRequest,
		responseAction: types.MCPEventLogTypeReverseRootsResponse,
		capCheck:       func(ps *ProxySession) bool { return ps.clientSession.CanRequestRoots() },
		capDeniedMsg:   "Client does not support roots capability",
		capDeniedCode:  jsonrpc.CodeMethodNotFound,
		forward: func(ctx context.Context, ps *ProxySession) (any, error) {
			return ps.ForwardRootsListToClient(ctx, request, proxyContext)
		},
	})
	if err != nil {
		return nil, err
	}
	res, ok := result.(*mcp.ListRootsResult)
	if !ok {
		return nil, errors.New("unexpected result type from roots forward")
	}
	return res, nil
}

func (r *GlobalRequestRouter) HandleElicitationRequest(ctx context.Context, serverID string, request *mcp.ElicitRequest, proxyContext string) (*mcp.ElicitResult, error) {
	var params any
	var meta map[string]any
	if request != nil {
		params = request.Params
		if request.Params != nil {
			meta = request.Params.GetMeta()
		}
	}
	result, err := r.handleReverse(ctx, serverID, proxyContext, params, meta, reverseHandlerConfig{
		requestAction:  types.MCPEventLogTypeReverseElicitRequest,
		responseAction: types.MCPEventLogTypeReverseElicitResponse,
		capCheck:       func(ps *ProxySession) bool { return ps.clientSession.CanRequestElicitation() },
		capDeniedMsg:   "Client is not allowed to request user input",
		capDeniedCode:  jsonrpc.CodeInvalidParams,
		forward: func(ctx context.Context, ps *ProxySession) (any, error) {
			return ps.ForwardElicitationToClient(ctx, request, proxyContext)
		},
	})
	if err != nil {
		return nil, err
	}
	res, ok := result.(*mcp.ElicitResult)
	if !ok {
		return nil, errors.New("unexpected result type from elicitation forward")
	}
	return res, nil
}

func (r *GlobalRequestRouter) HandleToolsListChanged(serverID string) {
	r.broadcastServerChange(serverID, "tools")
}

func (r *GlobalRequestRouter) HandleResourcesListChanged(serverID string) {
	r.broadcastServerChange(serverID, "resources")
}

func (r *GlobalRequestRouter) HandlePromptsListChanged(serverID string) {
	r.broadcastServerChange(serverID, "prompts")
}

func (r *GlobalRequestRouter) HandleResourceUpdated(serverID string, notification *mcp.ResourceUpdatedNotificationRequest) {
	if notification == nil || notification.Params == nil {
		return
	}
	subscriptionKey := fmt.Sprintf("%s::%s", serverID, notification.Params.URI)
	subscribers := ServerManagerInstance().GetResourceSubscribers(subscriptionKey)
	for _, session := range SessionStoreInstance().GetAllSessions() {
		if _, ok := subscribers[session.SessionID]; !ok {
			continue
		}
		if !session.CanAccessServer(serverID) {
			continue
		}
		notifKey := fmt.Sprintf("resource:%s:%s:%d", serverID, notification.Params.URI, time.Now().UnixNano())
		if !r.markNotificationSent(session.SessionID, notifKey) {
			continue
		}
		proxySession := session.GetProxySession()
		if proxySession != nil {
			proxySession.SendResourceUpdatedToClient(serverID, notification)
		}
	}
}

func (r *GlobalRequestRouter) HandleProgressNotification(proxyRequestID string, params *mcp.ProgressNotificationParams) {
	sessionID := sessionIDFromProxyRequestID(proxyRequestID)
	if sessionID == "" {
		return
	}
	proxySession := SessionStoreInstance().GetProxySession(sessionID)
	if proxySession == nil {
		return
	}
	proxySession.ForwardProgressToClient(params)
}

func (r *GlobalRequestRouter) RegisterDownstreamRequestMapping(serverID string, proxyRequestID string, downstreamRequestID any) {
	if strings.TrimSpace(serverID) == "" || strings.TrimSpace(proxyRequestID) == "" || downstreamRequestID == nil {
		return
	}
	sessionID := sessionIDFromProxyRequestID(proxyRequestID)
	if sessionID == "" {
		return
	}
	proxySession := SessionStoreInstance().GetProxySession(sessionID)
	if proxySession == nil {
		return
	}
	proxySession.RegisterDownstreamRequestMapping(proxyRequestID, downstreamRequestID, serverID)
}

func (r *GlobalRequestRouter) HandleCancelledNotification(requestID any, serverID ...string) bool {
	proxyRequestID := fmt.Sprintf("%v", requestID)
	if proxyRequestID != "" {
		sessionID := sessionIDFromProxyRequestID(proxyRequestID)
		if sessionID != "" {
			if proxySession := SessionStoreInstance().GetProxySession(sessionID); proxySession != nil {
				proxySession.HandleDownstreamCancellation(proxyRequestID)
				return true
			}
		}
	}

	sid := ""
	if len(serverID) > 0 {
		sid = strings.TrimSpace(serverID[0])
	}

	for _, proxySession := range SessionStoreInstance().GetAllProxySessions() {
		if proxySession == nil {
			continue
		}
		resolvedProxyRequestID, ok := proxySession.ResolveProxyRequestIDForServer(requestID, sid)
		if !ok || resolvedProxyRequestID == "" {
			continue
		}
		proxySession.HandleDownstreamCancellation(resolvedProxyRequestID)
		return true
	}

	return false
}

func (r *GlobalRequestRouter) CleanupSessionNotifications(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sentNotifications, sessionID)
}

func (r *GlobalRequestRouter) Destroy() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sentNotifications = map[string]*notificationTracker{}
}

func (r *GlobalRequestRouter) broadcastServerChange(serverID, kind string) {
	ServerManagerInstance().NotifyUserPermissionChangedByServer(serverID)

	for _, session := range SessionStoreInstance().GetAllSessions() {
		if !session.CanAccessServer(serverID) {
			continue
		}
		notifKey := fmt.Sprintf("%s:%s:%d", kind, serverID, time.Now().UnixNano())
		if !r.markNotificationSent(session.SessionID, notifKey) {
			continue
		}
		ps := session.GetProxySession()
		if ps == nil {
			continue
		}
		switch kind {
		case "tools":
			ps.SendToolsListChangedToClient()
		case "resources":
			ps.SendResourcesListChangedToClient()
		case "prompts":
			ps.SendPromptsListChangedToClient()
		}
	}
}

func (r *GlobalRequestRouter) markNotificationSent(sessionID, key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	tracker := r.sentNotifications[sessionID]
	if tracker == nil {
		tracker = &notificationTracker{set: map[string]struct{}{}}
		r.sentNotifications[sessionID] = tracker
	}
	if _, ok := tracker.set[key]; ok {
		return false
	}
	tracker.set[key] = struct{}{}
	tracker.order = append(tracker.order, key)
	if len(tracker.order) > 100 {
		evict := tracker.order[:len(tracker.order)-100]
		for _, k := range evict {
			delete(tracker.set, k)
		}
		tracker.order = tracker.order[len(tracker.order)-100:]
	}
	return true
}

func sessionIDFromProxyRequestID(proxyRequestID string) string {
	parts := strings.Split(proxyRequestID, ":")
	return parts[0]
}

func logReverseAudit(ctx context.Context, sessionID string, sessionLogger SessionLogger, entry map[string]any) {
	if sessionLogger == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := sessionLogger.LogReverseRequest(ctx, entry); err != nil {
		log.Warn().Err(err).Str("sessionId", sessionID).Msg("failed to emit reverse audit log")
	}
}

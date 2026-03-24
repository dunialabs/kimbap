package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/logger"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/dunialabs/kimbap-core/internal/user"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type SocketNotifier struct {
	mu                           sync.RWMutex
	socketService                *SocketService
	onlineSessionsNotifyInFlight map[string]*onlineSessionResult
	log                          zerolog.Logger
}

const defaultSocketRequestTimeout = 55 * time.Second

type onlineSessionResult struct {
	done chan struct{}
	ok   bool
}

var (
	globalSocketNotifier = &SocketNotifier{
		onlineSessionsNotifyInFlight: make(map[string]*onlineSessionResult),
		log:                          logger.CreateLogger("SocketNotifier"),
	}
)

func GetSocketNotifier() *SocketNotifier {
	return globalSocketNotifier
}

func (n *SocketNotifier) SetSocketService(service *SocketService) {
	n.mu.Lock()
	n.socketService = service
	n.mu.Unlock()
}

func (n *SocketNotifier) service() (*SocketService, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.socketService == nil {
		return nil, errors.New("socket notifier not initialized")
	}
	return n.socketService, nil
}

func (n *SocketNotifier) NotifyUser(userID string, event string, data any) bool {
	svc, err := n.service()
	if err != nil {
		return false
	}
	if !svc.IsUserOnline(userID) {
		return false
	}
	svc.EmitToUser(userID, event, data)
	return true
}

func (n *SocketNotifier) NotifyAll(event string, data any) {
	svc, err := n.service()
	if err != nil {
		return
	}
	svc.EmitToAll(event, data)
}

func (n *SocketNotifier) IsUserOnline(userID string) bool {
	svc, err := n.service()
	if err != nil {
		return false
	}
	return svc.IsUserOnline(userID)
}

func (n *SocketNotifier) GetUserDeviceCount(userID string) int {
	svc, err := n.service()
	if err != nil {
		return 0
	}
	return svc.GetUserDeviceCount(userID)
}

func (n *SocketNotifier) GetUserConnections(userID string) []UserConnection {
	svc, err := n.service()
	if err != nil {
		return []UserConnection{}
	}
	return svc.GetUserConnections(userID)
}

func (n *SocketNotifier) GetOnlineUserIDs() []string {
	svc, err := n.service()
	if err != nil {
		return []string{}
	}
	return svc.GetOnlineUserIDs()
}

func (n *SocketNotifier) GetTotalConnections() int {
	svc, err := n.service()
	if err != nil {
		return 0
	}
	return svc.GetTotalConnections()
}

func (n *SocketNotifier) UpdateServerInfo() {
	svc, err := n.service()
	if err != nil {
		return
	}
	svc.updateServerInfo()
}

func (n *SocketNotifier) SendNotification(userID string, notification NotificationData) bool {
	return n.NotifyUser(userID, SocketEventNotification, notification)
}

func (n *SocketNotifier) NotifyUserDisabled(userID string, reason string) bool {
	if strings.TrimSpace(reason) == "" {
		reason = "Your account has been disabled by administrator"
	}
	return n.SendNotification(userID, NotificationData{
		Type:      NotificationTypeUserDisabled,
		Message:   reason,
		Timestamp: time.Now().UnixMilli(),
		Severity:  "error",
	})
}

func (n *SocketNotifier) NotifyUserExpired(userID string) bool {
	return n.SendNotification(userID, NotificationData{
		Type:      NotificationTypeUserExpired,
		Message:   "Your authorization has expired",
		Timestamp: time.Now().UnixMilli(),
		Severity:  "error",
	})
}

func (n *SocketNotifier) SendRequest(userID string, action SocketActionType, data any, timeout time.Duration) SocketResponse[map[string]any] {
	_, responseCh := n.sendRequestAsync(userID, action, data, timeout)
	return <-responseCh
}

func (n *SocketNotifier) sendRequestAsync(userID string, action SocketActionType, data any, timeout time.Duration) (string, <-chan SocketResponse[map[string]any]) {
	responseCh := make(chan SocketResponse[map[string]any], 1)
	sendResponse := func(response SocketResponse[map[string]any]) {
		select {
		case responseCh <- response:
		default:
		}
	}

	svc, err := n.service()
	if err != nil {
		sendResponse(SocketResponse[map[string]any]{
			RequestID: "",
			Success:   false,
			Error: &SocketResponseError{
				Code:    SocketErrorServiceUnavailable,
				Message: sanitizeSocketError(err.Error()),
			},
			Timestamp: time.Now().UnixMilli(),
		})
		return "", responseCh
	}

	if !svc.IsUserOnline(userID) {
		sendResponse(SocketResponse[map[string]any]{
			RequestID: "",
			Success:   false,
			Error: &SocketResponseError{
				Code:    SocketErrorUserOffline,
				Message: fmt.Sprintf("User %s is offline", userID),
			},
			Timestamp: time.Now().UnixMilli(),
		})
		return "", responseCh
	}

	if timeout <= 0 {
		timeout = defaultSocketRequestTimeout
	}

	requestID := uuid.NewString()
	request := SocketRequest[any]{
		RequestID: requestID,
		Action:    action,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	timer := time.AfterFunc(timeout, func() {
		svc.RemovePendingRequest(requestID)
		sendResponse(SocketResponse[map[string]any]{
			RequestID: requestID,
			Success:   false,
			Error: &SocketResponseError{
				Code:    SocketErrorTimeout,
				Message: fmt.Sprintf("request timeout after %dms", timeout.Milliseconds()),
			},
			Timestamp: time.Now().UnixMilli(),
		})
	})

	svc.AddPendingRequest(requestID, pendingRequest{
		resolver:  sendResponse,
		timer:     timer,
		userID:    userID,
		action:    action,
		createdAt: time.Now(),
	})

	svc.EmitToUser(userID, ActionToEventName(action), request)
	return requestID, responseCh
}

func (n *SocketNotifier) GetClientStatus(userID string, timeout time.Duration) (any, error) {
	response := n.SendRequest(userID, SocketActionGetClientStatus, map[string]any{}, timeout)
	if !response.Success {
		if response.Error != nil {
			return nil, errors.New(response.Error.Message)
		}
		return nil, errors.New("request failed")
	}
	if response.Data == nil {
		return nil, nil
	}
	return *response.Data, nil
}

func (n *SocketNotifier) AskUserConfirm(ctx context.Context, userID string, userAgent string, ip string, toolName string, description string, params string, timeout time.Duration) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	requestID, responseCh := n.sendRequestAsync(userID, SocketActionAskUserConfirm, map[string]any{
		"userAgent":       userAgent,
		"ip":              ip,
		"toolName":        toolName,
		"toolDescription": description,
		"toolParams":      params,
	}, timeout)

	var response SocketResponse[map[string]any]
	select {
	case response = <-responseCh:
	case <-ctx.Done():
		if requestID != "" {
			if svc, err := n.service(); err == nil {
				svc.RemovePendingRequest(requestID)
			}
		}
		return false, nil
	}
	if !response.Success {
		return false, nil
	}
	if response.Data == nil {
		return false, nil
	}
	confirmed, _ := (*response.Data)["confirmed"].(bool)
	return confirmed, nil
}

func (n *SocketNotifier) NotifyApprovalCreated(userID string, approvalRequestID string, toolName string, serverID *string, redactedArgs any, expiresAt time.Time, createdAt time.Time, status string, uniformRequestID *string, policyVersion int, matchedRuleID *string, reason *string) {
	data := ApprovalCreatedData{
		ID:               approvalRequestID,
		ResumeToken:      approvalRequestID,
		ToolName:         toolName,
		ServerID:         serverID,
		RedactedArgs:     redactedArgs,
		ExpiresAt:        expiresAt,
		CreatedAt:        createdAt,
		Status:           status,
		UniformRequestID: uniformRequestID,
		PolicyVersion:    policyVersion,
		MatchedRuleID:    matchedRuleID,
		Reason:           reason,
	}
	n.NotifyUser(userID, SocketEventNotification, NotificationData{
		Type:      "approval_created",
		Message:   fmt.Sprintf("Approval required for tool: %s", data.ToolName),
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
		Severity:  "warning",
	})
}

func (n *SocketNotifier) NotifyApprovalDecided(userID string, approvalRequestID string, toolName string, decision string, reason *string) {
	data := ApprovalDecidedData{ID: approvalRequestID, ResumeToken: approvalRequestID, ToolName: toolName, Decision: decision, Reason: reason}
	verb := "rejected"
	severity := "warning"
	if strings.EqualFold(data.Decision, types.ApprovalStatusApproved) {
		verb = "approved"
		severity = "success"
	}
	n.NotifyUser(userID, SocketEventNotification, NotificationData{
		Type:      "approval_decided",
		Message:   fmt.Sprintf("Tool %s has been %s", data.ToolName, verb),
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
		Severity:  severity,
	})
}

func (n *SocketNotifier) NotifyApprovalExpired(userID string, approvalRequestID string, toolName string) {
	data := ApprovalExpiredData{ID: approvalRequestID, ResumeToken: approvalRequestID, ToolName: toolName}
	n.NotifyUser(userID, SocketEventNotification, NotificationData{
		Type:      "approval_expired",
		Message:   fmt.Sprintf("Approval for tool %s has expired", data.ToolName),
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
		Severity:  "info",
	})
}

func (n *SocketNotifier) NotifyApprovalExecuted(userID string, approvalRequestID string, toolName string, executionResultAvailable bool, executionResultPreview *string) {
	data := ApprovalExecutedData{ID: approvalRequestID, ResumeToken: approvalRequestID, ToolName: toolName, ExecutionResultAvailable: executionResultAvailable, ExecutionResultPreview: executionResultPreview}
	n.NotifyUser(userID, SocketEventNotification, NotificationData{
		Type:      "approval_executed",
		Message:   fmt.Sprintf("Tool %s executed successfully", data.ToolName),
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
		Severity:  "success",
	})
}

func (n *SocketNotifier) NotifyApprovalFailed(userID string, approvalRequestID string, toolName string, executionError string, executionResultAvailable bool, executionResultPreview *string) {
	data := ApprovalFailedData{ID: approvalRequestID, ResumeToken: approvalRequestID, ToolName: toolName, Error: executionError, ExecutionResultAvailable: executionResultAvailable, ExecutionResultPreview: executionResultPreview}
	n.NotifyUser(userID, SocketEventNotification, NotificationData{
		Type:      "approval_failed",
		Message:   fmt.Sprintf("Tool %s execution failed", data.ToolName),
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
		Severity:  "error",
	})
}

func (n *SocketNotifier) NotifyOnlineSessions(userID string) bool {
	_, err := n.service()
	if err != nil {
		return false
	}

	n.mu.Lock()
	if inFlight, ok := n.onlineSessionsNotifyInFlight[userID]; ok {
		n.mu.Unlock()
		<-inFlight.done
		return inFlight.ok
	}
	result := &onlineSessionResult{done: make(chan struct{})}
	n.onlineSessionsNotifyInFlight[userID] = result
	n.mu.Unlock()
	defer func() {
		n.mu.Lock()
		delete(n.onlineSessionsNotifyInFlight, userID)
		n.mu.Unlock()
		close(result.done)
	}()

	sessions, err := user.NewRequestHandler(nil).GetOnlineSessions(userID)
	if err != nil {
		result.ok = false
		return false
	}
	notification := NotificationData{
		Type:      NotificationTypeOnlineSessions,
		Message:   fmt.Sprintf("You have %d active session(s)", len(sessions)),
		Timestamp: time.Now().UnixMilli(),
		Severity:  "info",
		Data: map[string]any{
			"sessions": sessions,
		},
	}
	result.ok = n.NotifyUser(userID, SocketEventNotification, notification)
	return result.ok
}

func (n *SocketNotifier) NotifyUserPermissionChanged(userID string) {
	capabilities, err := user.NewRequestHandler(nil).GetCapabilities(userID)
	if err != nil {
		n.log.Error().Err(err).Str("userId", userID).Msg("failed to get capabilities for permission change notification")
		return
	}
	n.NotifyUser(userID, SocketEventNotification, NotificationData{
		Type:    NotificationTypePermissionChanged,
		Message: "User permissions have been updated",
		Data: map[string]any{
			"capabilities": capabilities,
		},
		Timestamp: time.Now().UnixMilli(),
		Severity:  "warning",
	})
}

func (n *SocketNotifier) NotifyUserPermissionChangedByServer(serverID string) {
	svc, err := n.service()
	if err != nil {
		return
	}
	users, err := repository.NewUserRepository(nil).FindAll()
	if err != nil {
		n.log.Error().Err(err).Str("serverId", serverID).Msg("failed loading users for permission notify")
		return
	}
	onlineUserMap := map[string]bool{}
	for _, id := range svc.GetOnlineUserIDs() {
		onlineUserMap[id] = true
	}

	for _, user := range users {
		if !onlineUserMap[user.UserID] {
			continue
		}
		if strings.TrimSpace(user.Permissions) == "" {
			n.NotifyUserPermissionChanged(user.UserID)
			continue
		}
		perms := mcptypes.Permissions{}
		if err := json.Unmarshal([]byte(user.Permissions), &perms); err != nil {
			n.log.Error().Err(err).Str("userId", user.UserID).Msg("invalid user permissions json")
			continue
		}
		if permission, exists := perms[serverID]; exists && !permission.Enabled {
			continue
		}
		n.NotifyUserPermissionChanged(user.UserID)
	}
}

func (n *SocketNotifier) NotifyServerStatusChanged(serverID string, serverName string, oldStatus int, newStatus int) {
	if oldStatus == newStatus {
		return
	}
	svc, err := n.service()
	if err != nil {
		return
	}

	onlineUserIDs := svc.GetOnlineUserIDs()
	if len(onlineUserIDs) == 0 {
		return
	}
	onlineUserMap := map[string]bool{}
	for _, id := range onlineUserIDs {
		onlineUserMap[id] = true
	}

	severity := "info"
	if newStatus == types.ServerStatusError {
		severity = "error"
	} else if newStatus == types.ServerStatusOffline {
		severity = "warning"
	}

	notification := NotificationData{
		Type:      NotificationTypeServerStatusChange,
		Message:   fmt.Sprintf("Server %s status changed", serverName),
		Timestamp: time.Now().UnixMilli(),
		Severity:  severity,
		Data: map[string]any{
			"serverId":   serverID,
			"serverName": serverName,
			"oldStatus":  oldStatus,
			"newStatus":  newStatus,
		},
	}

	users, err := repository.NewUserRepository(nil).FindAll()
	if err != nil {
		n.log.Error().Err(err).Str("serverId", serverID).Msg("failed loading users for server status notify")
		return
	}

	for _, user := range users {
		if !onlineUserMap[user.UserID] {
			continue
		}
		if strings.TrimSpace(user.Permissions) == "" {
			n.NotifyUser(user.UserID, SocketEventNotification, notification)
			continue
		}
		perms := mcptypes.Permissions{}
		if err := json.Unmarshal([]byte(user.Permissions), &perms); err != nil {
			n.log.Error().Err(err).Str("userId", user.UserID).Msg("invalid user permissions json")
			continue
		}
		if permission, exists := perms[serverID]; exists && !permission.Enabled {
			continue
		}
		n.NotifyUser(user.UserID, SocketEventNotification, notification)
	}
}

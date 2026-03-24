package socket

import (
	"fmt"
	"time"
)

type SocketErrorCode int

const (
	SocketErrorTimeout            SocketErrorCode = 1001
	SocketErrorUserOffline        SocketErrorCode = 1002
	SocketErrorServerError        SocketErrorCode = 1201
	SocketErrorServiceUnavailable SocketErrorCode = 1202
)

const (
	SocketEventClientMessage  = "client-message"
	SocketEventClientInfo     = "client-info"
	SocketEventNotification   = "notification"
	SocketEventAck            = "ack"
	SocketEventServerInfo     = "server_info"
	SocketEventSocketResponse = "socket_response"
)

const (
	NotificationTypeUserDisabled       = "user_disabled"
	NotificationTypePermissionChanged  = "permission_changed"
	NotificationTypeOnlineSessions     = "online_sessions"
	NotificationTypeServerStatusChange = "server_status_change"
	NotificationTypeUserExpired        = "user_expired"
)

type SocketActionType int

const (
	SocketActionAskUserConfirm      SocketActionType = 1001
	SocketActionGetClientStatus     SocketActionType = 2001
	SocketActionApprovalCreated     SocketActionType = 5001
	SocketActionApprovalDecided     SocketActionType = 5002
	SocketActionApprovalExpired     SocketActionType = 5003
	SocketActionGetPendingApprovals SocketActionType = 5004
)

type ApprovalCreatedData struct {
	ID               string    `json:"id"`
	ResumeToken      string    `json:"resumeToken"`
	ToolName         string    `json:"toolName"`
	ServerID         *string   `json:"serverId"`
	RedactedArgs     any       `json:"redactedArgs"`
	ExpiresAt        time.Time `json:"expiresAt"`
	CreatedAt        time.Time `json:"createdAt"`
	Status           string    `json:"status"`
	UniformRequestID *string   `json:"uniformRequestId,omitempty"`
	PolicyVersion    int       `json:"policyVersion"`
	MatchedRuleID    *string   `json:"matchedRuleId"`
	Reason           *string   `json:"reason"`
}

type ApprovalDecidedData struct {
	ID          string  `json:"id"`
	ResumeToken string  `json:"resumeToken"`
	ToolName    string  `json:"toolName"`
	Decision    string  `json:"decision"`
	Reason      *string `json:"reason,omitempty"`
}

type ApprovalExpiredData struct {
	ID          string `json:"id"`
	ResumeToken string `json:"resumeToken"`
	ToolName    string `json:"toolName"`
}

type ApprovalExecutedData struct {
	ID                       string  `json:"id"`
	ResumeToken              string  `json:"resumeToken"`
	ToolName                 string  `json:"toolName"`
	ExecutionResultAvailable bool    `json:"executionResultAvailable"`
	ExecutionResultPreview   *string `json:"executionResultPreview,omitempty"`
}

type ApprovalFailedData struct {
	ID                       string  `json:"id"`
	ResumeToken              string  `json:"resumeToken"`
	ToolName                 string  `json:"toolName"`
	Error                    string  `json:"error"`
	ExecutionResultAvailable bool    `json:"executionResultAvailable"`
	ExecutionResultPreview   *string `json:"executionResultPreview,omitempty"`
}

type UserConnection struct {
	UserID      string    `json:"userId"`
	SocketID    string    `json:"socketId"`
	DeviceType  string    `json:"deviceType,omitempty"`
	DeviceName  string    `json:"deviceName,omitempty"`
	AppVersion  string    `json:"appVersion,omitempty"`
	ConnectedAt time.Time `json:"connectedAt"`
}

type ClientInfo struct {
	DeviceType string `json:"deviceType,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
	AppVersion string `json:"appVersion,omitempty"`
	Platform   string `json:"platform,omitempty"`
}

type NotificationData struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
	Severity  string `json:"severity,omitempty"`
}

type SocketData struct {
	UserID     string `json:"userId"`
	UserToken  string `json:"userToken"`
	UserRole   int    `json:"userRole"`
	DeviceType string `json:"deviceType,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
	AppVersion string `json:"appVersion,omitempty"`
}

type SocketRequest[T any] struct {
	RequestID string           `json:"requestId"`
	Action    SocketActionType `json:"action"`
	Data      T                `json:"data"`
	Timestamp int64            `json:"timestamp"`
}

type SocketResponseError struct {
	Code    SocketErrorCode `json:"code"`
	Message string          `json:"message"`
	Details any             `json:"details,omitempty"`
}

type SocketResponse[T any] struct {
	RequestID string               `json:"requestId"`
	Success   bool                 `json:"success"`
	Data      *T                   `json:"data,omitempty"`
	Error     *SocketResponseError `json:"error,omitempty"`
	Timestamp int64                `json:"timestamp"`
}

func ActionToEventName(action SocketActionType) string {
	switch action {
	case SocketActionAskUserConfirm:
		return "ask_user_confirm"
	case SocketActionGetClientStatus:
		return "get_client_status"
	case SocketActionApprovalCreated:
		return "approval_created"
	case SocketActionApprovalDecided:
		return "approval_decided"
	case SocketActionApprovalExpired:
		return "approval_expired"
	case SocketActionGetPendingApprovals:
		return "get_pending_approvals"
	default:
		return fmt.Sprintf("unknown_action_%d", action)
	}
}

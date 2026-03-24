package core

import (
	"context"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DownstreamClient interface {
	ListTools(context.Context, *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	CallTool(context.Context, *mcp.CallToolParams) (*mcp.CallToolResult, error)
	ListResources(context.Context, *mcp.ListResourcesParams) (*mcp.ListResourcesResult, error)
	ListResourceTemplates(context.Context, *mcp.ListResourceTemplatesParams) (*mcp.ListResourceTemplatesResult, error)
	ReadResource(context.Context, *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error)
	Subscribe(context.Context, *mcp.SubscribeParams) error
	Unsubscribe(context.Context, *mcp.UnsubscribeParams) error
	ListPrompts(context.Context, *mcp.ListPromptsParams) (*mcp.ListPromptsResult, error)
	GetPrompt(context.Context, *mcp.GetPromptParams) (*mcp.GetPromptResult, error)
	Complete(context.Context, *mcp.CompleteParams) (*mcp.CompleteResult, error)
	Ping(context.Context, *mcp.PingParams) error
	Close() error
}

type TransportCloser interface {
	Close() error
}

type ServerRepository interface {
	FindAllEnabled(ctx context.Context) ([]database.Server, error)
	FindByServerID(ctx context.Context, serverID string) (*database.Server, error)
	UpdateCapabilities(ctx context.Context, serverID string, caps string) error
	UpdateCapabilitiesCache(ctx context.Context, serverID string, data map[string]any) error
	UpdateTransportType(ctx context.Context, serverID string, transportType string) error
	UpdateServerName(ctx context.Context, serverID string, serverName string) error
}

type UserRepository interface {
	FindByUserID(ctx context.Context, userID string) (*database.User, error)
	UpdateLaunchConfigs(ctx context.Context, userID string, launchConfigs string) error
	UpdateUserPreferences(ctx context.Context, userID string, userPreferences string) error
}

type EventRepository interface {
	Create(ctx context.Context, event *database.Event) error
	FindByStreamIDAfter(ctx context.Context, streamID string, afterEventID string) ([]database.Event, error)
	DeleteExpired(ctx context.Context) (int64, error)
	DeleteByStreamID(ctx context.Context, streamID string) (int64, error)
}

type SocketNotifier interface {
	AskUserConfirm(ctx context.Context, userID string, userAgent string, ip string, toolName string, description string, params string, timeout time.Duration) (bool, error)
	NotifyUserPermissionChanged(userID string)
	NotifyUserPermissionChangedByServer(serverID string)
	NotifyUserDisabled(userID string, reason string) bool
	NotifyUserExpired(userID string) bool
	NotifyOnlineSessions(userID string) bool
	NotifyApprovalCreated(userID string, approvalRequestID string, toolName string, serverID *string, redactedArgs any, expiresAt time.Time, createdAt time.Time, status string, uniformRequestID *string, policyVersion int, matchedRuleID *string, reason *string)
	NotifyApprovalDecided(userID string, approvalRequestID string, toolName string, decision string, reason *string)
	NotifyApprovalExpired(userID string, approvalRequestID string, toolName string)
	NotifyApprovalExecuted(userID string, approvalRequestID string, toolName string, executionResultAvailable bool, executionResultPreview *string)
	NotifyApprovalFailed(userID string, approvalRequestID string, toolName string, executionError string, executionResultAvailable bool, executionResultPreview *string)
	NotifyServerStatusChanged(serverID string, serverName string, oldStatus int, newStatus int)
	UpdateServerInfo()
}

type SessionLogger interface {
	LogClientRequest(ctx context.Context, entry map[string]any) error
	LogReverseRequest(ctx context.Context, entry map[string]any) error
	LogServerLifecycle(ctx context.Context, entry map[string]any) error
	LogError(ctx context.Context, entry map[string]any) error
	LogSessionLifecycle(action int, errMsg string)
	IP() string
}

type AuthStrategy interface {
	GetInitialToken(context.Context) (string, int64, error)
	RefreshToken(context.Context) (string, int64, error)
}

type AuthStrategyFactory interface {
	Build(ctx context.Context, server database.Server, launchConfig map[string]any, userToken string) (AuthStrategy, error)
}

type CapabilitiesComparator interface {
	ComparePermissions(oldPerms mcptypes.Permissions, newPerms mcptypes.Permissions) (toolsChanged bool, resourcesChanged bool, promptsChanged bool)
}

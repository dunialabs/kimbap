package types

const (
	UserStatusDisabled = 0
	UserStatusEnabled  = 1
)

const (
	UserRoleOwner = 1
	UserRoleAdmin = 2
	UserRoleUser  = 3
	UserRoleGuest = 4
)

const (
	ServerTypeLocal  = 1
	ServerTypeRemote = 2
)

const (
	ServerCategoryTemplate     = 1
	ServerCategoryCustomRemote = 2
	ServerCategoryRestAPI      = 3
	ServerCategorySkills       = 4
	ServerCategoryCustomStdio  = 5
)

const (
	ServerAuthTypeApiKey             = 1
	ServerAuthTypeGoogleAuth         = 2
	ServerAuthTypeNotionAuth         = 3
	ServerAuthTypeFigmaAuth          = 4
	ServerAuthTypeGoogleCalendarAuth = 5
	ServerAuthTypeGithubAuth         = 6
	ServerAuthTypeZendeskAuth        = 7
	ServerAuthTypeCanvasAuth         = 8
	ServerAuthTypeCanvaAuth          = 9
)

const (
	MCPEventLogTypeRequestTool             = 1001
	MCPEventLogTypeRequestResource         = 1002
	MCPEventLogTypeRequestPrompt           = 1003
	MCPEventLogTypeResponseTool            = 1004
	MCPEventLogTypeResponseResource        = 1005
	MCPEventLogTypeResponsePrompt          = 1006
	MCPEventLogTypeResponseToolList        = 1007
	MCPEventLogTypeResponseResourceList    = 1008
	MCPEventLogTypeResponsePromptList      = 1009
	MCPEventLogTypeReverseSamplingRequest  = 1201
	MCPEventLogTypeReverseSamplingResponse = 1202
	MCPEventLogTypeReverseRootsRequest     = 1203
	MCPEventLogTypeReverseRootsResponse    = 1204
	MCPEventLogTypeReverseElicitRequest    = 1205
	MCPEventLogTypeReverseElicitResponse   = 1206
	MCPEventLogTypeSessionInit             = 1301
	MCPEventLogTypeSessionClose            = 1302
	MCPEventLogTypeServerInit              = 1310
	MCPEventLogTypeServerClose             = 1311

	MCPEventLogTypeServerNotification = 1314

	MCPEventLogTypeAuthTokenValidation = 3001

	MCPEventLogTypeAuthRateLimit        = 3003
	MCPEventLogTypeAuthError            = 3010
	MCPEventLogTypeErrorInternal        = 4001
	MCPEventLogTypeAdminUserCreate      = 5001
	MCPEventLogTypeAdminUserEdit        = 5002
	MCPEventLogTypeAdminUserDelete      = 5003
	MCPEventLogTypeAdminServerCreate    = 5004
	MCPEventLogTypeAdminServerEdit      = 5005
	MCPEventLogTypeAdminServerDelete    = 5006
	MCPEventLogTypeAdminProxyReset      = 5007
	MCPEventLogTypeAdminBackupDatabase  = 5008
	MCPEventLogTypeAdminRestoreDatabase = 5009
	MCPEventLogTypeAdminDNSCreate       = 5010
	MCPEventLogTypeAdminDNSDelete       = 5011
)

const (
	ClientSessionStatusActive = 0
	ClientSessionStatusClosed = 1
)

const (
	ServerStatusOnline     = 0
	ServerStatusOffline    = 1
	ServerStatusConnecting = 2
	ServerStatusError      = 3
	ServerStatusSleeping   = 4
)

const (
	DangerLevelSilent       = 0
	DangerLevelNotification = 1
	DangerLevelApproval     = 2
)

const (
	ApprovalStatusPending   = "PENDING"
	ApprovalStatusApproved  = "APPROVED"
	ApprovalStatusRejected  = "REJECTED"
	ApprovalStatusExpired   = "EXPIRED"
	ApprovalStatusExecuting = "EXECUTING"
	ApprovalStatusExecuted  = "EXECUTED"
	ApprovalStatusFailed    = "FAILED"
)

const (
	PolicyDecisionAllow           = "ALLOW"
	PolicyDecisionRequireApproval = "REQUIRE_APPROVAL"
	PolicyDecisionDeny            = "DENY"
)

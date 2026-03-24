# LEGACY BASELINE: PETA-CORE-GO FUNCTIONALITY INVENTORY

> **⚠️ This document is a legacy migration artifact.** It describes the peta-core-go baseline
> that Kimbap is built from. It does NOT represent the current Kimbap architecture.
> See the main README.md for current architecture.

**Project**: peta-core-go (legacy Go implementation, now kimbap-core)  
**Go Version**: 1.24+  
**Database**: PostgreSQL 15+  
**Total Go Files**: 108 (excluding tests)  
**Total Exported Methods**: 586+

---

## TABLE OF CONTENTS

1. [Types & Enums](#types--enums)
2. [Database Models](#database-models)
3. [Repository Layer](#repository-layer)
4. [Security & Authentication](#security--authentication)
5. [Middleware](#middleware)
6. [MCP Core](#mcp-core)
7. [MCP OAuth](#mcp-oauth)
8. [MCP Auth Strategies](#mcp-auth-strategies)
9. [OAuth 2.0 Implementation](#oauth-20-implementation)
10. [Admin API](#admin-api)
11. [User API](#user-api)
12. [Socket.IO](#socketio)
13. [Services](#services)
14. [Logging & Utilities](#logging--utilities)
15. [Configuration](#configuration)

---

## TYPES & ENUMS

### internal/types/auth.go

**Structs:**
- `AuthContext` - User authentication context with permissions, token, role, status
- `AuthError` - Error type for authentication failures

**Constants:**
- `AuthErrorType*` - Error type constants (INVALID_TOKEN, USER_NOT_FOUND, USER_DISABLED, USER_EXPIRED, RATE_LIMIT_EXCEEDED)

**Functions:**
- `NewAuthError(errType, message, userID string, details interface{}) *AuthError`

### internal/types/admin.go

**Structs:**
- `AdminRequest` - Admin API request with action code and data
- `AdminResponse` - Admin API response with success flag and optional error
- `AdminResponseError` - Error details in admin response
- `AdminError` - Admin error type

**Constants:**
- `AdminAction*` - Action codes (1001-10043) for user, server, proxy, IP whitelist, backup, cloudflared, skills operations
- `AdminErrorCode*` - Error codes for admin operations

### internal/types/enums.go

**Constants:**
- `UserStatus*` - User status (DISABLED=0, ENABLED=1, PENDING=2, SUSPENDED=3)
- `UserRole*` - User roles (OWNER=1, ADMIN=2, USER=3, GUEST=4)
- `ServerType*` - Server types (LOCAL=1, REMOTE=2)
- `ServerCategory*` - Server categories (TEMPLATE=1, CUSTOM_REMOTE=2, REST_API=3, SKILLS=4)
- `ServerAuthType*` - Auth types (API_KEY=1, GOOGLE=2, NOTION=3, FIGMA=4, GOOGLE_CALENDAR=5, GITHUB=6, ZENDESK=7, CANVAS=8)
- `MCPEventLogType*` - Event log types (1001-5011) for tools, resources, prompts, OAuth, auth, admin operations
- `ClientSessionStatus*` - Session status (ACTIVE=0, CLOSED=1, EXPIRED=2)
- `ServerStatus*` - Server status (ONLINE=0, OFFLINE=1, CONNECTING=2, ERROR=3, SLEEPING=4)
- `DangerLevel*` - Danger levels (SILENT=0, NOTIFICATION=1, APPROVAL=2)

### internal/mcp/types/mcp.go

**Structs:**
- `BaseCapabilityConfig` - Base capability configuration
- `ToolCapabilityConfig` - Tool capability with enabled flag
- `ResourceCapabilityConfig` - Resource capability (alias to BaseCapabilityConfig)
- `PromptCapabilityConfig` - Prompt capability (alias to BaseCapabilityConfig)
- `ServerConfigCapabilities` - Server capabilities configuration
- `ServerConfigWithEnabled` - Server config with enabled flag
- `McpRequest` - MCP request structure
- `JSONRPCError` - JSON-RPC error
- `McpResponse` - MCP response
- `McpNotification` - MCP notification
- `ProxyContext` - Proxy context
- `AuthContext` - Auth context in MCP
- `JSONRPCMessage` - JSON-RPC message
- `ReplayOptions` - Event replay options
- `CachedEvent` - Cached event

**Type Aliases:**
- `Permissions` - Map of server ID to ServerConfigWithEnabled
- `McpServerCapability` - Alias to ServerConfigWithEnabled
- `McpServerCapabilities` - Alias to Permissions
- `StreamID` - String type for stream ID
- `EventID` - String type for event ID
- `DisconnectReason` - String type for disconnect reason

**Interfaces:**
- `EventStore` - Event storage interface

---

## DATABASE MODELS

### internal/database/models.go

**Proxy Model:**
- `ID` (int, PK, auto-increment)
- `Name` (varchar 128)
- `Addtime` (int)
- `ProxyKey` (varchar 255)
- `StartPort` (int, default 3002)
- `LogWebhookURL` (text, nullable)
- `LastSyncedLogID` (int, default 0)

**User Model:**
- `UserID` (varchar 64, PK)
- `Status` (int)
- `Role` (int)
- `Permissions` (text)
- `UserPreferences` (text, default '{}')
- `LaunchConfigs` (text, default '{}')
- `ExpiresAt` (int, default 0)
- `CreatedAt` (int, default 0)
- `UpdatedAt` (int, default 0)
- `Ratelimit` (int)
- `Name` (varchar 128)
- `EncryptedToken` (text, nullable)
- `ProxyID` (int, default 0)
- `Notes` (text, nullable)

**Server Model:**
- `ServerID` (varchar 128, PK)
- `ServerName` (varchar 128)
- `Enabled` (bool, default true)
- `LaunchConfig` (text)
- `Capabilities` (text)
- `CreatedAt` (int, default 0)
- `UpdatedAt` (int, default 0)
- `AllowUserInput` (bool, default false)
- `ConfigTemplate` (text, nullable)
- `ProxyID` (int, default 0)
- `ToolTmplID` (varchar 128, nullable)
- `AuthType` (int, default 1)
- `Category` (int, default 1)
- `PublicAccess` (bool, default false)
- `UseKimbapOauthConfig` (bool, default true)
- `TransportType` (varchar 10, nullable)
- `LazyStartEnabled` (bool, default true)
- `CachedTools` (text, nullable)
- `CachedResources` (text, nullable)
- `CachedResourceTemplates` (text, nullable)
- `CachedPrompts` (text, nullable)

**Log Model:**
- `ID` (int, PK, auto-increment)
- `CreatedAt` (int, indexed)
- `Action` (int)
- `UserID` (varchar, indexed)
- `ServerID` (varchar 128, indexed, nullable)
- `SessionID` (varchar, indexed)
- `UpstreamRequestID` (varchar)
- `UniformRequestID` (varchar, indexed, nullable)
- `ParentUniformRequestID` (varchar, nullable)
- `ProxyRequestID` (varchar, nullable)
- `IP` (varchar)
- `UA` (varchar)
- `TokenMask` (varchar)
- `RequestParams` (text)
- `ResponseResult` (text)
- `Error` (text)
- `Duration` (int, nullable)
- `StatusCode` (int, nullable)

**Event Model:**
- `ID` (int, PK, auto-increment)
- `EventID` (varchar 255, unique)
- `StreamID` (varchar 255, indexed)
- `SessionID` (varchar 255, indexed)
- `MessageType` (varchar 50)
- `MessageData` (text)
- `CreatedAt` (timestamp, indexed, auto-create)
- `ExpiresAt` (timestamp, indexed)

**DnsConf Model:**
- `ID` (int, PK, auto-increment)
- `Subdomain` (varchar 128)
- `Type` (int)
- `PublicIP` (varchar 128)
- `Addtime` (int)
- `UpdateTime` (int)
- `TunnelID` (varchar 256)
- `ProxyID` (int)
- `CreatedBy` (int)
- `Credentials` (text)

**IpWhitelist Model:**
- `ID` (int, PK, auto-increment)
- `IP` (varchar 128)
- `Addtime` (int)

**License Model:**
- `ID` (int, PK, auto-increment)
- `LicenseStr` (text)
- `Addtime` (int)
- `Status` (int)

**OAuthClient Model:**
- `ClientID` (varchar 255, PK)
- `ClientSecret` (varchar 255, nullable)
- `TokenEndpointAuthMethod` (varchar 50, default 'client_secret_basic')
- `Name` (varchar 255)
- `RedirectUris` (jsonb)
- `Scopes` (jsonb)
- `GrantTypes` (jsonb)
- `ResponseTypes` (jsonb, default '[]')
- `UserID` (varchar 64, nullable)
- `Trusted` (bool, default false)
- `CreatedAt` (timestamp, auto-create)
- `UpdatedAt` (timestamp, auto-update)
- Relations: AuthorizationCodes, Tokens

**OAuthAuthorizationCode Model:**
- `Code` (varchar 255, PK)
- `ClientID` (varchar 255, indexed)
- `UserID` (varchar 64, indexed)
- `RedirectURI` (text)
- `Scopes` (jsonb)
- `Resource` (text, nullable)
- `CodeChallenge` (varchar 255, nullable)
- `ChallengeMethod` (varchar 10, nullable)
- `ExpiresAt` (timestamp, indexed)
- `CreatedAt` (timestamp, auto-create)
- `Used` (bool, default false)
- Relation: Client

**OAuthToken Model:**
- `TokenID` (varchar, PK)
- `AccessToken` (text, unique, indexed)
- `RefreshToken` (text, unique, indexed, nullable)
- `ClientID` (varchar 255, indexed)
- `UserID` (varchar 64, indexed)
- `Scopes` (jsonb)
- `Resource` (text, nullable)
- `AccessTokenExpiresAt` (timestamp, indexed)
- `RefreshTokenExpiresAt` (timestamp, nullable)
- `CreatedAt` (timestamp, auto-create)
- `UpdatedAt` (timestamp, auto-update)
- `Revoked` (bool, default false)
- Relation: Client

---

## REPOSITORY LAYER

### internal/repository/server.go

**ServerRepository Struct:**
- `NewServerRepository(db *gorm.DB) *ServerRepository`

**Methods:**
- `FindAll() ([]database.Server, error)`
- `FindEnabled() ([]database.Server, error)`
- `FindByServerId(serverID string) (*database.Server, error)`
- `Create(server *database.Server) (*database.Server, error)`
- `Update(serverID string, updates map[string]any) (*database.Server, error)`
- `Delete(serverID string) (*database.Server, error)`
- `Enable(serverID string) (*database.Server, error)`
- `Disable(serverID string) (*database.Server, error)`
- `Exists(serverID string) (bool, error)`
- `UpdateLaunchConfig(serverID string, launchConfig string) (*database.Server, error)`
- `UpdateCapabilities(serverID string, capabilities string) (*database.Server, error)`
- `UpdateCapabilitiesCache(serverID string, data ServerCapabilitiesCacheInput) error`
- `FindByProxyId(proxyID int) ([]database.Server, error)`
- `CountAll() (int64, error)`
- `Count() (int64, error)`
- `DeleteByProxyId(proxyID int) (int64, error)`
- `BulkCreate(servers []database.Server) (int64, error)`
- `Upsert(serverID string, createData *database.Server, updateData map[string]any) (*database.Server, error)`

### internal/repository/user.go

**UserRepository Struct:**
- `NewUserRepository(db *gorm.DB) *UserRepository`

**Methods:**
- `FindAll() ([]database.User, error)`
- `FindById(userID string) (*database.User, error)`
- `FindByUserId(userID string) (*database.User, error)`
- `Create(user *database.User) (*database.User, error)`
- `Update(userID string, updates map[string]any) (*database.User, error)`
- `Delete(userID string) (*database.User, error)`
- `UpdatePermissions(userID string, permissions any) (*database.User, error)`
- `UpdateUserPreferences(userID string, userPreferences any) (*database.User, error)`
- `UpdateLaunchConfigs(userID string, launchConfigs any) (*database.User, error)`
- `FindByProxyId(proxyID int) ([]database.User, error)`
- `FindByStatus(status int) ([]database.User, error)`
- `FindByRole(role int) ([]database.User, error)`
- `Exists(userID string) (bool, error)`
- `RemoveServerFromAllUsers(serverID string) error`
- `Upsert(userID string, createData *database.User, updateData map[string]any) (*database.User, error)`

### internal/repository/event.go

**EventRepository Struct:**
- `NewEventRepository(db *gorm.DB) *EventRepository`

**Methods:**
- `Save(event *database.Event) (*database.Event, error)`
- `Create(event *database.Event) (*database.Event, error)`
- `FindByEventId(eventID string) (*database.Event, error)`
- `FindByStreamId(streamID string) ([]database.Event, error)`
- `FindBySessionId(sessionID string) ([]database.Event, error)`
- `FindAfterEventId(streamID, afterEventID string) ([]database.Event, error)`
- `CreateMany(events []database.Event) (int64, error)`
- `DeleteByStreamId(streamID string) (int64, error)`
- `DeleteBySessionId(sessionID string) (int64, error)`
- `DeleteExpired() (int64, error)`
- `Count() (int64, error)`
- `DeleteBefore(date time.Time) (int64, error)`
- `GetStats() (*EventStats, error)`
- `DeleteAll() error`

**EventStats Struct:**
- Contains event statistics

### internal/repository/log.go

**LogRepository Struct:**
- `NewLogRepository(db *gorm.DB) *LogRepository`

**Methods:**
- `BatchInsert(entries []database.Log) error`
- `Save(entry database.Log) (*database.Log, error)`
- `FindLogsFromId(startID, limit int) ([]database.Log, error)`
- `FindByFilters(filters LogFilters, page, pageSize int) ([]database.Log, error)`
- `Count(filters LogFilters) (int64, error)`
- `DeleteOlderThan(timestamp int) (int64, error)`
- `FindById(id int) (*database.Log, error)`
- `GetLatestId() (int, error)`
- `GetMaxLogId() (int, error)`
- `DeleteAll() error`

**LogFilters Struct:**
- Filtering criteria for logs

### internal/repository/oauth_token.go

**OAuthTokenRepository Struct:**
- `NewOAuthTokenRepository(db *gorm.DB) *OAuthTokenRepository`

**Methods:**
- Token CRUD operations for OAuth tokens

### internal/repository/proxy.go

**ProxyRepository Struct:**
- `NewProxyRepository(db *gorm.DB) *ProxyRepository`

**Methods:**
- Proxy CRUD operations

### internal/repository/ipwhitelist.go

**IpWhitelistRepository Struct:**
- `NewIpWhitelistRepository(db *gorm.DB) *IpWhitelistRepository`

**Methods:**
- IP whitelist CRUD operations

---

## SECURITY & AUTHENTICATION

### internal/security/token.go

**TokenValidator Struct:**
- `NewTokenValidator() *TokenValidator`

**Methods:**
- Token validation methods

### internal/security/crypto.go

**EncryptedData Struct:**
- Encrypted data structure

**Functions:**
- `CalculateUserID(token string) string` - Calculate user ID from token
- `EncryptData(plaintext string, key string) (string, error)` - Encrypt data with AES-GCM
- `DecryptDataFromString(encryptedStr string, key string) (string, error)` - Decrypt data

### internal/security/oauth_token.go

**OAuthTokenValidator Struct:**
- `NewOAuthTokenValidator(repository OAuthTokenRepository, userRepository UserRepository) *OAuthTokenValidator`

**Methods:**
- OAuth token validation and refresh

**OAuthTokenRecord & UserRecord Structs:**
- Data structures for OAuth tokens and users

### internal/security/ratelimit.go

**RateLimitService Struct:**
- `NewRateLimitService() *RateLimitService`

**Methods:**
- Rate limiting operations

### internal/security/ipwhitelist.go

**IpWhitelistService Struct:**
- `NewIpWhitelistService(store IPWhitelistStore) *IpWhitelistService`

**Methods:**
- IP whitelist checking and management

**IPWhitelistStore Interface:**
- Interface for IP whitelist storage

---

## MIDDLEWARE

### internal/middleware/auth.go

**AuthMiddleware Struct:**
- `NewAuthMiddleware(tv tokenValidator, oauth *security.OAuthTokenValidator, repo UserRepository) *AuthMiddleware`

**Methods:**
- `Middleware(next http.Handler) http.Handler` - HTTP middleware
- `AuthenticateRequest(r *http.Request) (*types.AuthContext, error)` - Authenticate request
- `validateToken(ctx context.Context, token string) (userID, clientID string, scopes []string, err error)`
- `refreshUserInfo(ctx context.Context, session *core.ClientSession) error`
- `refreshUserInfoIfNeeded(ctx context.Context, session *core.ClientSession)`
- `shouldRefresh(userID string) bool`
- `markRefreshed(userID string)`
- `cleanupRefreshedLocked(now time.Time)`

**Functions:**
- `GetAuthContext(ctx context.Context) (*types.AuthContext, bool)` - Get auth context from request
- `ExtractAuthToken(r *http.Request) (string, error)` - Extract token from request
- `WriteRequestEntityTooLargeLikeExpress(w http.ResponseWriter)` - Write 413 response

**User Interface:**
- User data interface for middleware

**UserRepository Interface:**
- User repository interface for middleware

### internal/middleware/adminauth.go

**AdminAuthMiddleware Struct:**
- `NewAdminAuthMiddleware(auth *AuthMiddleware) *AdminAuthMiddleware`

**Methods:**
- `Middleware(next http.Handler) http.Handler` - Admin auth middleware

### internal/middleware/ratelimit.go

**RateLimitMiddleware Struct:**
- `NewRateLimitMiddleware(service rateLimiter, defaultLimit int) *RateLimitMiddleware`

**Methods:**
- `Middleware(next http.Handler) http.Handler` - Rate limit middleware

### internal/middleware/ipwhitelist.go

**IPWhitelistMiddleware Struct:**
- `NewIPWhitelistMiddleware(service ipWhitelistChecker) *IPWhitelistMiddleware`

**IPWhitelistStatus Struct:**
- Status information for IP whitelist

**Methods:**
- `Middleware(next http.Handler) http.Handler` - IP whitelist middleware
- `CreatePathSpecificMiddleware(paths []string) func(http.Handler) http.Handler` - Path-specific middleware
- `GetStatus() IPWhitelistStatus` - Get whitelist status

**Functions:**
- `ClientIPFromRequest(r *http.Request) string` - Extract client IP

### internal/middleware/context.go

**Context utilities:**
- Context key definitions and helpers

---

## MCP CORE

### internal/mcp/core/interfaces.go

**DownstreamClient Interface:**
- `ListTools(context.Context, *mcp.ListToolsParams) (*mcp.ListToolsResult, error)`
- `CallTool(context.Context, *mcp.CallToolParams) (*mcp.CallToolResult, error)`
- `ListResources(context.Context, *mcp.ListResourcesParams) (*mcp.ListResourcesResult, error)`
- `ListResourceTemplates(context.Context, *mcp.ListResourceTemplatesParams) (*mcp.ListResourceTemplatesResult, error)`
- `ReadResource(context.Context, *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error)`
- `Subscribe(context.Context, *mcp.SubscribeParams) error`
- `Unsubscribe(context.Context, *mcp.UnsubscribeParams) error`
- `ListPrompts(context.Context, *mcp.ListPromptsParams) (*mcp.ListPromptsResult, error)`
- `GetPrompt(context.Context, *mcp.GetPromptParams) (*mcp.GetPromptResult, error)`
- `Complete(context.Context, *mcp.CompleteParams) (*mcp.CompleteResult, error)`
- `Ping(context.Context, *mcp.PingParams) error`
- `Close() error`

**TransportCloser Interface:**
- `Close() error`

**ServerRepository Interface:**
- `FindAllEnabled(ctx context.Context) ([]database.Server, error)`
- `FindByServerID(ctx context.Context, serverID string) (*database.Server, error)`
- `UpdateCapabilities(ctx context.Context, serverID string, caps string) error`
- `UpdateCapabilitiesCache(ctx context.Context, serverID string, data map[string]any) error`
- `UpdateTransportType(ctx context.Context, serverID string, transportType string) error`
- `UpdateServerName(ctx context.Context, serverID string, serverName string) error`

**UserRepository Interface:**
- `FindByUserID(ctx context.Context, userID string) (*database.User, error)`
- `UpdateLaunchConfigs(ctx context.Context, userID string, launchConfigs string) error`
- `UpdateUserPreferences(ctx context.Context, userID string, userPreferences string) error`

**EventRepository Interface:**
- `Create(ctx context.Context, event *database.Event) error`
- `FindByStreamIDAfter(ctx context.Context, streamID string, afterEventID string) ([]database.Event, error)`
- `DeleteExpired(ctx context.Context) (int64, error)`
- `DeleteByStreamID(ctx context.Context, streamID string) (int64, error)`

**SocketNotifier Interface:**
- `AskUserConfirm(ctx context.Context, userID string, userAgent string, ip string, toolName string, description string, params string, timeout time.Duration) (bool, error)`
- `NotifyUserPermissionChanged(userID string)`
- `NotifyUserPermissionChangedByServer(serverID string)`
- `NotifyUserDisabled(userID string, reason string) bool`
- `NotifyOnlineSessions(userID string) bool`
- `UpdateServerInfo()`

**SessionLogger Interface:**
- `LogClientRequest(ctx context.Context, entry map[string]any) error`
- `LogReverseRequest(ctx context.Context, entry map[string]any) error`
- `LogServerLifecycle(ctx context.Context, entry map[string]any) error`
- `LogError(ctx context.Context, entry map[string]any) error`
- `LogSessionLifecycle(action int, errMsg string)`
- `IP() string`

**AuthStrategy Interface:**
- `GetInitialToken(context.Context) (string, int64, error)`
- `RefreshToken(context.Context) (string, int64, error)`
- `Cleanup()`

**AuthStrategyFactory Interface:**
- `Build(ctx context.Context, server database.Server, launchConfig map[string]any, userToken string) (AuthStrategy, error)`

**ProxySessionReverse Interface:**
- `CanServerRequestSampling() bool`
- `CanServerRequestElicitation() bool`
- `ClientSupportsRoots() bool`
- `ForwardSamplingToClient(context.Context, *mcp.CreateMessageRequest, string) (*mcp.CreateMessageResult, error)`
- `ForwardRootsListToClient(context.Context, *mcp.ListRootsRequest, string) (*mcp.ListRootsResult, error)`
- `ForwardElicitationToClient(context.Context, *mcp.ElicitRequest, string) (*mcp.ElicitResult, error)`

**SessionReader Interface:**
- `GetProxySession(sessionID string) *ProxySession`
- `GetAllSessions() []*ClientSession`
- `GetSessionLogger(sessionID string) SessionLogger`
- `GetUserFirstSession(userID string) *ClientSession`

**CapabilitiesComparator Interface:**
- `ComparePermissions(oldPerms mcptypes.Permissions, newPerms mcptypes.Permissions) (toolsChanged bool, resourcesChanged bool, promptsChanged bool)`

### internal/mcp/core/proxysession.go

**ProxySession Struct:**
- `NewProxySession(sessionID, userID string, clientSession *ClientSession, logger SessionLogger, eventStore *PersistentEventStore, onClose func(string, mcptypes.DisconnectReason)) *ProxySession`

**Methods:**
- `setupRequestHandlers()`
- `handleToolsList(ctx context.Context, req mcp.Request, requestID string) *mcp.ListToolsResult`
- `handleResourcesList(ctx context.Context, req mcp.Request, requestID string) *mcp.ListResourcesResult`
- `handleResourcesTemplatesList(ctx context.Context, req mcp.Request, requestID string) *mcp.ListResourceTemplatesResult`
- `handlePromptsList(ctx context.Context, req mcp.Request, requestID string) *mcp.ListPromptsResult`
- `handleToolCall(ctx context.Context, req mcp.Request, requestID string) (mcp.Result, error)`
- `handleResourceRead(ctx context.Context, req mcp.Request, requestID string) (mcp.Result, error)`
- `handlePromptGet(ctx context.Context, req mcp.Request, requestID string) (mcp.Result, error)`
- `handleComplete(ctx context.Context, params *mcp.CompleteParams, requestID string, depth int) (*mcp.CompleteResult, error)`
- `HandleSubscribe(ctx context.Context, params *mcp.SubscribeParams, requestID string) error`
- `HandleUnsubscribe(ctx context.Context, params *mcp.UnsubscribeParams, requestID string) error`
- `handlePing(ctx context.Context, req mcp.Request, requestID string) error`
- `handleLoggingSetLevel(ctx context.Context, req mcp.Request, requestID string)`
- `Close(reason mcptypes.DisconnectReason) error`
- `GetHTTPHandler() *mcp.StreamableHTTPHandler`
- `GetServer() *mcp.Server`

### internal/mcp/core/clientsession.go

**ClientSession Struct:**
- `NewClientSession(sessionID, userID, token string, authContext mcptypes.AuthContext) *ClientSession`

**Methods:**
- `ConnectionInitialized(server *mcp.Server)`
- `SetProxySession(ps *ProxySession)`
- `GetProxySession() *ProxySession`
- `GetSessionID() string`
- `GetUserID() string`
- `GetToken() string`
- `GetAuthContext() mcptypes.AuthContext`
- `Close(reason mcptypes.DisconnectReason) error`

**ClientInfo Struct:**
- Client information

### internal/mcp/core/servermanager.go

**serverManager Struct (Singleton):**
- `ServerManagerInstance() *serverManager`

**Methods:**
- `Configure(repo ServerRepository, users UserRepository, authFactory AuthStrategyFactory, notifier SocketNotifier)`
- `Notifier() SocketNotifier`
- `UserRepository() UserRepository`
- `Repository() ServerRepository`
- `NotifyUserPermissionChangedByServer(serverID string)`
- `NotifyUserPermissionChanged(userID string)`
- `SetOwnerToken(token string)`
- `GetOwnerToken() (string, error)`
- `AddServer(ctx context.Context, server database.Server, token string) (*ServerContext, error)`
- `GetServer(serverID string) *ServerContext`
- `GetAllServers() []*ServerContext`
- `RemoveServer(serverID string) error`
- `ConnectServer(ctx context.Context, serverID string) (*ServerConnectResult, error)`
- `DisconnectServer(ctx context.Context, serverID string) error`
- `GetServerStatus(serverID string) int`
- `RefreshServerCapabilities(ctx context.Context, serverID string) error`
- `Shutdown(ctx context.Context) error`

**ServerConnectResult Struct:**
- Result of server connection

### internal/mcp/core/sessionstore.go

**SessionStore Struct (Singleton):**
- `SessionStoreInstance() *SessionStore`

**Methods:**
- `SetEventRepository(repo EventRepository)`
- `SetNotifier(notifier SocketNotifier)`
- `CreateSession(ctx context.Context, sessionID, userID, token string, authContext mcptypes.AuthContext, logger SessionLogger) (*ClientSession, error)`
- `GetSession(sessionID string) *ClientSession`
- `GetProxySession(sessionID string) *ProxySession`
- `GetAllProxySessions() []*ProxySession`
- `GetEventStore(sessionID string) *PersistentEventStore`
- `GetSessionLogger(sessionID string) SessionLogger`
- `GetAllSessions() []*ClientSession`
- `GetUserSessions(userID string) []*ClientSession`
- `GetSessionsUsingServer(serverID string) []*ClientSession`
- `RemoveSession(sessionID string, reason mcptypes.DisconnectReason) error`
- `GetTotalCreated() int64`
- `Shutdown(ctx context.Context) error`

### internal/mcp/core/servercontext.go

**ServerContext Struct:**
- `NewServerContext(server database.Server) *ServerContext`

**Methods:**
- Server context management methods

### internal/mcp/core/eventstore.go

**PersistentEventStore Struct:**
- `NewPersistentEventStore(sessionID, userID string, repo EventRepository) *PersistentEventStore`

**Methods:**
- `Store(ctx context.Context, event *database.Event) error`
- `GetEvents(ctx context.Context, afterEventID string) ([]database.Event, error)`
- `Cleanup(ctx context.Context) error`

### internal/mcp/core/eventcleanup.go

**EventCleanupService Struct:**
- `NewEventCleanupService(repo EventRepository) *EventCleanupService`

**Methods:**
- Event cleanup operations

### internal/mcp/core/eventreplay.go

**EventReplayService Struct:**
- `NewEventReplayService(store *PersistentEventStore) *EventReplayService`

**ReplayStats Struct:**
- Replay statistics

**Methods:**
- Event replay operations

### internal/mcp/core/requestidmapper.go

**RequestIDMapper Struct:**
- `NewRequestIDMapper(sessionID string) *RequestIDMapper`

**MappingEntry Struct:**
- Request ID mapping entry

**Methods:**
- Request ID mapping operations

### internal/mcp/core/globalrequestrouter.go

**GlobalRequestRouter Struct (Singleton):**
- `GlobalRequestRouterInstance() *GlobalRequestRouter`

**Methods:**
- Global request routing operations

### internal/mcp/core/transportfactory.go

**DownstreamTransportFactory Struct:**
- `NewDownstreamTransportFactory() *DownstreamTransportFactory`

**DownstreamTransportType Type:**
- Transport type definition

**CreatedTransport Struct:**
- Created transport information

**Methods:**
- Transport creation methods

### internal/mcp/core/notifications.go

**ResourceUpdatedNotification Struct:**
- Resource update notification

---

## MCP OAUTH

### internal/mcp/oauth/types.go

**OAuthProvider Type:**
- OAuth provider type

**OAuthConfig Struct:**
- OAuth configuration

**ExchangeContext Struct:**
- OAuth exchange context

**TokenResponse Struct:**
- OAuth token response

**ExchangeResult Struct:**
- OAuth exchange result

**ProviderRequest Struct:**
- Provider request structure

**HTTPResponse Struct:**
- HTTP response structure

**ProviderAdapter Struct:**
- Provider adapter

### internal/mcp/oauth/errors.go

**OAuthExchangeErrorType Type:**
- Error type for OAuth exchange

**OAuthExchangeError Struct:**
- OAuth exchange error

**Functions:**
- `NewOAuthExchangeError(message string, errType OAuthExchangeErrorType, provider string, status int, responseBody string, cause error) *OAuthExchangeError`
- `NewOAuthHTTPError(provider string, status int, responseBody string) *OAuthExchangeError`
- `NewOAuthParseError(provider string, responseBody string, cause error) *OAuthExchangeError`
- `NewOAuthUnknownProviderError(provider string) *OAuthExchangeError`

### internal/mcp/oauth/exchange.go

**Functions:**
- `ExchangeAuthorizationCode(ctx ExchangeContext) (*ExchangeResult, error)` - Exchange OAuth authorization code

### internal/mcp/oauth/http.go

**Functions:**
- `OAuthHTTPPost(url string, request ProviderRequest, provider string) (*HTTPResponse, error)` - HTTP POST for OAuth

### internal/mcp/oauth/index.go

**Functions:**
- `GetSupportedProviders() []string` - Get list of supported OAuth providers

### internal/mcp/oauth/utils.go

**Functions:**
- `ResolveExpires(responseExpiresIn *int64, defaultExpiresIn *int64) (*int64, *int64)` - Resolve token expiration
- `NumberToInt64(v interface{}) (int64, bool)` - Convert number to int64

### internal/mcp/oauth/providers/registry.go

**UnknownProviderError Struct:**
- Unknown provider error

**Functions:**
- `GetProviderAdapter(provider string) (ProviderAdapter, error)` - Get provider adapter
- `GetSupportedProviders() []string` - Get supported providers

### internal/mcp/oauth/providers/types.go

**ExchangeContext Struct:**
- Provider exchange context

**ProviderRequest Struct:**
- Provider request

**ProviderAdapter Struct:**
- Provider adapter

### internal/mcp/oauth/providers/google.go

**Functions:**
- Google OAuth provider implementation

### internal/mcp/oauth/providers/github.go

**Functions:**
- GitHub OAuth provider implementation

### internal/mcp/oauth/providers/notion.go

**Functions:**
- Notion OAuth provider implementation

### internal/mcp/oauth/providers/figma.go

**Functions:**
- Figma OAuth provider implementation

### internal/mcp/oauth/providers/zendesk.go

**Functions:**
- Zendesk OAuth provider implementation

### internal/mcp/oauth/providers/stripe.go

**Functions:**
- Stripe OAuth provider implementation

### internal/mcp/oauth/providers/canvas.go

**Functions:**
- Canvas OAuth provider implementation

---

## MCP AUTH STRATEGIES

### internal/mcp/auth/strategy.go

**AuthStrategy Interface:**
- `GetInitialToken(context.Context) (string, int64, error)`
- `RefreshToken(context.Context) (string, int64, error)`
- `Cleanup()`

**TokenInfo Struct:**
- Token information

### internal/mcp/auth/factory.go

**AuthStrategyFactory Struct:**
- `Create(authType int, oauthConfig map[string]interface{}) (AuthStrategy, error)` - Create auth strategy

### internal/mcp/auth/kimbap.go

**KimbapAuthStrategy Struct:**
- `NewKimbapAuthStrategy(config map[string]interface{}) (*KimbapAuthStrategy, error)`

**Methods:**
- `GetInitialToken(context.Context) (string, int64, error)`
- `RefreshToken(context.Context) (string, int64, error)`
- `Cleanup()`

### internal/mcp/auth/google.go

**GoogleAuthStrategy Struct:**
- `NewGoogleAuthStrategy(config map[string]interface{}) (*GoogleAuthStrategy, error)`

**Methods:**
- `GetInitialToken(context.Context) (string, int64, error)`
- `RefreshToken(context.Context) (string, int64, error)`
- `Cleanup()`

### internal/mcp/auth/github.go

**GithubAuthStrategy Struct:**
- `NewGithubAuthStrategy(config map[string]interface{}) (*GithubAuthStrategy, error)`

**Methods:**
- `GetInitialToken(context.Context) (string, int64, error)`
- `RefreshToken(context.Context) (string, int64, error)`
- `Cleanup()`

### internal/mcp/auth/notion.go

**NotionAuthStrategy Struct:**
- `NewNotionAuthStrategy(config map[string]interface{}) (*NotionAuthStrategy, error)`

**Methods:**
- `GetInitialToken(context.Context) (string, int64, error)`
- `RefreshToken(context.Context) (string, int64, error)`
- `Cleanup()`

### internal/mcp/auth/figma.go

**FigmaAuthStrategy Struct:**
- `NewFigmaAuthStrategy(config map[string]interface{}) (*FigmaAuthStrategy, error)`

**Methods:**
- `GetInitialToken(context.Context) (string, int64, error)`
- `RefreshToken(context.Context) (string, int64, error)`
- `Cleanup()`

---

## OAUTH 2.0 IMPLEMENTATION

### internal/oauth/service/oauth.go

**OAuthService Struct:**
- `NewOAuthService(db *gorm.DB) *OAuthService`

**UserTokenResolver Type:**
- Function type for resolving user tokens

**BasicAuthCredentials Struct:**
- Basic auth credentials

**Methods:**
- OAuth service methods (authorize, token, refresh, revoke, introspect)

### internal/oauth/service/client.go

**OAuthClientService Struct:**
- `NewOAuthClientService(db *gorm.DB, oauth *OAuthService) *OAuthClientService`

**Methods:**
- OAuth client management methods

### internal/oauth/service/metadata_fetcher.go

**ClientMetadataFetcher Struct:**
- `NewClientMetadataFetcher() *ClientMetadataFetcher`

**Methods:**
- Metadata fetching methods

### internal/oauth/controller/oauth.go

**OAuthController Struct:**
- `NewOAuthController(oauthService *service.OAuthService, clientService *service.OAuthClientService, userValidator UserTokenValidator) *OAuthController`

**UserTokenValidator Interface:**
- User token validation interface

**Methods:**
- OAuth endpoint handlers

### internal/oauth/controller/client.go

**OAuthClientController Struct:**
- `NewOAuthClientController(clientService *service.OAuthClientService) *OAuthClientController`

**Methods:**
- OAuth client endpoint handlers

### internal/oauth/controller/metadata.go

**OAuthMetadataController Struct:**
- `NewOAuthMetadataController(oauthService *service.OAuthService) *OAuthMetadataController`

**Methods:**
- OAuth metadata endpoint handlers

### internal/oauth/router.go

**Functions:**
- `RegisterRoutes(r chi.Router, deps RouterDependencies)` - Register OAuth routes

---

## ADMIN API

### internal/admin/controller.go

**Controller Struct:**
- `NewController(ipWhitelistService *security.IpWhitelistService) *Controller`

**Methods:**
- Admin request routing and handling

### internal/admin/handlers/user.go

**UserHandler Struct:**
- `NewUserHandler(db *gorm.DB, sessionStore *core.SessionStore, socketNotifier core.SocketNotifier, serverManager userRuntimeManager) *UserHandler`

**Methods:**
- `HandleCreateUser(data map[string]any) (interface{}, error)`
- `HandleGetUsers(data map[string]any) (interface{}, error)`
- `HandleUpdateUser(data map[string]any) (interface{}, error)`
- `HandleDeleteUser(data map[string]any) (interface{}, error)`
- `HandleDisableUser(data map[string]any) (interface{}, error)`
- `HandleUpdateUserPermissions(data map[string]any) (interface{}, error)`
- `HandleCountUsers(data map[string]any) (interface{}, error)`
- `HandleGetOwner(data map[string]any) (interface{}, error)`
- `HandleDeleteUsersByProxy(data map[string]any) (interface{}, error)`

### internal/admin/handlers/server.go

**ServerHandler Struct:**
- `NewServerHandler(db *gorm.DB, serverManager serverRuntimeManager, sessionStore *core.SessionStore, socketNotifier core.SocketNotifier) *ServerHandler`

**Methods:**
- `HandleCreateServer(data map[string]any) (interface{}, error)`
- `HandleGetServers(data map[string]any) (interface{}, error)`
- `HandleUpdateServer(data map[string]any) (interface{}, error)`
- `HandleDeleteServer(data map[string]any) (interface{}, error)`
- `HandleStartServer(data map[string]any) (interface{}, error)`
- `HandleStopServer(data map[string]any) (interface{}, error)`
- `HandleUpdateServerCapabilities(data map[string]any) (interface{}, error)`
- `HandleUpdateServerLaunchCmd(data map[string]any) (interface{}, error)`
- `HandleConnectAllServers(data map[string]any) (interface{}, error)`
- `HandleCountServers(data map[string]any) (interface{}, error)`
- `HandleGetAvailableServersCapabilities(data map[string]any) (interface{}, error)`
- `HandleGetUserAvailableServersCapabilities(data map[string]any) (interface{}, error)`
- `HandleGetServersStatus(data map[string]any) (interface{}, error)`
- `HandleGetServersCapabilities(data map[string]any) (interface{}, error)`
- `HandleDeleteServersByProxy(data map[string]any) (interface{}, error)`

### internal/admin/handlers/proxy.go

**ProxyHandler Struct:**
- `NewProxyHandler(db *gorm.DB, sessionStore *core.SessionStore, serverManager proxyRuntimeManager, socketNotifier core.SocketNotifier) *ProxyHandler`

**Methods:**
- `HandleGetProxy(data map[string]any) (interface{}, error)`
- `HandleCreateProxy(data map[string]any) (interface{}, error)`
- `HandleUpdateProxy(data map[string]any) (interface{}, error)`
- `HandleDeleteProxy(data map[string]any) (interface{}, error)`
- `HandleStopProxy(data map[string]any) (interface{}, error)`

### internal/admin/handlers/ipwhitelist.go

**IpWhitelistHandler Struct:**
- `NewIpWhitelistHandler(db *gorm.DB, svc *security.IpWhitelistService) *IpWhitelistHandler`

**Methods:**
- `HandleUpdateIPWhitelist(data map[string]any) (interface{}, error)`
- `HandleGetIPWhitelist(data map[string]any) (interface{}, error)`
- `HandleDeleteIPWhitelist(data map[string]any) (interface{}, error)`
- `HandleAddIPWhitelist(data map[string]any) (interface{}, error)`
- `HandleSpecialIPWhitelistOp(data map[string]any) (interface{}, error)`

### internal/admin/handlers/log.go

**LogHandler Struct:**
- `NewLogHandler(db *gorm.DB) *LogHandler`

**Methods:**
- `HandleGetLogs(data map[string]any) (interface{}, error)`
- `HandleSetLogWebhookURL(data map[string]any) (interface{}, error)`

### internal/admin/handlers/backup.go

**BackupHandler Struct:**
- `NewBackupHandler(db *gorm.DB, ipWhitelistService *security.IpWhitelistService) *BackupHandler`

**Methods:**
- `HandleBackupDatabase(data map[string]any) (interface{}, error)`
- `HandleRestoreDatabase(data map[string]any) (interface{}, error)`

### internal/admin/handlers/cloudflared.go

**CloudflaredHandler Struct:**
- `NewCloudflaredHandler(db *gorm.DB) *CloudflaredHandler`

**Methods:**
- `HandleUpdateCloudflaredConfig(data map[string]any) (interface{}, error)`
- `HandleGetCloudflaredConfigs(data map[string]any) (interface{}, error)`
- `HandleDeleteCloudflaredConfig(data map[string]any) (interface{}, error)`
- `HandleRestartCloudflared(data map[string]any) (interface{}, error)`
- `HandleStopCloudflared(data map[string]any) (interface{}, error)`

### internal/admin/handlers/skills.go

**SkillsHandler Struct:**
- `NewSkillsHandler(svc *service.SkillsService) *SkillsHandler`

**Methods:**
- `HandleListSkills(data map[string]any) (interface{}, error)`
- `HandleUploadSkill(data map[string]any) (interface{}, error)`
- `HandleDeleteSkill(data map[string]any) (interface{}, error)`
- `HandleDeleteServerSkills(data map[string]any) (interface{}, error)`

### internal/admin/handlers/query.go

**QueryHandler Struct:**
- `NewQueryHandler(db *gorm.DB) *QueryHandler`

**Methods:**
- Query execution methods

---

## USER API

### internal/user/controller.go

**Controller Struct:**
- `NewController() *Controller`

**Methods:**
- User API routing

### internal/user/handler.go

**RequestHandler Struct:**
- `NewRequestHandler(db *gorm.DB) *RequestHandler`

**Methods:**
- User request handling

### internal/user/auth.go

**UserAuthMiddleware Struct:**
- `NewUserAuthMiddleware(tokenValidator TokenValidator) *UserAuthMiddleware`

**TokenValidator Interface:**
- Token validation interface

**Methods:**
- User authentication middleware

### internal/user/types.go

**UserActionType Type:**
- User action type

**UserRequest Struct:**
- User API request

**UserResponse Struct:**
- User API response

**UserRespErr Struct:**
- User API error response

**UserError Struct:**
- User error type

---

## SOCKET.IO

### internal/socket/service.go

**SocketService Struct:**
- `NewSocketService(validator tokenValidator, userRepo *repository.UserRepository) *SocketService`

**Methods:**
- Socket service methods

### internal/socket/notifier.go

**SocketNotifier Struct:**
- `GetSocketNotifier() *SocketNotifier`

**Methods:**
- `AskUserConfirm(ctx context.Context, userID string, userAgent string, ip string, toolName string, description string, params string, timeout time.Duration) (bool, error)`
- `NotifyUserPermissionChanged(userID string)`
- `NotifyUserPermissionChangedByServer(serverID string)`
- `NotifyUserDisabled(userID string, reason string) bool`
- `NotifyOnlineSessions(userID string) bool`
- `UpdateServerInfo()`

### internal/socket/types.go

**SocketErrorCode Type:**
- Socket error code

**SocketActionType Type:**
- Socket action type

**UserConnection Struct:**
- User connection information

**OnlineSessionData Struct:**
- Online session data

**ClientInfo Struct:**
- Client information

**NotificationData Struct:**
- Notification data

**SocketData Struct:**
- Socket data

**SocketRequest[T] Struct:**
- Generic socket request

**SocketResponseError Struct:**
- Socket response error

**SocketResponse[T] Struct:**
- Generic socket response

**ApprovalRequestPayload Struct:**
- Approval request payload

**ApprovalResponsePayload Struct:**
- Approval response payload

**Functions:**
- `ActionToEventName(action SocketActionType) string` - Convert action to event name

---

## SERVICES

### internal/service/cloudflared.go

**CloudflaredService Struct:**
- `NewCloudflaredService() *CloudflaredService`

**TunnelCredentials Struct:**
- Tunnel credentials

**RestartResult Struct:**
- Restart result

**TunnelConfigInfo Struct:**
- Tunnel configuration info

**TunnelCreateResponse Struct:**
- Tunnel creation response

**StopResult Struct:**
- Stop result

**Methods:**
- Cloudflared service methods

### internal/service/cloudflared_docker.go

**CloudflaredDockerService Struct:**
- `NewCloudflaredDockerService() *CloudflaredDockerService`

**ContainerStatus Type:**
- Container status type

**Methods:**
- Docker-based cloudflared service methods

### internal/service/skills.go

**SkillsService Struct:**
- `NewSkillsService() *SkillsService`

**SkillInfo Struct:**
- Skill information

**Methods:**
- Skills service methods

---

## LOGGING & UTILITIES

### internal/log/session.go

**SessionLogger Struct:**
- `NewSessionLogger(userID string, sessionID string, tokenMask string, ip string, userAgent string) *SessionLogger`

**SessionClientRequestLog Struct:**
- Client request log

**SessionReverseRequestLog Struct:**
- Reverse request log

**SessionServerRequestLog Struct:**
- Server request log

**Methods:**
- `LogClientRequest(ctx context.Context, entry map[string]any) error`
- `LogReverseRequest(ctx context.Context, entry map[string]any) error`
- `LogServerLifecycle(ctx context.Context, entry map[string]any) error`
- `LogError(ctx context.Context, entry map[string]any) error`
- `LogSessionLifecycle(action int, errMsg string)`
- `IP() string`

### internal/log/server.go

**ServerLogger Struct:**
- `NewServerLogger(serverID string) *ServerLogger`

**Methods:**
- Server logging methods

### internal/log/service.go

**LogService Struct (Singleton):**
- `GetLogService() *LogService`

**Methods:**
- Log service methods

### internal/log/sync.go

**LogSyncService Struct (Singleton):**
- `GetLogSyncService() *LogSyncService`

**Methods:**
- Log synchronization methods

### internal/logger/logger.go

**Functions:**
- `CreateLogger(module string, fields ...map[string]interface{}) zerolog.Logger` - Create zerolog logger

### internal/utils/auth.go

**Functions:**
- `GenerateSessionID() string` - Generate session ID
- `OAuthProviderFromAuthType(authType int) string` - Get OAuth provider from auth type

### internal/utils/truncate.go

**Functions:**
- `TruncateResponseResult(data any, maxLength int) string` - Truncate response result

---

## CONFIGURATION

### internal/config/config.go

**AppInfo Struct:**
- Application information (Name, Version)

**Functions:**
- `Env(key string, defaultVal ...string) string` - Get environment variable

### internal/config/kimbapAuth.go

**KIMBAP_AUTH_CONFIG Variable:**
- Kimbap authentication configuration

### internal/config/cloudflared.go

**CLOUDFLARED_CONFIG Variable:**
- Cloudflared configuration

### internal/config/skills.go

**SKILLS_CONFIG Variable:**
- Skills configuration

### internal/config/reverseRequest.go

**REVERSE_REQUEST_TIMEOUTS Variable:**
- Reverse request timeout configuration

**ReverseRequestTimeoutError Struct:**
- Timeout error

**Functions:**
- `GetReverseRequestTimeout(requestType string) int` - Get reverse request timeout

---

## DATABASE

### internal/database/database.go

**Functions:**
- `Initialize(databaseURL string) error` - Initialize database connection
- `Close() error` - Close database connection
- `SQLDB() (*sql.DB, error)` - Get underlying SQL database

---

## MCP CONTROLLER

### internal/mcp/controller/mcp.go

**MCPController Struct:**
- `NewMCPController() *MCPController`

**Methods:**
- MCP request handling

### internal/mcp/router.go

**Functions:**
- MCP routing setup

---

## ENTRY POINT

### cmd/server/main.go

**Main Function:**
- Application entry point
- HTTP server setup
- Route registration
- Graceful shutdown

**Adapters:**
- Various adapter types for dependency injection

---

## SUMMARY STATISTICS

| Category | Count |
|----------|-------|
| Total Go Files | 108 |
| Exported Methods | 586+ |
| Structs | 150+ |
| Interfaces | 25+ |
| Type Aliases | 10+ |
| Constants | 200+ |
| Database Models | 11 |
| Repository Types | 7 |
| Middleware Types | 4 |
| Service Types | 3 |
| Handler Types | 9 |
| OAuth Providers | 7 |
| Auth Strategies | 5 |

---

## KEY ARCHITECTURAL PATTERNS

1. **Singleton Pattern**: ServerManager, SessionStore, LogService, LogSyncService, SocketNotifier
2. **Repository Pattern**: Data access layer with GORM
3. **Middleware Pattern**: HTTP middleware chain for auth, rate limiting, IP whitelist
4. **Factory Pattern**: AuthStrategyFactory, DownstreamTransportFactory
5. **Adapter Pattern**: Various adapters for dependency injection
6. **Interface-based Design**: Extensive use of interfaces for loose coupling
7. **Context-based Cancellation**: Context usage throughout for cancellation and timeouts
8. **Event Sourcing**: PersistentEventStore for session event persistence
9. **Session Management**: ClientSession and ProxySession for managing connections
10. **OAuth 2.0 Support**: Full OAuth 2.0 implementation with multiple providers

---

## CROSS-REFERENCE GUIDE FOR TS MIGRATION

When comparing with kimbap-core (TypeScript), use this mapping:

| Go Module | TypeScript Equivalent |
|-----------|----------------------|
| internal/mcp/core | src/mcp/core |
| internal/mcp/auth | src/mcp/auth |
| internal/mcp/oauth | src/mcp/oauth |
| internal/oauth | src/oauth |
| internal/admin | src/admin |
| internal/user | src/user |
| internal/middleware | src/middleware |
| internal/security | src/security |
| internal/repository | src/repository |
| internal/service | src/service |
| internal/socket | src/socket |
| internal/log | src/log |
| internal/types | src/types |
| internal/config | src/config |
| cmd/server | src/index.ts |


# Kimbap Core Admin API Protocol Documentation

## Overview

This document describes the complete protocol specification for Kimbap Core Admin API. All admin operations are provided through a unified `/admin` endpoint using an action-based request routing mechanism.

## Basic Information

- **Endpoint**: `POST /admin`
- **Authentication**: Bearer Token (Kimbap access token; opaque bearer token)
- **Content Type**: `application/json`
- **Character Encoding**: UTF-8

## Unified Request Format

All admin requests use a unified `AdminRequest` structure:

```go
type AdminRequest struct {
    Action int         `json:"action"`
    Data   interface{} `json:"data"`
}
```

*Admin operation type constants - uses numeric values for performance*

```go
const (
    // User operations (1000-1999)
    AdminActionDisableUser                         = 1001  // Disable access for specified user
    AdminActionUpdateUserPermissions               = 1002  // Update user permissions
    AdminActionCreateUser                          = 1010  // Create user
    AdminActionGetUsers                            = 1011  // Query user list
    AdminActionUpdateUser                          = 1012  // Update user
    AdminActionDeleteUser                          = 1013  // Delete user
    AdminActionDeleteUsersByProxy                  = 1014  // Batch delete users by proxy
    AdminActionCountUsers                          = 1015  // Count users
    AdminActionGetOwner                            = 1016  // Get Owner information

    // Server operations (2000-2999)
    AdminActionStartServer                         = 2001  // Start specified server
    AdminActionStopServer                          = 2002  // Stop specified server
    AdminActionUpdateServerCapabilities            = 2003  // Update server capability configuration
    AdminActionUpdateServerLaunchCmd               = 2004  // Update launch command
    AdminActionConnectAllServers                   = 2005  // Connect all servers
    AdminActionCreateServer                        = 2010  // Create server
    AdminActionGetServers                          = 2011  // Query server list
    AdminActionUpdateServer                        = 2012  // Update server
    AdminActionDeleteServer                        = 2013  // Delete server
    AdminActionDeleteServersByProxy                = 2014  // Batch delete servers by proxy
    AdminActionCountServers                        = 2015  // Count servers

    // Query operations (3000-3999)
    AdminActionGetAvailableServersCapabilities     = 3002  // Get all server capability configurations
    AdminActionGetUserAvailableServersCapabilities = 3003  // Get user accessible server capability configurations
    AdminActionGetServersStatus                    = 3004  // Get all server status
    AdminActionGetServersCapabilities              = 3005  // Get specified server capability configuration

    // IP whitelist operations (4000-4999)
    AdminActionUpdateIPWhitelist                   = 4001  // Replace mode: delete all existing IPs, save new IP list
    AdminActionGetIPWhitelist                      = 4002  // Query IP whitelist
    AdminActionDeleteIPWhitelist                   = 4003  // Delete specified IP whitelist
    AdminActionAddIPWhitelist                      = 4004  // Append mode: add IPs to whitelist
    AdminActionSpecialIPWhitelistOp                = 4005  // IP filter switch: allow-all/deny-all

    // Proxy operations (5000-5099)
    AdminActionGetProxy                            = 5001  // Query proxy information
    AdminActionCreateProxy                         = 5002  // Create proxy
    AdminActionUpdateProxy                         = 5003  // Update proxy
    AdminActionDeleteProxy                         = 5004  // Delete proxy
    AdminActionStopProxy                           = 5005  // Stop all proxy servers

    // Backup and restore (6000-6099)
    AdminActionBackupDatabase                      = 6001  // Full database backup
    AdminActionRestoreDatabase                     = 6002  // Full database restore

    // Log operations (7000-7099)
    AdminActionSetLogWebhookURL                    = 7001  // Set log sync webhook URL
    AdminActionGetLogs                             = 7002  // Get log records

    // Cloudflared operations (8000-8099)
    AdminActionUpdateCloudflaredConfig             = 8001  // Update cloudflared configuration
    AdminActionGetCloudflaredConfigs               = 8002  // Query cloudflared configuration list
    AdminActionDeleteCloudflaredConfig             = 8003  // Delete cloudflared configuration
    AdminActionRestartCloudflared                  = 8004  // Restart cloudflared
    AdminActionStopCloudflared                     = 8005  // Stop cloudflared

    // Skills operations (10040-10043)
    AdminActionListSkills                          = 10040 // List skills for a server
    AdminActionUploadSkill                         = 10041 // Upload a skill to a server
    AdminActionDeleteSkill                         = 10042 // Delete a skill
    AdminActionDeleteServerSkills                  = 10043 // Delete all skills for a server

    // Policy operations (9100-9199)
    AdminActionCreatePolicySet                     = 9101  // Create policy set
    AdminActionGetPolicySets                       = 9102  // Get policy sets
    AdminActionUpdatePolicySet                     = 9103  // Update policy set
    AdminActionDeletePolicySet                     = 9104  // Delete policy set
    AdminActionGetEffectivePolicy                  = 9105  // Get effective policy

    // Approval operations (9200-9299)
    AdminActionListApprovalRequests                = 9201  // List approval requests
    AdminActionGetApprovalRequest                  = 9202  // Get approval request
    AdminActionDecideApprovalRequest               = 9203  // Decide approval request
    AdminActionCountPendingApprovals               = 9204  // Count pending approvals
)
```

### Request Examples

**curl example:**
```bash
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_KIMBAP_ACCESS_TOKEN" \
  -d '{
    "action": 1011,
    "data": {
      "proxyId": 0
    }
  }'
```

**HTTP client example:**
```javascript
const response = await fetch('http://localhost:3002/admin', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    action: 1011,  // GET_USERS
    data: {
      proxyId: 0
    }
  })
});

const result = await response.json();
if (!result.success) {
  console.error('Operation failed:', result.error);
}
```

## Unified Response Format

All admin requests return a unified `AdminResponse` structure:

```go
type AdminResponse struct {
    Success bool                `json:"success"`
    Data    interface{}         `json:"data,omitempty"`
    Error   *AdminResponseError `json:"error,omitempty"`
}

type AdminResponseError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}
```

**Success response example:**
```json
{
  "success": true,
  "data": {
    "users": [...]
  }
}
```

**Error response example:**
```json
{
  "success": false,
  "error": {
    "code": 2001,
    "message": "User user123 not found"
  }
}
```

## Data Structures Overview

### Enum Reference

**UserStatus**
- `0` Disabled
- `1` Enabled
- `2` Pending
- `3` Suspended

**UserRole**
- `1` Owner
- `2` Admin
- `3` User
- `4` Guest

**ServerCategory**
- `1` Template
- `2` CustomRemote
- `3` RestApi
- `4` Skills

**ServerAuthType**
- `1` ApiKey
- `2` GoogleAuth
- `3` NotionAuth
- `4` FigmaAuth
- `5` GoogleCalendarAuth
- `6` GithubAuth
- `7` ZendeskAuth
- `8` CanvasAuth

> **Note**: Legacy versions included `StripeAuth = 7`, `ZendeskAuth = 8`, `CanvasAuth = 9`. In this Go implementation, `StripeAuth` was removed and the values are `ZendeskAuth = 7`, `CanvasAuth = 8`.

**ServerStatus**
- `0` Online
- `1` Offline
- `2` Connecting
- `3` Error
- `4` Sleeping (lazy start)

**DangerLevel**
- `0` Silent
- `1` Notification
- `2` Approval

### Permissions (used by UPDATE_USER_PERMISSIONS / CREATE_USER / UPDATE_USER)

```go
// Permissions is a map of serverId to server permission config
type Permissions map[string]ServerPermission

type ServerPermission struct {
    Enabled        bool                       `json:"enabled"`
    ServerName     string                     `json:"serverName,omitempty"`
    AllowUserInput bool                       `json:"allowUserInput,omitempty"`
    AuthType       int                        `json:"authType,omitempty"`
    Category       int                        `json:"category,omitempty"`
    ConfigTemplate string                     `json:"configTemplate,omitempty"`
    Configured     bool                       `json:"configured,omitempty"`
    Status         int                        `json:"status,omitempty"`
    Tools          map[string]ToolPermission  `json:"tools,omitempty"`
    Resources      map[string]ItemPermission  `json:"resources,omitempty"`
    Prompts        map[string]ItemPermission  `json:"prompts,omitempty"`
}

type ToolPermission struct {
    Enabled     bool   `json:"enabled"`
    Description string `json:"description,omitempty"`
    DangerLevel int    `json:"dangerLevel,omitempty"` // 0=Silent, 1=Notification, 2=Approval
}

type ItemPermission struct {
    Enabled     bool   `json:"enabled"`
    Description string `json:"description,omitempty"`
}
```

**Minimum valid structure**:
- `Enabled` must be boolean
- `Tools/Resources/Prompts` must be maps when present
- Invalid structure will fail authentication (`INVALID_PERMISSIONS`)

### ServerConfigCapabilities (used by UPDATE_SERVER_CAPABILITIES)

```go
type ServerConfigCapabilities struct {
    Tools     map[string]ToolCapability `json:"tools"`
    Resources map[string]ItemCapability `json:"resources"`
    Prompts   map[string]ItemCapability `json:"prompts"`
}

type ToolCapability struct {
    Enabled     bool   `json:"enabled"`
    Description string `json:"description,omitempty"`
    DangerLevel int    `json:"dangerLevel,omitempty"` // 0=Silent, 1=Notification, 2=Approval
}

type ItemCapability struct {
    Enabled     bool   `json:"enabled"`
    Description string `json:"description,omitempty"`
}
```

### EncryptedData (common encrypted field format)

`encryptedToken` / `launchConfig` are JSON strings with this structure:
```go
type EncryptedData struct {
    Data string `json:"data"` // Base64
    IV   string `json:"iv"`   // Base64
    Salt string `json:"salt"` // Base64
    Tag  string `json:"tag"`  // Base64
}
```

### Decrypted launchConfig Structure (used internally when connecting)

**stdio**
```json
{ "type": "stdio", "command": "./mcp-server", "args": ["--mode", "stdio"], "env": { "A": "B" }, "cwd": "/path" }
```

**http**
```json
{ "type": "http", "url": "https://example.com/mcp", "headers": { "Authorization": "Bearer xxx" } }
```

**sse**
```json
{ "type": "sse", "url": "https://example.com/sse" }
```

## Permission Description

Admin API supports two role permissions:

- **Owner + Admin**: Most operations allow Owner and Admin roles to execute
- **Owner only**: Some sensitive operations only allow Owner role to execute (specially marked in API list)

Permission verification is performed in AdminController. Requests that do not meet permission requirements will return `FORBIDDEN (1003)` error.

**publicAccess and permissions merge logic**:
- If `permissions` does not include a `serverId`, availability falls back to `server.publicAccess` (true allows access)
- If `permissions` includes a `serverId`, `permissions[serverId].enabled` takes precedence over `publicAccess`
- Final tool/resource/prompt visibility is the merge of server capabilities and admin permissions; you can only override `enabled` for existing capability items

---

## API List

### User Operations (1000-1999)

#### 1001 DISABLE_USER

**Permission**: Owner + Admin
**Function**: Disable access for specified user, disconnect all active sessions for that user

**Request Parameters** (data):
- `targetId` (string, required): User ID

**Return Result** (data):
```json
null
```

**Function Description**:
- Update user status to `Disabled`
- Disconnect all active MCP sessions for that user
- User can no longer establish new connections

---

#### 1002 UPDATE_USER_PERMISSIONS

**Permission**: Owner + Admin
**Function**: Update server permission configuration for specified user

**Request Parameters** (data):
- `targetId` (string, required): User ID
- `permissions` (string or object, required): Permission configuration (Permissions object or its JSON string)

**Return Result** (data):
```json
null
```

**Function Description**:
- **Replace semantics**: `permissions` replaces the user's existing permissions (not a patch)
- If only some serverIds are provided, others are removed and fall back to `publicAccess`
- Update permissions field in user database
- If user has active sessions, push permission change notifications in real-time:
  - Send `tools/list_changed` when tools change
  - Send `resources/list_changed` when resources change
  - Send `prompts/list_changed` when prompts change

**Minimal example** (disable a single server):
```json
{
  "targetId": "user123",
  "permissions": {
    "filesystem": {
      "enabled": false,
      "tools": {},
      "resources": {},
      "prompts": {}
    }
  }
}
```

---

#### 1010 CREATE_USER

**Permission**: **Owner only** No verification when creating owner for the first time, database is empty at this time
**Function**: Create new user

**Request Parameters** (data):
- `userId` (string, required): User ID (unique identifier)
- `status` (number, optional): User status, defaults to `UserStatus.Enabled (1)`
- `role` (number, optional): User role, defaults to `UserRole.User (3)`
- `permissions` (string or object, optional): Permission configuration, defaults to `{}`
- `expiresAt` (number, optional): Expiration time (Unix timestamp, seconds), defaults to `0` (never expires)
- `createdAt` (number, optional): Creation time (Unix timestamp, seconds), defaults to current time
- `updatedAt` (number, optional): Update time (Unix timestamp, seconds), defaults to current time
- `ratelimit` (number, optional): Rate limit, defaults to `100`
- `name` (string, optional): User name, defaults to empty string
- `encryptedToken` (string, required): Encrypted token
- `proxyId` (number, optional): Associated proxy ID, defaults to `0`
- `notes` (string, optional): Notes, defaults to `null`

**Return Result** (data):
```json
{
  "user": {
    "userId": "user123",
    "status": 1,
    "role": 3,
    "permissions": "{}",
    ...
  }
}
```

**Important Notes**:
- `encryptedToken` must be an `EncryptedData` JSON string (see Data Structures Overview)
- For non-Owner creation, the admin token is used to decrypt `encryptedToken`, and `userId` must equal the first 32 chars of `SHA-256(token)`
- Owner can be created only once, and must be the first user (empty database)

---

#### 1011 GET_USERS

**Permission**: Owner + Admin
**Function**: Query user list, supports multiple filter conditions

**Request Parameters** (data):
- `userId` (string, optional): Exact query for specified user ID
- `proxyId` (number, optional): Filter by proxyId
- `role` (number, optional): Filter by role
- `excludeRole` (number, optional): Exclude specified role

**Return Result** (data):
```json
{
  "users": [
    {
      "userId": "user123",
      "status": 1,
      "role": 3,
      "permissions": "{}",
      "serverApiKeys": "[]",
      ...
    }
  ]
}
```

**Function Description**:
- If `userId` is provided, returns single user (in array form) or empty array
- Other filter conditions can be combined
- Returns all users if no filter conditions provided

---

#### 1012 UPDATE_USER

**Permission**: Owner + Admin
**Function**: Update user information

**Request Parameters** (data):
- `userId` (string, required): User ID
- `name` (string, optional): User name
- `notes` (string, optional): Notes
- `permissions` (string or object, optional): Permission configuration
- `status` (number, optional): User status
- `encryptedToken` (string, optional): Encrypted user access token

**Return Result** (data):
```json
{
  "user": {
    "userId": "user123",
    ...
  }
}
```

**Function Description**:
- If `permissions` is updated, will push to user's active sessions in real-time
- If `status` changes to `Disabled`, will disable user first (disconnect sessions)

---

#### 1013 DELETE_USER

**Permission**: Owner + Admin
**Function**: Delete specified user

**Request Parameters** (data):
- `userId` (string, required): User ID

**Return Result** (data):
```json
{
  "message": "User deleted successfully"
}
```

**Function Description**:
- Disable user before deletion (disconnect all sessions)
- Permanently delete user record from database

---

#### 1014 DELETE_USERS_BY_PROXY

**Permission**: Owner + Admin
**Function**: Batch delete users by proxyId

**Request Parameters** (data):
- `proxyId` (number, required): Proxy ID

**Return Result** (data):
```json
{
  "deletedCount": 10
}
```

**Function Description**:
- Disable all matching users before deletion (disconnect sessions)
- Returns actual number of deleted users

---

#### 1015 COUNT_USERS

**Permission**: Owner + Admin
**Function**: Count users

**Request Parameters** (data):
- `excludeRole` (number, optional): Exclude specified role

**Return Result** (data):
```json
{
  "count": 50
}
```

---

#### 1016 GET_OWNER

**Permission**: Public (no authentication required)
**Function**: Get complete information of system Owner user

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "owner": {
    "userId": "owner123",
    "status": 1,
    "role": 1,
    "permissions": "{}",
    "serverApiKeys": "[]",
    "expiresAt": 0,
    "createdAt": 1729431234,
    "updatedAt": 1729431234,
    "ratelimit": 100,
    "name": "System Owner",
    "encryptedToken": "...",
    "proxyId": 0,
    "notes": null
  }
}
```

**Function Description**:
- Returns complete information of the unique Owner role user in the system
- If no Owner user exists in the system, returns error (code: 2001, USER_NOT_FOUND)
- This endpoint requires no authentication and is publicly accessible
- Returns all user fields, including sensitive information

---

### Server Operations (2000-2999)

#### 2001 START_SERVER

**Permission**: **Owner only**
**Function**: Start specified MCP server

**Request Parameters** (data):
- `targetId` (string, required): Server ID

**Return Result** (data):
```json
null
```

**Function Description**:
- Set server's `Enabled` field to `true`
- Start MCP server process and establish connection
- Notify all active user sessions using this server of capability changes

---

#### 2002 STOP_SERVER

**Permission**: Owner + Admin
**Function**: Stop specified MCP server

**Request Parameters** (data):
- `targetId` (string, required): Server ID

**Return Result** (data):
```json
null
```

**Function Description**:
- Disconnect MCP server connection
- Set server's `Enabled` field to `false`
- Notify all active user sessions using this server of capability changes

---

#### 2003 UPDATE_SERVER_CAPABILITIES

**Permission**: Owner + Admin
**Function**: Update server capability configuration (tools/resources/prompts)

**Request Parameters** (data):
- `targetId` (string, required): Server ID
- `capabilities` (string or object, required): Capability configuration (ServerConfigCapabilities object or its JSON string)

**Return Result** (data):
```json
null
```

**Function Description**:
- Update server capability configuration in database
- If server is running, reload configuration and notify related user sessions

---

#### 2004 UPDATE_SERVER_LAUNCH_CMD

**Permission**: **Owner only**
**Function**: Update server launch command configuration

**Request Parameters** (data):
- `targetId` (string, required): Server ID
- `launchConfig` (string, required): Launch configuration (contains command/args/env, etc.), string encrypted with owner token

**Return Result** (data):
```json
null
```

**Function Description**:
- Update launchConfig in database
- If server is running, reconnect server (restart)
- Notify related user sessions of capability changes

---

#### 2005 CONNECT_ALL_SERVERS

**Permission**: **Owner only**
**Function**: Connect all enabled MCP servers

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "successServers": [
    {
      "serverId": "server1",
      "serverName": "Server 1",
      ...
    }
  ],
  "failedServers": [
    {
      "serverId": "server2",
      "serverName": "Server 2",
      ...
    }
  ]
}
```

**Function Description**:
- Attempt to connect all servers with `Enabled = true`
- Returns lists of successful and failed servers

---

#### 2010 CREATE_SERVER

**Permission**: **Owner only**
**Function**: Create new MCP server configuration

**Request Parameters** (data):
- `serverId` (string, required): Server ID (unique identifier)
- `serverName` (string, optional): Server name, defaults to empty string
- `enabled` (boolean, optional): Whether enabled, defaults to `true`
- `launchConfig` (string, **required**): Encrypted launch configuration JSON (EncryptedData); must be non-empty and not `{}`; encrypted with Owner token
- `capabilities` (string or object, optional): **ignored on create**; use `UPDATE_SERVER_CAPABILITIES (2003)` to set capabilities
- `createdAt` (number, optional): Creation time (Unix timestamp, seconds), defaults to current time
- `updatedAt` (number, optional): Update time (Unix timestamp, seconds), defaults to current time
- `allowUserInput` (boolean, optional): Whether to allow user input, defaults to `false`
- `proxyId` (number, optional): Associated proxy ID, defaults to `0`
- `toolTmplId` (string, optional): Tool template ID, defaults to `null`
- `authType` (number, required): Server authorization type, defaults to 1, API Key authentication, 2 Google OAuth authentication
- `configTemplate` (string, **required**): JSON config template string; must be non-empty and not `{}` (validated regardless of allowUserInput)
- `category` (number, required): Server category. 1: template server, 2: custom remote server, 3: RESTful API server
- `lazyStartEnabled` (boolean, optional): Enable lazy loading for this server. When true, server loads into memory but delays startup until first use, and auto-shuts down when idle. Defaults to `true`
- `publicAccess` (boolean, optional): Whether public access is enabled. If true, users without explicit permissions can still access this server. Defaults to `false`

**Return Result** (data):
```json
{
  "server": {
    "serverId": "server123",
    "serverName": "My Server",
    "enabled": true,
    ...
  }
}
```

**ConfigTemplate notes (by category)**
- **Template (1)**: must include `mcpJsonConf` and `credentials`, optional `oAuthConfig`
- **CustomRemote (2)**: must include base URL info (used to assemble launchConfig for user config)
- **RestApi (3)**: must include `apis[0].auth`; system injects `launchConfig.auth`

**OAuth Template note**:
- If `configTemplate.oAuthConfig.clientId` exists, the decrypted `launchConfig` must include `oauth` fields (`clientId`, `clientSecret`, `code`, `redirectUri`, etc.); system exchanges and persists tokens

**Input examples (by category)**:

**Template (1) - ApiKey (Postgres)**:
```json
{
  "action": 2010,
  "data": {
    "serverId": "postgres",
    "serverName": "Postgres",
    "category": 1,
    "authType": 1,
    "allowUserInput": true,
    "configTemplate": "{\"toolId\":\"5f2504e04f8911d39a0c0331222c312\",\"name\":\"Postgres\",\"description\":\"PostgreSQL MCP server running in Docker with full read-write access.\",\"credentials\":\"[{\\\"name\\\":\\\"POSTGRES URL\\\",\\\"description\\\":\\\"\\\",\\\"dataType\\\":1,\\\"key\\\":\\\"YOUR_POSTGRES_URL\\\"}]\",\"mcpJsonConf\":\"{\\\"command\\\":\\\"docker\\\",\\\"args\\\":[\\\"run\\\",\\\"-i\\\",\\\"--rm\\\",\\\"--pull=always\\\",\\\"-e\\\",\\\"POSTGRES_URL\\\",\\\"-e\\\",\\\"ACCESS_MODE\\\",\\\"ghcr.io/dunialabs/mcp-servers/postgres:latest\\\"],\\\"env\\\":{\\\"POSTGRES_URL\\\":\\\"YOUR_POSTGRES_URL\\\",\\\"ACCESS_MODE\\\":\\\"readwrite\\\"}}\",\"authType\":1,\"authConfig\":\"\",\"toolDefaultConfig\":\"{\\\"postgresListSchemas\\\":{\\\"enabled\\\":true,\\\"dangerLevel\\\":0},\\\"postgresListTables\\\":{\\\"enabled\\\":true,\\\"dangerLevel\\\":0},\\\"postgresDescribeTable\\\":{\\\"enabled\\\":true,\\\"dangerLevel\\\":0},\\\"postgresGetTableStats\\\":{\\\"enabled\\\":true,\\\"dangerLevel\\\":0},\\\"postgresExecuteQuery\\\":{\\\"enabled\\\":true,\\\"dangerLevel\\\":1},\\\"postgresExecuteWrite\\\":{\\\"enabled\\\":true,\\\"dangerLevel\\\":2},\\\"postgresExplainQuery\\\":{\\\"enabled\\\":true,\\\"dangerLevel\\\":2}}\"}",
    "launchConfig": "{\"data\":\"...\",\"iv\":\"...\",\"salt\":\"...\",\"tag\":\"...\"}"
  }
}
```

**Template (1) - OAuth (Notion)**:
```json
{
  "action": 2010,
  "data": {
    "serverId": "notion",
    "serverName": "Notion",
    "category": 1,
    "authType": 3,
    "allowUserInput": true,
    "configTemplate": "{\"toolId\":\"a7c3f8e9b2d64a5891f3c7b0e4d2a8f6\",\"name\":\"Notion\",\"description\":\"Model Context Protocol (MCP) server for Notion integration.\",\"credentials\":\"[...]...\"}",
    "launchConfig": "{\"data\":\"...\",\"iv\":\"...\",\"salt\":\"...\",\"tag\":\"...\"}"
  }
}
```

**CustomRemote (2)**:
```json
{
  "action": 2010,
  "data": {
    "serverId": "custom-remote",
    "serverName": "Custom Remote",
    "category": 2,
    "authType": 1,
    "allowUserInput": true,
    "configTemplate": "{\"url\":\"https://example.com/mcp\"}",
    "launchConfig": "{\"data\":\"...\",\"iv\":\"...\",\"salt\":\"...\",\"tag\":\"...\"}"
  }
}
```

**RestApi (3)**:
```json
{
  "action": 2010,
  "data": {
    "serverId": "rest-api",
    "serverName": "REST API",
    "category": 3,
    "authType": 1,
    "allowUserInput": true,
    "configTemplate": "{\"baseUrl\":\"https://api.example.com\",\"apis\":[{\"path\":\"/\",\"auth\":{}}]}",
    "launchConfig": "{\"data\":\"...\",\"iv\":\"...\",\"salt\":\"...\",\"tag\":\"...\"}"
  }
}
```

---

#### 2011 GET_SERVERS

**Permission**: Owner + Admin
**Function**: Query server list, supports multiple filter conditions

**Request Parameters** (data):
- `serverId` (string, optional): Exact query for specified server ID
- `proxyId` (number, optional): Filter by proxyId
- `enabled` (boolean, optional): Filter by enabled status

**Return Result** (data):
```json
{
  "servers": [
    {
      "serverId": "server123",
      "serverName": "My Server",
      "enabled": true,
      "launchConfig": "{}",
      "capabilities": "{}",
      ...
    }
  ]
}
```

---

#### 2012 UPDATE_SERVER

**Permission**: **Owner only**
**Function**: Update server configuration

**Request Parameters** (data):
- `serverId` (string, required): Server ID
- `serverName` (string, optional): Server name
- `launchConfig` (string or object, optional): Launch configuration encrypted with owner token. **Not updatable for Template with allowUserInput=true**
- `capabilities` (string or object, optional): Capability configuration. **Merge semantics; omitted fields are not removed**
- `enabled` (boolean, optional): Whether enabled
- `configTemplate` (string, optional): Only RestApi/CustomRemote can update
- `lazyStartEnabled` (boolean, optional): Enable lazy loading for this server. When true, server loads into memory but delays startup until first use, and auto-shuts down when idle
- `publicAccess` (boolean, optional): Public access flag

**Return Result** (data):
```json
{
  "server": {
    "serverId": "server123",
    ...
  }
}
```

**Function Description**:
- `allowUserInput` is immutable after creation
- `configTemplate` is only updatable for RestApi/CustomRemote; Template updates will error
- `capabilities` uses **merge semantics**; explicitly set `enabled=false` to disable
- If server is running and `capabilities` or `launchConfig` is updated, will trigger reload or restart
- If `enabled` changes to `false`, will stop server

**Input examples (by category)**:

**Template (1) - update name/enable only**:
```json
{
  "action": 2012,
  "data": {
    "serverId": "notion",
    "serverName": "Notion Personal",
    "enabled": true
  }
}
```

**CustomRemote (2) - update configTemplate and launchConfig**:
```json
{
  "action": 2012,
  "data": {
    "serverId": "custom-remote",
    "configTemplate": "{\"url\":\"https://example.com/mcp\"}",
    "launchConfig": "{\"data\":\"...\",\"iv\":\"...\",\"salt\":\"...\",\"tag\":\"...\"}"
  }
}
```

**RestApi (3) - update capabilities (merge)**:
```json
{
  "action": 2012,
  "data": {
    "serverId": "rest-api",
    "capabilities": {
      "tools": {
        "write_record": { "enabled": false, "description": "Disable write" }
      },
      "resources": {},
      "prompts": {}
    }
  }
}
```

---

#### 2013 DELETE_SERVER

**Permission**: Owner + Admin
**Function**: Delete specified server

**Request Parameters** (data):
- `serverId` (string, required): Server ID

**Return Result** (data):
```json
{
  "message": "Server deleted successfully"
}
```

**Function Description**:
- Remove server from ServerManager (stop connection)
- Permanently delete server record from database

---

#### 2014 DELETE_SERVERS_BY_PROXY

**Permission**: Owner + Admin
**Function**: Batch delete servers by proxyId

**Request Parameters** (data):
- `proxyId` (number, required): Proxy ID

**Return Result** (data):
```json
{
  "deletedCount": 5
}
```

**Function Description**:
- Stop connections for all matching servers
- Returns actual number of deleted servers

---

#### 2015 COUNT_SERVERS

**Permission**: Owner + Admin
**Function**: Count servers

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "count": 10
}
```

---

### Query Operations (3000-3999)

#### 3002 GET_AVAILABLE_SERVERS_CAPABILITIES

**Permission**: Owner + Admin
**Function**: Get capability configurations of all available servers

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "capabilities": {
    "server1": {
      "enabled": true,
      "tools": {
        "toolName": {
          "enabled": true,
          "description": "Tool description",
          "dangerLevel": 0
        }
      },
      "resources": {
        "resourceName": {
          "enabled": true,
          "description": "Resource description"
        }
      },
      "prompts": {
        "promptName": {
          "enabled": true,
          "description": "Prompt description"
        }
      }
    }
  }
}
```

**Function Description**:
- Returns capability configurations of all running servers

---

#### 3003 GET_USER_AVAILABLE_SERVERS_CAPABILITIES

**Permission**: Owner + Admin
**Function**: Get capability configurations of servers accessible to specified user

**Request Parameters** (data):
- `targetId` (string, required): User ID

**Return Result** (data):
```json
{
  "capabilities": {
    "server1": {
      "enabled": true,
      "tools": { ... },
      "resources": { ... },
      "prompts": { ... }
    }
  }
}
```

**Function Description**:
- Prioritizes getting capability configuration from user's active sessions
- If user has no active sessions, calculates capabilities based on user permission configuration
- The returned `enabled` field reflects user's permission for that server

---

#### 3004 GET_SERVERS_STATUS

**Permission**: Owner + Admin
**Function**: Get current status of all servers

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "serversStatus": {
    "server1": 0,
    "server2": 1,
    "server3": 2
  }
}
```

**ServerStatus Constants**:
- `0`: Online
- `1`: Offline
- `2`: Connecting
- `3`: Error
- `4`: Sleeping (lazy start)

---

#### 3005 GET_SERVERS_CAPABILITIES

**Permission**: Owner + Admin
**Function**: Get capability configuration of specified server

**Request Parameters** (data):
- `targetId` (string, required): Server ID

**Return Result** (data):
```json
{
  "capabilities": {
    "tools": {
      "toolName": {
        "enabled": true,
        "description": "Tool description",
        "dangerLevel": 0
      }
    },
    "resources": { ... },
    "prompts": { ... }
  }
}
```

**Function Description**:
- If server is running, returns real-time capability configuration
- If server is not running, returns configuration stored in database

---

### IP Whitelist Operations (4000-4999)

#### 4001 UPDATE_IP_WHITELIST

**Permission**: Owner + Admin
**Function**: Replace mode update IP whitelist (delete all existing IPs, save new IP list to database and load to memory)

**Request Parameters** (data):
- `whitelist` (array, required): IP address array (supports single IP or CIDR format)

**Return Result** (data):
```json
{
  "whitelist": ["192.168.1.0/24", "10.0.0.1"],
  "message": "IP whitelist updated successfully. 2 IPs loaded."
}
```

**Function Description**:
- Delete all existing IP records and insert new records in database transaction
- Automatically reload from database to memory, takes effect immediately
- Supported IP formats:
  - Single IP: `"192.168.1.100"`
  - CIDR: `"192.168.1.0/24"`
  - Special value: `"0.0.0.0/0"` means allow all IPs (disable filtering)

---

#### 4002 GET_IP_WHITELIST

**Permission**: Owner + Admin
**Function**: Query current IP whitelist

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "whitelist": [
    "192.168.1.0/24",
    "10.0.0.1"
  ],
  "count": 2
}
```

---

#### 4003 DELETE_IP_WHITELIST

**Permission**: Owner + Admin
**Function**: Delete specified IP whitelist records

**Request Parameters** (data):
- `ips` (array, required): Array of IP addresses to delete

**Return Result** (data):
```json
{
  "deletedCount": 2,
  "message": "2 IP(s) deleted from whitelist"
}
```

**Function Description**:
- Delete specified IPs from database
- Automatically reload to memory, takes effect immediately
- If specified IPs don't exist, returns `deletedCount: 0`

---

#### 4004 ADD_IP_WHITELIST

**Permission**: Owner + Admin
**Function**: Append mode add IPs to whitelist (without deleting existing IPs)

**Request Parameters** (data):
- `ips` (array, required): Array of IP addresses to add

**Return Result** (data):
```json
{
  "addedIds": [10, 11, 12],
  "addedCount": 3,
  "skippedCount": 1,
  "message": "3 IP(s) added to whitelist, 1 skipped (duplicates)"
}
```

**Function Description**:
- Validates IP format (invalid format returns `INVALID_IP_FORMAT (5102)` error)
- Automatically skips duplicate IPs that already exist
- Automatically reloads to memory, takes effect immediately

---

#### 4005 SPECIAL_IP_WHITELIST_OPERATION

**Permission**: Owner + Admin
**Function**: IP filter switch operation (allow-all disable filter / deny-all enable filter)

**Request Parameters** (data):
- `operation` (string, required): Operation type, optional values `"allow-all"` or `"deny-all"`

**Return Result** (data):
```json
null
```

**Function Description**:

**allow-all operation (disable IP filtering)**:
- Add `"0.0.0.0/0"` record to database (if already exists, don't add duplicate)
- Effect: Allow all IP access

**deny-all operation (enable IP filtering)**:
- Delete all `"0.0.0.0/0"` records from database
- Prerequisite: Database must have other IP configurations, otherwise returns error
- Effect: Enable strict IP whitelist filtering

**Usage Recommendations**:
1. First use ADD_IP_WHITELIST (4004) to add allowed IPs
2. Then use deny-all operation to enable IP filtering
3. Use allow-all operation when temporarily disabling filtering

---

### Proxy Operations (5000-5099)

#### 5001 GET_PROXY

**Function**: Query proxy information (system only supports single proxy)

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "proxy": {
    "id": 1,
    "name": "My MCP Server",
    "proxyKey": "xxx",
    "addtime": 1234567890,
    "startPort": 3002
  }
}
```

**Function Description**:
- If no proxy exists, returns `proxy: null`

---

#### 5002 CREATE_PROXY

**Function**: Create proxy (system only allows one proxy)

**Request Parameters** (data):
- `name` (string, required): Proxy name
- `proxyKey` (string, required): Proxy key

**Return Result** (data):
```json
{
  "proxy": {
    "id": 1,
    "name": "My MCP Server",
    "proxyKey": "xxx",
    "startPort": 3002,
    "addtime": 1234567890
  }
}
```

**Function Description**:
- System only allows one proxy to exist
- If proxy already exists, returns `PROXY_ALREADY_EXISTS (5002)` error
- `startPort` automatically reads environment variable `BACKEND_PORT`

---

#### 5003 UPDATE_PROXY

**Permission**: Owner + Admin
**Function**: Update proxy information

**Request Parameters** (data):
- `proxyId` (number, required): Proxy ID (the `proxy.id` returned by GET_PROXY/CREATE_PROXY)
- `name` (string, required): New Proxy name

**Return Result** (data):
```json
{
  "proxy": {
    "id": 1,
    "name": "Updated Name",
    ...
  }
}
```

---

#### 5004 DELETE_PROXY

**Permission**: **Owner only**
**Function**: Delete proxy (will clear all related data)

**Request Parameters** (data):
- `proxyId` (number, required): Proxy ID (the `proxy.id` returned by GET_PROXY/CREATE_PROXY)

**Return Result** (data):
```json
{
  "message": "Proxy deleted successfully"
}
```

**Function Description**:
- Delete proxy record
- Clear all users, servers, IP whitelist, logs
- Disconnect all active sessions
- Stop all MCP servers

---

#### 5005 STOP_PROXY

**Permission**: **Owner only**
**Function**: Stop proxy application (completely shut down application process)

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "message": "Proxy shutdown initiated successfully"
}
```

**Function Description**:
- Triggers complete application shutdown process (equivalent to SIGTERM/SIGINT signal)
- Stop HTTP/HTTPS server from accepting new connections
- Stop event cleanup service
- Close log sync service (flush remaining logs)
- Clean up all client sessions
- Close all downstream MCP server connections
- Call `os.Exit(0)` to exit application process

**Important Notes**:
- After executing this operation, the application will completely stop and requires manual service restart
- Response will be sent to client before application closes
- Recommend notifying all users and saving important data before execution

---

### Backup and Restore Operations (6000-6099)

#### 6001 BACKUP_DATABASE

**Permission**: Owner + Admin
**Function**: Full database backup

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "backup": {
    "version": "1.0",
    "timestamp": 1729431234,
    "tables": {
      "users": [ ... ],
      "servers": [ ... ],
      "proxies": [ ... ],
      "ipWhitelist": [ ... ]
    }
  },
  "stats": {
    "usersCount": 50,
    "serversCount": 10,
    "proxiesCount": 1,
    "ipWhitelistCount": 5
  }
}
```

**Function Description**:
- Export all user, server, proxy, IP whitelist data
- Returned backup data can be used for restore operations

---

#### 6002 RESTORE_DATABASE

**Permission**: Owner + Admin
**Function**: Full database restore

**Request Parameters** (data):
- `backup` (object, required): Backup data object (format returned by BACKUP_DATABASE)

**Return Result** (data):
```json
{
  "message": "Database restored successfully",
  "stats": {
    "usersRestored": 50,
    "serversRestored": 10,
    "proxiesRestored": 1,
    "ipWhitelistRestored": 5,
    "serversStarted": 8,
    "serversFailed": 2
  }
}
```

**Restore Process**:
1. Stop all MCP server connections
2. Disconnect all user sessions
3. In database transaction: delete all existing data -> insert backup data
4. Reload IP whitelist to memory
5. Reinitialize enabled MCP servers

---

### Log Operations (7000-7099)

#### 7001 SET_LOG_WEBHOOK_URL

**Permission**: **Owner only**
**Function**: Set log sync webhook URL

**Request Parameters** (data):
- `proxyKey` (string, required): Proxy key
- `webhookUrl` (string or null, required): Webhook URL (`null` means disable sync)

**Return Result** (data):
```json
{
  "proxyId": 1,
  "proxyName": "My MCP Server",
  "webhookUrl": "https://example.com/webhook",
  "message": "Log webhook URL set successfully"
}
```

**Function Description**:
- After setting webhook URL, logs will automatically sync to specified URL
- Set to `null` to disable log sync
- URL must use http or https protocol

---

#### 7002 GET_LOGS

**Permission**: **Owner only**
**Function**: Get log records

**Request Parameters** (data):
- `id` (number, optional): Starting log ID (defaults to 0, starts from first record)
- `limit` (number, optional): Number of records to return (default 1000, max 5000)

**Return Result** (data):
```json
{
  "logs": [
    {
      "id": 1,
      "action": 1,
      "userid": "user123",
      "serverId": null,
      "createdAt": 1729431234,
      "sessionId": "",
      "upstreamRequestId": "",
      "uniformRequestId": null,
      ...
    }
  ],
  "count": 1,
  "startId": 1,
  "limit": 1000
}
```

**Function Description**:
- When `id` is 0, starts from first log
- `limit` exceeding 5000 will be automatically limited to 5000
- `createdAt` field is Unix timestamp (seconds, Int type)

---

### Cloudflared Operations (8000-8099)

#### 8001 UPDATE_CLOUDFLARED_CONFIG

**Permission**: Owner + Admin
**Function**: Update or create cloudflared configuration and immediately restart container to apply configuration

**Request Parameters** (data):
- `proxyKey` (string, required): Proxy key (for finding proxyId)
- `tunnelId` (string, required): Cloudflare Tunnel ID
- `subdomain` (string, required): Subdomain (e.g., `xxx.trycloudflare.com`)
- `credentials` (object or string, required): Tunnel credentials (object or JSON string, must contain `TunnelSecret` field)
- `publicIp` (string, optional): Public IP address (for record only), defaults to empty string

**Return Result** (data):
```json
{
  "dnsConf": {
    "id": 1,
    "tunnelId": "abc123",
    "subdomain": "my-app.trycloudflare.com",
    "type": 1,
    "proxyId": 1,
    "publicIp": "1.2.3.4",
    "createdBy": 1,
    "addtime": 1729431234,
    "updateTime": 1729431235
  },
  "restarted": true,
  "message": "Cloudflared config updated and restarted successfully",
  "publicUrl": "https://my-app.trycloudflare.com"
}
```

**Function Description**:
- If no configuration exists for this proxy in database, automatically creates new record
- If configuration already exists, updates existing record
- If old configuration was locally created (`createdBy = 0`), automatically calls Cloud API to delete old tunnel
- Externally created configurations (`createdBy = 1`) will not delete old tunnel
- Automatically writes credential files to `./cloudflared/{tunnelId}.json` and `./cloudflared/credentials.json`
- Calls cloudflared restart script to restart cloudflared container
- If restart fails, still returns success (data saved), but `restarted: false` and includes error information

**credentials object example**:
```json
{
  "AccountTag": "xxx",
  "TunnelSecret": "xxx",
  "TunnelID": "abc123",
  "TunnelName": "my-tunnel"
}
```

---

#### 8002 GET_CLOUDFLARED_CONFIGS

**Permission**: Owner + Admin
**Function**: Query cloudflared configuration list (supports multi-condition filtering, AND relationship), and returns Docker container running status

**Request Parameters** (data, all parameters optional, AND relationship):
- `proxyKey` (string, optional): Filter by Proxy key
- `tunnelId` (string, optional): Filter by Tunnel ID
- `subdomain` (string, optional): Filter by subdomain
- `type` (number, optional): Filter by type (usually 1)

**Return Result** (data):
```json
{
  "dnsConfs": [
    {
      "id": 1,
      "tunnelId": "abc123",
      "subdomain": "my-app.trycloudflare.com",
      "type": 1,
      "proxyId": 1,
      "publicIp": "1.2.3.4",
      "createdBy": 1,
      "addtime": 1729431234,
      "updateTime": 1729431235,
      "status": "running"
    }
  ]
}
```

**Function Description**:
- All provided parameters must match simultaneously (AND relationship)
- Returns all configurations if no parameters provided
- Return result is an array, may be empty array
- Each record includes Docker container's real-time running status

**Field Description**:
- `type`: Configuration type (`1` = Cloudflare Tunnel)
- `createdBy`: Creation source (`0` = locally auto-created, `1` = externally API created)
- `proxyId`: Associated Proxy ID
- `addtime`: Creation time (Unix timestamp, seconds)
- `updateTime`: Last update time (Unix timestamp, seconds)
- `status`: Docker container status (`"running"` = running, `"stopped"` = stopped, `"not_exist"` = not exist)

---

#### 8003 DELETE_CLOUDFLARED_CONFIG

**Permission**: Owner + Admin
**Function**: Delete cloudflared configuration, stop and delete Docker container, clean up local files and database records

**Request Parameters** (data, at least one required):
- `id` (number, optional): DNS configuration record ID
- `tunnelId` (string, optional): Tunnel ID

**Return Result** (data):
```json
{
  "success": true,
  "message": "Cloudflared configuration deleted successfully",
  "deletedConfig": {
    "id": 1,
    "tunnelId": "abc123",
    "subdomain": "my-app.trycloudflare.com"
  }
}
```

**Function Description**:
- Stop Docker container (`docker stop kimbap-core-cloudflared`)
- Delete Docker container (`docker rm kimbap-core-cloudflared`)
- Delete local credential files:
  - `cloudflared/{tunnelId}.json`
  - `cloudflared/credentials.json`
  - `cloudflared/config.yml`
- Delete configuration record from database
- **Will not call Cloud API to delete remote tunnel** (remote tunnel remains in Cloudflare account)
- If container or files don't exist, doesn't affect deletion process (ignore errors and continue)

**Error Cases**:
- If corresponding configuration not found in database, returns `CLOUDFLARED_CONFIG_NOT_FOUND (8001)` error
- If container operation fails (and container actually exists), returns `TUNNEL_DELETE_FAILED (8004)` error

---

#### 8004 RESTART_CLOUDFLARED

**Permission**: Owner + Admin
**Function**: Restart cloudflared service, verify configuration completeness then restart Docker container

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "success": true,
  "message": "Cloudflared restarted successfully",
  "containerStatus": "running",
  "config": {
    "tunnelId": "abc123",
    "subdomain": "my-app.trycloudflare.com",
    "publicUrl": "https://my-app.trycloudflare.com"
  }
}
```

**Function Description**:
1. **Strictly verify local settings**:
   - Check if database has configuration record with `type=1`
   - Check if local credential file `cloudflared/{tunnelId}.json` exists
   - Verify credential file contains required `TunnelSecret` field
2. **Execute restart**:
   - Call cloudflared restart script to restart container
   - Verify container started successfully (status is `running`)
3. **Return configuration information**:
   - Includes Tunnel ID, subdomain, and public access URL

**Error Cases**:
- If no configuration in database, returns `CLOUDFLARED_DATABASE_CONFIG_NOT_FOUND (8005)` error
- If local file missing or format error, returns `CLOUDFLARED_LOCAL_FILE_NOT_FOUND (8006)` error
- If restart script execution fails, returns `CLOUDFLARED_RESTART_FAILED (8003)` error
- If container status after startup is not `running`, returns `CLOUDFLARED_RESTART_FAILED (8003)` error

**Important Notes**:
- This operation will not automatically fix missing data or files
- If data is incomplete, please use `UPDATE_CLOUDFLARED_CONFIG (8001)` to reconfigure first

---

#### 8005 STOP_CLOUDFLARED

**Permission**: Owner + Admin
**Function**: Stop cloudflared service (stop Docker container, do not delete container and configuration)

**Request Parameters** (data):
```json
{}
```

**Return Result** (data):
```json
{
  "success": true,
  "message": "Cloudflared stopped successfully",
  "containerStatus": "stopped",
  "alreadyStopped": false
}
```

**If container already stopped**:
```json
{
  "success": true,
  "message": "Cloudflared container is already stopped",
  "containerStatus": "stopped",
  "alreadyStopped": true
}
```

**Function Description**:
1. Check container current running status
2. If container is not running (`stopped` or `not_exist`), directly return success
3. If container is running, execute `docker stop kimbap-core-cloudflared`
4. Verify container has stopped
5. **Preserved**:
   - Docker container (stopped state)
   - Local configuration files
   - Database configuration records

**Error Cases**:
- If container stop command execution fails, returns `CLOUDFLARED_STOP_FAILED (8007)` error
- If container still running after executing stop command, returns `CLOUDFLARED_STOP_FAILED (8007)` error

**Important Notes**:
- After stopping, can use `RESTART_CLOUDFLARED (8004)` to quickly restore service
- For complete cleanup, use `DELETE_CLOUDFLARED_CONFIG (8003)`

---

### Policy Operations (9100-9199)

#### 9101 CREATE_POLICY_SET

**Permission**: Owner + Admin
**Function**: Create a new tool policy set. If `serverId` is omitted, the policy set is global (applies to all servers).

**Request Parameters** (data):
- `serverId` (string, optional): Server ID to scope this policy set. Omit for a global policy set.
- `dsl` (object, required): Policy DSL object defining the rules for this policy set.

**Return Result** (data):
```json
{
  "id": "clxxx...",
  "serverId": "server-abc",
  "version": 1,
  "status": "active",
  "dsl": { "schemaVersion": 1, "rules": [] },
  "createdAt": "2026-01-01T00:00:00.000Z",
  "updatedAt": "2026-01-01T00:00:00.000Z"
}
```

**DSL Schema Reference**:
The `dsl` object follows this structure:
- `schemaVersion` (number): Schema version, currently `1`
- `rules` (array): List of policy rules. Each rule contains:
  - `match` (object): Criteria to match a tool call (e.g., `toolName`, `serverId`)
  - `extract` (object, optional): Fields to extract from tool arguments for condition evaluation
  - `when` (object, optional): Conditions that must be true for the rule to apply
  - `effect` (string): Action to take — `"allow"`, `"deny"`, or `"require_approval"`

---

#### 9102 GET_POLICY_SETS

**Permission**: Owner + Admin
**Function**: List policy sets. If `serverId` is provided, returns only policy sets for that server. Otherwise returns all policy sets.

**Request Parameters** (data):
- `serverId` (string, optional): Filter by server ID.

**Return Result** (data):
```json
{
  "policySets": [
    {
      "id": "clxxx...",
      "serverId": "server-abc",
      "version": 1,
      "status": "active",
      "dsl": { "schemaVersion": 1, "rules": [] },
      "createdAt": "2026-01-01T00:00:00.000Z",
      "updatedAt": "2026-01-01T00:00:00.000Z"
    }
  ]
}
```

---

#### 9103 UPDATE_POLICY_SET

**Permission**: Owner + Admin
**Function**: Update an existing policy set's DSL or status. The `version` field increments when `dsl` is updated.

**Request Parameters** (data):
- `id` (string, required): Policy set ID.
- `dsl` (object, optional): New DSL object to replace the existing one.
- `status` (string, optional): New status — `"active"` or `"archived"`.

**Return Result** (data):
```json
{
  "id": "clxxx...",
  "serverId": "server-abc",
  "version": 2,
  "status": "active",
  "dsl": { "schemaVersion": 1, "rules": [] },
  "createdAt": "2026-01-01T00:00:00.000Z",
  "updatedAt": "2026-01-01T00:01:00.000Z"
}
```

---

#### 9104 DELETE_POLICY_SET

**Permission**: Owner + Admin
**Function**: Permanently delete a policy set by ID. Returns the deleted record.

**Request Parameters** (data):
- `id` (string, required): Policy set ID to delete.

**Return Result** (data):
```json
{
  "id": "clxxx...",
  "serverId": "server-abc",
  "version": 2,
  "status": "archived",
  "dsl": { "schemaVersion": 1, "rules": [] },
  "createdAt": "2026-01-01T00:00:00.000Z",
  "updatedAt": "2026-01-01T00:01:00.000Z"
}
```

---

#### 9105 GET_EFFECTIVE_POLICY

**Permission**: Owner + Admin
**Function**: Get the effective (active) policy sets for a given server. Returns active server-specific policy sets combined with active global policy sets (where `serverId` is null). If `serverId` is omitted, returns only global active policy sets.

**Request Parameters** (data):
- `serverId` (string, optional): Server ID to resolve effective policy for.

**Return Result** (data):
```json
{
  "policySets": [
    {
      "id": "clxxx...",
      "serverId": "server-abc",
      "version": 1,
      "status": "active",
      "dsl": { "schemaVersion": 1, "rules": [] },
      "createdAt": "2026-01-01T00:00:00.000Z",
      "updatedAt": "2026-01-01T00:00:00.000Z"
    }
  ]
}
```

**Function Description**:
- Only returns policy sets with `status = "active"`
- Combines server-specific policy sets (matching `serverId`) with global policy sets (`serverId` is null)
- Use this to determine which rules are currently enforced for a given server

---

### Approval Operations (9200-9299)

#### 9201 LIST_APPROVAL_REQUESTS

**Permission**: Owner + Admin
**Function**: List pending, non-expired approval requests with optional filtering. All filter parameters are optional and combined with AND logic.

**Request Parameters** (data):
- `userId` (string, optional): Filter by user ID.
- `serverId` (string, optional): Filter by server ID.
- `toolName` (string, optional): Filter by tool name.

**Return Result** (data):
```json
{
  "requests": [
    {
      "id": "clxxx...",
      "userId": "user-abc",
      "serverId": "server-abc",
      "toolName": "send_email",
      "canonicalArgs": {},
      "redactedArgs": {},
      "requestHash": "sha256-hex",
      "status": "PENDING",
      "decidedAt": null,
      "decisionReason": null,
      "executedAt": null,
      "executionError": null,
      "uniformRequestId": null,
      "expiresAt": "2026-01-01T01:00:00.000Z",
      "policyVersion": 1,
      "createdAt": "2026-01-01T00:00:00.000Z",
      "updatedAt": "2026-01-01T00:00:00.000Z"
    }
  ]
}
```

---

#### 9202 GET_APPROVAL_REQUEST

**Permission**: Owner + Admin
**Function**: Get a single approval request by ID.

**Request Parameters** (data):
- `id` (string, required): Approval request ID.

**Return Result** (data):
```json
{
  "id": "clxxx...",
  "userId": "user-abc",
  "serverId": "server-abc",
  "toolName": "send_email",
  "canonicalArgs": {},
  "redactedArgs": {},
  "requestHash": "sha256-hex",
  "status": "PENDING",
  "decidedAt": null,
  "decisionReason": null,
  "executedAt": null,
  "executionError": null,
  "uniformRequestId": null,
  "expiresAt": "2026-01-01T01:00:00.000Z",
  "policyVersion": 1,
  "createdAt": "2026-01-01T00:00:00.000Z",
  "updatedAt": "2026-01-01T00:00:00.000Z"
}
```

**Field Description**:
- `status`: Current state — `"PENDING"`, `"APPROVED"`, `"REJECTED"`, or `"EXPIRED"`
- `decidedAt`: Timestamp when the decision was made, or `null`
- `decisionReason`: Optional reason provided when decided, or `null`
- `canonicalArgs`: Normalized tool arguments used for hashing/comparison
- `redactedArgs`: Redacted arguments safe for logging/display
- `requestHash`: Deterministic hash used for deduplication
- `expiresAt`: Timestamp after which the request automatically expires
- `policyVersion`: The policy set version that triggered this approval request
- `executedAt` / `executionError`: Execution result metadata after approval
- `uniformRequestId`: Optional correlation ID for upstream request mapping

---

#### 9203 DECIDE_APPROVAL_REQUEST

**Permission**: Owner + Admin
**Function**: Approve or reject a pending approval request. The deciding admin's identity is recorded.

**Request Parameters** (data):
- `id` (string, required): Approval request ID.
- `decision` (string, required): Decision to apply — `"APPROVED"` or `"REJECTED"`.
- `reason` (string, optional): Human-readable reason for the decision.

**Return Result** (data):
```json
{
  "id": "clxxx...",
  "userId": "user-abc",
  "serverId": "server-abc",
  "toolName": "send_email",
  "canonicalArgs": {},
  "redactedArgs": {},
  "requestHash": "sha256-hex",
  "status": "APPROVED",
  "decidedAt": "2026-01-01T00:05:00.000Z",
  "decisionReason": "Reviewed and approved",
  "executedAt": null,
  "executionError": null,
  "uniformRequestId": null,
  "expiresAt": "2026-01-01T01:00:00.000Z",
  "policyVersion": 1,
  "createdAt": "2026-01-01T00:00:00.000Z",
  "updatedAt": "2026-01-01T00:05:00.000Z"
}
```

**Error Cases**:
- If the request is not in `PENDING` status, the decision cannot be applied

---

#### 9204 COUNT_PENDING_APPROVALS

**Permission**: Owner + Admin
**Function**: Count the number of pending approval requests for a specific user.

**Request Parameters** (data):
- `userId` (string, required): User ID to count pending requests for.

**Return Result** (data):
```json
{
  "count": 5
}
```

---

### Skills Operations (10040-10043)

#### 10040 LIST_SKILLS

**Permission**: Owner + Admin
**Function**: List all skills associated with a server

**Request Parameters** (data):
- `serverId` (string, required): Server ID

**Return Result** (data):
```json
{
  "skills": [...]
}
```

---

#### 10041 UPLOAD_SKILL

**Permission**: Owner + Admin
**Function**: Upload a skill to a server

**Request Parameters** (data):
- `serverId` (string, required): Server ID
- `skill` (object, required): Skill definition data

**Return Result** (data):
```json
{
  "message": "Skill uploaded successfully"
}
```

---

#### 10042 DELETE_SKILL

**Permission**: Owner + Admin
**Function**: Delete a specific skill

**Request Parameters** (data):
- `skillId` (string, required): Skill ID

**Return Result** (data):
```json
{
  "message": "Skill deleted successfully"
}
```

---

#### 10043 DELETE_SERVER_SKILLS

**Permission**: Owner + Admin
**Function**: Delete all skills associated with a server

**Request Parameters** (data):
- `serverId` (string, required): Server ID

**Return Result** (data):
```json
{
  "deletedCount": 5,
  "message": "Server skills deleted successfully"
}
```

---

## Appendix: Error Code Reference

### General Errors (1000-1999)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 1001 | INVALID_REQUEST | Request format error, missing required fields, parameter type error |
| 1002 | UNAUTHORIZED | No valid authentication Token provided |
| 1003 | FORBIDDEN | Insufficient permissions (e.g., non-Owner role attempting Owner-only operation) |

### User Related Errors (2000-2999)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 2001 | USER_NOT_FOUND | Specified user ID does not exist |
| 2003 | USER_ALREADY_EXISTS | When creating user, userId already exists |

### Server Related Errors (3000-3999)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 3001 | SERVER_NOT_FOUND | Specified server ID does not exist |
| 3003 | SERVER_ALREADY_EXISTS | When creating server, serverId already exists |

### Proxy Related Errors (5000-5099)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 5001 | PROXY_NOT_FOUND | Specified proxy does not exist |
| 5002 | PROXY_ALREADY_EXISTS | System already has proxy (only one allowed) |

### IP Whitelist Related Errors (5100-5199)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 5101 | IPWHITELIST_NOT_FOUND | Specified IP whitelist record does not exist |
| 5102 | INVALID_IP_FORMAT | IP address or CIDR format invalid |

### Database Operation Errors (5200-5299)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 5201 | DATABASE_OPERATION_FAILED | Database operation failed |

### Backup and Restore Errors (5300-5399)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 5301 | BACKUP_FAILED | Database backup failed |
| 5302 | RESTORE_FAILED | Database restore failed |
| 5303 | INVALID_BACKUP_DATA | Backup data format invalid |

### Cloudflared Related Errors (8000-8099)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 8001 | CLOUDFLARED_CONFIG_NOT_FOUND | Specified cloudflared configuration not found in database |
| 8002 | INVALID_CREDENTIALS_FORMAT | Tunnel credentials format invalid or missing TunnelSecret field |
| 8003 | CLOUDFLARED_RESTART_FAILED | Cloudflared restart failed (script execution failed or container not started) |
| 8004 | TUNNEL_DELETE_FAILED | Failed to delete tunnel or container |
| 8005 | CLOUDFLARED_DATABASE_CONFIG_NOT_FOUND | Cloudflared configuration does not exist in database (during restart) |
| 8006 | CLOUDFLARED_LOCAL_FILE_NOT_FOUND | Local credential file does not exist or format error |
| 8007 | CLOUDFLARED_STOP_FAILED | Failed to stop cloudflared container |

### Skills Related Errors (9000-9099)

| Error Code | Name | Trigger Condition |
|--------|------|----------|
| 9001 | SKILL_NOT_FOUND | Specified skill does not exist |
| 9002 | SKILL_UPLOAD_FAILED | Failed to upload skill |
| 9003 | SKILL_DELETE_FAILED | Failed to delete skill |
| 9004 | INVALID_SKILL_FORMAT | Skill format is invalid |

---

## Version Information

- **Protocol Version**: 2.1 (Go port)
- **Last Updated**: February 2026
- **Update Content**:
  - Updated to match the current Go Admin API implementation
  - Added Skills operations (10040-10043)
  - Added Skills error codes (9001-9004)
  - Added TunnelCreateFailed error code (8008)
  - Added UserRole.Guest (4)
  - Added ServerCategory.Skills (4)
  - ServerAuthType: Removed StripeAuth, shifted ZendeskAuth to 7, CanvasAuth to 8
  - Built with chi v5 router and GORM v2

**Version History**:
- **2.1-go** (February 2026):
  - Complete Go port with additional Skills operations
  - Updated enum values to match Go implementation
- **2.1** (November 7, 2025):
  - Added Cloudflared operation APIs:
    - 8003 DELETE_CLOUDFLARED_CONFIG - Delete cloudflared configuration
    - 8004 RESTART_CLOUDFLARED - Restart cloudflared service
    - 8005 STOP_CLOUDFLARED - Stop cloudflared service
  - Enhanced 8002 GET_CLOUDFLARED_CONFIGS - Added status field to return result (container running status)
  - Added Cloudflared related error codes (8005-8007)
  - Improved error handling and status management descriptions for Cloudflared operations
- **2.0** (October 20, 2025):
  - Complete document rewrite, sorted by AdminActionType numbers (1001-7002)
  - Added detailed request parameters and return result descriptions for each API
  - Marked all Owner-only operations
  - Unified error code reference table
  - Added curl and HTTP client calling examples

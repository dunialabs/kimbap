# Socket.IO Usage Documentation

This document introduces the Socket.IO bidirectional communication functionality of Kimbap Core.

## Overview

Socket.IO provides real-time bidirectional communication capabilities between server and clients, supporting:

- ✅ **Server-initiated Push**: Push notifications to specified users or all users
- ✅ **Client Messages**: Clients can send messages to server
- ✅ **Token Authentication**: Verify user identity during handshake
- ✅ **Multi-device Login**: Same user can connect on multiple devices simultaneously
- ✅ **Auto-reconnection**: Client automatically reconnects after disconnection
- ✅ **Independent from MCP**: Does not affect existing MCP SSE push mechanism

## Architecture Description

Socket.IO server is attached to the existing HTTP/HTTPS server (port 3002), completely independent from MCP routes:

```
┌─────────────────────────────────────────────────────────────┐
│                    Kimbap Core (3002)                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐         ┌──────────────────────────────┐   │
│  │     chi     │────────>│   HTTP/HTTPS Server          │   │
│  │   Router    │         └──────────────────────────────┘   │
│  └─────────────┘                      │                     │
│        │                              │                     │
│        │                              ▼                     │
│        │                    ┌──────────────────┐            │
│        │                    │   Socket.IO      │            │
│        │                    │   Server         │            │
│        │                    └──────────────────┘            │
│        │                               │                    │
│        ▼                               ▼                    │
│  ┌──────────┐              ┌──────────────────┐             │
│  │  /mcp    │              │  /socket.io      │             │
│  │  (SSE)   │              │  (WebSocket)     │             │
│  └──────────┘              └──────────────────┘             │
│        │                               │                    │
└────────┼───────────────────────────────┼────────────────────┘
         │                               │
         ▼                               ▼
   ┌──────────┐                 ┌──────────────┐
   │   MCP    │                 │   Electron   │
   │ Clients  │                 │   Clients    │
   │ (Claude) │                 │  (Desktop)   │
   └──────────┘                 └──────────────┘
```

---

## Server Usage

### 1. Push Notification to Specified User

Use `SocketNotifier` to push notifications:

```go
import "github.com/dunialabs/kimbap-core/internal/socket"

notifier := socket.GetSocketNotifier()

// Push to all devices of specified user
notifier.NotifyUser("user123", "notification", map[string]interface{}{
	"type":      "system_message",
	"message":   "Hello from server!",
	"timestamp": time.Now().UnixMilli(),
	"severity":  "info",
})
```

### 2. Using Convenient Push Functions

```go
import "github.com/dunialabs/kimbap-core/internal/socket"

notifier := socket.GetSocketNotifier()

// Push user disabled notification
notifier.NotifyUserDisabled("user123", "Violates terms of service")

// Push permission change notification
notifier.NotifyUserPermissionChanged("user123")

// Push custom notification (to specified user)
notifier.SendNotification("user123", socket.NotificationData{
	Type:      "system_message",
	Message:   "System maintenance in 10 minutes",
	Timestamp: time.Now().UnixMilli(),
	Severity:  "warning",
})

// Push user online session list
notifier.NotifyOnlineSessions("user123")

// Push user expired notification (before session teardown)
notifier.NotifyUserExpired("user123")

// Push server status change notification (filtered by user permissions)
notifier.NotifyServerStatusChanged("server-id", "My Server", types.ServerStatusOnline, types.ServerStatusError)

// Push permission change notification to all users with access to a server
notifier.NotifyUserPermissionChangedByServer("server-id")
```

### 3. Push User Online Session List

The system automatically tracks user MCP session status and proactively pushes notifications when sessions are created, initialized, or closed.

#### Automatic Trigger Timing

The following situations automatically trigger online session list notifications:

1. **Socket.IO Connection Established**: When user successfully connects via Socket.IO, immediately push current all active MCP session list
2. **MCP Session Initialization Completed**: When MCP client completes `initialize` request, push updated session list
3. **MCP Session Closed**: When MCP session closes (normal close or timeout), push updated session list

#### Manual Trigger

```go
import "github.com/dunialabs/kimbap-core/internal/socket"

notifier := socket.GetSocketNotifier()

// Push user's online session list
success := notifier.NotifyOnlineSessions("user123")

if success {
	log.Println("✅ Online session notification sent")
} else {
	log.Println("❌ User offline or notification failed")
}
```

#### Notification Data Structure

Clients will receive notifications in the following format:

```javascript
{
  type: 'online_sessions',
  message: 'You have 3 active session(s)',  // Dynamically generated based on session count
  data: {
    sessions: [
      {
        sessionId: "sess_abc123",           // MCP session ID
        clientName: "Claude Desktop",       // Client application name
        userAgent: "Mozilla/5.0...",        // HTTP User-Agent
        lastActive: "2025-01-15T10:00:00Z"  // Last active time (ISO 8601)
      },
      {
        sessionId: "sess_xyz789",
        clientName: "Web Client",
        userAgent: "Mozilla/5.0...",
        lastActive: "2025-01-15T10:05:00Z"
      }
    ]
  },
  timestamp: 1736935200000,
  severity: 'info'
}
```

#### Field Description

- `sessionId`: Unique identifier for MCP session
- `clientName`: Client application name (obtained from MCP `initialize` request's `clientInfo.name`)
  - If client doesn't provide, displays as `"Unknown Client"`
- `userAgent`: HTTP User-Agent string (obtained from HTTP request header)
  - If not obtained, displays as `"Unknown"`
- `lastActive`: Session last active time (ISO 8601 format)

#### Client Handling Example

```javascript
import { io } from 'socket.io-client';

const socket = io('http://localhost:3002', {
  auth: { token: 'your-token' }
});

// Listen for online session notifications
socket.on('notification', (data) => {
  if (data.type === 'online_sessions') {
    console.log(`You have ${data.data.sessions.length} active sessions:`);

    data.data.sessions.forEach(session => {
      console.log(`- ${session.clientName} (${session.sessionId})`);
      console.log(`  Last active: ${new Date(session.lastActive).toLocaleString()}`);
      console.log(`  User-Agent: ${session.userAgent}`);
    });

    // Update UI display
    updateSessionList(data.data.sessions);
  }
});
```

#### Use Cases

1. **Multi-device Management**: Display user's current all active connections in desktop app, allow user to close sessions from other devices
2. **Security Monitoring**: Detect abnormal logins, if user sees unknown sessions, can immediately disconnect
3. **Session Synchronization**: Synchronize session status between multiple clients, remind user that other devices are in use
4. **Debugging Tool**: Developers can view current active session details, convenient for troubleshooting

#### Technical Implementation Notes

**Data Sources:**
- `sessionId`: From `ClientSession.sessionId` (generated when creating MCP session)
- `clientName`: From MCP `initialize` request's `clientInfo.name` field
- `userAgent`: From HTTP request header `User-Agent` when creating MCP session
- `lastActive`: From `ClientSession.lastActive` (automatically updated on each request)

**Storage Location:**
- Session information stored in `SessionStore`'s in-memory map
- Each user's session collection managed through user sessions map

**Notification Trigger Points:**
1. After Socket.IO connection established
2. After MCP Session initialization completed
3. After MCP Session closed

### 4. Broadcast to All Online Users

```go
import "github.com/dunialabs/kimbap-core/internal/socket"

notifier := socket.GetSocketNotifier()

notifier.NotifyAll("notification", map[string]interface{}{
	"type":      "system_update",
	"message":   "New features available!",
	"timestamp": time.Now().UnixMilli(),
	"severity":  "info",
})
```

### 5. Query Online Status

```go
import "github.com/dunialabs/kimbap-core/internal/socket"

notifier := socket.GetSocketNotifier()

// Check if user is online
isOnline := notifier.IsUserOnline("user123")

// Get user's online device count
deviceCount := notifier.GetUserDeviceCount("user123")

// Get user's connection information
connections := notifier.GetUserConnections("user123")

// Get all online user IDs
onlineUsers := notifier.GetOnlineUserIDs()

// Get total connection count
totalConnections := notifier.GetTotalConnections()
```

### 6. Using in Admin Operations

Example: Push notification when disabling user

```go
import "github.com/dunialabs/kimbap-core/internal/socket"

// In UserHandler or other admin controllers
func (h *UserHandler) DisableUser(userId string, reason string) error {
	// 1. Update database
	err := h.userRepo.UpdateStatus(userId, types.UserStatusDisabled)
	if err != nil {
		return err
	}

	// 2. Push notification to all devices of that user
	notifier := socket.GetSocketNotifier()
	notifier.NotifyUserDisabled(userId, reason)

	// 3. Optional: Disconnect all MCP sessions of that user
	sessionStore.RemoveUserSessions(userId)

	return nil
}
```

---

## Request-Response Pattern

### Overview

In addition to one-way notifications, Socket.IO also supports **request-response pattern**, allowing server to send requests and wait for client responses.

**Core Features:**
- ✅ Standardized message structure similar to AdminRequest/Response
- ✅ Automatically generate unique requestId to associate requests and responses
- ✅ Configurable response timeout (default 55 seconds)
- ✅ Complete type safety support
- ✅ Never throws exceptions, always returns response object
- ✅ Automatically clean up timed-out and disconnected requests

### SocketActionType Operation Types

Currently supported operation types (can be extended at any time):

```go
const (
	// ========== 1000-1999: User Confirmation ==========
	SocketActionAskUserConfirm = 1001 // Request user confirmation for operation

	// ========== 2000-2999: Client Status Query ==========
	SocketActionGetClientStatus = 2001 // Get client status
)
```

### Core Method: SendRequest()

Send request and wait for client response (blocking method).

**Method Signature:**

```go
func (n *SocketNotifier) SendRequest(
	userID string,
	action SocketActionType,
	data any,
	timeout time.Duration,
) SocketResponse[map[string]any]
```

**Parameter Description:**
- `userID` - User ID
- `action` - Operation type (`SocketActionType` constant)
- `data` - Request data
- `timeout` - Timeout duration (default 55 seconds)

**Return Value:**
- `SocketResponse[map[string]any]` - Always returns response object, never throws exception

**Usage Example:**

```go
import (
	"time"
	"github.com/dunialabs/kimbap-core/internal/socket"
)

notifier := socket.GetSocketNotifier()

// Example 1: Use default timeout (55 seconds)
response := notifier.SendRequest(
	"user123",
	socket.SocketActionAskUserConfirm,
	map[string]any{
		"message": "Are you sure you want to delete this server?",
	},
	55*time.Second,
)

if response.Success {
	confirmed := (*response.Data)["confirmed"].(bool)
	log.Printf("User confirmed: %v", confirmed)
	if confirmed {
		deleteServer(serverId)
	}
} else {
	log.Printf("Request failed: %s", response.Error.Message)
	// Possible errors: USER_OFFLINE, TIMEOUT, SERVER_ERROR, etc.
}

// Example 2: Custom timeout
response := notifier.SendRequest(
	"user123",
	socket.SocketActionGetClientStatus,
	map[string]any{},
	10*time.Second, // 10 second timeout
)
```

### Convenient Wrapper Methods

#### AskUserConfirm() - Request User Confirmation

```go
func (n *SocketNotifier) AskUserConfirm(
	ctx context.Context,
	userID string,
	userAgent string,
	ip string,
	toolName string,
	description string,
	params string,
	timeout time.Duration,
) (bool, error)
```

**Parameter Description:**
- `ctx` - Context (for cancellation)
- `userID` - User ID
- `userAgent` - Client User-Agent string
- `ip` - Client IP address
- `toolName` - Tool name
- `description` - Tool description
- `params` - Tool parameters (JSON string format)
- `timeout` - Timeout duration (default 55 seconds)

**Return Value:**
- `(true, nil)` - User explicitly confirmed
- `(false, nil)` - User rejected, timed out, or context cancelled

**Example:**

```go
notifier := socket.GetSocketNotifier()

confirmed, err := notifier.AskUserConfirm(
	ctx,
	"user123",
	r.UserAgent(),
	r.RemoteAddr,
	"delete_server",
	"Delete a server permanently",
	`{"serverId": "abc123", "force": true}`,
	55*time.Second,
)

if confirmed {
	// User confirmed, execute operation
	deleteServer("abc123")
} else {
	// User rejected or timed out
	log.Println("Operation cancelled by user or timed out")
}
```

#### GetClientStatus() - Get Client Status

```go
func (n *SocketNotifier) GetClientStatus(userID string, timeout time.Duration) (any, error)
```

**Example:**

```go
notifier := socket.GetSocketNotifier()

status, err := notifier.GetClientStatus("user123", 5*time.Second)

if err == nil {
	log.Printf("Client status: %+v", status)
} else {
	log.Println("Failed to get client status")
}
```

---

## Capability Configuration Management

### Overview

User capability configuration management allows clients to get and set their own MCP server capability configurations (`user_preferences`) through Socket.IO, enabling client-side custom permission control.

**Core Features:**
- ✅ Get current complete capability configuration (includes merged result of admin permissions + user custom configuration)
- ✅ Set user custom configuration (can only further restrict, cannot expand permissions)
- ✅ Real-time notification to all active sessions of configuration changes
- ✅ Automatic validation of configuration legality

### Permission Merge Rules

```
Final Permission = Admin Configured Permissions && User Custom Configuration

final_enabled = admin_permissions.enabled && user_preferences.enabled
```

**Description:**
- Admin configured `permissions` is the baseline (upper limit) of permissions
- Users can only further restrict through `user_preferences`, cannot expand permissions
- Items not configured by user default to follow admin configuration

### 1. Get Capability Configuration

Get current user's complete capability configuration through Socket.IO request-response pattern.

#### Client Sends Request

```javascript
// Client emits get_capabilities to request current configuration
const requestId = crypto.randomUUID();
socket.emit('get_capabilities', { requestId, data: {} });

// Listen for server response
socket.on('socket_response', (response) => {
  if (response.requestId === requestId && response.success) {
    console.log('Capability configuration:', response.data);
  }
});
```

#### Server-Side API (Go)

```go
import "github.com/dunialabs/kimbap-core/internal/user"

// The server handles get_capabilities internally.
// To query capabilities from Go code directly:
handler := user.NewRequestHandler(nil)
capabilities, err := handler.GetCapabilities("user123")

if err == nil {
	log.Printf("User capability configuration: %+v", capabilities)
} else {
	log.Println("Failed to get capabilities")
}
```

#### Returned Data Format

```javascript
// McpServerCapabilities structure
{
  "server-id-1": {
    "enabled": true,
    "tools": {
      "tool-name-1": {
        "enabled": true,
        "description": "Tool description",
        "dangerLevel": 0
      },
      "tool-name-2": {
        "enabled": false,  // User disabled
        "description": "Another tool",
        "dangerLevel": 1
      }
    },
    "resources": {
      "resource-name-1": {
        "enabled": true,
        "description": "Resource description"
      }
    },
    "prompts": {
      "prompt-name-1": {
        "enabled": true,
        "description": "Prompt description"
      }
    }
  },
  "server-id-2": {
    "enabled": false,  // Entire server disabled
    "tools": {},
    "resources": {},
    "prompts": {}
  }
}
```

### 2. Set Capability Configuration

Set user custom configuration directly through Socket.IO events.

#### Client Sends Setting Request

```javascript
// Client sends set_capabilities event
socket.emit('set_capabilities', {
  requestId: 'req-' + Date.now(),  // Optional, for tracking
  data: {
    // Only need to set parts to modify
    "server-id-1": {
      "enabled": true,
      "tools": {
        "dangerous-tool": {
          "enabled": false,  // Disable dangerous tool
          "description": "...",
          "dangerLevel": 2
        }
      },
      "resources": {},
      "prompts": {}
    }
  }
});

// Listen for response
socket.on('socket_response', (data) => {
  if (data.requestId && data.success) {
    console.log('Configuration updated:', data);
  }
});
```

#### Server Processing

Server automatically handles `set_capabilities` event:

1. Verify user identity
2. Get current complete configuration
3. Extract and validate user submitted `enabled` fields
4. Update database (`user_preferences` field)
5. Notify all active sessions of that user

#### Validation Rules

- ✅ Only accept existing server/tool/resource/prompt
- ✅ Only save `enabled`, `description`, `dangerLevel` fields
- ✅ Ignore non-existent items or invalid fields
- ✅ Do not allow expanding permissions (can only disable, cannot enable items disabled by admin)

### 3. Automatic Notification Mechanism

When user configuration is updated, the system automatically:

1. **Update Database**: Save to `users.user_preferences` field
2. **Notify All Sessions**: Send notifications to all active MCP sessions of that user
3. **Incremental Notification**: Only notify changed parts (tools/resources/prompts)

#### Notification Events

Clients will receive the following MCP protocol notifications (via SSE):

```json
// When tools change
{
  "jsonrpc": "2.0",
  "method": "notifications/tools/list_changed"
}

// When resources change
{
  "jsonrpc": "2.0",
  "method": "notifications/resources/list_changed"
}

// When prompts change
{
  "jsonrpc": "2.0",
  "method": "notifications/prompts/list_changed"
}
```

At the same time, if client is connected via Socket.IO, will also receive:

```json
// Socket.IO notification event
{
  "type": "permission_changed",
  "message": "User permissions have been updated",
  "data": {
    "capabilities": { /* latest configuration */ }
  },
  "timestamp": 1234567890,
  "severity": "warning"
}
```

### 4. Complete Example: Client Implementation

```javascript
import { io } from 'socket.io-client';

// Connect to server
const socket = io('http://localhost:3002', {
  auth: { token: 'your-token' }
});

// ========== Get Current Configuration ==========
async function getCurrentCapabilities() {
  return new Promise((resolve) => {
    const requestId = 'req-' + Date.now();

    // Send request
    socket.emit('get_capabilities', {
      requestId,
      data: {}
    });

    // Wait for response (one-time listener)
    socket.once('socket_response', (response) => {
      if (response.requestId === requestId && response.success) {
        resolve(response.data?.capabilities);
      } else {
        resolve(null);
      }
    });
  });
}

// ========== Set Configuration ==========
function setCapabilities(newConfig) {
  socket.emit('set_capabilities', {
    requestId: 'req-' + Date.now(),
    data: newConfig
  });
}

// ========== Usage Example ==========

// Get configuration
const currentConfig = await getCurrentCapabilities();
console.log('Current configuration:', currentConfig);

// Disable a tool
setCapabilities({
  "filesystem": {
    "enabled": true,
    "tools": {
      "delete_file": {
        "enabled": false,  // Disable delete file tool
        "description": "Delete a file",
        "dangerLevel": 2
      }
    },
    "resources": {},
    "prompts": {}
  }
});

// Listen for confirmation
socket.on('socket_response', (data) => {
  if (data.requestId && data.success) {
    console.log('✅ Configuration updated');

    // Re-fetch latest configuration
    getCurrentCapabilities().then(config => {
      console.log('Latest configuration:', config);
    });
  }
});
```

### 5. Database Field

User custom configuration is stored in `users` table's `user_preferences` field:

```sql
-- user_preferences field stores JSON string
-- Example:
{
  "server-id-1": {
    "enabled": true,
    "tools": {
      "tool-1": { "enabled": false, "description": "...", "dangerLevel": 1 }
    },
    "resources": {},
    "prompts": {}
  }
}
```

### 6. API Summary

| Operation | Method | Event Name | Data Format |
|-----|------|--------|---------|
| Get Configuration | Request-Response | `get_capabilities` | `{ requestId }` |
| Set Configuration | Client Event | `set_capabilities` | `{ requestId, data: McpServerCapabilities }` |
| Configuration Update Notification | Server Push | `notification` | `{ type: 'permission_changed', data: { capabilities } }` |

### Error Handling

All request-response methods **never throw exceptions**, always return response object.

**SocketErrorCode Error Codes:**

```go
const (
	// General errors (1000-1099)
	SocketErrorTimeout        = 1001 // Response timeout
	SocketErrorUserOffline    = 1002 // User offline

	// Server errors (1200-1299)
	SocketErrorServerError        = 1201 // Server internal error
	SocketErrorServiceUnavailable = 1202 // Service unavailable
)
```

**Error Handling Example:**

```go
notifier := socket.GetSocketNotifier()
response := notifier.SendRequest(...)

if !response.Success {
	switch response.Error.Code {
	case socket.SocketErrorUserOffline:
		log.Println("User is offline")
	case socket.SocketErrorTimeout:
		log.Println("Request timed out")
	default:
		log.Printf("Request failed: %s", response.Error.Message)
	}
}
```

### Client Implementation Request Handling

Clients need to listen for corresponding events and send responses.

**Event Name Automatic Mapping Rules:**
- `SocketActionAskUserConfirm` → Event name `'ask_user_confirm'`
- `SocketActionGetClientStatus` → Event name `'get_client_status'`

**Complete Client Implementation Example (Electron):**

```javascript
import { io, Socket } from 'socket.io-client';
import { dialog } from 'electron';
import {
  SocketRequest,
  SocketResponse,
  SocketActionType,
  SocketErrorCode
} from './socket.types';

// Connect to server
const socket = io('http://localhost:3002', {
  auth: { token: 'your-token-here' }
});

// Listen for 'ask_user_confirm' event
socket.on('ask_user_confirm', async (request: SocketRequest<{
  userAgent: string;
  ip: string;
  toolName: string;
  toolDescription: string;
  toolParams: string;
}>) => {
  try {
    // Parse tool parameters
    const params = JSON.parse(request.data.toolParams);

    // Construct confirmation message
    const message = `Tool: ${request.data.toolName}\nDescription: ${request.data.toolDescription}\nParameters: ${JSON.stringify(params, null, 2)}\n\nAre you sure you want to execute this operation?`;

    // Show confirmation dialog
    const result = await dialog.showMessageBox({
      type: 'question',
      message: message,
      buttons: ['Confirm', 'Cancel'],
      defaultId: 0,
      cancelId: 1
    });

    // Send response
    const response: SocketResponse = {
      requestId: request.requestId,
      success: true,
      data: { confirmed: result.response === 0 },
      timestamp: Date.now()
    };

    socket.emit('socket_response', response);
    console.log(`✅ Responded to request: ${request.requestId}`);

  } catch (error) {
    // Send error response
    const response: SocketResponse = {
      requestId: request.requestId,
      success: false,
      error: {
        code: 1201, // SocketErrorServerError
        message: error.message || 'Client error occurred'
      },
      timestamp: Date.now()
    };

    socket.emit('socket_response', response);
    console.error(`❌ Failed to handle request: ${request.requestId}`);
  }
});

// Listen for 'get_client_status' event
socket.on('get_client_status', (request: SocketRequest) => {
  const status = {
    platform: process.platform,
    appVersion: app.getVersion(),
    memoryUsage: process.memoryUsage(),
    uptime: process.uptime()
  };

  const response: SocketResponse = {
    requestId: request.requestId,
    success: true,
    data: status,
    timestamp: Date.now()
  };

  socket.emit('socket_response', response);
});
```

### Practical Application Scenarios

#### Scenario 1: Confirm Before Dangerous Tool Call

```go
// Confirm dangerous tool call in ProxySession
func (s *ProxySession) HandleToolCall(request CallToolRequest) error {
	// Check tool danger level
	if dangerLevel == types.DangerLevelApproval {
		toolDescription := s.getToolDescription(toolName)
		toolParams, _ := json.Marshal(request.Params.Arguments)

		// Request user confirmation
		notifier := socket.GetSocketNotifier()
		confirmed, _ := notifier.AskUserConfirm(
			ctx,
			userId,
			userAgent,
			ip,
			toolName,
			toolDescription,
			string(toolParams),
			55*time.Second,
		)

		if !confirmed {
			return errors.New("User denied tool execution")
		}
	}

	// Execute tool call after user confirmation
	return s.callTool(toolName, request.Params.Arguments)
}
```

#### Scenario 2: Get Client Status for Troubleshooting

```go
func diagnoseClientIssue(userId string) map[string]interface{} {
	// Get client status
	notifier := socket.GetSocketNotifier()
	status, err := notifier.GetClientStatus(userId, 10*time.Second)

	if err != nil {
		return map[string]interface{}{
			"error": "Unable to get client status",
		}
	}

	// Analyze status data
	issues := []string{}
	statusMap := status.(map[string]interface{})
	
	if memUsage, ok := statusMap["memoryUsage"].(map[string]interface{}); ok {
		if heapUsed, ok := memUsage["heapUsed"].(float64); ok && heapUsed > 1000000000 {
			issues = append(issues, "High memory usage")
		}
	}
	
	if uptime, ok := statusMap["uptime"].(float64); ok && uptime > 86400 {
		issues = append(issues, "Client needs restart")
	}

	return map[string]interface{}{
		"status": status,
		"issues": issues,
	}
}
```

### Timeout and Performance Considerations

**Default Timeout**: 55 seconds

**Recommended Timeout Configuration:**
- Quick operations (get status): 5-10 seconds
- User confirmation operations: 30-60 seconds
- Complex operations: 60-120 seconds

**Performance Monitoring:**

```go
// Get total connection count
connCount := socketService.GetTotalConnections()
log.Printf("Current connections: %d", connCount)
```

---

## Client Usage

### Install Dependencies

Prepare the Go server dependencies in this repository:

```bash
go mod download
```

For browser/Electron clients, install `socket.io-client` in your client project using your preferred JavaScript package manager.

### Basic Connection Example

```javascript
import { io } from 'socket.io-client';

// Create connection
const socket = io('http://localhost:3002', {
  auth: {
    token: 'your-user-token-here'  // Recommended method
  },
  reconnection: true,
  reconnectionAttempts: 5,
  reconnectionDelay: 1000
});

// Listen for successful connection
socket.on('connect', () => {
  console.log('Connected! Socket ID:', socket.id);

  // Send client information (optional)
  socket.emit('client-info', {
    deviceType: 'desktop',
    deviceName: 'MacBook Pro',
    appVersion: '1.0.0'
  });
});

// Listen for server push notifications
socket.on('notification', (data) => {
  console.log('Notification:', data);
  // Show desktop notification or update UI
});

// Send message to server
socket.emit('client-message', {
  action: 'test',
  data: { foo: 'bar' }
});

// Listen for message acknowledgment
socket.on('ack', (data) => {
  console.log('Message acknowledged:', data);
});

// Disconnect
socket.disconnect();
```

### Electron Complete Example

Use the complete JavaScript example in this document as the baseline integration reference.

---

## Event List

### Client → Server

| Event Name | Description | Data Format |
|---------|------|---------|
| `client-info` | Send device information (optional) | `{ deviceType?, deviceName?, appVersion?, platform? }` |
| `client-message` | Client message | Any data |
| `get_capabilities` | Get user capability configuration | `{ requestId?, data: {} }` |
| `set_capabilities` | Set user capability configuration | `{ requestId?, data: McpServerCapabilities }` |
| `configure_server` | Configure a server for user | `{ requestId?, data: { serverId, authConf?, remoteAuth?, restfulApiAuth? } }` |
| `unconfigure_server` | Unconfigure a server for user | `{ requestId?, data: { serverId } }` |
| `socket_response` | Respond to server request | `SocketResponse<T>` |

### Server → Client

| Event Name | Description | Data Format |
|---------|------|---------|
| `notification` | Server push notification | `{ type, message, timestamp, severity?, data? }` |
| `ack` | Message acknowledgment (for `client-message` only) | `{ message, timestamp }` |
| `socket_response` | Response to client action requests | `SocketResponse<T>` (contains `requestId`, `success`, `data?`, `error?`) |
| `server_info` | Server information (sent on connect) | `{ serverId, serverName, version }` |
| `ask_user_confirm` | Request user confirmation | `SocketRequest<{ toolName, toolDescription, toolParams }>` |
| `get_client_status` | Get client status | `SocketRequest<{}>` |

> **Note**: `socket_response` is used in both directions. The **server** emits it as a reply to client action requests (`get_capabilities`, `set_capabilities`, `configure_server`, `unconfigure_server`). The **client** emits it as a reply to server-initiated requests (`ask_user_confirm`, `get_client_status`). Always correlate responses using `requestId`.

### Socket.IO Built-in Events

| Event Name | Description |
|---------|------|
| `connect` | Connection successful |
| `disconnect` | Connection disconnected |
| `reconnect` | Reconnection successful |
| `connect_error` | Connection error (e.g., authentication failed) |

---

## Notification Types

Predefined notification type constants (`NotificationData.Type`):

```go
const (
	NotificationTypeUserDisabled       = "user_disabled"        // User disabled by admin
	NotificationTypePermissionChanged  = "permission_changed"   // User permissions updated
	NotificationTypeOnlineSessions     = "online_sessions"      // Online session list changed
	NotificationTypeServerStatusChange = "server_status_change" // MCP server status changed
	NotificationTypeUserExpired        = "user_expired"         // User authorization expired
)
```

#### `server_status_change` Data

Emitted when an MCP server transitions between statuses (Online/Offline/Error/Sleeping). Only sent to users who have access to the server.

```javascript
{
  type: 'server_status_change',
  message: 'Server MyServer status changed',
  data: {
    serverId: 'server-abc',
    serverName: 'MyServer',
    oldStatus: 1,  // ServerStatusOffline
    newStatus: 0   // ServerStatusOnline
  },
  timestamp: 1736935200000,
  severity: 'info'
}
```

> **Note**: The `type` field is a free-form string. Clients should handle unknown type values gracefully. Additional types may be introduced in the future without corresponding constant definitions.

---

## Data Formats

### SocketRequest[T]

```go
type SocketRequest[T any] struct {
	RequestID string           `json:"requestId"`
	Action    SocketActionType `json:"action"`
	Data      T                `json:"data"`
	Timestamp int64            `json:"timestamp"`
}
```

### SocketResponse[T]

```go
type SocketResponse[T any] struct {
	RequestID string               `json:"requestId"`
	Success   bool                 `json:"success"`
	Data      *T                   `json:"data,omitempty"`
	Error     *SocketResponseError `json:"error,omitempty"`
	Timestamp int64                `json:"timestamp"`
}
```

### SocketResponseError

```go
type SocketResponseError struct {
	Code    SocketErrorCode `json:"code"`
	Message string          `json:"message"`
	Details any             `json:"details,omitempty"`
}
```

### ClientInfo

```go
type ClientInfo struct {
	DeviceType string `json:"deviceType,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
	AppVersion string `json:"appVersion,omitempty"`
	Platform   string `json:"platform,omitempty"`
}
```

### NotificationData

```go
type NotificationData struct {
	Type      string      `json:"type"`
	Message   string      `json:"message"`
	Timestamp int64       `json:"timestamp"`
	Data      any         `json:"data,omitempty"`
	Severity  string      `json:"severity,omitempty"` // info, warning, error, success
}
```

### UserConnection

```go
type UserConnection struct {
	UserID      string    `json:"userId"`
	SocketID    string    `json:"socketId"`
	DeviceType  string    `json:"deviceType,omitempty"`
	DeviceName  string    `json:"deviceName,omitempty"`
	AppVersion  string    `json:"appVersion,omitempty"`
	ConnectedAt time.Time `json:"connectedAt"`
}
```

### OnlineSessionData

Online session entries are returned as `[]map[string]any` (not a named Go struct). Each entry has the following shape:

```json
{
  "sessionId":  "string",
  "clientName": "string",
  "userAgent":  "string",
  "lastActive": "2025-01-15T10:00:00Z"
}
```

> `lastActive` is a `time.Time` value serialized as an RFC 3339 / ISO 8601 string.

---

## Authentication Mechanism

### Server Authentication Flow

1. Client carries Token in `auth.token` or `Authorization` header when connecting
2. Server calls token validation during handshake
3. If validation fails, throws authentication error and disconnects
4. If validation succeeds, gets authentication context, stores `userId` in socket data
5. Join socket to Room named with `userId`

### Client Authentication Methods

**Recommended Method** (using `auth` object):

```javascript
const socket = io('http://localhost:3002', {
  auth: {
    token: 'your-access-token-here'
  }
});
```

**Alternative Method** (using `extraHeaders`):

```javascript
const socket = io('http://localhost:3002', {
  extraHeaders: {
    Authorization: 'Bearer your-access-token-here'
  }
});
```

---

## Health Check

Access `/health` endpoint to view Socket.IO status:

```bash
curl http://localhost:3002/health
```

Response example:

```json
{
  "status": "healthy",
  "timestamp": "2025-01-15T10:00:00.000Z",
  "uptime": 3600,
  "sessions": {
    "active": 5,
    "total": 10
  },
  "socketio": {
    "onlineUsers": 3,
    "totalConnections": 5
  },
  "servers": { ... },
  "memory": { ... }
}
```

---

## Testing

### Using Postman or curl to Test Server Push

Since Socket.IO requires WebSocket connection, recommend using the following tools for testing:

1. **Socket.IO Client Testing Tool**: https://socket.io/docs/v4/testing/
2. **Chrome Extension**: Socket.IO Tester
3. **Go test command**: Run integration and smoke checks from this repository

### Quick Test Commands

```bash
# Verify service is healthy
curl http://localhost:3002/health

# Run Go test suite
go test ./...
```

---

## Troubleshooting

### Connection Failure

1. **Check if server is running**: Access `http://localhost:3002/health`
2. **Check if Token is valid**: Ensure Token is not expired and user is not disabled
3. **View server logs**: Server will output detailed authentication failure reasons
4. **Check firewall**: Ensure port 3002 is accessible

### Authentication Failure

Server logs will show specific errors:

```
❌ Socket authentication failed: User not found (type: USER_NOT_FOUND)
❌ Socket authentication failed: User is Disabled (type: USER_DISABLED)
❌ Socket authentication failed: User authorization has expired (type: USER_EXPIRED)
```

### Notification Not Received

1. **Check if user is online**: Use `socket.GetSocketNotifier().IsUserOnline(userId)`
2. **Check if userId is correct**: Ensure using correct user ID
3. **View server logs**: Confirm if notification was sent successfully

---

## Performance Considerations

- **Connection Limit**: Default unlimited, production environment recommend configuring limits and timeouts
- **Memory Management**: Socket service uses maps to store connection mappings, automatically cleans up disconnected connections
- **Log Output**: Production environment can adjust log level to reduce output

---

## Security Recommendations

1. **Token Protection**: Clients should not hardcode Token in code
2. **CORS Configuration**: Production environment should configure specific `origin`, do not use `*`
3. **Rate Limiting**: Recommend adding rate limiting for Socket.IO events
4. **Message Validation**: Server should validate message format sent by clients

---

## Extended Features (Future)

The following features can be implemented in future versions:

- [ ] Offline message queue
- [ ] Redis Adapter (multi-server cluster)
- [ ] Limit maximum connections per user
- [ ] Token refresh mechanism
- [ ] Message persistence
- [ ] Heartbeat detection optimization
- [ ] Custom event permission control

---

## References

- **Socket.IO Official Documentation**: https://socket.io/docs/v4/
- **Socket.IO Client API**: https://socket.io/docs/v4/client-api/
- **Electron Integration**: https://www.electronjs.org/docs/latest/tutorial/notifications

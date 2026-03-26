# Kimbap Core API Overview

This document provides an overview and navigation for all APIs in Kimbap Core.

> **Status note:** The active connected-mode HTTP surface is implemented in `internal/api/routes.go` and is mounted under `/v1/*` (plus optional `/console/*`).
> OAuth and other sections in this file are retained for historical context and may not reflect the current build.

## Table of Contents

- [Authentication](#authentication)
- [API Categories](#api-categories)
  - [OAuth 2.0 Authentication](#1-oauth-20-authentication)
    - [OAuth Client Management (Admin)](#oauth-client-management-admin)
  - [Admin API](#2-admin-api)
  - [User API](#3-user-api)
- [Error Handling](#error-handling)
- [Complete Examples](#complete-examples)

---

## Authentication

Kimbap Core uses two kinds of bearer tokens:

- **OAuth 2.0 access tokens (JWT)** issued by Kimbap Core: accepted by Kimbap Core API endpoints that require OAuth bearer authentication.
- **Kimbap access tokens (opaque)** associated with a user: used by `/admin` and `/user`.

**Get an OAuth token**: Obtain an OAuth 2.0 access token through the OAuth endpoints. See [OAuth 2.0 Authentication](#1-oauth-20-authentication) for details.

---

## API Categories

### 1. OAuth 2.0 Authentication

Kimbap Core exposes an OAuth 2.0 authorization server for obtaining access tokens used to authenticate to Kimbap Core.

These endpoints are separate from downstream connector OAuth tokens used by downstream MCP servers to access third-party APIs. Those credentials are brokered internally by Kimbap Core and are not exposed here.

#### Endpoint List

| Endpoint | Description |
|------|------|
| `GET /.well-known/oauth-authorization-server` | OAuth Authorization Server Metadata (RFC 8414) |
| `GET /.well-known/oauth-protected-resource` | OAuth Protected Resource Metadata (RFC 9728) |
| `POST /register` | Dynamic client registration |
| `POST /token` | Get or refresh access token |
| `GET /authorize` | User authorization page for authorization code flow |
| `POST /introspect` | Check token validity |
| `POST /revoke` | Revoke token |

#### Dynamic Client Registration

```bash
curl -X POST http://localhost:3002/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "my-client",
    "redirect_uris": ["http://localhost:3000/callback"],
    "token_endpoint_auth_method": "none"
  }'
```

If you provide `grant_types` in client metadata, Kimbap Core accepts `authorization_code`, `refresh_token`, and `client_credentials` (for compatibility). The `/token` endpoint currently supports `authorization_code` and `refresh_token` grants only.

#### Supported Grant Types

##### 1. Authorization Code Grant with PKCE (Web/Mobile Apps)

**Step 1**: Generate PKCE parameters

```bash
CODE_VERIFIER=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-43)
CODE_CHALLENGE=$(echo -n $CODE_VERIFIER | openssl dgst -sha256 -binary | base64 | tr -d "=+/" | cut -c1-43)
```

**Step 2**: Get authorization code (open in browser)

```
http://localhost:3002/authorize?
  client_id=your_client_id&
  response_type=code&
  redirect_uri=http://localhost:3000/callback&
  code_challenge=$CODE_CHALLENGE&
  code_challenge_method=S256
```

**Step 3**: Exchange authorization code for token

```bash
curl -X POST http://localhost:3002/token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "authorization_code",
    "code": "authorization_code",
    "client_id": "your_client_id",
    "redirect_uri": "http://localhost:3000/callback",
    "code_verifier": "'$CODE_VERIFIER'"
  }'
```

##### 2. Refresh Token

```bash
curl -X POST http://localhost:3002/token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "refresh_token",
    "refresh_token": "your_refresh_token",
    "client_id": "your_client_id",
    "client_secret": "your_client_secret"
  }'
```

#### Token Introspection

```bash
curl -X POST http://localhost:3002/introspect \
  -H "Content-Type: application/json" \
  -d '{
    "token": "YOUR_OAUTH_ACCESS_TOKEN",
    "token_type_hint": "access_token"
  }'
```

#### OAuth Client Management (Admin)

These endpoints allow administrators to manage OAuth clients. All require admin authentication (Owner or Admin role).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/oauth/admin/clients` | List all registered OAuth clients |
| `GET` | `/oauth/admin/clients/{clientId}` | Get details of a specific client |
| `PUT` | `/oauth/admin/clients/{clientId}` | Update client metadata |
| `DELETE` | `/oauth/admin/clients/{clientId}` | Delete a client |

**Authentication**: Bearer token (Kimbap access token), Owner or Admin role required.

**Example: List clients**

```bash
curl http://localhost:3002/oauth/admin/clients \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

**Example: Delete client**

```bash
curl -X DELETE http://localhost:3002/oauth/admin/clients/client_abc123 \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

**Error responses** follow the standard OAuth error format (`error` + `error_description`).

---

### 2. Admin API

Admin API provides user management, server configuration, IP whitelist, log querying, and other functions.

**Complete Documentation**: 📚 **[ADMIN_API.md](./ADMIN_API.md)**

#### Core Features

| Category | Operations | Permission Required |
|------|---------|---------|
| **User Management** | Create, query, update, delete users | Owner/Admin |
| **Server Management** | Configure downstream MCP servers | Owner/Admin |
| **Capability Configuration** | Manage tool/resource/prompt permissions | Owner/Admin |
| **IP Whitelist** | IP access control | Owner/Admin |
| **Proxy Management** | Proxy configuration and control | Owner/Admin |
| **Backup & Restore** | Database backup and restore | Owner/Admin |
| **Log Management** | Query audit logs | Owner |
| **Cloudflared** | Manage Cloudflare Tunnel | Owner/Admin |

#### Unified Request Format

All admin requests use a **single endpoint** `POST /admin`, distinguished by the `action` field:

```go
type AdminRequest struct {
	Action int         `json:"action"`
	Data   interface{} `json:"data"`
}
```

**Example**:
```bash
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_KIMBAP_ACCESS_TOKEN" \
  -d '{
    "action": 1011,
    "data": { "proxyId": 0 }
  }'
```

#### Quick Reference

| Operation | Action | Description |
|------|--------|------|
| Get User List | `1011` | GET_USERS |
| Create User | `1010` | CREATE_USER |
| Update User Permissions | `1002` | UPDATE_USER_PERMISSIONS |
| Disable User | `1001` | DISABLE_USER |
| Get Server List | `2011` | GET_SERVERS |
| Start Server | `2001` | START_SERVER |
| Get Server Status | `3004` | GET_SERVERS_STATUS |
| Get IP Whitelist | `4002` | GET_IP_WHITELIST |
| Update IP Whitelist | `4001` | UPDATE_IP_WHITELIST |

**Detailed Documentation**: See [ADMIN_API.md](./ADMIN_API.md) for all available admin operations.

---

### 3. User API

User API provides user-facing operations for capability management, server configuration, and session queries.

**Complete Documentation**: 📚 **[USER_API.md](./USER_API.md)**

#### Core Features

| Category | Operations | Permission Required |
|------|---------|---------|
| **Capability Management** | Get/Set user capability preferences | Valid User Token |
| **Server Configuration** | Configure/Unconfigure user-specific servers | Valid User Token |
| **Session Queries** | Get online sessions | Valid User Token |

**Key Features**:
- ✅ Action-based routing (same pattern as Admin API)
- ✅ HTTP-based API surface
- ✅ No role checking (any valid user can access)
- ✅ Shared business logic across Admin/User API handlers
- ✅ Real-time capability updates

#### Unified Request Format

All user requests use a **single endpoint** `POST /user`, distinguished by the `action` field:

```go
type UserRequest struct {
	Action int         `json:"action"`
	Data   interface{} `json:"data,omitempty"`
}
```

**Example**:
```bash
curl -X POST http://localhost:3002/user \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "action": 1001
  }'
```

#### Quick Reference

| Operation | Action | Description |
|------|--------|------|
| Get Capabilities | `1001` | GET_CAPABILITIES |
| Set Capabilities | `1002` | SET_CAPABILITIES |
| Configure Server | `2001` | CONFIGURE_SERVER |
| Unconfigure Server | `2002` | UNCONFIGURE_SERVER |
| Get Online Sessions | `3001` | GET_ONLINE_SESSIONS |

**Detailed Documentation**: See [USER_API.md](./USER_API.md) for all 5 user operations.

---


## Error Handling

### HTTP Status Codes

| Status Code | Description |
|--------|------|
| `200` | Success |
| `400` | Bad Request |
| `401` | Unauthorized (Token invalid or expired) |
| `403` | Forbidden |
| `404` | Not Found |
| `429` | Too Many Requests (Rate limit) |
| `500` | Internal Server Error |

### Standard Error Responses

#### MCP Protocol Error (JSON-RPC 2.0)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid Request",
    "data": {
      "details": "Missing required parameter"
    }
  }
}
```

**Error Codes**:
- `-32700` - Parse error
- `-32600` - Invalid Request
- `-32601` - Method not found
- `-32602` - Invalid params
- `-32603` - Internal error

#### Admin/User API Error

Admin API and User API both use the same error response format:

```json
{
  "success": false,
  "error": {
    "code": 2001,
    "message": "Server notion not found"
  }
}
```

**Common Error Codes**:
- `1001` - Invalid request
- `1002` - Unauthorized
- `1003` - User disabled / Insufficient permissions
- `2001` - User/Server not found
- `3001` - Server not found / Invalid capabilities
- `5102` - Invalid IP format

See [ADMIN_API.md - Error Code Reference](./ADMIN_API.md#appendix-error-code-reference) for admin error codes.
See [USER_API.md - Error Code Reference](./USER_API.md#appendix-error-code-reference) for user error codes.

#### Authentication Error

```json
{
  "error": "Unauthorized",
  "message": "Invalid or expired token",
  "code": "AUTH_INVALID_TOKEN"
}
```

#### Rate Limit Error

```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded",
  "retryAfter": 60,
  "code": "RATE_LIMIT_EXCEEDED"
}
```

---

## Complete Examples

### Admin API Example

```bash
# Get all users
curl -X POST http://localhost:3002/admin \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "action": 1011,
    "data": { "proxyId": 0 }
  }'

# Get all server status
curl -X POST http://localhost:3002/admin \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "action": 3004,
    "data": {}
  }'
```

### User API Example

```bash
# Get user's capability configuration
curl -X POST http://localhost:3002/user \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "action": 1001
  }'

# Configure a user-specific server
curl -X POST http://localhost:3002/user \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "action": 2001,
    "data": {
      "serverId": "notion",
      "authConf": [
        {
          "key": "{{NOTION_API_KEY}}",
          "value": "secret_xxx",
          "dataType": 1
        }
      ]
    }
  }'
```


## Related Documentation

- **[ADMIN_API.md](./ADMIN_API.md)** - Complete Admin API protocol documentation
- **[USER_API.md](./USER_API.md)** - Complete User API protocol documentation
- **[OAuth 2.0 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749)** - OAuth 2.0 Authorization Framework
- **[CLAUDE.md](../../CLAUDE.md)** - Project architecture and development guide

---

**Version**: 2.0
**Last Updated**: 2026-02-17
**Change Notes**: Updated to match the current Go implementation and route behavior

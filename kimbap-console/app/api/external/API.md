# KIMBAP Console External REST API Documentation

## Overview

This document describes the external REST API for KIMBAP Console.

**Base URL**: `/api/external`

**Method**: All 47 external API endpoints use `POST`.

**Authentication**: Most endpoints require Bearer token authentication. The token must be an Owner token (`role=1`). Admin and Member tokens cannot access the external API.

```http
Authorization: Bearer <owner_access_token>
```

**Public endpoints (no authentication required)**:
- `POST /auth/init`
- `POST /proxy`

**Authentication Error Codes**:

| Code | HTTP | Description |
|------|------|-------------|
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

**Content-Type**: All requests with a body must include `Content-Type: application/json`.

**Response Envelope**: Every response uses a consistent envelope:
- Success: `{ "success": true, "data": { ... } }`
- Error: `{ "success": false, "error": { "code": "E1001", "message": "..." } }`

---

## Authentication

### Initialize Proxy and Create Owner Token

Initialize the KIMBAP MCP proxy system and create an owner token.

**Endpoint**: `POST /api/external/auth/init`

**Request Body**:
```json
{
  "masterPwd": "string"
}
```

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "accessToken": "a1b2c3d4e5f6...128-char-hex-string...",
    "proxyId": 1,
    "proxyName": "My MCP Server",
    "proxyKey": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "role": 1,
    "userid": "string"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: masterPwd |
| E3007 | 409 | Proxy has already been initialized |
| E5001 | 500 | Database error |

---

## Proxy

### Get Proxy Info

Get current proxy server information.

**Endpoint**: `POST /api/external/proxy`

**Request Body**: None required (empty object `{}` is acceptable)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "proxyId": 1,
    "proxyKey": "550e8400e29b41d4a716446655440000",
    "proxyName": "My MCP Server",
    "createdAt": 1234567890,
    "fingerprint": "abc123..."
  }
}
```

**Notes**:
- This endpoint implicitly validates KIMBAP Core connectivity (if unreachable, returns error)

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E3001 | 404 | Proxy not found |

### Update Proxy

Update proxy configuration.

**Endpoint**: `POST /api/external/proxy/update`

**Request Body**:
```json
{
  "proxyId": 1,
  "proxyName": "New Server Name"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "proxyId": 1,
    "proxyName": "New Server Name"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: proxyId / proxyName |
| E1003 | 400 | Invalid field value: proxyId |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

---

## Access Tokens

### List Tokens

List all tokens for the proxy, with optional filtering by namespace and/or tags.

**Endpoint**: `POST /api/external/tokens`

**Request Body** (optional — empty body or `{}` returns all tokens):
```json
{
  "namespace": "production",
  "tags": ["team-a", "backend"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | No | Filter by exact namespace match (normalized to lowercase, trimmed) |
| `tags` | string[] | No | Filter by tags with AND semantics — token must have all specified tags (normalized to lowercase, trimmed) |

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "tokens": [
      {
        "tokenId": "user-id",
        "name": "Developer Token",
        "role": 3,
        "notes": "",
        "lastUsed": 0,
        "createdAt": 1234567890,
        "expiresAt": 0,
        "rateLimit": 10,
        "namespace": "default",
        "tags": [],
        "permissions": [
          {
            "toolId": "server-id",
            "toolName": "GitHub MCP",
            "functions": [
              {
                "funcName": "create_issue",
                "enabled": true
              }
            ],
            "resources": [
              {
                "uri": "repo://owner/repo",
                "enabled": true
              }
            ]
          }
        ]
      }
    ]
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Invalid request body (malformed JSON) |
| E1003 | 400 | Invalid field value (wrong type for namespace or tags) |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

### Get Token

Get one token by ID.

**Endpoint**: `POST /api/external/tokens/get`

**Request Body**:
```json
{
  "tokenId": "user-id"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "tokenId": "user-id",
    "name": "Developer Token",
    "role": 3,
    "notes": "",
    "lastUsed": 0,
    "createdAt": 1234567890,
    "expiresAt": 0,
    "rateLimit": 10,
    "namespace": "default",
    "tags": [],
    "permissions": []
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: tokenId |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3003 | 404 | Token not found: {tokenId} |

### Batch Create Tokens

Create tokens in batch.

**Endpoint**: `POST /api/external/tokens/create`

**Request Body**:
```json
{
  "tokens": [
    {
      "name": "Developer Token",
      "role": 3,
      "notes": "Optional",
      "expiresAt": 0,
      "rateLimit": 10,
      "namespace": "default",
      "tags": ["tag-1"],
      "permissions": [
        {
          "toolId": "server-id",
          "functions": [
            {
              "funcName": "create_issue",
              "enabled": true
            }
          ],
          "resources": [
            {
              "uri": "repo://owner/repo",
              "enabled": true
            }
          ]
        }
      ]
    }
  ]
}
```

**Fields** (per token object):

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | — | Display name for the token |
| `role` | number | Yes | — | `2` = admin, `3` = member (owner `1` is rejected) |
| `notes` | string | No | `""` | Optional notes |
| `expiresAt` | number | No | `0` | Unix timestamp; `0` means no expiration |
| `rateLimit` | number | No | `10` | Requests per second |
| `namespace` | string | No | `"default"` | Namespace for the token |
| `tags` | string[] | No | `[]` | Arbitrary tags |
| `permissions` | object[] | No | `[]` | Tool permission grants (see token object in List Tokens) |

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "tokens": [
      {
        "tokenId": "user-id",
        "accessToken": "generated-token",
        "name": "Developer Token",
        "role": 3,
        "createdAt": 1234567890,
        "namespace": "default",
        "tags": []
      }
    ]
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: tokens / tokens[i].name / tokens[i].role |
| E1003 | 400 | Invalid field value (role/metadata/permissions) |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E4002 | 422 | Token limit reached / Token limit exceeded |
| E4007 | 422 | Cannot create owner role |

### Update Token

Update an existing token.

**Endpoint**: `POST /api/external/tokens/update`

**Request Body** (`tokenId` required, other fields optional):
```json
{
  "tokenId": "user-id",
  "name": "Updated Name",
  "notes": "Updated notes",
  "namespace": "updated-namespace",
  "tags": ["tag-1"],
  "permissions": [
    {
      "toolId": "server-id",
      "functions": [
        {
          "funcName": "create_issue",
          "enabled": true
        }
      ],
      "resources": [
        {
          "uri": "repo://owner/repo",
          "enabled": true
        }
      ]
    }
  ]
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `tokenId` | string | Yes | — | ID of the token to update |
| `name` | string | No | — | New display name |
| `notes` | string | No | — | Updated notes |
| `namespace` | string | No | — | Updated namespace |
| `tags` | string[] | No | — | Replaces existing tags |
| `permissions` | object[] | No | — | Replaces existing permissions |

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "tokenId": "user-id",
    "message": "Token updated successfully"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: tokenId |
| E1003 | 400 | Invalid field value (metadata/permissions) |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API / cannot modify owner token |
| E3003 | 404 | Token not found: {tokenId} |

### Delete Token

Delete an existing token.

**Endpoint**: `POST /api/external/tokens/delete`

**Request Body**:
```json
{
  "tokenId": "user-id"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "tokenId": "user-id",
    "message": "Token deleted successfully"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: tokenId |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3003 | 404 | Token not found: {tokenId} |
| E4006 | 422 | Cannot delete owner token |

### Batch Update Tokens

Bulk update token permissions and/or metadata.

**Endpoint**: `POST /api/external/tokens/batch-update`

**Request Body**:
```json
{
  "tokenIds": ["user-id-1", "user-id-2"],
  "permissionsMode": "replace",
  "permissions": [
    {
      "toolId": "server-id",
      "functions": [
        {
          "funcName": "create_issue",
          "enabled": true
        }
      ],
      "resources": [
        {
          "uri": "repo://owner/repo",
          "enabled": true
        }
      ]
    }
  ],
  "namespace": "namespace",
  "tagsMode": "add",
  "tags": ["tag-1"]
}
```

**Fields**:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `tokenIds` | string[] | Yes | — | IDs of tokens to update |
| `permissionsMode` | string | No | — | `replace` or `merge` |
| `permissions` | object[] | No | — | Permission grants to apply |
| `namespace` | string | No | — | New namespace value |
| `tagsMode` | string | No | — | `replace`, `add`, `remove`, or `clear` |
| `tags` | string[] | No | — | Tags to apply (behavior depends on `tagsMode`) |

At least one of permissions update or metadata update must be provided. Operation is non-atomic; per-token failures are returned in `failures`.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "updatedCount": 1,
    "failedCount": 1,
    "failures": [
      {
        "tokenId": "user-id-2",
        "error": "Cannot bulk update owner token"
      }
    ]
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field (tokenIds/tags/permissions,namespace,tags) |
| E1003 | 400 | Invalid field value |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

---

## Tool Templates

### List Tool Templates

Get available tool templates.

**Endpoint**: `POST /api/external/templates`

**Request Body**: None required (empty object `{}` is acceptable)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "templates": [
      {
        "toolTmplId": "github-mcp",
        "name": "GitHub MCP",
        "description": "GitHub integration",
        "mcpJsonConf": {}
      }
    ]
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

---

## Tools (MCP Servers)

### List Tools

List configured tools.

**Endpoint**: `POST /api/external/tools`

**Request Body** (optional):
```json
{
  "enabled": true
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "tools": [
      {
        "toolId": "server-id",
        "toolTmplId": "template-id",
        "name": "GitHub MCP",
        "description": "",
        "enabled": true,
        "runningState": 0,
        "functions": [
          {
            "funcName": "create_issue",
            "enabled": true,
            "dangerLevel": 0,
            "description": ""
          }
        ],
        "resources": [
          {
            "uri": "repo://owner/repo",
            "enabled": true
          }
        ],
        "lazyStartEnabled": true,
        "publicAccess": false,
        "lastUsed": 0,
        "createdAt": 1234567890
      }
    ]
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

### Get Tool

Get one tool by ID.

**Endpoint**: `POST /api/external/tools/get`

**Request Body**:
```json
{
  "toolId": "server-id"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id",
    "toolTmplId": "template-id",
    "name": "GitHub MCP",
    "description": "",
    "enabled": true,
    "runningState": 0,
    "functions": [],
    "resources": [],
    "lazyStartEnabled": true,
    "publicAccess": false,
    "allowUserInput": false,
    "createdAt": 1234567890
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3002 | 404 | Tool not found: {toolId} |

### Create Basic MCP Tool

**Endpoint**: `POST /api/external/tools/basic-mcp`

**Request Body**:
```json
{
  "toolTmplId": "github-mcp",
  "mcpJsonConf": {},
  "allowUserInput": false,
  "lazyStartEnabled": true,
  "publicAccess": false
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `toolTmplId` | string | Yes | — | Template ID from `/templates` |
| `mcpJsonConf` | object | Yes | — | MCP server configuration (template-specific) |
| `allowUserInput` | boolean | No | `false` | Allow end-users to provide input |
| `lazyStartEnabled` | boolean | No | `true` | Start server on first use instead of immediately |
| `publicAccess` | boolean | No | `false` | Allow unauthenticated access |

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id",
    "isStarted": true
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolTmplId / mcpJsonConf |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3004 | 404 | Template not found: {toolTmplId} |
| E4001 | 422 | Tool limit reached |

### Update Basic MCP Tool

Update a basic MCP tool's configuration.

**Endpoint**: `POST /api/external/tools/basic-mcp/update`

**Request Body** (`toolId` required, all other fields optional):
```json
{
  "toolId": "550e8400-e29b-41d4-a716-446655440000",
  "mcpJsonConf": {
    "command": "npx",
    "args": ["-y", "@anthropic/github-mcp"],
    "env": {
      "GITHUB_TOKEN": "ghp_new_token"
    }
  },
  "functions": [
    {
      "funcName": "create_issue",
      "enabled": true,
      "dangerLevel": 1,
      "description": "Create a GitHub issue"
    }
  ],
  "resources": [
    {
      "uri": "repo://owner/repo",
      "enabled": true
    }
  ],
  "lazyStartEnabled": true,
  "publicAccess": false
}
```

**Notes**:
- `toolId` (required): The tool ID to update
- `mcpJsonConf` (optional): Update MCP server launch configuration (same format as create endpoint)
  - **Restriction**: Can only be modified when BOTH conditions are met:
    1. `authType === ApiKey` (tool uses API Key authentication)
    2. `allowUserInput === false` (not a personal tool template)
  - If either condition is not met, the update will be rejected
  - This is a **full replacement** of the launch configuration
- `functions` (optional): Update enabled functions and their settings
  - `funcName`: Function name
  - `enabled`: Whether the function is enabled
  - `dangerLevel`: Danger level (0=safe, 1=moderate, 2=dangerous)
  - `description`: Function description
- `resources` (optional): Update enabled resources
  - `uri`: Resource URI
  - `enabled`: Whether the resource is enabled
- `lazyStartEnabled` (optional): Whether to enable lazy start
- `publicAccess` (optional): Whether the tool is available to all users by default

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId |
| E1003 | 400 | mcpJsonConf cannot be modified (authType is not ApiKey or allowUserInput is true) |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3002 | 404 | Tool not found: {toolId} |

### Create Custom MCP Tool

**Endpoint**: `POST /api/external/tools/custom-mcp`

**Request Body** (provide either `customRemoteConfig` or `stdioConfig`, not both):

*URL-based MCP server:*
```json
{
  "customRemoteConfig": {
    "url": "https://example.com/sse",
    "headers": {
      "Authorization": "Bearer xxx"
    }
  },
  "allowUserInput": false,
  "lazyStartEnabled": true,
  "publicAccess": false
}
```

*Stdio-based MCP server:*
```json
{
  "stdioConfig": {
    "command": "npx",
    "cwd": "/path/to/dir",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"],
    "env": {
      "NODE_ENV": "production"
    }
  },
  "allowUserInput": false,
  "lazyStartEnabled": true,
  "publicAccess": false
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `customRemoteConfig` | object | One of `customRemoteConfig` or `stdioConfig` | — | Remote URL MCP server config |
| `customRemoteConfig.url` | string | Yes (when using customRemoteConfig) | — | SSE endpoint URL |
| `customRemoteConfig.headers` | object | No | `{}` | HTTP headers to send with requests |
| `stdioConfig` | object | One of `customRemoteConfig` or `stdioConfig` | — | Stdio MCP server config |
| `stdioConfig.command` | string | Yes (when using stdioConfig) | — | Executable command to start the server |
| `stdioConfig.cwd` | string | No | inherited | Working directory used when spawning the process (trimmed; empty string ignored) |
| `stdioConfig.args` | string[] | No | `[]` | Command-line arguments |
| `stdioConfig.env` | object | No | `{}` | Environment variables (string values only) |
| `allowUserInput` | boolean | No | `false` | Allow end-users to provide input |
| `lazyStartEnabled` | boolean | No | `true` | Start server on first use instead of immediately |
| `publicAccess` | boolean | No | `false` | Allow unauthenticated access |

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id",
    "isStarted": true
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: customRemoteConfig or stdioConfig |
| E1001 | 400 | Cannot provide both stdioConfig and customRemoteConfig |
| E1005 | 400 | Invalid URL format |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E4001 | 422 | Tool limit reached |

### Update Custom MCP Tool

Update a custom MCP tool's configuration.

**Endpoint**: `POST /api/external/tools/custom-mcp/update`

**Request Body** (`toolId` required, all other fields optional):

*Update URL-based tool:*
```json
{
  "toolId": "550e8400-e29b-41d4-a716-446655440000",
  "customRemoteConfig": {
    "url": "https://my-mcp-server.com/sse?param=value",
    "headers": {
      "Authorization": "Bearer new_token"
    }
  },
  "functions": [...],
  "resources": [...],
  "lazyStartEnabled": true,
  "publicAccess": false
}
```

*Update stdio-based tool:*
```json
{
  "toolId": "550e8400-e29b-41d4-a716-446655440000",
  "stdioConfig": {
    "command": "npx",
    "cwd": "/new/working/dir",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/new/path"],
    "env": {
      "NODE_ENV": "production"
    }
  },
  "functions": [...],
  "resources": [...],
  "lazyStartEnabled": true
}
```

**Notes**:
- `toolId` (required): The tool ID to update
- `customRemoteConfig` (optional): Update URL-based MCP configuration
  - `url` (required when customRemoteConfig provided): The URL of the remote MCP server
    - **For personal tools (`allowUserInput=true`)**: Only query parameters can be changed. The base URL path (before `?`) cannot be modified.
    - **For non-personal tools**: The full URL can be changed.
  - `headers` (optional): HTTP headers to include in requests
- `stdioConfig` (optional): Update stdio-based MCP configuration
  - `command` (required when stdioConfig provided): The executable command
    - **For personal tools (`allowUserInput=true`)**: The command cannot be changed.
  - `cwd` (optional): Working directory used when spawning the process
  - `args` (optional): Command-line arguments as string array
  - `env` (optional): Environment variables as object with string values
- Cannot provide both `customRemoteConfig` and `stdioConfig` in the same request
- Cannot switch transport type (URL ↔ stdio) on an existing tool. Delete and recreate instead.
- `functions` (optional): Update enabled functions and their settings
- `resources` (optional): Update enabled resources
- `lazyStartEnabled` (optional): Whether to enable lazy start
- `publicAccess` (optional): Whether the tool is available to all users by default

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId / stdioConfig.command |
| E1003 | 400 | Cannot provide both stdioConfig and customRemoteConfig |
| E1003 | 400 | Cannot switch from stdio to URL transport (or vice versa) |
| E1003 | 400 | Command cannot be changed for personal stdio tools |
| E1003 | 400 | URL cannot be changed for personal tools |
| E1003 | 400 | stdioConfig.args must be an array of strings |
| E1003 | 400 | stdioConfig.env must be an object with string values |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3002 | 404 | Tool not found: {toolId} |

### Create REST API Tool

**Endpoint**: `POST /api/external/tools/rest-api`

**Request Body**:
```json
{
  "restApiConfig": {
    "name": "My REST API",
    "baseUrl": "https://api.example.com",
    "auth": {
      "type": "bearer",
      "token": "token"
    },
    "tools": [
      {
        "name": "getUsers",
        "method": "GET",
        "path": "/users"
      }
    ]
  },
  "allowUserInput": false,
  "lazyStartEnabled": true,
  "publicAccess": false
}
```

**`restApiConfig` fields**:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `restApiConfig.name` | string | Yes | — | Display name for this REST API tool |
| `restApiConfig.baseUrl` | string | Yes | — | Base URL for all API calls |
| `restApiConfig.auth.type` | string | No | `"none"` | `none`, `bearer`, `basic`, or `api_key` |
| `restApiConfig.auth.token` | string | No | — | Token value (for `bearer` or `api_key`) |
| `restApiConfig.tools` | object[] | Yes | — | List of API endpoints to expose as tools |
| `allowUserInput` | boolean | No | `false` | Allow end-users to provide input |
| `lazyStartEnabled` | boolean | No | `true` | Start server on first use instead of immediately |
| `publicAccess` | boolean | No | `false` | Allow unauthenticated access |

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id",
    "isStarted": true
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: restApiConfig |
| E1003 | 400 | Invalid field value: baseUrl/tools/tool item validation |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E4001 | 422 | Tool limit reached |

### Update REST API Tool

Update a REST API tool's configuration.

**Endpoint**: `POST /api/external/tools/rest-api/update`

**Request Body** (`toolId` required, all other fields optional):
```json
{
  "toolId": "550e8400-e29b-41d4-a716-446655440000",
  "restApiConfig": {
    "name": "Updated REST API",
    "baseUrl": "https://api.example.com",
    "auth": {
      "type": "bearer",
      "token": "new-api-token"
    },
    "tools": [
      {
        "name": "getUsers",
        "description": "Get all users",
        "method": "GET",
        "path": "/users"
      }
    ]
  },
  "functions": [
    {
      "funcName": "getUsers",
      "enabled": true,
      "dangerLevel": 0,
      "description": "Get all users"
    }
  ],
  "resources": [
    {
      "uri": "resource://path",
      "enabled": true
    }
  ],
  "lazyStartEnabled": true,
  "publicAccess": false
}
```

**Notes**:
- `toolId` (required): The tool ID to update
- `restApiConfig` (optional): Update REST API configuration
  - `name` (optional): Update tool name (will add " Personal" suffix if `allowUserInput=true`)
  - `baseUrl` (required when restApiConfig provided): The base URL for the REST API
    - **For personal tools (`allowUserInput=true`)**: Base URL cannot be changed from the original value
    - **For non-personal tools**: Base URL can be changed
  - `auth` (optional): Update authentication configuration
  - `tools` (required when restApiConfig provided): Array of API endpoint definitions (at least one)
- `functions` (optional): Update enabled functions and their settings
- `resources` (optional): Update enabled resources
- `lazyStartEnabled` (optional): Whether to enable lazy start
- `publicAccess` (optional): Whether the tool is available to all users by default

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId |
| E1003 | 400 | At least one tool is required for REST API tools |
| E1003 | 400 | Base URL is required / Base URL cannot be changed (for personal tools) |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3002 | 404 | Tool not found: {toolId} |

### Create Skills Tool

**Endpoint**: `POST /api/external/tools/skills`

**Request Body**:
```json
{
  "serverName": "My Skills Server",
  "allowUserInput": false,
  "lazyStartEnabled": true,
  "publicAccess": false
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `serverName` | string | No | — | Display name for the skills server |
| `allowUserInput` | boolean | No | `false` | Allow end-users to provide input |
| `lazyStartEnabled` | boolean | No | `true` | Start server on first use instead of immediately |
| `publicAccess` | boolean | No | `false` | Allow unauthenticated access |

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id",
    "isStarted": true
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E4001 | 422 | Tool limit reached |
### Update Skills Tool

**Endpoint**: `POST /api/external/tools/skills/update`

**Request Body**:
```json
{
  "toolId": "server-id",
  "authConfig": {},
  "functions": [],
  "resources": [],
  "lazyStartEnabled": true,
  "publicAccess": false,
  "allowUserInput": false
}
```
**Fields** (`toolId` required, all others optional):

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `toolId` | string | Yes | — | ID of the tool to update |
| `authConfig` | object | No | — | Auth config (encrypted before storage) |
| `functions` | object[] | No | — | Function capability overrides |
| `resources` | object[] | No | — | Resource capability overrides |
| `lazyStartEnabled` | boolean | No | `true` | Start server on first use instead of immediately |
| `publicAccess` | boolean | No | `false` | Allow unauthenticated access |
| `allowUserInput` | boolean | No | `false` | Allow end-users to provide input |

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId |
| E1003 | 400 | Invalid field value |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3002 | 404 | Tool not found: {toolId} |

### Delete Tool

**Endpoint**: `POST /api/external/tools/delete`

**Request Body**:
```json
{
  "toolId": "server-id"
}
```

**Behavior**:
- Validates `toolId`, checks tool exists.
- Attempts `stopMCPServer` (failure ignored).
- Deletes server.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "550e8400-e29b-41d4-a716-446655440000",
    "message": "Tool deleted successfully"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3002 | 404 | Tool not found: {toolId} |

### Enable Tool

**Endpoint**: `POST /api/external/tools/enable`

**Request Body**:
```json
{
  "toolId": "server-id",
  "masterPwd": "string"
}
```

**Behavior**:
- Validates input and tool existence.
- Fails if already enabled.
- Decrypts owner token using `masterPwd`.
- Starts MCP server.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id",
    "isStarted": true
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId / masterPwd |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E2004 | 401 | Invalid master password |
| E3002 | 404 | Tool not found: {toolId} |
| E3008 | 409 | Tool already enabled |
| E4008 | 400 | Tool start failed |

### Disable Tool

**Endpoint**: `POST /api/external/tools/disable`

**Request Body**:
```json
{
  "toolId": "server-id",
  "masterPwd": "string"
}
```

**Behavior**:
- Validates input and tool existence.
- Fails if already disabled.
- Decrypts owner token using `masterPwd`.
- Stops MCP server.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId / masterPwd |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E2004 | 401 | Invalid master password |
| E3002 | 404 | Tool not found: {toolId} |
| E3009 | 409 | Tool already disabled |

### Start Tool

**Endpoint**: `POST /api/external/tools/start`

**Request Body**:
```json
{
  "toolId": "server-id",
  "masterPwd": "string"
}
```

**Behavior**:
- Validates input and tool existence.
- Decrypts owner token using `masterPwd`.
- Starts MCP server.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "toolId": "server-id",
    "isStarted": true
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: toolId / masterPwd |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E2004 | 401 | Invalid master password |
| E3002 | 404 | Tool not found: {toolId} |
| E4008 | 400 | Tool start failed |

### Connect All Tools

**Endpoint**: `POST /api/external/tools/connect-all`

**Request Body**: None required (empty object `{}` is acceptable)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "successServers": ["server-id-1", "server-id-2"],
    "failedServers": ["server-id-3"]
  }
}
```

(`successServers` and `failedServers` arrays contain server IDs or server objects, depending on the proxy backend response.)
**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

---

## Scopes

### Get Available Scopes

Get all available tools with function/resource capability maps.

**Endpoint**: `POST /api/external/scopes`

**Request Body**: None required (empty object `{}` is acceptable)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "scopes": [
      {
        "toolId": "server-id",
        "name": "GitHub MCP",
        "enabled": true,
        "functions": [
          {
            "funcName": "create_issue",
            "enabled": true
          }
        ],
        "resources": [
          {
            "uri": "repo://owner/repo",
            "enabled": true
          }
        ]
      }
    ]
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

---

## IP Whitelist

### Get IP Whitelist

**Endpoint**: `POST /api/external/ip-whitelist`

**Request Body**: None required (empty object `{}` is acceptable)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "allowAll": false,
    "ips": ["192.168.1.0/24", "10.0.0.1"]
  }
}
```

(`data` contains `allowAll` (boolean, true when all IPs are allowed) and `ips` (array of whitelisted IP/CIDR strings, excluding the allow-all sentinel).)

### Add IPs to Whitelist

**Endpoint**: `POST /api/external/ip-whitelist/add`

**Request Body**:
```json
{
  "ips": ["192.168.1.0/24", "10.0.0.1"]
}
```

**Validation**:
- `ips` is required.
- Must be a non-empty `string[]`.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "addedIds": [1, 2],
    "addedCount": 2,
    "skippedCount": 0,
    "message": "Successfully added 2 IPs"
  }
}
```

(`data` contains the result of `addIpWhitelist(...)`: `addedIds` (array of DB record IDs), `addedCount`, `skippedCount`, and `message`.)

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: ips |
| E1003 | 400 | Invalid field value: ips |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

### Allow All IPs

**Endpoint**: `POST /api/external/ip-whitelist/allow-all`

**Request Body**: None required (empty object `{}` is acceptable)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "success": true
  }
}
```

### Restrict to Whitelist

**Endpoint**: `POST /api/external/ip-whitelist/restrict`

**Request Body**: None required (empty object `{}` is acceptable)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "success": true
  }
}
```

### Delete IPs from Whitelist

**Endpoint**: `POST /api/external/ip-whitelist/delete`

**Request Body**:
```json
{
  "ips": ["192.168.1.0/24", "10.0.0.1"]
}
```

**Validation**:
- `ips` is required.
- Must be a non-empty `string[]`.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "deletedCount": 2,
    "message": "Successfully deleted 2 IPs"
  }
}
```

(`data` contains the result of `deleteIpWhitelist(...)`: `deletedCount` and `message`.)

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: ips |
| E1003 | 400 | Invalid field value: ips |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

---

## KIMBAP Core

### Get KIMBAP Core Connection

**Endpoint**: `POST /api/external/kimbap-core/connect`

**Request Body**: None required (empty object `{}` is acceptable)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "host": "kimbap-core.example.com",
    "port": 443,
    "connected": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `host` | string | KIMBAP Core host address |
| `port` | number \| null | KIMBAP Core port (`null` if not configured) |
| `connected` | boolean | Always `true` — indicates KIMBAP Core is configured in the database, not that a live network connection was verified |

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E4014 | 422 | KIMBAP Core not configured |

---

## User Capabilities

### Get User Capabilities

**Endpoint**: `POST /api/external/user-capabilities`

**Request Body**:
```json
{
  "userId": "user-id"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "capabilities": {
      "server-id": {
        "tools": {
          "create_issue": {
            "enabled": true
          }
        },
        "resources": {
          "repo://owner/repo": {
            "enabled": true
          }
        }
      }
    }
  }
}
```

(`capabilities` is returned directly from `getUserAvailableServersCapabilities(...)`.)

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: userId |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

### Set User Capabilities

**Endpoint**: `POST /api/external/user-capabilities/set`

**Request Body**:
```json
{
  "userId": "user-id",
  "permissions": [
    {
      "toolId": "server-id",
      "functions": [
        {
          "funcName": "create_issue",
          "enabled": true
        }
      ],
      "resources": [
        {
          "uri": "repo://owner/repo",
          "enabled": true
        }
      ]
    }
  ]
}
```
**Fields**:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `userId` | string | Yes | — | ID of the user to update |
| `permissions` | object[] | Yes | — | Array of tool permission grants |
| `permissions[].toolId` | string | Yes | — | Tool to grant access to |
| `permissions[].functions` | object[] | No | `[]` | Function-level grants (`funcName`, `enabled`) |
| `permissions[].resources` | object[] | No | `[]` | Resource-level grants (`uri`, `enabled`) |

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "userId": "user-id"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: userId / permissions |
| E1003 | 400 | Invalid permission/function/resource format |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

---

## User Servers

### Assign Server to User

**Endpoint**: `POST /api/external/user-servers/configure`

**Request Body**:
```json
{
  "userId": "user-id",
  "serverId": "server-id"
}
```

**Behavior**:
- Validates request.
- Loads user.
- Parses `serverApiKeys`.
- Adds `serverId` if missing.
- Updates user.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "userId": "user-id",
    "serverId": "server-id"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: userId / serverId |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3002 | 404 | User not found: {userId} |

### Remove Server from User

**Endpoint**: `POST /api/external/user-servers/unconfigure`

**Request Body**:
```json
{
  "userId": "user-id",
  "serverId": "server-id"
}
```

**Behavior**:
- Validates request.
- Loads user.
- Parses `serverApiKeys`.
- Removes `serverId`.
- Updates user.

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "userId": "user-id",
    "serverId": "server-id"
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E1001 | 400 | Missing required field: userId / serverId |
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |
| E3002 | 404 | User not found: {userId} |

---

## Sessions

### Get Online Sessions

Get recently active sessions from the last 5 minutes.

**Endpoint**: `POST /api/external/sessions`

**Request Body** (optional):
```json
{
  "limit": 50
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `limit` | number | No | `50` | Max sessions to return (max `200`) |

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "sessions": [
      {
        "sessionId": "session-id",
        "userId": "user-id",
        "serverId": "server-id",
        "lastActivity": 1234567890,
        "ip": "127.0.0.1"
      }
    ],
    "count": 1
  }
}
```

**Error Responses**:

| Code | HTTP | Message |
|------|------|---------|
| E2001 | 401 | Access token is required |
| E2002 | 401 | Invalid access token |
| E2003 | 403 | Permission denied: only owner can access external API |

---

## Policies

### List Policy Sets

List policy sets, optionally filtered by `serverId`.

**Endpoint**: `POST /api/external/policies`

**Request Body** (optional):
```json
{
  "serverId": "optional-server-id"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "policySets": [
      {
        "id": "cm123...",
        "serverId": "server-1",
        "version": 3,
        "status": "active",
        "dsl": {
          "schemaVersion": 1,
          "rules": []
        },
        "createdAt": "2026-02-27T09:00:00.000Z",
        "updatedAt": "2026-02-27T09:10:00.000Z"
      }
    ]
  }
}
```

### Get Policy Set

Get a policy set by ID.

**Endpoint**: `POST /api/external/policies/get`

**Request Body**:
```json
{
  "id": "cm123..."
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "id": "cm123...",
    "serverId": "server-1",
    "version": 3,
    "status": "active",
    "dsl": {
      "schemaVersion": 1,
      "rules": []
    },
    "createdAt": "2026-02-27T09:00:00.000Z",
    "updatedAt": "2026-02-27T09:10:00.000Z"
  }
}
```

### Create Policy Set

Create a new policy set.

**Endpoint**: `POST /api/external/policies/create`

**Request Body**:
```json
{
  "serverId": "optional-server-id",
  "dsl": {
    "schemaVersion": 1,
    "rules": []
  }
}
```

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "id": "cm123...",
    "serverId": "server-1",
    "version": 4,
    "status": "active",
    "dsl": {
      "schemaVersion": 1,
      "rules": []
    },
    "createdAt": "2026-02-27T09:20:00.000Z",
    "updatedAt": "2026-02-27T09:20:00.000Z"
  }
}
```

### Update Policy Set

Update policy set DSL and/or status.

**Endpoint**: `POST /api/external/policies/update`

**Request Body**:
```json
{
  "id": "cm123...",
  "dsl": {
    "schemaVersion": 1,
    "rules": []
  },
  "status": "active"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "id": "cm123...",
    "serverId": "server-1",
    "version": 5,
    "status": "active",
    "dsl": {
      "schemaVersion": 1,
      "rules": []
    },
    "createdAt": "2026-02-27T09:00:00.000Z",
    "updatedAt": "2026-02-27T09:30:00.000Z"
  }
}
```

### Delete Policy Set

Delete a policy set by ID.

**Endpoint**: `POST /api/external/policies/delete`

**Request Body**:
```json
{
  "id": "cm123..."
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "id": "cm123...",
    "serverId": "server-1",
    "version": 5,
    "status": "active",
    "dsl": {
      "schemaVersion": 1,
      "rules": []
    },
    "createdAt": "2026-02-27T09:00:00.000Z",
    "updatedAt": "2026-02-27T09:30:00.000Z"
  }
}
```

### Get Effective Policy

Get only active policy sets that apply to a given server (`server-specific + global`).

**Endpoint**: `POST /api/external/policies/effective`

**Request Body** (optional):
```json
{
  "serverId": "optional-server-id"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "policySets": [
      {
        "id": "cm123...",
        "serverId": "server-1",
        "version": 5,
        "status": "active",
        "dsl": {
          "schemaVersion": 1,
          "rules": []
        },
        "createdAt": "2026-02-27T09:00:00.000Z",
        "updatedAt": "2026-02-27T09:30:00.000Z"
      }
    ]
  }
}
```

---

## Approvals

### List Approval Requests

List approval requests with optional filters and pagination.

**Endpoint**: `POST /api/external/approvals`

**Request Body** (optional):
```json
{
  "userId": "optional-user-id",
  "serverId": "optional-server-id",
  "toolName": "optional-tool-name",
  "status": "optional-status",
  "page": 1,
  "pageSize": 20
}
```

**Response** `200 OK` (returns Core admin approval objects as-is):
```json
{
  "success": true,
  "data": {
    "page": 1,
    "pageSize": 20,
    "hasMore": true,
    "requests": [
      {
        "id": "cm234...",
        "userId": "user-1",
        "serverId": "server-1",
        "toolName": "github.create_issue",
        "status": "PENDING"
      }
    ]
  }
}
```

### Get Approval Request

Get one approval request by ID.

**Endpoint**: `POST /api/external/approvals/get`

**Request Body**:
```json
{
  "id": "cm234..."
}
```

**Response** `200 OK` (returns Core admin approval object as-is):
```json
{
  "success": true,
  "data": {
    "id": "cm234...",
    "userId": "user-1",
    "serverId": "server-1",
    "toolName": "github.create_issue",
    "status": "PENDING"
  }
}
```

### Decide Approval Request

Approve or reject an approval request.

**Endpoint**: `POST /api/external/approvals/decide`

**Request Body**:
```json
{
  "id": "cm234...",
  "decision": "APPROVED",
  "reason": "Looks safe"
}
```

**Response** `200 OK` (returns Core admin approval object as-is):
```json
{
  "success": true,
  "data": {
    "id": "cm234...",
    "status": "APPROVED",
    "decisionReason": "Looks safe"
  }
}
```

### Count Pending Approvals

Count pending approvals. If `userId` is omitted, it counts pending approvals across all users.

**Endpoint**: `POST /api/external/approvals/count-pending`

**Request Body** (optional):
```json
{
  "userId": "optional-user-id"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "count": 12
  }
}
```

---

## Error Codes

All error responses follow this format:

```json
{
  "success": false,
  "error": {
    "code": "E1001",
    "message": "Error description"
  }
}
```

### Error Code Categories

| Range | Category | Description |
|-------|----------|-------------|
| E1xxx | Validation Errors | Request parameter validation failures |
| E2xxx | Authentication Errors | Authentication and authorization failures |
| E3xxx | Resource Errors | Resource not found or state errors |
| E4xxx | Business Logic Errors | Business rule violations |
| E5xxx | System Errors | Internal server errors |

---

### Validation Errors (E1xxx)

| Code | Message | Description |
|------|---------|-------------|
| E1001 | Missing required field: {field} | A required field is not provided |
| E1003 | Invalid field value: {field} | Field value is invalid or out of range |
| E1004 | Invalid JSON format | Request body is not valid JSON |
| E1005 | Invalid URL format | URL field is not a valid URL |

### Authentication Errors (E2xxx)

| Code | HTTP | Message | Description |
|------|------|---------|-------------|
| E2001 | 401 | Access token is required | No authorization header provided |
| E2002 | 401 | Invalid access token | Token format or token value is invalid |
| E2003 | 403 | Permission denied | Only owner token can access external API |
| E2004 | 401 | Invalid master password | Master password verification failed |
| E2005 | 403 | Insufficient permissions | User role cannot perform this action |
| E2006 | 403 | Owner permission required | Only owner can perform this action |
| E2007 | 403 | Admin or owner permission required | Member cannot perform this action |
| E2008 | 429 | Rate limit exceeded | Too many requests |

### Resource Errors (E3xxx)

| Code | HTTP | Message | Description |
|------|------|---------|-------------|
| E3001 | 404 | Proxy not found | No proxy has been initialized |
| E3002 | 404 | Tool not found: {toolId} | Target tool/user not found in endpoint-specific context |
| E3003 | 404 | Token not found: {tokenId} | Specified token does not exist |
| E3004 | 404 | Template not found: {templateId} | Tool template does not exist |
| E3007 | 409 | Proxy already initialized | Cannot initialize an already initialized proxy |
| E3008 | 409 | Tool already enabled | Tool is already enabled |
| E3009 | 409 | Tool already disabled | Tool is already disabled |

### Business Logic Errors (E4xxx)

| Code | HTTP | Message | Description |
|------|------|---------|-------------|
| E4001 | 422 | Tool limit reached | License tool quota exceeded |
| E4002 | 422 | Token limit reached | License token quota exceeded |
| E4006 | 422 | Cannot delete owner token | Owner token cannot be deleted |
| E4007 | 422 | Cannot create owner role | Owner creation blocked |
| E4008 | 400 | Tool start failed | MCP server failed to start |
| E4014 | 422 | KIMBAP Core not available | KIMBAP Core config/availability check failed |

### System Errors (E5xxx)

| Code | HTTP | Message | Description |
|------|------|---------|-------------|
| E5001 | 500 | Database error | Database operation failed |
| E5002 | 500 | Encryption error | Encryption/decryption operation failed |
| E5003 | 500 | Internal server error | Unexpected server error |
| E5004 | 500 | External service error | External service call failed |
| E5005 | 503 | Service temporarily unavailable | Service unavailable |

---

## API Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /auth/init | Initialize proxy and create owner token |
| POST | /proxy | Get proxy info |
| POST | /proxy/update | Update proxy config |
| POST | /tokens | List all tokens |
| POST | /tokens/get | Get token details |
| POST | /tokens/create | Batch create tokens |
| POST | /tokens/update | Update token |
| POST | /tokens/delete | Delete token |
| POST | /tokens/batch-update | Batch update tokens |
| POST | /templates | List tool templates |
| POST | /tools | List all tools |
| POST | /tools/get | Get tool details |
| POST | /tools/basic-mcp | Create basic MCP tool |
| POST | /tools/basic-mcp/update | Update basic MCP tool |
| POST | /tools/custom-mcp | Create custom MCP tool |
| POST | /tools/custom-mcp/update | Update custom MCP tool |
| POST | /tools/rest-api | Create REST API tool |
| POST | /tools/rest-api/update | Update REST API tool |
| POST | /tools/skills | Create skills tool |
| POST | /tools/skills/update | Update skills tool |
| POST | /tools/delete | Delete tool |
| POST | /tools/enable | Enable tool |
| POST | /tools/disable | Disable tool |
| POST | /tools/start | Start tool |
| POST | /tools/connect-all | Connect all tools |
| POST | /scopes | Get available scopes |
| POST | /ip-whitelist | Get IP whitelist |
| POST | /ip-whitelist/add | Add IPs to whitelist |
| POST | /ip-whitelist/allow-all | Allow all IPs |
| POST | /ip-whitelist/restrict | Restrict to whitelist |
| POST | /ip-whitelist/delete | Remove IPs from whitelist |
| POST | /kimbap-core/connect | Get KIMBAP Core connection |
| POST | /user-capabilities | Get user capabilities |
| POST | /user-capabilities/set | Set user capabilities |
| POST | /user-servers/configure | Assign server to user |
| POST | /user-servers/unconfigure | Remove server from user |
| POST | /sessions | Get online sessions |
| POST | /policies | List policy sets |
| POST | /policies/get | Get policy set by ID |
| POST | /policies/create | Create policy set |
| POST | /policies/update | Update policy set |
| POST | /policies/delete | Delete policy set |
| POST | /policies/effective | Get effective policy sets |
| POST | /approvals | List approval requests |
| POST | /approvals/get | Get approval request by ID |
| POST | /approvals/decide | Approve/reject approval request |
| POST | /approvals/count-pending | Count pending approvals |

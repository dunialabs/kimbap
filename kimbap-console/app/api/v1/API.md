# KIMBAP Console V1 Protocol API Documentation

## Overview

This document describes the internal protocol-style API exposed by KIMBAP Console at `/api/v1`.

Unlike the external REST API under `/api/external`, the v1 API uses a single POST endpoint and routes requests by `cmdId`.

**Base URL**: `/api/v1`

**Method**: All requests use `POST`.

**Runtime**:
- Node.js runtime
- Dynamic responses (`force-dynamic`)

## Request Format

All requests must use the following JSON envelope:

```json
{
  "common": {
    "cmdId": 10001
  },
  "params": {}
}
```

Notes:
- `common.cmdId` is required.
- For authenticated endpoints, the server injects `common.userid` based on the Bearer token.
- `params` must be a JSON object, even when empty.

## Authentication

The `/api/v1` router treats the following `cmdId` values as public:

- `10001` Initialize proxy
- `10002` Get proxy info
- `10015` Login
- `10021` Auto-detect KIMBAP Core
- `10022` Validate and save KIMBAP Core host

All other protocols require:

```http
Authorization: Bearer <access_token>
```

If the token matches a record in the local `user` table, the request is authenticated and `common.userid` is set automatically.

## Response Envelope

All responses use the same wrapper:

Success:

```json
{
  "common": {
    "code": 0,
    "message": "Success",
    "cmdId": 10001
  },
  "data": {}
}
```

Error:

```json
{
  "common": {
    "code": 400,
    "message": "Invalid or missing parameter: masterPwd",
    "cmdId": 10001,
    "errorCode": "ERR_3000"
  }
}
```

Notes:
- `common.code = 0` means success.
- On error, `common.code` is the HTTP status code.
- `common.errorCode` is the internal application error code such as `ERR_3000`.

## Common Error Codes

| Error Code | HTTP | Meaning |
|------|------|-------------|
| `ERR_1000` | 400 | Missing `cmdId` |
| `ERR_1001` | 404 | Protocol not implemented |
| `ERR_1002` | 400 | Invalid request |
| `ERR_1003` | 500 | Internal server error |
| `ERR_2000` | 400 | Master password required |
| `ERR_2001` | 401 | Invalid master password |
| `ERR_2002` | 401 | Unauthorized |
| `ERR_2004` | 401 | Token expired |
| `ERR_2005` | 401 | Invalid token |
| `ERR_2006` | 403 | User disabled |
| `ERR_3000` | 400 | Missing required field |
| `ERR_3001` | 400 | Invalid field format |
| `ERR_3004` | 400 | Invalid parameters |
| `ERR_4001` | 404 | Record not found |
| `ERR_5000` | 409 | Proxy already initialized |
| `ERR_5001` | 404 | Server not found |
| `ERR_5002` | 404 | User not found |
| `ERR_5003` | 403 | Permission denied |
| `ERR_6000` | 400 | License activation failed |
| `ERR_6004` | 400 | Tool creation limit exceeded |
| `ERR_6005` | 400 | Access token limit exceeded |
| `ERR_7000` | 500 | KIMBAP Core config not found |

## Generic Example

```bash
curl -X POST http://localhost:3000/api/v1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{
    "common": {
      "cmdId": 10023
    },
    "params": {
      "timeRange": "30d"
    }
  }'
```

---

## Core Management Protocols

### Protocol 10001 - Initialize Proxy

Creates the first proxy, owner token, and local user record.

**Auth**: Public

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `masterPwd` | string | Yes | Initial master password |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `accessToken` | string | Generated owner access token |
| `proxyId` | number | Created proxy ID |
| `proxyName` | string | Initial name, currently `My MCP Server` |
| `proxyKey` | string | Generated proxy key |
| `role` | number | Owner role, always `1` |
| `userid` | string | Calculated owner user ID |

**Notes**:
- Fails with `ERR_5000` if a proxy already exists.
- Also creates a local `user` table entry for login validation.

### Protocol 10002 - Get Proxy Info

Returns the current proxy metadata.

**Auth**: Public

**Request Params**: none

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `proxyId` | number | Proxy ID |
| `proxyKey` | string | Proxy key |
| `proxyName` | string | Proxy name |
| `status` | number | `1` running, `2` stopped |
| `createdAt` | number | Unix timestamp |
| `fingerprint` | string | Hardware fingerprint |

**Notes**:
- Current implementation hardcodes status to `1`.

### Protocol 10003 - Update Proxy / Reset Server

Handles proxy base-info changes and full server reset.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `handleType` | number | Yes | `1` edit base info, `8` reset server |
| `proxyId` | number | Yes | Target proxy ID |
| `proxyName` | string | No | New proxy name for `handleType=1` |
| `masterPwd` | string | No | Required for `handleType=8` |
| `proxyKey` | string | No | Optional compatibility check during reset |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `success` | boolean | Operation result |
| `message` | string | Optional status message |

**Notes**:
- Reset deletes tunnel records, logs, licenses, and proxy data.
- Reset validates the owner token using `masterPwd`.

### Protocol 10004 - List Tool Templates

Fetches tool templates from the Kimbap Cloud API.

**Auth**: Bearer token required

**Request Params**: none

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `toolTmplList` | array | Cloud template list |

**Notes**:
- Falls back to an empty list on non-fatal fetch failures.

### Protocol 10005 - Tool Management

Creates, edits, enables, disables, deletes, or starts tools.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `handleType` | number | Yes | `1` add, `2` edit, `3` enable, `4` disable, `5` delete, `6` start |
| `proxyId` | number | Conditional | Required for add, disable, and start; also used for owner lookup |
| `toolId` | string | Conditional | Required for edit, enable, disable, delete, start |
| `toolTmplId` | string | Conditional | Required when adding a template-based tool |
| `masterPwd` | string | Conditional | Required for encrypted launch config, disable, and start |
| `allowUserInput` | number | No | `1` enables user input passthrough |
| `category` | number | No | `1` template, `2` custom remote HTTP, `3` REST API, `4` skills, `5` custom stdio |
| `serverName` | string | No | Display name, especially for skills tools |
| `customRemoteConfig` | object | No | Used for category `2` |
| `stdioConfig` | object | No | Used for category `5` |
| `restApiConfig` | string | No | JSON string used for category `3` |
| `authConf` | array | No | Authentication config entries |
| `functions` | array | No | Tool function enablement config |
| `resources` | array | No | Tool resource enablement config |
| `lazyStartEnabled` | boolean | No | Enables lazy start |
| `publicAccess` | boolean | No | Enables public access |
| `anonymousAccess` | boolean | No | Enables anonymous access |
| `anonymousRateLimit` | number | No | Anonymous requests per minute per IP |

**Response Data**:
- Varies by operation.
- Add returns the new server information.
- Enable/disable/start/delete return success-oriented payloads.

**Notes**:
- Enforces license-based tool creation limits.
- Supports encrypted launch configuration using the owner token.
- Category-specific validation is strict; invalid combinations return `ERR_1002` or `ERR_3000`.

### Protocol 10006 - List Tools

Returns configured tools with capability and launch-config-derived metadata.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `handleType` | number | Yes | `1` all tools, `2` enabled tools |
| `proxyId` | number | No | Optional filter |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `toolList` | array | Tools with template metadata, capabilities, runtime state, access flags, and per-category config |

Each tool item includes:
- `toolId`, `name`, `description`, `toolTmplId`
- `toolFuncs`, `toolResources`
- `enabled`, `runningState`, `allowUserInput`
- `category`, `authType`, `lazyStartEnabled`
- `publicAccess`, `anonymousAccess`, `anonymousRateLimit`
- `restApiConfig`, `customRemoteConfig`, `stdioConfig` when applicable

### Protocol 10007 - List Access Tokens

Lists users/tokens for a proxy and expands their current tool capabilities.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `proxyId` | number | Yes | Proxy filter |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `tokenList` | array | Token list |

Each token includes:
- `tokenId`, `name`, `role`, `notes`
- `createAt`, `expireAt`, `rateLimit`
- `namespace`, `tags`
- `toolList` with `toolId`, `name`, `toolFuncs`, `toolResources`, `enabled`

### Protocol 10008 - Maintain Access Tokens

Creates, edits, deletes, or batch-updates access tokens.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `handleType` | number | Yes | `1` add, `2` edit, `3` delete, `4` batch update |
| `userid` | string | Conditional | Required for edit and delete |
| `userids` | string[] | Conditional | Required for batch update |
| `name` | string | Conditional | Required for add |
| `role` | number | Conditional | Required for add |
| `expireAt` | number | No | Expiration timestamp |
| `rateLimit` | number | No | Per-token rate limit |
| `notes` | string | No | Token notes |
| `permissions` | array | No | Tool-scoped permissions |
| `permissionsMode` | string | No | Batch permission update mode |
| `tagsMode` | string | No | Batch tag update mode |
| `namespace` | string | No | Token namespace |
| `tags` | string[] | No | Token tags |
| `masterPwd` | string | Conditional | Required for add |
| `proxyId` | number | Conditional | Required for add and batch update |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `accessToken` | string | Newly generated token for add, empty otherwise |
| `updatedCount` | number | Batch update success count |
| `failedCount` | number | Batch update failure count |
| `failures` | array | Per-user batch update failure details |

**Notes**:
- Add validates the owner token using `masterPwd`.
- Metadata validation covers namespace and tags.
- Batch update supports permission and tag merge modes.

### Protocol 10009 - List Available Scopes

Returns enabled functions and resources from available servers.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `proxyId` | number | Yes | Proxy filter |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `scopes` | array | Server-scoped capabilities |

Each item includes `toolId`, `name`, `enabled`, `toolFuncs`, and `toolResources`.

### Protocol 10010 - Get Tool Capabilities

Returns detailed capabilities for a single tool.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `toolId` | string | Yes | Server ID |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `functions` | array | Function list with `funcName`, `enabled`, `dangerLevel`, `description` |
| `resources` | array | Resource list with `uri`, `enabled` |

### Protocol 10011 - Operate DNS Records / Remote Access

Starts and stops remote access, or maintains manual DNS entries.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `handleType` | number | Yes | `1` start remote access, `2` stop remote access, `3` add manual DNS, `4` delete manual DNS |
| `domain` | string | Conditional | Required for `handleType=3` |
| `publicIP` | string | Conditional | Required for `handleType=3` |
| `recordId` | number | Conditional | Required for `handleType=4` |
| `proxyKey` | string | No | Optional proxy-key consistency check |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `subdomain` | string | Returned when remote access is started |

**Notes**:
- Remote access uses Kimbap Cloud tunnel provisioning plus proxy-side cloudflared management.
- Manual DNS entries are stored locally in `dns_conf` with `type=2`.

### Protocol 10012 - Get IP Whitelist

Fetches IP whitelist entries from the proxy layer.

**Auth**: Bearer token required

**Request Params**: none

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `ipList` | array | Array of `{ id, ip }` |

**Notes**:
- IDs are generated from array position, not stable database IDs.

### Protocol 10013 - Maintain IP Whitelist

Adds IPs, toggles allow-all, or attempts deletion by ID.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `handleType` | number | Yes | `1` add IPs, `2` allow all, `3` deny all, `4` delete by ID |
| `ipList` | string[] | Conditional | Required for `handleType=1` |
| `idList` | number[] | Conditional | Required for `handleType=4` |

**Response Data**: empty object

**Notes**:
- `handleType=4` currently returns `ERR_1001`/501 because the proxy API only supports IP-based deletion.

### Protocol 10014 - Get DNS Configuration

Returns remote-access and manual DNS information.

**Auth**: Bearer token required

**Request Params**: none

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `kimbapSubdomain` | string | Running tunnel subdomain, if any |
| `manualDnsList` | array | Manual DNS records `{ domain, publicIP, recordId }` |
| `localPublicIP` | string | Public IP discovered via external IP services |
| `manualConnection` | string | Saved or inferred KIMBAP Core connection URL |

### Protocol 10015 - Login

Authenticates either by access token or by owner master password.

**Auth**: Public

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `accessToken` | string | Conditional | Plain access token |
| `masterPwd` | string | Conditional | Owner master password |

Rules:
- Exactly one of `accessToken` or `masterPwd` must be provided.

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `tokenInfo.userid` | string | Authenticated user ID |
| `tokenInfo.role` | number | User role |
| `tokenInfo.createAt` | number | Creation timestamp |
| `accessToken` | string | Present when logging in via `masterPwd` |

**Notes**:
- Successful login upserts the local `user` table.
- Owner login may reconnect all servers in the background.

### Protocol 10016 - Backup Server to Local

Exports proxy-side table data and encrypts the backup payload with the provided master password.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `masterPwd` | string | Yes | Used to decrypt the owner token |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `encryptedData` | string | JSON-stringified encrypted backup payload |

**Notes**:
- Backup content includes `proxy`, `server`, `user`, and `ipWhitelist` tables.
- The plaintext structure before encryption is `{ "tables": { ... } }`.
- Fails with `ERR_2001` if `masterPwd` is incorrect.

### Protocol 10017 - Restore Server from Local

Restores a locally exported backup payload and attempts to restart the proxy backend process afterward.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `masterPwd` | string | Yes | Used to validate owner token |
| `encryptedData` | string | Yes | Backup payload |

**Response Data**: empty object

**Notes**:
- Validates backup shape and decrypted content before restore.
- Expected decrypted structure is `{ "tables": { "proxy": [], "server": [], "user": [], "ipWhitelist": [] } }`.
- After restore, the handler tries to restart `proxy-server/index.js`, but restart failure does not roll back restored data.

### Protocol 10018 - Reconnect All Servers

Reconnects configured MCP servers using the owner token.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `masterPwd` | string | Yes | Used to decrypt owner token |

**Response Data**: empty object

**Notes**:
- The proxy API returns per-server success and failure lists internally, but this protocol does not expose them in the response body.

### Protocol 10019 - Activate License

Validates proxy ownership and activates a license string.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `proxyKey` | string | Yes | Current proxy key |
| `licenseStr` | string | Yes | License string |

**Response Data**: empty object

### Protocol 10020 - Get License History

Returns the full local license history, including active, inactive, and expired records.

**Auth**: Bearer token required

**Request Params**: none

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `licenseList` | array | License history list |

Each license item includes:
- `plan`
- `status` where `100` active, `1` inactive, `2` expired
- `expiresAt`, `createdAt`
- `customerEmail`
- `licenseStr`
- `maxToolCreations`, `maxAccessTokens`
- `currentToolCreations`, `currentAccessTokens`

**Notes**:
- Active licenses attempt to include current usage counts from KIMBAP Core.
- Inactive and expired licenses report current usage as `0`.
- Requires the `LICENSE_MASTER_PASSWORD` environment variable to decrypt stored license strings.

### Protocol 10021 - Auto-detect KIMBAP Core

Tries multiple candidate hosts and validates whether they are running KIMBAP Core.

**Auth**: Public

**Request Params**: none

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `isAvailable` | number | `1` available, `2` not started, `3` running but not KIMBAP Core, `4` saved config already exists |
| `kimbapCoreHost` | string | Returned when detected or already configured |
| `kimbapCorePort` | number | Returned when detected or already configured |

**Notes**:
- On success, configuration may be persisted to the local `config` table.

### Protocol 10022 - Validate and Save KIMBAP Core Configuration

Validates a user-provided host and optional port, then stores it in the local config table.

**Auth**: Public

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `host` | string | Yes | Hostname or URL |
| `port` | number | No | Optional port |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `isValid` | number | `1` valid and saved, `2` invalid/unreachable, `3` reachable but not KIMBAP Core |
| `message` | string | Validation result |

### Protocol 10023 - Dashboard Overview

Returns aggregate dashboard cards, usage charts, connected clients, and recent activity.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | string | No | `24h`, `7d`, `30d`, `90d`; defaults to `30d` |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `uptime` | string | Server uptime string |
| `apiRequests` | number | Request count |
| `activeTokens` | number | Active token count |
| `configuredTools` | number | Configured tool count |
| `connectedClientsCount` | number | Connected client total |
| `monthlyUsage` | number | Current-month usage |
| `toolsUsage` | array | Tool usage shares |
| `tokenUsage` | array | Token usage shares |
| `connectedClients` | array | Client details |
| `recentActivity` | array | Recent actions |
| `sshTunnelAddress` | string | Running tunnel subdomain |
| `manualConnection` | string | Saved KIMBAP Core URL |

### Protocol 10024 - Reset Master Password

Re-encrypts the owner token using a new master password.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `accessToken` | string | Yes | Owner access token |
| `masterPwd` | string | Yes | New master password |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `success` | boolean | Reset result |
| `message` | string | Status message |

### Protocol 10025 - Get Operation Limits

Returns current plan and license limits for tools and access tokens.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `proxyKey` | string | No | Optional consistency check |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `isFreeTier` | boolean | Whether current instance is free tier |
| `expiresAt` | number | License expiry timestamp |
| `maxToolCreations` | number | Tool limit |
| `currentToolCount` | number | Current tool count |
| `remainingToolCount` | number | Remaining tool slots |
| `maxAccessTokens` | number | Token limit |
| `currentAccessTokenCount` | number | Current token count |
| `remainingAccessTokenCount` | number | Remaining token slots |
| `licenseKey` | string | Paid license key if available |
| `fingerprintHash` | string | Paid license hardware hash if available |

---

## Skills Management Protocols

### Protocol 10040 - List Skills

Lists installed skills for a skills server.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `serverId` | string | Yes | Skills server ID |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `skills` | array | Skill list with `name`, `description`, `version` |

### Protocol 10041 - Upload Skills Bundle

Uploads a base64-encoded ZIP bundle to a skills server.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `serverId` | string | Yes | Skills server ID |
| `data` | string | Yes | Base64 ZIP content |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `success` | boolean | Upload result |
| `message` | string | Result message |
| `skillName` | string | Optional installed skill name |

### Protocol 10042 - Delete Skill

Deletes a named skill from a skills server.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `serverId` | string | Yes | Skills server ID |
| `skillName` | string | Yes | Skill name |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `success` | boolean | Delete result |
| `message` | string | Result message |

### Protocol 10043 - Delete Server Skills

Deletes all skills for a given skills server.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `serverId` | string | Yes | Skills server ID |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `success` | boolean | Delete result |
| `message` | string | Result message |

---

## Policy and Approval Protocols

### Protocol 10050 - Create Tool Policy

Creates a content-aware policy set.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `serverId` | string | No | Optional server-specific policy target |
| `dsl.rules` | array | Yes | Policy rule definitions |

**Response Data**:
- Policy set object containing `id`, `serverId`, `version`, `status`, `dsl`, `createdAt`, `updatedAt`

### Protocol 10051 - Get Tool Policy

Returns a single policy set by ID.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | Yes | Policy set ID |

**Response Data**:
- Policy set object

### Protocol 10052 - Update Tool Policy

Updates a policy set.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | Yes | Policy set ID |
| `dsl.rules` | array | No | Replacement or updated rules |
| `status` | string | No | Policy status |

**Response Data**:
- Updated policy set object

### Protocol 10053 - Delete Tool Policy

Deletes a policy set by ID.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | Yes | Policy set ID |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `success` | boolean | Delete result |

### Protocol 10054 - List Tool Policies

Lists policy sets, optionally filtered by server.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `serverId` | string | No | Optional server filter |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `policySets` | array | Policy set list |

### Protocol 10055 - List Approval Requests

Lists approval requests generated by policies.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `userId` | string | No | Filter by user |
| `serverId` | string | No | Filter by server |
| `toolName` | string | No | Filter by tool |
| `status` | string | No | Request status |
| `page` | number | No | Defaults to `1` |
| `pageSize` | number | No | Defaults to `20` |

**Response Data**:
- `page`, `pageSize`, `hasMore`
- `requests` array with approval metadata, decision data, execution state, and timestamps

### Protocol 10056 - Get Approval Request

Returns a single approval request.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | Yes | Approval request ID |

**Response Data**:
- Approval request object

### Protocol 10057 - Decide Approval Request

Approves or rejects an approval request.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | Yes | Approval request ID |
| `decision` | string | Yes | `APPROVED` or `REJECTED` |
| `reason` | string | No | Optional decision reason |

**Response Data**:
- Decision result containing `id`, `status`, `decidedAt`, and optional execution fields

### Protocol 10058 - Count Pending Approvals

Counts currently pending approval requests.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `userId` | string | No | Optional filter |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `count` | number | Pending approval count |

### Protocol 10059 - Get Effective Policy

Returns the active policy sets that apply to a server.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `serverId` | string | No | Optional server filter |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `policySets` | array | Effective policy set list |

---

## Tool Usage Statistics Protocols

### Protocol 20001 - Tool Usage Summary

Returns top-level tool usage counters for a time range.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |

**Response Data**:
- `totalTools`, `activeTools`
- `totalRequests`, `successRequests`, `failedRequests`
- `avgSuccessRate`, `avgResponseTime`
- `totalUsers`

### Protocol 20002 - Tool Detailed Metrics

Returns paginated per-tool usage metrics.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `toolIds` | string[] | No | Restrict to specific tools |
| `page` | number | No | Defaults to `1` |
| `pageSize` | number | No | Defaults to `50` |

**Response Data**:
- `tools` array with:
  - `toolId`, `toolName`
  - `totalRequests`, `successfulRequests`, `failedRequests`
  - `averageResponseTime`, `successRate`
  - `lastUsed`, `status`
  - `errorTypes`
- `totalCount`

### Protocol 20003 - Tool Usage Trends

Returns trend points for tools over time.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `7`, `30`, `90` days |
| `toolIds` | string[] | No | Tool filter |
| `granularity` | number | No | `1` hourly, `2` daily, `3` weekly |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `trends` | array | Each point contains `date` plus per-tool request counts |

### Protocol 20004 - Tool Error Analysis

Returns per-tool error distributions.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `serverId` | number | Yes | `0` for all |
| `toolId` | string | Yes | Empty for all |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `toolErrors` | array | Tool-level error totals and error-type breakdowns |

### Protocol 20005 - Tool Usage Distribution

Returns pie-chart style distribution data.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `serverId` | number | Yes | `0` for all |
| `metricType` | number | Yes | `1` requests, `2` users, `3` avg response time |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `distribution` | array | `{ toolId, toolName, value, percentage }` |

### Protocol 20006 - Tool Performance Comparison

Returns bar-chart style comparison data.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `serverId` | number | Yes | `0` for all |
| `metricType` | number | Yes | `1` response time, `2` success rate, `3` request volume |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `comparison` | array | `{ toolId, toolName, avgValue, minValue, maxValue }` |

### Protocol 20007 - User Tool Usage

Returns paginated per-user tool usage.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `userId` | string | Yes | Empty for all users |
| `page` | number | Yes | Page number |
| `pageSize` | number | Yes | Page size |

**Response Data**:
- `userUsage` array with `userId`, `userName`, `role`, `totalRequests`, `toolsUsed`, `topTools`, `lastActive`
- `totalCount`

### Protocol 20008 - Real-time Tool Status

Builds a runtime status view from recent logs.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `serverId` | number | Yes | `0` for all tools |

**Response Data**:
- `toolStatus` array with:
  - `toolId`, `toolName`
  - `status` where `0` online, `1` offline, `2` connecting, `3` error
  - `activeConnections`, `queuedRequests`, `lastHeartbeat`, `errorMessage`

### Protocol 20009 - Export Tool Usage Report

Exports tool-usage logs to a downloadable file.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `serverId` | number | Yes | `0` for all |
| `format` | number | Yes | `1` CSV, `2` JSON, `3` PDF |
| `toolIds` | string[] | Yes | Tool filter when `serverId=0` |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `downloadUrl` | string | Download endpoint under `/api/download/exports/...` |
| `fileName` | string | Export file name |
| `expiresAt` | number | Expiration timestamp |

**Notes**:
- Export is capped at 10,000 records.

---

## Token Usage Statistics Protocols

### Protocol 21001 - Token Usage Summary

Returns top-level token-usage counters for a time range.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |

**Response Data**:
- Summary counters such as active tokens, total requests, success rate, and average response time

### Protocol 21002 - Token Detailed Metrics

Returns paginated per-token metrics.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `tokenIds` | string[] | No | Token filter |
| `page` | number | No | Defaults to `1` |
| `pageSize` | number | No | Defaults to `50` |

**Response Data**:
- `tokens` array with token-level request counts, success rates, status, rate limit info, and last used time
- `totalCount`, `page`, `pageSize`

### Protocol 21003 - Token Usage Trends

Returns trend data across tokens.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `7`, `30`, `90` days |
| `tokenIds` | string[] | No | Token filter |
| `granularity` | number | Yes | `1` hourly, `2` daily, `3` weekly |

**Response Data**:
- `trends` array with `date` plus per-token request counts

### Protocol 21004 - Token Geographic Distribution

Returns request distribution by location for a token or token set.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `tokenId` | string | Yes | Empty for all |

**Response Data**:
- Geographic distribution payload for the frontend map/chart view

### Protocol 21005 - Token Usage Pattern

Returns a pattern analysis for a single token.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `tokenId` | string | Yes | Token user ID |
| `patternType` | number | Yes | `1` last 60 minutes by minute, `2` last 24 hours by hour, `3` last 7 days by hour |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `tokenId` | string | Token user ID |
| `tokenName` | string | Token display name |
| `patterns` | array | Time-bucketed usage points |

Each pattern point includes:
- `timeLabel`
- `requests`
- `successRequests`
- `failedRequests`
- `rateLimitHits`

### Protocol 21006 - Token Usage Distribution

Returns pie-chart style token distribution data.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `metricType` | number | Yes | Distribution metric type |

**Response Data**:
- `distribution` array with token-level values and percentages

### Protocol 21007 - Token Rate Limit Analysis

Returns rate-limit related statistics for tokens.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `tokenId` | string | Yes | Empty for all |

**Response Data**:
- Rate-limit analysis payload

### Protocol 21008 - Token Client Analysis

Returns paginated client/device analysis for a token.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `tokenId` | string | Yes | Empty for all |
| `page` | number | No | Defaults to `1` |
| `pageSize` | number | No | Defaults to `20` |

**Response Data**:
- Client analysis list with pagination metadata

### Protocol 21009 - Export Token Usage Report

Exports token-usage analytics.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | `1`, `7`, `30`, `90` days |
| `format` | number | Yes | `1` CSV, `2` JSON, `3` PDF |
| `tokenIds` | string[] | Yes | Token filter |
| `includeGeoData` | boolean | Yes | Include geographic summary |
| `includeClientData` | boolean | Yes | Include client summary |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `exportUrl` | string | Download URL |
| `filename` | string | Export file name |
| `fileSize` | number | File size in bytes |
| `recordCount` | number | Number of exported token detail rows |
| `format` | string | Human-readable format name |

**Notes**:
- Scan is capped at 20,000 logs.
- Current geo output is placeholder `UNKNOWN` buckets.

### Protocol 21010 - Token Audit Logs

Returns paginated token audit events.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | Supported: `1`, `7`, `30`, `90` |
| `tokenId` | string | No | Optional token filter |
| `eventType` | string | No | Event-type filter |
| `page` | number | No | Defaults to `1` |
| `pageSize` | number | No | Defaults to `20`, max `100` |

**Response Data**:
- Paginated audit event list with summary metadata

### Protocol 21011 - Recent Token Usage Logs

Returns recent log records for the token-usage page, including incremental polling support.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `limit` | number | No | Max rows to return, defaults to `50` |
| `lastId` | number | No | If provided, only rows with larger log IDs are returned |
| `timeRange` | number | No | Optional days filter |
| `userIds` | string[] | No | Restrict to specific user IDs |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `logs` | array | Recent log records |
| `totalCount` | number | Total rows matching the current filter |
| `maxId` | number | Largest log ID in the current batch |

Each log item includes:
- `id`, `addtime`, `timestamp`
- `action`, `actionName`, `source`
- `userid`, `userName`
- `serverId`, `sessionId`
- `ip`, `ua`, `tokenMask`
- `error`, `duration`, `statusCode`
- `requestParams`, `responseResult`

---

## Dashboard Overview Protocols

### Protocol 22001 - Overview KPI Cards

Returns dashboard headline metrics.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | No | Day window, defaults to `1` |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `totalRequests24h` | number | Total requests in the selected current range |
| `requestsChangePercent` | number | Change vs the previous equal-length range |
| `activeTokens` | number | Distinct active tokens in the current range |
| `tokensUsedLastHour` | number | Distinct tokens used in the last hour |
| `toolsInUse` | number | Distinct tools used in the current range |
| `mostActiveToolName` | string | Tool with the most requests |
| `avgResponseTime` | number | Average successful response time in milliseconds |
| `responseTimeChange` | number | Difference vs the previous equal-length range |

**Notes**:
- Despite the field name `totalRequests24h`, the actual window follows the provided `timeRange` value.

### Protocol 22002 - Top Tools by Usage

Returns the most-used tools for the dashboard.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | Days, normalized to at least `1` |
| `limit` | number | Yes | Result count, defaults to `10` |

**Response Data**:
- `tools` array with `toolName`, `toolType`, `requestCount`, `percentage`, `color`
- `totalRequests`

### Protocol 22003 - Active Tokens Overview

Returns currently active or recently active tokens for the dashboard.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | number | Yes | Days, normalized to at least `1` |
| `limit` | number | Yes | Result count, defaults to `5` |

**Response Data**:
- `tokens` array with `tokenName`, `tokenMask`, `requestCount`, `isCurrentlyActive`, `lastUsedMinutesAgo`

### Protocol 22004 - Recent Activity

Returns dashboard-friendly activity feed items derived from recent logs.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `limit` | number | Yes | Defaults to `10` |
| `timeRange` | number | No | Defaults to `1` day |

**Response Data**:
- `activities` array with:
  - `eventType`
  - `description`
  - `details`
  - `timestamp`
  - `icon`
  - `color`

---

## Log Management Protocols

### Protocol 23001 - Query Logs

Returns filtered, paginated logs.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `page` | number | Yes | Page number |
| `pageSize` | number | Yes | Page size |
| `timeRange` | string | Yes | `1h`, `6h`, `24h`, `7d`, `30d`, `all` |
| `level` | string | Yes | `all`, `INFO`, `WARN`, `ERROR`, `DEBUG` |
| `source` | string | Yes | `all` or a domain like `mcp-request`, `lifecycle`, `auth` |
| `search` | string | Yes | Free-text search |
| `requestId` | string | Yes | Session/request ID filter |
| `userId` | string | Yes | User filter |

**Response Data**:
- `logs` array with normalized entries and `details`
- `totalCount`
- `totalPages`
- `availableSources`

### Protocol 23002 - Log Statistics

Returns log counters and time-series aggregations.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | string | Yes | `1h`, `6h`, `24h`, `7d`, `30d`, `all` |

**Response Data**:
- `statistics.totalLogs`, `errorLogs`, `warnLogs`, `infoLogs`, `debugLogs`
- `statistics.errorRate`
- `statistics.domainStats`
- `statistics.hourlyStats`

### Protocol 23004 - Export Logs

Exports logs to TXT, JSON, or CSV.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `timeRange` | string | Yes | `1h`, `6h`, `24h`, `7d`, `30d` |
| `level` | string | Yes | Log level filter |
| `source` | string | Yes | Source filter |
| `search` | string | Yes | Search text |
| `format` | number | Yes | `1` TXT, `2` JSON, `3` CSV |
| `maxRecords` | number | Yes | Max export rows, capped at `10000` |

**Response Data**:

| Field | Type | Description |
|------|------|-------------|
| `downloadUrl` | string | Download URL |
| `fileName` | string | Export file name |
| `fileSize` | number | File size in bytes |
| `recordCount` | number | Exported record count |
| `expiresAt` | number | Expiration timestamp |

### Protocol 23005 - Real-time Logs

Fetches logs newer than the last seen log ID.

**Auth**: Bearer token required

**Request Params**:

| Field | Type | Required | Description |
|------|------|----------|-------------|
| `lastLogId` | number | Yes | Last received log ID |
| `level` | string | Yes | Level filter |
| `source` | string | Yes | Source filter |
| `limit` | number | Yes | Max rows to return |

**Response Data**:
- `newLogs` array
- `latestLogId`
- `hasMore`

---

## Implementation Notes

- The v1 router implementation lives in `app/api/v1/route.ts`.
- Protocol handlers are registered in `app/api/v1/handlers/index.ts`.
- There is currently no implemented `23003` handler.
- Some older handlers still use placeholder names like `Tool <serverId>` where proxy metadata is unavailable.
- Several analytics endpoints build results from local `log` records plus proxy-side metadata from `proxy-api`.

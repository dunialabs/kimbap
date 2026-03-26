# Kimbap REST API

Base URL: `http://localhost:8080/v1`

## Authentication

Protected endpoints require both Bearer authentication and tenant-context resolution (middleware chain: `BearerAuth` + `TenantContext`):

```
Authorization: Bearer <kimbap-token>
```

Create tokens with `kimbap token create` or `POST /v1/tokens`.

Public endpoints (no auth required): `/v1/health`, `/v1/actions`, `/v1/actions/{service}/{action}`.

All other `/v1/*` routes are protected by Bearer + tenant middleware. Management routes add route-specific scope checks.

## Response Format

```json
{
  "success": true,
  "data": { ... },
  "request_id": "req_abc123"
}
```

```json
{
  "success": false,
  "error": {
    "code": "INVALID_INPUT",
    "message": "missing required field: service",
    "retryable": false
  },
  "request_id": "req_abc123"
}
```

## Endpoints

### Health

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/v1/health` | No | Health check |

### Actions

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/v1/actions` | No | List installed actions |
| `GET` | `/v1/actions/{service}/{action}` | No | Describe an action |
| `POST` | `/v1/actions/{service}/{action}:execute` | Yes | Execute an action |
| `POST` | `/v1/actions/validate` | Yes | Validate action input |

### Tokens

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| `POST` | `/v1/tokens` | `tokens:write` | Create token |
| `GET` | `/v1/tokens` | `tokens:read` | List tokens |
| `GET` | `/v1/tokens/{id}` | `tokens:read` | Inspect token |
| `DELETE` | `/v1/tokens/{id}` | `tokens:write` | Revoke token |

### Policies

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| `GET` | `/v1/policies` | `policies:read` | Get policy |
| `PUT` | `/v1/policies` | `policies:write` | Set policy |
| `POST` | `/v1/policies:evaluate` | `policies:read` | Evaluate policy |

### Approvals

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| `GET` | `/v1/approvals` | `approvals:read` | List pending approvals |
| `POST` | `/v1/approvals/{id}:approve` | `approvals:write` | Approve request |
| `POST` | `/v1/approvals/{id}:deny` | `approvals:write` | Deny request |

### Audit

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| `GET` | `/v1/audit` | `audit:read` | Query audit log |
| `GET` | `/v1/audit/export` | `audit:read` | Export audit log |

### Vault

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| `GET` | `/v1/vault` | `vault:read` | List vault keys |

### Webhooks

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| `POST` | `/v1/webhooks` | `webhooks:write` | Register webhook (when webhook dispatcher is configured) |
| `GET` | `/v1/webhooks` | `webhooks:read` | List webhooks (when webhook dispatcher is configured) |
| `DELETE` | `/v1/webhooks/{id}` | `webhooks:write` | Remove webhook (when webhook dispatcher is configured) |
| `GET` | `/v1/webhooks/events` | `webhooks:read` | List recent webhook events (when webhook dispatcher is configured) |

## Examples

### Execute an action

```bash
curl -X POST http://localhost:8080/v1/actions/github/create-issue:execute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"input": {"title": "Bug report", "body": "Details here", "repo": "org/repo"}}'
```

### Create a token

```bash
curl -X POST http://localhost:8080/v1/tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"agent_name": "my-agent", "scopes": ["actions:execute"]}'
```

### List pending approvals

```bash
curl http://localhost:8080/v1/approvals \
  -H "Authorization: Bearer $TOKEN"
```

## HTTP Status Codes

| Status | Meaning |
|--------|---------|
| `200` | Success |
| `400` | Bad request (invalid input) |
| `401` | Unauthorized (missing/invalid token) |
| `403` | Forbidden (insufficient scope) |
| `404` | Not found |
| `409` | Conflict (approval already resolved) |
| `500` | Internal server error |

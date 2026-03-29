# API

Kimbap exposes a REST surface when running `kimbap serve`.

CLI remains the primary interface; the API is primarily for server-mode integrations.

Base path: `/v1`

## Authentication and scopes

- `GET /v1/health`, `GET /v1/actions`, and `GET /v1/actions/{service}/{action}` are public.
- All other `/v1/*` endpoints require `Authorization: Bearer <token>`.
- Access is scope-gated (for example: `actions:execute`, `audit:read`, `tokens:write`, `webhooks:read`).

## Response envelope

All API responses use a common envelope:

- `success` (boolean)
- `data` (present on success responses)
- `error` (present on error responses)
- `request_id` (string)

`data` and `error` are omitted when empty (they are not serialized as `null`).

### Success example

```json
{
  "success": true,
  "data": {"status": "ok"},
  "request_id": "req_01..."
}
```

### Error example

```json
{
  "success": false,
  "error": {
    "code": "ERR_VALIDATION_FAILED",
    "message": "url is required",
    "retryable": false,
    "details": {}
  },
  "request_id": "req_01..."
}
```

## CLI equivalents

| API (`/v1/...`) | CLI (`kimbap ...`) |
|---|---|
| `/v1/actions/{svc}/{action}:execute` | `kimbap call <svc>.<action>` |
| `/v1/actions/validate` | `kimbap call ... --help` / local schema validation flows |
| `/v1/approvals` | `kimbap approve list` |
| `/v1/approvals/{id}:approve` | `kimbap approve accept <id>` |
| `/v1/approvals/{id}:deny` | `kimbap approve deny <id> --reason ...` |
| `/v1/audit` | `kimbap audit tail` / `kimbap audit export` |
| `/v1/policies` | `kimbap policy set` / `kimbap policy get` |
| `/v1/policies:evaluate` | `kimbap policy` evaluation workflows |
| `/v1/vault` | `kimbap vault list` / `kimbap vault set` |
| `/v1/tokens` | `kimbap token ...` |
| `/v1/webhooks` | No direct CLI equivalent (API-only in serve mode) |

## Endpoint index

### Public endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Server health/version |
| `GET` | `/actions` | List actions (supports `namespace`, `resource`, `verb`, `limit`) |
| `GET` | `/actions/{service}/{action}` | Describe one action |

### Authenticated endpoints

| Method | Path | Scope | Description |
|---|---|---|---|
| `POST` | `/actions/{service}/{action}:execute` | `actions:execute` | Execute action with input payload |
| `POST` | `/actions/validate` | token required | Validate input against schema |
| `GET` | `/vault` | `vault:read` | List stored secret metadata |
| `POST` | `/tokens` | `tokens:write` | Create API token |
| `GET` | `/tokens` | `tokens:read` | List tokens |
| `GET` | `/tokens/{id}` | `tokens:read` | Inspect token metadata |
| `DELETE` | `/tokens/{id}` | `tokens:write` | Revoke token |
| `GET` | `/policies` | `policies:read` | Get active policy document |
| `PUT` | `/policies` | `policies:write` | Replace active policy document |
| `POST` | `/policies:evaluate` | `policies:read` | Evaluate hypothetical policy request |
| `GET` | `/approvals` | `approvals:read` | List approvals (`status` filter supported) |
| `POST` | `/approvals/{id}:approve` | `approvals:write` | Approve request |
| `POST` | `/approvals/{id}:deny` | `approvals:write` | Deny request |
| `GET` | `/audit` | `audit:read` | Query audit events (`from`, `to`, `agent_name`, `service`, `action`, `status`, `limit`, `offset`) |
| `GET` | `/audit/export` | `audit:read` | Export audit events (`format=jsonl|csv`) |
| `GET` | `/webhooks` | `webhooks:read` | List webhook subscriptions |
| `POST` | `/webhooks` | `webhooks:write` | Create webhook subscription |
| `DELETE` | `/webhooks/{id}` | `webhooks:write` | Delete webhook subscription |
| `GET` | `/webhooks/events` | `webhooks:read` | List webhook events (`limit`, capped at 1000) |

## Webhook endpoints

- `GET /v1/webhooks` (scope: `webhooks:read`)
- `POST /v1/webhooks` (scope: `webhooks:write`)
- `DELETE /v1/webhooks/{id}` (scope: `webhooks:write`)
- `GET /v1/webhooks/events` (scope: `webhooks:read`)

### Webhook validation and constraints

- Webhook URL must use `https`.
- Private/loopback targets are rejected at registration time.
- `GET /v1/webhooks/events` enforces a bounded `limit` (capped at 1000).

## Common request payloads

### Execute action

`POST /v1/actions/{service}/{action}:execute`

```json
{
  "input": {
    "key": "value"
  }
}
```

### Validate action input

`POST /v1/actions/validate`

```json
{
  "schema": {
    "type": "object",
    "properties": {
      "title": {"type": "string"}
    }
  },
  "input": {
    "title": "hello"
  }
}
```

### Create token

`POST /v1/tokens`

```json
{
  "tenant_id": "tenant-a",
  "agent_name": "agent-a",
  "scopes": ["actions:execute", "audit:read"],
  "ttl_seconds": 3600
}
```

### Create webhook

`POST /v1/webhooks`

```json
{
  "url": "https://example.com/webhook",
  "secret": "optional-shared-secret",
  "events": ["approval.requested", "approval.denied"]
}
```

See [cli-reference.md](../cli-reference.md) for full CLI command coverage.

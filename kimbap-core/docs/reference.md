# Reference

## Usage Examples

### REST v1 API

The REST API is available when running `kimbap serve`.

Public routes: `/v1/health`, `/v1/actions`, `/v1/actions/{service}/{action}`.

Authenticated action execution routes (`POST /v1/actions/{service}/{action}:execute`, `POST /v1/actions/validate`) require bearer token + tenant context. Scope-gated management routes (tokens, policies, approvals, audit, vault, webhooks) also require bearer token + tenant context, then apply route-specific scope checks.

**Example: list available actions**

```bash
curl http://localhost:8080/v1/actions
```

**Example: execute an action**

```bash
curl -X POST "http://localhost:8080/v1/actions/github/list-pull-requests:execute" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ "input": { "repo": "owner/repo" } }'
```

**Example: approve a pending action**

```bash
curl -X POST "http://localhost:8080/v1/approvals/APPROVAL_ID:approve" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Example: query audit logs**

```bash
curl "http://localhost:8080/v1/audit?limit=50" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Approval & Notification Flow

When a policy rule marks an action `require_approval`, execution suspends and Kimbap notifies configured webhook channels (Slack, Telegram, email, generic webhook). Operators approve or deny via the REST API or the embedded console.

---

## API Surfaces

### REST v1 API (`/v1`)

Used by agents and operators to execute actions, manage tokens, policies, approvals, audit, and vault entries.

Authentication: public discovery routes are unauthenticated; action execution and management routes require bearer token + tenant context (with additional route-specific scopes on management endpoints).

| Prefix | Purpose |
|--------|---------|
| `/v1/health` | Health check |
| `/v1/actions` | List and describe installed actions |
| `/v1/actions/{service}/{action}` | Describe a specific action |
| `/v1/actions/{service}/{action}:execute` | Execute an action |
| `/v1/actions/validate` | Validate action input against schema |
| `/v1/tokens` | Issue, list, and revoke access tokens |
| `/v1/policies` | Get/set policy documents, evaluate policy |
| `/v1/approvals` | List pending approvals, approve or deny |
| `/v1/audit` | Query and export audit records |
| `/v1/vault` | List vault key metadata |
| `/v1/webhooks` | Manage webhook subscriptions and inspect recent webhook events (when webhook dispatcher is configured) |
| `/console` | Embedded console SPA (optional) |

### Webhook Event Coverage

Currently emitted management events include:

- `token.created`, `token.deleted`
- `policy.created`, `policy.updated`
- `approval.requested`, `approval.approved`, `approval.denied`, `approval.expired`

`service.installed` and `service.removed` are reserved event names and are not emitted in the current API surface because service lifecycle management currently runs through CLI commands.

Webhook subscriptions may include an `events` array. If omitted or empty, the subscription receives all currently emitted events. Unknown or inactive event names are rejected at creation time.

### CLI

Service installation and management run through the CLI:

```bash
# Service management
kimbap service install slack
kimbap service validate slack.yaml
kimbap service list

# Action discovery and execution
kimbap actions list
kimbap search "send message"
kimbap call slack.post-message --channel C123 --text "hello"

# Credential management
kimbap vault set MY_KEY
kimbap link slack                          # link service to vault or OAuth connector

# OAuth connector (browser flow, e.g. Zoom — BYO credentials from Zoom OAuth app)
kimbap auth connect zoom
kimbap connector status zoom

# Code generation
kimbap generate ts --service github        # TypeScript input interfaces
kimbap generate py --service github        # Python TypedDict inputs

# Policy and audit
kimbap policy set --file policy.yaml
kimbap audit tail
```

### macOS AppleScript services (Spotify and Shortcuts)

After installing the service (`kimbap service install spotify`), actions map to registered
AppleScript commands. Examples:

```bash
# Spotify playback control (macOS only)
kimbap call spotify.get-current-track
kimbap call spotify.play
kimbap call spotify.pause
kimbap call spotify.next-track
kimbap call spotify.set-volume --volume 70
kimbap call spotify.search-play --query "Bonobo"

# macOS Shortcuts automation
kimbap call shortcuts.list
kimbap call shortcuts.run --name "Morning Routine"
kimbap call shortcuts.run-with-input --name "Resize Image" --input "photo.jpg"
```

---

## Tech Stack

- **Runtime**: Go 1.24+
- **Framework**: chi v5
- **Database**: SQLite (default, embedded) or PostgreSQL (optional)
- **Notifications**: webhook dispatch (Slack/Telegram/Email/Webhook)
- **Logging**: zerolog + JSONL audit logs
- **Containerization**: Docker and Docker Compose

---

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific test package
go test ./internal/policy/...

# Run tests with coverage
go test -cover ./...

# Run tests in verbose mode
go test -v ./...

# Run tests with race detection
go test -race ./...
```

### Test Structure

Test files follow these naming conventions:
- Unit tests: `*_test.go` (same directory as source file)
- Integration tests: `*_integration_test.go`

### Testing Best Practices

1. **Mock interfaces for testability**:

   ```go
   type ActionRuntime interface {
       Execute(ctx context.Context, req *ActionRequest) (*ActionResult, error)
   }

   mockRuntime := &MockActionRuntime{
       ExecuteFunc: func(ctx context.Context, req *ActionRequest) (*ActionResult, error) {
           return &ActionResult{Status: "ok"}, nil
       },
   }
   ```

2. **Use in-memory stores for fast tests**:

   ```go
   store := store.NewMemoryStore()
   ```

3. **Clean up resources**:

   ```go
   func TestApprovalFlow(t *testing.T) {
       mgr := approvals.NewManager(store)
       defer mgr.Close()
       // test code
   }
   ```

See `../CONTRIBUTING.md` for details.

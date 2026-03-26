# Reference

## Usage Examples

### REST v1 API

The REST API is available when running `kimbap serve`. All routes require a bearer token.

**Example: list available actions**

```bash
curl http://localhost:8080/v1/actions \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Example: execute an action**

```bash
curl -X POST "http://localhost:8080/v1/actions/github/list_pull_requests:execute" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ "args": { "repo": "owner/repo" } }'
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

Authentication: bearer token.

| Prefix | Purpose |
|--------|---------|
| `/v1/health` | Health check |
| `/v1/actions` | List and describe installed actions |
| `/v1/actions/{service}/{action}:execute` | Execute an action |
| `/v1/tokens` | Issue, list, and revoke access tokens |
| `/v1/policies` | Get/set policy documents, evaluate policy |
| `/v1/approvals` | List pending approvals, approve or deny |
| `/v1/audit` | Query and export audit records |
| `/v1/vault` | List vault key metadata |
| `/console` | Embedded console SPA (optional) |

### CLI

Service installation and management run through the CLI:

```bash
kimbap service install slack.yaml
kimbap service validate slack.yaml
kimbap actions list
kimbap call slack.post_message --channel C123 --text "hello"
kimbap vault set MY_KEY
kimbap policy set --file policy.yaml
kimbap audit tail
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

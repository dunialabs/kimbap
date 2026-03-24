# Reference

## Usage Examples

### Admin API (Kimbap Console)

Kimbap Console uses a single `/admin` endpoint to perform administrative operations.

**Example: create a user**

```bash
curl -X POST http://localhost:3002/admin \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ADMIN_ACCESS_TOKEN" \
  -d '{
    "action": 1010,
    "data": {
      "userId": "user123",
      "status": 1,
      "role": 0
    }
  }'
```

The exact action codes and payloads are defined in `api/ADMIN_API.md`.

### Socket.IO (Kimbap Desk)

Kimbap Desk uses Socket.IO for real-time communication with Kimbap Core.

**Example: connect and fetch capabilities**

```javascript
import { io } from "socket.io-client";

const socket = io("http://localhost:3002", {
  auth: { token: "USER_ACCESS_TOKEN" },
});

socket.on("connect", () => {
  console.log("connected", socket.id);

  socket.emit("get_capabilities", { requestId: "req-123" });
});

socket.on("socket_response", (response) => {
  if (response.requestId === "req-123" && response.success) {
    console.log("capabilities", response.data);
  }
});

socket.on("notification", (payload) => {
  // handle capability changes, approval requests, etc.
});
```

See `api/SOCKET_USAGE.md` for the full event list and payload schemas.

### OAuth 2.0

Kimbap Core exposes an OAuth 2.0 service for obtaining access tokens that can be used with MCP clients.

These OAuth endpoints issue access tokens for authenticating to Kimbap Core. They are separate from downstream connector OAuth credentials (for example Google/Notion/Figma) which are stored encrypted and refreshed internally by Kimbap Core.

**Dynamic Client Registration (optional)**

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

**Authorization Code + PKCE (user-interactive)**

```bash
# 1. Create code_verifier and code_challenge
CODE_VERIFIER=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-43)
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | openssl dgst -sha256 -binary | base64 | tr -d "=+/" | cut -c1-43)

# 2. Open the authorization URL in a browser
echo "http://localhost:3002/authorize?client_id=YOUR_CLIENT_ID&redirect_uri=YOUR_CALLBACK&response_type=code&code_challenge=$CODE_CHALLENGE&code_challenge_method=S256"

# 3. After the user authorizes, exchange the code for a token
curl -X POST http://localhost:3002/token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "authorization_code",
    "code": "AUTHORIZATION_CODE_FROM_CALLBACK",
    "client_id": "YOUR_CLIENT_ID",
    "code_verifier": "'"$CODE_VERIFIER"'"
  }'
```

See `api/API.md` for full OAuth 2.0 details.

**Token Introspection**

```bash
curl -X POST http://localhost:3002/introspect \
  -H "Content-Type: application/json" \
  -d '{ "token": "YOUR_OAUTH_ACCESS_TOKEN", "token_type_hint": "access_token" }'
```

---

## API & Documentation

### API Surfaces

Kimbap Core exposes different APIs for different roles:

- **MCP protocol interface** (`/mcp`)
  Standard MCP endpoints for MCP-compatible clients such as Claude Desktop, ChatGPT MCP, or Cursor.
  Authentication: bearer token (OAuth access token (JWT) or Kimbap access token (opaque)).
  Transport: HTTP/SSE depending on your MCP host.

- **Admin API** (`/admin`)
  Used by Kimbap Console and automation scripts to manage users, servers, permissions, and quotas.
  Authentication: bearer token (Kimbap access token (opaque)).

- **Socket.IO channel** (`/socket.io`)
  Used by Kimbap Desk for real-time notifications, capability configuration, and approval workflows.
  Authentication: bearer token (Kimbap access token (opaque)).

- **OAuth 2.0 endpoints** (`/.well-known/*`, `/register`, `/authorize`, `/token`, `/introspect`, `/revoke`)
  Used by clients to obtain access tokens (dynamic client registration, authorization code with PKCE, refresh tokens) and check token validity.

### Reference Docs

| Document | Target Users | Description | Link |
|----------|-------------|-------------|------|
| **API.md** | End Users | API overview, authentication, MCP protocol, OAuth 2.0 | [View](./api/API.md) |
| **ADMIN_API.md** | Administrators | Complete admin API protocol (47 operations) | [View](./api/ADMIN_API.md) |
| **SOCKET_USAGE.md** | Kimbap Desk Users | Complete Socket.IO real-time communication guide | [View](./api/SOCKET_USAGE.md) |
| **MCP Official Docs** | Developers | Model Context Protocol standard | [View](https://modelcontextprotocol.io/docs/) |

### Quick Links

- **[OAuth 2.0 Authentication](./api/API.md#2-oauth-20-authentication)** - Get access tokens for MCP connections
- **[MCP Protocol](./api/API.md#1-mcp-protocol-interface)** - MCP endpoints and namespaces
- **[Admin API](./api/ADMIN_API.md)** - User, server, permission management (for Kimbap Console)
- **[Socket.IO](./api/SOCKET_USAGE.md)** - Real-time notifications and request-response (for Kimbap Desk)
- **[Complete Examples](./api/API.md#complete-examples)** - OAuth + MCP workflow

---

## Tech Stack

- **Runtime**: Go 1.24+
- **Framework**: chi v5
- **Database**: PostgreSQL with GORM v2
- **Real-time**: go-socket.io
- **Logging**: zerolog + database audit logs
- **Containerization**: Docker and Docker Compose

---

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific test package
go test ./internal/mcp/core/...

# Run tests with coverage
go test -cover ./...

# Run tests in verbose mode
go test -v ./...

# Run specific test function
go test -run TestRequestIdMapper ./internal/mcp/core/...

# Run tests with race detection
go test -race ./...
```

### Test Structure

Test files follow these naming conventions:
- Unit tests: `*_test.go` (same directory as source file)
- Integration tests: `*_integration_test.go`
- E2E tests: `*_e2e_test.go`

### Testing Best Practices

1. **Mock Singleton Services**:

   ```go
   // Use interfaces for testability
   type ServerManager interface {
       CreateServerConnection(ctx context.Context, serverID string) error
       // ...
   }

   // Inject mock in tests
   mockManager := &MockServerManager{
       CreateServerConnectionFunc: func(ctx context.Context, serverID string) error {
           return nil
       },
   }
   ```

2. **Use In-Memory EventStore**:

   ```go
   eventStore := NewPersistentEventStore(&EventStoreConfig{
       UseInMemory: true,  // Speeds up tests
   })
   ```

3. **Clean Up Resources**:

   ```go
   func TestProxySession(t *testing.T) {
       session := NewProxySession(...)
       defer session.Cleanup()

       // test code
   }
   ```

4. **Test RequestId Mapping**:

   ```go
   func TestRequestIdMapper(t *testing.T) {
       mapper := NewRequestIdMapper("session123")
       proxyID := mapper.MapToProxy("client-req-1")
       
       // Format: {sessionId}:{originalId}:{timestamp}
       if !strings.HasPrefix(proxyID, "session123:client-req-1:") {
           t.Errorf("unexpected proxy ID format: %s", proxyID)
       }
   }
   ```

### Current Test Status

Unit tests exist for several core packages:

- `internal/admin/` — Admin controller
- `internal/admin/handlers/` — Server and user admin handlers
- `internal/mcp/core/` — EventStore, ProxySession, RequestIdMapper, ServerContext, SessionStore
- `internal/mcp/controller/` — MCP controller
- `internal/mcp/service/` — Capabilities service
- `internal/middleware/` — Auth and IP whitelist middleware
- `internal/oauth/controller/` — OAuth metadata controller
- `internal/repository/` — User repository
- `internal/security/` — Rate limiting
- `internal/socket/` — Socket.IO service
- `internal/user/` — User handler

Additional test contributions are especially useful for:

- `GlobalRequestRouter` routing behavior.
- Concurrency tests for the persistent event store.
- OAuth 2.0 flows.
- Socket.IO connection and notification scenarios.
- End-to-end integration tests.

See `../CONTRIBUTING.md` for details.

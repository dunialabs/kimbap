# Architecture

## Overview

Kimbap is a **secure action runtime for AI agents**. Its job is to sit between agents and external systems, handling identity, policy, credential injection, OAuth lifecycle, approvals, and audit so agents never touch raw credentials.

The central concept is the **Action Runtime**: a canonical execution pipeline that every product surface converges to. Whether a request arrives from the CLI, an HTTP proxy, a subprocess wrapper, a connected server, or the REST API, it passes through the same pipeline before anything executes.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Product Surfaces                         │
│                                                                 │
│  kimbap call   kimbap proxy   kimbap run   kimbap serve   /v1  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Action Runtime                            │
│                                                                 │
│  identify → resolve → policy → credential → execute → audit    │
└─────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Integration Backends                        │
│                                                                 │
│   Tier 1 Skills (YAML)   Tier 2 Connectors   Tier 2b CLI Wrap  │
└─────────────────────────────────────────────────────────────────┘
```

MCP is one adapter surface in this model, not the center of it.

---

## Execution Pipeline

Every action request, regardless of how it arrived, passes through these six stages in order.

### 1. Identify

Resolve the caller's identity from the incoming token or session context.

- Validate the bearer token (JWT or opaque) against `internal/security/`
- Resolve the associated user, agent, and tenant
- Establish session context via `internal/sessions/`
- Reject unauthenticated or expired requests before any further processing

### 2. Resolve

Determine which action is being requested and whether it exists.

- Parse the action reference (`service.action_name`)
- Look up the action definition in `internal/actions/` and `internal/registry/`
- Classify the request type via `internal/classifier/`
- Return a structured action descriptor with parameter schema, risk metadata, and execution backend

### 3. Policy

Evaluate whether this caller is allowed to run this action with these parameters.

- Load applicable policies from `internal/policy/`
- Evaluate RBAC/ABAC rules against the resolved identity and action descriptor
- Apply tenant-scoped restrictions
- Determine the outcome: `allow`, `deny`, or `require_approval`

If the outcome is `deny`, the pipeline stops here and returns an error. If `require_approval`, execution is suspended and an approval record is created.

### 4. Credential

Inject the credentials needed to execute the action without exposing them to the caller.

- Retrieve encrypted credential blobs from `internal/vault/`
- Decrypt using the tenant key hierarchy via `internal/crypto/`
- For OAuth-backed services, check token expiry and refresh via `internal/connectors/`
- Inject credentials into the execution context only, not into any agent-visible scope

The caller never sees the raw credential. It exists only inside the runtime's execution context for the duration of this call.

### 5. Execute

Run the action against the appropriate backend.

- Dispatch to the correct adapter via `internal/adapters/`
- For Tier 1 skills: interpret the YAML skill definition and make the outbound HTTP call
- For Tier 2 connectors: delegate to the connector's execution logic
- For Tier 2b CLI adapters: invoke the subprocess with injected credentials
- Stream or collect the response

### 6. Audit

Record the full decision path and outcome.

- Write a structured audit record via `internal/audit/`
- Include: caller identity, action, parameters (sanitized), policy outcome, approval state, execution result, timestamp
- Emit to the log service via `internal/log/`
- Push real-time notifications via `internal/socket/` where applicable

---

## Product Surfaces

Each surface is an entry point into the same Action Runtime pipeline.

### CLI Mode (`kimbap call`)

The most direct surface. An agent or script invokes an action by name.

```bash
kimbap call github.list_pull_requests --repo owner/repo
kimbap call stripe.refund_charge --charge ch_abc --amount 500
```

**Entry point:** `cmd/kimbap/`

**Flow:**

```
CLI invocation
  │
  ▼
Parse action reference + arguments
  │
  ▼
Resolve local or remote runtime (embedded or connected)
  │
  ▼
Action Runtime pipeline (identify → ... → audit)
  │
  ▼
Print result to stdout
```

In embedded mode, the runtime runs in-process. In connected mode, the CLI talks to a running `kimbap serve` instance over the REST API.

### Proxy Mode (`kimbap proxy`)

Intercepts outbound HTTP traffic from an existing agent and routes matching requests through the Action Runtime.

```bash
kimbap proxy --port 10255
export HTTP_PROXY=http://127.0.0.1:10255
python agent.py
```

**Entry point:** `internal/proxy/`

**Flow:**

```
Agent makes outbound HTTP request
  │
  ▼
kimbap proxy intercepts (CONNECT tunnel or HTTP proxy)
  │
  ▼
Classifier identifies whether request maps to a known action
  │
  ├── Matches known action → Action Runtime pipeline
  │
  └── No match → pass through (or block, per policy)
  │
  ▼
Response returned to agent
```

The agent's code doesn't change. The proxy is the trust boundary.

### Subprocess Mode (`kimbap run`)

Wraps an agent process and intercepts its credential usage at the environment and subprocess level.

```bash
kimbap run -- python agent.py
```

**Entry point:** `internal/runner/`

**Flow:**

```
kimbap run forks the agent process
  │
  ▼
Inject sanitized environment (no raw credentials)
  │
  ▼
Agent process runs
  │
  ▼
Agent calls kimbap (via injected env vars or local socket)
  │
  ▼
Action Runtime pipeline
  │
  ▼
Result returned to agent process
```

The subprocess never receives long-lived credentials. It receives only what the runtime decides to inject per-call.

### Connected Server (`kimbap serve`)

A long-running server deployment for shared use, multi-tenant environments, and centralized audit.

```bash
kimbap serve --port 8080
```

**Entry point:** `cmd/server/`

Exposes:

- `/api/v1` — canonical REST management API (tokens, policies, approvals, audit, actions)
- `/health`, `/ready` — liveness and readiness probes
- OAuth2 endpoints
- Socket.IO for real-time push (approval requests, status changes, session updates)

**Flow:**

```
HTTP request arrives
  │
  ▼
chi router (internal/middleware/: IP check → auth → rate limit)
  │
  ▼
Route handler (api/, oauth/, socket/)
  │
  ▼
Action Runtime pipeline
  │
  ▼
Response + audit record
```

### Daemon Mode

A background process that manages long-lived connector sessions, token refresh cycles, and job scheduling.

**Entry point:** `internal/jobs/`

Responsibilities:

- Periodic OAuth token refresh for active connectors
- Connector health checks and reconnection
- Scheduled audit log sync
- Background policy cache invalidation

The daemon does not handle inbound action requests directly. It keeps the runtime's state consistent so that execution-time credential injection is fast and reliable.

### REST API (`/v1`)

The canonical management interface for operators, consoles, and automation.

**Entry point:** `internal/api/`

Resource groups:

| Prefix | Purpose |
|---|---|
| `/api/v1/tokens` | Issue, list, and revoke access tokens |
| `/api/v1/policies` | Create and manage policy rules |
| `/api/v1/approvals` | List pending approvals, approve or reject |
| `/api/v1/audit` | Query audit records |
| `/api/v1/actions` | List and describe available actions |
| `/api/v1/vault` | Manage encrypted credential entries |
| `/api/v1/connectors` | Configure and manage OAuth connectors |
| `/api/v1/skills` | Install and manage skill definitions |

All routes require a valid bearer token. Admin-scoped routes require an Owner or Admin role.

---

## Integration Model

Kimbap avoids a bespoke codebase for every service. The integration tiers reflect how much custom code a service actually needs.

### Tier 1 — Declarative Skills (YAML)

Most modern REST APIs can be expressed as a YAML skill file. The runtime interprets the skill at execution time.

**Module:** `internal/skills/`

A skill definition includes:

- `service` and `action` identifiers
- auth type (`api_key`, `bearer`, `oauth2`, `basic`)
- endpoint template and HTTP method
- argument schema with types and validation
- output extraction path (JSONPath or jq)
- error mapping
- pagination strategy
- risk metadata (`risk_level`, `require_approval`)

Example structure:

```yaml
service: github
action: list_pull_requests
auth: oauth2
connector: github
method: GET
endpoint: https://api.github.com/repos/{repo}/pulls
args:
  repo:
    type: string
    required: true
  state:
    type: string
    default: open
output: "$[*]"
risk_level: low
```

The skill loader in `internal/skills/` parses these at startup and registers them with the action registry.

### Tier 2 — Connectors (OAuth)

Services that need OAuth device flow, token refresh, or provider-specific lifecycle handling get a connector.

**Module:** `internal/connectors/`

A connector manages:

- Initial OAuth authorization (device flow or redirect)
- Token storage in the vault
- Automatic refresh before expiry
- Revocation on disconnect

Current connector examples: Gmail, GitHub Apps, Slack, HubSpot, Stripe Connect.

Connectors are not execution backends on their own. They supply credentials to the Action Runtime's credential injection stage. The actual HTTP call still goes through the skill or adapter layer.

### Tier 2b — Existing CLI Adapters

Some mature CLIs are worth wrapping, but the CLI is never the trust boundary. Kimbap is.

**Module:** `internal/adapters/`

A CLI adapter:

- Defines the action interface (arguments, output schema)
- Constructs the subprocess invocation
- Injects credentials as short-lived env vars scoped to the subprocess
- Captures and parses stdout/stderr
- Maps exit codes to structured errors

The agent calls `kimbap call tool.action`. It never invokes the underlying CLI directly.

---

## Security Model

### Vault and Credential Injection

**Module:** `internal/vault/`, `internal/crypto/`

Credentials are stored encrypted at rest. The encryption model uses a per-tenant key hierarchy:

```
Master key (operator-held or KMS-backed)
  └── Tenant key (derived per tenant)
        └── Credential blob (AES-GCM encrypted)
```

At execution time:

1. The runtime retrieves the encrypted blob for the required credential
2. Decrypts it using the tenant key
3. Injects it into the execution context
4. The context is discarded after the call completes

The decrypted credential is never written to disk, never logged, and never returned to the caller.

### Token Brokerage

**Module:** `internal/auth/`, `internal/oauth/`, `internal/security/`

Kimbap issues its own short-lived access tokens to agents. These tokens:

- identify the agent and tenant
- carry a scope (which actions are permitted)
- expire on a configurable TTL
- can be revoked individually or by tenant

Agents present these tokens to Kimbap. Kimbap exchanges them for the actual service credentials internally. The agent never holds the service credential.

For OAuth-backed services, Kimbap holds the refresh token. The agent only ever sees a Kimbap token.

### Policy Enforcement

**Module:** `internal/policy/`

Policy evaluation happens at stage 3 of the pipeline, before any credential is touched.

Policy rules can match on:

- caller identity (user, agent, role)
- action identifier (`service.action`)
- parameter values (e.g., block deletes on production resources)
- time of day or rate limits
- tenant context

Outcomes:

- `allow` — proceed to credential injection
- `deny` — return error immediately, write audit record
- `require_approval` — suspend execution, create approval record, notify operator

Policy is evaluated in tenant context. A rule in tenant A cannot affect tenant B.

### Human Approval Gates

**Module:** `internal/approvals/`

When policy marks an action `require_approval`, the runtime:

1. Creates an approval record with the full request context
2. Suspends the execution
3. Notifies the operator via Socket.IO (and optionally messaging adapters)
4. Waits for an explicit approve or reject decision
5. On approval: resumes the pipeline from the credential stage
6. On rejection: returns an error to the caller
7. Records the full decision path in audit

Approval records include: caller identity, action, parameters (sanitized), policy rule that triggered the gate, operator who decided, timestamp, and outcome.

---

## Data Flow Diagrams

### CLI Mode (embedded runtime)

```
kimbap call github.list_pull_requests --repo owner/repo
  │
  ├── Parse: service=github, action=list_pull_requests, args={repo: "owner/repo"}
  │
  ├── Identify: read local token from ~/.kimbap/token
  │
  ├── Resolve: look up skill definition for github.list_pull_requests
  │
  ├── Policy: evaluate rules for this caller + action
  │   └── outcome: allow
  │
  ├── Credential: decrypt GitHub OAuth token from local vault
  │
  ├── Execute: GET https://api.github.com/repos/owner/repo/pulls
  │            Authorization: Bearer <decrypted_token>
  │
  ├── Audit: write record to local audit log
  │
  └── Print: JSON result to stdout
```

### Proxy Mode

```
Agent: GET https://api.github.com/repos/owner/repo/pulls
  │
  ▼
kimbap proxy (port 10255)
  │
  ├── Classifier: matches github.list_pull_requests pattern
  │
  ├── Action Runtime pipeline (identify → resolve → policy → credential → execute → audit)
  │
  └── Response forwarded to agent
```

### Connected Server (inbound API call)

```
POST /api/v1/actions/call
Authorization: Bearer <kimbap_token>
Body: { "action": "stripe.refund_charge", "args": { ... } }
  │
  ▼
chi router
  │
  ├── Middleware: IP check → token auth → rate limit
  │
  ├── Handler: internal/api/
  │
  ├── Action Runtime pipeline
  │   ├── Identify: resolve token → user + tenant
  │   ├── Resolve: look up stripe.refund_charge
  │   ├── Policy: evaluate → require_approval
  │   │   └── Create approval record, suspend, notify operator
  │   │       (execution resumes when operator approves)
  │   ├── Credential: decrypt Stripe API key
  │   ├── Execute: POST https://api.stripe.com/v1/refunds
  │   └── Audit: write full decision path
  │
  └── Response: { "result": { ... } }
```

### Approval Flow

```
Action Runtime: policy outcome = require_approval
  │
  ├── Create approval record (internal/approvals/)
  │
  ├── Suspend execution
  │
  ├── Notify operator (Socket.IO → Kimbap Desk / Console)
  │
  ├── Operator reviews: action, args, caller identity
  │
  ├── Operator decision: approve / reject
  │
  ├── On approve:
  │   └── Resume pipeline at credential stage → execute → audit
  │
  └── On reject:
      └── Return error to caller → audit
```

---

## Project Structure

```
kimbap-core/
├── cmd/
│   ├── kimbap/           # CLI entry point (kimbap call, proxy, run, serve, ...)
│   └── server/           # Server entry point (kimbap serve)
│
└── internal/
    ├── runtime/          # Action Runtime core: pipeline orchestration
    ├── actions/          # Action definitions and registry
    ├── registry/         # Action lookup and resolution
    ├── classifier/       # Request classification (maps inbound to action refs)
    │
    ├── skills/           # Tier 1 declarative skill loader and executor
    ├── adapters/         # Tier 2b CLI adapter wrappers
    ├── connectors/       # Tier 2 OAuth connector lifecycle
    │
    ├── vault/            # Encrypted credential storage
    ├── crypto/           # Key hierarchy, AES-GCM encryption/decryption
    ├── auth/             # Token issuance and validation
    ├── security/         # Authentication and authorization
    ├── oauth/            # OAuth 2.0 flows (device, PKCE, refresh)
    │
    ├── policy/           # Policy rule evaluation
    ├── approvals/        # Approval record management and workflow
    ├── audit/            # Audit record writing
    │
    ├── sessions/         # Session context management
    ├── profiles/         # Agent and user profile management
    │
    ├── proxy/            # HTTP proxy intercept (kimbap proxy)
    ├── runner/           # Subprocess wrapper (kimbap run)
    │
    ├── api/              # REST /api/v1 handlers
    ├── mcp/              # MCP JSON-RPC adapter surface
    ├── socket/           # Socket.IO real-time push
    │
    ├── jobs/             # Background jobs (token refresh, health checks)
    ├── observability/    # Metrics and tracing
    ├── doctor/           # Runtime health diagnostics
    │
    ├── middleware/       # HTTP middleware (auth, rate limit, IP allowlist)
    ├── repository/       # Data access layer (GORM)
    ├── database/         # PostgreSQL connection and migrations
    ├── store/            # In-memory and persistent store abstractions
    │
    ├── config/           # Configuration loading and validation
    ├── log/              # Audit log service and sync
    ├── logger/           # Zerolog structured logger wrapper
    ├── service/          # Cross-cutting application services
    ├── types/            # Shared types and enums
    └── utils/            # Utility functions
```

---

## Core Design Patterns

### Single Pipeline, Multiple Entry Points

Every surface (CLI, proxy, subprocess, server, REST) is an adapter that feeds into the same six-stage pipeline. There is no special-cased execution path. Adding a new surface means writing an adapter that produces a standard action request, not reimplementing security logic.

### Credentials Never Cross the Trust Boundary

The trust boundary is the Action Runtime. Credentials are decrypted inside it and discarded after the call. They don't appear in:

- agent-visible environment variables
- CLI arguments
- log output
- audit records
- HTTP responses to callers

The final outbound HTTP request to the external service does carry the credential, but that request is made by the runtime, not the agent.

### Tenant Isolation at Every Layer

Vault entries, policy rules, approval records, and audit logs are all tenant-scoped. The key hierarchy is per-tenant. Policy evaluation runs in tenant context. A misconfigured policy in one tenant cannot affect another.

### Declarative First

Most integrations should be YAML skill files, not Go code. The skill loader interprets them at runtime. This keeps the integration surface maintainable and auditable without requiring a new binary for each service.

### Audit is Mandatory

Every pipeline execution writes an audit record, including denied requests and approval-gated requests that were never executed. The audit trail is complete by construction, not by convention.

## HTTP Endpoints (Connected Server)

```
GET  /health                    liveness probe
GET  /ready                     readiness probe

POST /oauth/token               OAuth2 token endpoint
GET  /oauth/authorize           OAuth2 authorization endpoint
POST /oauth/device/code         Device flow initiation

GET  /api/v1/actions            List available actions
GET  /api/v1/actions/:id        Describe a specific action
POST /api/v1/actions/call       Execute an action

GET  /api/v1/tokens             List issued tokens
POST /api/v1/tokens             Issue a new token
DEL  /api/v1/tokens/:id         Revoke a token

GET  /api/v1/policies           List policy rules
POST /api/v1/policies           Create a policy rule
DEL  /api/v1/policies/:id       Delete a policy rule

GET  /api/v1/approvals          List pending approvals
POST /api/v1/approvals/:id/approve   Approve a pending action
POST /api/v1/approvals/:id/reject    Reject a pending action

GET  /api/v1/audit              Query audit records

GET  /api/v1/vault              List vault entries (metadata only)
POST /api/v1/vault              Store a credential
DEL  /api/v1/vault/:id          Delete a credential

GET  /api/v1/connectors         List connectors
POST /api/v1/connectors/:id/login    Initiate connector OAuth flow
POST /api/v1/connectors/:id/logout   Revoke connector credentials

GET  /api/v1/skills             List installed skills
POST /api/v1/skills             Install a skill

/admin/*                        Legacy action-code admin (frozen)
/user/*                         Legacy action-code user (frozen)
```

Socket.IO connects on the root path and uses room-based push for approval notifications, session updates, and connector status changes.

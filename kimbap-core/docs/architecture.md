# Architecture

## Overview

Kimbap is a **secure action runtime for AI agents**. Its job is to sit between agents and external systems, handling identity, policy, credential injection, OAuth lifecycle, approvals, and audit so agents never touch raw credentials.

The central concept is the **Action Runtime**: a canonical execution pipeline that every product surface converges to. Whether a request arrives from the CLI, an HTTP proxy, a subprocess wrapper, a connected server, or the REST API, it passes through the same pipeline before anything executes.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Product Surfaces                         │
│                                                                 │
│      kimbap call   kimbap proxy   kimbap run   kimbap daemon      │
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
│  Tier 1 Services (YAML)  Tier 2 Connectors   Tier 2b CLI Wrap   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Execution Pipeline

Every action request, regardless of how it arrived, passes through these six stages in order.

### 1. Identify

Resolve the caller's identity from the incoming token or session context.

- Resolve the caller's identity from the local session context
- For CLI and proxy modes, a local operator principal is constructed automatically

### 2. Resolve

Determine which action is being requested and whether it exists.

- Parse the action reference (`service.action_name`)
- Look up the action definition in `internal/actions/` and `internal/services/`
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
- For Tier 1 services: interpret the YAML service definition and make the outbound HTTP call
- For Tier 2 connectors: delegate to the connector's execution logic
- For Tier 2b CLI adapters: invoke the subprocess with injected credentials
- Stream or collect the response

### 6. Audit

Record the full decision path and outcome.

- Write a structured audit record via `internal/audit/`
- Include: caller identity, action, parameters (sanitized), policy outcome, approval state, execution result, timestamp
- Deliver approval notifications via configured notifiers (Slack, Telegram, email, webhook)

---

## Product Surfaces

Each surface is an entry point into the same Action Runtime pipeline.

### CLI Mode (`kimbap call`)

The most direct surface. An agent or script invokes an action by name.

```bash
kimbap call github.list-pull-requests --repo owner/repo
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

The runtime runs in-process. No server required.

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


### Daemon Mode

A background process that manages long-lived connector sessions, token refresh cycles, and job scheduling.

**Entry point:** `internal/jobs/`

Responsibilities:

- Periodic OAuth token refresh for active connectors
- Connector health checks and reconnection
- Scheduled audit log sync
- Background policy cache invalidation

The daemon does not handle inbound action requests directly. It keeps the runtime's state consistent so that execution-time credential injection is fast and reliable.


---

## Integration Model

Kimbap avoids a bespoke codebase for every service. The integration tiers reflect how much custom code a service actually needs.

### Tier 1 — Declarative Services (YAML)

Most modern REST APIs can be expressed as a YAML service file. The runtime interprets the service definition at execution time.

**Module:** `internal/services/`

A service definition includes:

- top-level service metadata (`name`, `version`, `description`)
- adapter type (`http`, `applescript`, `command`)
- service-level auth contract (`none`, `header`, `bearer`, `basic`, `query`, `body`)
- one or more named `actions` with adapter-specific fields
- action argument schema (`args`) and request/response mapping
- action-level idempotency (`idempotent`) and risk level (`low|medium|high|critical`)
- optional retry, pagination, and error mapping metadata

Example structure:

```yaml
name: github
version: 1.0.0
description: GitHub API integration
base_url: https://api.github.com
auth:
  type: bearer
  credential_ref: github.token
actions:
  list-pull-requests:
    method: GET
    path: /repos/{owner}/{repo}/pulls
    idempotent: true
    args:
      - name: owner
        type: string
        required: true
      - name: repo
        type: string
        required: true
    request:
      path_params:
        owner: "{owner}"
        repo: "{repo}"
    response:
      type: array
    risk:
      level: low
```

The service loader in `internal/services/` parses these at startup and registers them with the action registry.

### Tier 2 — Connectors (OAuth)

Services that need OAuth device flow, token refresh, or provider-specific lifecycle handling get a connector.

**Module:** `internal/connectors/`

A connector manages:

- Initial OAuth authorization (device flow or redirect)
- Token storage in the vault
- Automatic refresh before expiry
- Revocation on disconnect

Current connector examples: Gmail, GitHub Apps, Slack, HubSpot, Stripe Connect.

Connectors are not execution backends on their own. They supply credentials to the Action Runtime's credential injection stage. The actual HTTP call still goes through the service or adapter layer.

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

**Module:** `internal/connectors/`

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
3. Notifies the operator via Console APIs and configured webhook adapters
4. Waits for an explicit approve or reject decision
5. On approval: resumes the pipeline from the credential stage
6. On rejection: returns an error to the caller
7. Records the full decision path in audit

Approval records include: caller identity, action, parameters (sanitized), policy rule that triggered the gate, operator who decided, timestamp, and outcome.

---

## Data Flow Diagrams

### CLI Mode (embedded runtime)

```
kimbap call github.list-pull-requests --repo owner/repo
  │
├── Parse: service=github, action=list-pull-requests, args={repo: "owner/repo"}
  │
  ├── Identify: construct local CLI principal (tenant-scoped)
  │
├── Resolve: look up service definition for github.list-pull-requests
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
  ├── Classifier: matches github.list-pull-requests pattern
  │
  ├── Action Runtime pipeline (identify → resolve → policy → credential → execute → audit)
  │
  └── Response forwarded to agent
```

### Connected Server (inbound API call)

```
POST /v1/actions/{service}/{action}:execute
Authorization: Bearer <kimbap_token>
Body: { "input": { ... } }
  │
  ▼
chi router
  │
  ├── Auth: token validation → tenant context
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
├── Notify operator (webhook adapters + Console polling)
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
│   └── kimbap/           # CLI entry point (kimbap call, proxy, run, serve, ...)
│
└── internal/
    ├── runtime/          # Action Runtime core: pipeline orchestration
    ├── actions/          # Action types and interfaces
    │
    ├── services/         # Tier 1 declarative service loader and executor
    ├── connectors/       # OAuth2 connector flows
    │
    ├── vault/            # Encrypted credential storage
    ├── crypto/           # Encryption utilities
    │
    ├── policy/           # Policy evaluator (YAML DSL)
    ├── approvals/        # Approval manager (email/slack/telegram/webhook notifiers)
    ├── audit/            # Audit log writers (JSONL, multi-writer)
    │
    │
    ├── store/            # SQL store (SQLite default, Postgres supported)
    │
    ├── config/           # Config loading (config.yaml)
    ├── app/              # Runtime bootstrap and adapters
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

Most integrations should be YAML service files, not Go code. The service loader interprets them at runtime. This keeps the integration surface maintainable and auditable without requiring a new binary for each service.

### Audit is Mandatory

Every pipeline execution writes an audit record, including denied requests and approval-gated requests that were never executed. The audit trail is complete by construction, not by convention.

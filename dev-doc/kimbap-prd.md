# Product Requirements Document (PRD)

## Kimbap
**Secure Action Runtime for AI Agents**

| Field | Value |
|---|---|
| Version | 0.6 |
| Status | Draft |
| Date | March 23, 2026 |
| Owner | Dunia Labs Product Team |
| Product | Kimbap |
| Domain | kimbap.sh |
| Delivery model | Single Go binary; embedded mode and connected mode |
| Source baseline | Private `peta-core-go` + `peta-console` |
| External design reference | `onecli/onecli` |
| Core thesis | The **Action Runtime** is the canonical core. CLI, proxy, REST, and existing service CLIs are adapters into that runtime, not separate products. |

---

## 1. Executive Summary

Kimbap is a **secure action runtime for AI agents**. It lets agents call external services without raw credentials ever becoming part of the agent's prompts, environment variables, shell history, code, or persisted traces. Kimbap is built by extracting and reorienting the core value of `peta-core-go`—vault, downstream OAuth brokerage, policy engine, approval workflow, runtime supervision, REST adapter logic, and audit—into a **CLI-first, subprocess-friendly, proxy-capable** product.

The product is intentionally **not** "an MCP gateway with a CLI added later." It is the inverse:

- **Action Runtime first**
- **Adapters second**
- **MCP compatibility optional**

Kimbap ships as a **single Go binary** with four primary operating modes:

1. **`kimbap call`** — explicit action execution (`service.action`)
2. **`kimbap run`** — run an existing agent process with Kimbap-mediated networking and policy
3. **`kimbap proxy`** — transparent HTTP/HTTPS proxy for zero-code-change HTTP agents
4. **`kimbap serve`** — REST server for connected/team mode

It also includes:

- credential vault
- downstream OAuth login and refresh
- policy engine with dry-run simulation
- human-in-the-loop approval queue
- audit trail
- skill/adapter system for turning REST APIs into Kimbap actions
- agent operating profiles/skills so agents know to use Kimbap by default

Kimbap exists because current options each miss something critical:

- secret managers store credentials but do not govern execution
- service CLIs assume human-owned credentials and leak too much trust into the agent process
- proxy-only tools solve injection but not lifecycle, policy, or canonical action modeling
- `peta-core` contains the right security/runtime primitives, but its public surface and complexity are tightly optimized for MCP clients rather than CLI-first or subprocess agents

Kimbap's job is to make "safe external actions for agents" **boring, universal, and installable**.

---

## 2. Product Context and Problem Statement

### 2.1 The underlying problem

Modern AI agents routinely need to:

- call REST APIs
- use SDKs that make HTTP requests
- invoke subprocesses
- work across multiple services in one task
- run headlessly for hours or days
- operate under approval and audit constraints

Today, teams usually solve this badly:

- put raw API keys into env vars
- log in service CLIs in a shared user profile
- mount credential files into containers
- re-implement one-off wrappers for each service
- bolt policy and approval onto the outside after the fact

This creates five recurring failures:

1. **Credential exposure**  
   Agents can read env vars, shell history, token files, config directories, crash dumps, logs, and prompt traces.

2. **No stable identity model for agents**  
   Tools assume "one human logged in," not "many agents with independently scoped authority."

3. **No cross-service policy model**  
   Teams can restrict Stripe or GitHub in isolation, but not express one coherent policy across all external actions.

4. **OAuth lifecycle failure**  
   Short-lived access tokens expire. Most human-oriented CLIs and wrappers simply fail when this happens.

5. **No canonical action layer**  
   Teams reason at the level of hostnames, SDK calls, headers, or individual CLIs, not a stable, auditable `service.action` namespace.

### 2.2 Why individual service CLIs fail for agents

Service-specific CLIs are not enough, even when the service already has a good CLI.

Examples of failure modes:

- `gh auth login` or `STRIPE_KEY=...` gives the agent process access to raw credentials.
- OAuth-backed CLIs cache tokens under a shared home directory.
- Read-only vs write access requires separate service-native credential provisioning, repeated per tool.
- The CLI becomes the security boundary, but most CLIs were designed for a human developer's laptop, not an untrusted or semi-trusted agent execution environment.
- Audit and policy become fragmented by service.

**Conclusion:** existing CLIs may be useful as **implementation details** or **optional execution adapters**, but they are not the security boundary. Kimbap must remain the security boundary.

### 2.3 Why secret managers alone are not enough

Traditional vault tools solve storage, not execution.

They can help with:

- encrypted secret storage
- rotation workflows
- centralization

They do **not** inherently provide:

- server-side credential injection
- downstream OAuth refresh brokerage
- action-level policy
- approval gating
- canonical action discovery
- unified audit
- per-agent runtime isolation

A vault is necessary but insufficient.

### 2.4 Why peta-core is the right source baseline

`peta-core-go` already contains the hardest and most defensible parts of the system:

- encrypted vault
- downstream OAuth configuration and refresh
- policy enforcement
- approval orchestration
- audit logging
- runtime supervision
- REST-to-tool/action adaptation
- packaged skills

However, the public surface of Peta is built around the MCP gateway model:

- upstream MCP compatibility
- downstream MCP multiplexing
- stateful sessions
- request-id mapping
- reverse routing
- event persistence and resumption
- Webhook notification + console polling for approval flows
- per-client capability filtering

Those concerns are valuable for MCP infrastructure, but they are not the right center of gravity for a CLI-first runtime.

**Kimbap extracts the value and discards the wrapper.**

### 2.5 Why onecli matters

OneCLI is the clearest adjacent reference for the **proxy-first, zero-code-change** entry point:

- fast gateway in Rust
- encrypted secret storage
- agent access tokens
- host/path matching
- transparent credential injection
- MITM proxy as the main adoption path
- external vault integration concepts
- agent skill/instruction directories in the repo

Kimbap should learn from this aggressively.

What Kimbap should borrow:

- proxy as the easiest first adoption surface
- per-agent token model
- host/path matcher for request classification
- split between hot-path gateway and management plane
- pragmatic CA-cert UX
- agent instruction packs/skills

What Kimbap should **not** become:

- only a credential-swapping proxy
- only a dashboard product
- only host/path policy
- a system without canonical actions, approvals, or OAuth lifecycle brokerage

Kimbap must be a **secure action runtime**, not just a credential injector.

---

## 3. Product Vision, Positioning, and Principles

### 3.1 Product vision

> Any AI agent should be able to call any external service safely, with one install, one action model, and no raw credentials ever entering the agent process.

### 3.2 Positioning

Kimbap is:

- **not** a generic human CLI replacement
- **not** a standalone password manager
- **not** an MCP gateway in disguise
- **not** only a proxy
- **not** only a policy engine

Kimbap is the **execution-time control plane for agent actions**.

### 3.3 Product principles

#### P1. Action Runtime is the canonical core
All execution modes must pass through the same internal path:

`identity -> action resolution -> policy -> credential brokerage -> execution -> audit`

No adapter may bypass the runtime.

#### P2. One binary is a packaging choice, not an architecture choice
Kimbap should install as one thing, but internally remain modular:

- runtime
- vault
- policy
- audit
- oauth
- approval
- skill
- connectors
- modes

#### P3. Security claims must be precise
Kimbap must promise only what it can enforce:

- credentials do not appear in the agent process
- credentials still necessarily appear on outbound authenticated network requests
- long-lived refresh tokens remain server-side
- proxy and child-process behavior must be described honestly

#### P4. Canonical action namespace over transport details
The system should standardize on `service.action` rather than:

- raw URLs
- random CLI subcommands
- numeric admin codes
- SDK-specific method names

#### P5. Existing CLIs are adapters, not trust anchors
If a service CLI is used, it must be because Kimbap chose it as an execution strategy—not because the agent directly owns the CLI's auth state.

#### P6. Progressive adoption matters
Teams should be able to start with the least disruptive mode:

- proxy for zero-code-change HTTP
- run for subprocess agents
- call for explicit integration
- serve for shared/team deployment

#### P7. Self-hosted first
Kimbap v1 is self-hosted. No mandatory managed control plane.

#### P8. Teach agents and enforce at runtime
Prompt instructions alone are insufficient. Kimbap must provide:

- agent operating skills/profile snippets
- runtime wrappers (`run`, `proxy`)
- optional egress controls
- discoverability (`actions list`, `actions describe`)

---

## 4. Personas and Primary Jobs To Be Done

### 4.1 Primary personas

#### A. Agent Platform Engineer
Builds or operates agent systems in CI, production, or internal tools.

Needs:
- safe service access
- policy and approvals
- easy service onboarding
- stable automation surface
- auditability

#### B. Security / IT / Platform Governance Owner
Wants agents to be productive without letting them hold raw secrets.

Needs:
- tenancy and scoping
- credential isolation
- approval hooks
- logs and forensics
- revocation and rotation

#### C. Application Developer / Tooling Engineer
Wants to add services quickly.

Needs:
- OpenAPI/Postman -> action generation
- straightforward CLI
- local dev flow
- minimal server plumbing

#### D. Human Approver / Operator
Needs to approve risky actions quickly.

Needs:
- context-rich approval request
- Slack/Telegram/email delivery
- one-click approve/deny
- clear audit trail

### 4.2 Jobs to be done

1. "Run this agent with access to Stripe and GitHub without giving it those keys."
2. "Add a missing service that has only a REST API and no CLI."
3. "Let the agent read, but require approval for writes and destructive operations."
4. "Run long-lived agents without OAuth expiry failures."
5. "See exactly who/what did what, with correlation IDs and policy decisions."
6. "Make the agent use Kimbap by default instead of direct API calls."

---

## 5. Goals, Non-Goals, and Success Metrics

### 5.1 Primary goals

1. Any agent that runs a subprocess, makes HTTP calls, or invokes tools can adopt Kimbap with minimal or zero code changes.
2. New Tier 1 services can be integrated in minutes through skill YAML.
3. Raw credentials never appear in agent process env, prompts, shell history, or persisted traces.
4. Multi-tenant isolation is enforced through namespace, policy, and tenant-scoped key hierarchy.
5. OAuth token lifecycle is handled server-side so agents do not see token-expiry failures.
6. All surfaces converge on one action and audit model.

### 5.2 Non-goals for v1

- inbound MCP gateway replacement
- desktop companion app as a required control surface
- managed/cloud-hosted Kimbap
- WebSocket/gRPC/QUIC proxying
- Windows support
- workload identity as a launch requirement
- broad policy UI editing in the console
- executing arbitrary third-party CLIs as "safe" unless Kimbap fully owns their auth/runtime boundary

### 5.3 Success metrics

| Metric | Target | Measurement |
|---|---:|---|
| Time to first successful action | < 5 minutes from install to first call | Onboarding test |
| Tier 1 service authoring time | < 30 minutes from API spec to working action | Setup benchmark |
| Proxy adoption path | Existing HTTP agent works with zero code changes | Integration test |
| OAuth reliability | 0 surfaced token-expiry errors across 72h soak for supported connectors | Error logs |
| Credential safety | 0 raw credentials in agent env, logs, or prompt traces | Red-team / audit scan |
| Multi-tenant isolation | Token A cannot read or influence Token B's namespace | Security test |
| Policy evaluation latency | < 50 ms p99 | Benchmark |
| Action metadata consistency | Same request_id / trace fields across call, run, proxy, serve | Integration test |
| Approval workflow | Approver can decide from Slack/console and action resumes correctly | End-to-end test |

---

## 6. Product Definition

### 6.1 What Kimbap is

Kimbap is a **single-binary secure action runtime** that exposes one canonical action model through multiple interfaces.

### 6.2 What Kimbap is not

Kimbap is not:

- a new shell for all developer workflows
- a general API gateway for arbitrary enterprise traffic
- a replacement for Peta Core's MCP gateway
- a promise that every existing service CLI becomes agent-safe automatically

### 6.3 Product surfaces

#### Canonical surfaces
- `kimbap call`
- `kimbap run`
- `kimbap proxy`
- `kimbap serve`

#### Core supporting surfaces
- `kimbap actions`
- `kimbap vault`
- `kimbap token`
- `kimbap connector`
- `kimbap policy`
- `kimbap approve`
- `kimbap audit`
- `kimbap skill`
- `kimbap profile`
- `kimbap config`

---

## 7. User Experience: Primary Modes and Journeys

### 7.1 Mode A — Explicit action execution (`kimbap call`)

Best for:
- new integrations
- deterministic automation
- auditable workflows
- SDK-independent usage

Example:

```bash
kimbap call github.pull_requests.list \
  --repo dunia-labs/kimbap \
  --state open
```

Properties:
- fully typed action metadata
- explicit service/action namespace
- easiest to reason about for policy and audit
- preferred long-term integration model

### 7.2 Mode B — Run an existing agent (`kimbap run`)

Best for:
- coding agents
- subprocess-based agents
- local experimentation
- lifting an existing tool into a Kimbap-controlled environment

Example:

```bash
kimbap run --token <service-token> -- python agent.py
```

Responsibilities of `run`:
- provision proxy environment variables
- provision CA trust bundle or cert path
- establish/refresh Kimbap session
- inject only Kimbap-specific metadata, never downstream raw credentials
- optionally isolate working directories for wrapped tools when safe

`run` is **not** a blanket promise that any existing authenticated CLI becomes safe.  
It is the safe wrapper for an agent process that uses Kimbap-controlled action or network paths.

### 7.3 Mode C — Transparent proxy (`kimbap proxy`)

Best for:
- existing HTTP clients
- SDKs that respect `HTTP_PROXY` / `HTTPS_PROXY`
- zero-code-change adoption

Example:

```bash
kimbap proxy --port 10255
```

Properties:
- intercepts outbound HTTP/HTTPS
- classifies requests into canonical actions where possible
- injects credentials server-side
- enforces policy before request release
- writes audit records with request correlation

### 7.4 Mode D — Connected service (`kimbap serve`)

Best for:
- teams
- shared agents
- multi-tenant setups
- central policy / approval / audit
- REST/SDK clients

Example:

```bash
kimbap serve --port 8080
```

This exposes typed resource endpoints rather than a numeric action multiplexer.

---

## 8. System Overview

### 8.1 Canonical architecture

```text
┌──────────────────────────────────────────────────────────────────┐
│                    Kimbap Action Runtime (core)                 │
│  identity · registry · policy · vault · oauth · approval       │
│  executor · rate limiting · audit · request classification      │
└───────────────┬───────────────────────┬──────────────────────────┘
                │                       │
        ┌───────┴────────┐     ┌────────┴─────────┐
        │ Transport/UX    │     │ Integration      │
        │ Adapters        │     │ Adapters         │
        │                 │     │                  │
        │ - call          │     │ - skill YAML     │
        │ - run           │     │ - connector      │
        │ - proxy         │     │ - plugin binary  │
        │ - serve         │     │ - downstream svc │
        └─────────────────┘     └──────────────────┘
```

### 8.2 Request lifecycle

All modes converge on the same lifecycle:

1. Authenticate principal
2. Resolve tenant / agent / session context
3. Resolve action identity (`service.action`) or classify raw request into one
4. Validate arguments / resource selectors
5. Evaluate policy
6. If needed, create approval request and pause
7. Resolve credential material / refresh if needed
8. Execute action via selected adapter
9. Capture normalized result / error
10. Write audit record
11. Return result with correlation metadata

### 8.3 Why REST is not the core

If REST were canonical, `kimbap call` in embedded mode would HTTP into itself. That adds:

- unnecessary latency
- extra moving parts
- more failure modes
- duplicated auth and transport logic
- mismatch between local and connected behavior

Therefore:

- **Action Runtime** is canonical
- **REST is an adapter**
- embedded and connected modes must share the same semantics and audit schema

---

## 9. Embedded Mode vs Connected Mode

### 9.1 Embedded mode

Embedded mode is for local or single-user operation.

Characteristics:
- no always-on server required
- local encrypted vault
- local skill registry/cache
- local policy file
- local audit JSONL (and optionally local structured DB)
- ideal for development and laptop usage

Primary commands:
- `kimbap call`
- `kimbap run`
- `kimbap proxy`
- `kimbap vault unlock`

### 9.2 Connected mode

Connected mode is for shared/team operation.

Characteristics:
- `kimbap serve`
- multi-tenant
- central vault and connectors
- token issuance and revocation
- approval queue
- console integration
- durable audit store
- REST API for automation/SDKs

### 9.3 Parity requirement

Every capability exposed in connected mode must have a coherent embedded counterpart where feasible.

Examples:

| Capability | Embedded | Connected |
|---|---|---|
| `call` | in-process | REST adapter + runtime |
| vault | local encrypted store | central encrypted store |
| policy eval | local policy file | tenant policy in datastore |
| audit | JSONL | durable structured store |
| connectors | local login state | central connector lifecycle |
| approvals | optional local/manual | shared queue + adapters |

---

## 10. Detailed Functional Requirements

### FR-1. Canonical Action Model

Kimbap must model external capabilities as actions.

#### Requirements
- Every supported capability has a canonical name: `service.action`
- Actions may expose aliases, but policy, audit, and docs use the canonical name
- Actions declare:
  - description
  - input schema
  - output shape
  - auth requirements
  - risk category
  - idempotency behavior
  - approval hints
  - pagination/streaming semantics
- `kimbap actions list` and `kimbap actions describe` must be first-class discovery surfaces
- `kimbap actions validate` must validate inputs without side effects

#### Canonical naming rules
- stable over time
- transport-agnostic
- human-readable
- not tied to numeric enums or MCP tool names
- not tied to one SDK method name

### FR-2. Request Classification Layer

Proxy and run modes may begin with raw HTTP requests rather than explicit action calls. Kimbap must classify these into actions when possible.

#### Requirements
- Skills declare request matchers: host, method, path template, optional body selector
- A proxy request should map to a canonical action when an unambiguous matcher exists
- If no canonical action exists:
  - Kimbap may assign a generic pseudo-action such as `service.raw_request`
  - or deny by default in connected mode unless policy explicitly allows host/path rules
- Destructive or sensitive unmapped requests must not silently bypass action policy

#### Why this matters
This preserves one consistent policy and audit model across explicit and transparent modes.

### FR-3. CLI Command Surface

Kimbap must expose a coherent CLI taxonomy.

#### Canonical command families
- `call`
- `run`
- `proxy`
- `serve`
- `actions`
- `vault`
- `token`
- `auth`
- `connector`
- `policy`
- `approve`
- `audit`
- `skill`
- `profile`
- `config`
- `doctor`

#### Representative commands

| Command | Description |
|---|---|
| `kimbap call <service>.<action>` | Execute a registered action |
| `kimbap run -- <cmd>` | Run an agent process inside a Kimbap-controlled environment |
| `kimbap proxy --port 10255` | Start the transparent proxy |
| `kimbap serve --port 8080` | Start connected-mode REST service |
| `kimbap connector login <service>` | Perform downstream OAuth/device login |
| `kimbap actions list [--service]` | List actions |
| `kimbap actions describe <service.action>` | Show schema, auth, risk, examples |
| `kimbap vault set <key> --file value.txt` | Store a credential safely |
| `kimbap token create --agent billing-bot` | Issue a service token |
| `kimbap policy eval --agent billing-bot --action stripe.refund` | Simulate policy decision |
| `kimbap approve <request-id>` | Approve a held action |
| `kimbap skill install github:user/repo/skill` | Install a skill |
| `kimbap profile install claude-code` | Install agent operating instructions/profile |
| `kimbap doctor proxy` | Diagnose CA/proxy/env issues |

### FR-4. Structured result envelope

All action responses must include normalized metadata.

Example:

```json
{
  "result": {
    "id": "pr_123",
    "title": "Fix token rotation"
  },
  "_meta": {
    "request_id": "req_01HXYZ...",
    "trace_id": "tr_01HXYZ...",
    "tenant": "acme",
    "agent": "billing-bot",
    "service": "github",
    "action": "pull_requests.list",
    "latency_ms": 142,
    "policy_decision": "allow"
  }
}
```

### FR-5. Idempotency

Mutating actions must support idempotency where semantically possible.

#### Requirements
- Actions declare whether they are:
  - read-only
  - idempotent mutation
  - non-idempotent mutation
- High-risk or destructive actions require an idempotency key
- Audit records must store the idempotency key when provided
- Duplicate retries must not cause duplicate external mutations when the downstream service supports idempotency

### FR-6. Vault

Kimbap must store credentials securely and inject them only at execution time.

#### Requirements
- AES-256-GCM encryption at rest
- record-level encryption
- support for:
  - static API keys
  - usernames/passwords
  - bearer tokens
  - OAuth client credentials
  - downstream refresh tokens
  - certificate or file-based secrets (roadmap extension)
- `vault set` must not accept inline secret values in CLI args
- `vault get` must be masked by default
- values must never appear in audit logs
- last-used and last-rotated metadata must be tracked

### FR-7. Downstream OAuth brokerage

Kimbap must own downstream provider OAuth lifecycle.

#### Requirements
- connector login flow for supported services
- encrypted storage of refresh tokens and client secrets
- automatic access-token refresh before use or on expiry
- refresh retry/backoff handling
- forced refresh command
- revocation support where provider supports it
- agents must never directly receive refresh tokens
- wherever feasible, agents should not directly receive access tokens either

### FR-8. Policy Engine

Kimbap must evaluate policy on every execution.

#### Decision outcomes
- `allow`
- `deny`
- `require_approval`

#### Requirements
- rules must be expressible against:
  - tenant
  - agent
  - service
  - action
  - risk class
  - resource selector
  - method / path matcher (for proxy fallback)
  - time / rate limits
  - parameter constraints
- `policy eval` must be a pure simulation
- policy must exist independent of transport mode
- rate limiting and quotas must be part of policy evaluation
- deny-by-default must be supported in connected mode

### FR-9. Approval workflow

Kimbap must support human-in-the-loop for risky actions.

#### Requirements
- durable approval request record
- lifecycle states:
  - pending
  - approved
  - denied
  - expired
  - cancelled
  - executed
- adapters:
  - webhook
  - Slack
  - Telegram
  - email
  - future WhatsApp / others
- action execution must pause and resume safely
- approval payload must include:
  - agent
  - tenant
  - action
  - summarized parameters
  - risk level
  - policy reason
  - correlation IDs
- audit must record requester, approver, decision, and timing

### FR-10. Audit trail

Kimbap must create a durable record for every action attempt and decision.

#### Required fields
- request_id
- trace_id
- timestamp
- tenant
- agent
- auth mechanism
- mode (`call`, `run`, `proxy`, `serve`)
- service
- action
- raw request classification info (if proxy)
- policy decision
- approval status
- execution status
- latency
- redacted parameter summary
- downstream error class if applicable

### FR-11. Skill system

Kimbap must support declarative service integration.

#### Skill types
1. **Service Skill** — turns an external service/API into Kimbap actions
2. **Agent Operating Skill / Profile** — teaches an agent to use Kimbap and avoid raw credentials

### FR-12. Agent operating profile system

Kimbap must help users tell agents to use Kimbap correctly.

#### Requirements
- generate installable instruction bundles for popular agent environments
- output to standard locations such as:
  - `.agents/skills/...`
  - `.claude/skills/...`
  - `AGENTS.md` snippets
  - shell/env setup files
- include mandatory instructions such as:
  - discover with `kimbap actions list`
  - inspect with `kimbap actions describe`
  - execute with `kimbap call`
  - prefer `kimbap run` / `kimbap proxy` for legacy tools
  - never request or print raw credentials
  - request a new Kimbap integration instead of using a raw API key

### FR-13. Console

Kimbap Console must exist, but with a narrower scope than `peta-console`.

#### Console should include
- approval UI
- audit viewer
- usage dashboard
- active agent / token status
- read-only policy viewer
- read-only vault key names
- read-only skill inventory

#### Console should not be the primary place for
- skill authoring
- policy editing
- token issuance
- raw credential management

Those remain CLI / config driven.

### FR-14. Existing CLI integration strategy

Kimbap must support existing service CLIs only under explicit, safe models.

#### Allowed models
1. **Internal executor adapter**  
   Kimbap itself uses a service CLI under the hood to implement an action.

2. **Proxy-mediated subprocess compatibility**  
   A CLI works through proxy mode without direct access to real credentials.

3. **Explicit plugin adapter**  
   A service-specific adapter manages the CLI's auth/runtime boundary safely.

#### Disallowed default assumption
> "Any existing CLI can just serve as the vault role."

This is false as a general rule.  
If the agent can inspect the CLI's config directory or auth files, the boundary is broken.

---

## 11. Architecture and Repository Structure

### 11.1 Proposed repository structure

```text
kimbap/
  cmd/
    kimbap/
  internal/
    runtime/
    identity/
    registry/
    classifier/
    executor/
    vault/
    policy/
    approval/
    audit/
    oauth/
    rate/
    skill/
    connector/
    session/
    modes/
      call/
      run/
      proxy/
      serve/
  console/
  skills/
    official/
  examples/
  docs/
```

### 11.2 Key internal modules

#### `runtime/`
Canonical action execution orchestration.

#### `registry/`
Action and skill metadata registry.

#### `classifier/`
Maps raw requests or legacy inputs into canonical action identities.

#### `vault/`
Encryption, storage, rotation, metadata.

#### `oauth/`
Downstream connector auth lifecycle.

#### `policy/`
Decision engine, dry-run simulation, quotas.

#### `approval/`
Queue, adapters, resume semantics.

#### `audit/`
Structured event schema and export.

#### `modes/*`
Transport and UX adapters only.

---

## 12. Identity, Authentication, and Session Model

### 12.1 Core identity entities

| Entity | Purpose |
|---|---|
| Tenant | Top-level namespace and isolation boundary |
| Agent principal | Named automation identity within a tenant |
| Operator | Human/admin identity managing Kimbap |
| Service token | Long-lived opaque credential used to bootstrap or represent an agent |
| Session token | Short-lived runtime credential used by active clients/processes |
| Connector session | Downstream provider auth state |
| Approval actor | Human who approved or denied a request |

### 12.2 Authentication flows

#### Flow A — Service token (critical for v1)
Operator creates a service token for a headless agent.

Use cases:
- CI jobs
- daemonized agents
- production workers

Properties:
- opaque
- display-once
- stored hashed at rest
- bound to tenant + agent + scopes
- rotatable and revocable

#### Flow B — Session token exchange
A service token may be exchanged for a short-lived session token.

Why:
- avoid sending long-lived credentials on every request
- support narrower audiences (`proxy`, `rest`, `cli`)
- allow shorter TTLs for active sessions

Default:
- session TTL 1 hour
- renewable only through a still-valid service token or trusted workload identity

#### Flow C — Connector login (device/browser)
Human performs provider login to bootstrap downstream OAuth state.

Example:
```bash
kimbap connector login gmail
```

#### Flow D — Embedded vault unlock
Local session unlock for embedded mode.

Examples:
- passphrase
- OS keychain
- env-provided master material (carefully bounded)

#### Flow E — Workload identity (future)
SPIFFE/SPIRE or equivalent for enterprise environments.

### 12.3 Why service token and session token are separate

The original instinct of "default all agent tokens to 1h TTL" conflicts with long-running headless agents.

Kimbap therefore separates:

- **service token** — durable identity bootstrap
- **session token** — short-lived runtime credential

This allows:
- headless agents to exist
- short-lived runtime credentials to exist
- better revocation granularity
- safer transport in proxy and REST modes

### 12.4 Authentication requirements

- all tokens are tenant-scoped
- token values are never used directly as encryption keys
- tokens must support revocation, inspection, last-used, and rotation
- token issuance must record actor, time, scopes, and expiry
- tokens must be audience-bound where appropriate

---

## 13. Trust Boundary and Security Model

### 13.1 Threat model

Kimbap assumes the agent process may be:

- prompt-influenced
- able to inspect its own environment
- able to read files it can access
- able to log or exfiltrate outputs
- able to call arbitrary HTTP destinations unless constrained

Kimbap does **not** assume the agent is fully trusted.

### 13.2 Security objectives

1. Keep raw credentials out of the agent process.
2. Ensure long-lived provider credentials remain in Kimbap-controlled storage.
3. Enforce policy and approvals before external mutation.
4. Preserve tenant isolation.
5. Produce forensically useful audit trails.
6. Make security guarantees explicit and testable.

### 13.3 Precise trust claims

#### Guaranteed on the agent side
Raw credentials must not appear in:
- agent env vars
- prompt context injected by Kimbap
- shell history
- CLI argument lists
- persisted traces or audit payloads
- generated code or config files created by Kimbap for the agent

#### Guaranteed on the Kimbap side
- long-lived static secrets are encrypted at rest
- downstream refresh tokens are encrypted at rest
- credentials are injected only at execution time
- secrets are never written to audit logs

#### What Kimbap cannot honestly claim
The final outbound authenticated request to the external service necessarily carries authentication material. The security property is that **the agent does not get to own or persist it**.

### 13.4 Proxy-side handling

Kimbap should describe proxy behavior precisely:

- for a proxied request, Kimbap may hold the injected credential in memory long enough to construct and forward the outbound request
- Kimbap may cache classification or policy results briefly
- Kimbap must not persist raw injected credentials in logs or durable stores
- long-lived refresh tokens and primary secret material remain in the vault side of the system, not in the agent

### 13.5 Existing CLI handling

Kimbap must not claim that direct use of a human-oriented CLI is safe unless:

- Kimbap owns the subprocess
- Kimbap owns the auth material path
- the agent cannot inspect or retain that auth material
- audit/policy remain enforceable

This is a deliberate product boundary.

---

## 14. Multi-Tenant Isolation and Cryptography

### 14.1 Tenant isolation model

Tenant isolation must use three independent layers:

1. **Namespace isolation**  
   Every credential, policy, token, approval request, connector, and audit record belongs to a tenant namespace.

2. **Policy isolation**  
   Every decision is evaluated in tenant context.

3. **Key hierarchy isolation**  
   Each tenant has an independent encryption boundary.

### 14.2 Crypto model

Kimbap v1 should use:

- AES-256-GCM for data encryption
- per-record DEKs
- tenant-scoped KEKs
- a root wrapping key or KMS-backed master material
- authenticated encryption only
- fresh nonce/IV per encryption
- integrity failure on tamper

### 14.3 Why token and crypto boundaries must be separate

Token compromise and key rotation are different problems.

If the token determines the encryption boundary:
- revoking a token may force re-encryption
- rotation becomes operationally expensive
- token design leaks into storage architecture

Kimbap must decouple:
- **identity/authz** = token model
- **encryption boundary** = tenant KEK/DEK model

### 14.4 Rotation requirements

Kimbap must support:
- service token rotation
- session token expiry/renewal
- tenant KEK rotation
- record re-encryption workflow
- provider refresh token rotation where supported
- last-used / rotated metadata

---

## 15. Policy Model

### 15.1 Policy language choice

For v1, Kimbap should use a **custom YAML/JSON DSL with an internal AST**, not OPA/Rego as the primary user-facing language.

Rationale:
- easier onboarding
- tighter fit to action-oriented semantics
- lower implementation and support burden
- can later compile or bridge to Rego if needed

### 15.2 Policy objects

Policies must be able to match on:

- tenant
- agent
- service
- action
- risk class
- resource selector
- host/path/method (for proxy fallback)
- environment (dev/staging/prod)
- parameter constraints
- time windows
- rate limits

### 15.3 Policy outcomes

- `allow`
- `deny`
- `require_approval`

### 15.4 Example policy

```yaml
version: "1.0.0"

rules:
  - id: allow-github-read
    priority: 10
    match:
      tenants: [acme]
      agents: [repo-bot]
      actions:
        - github.pull_requests.list
        - github.issues.list
        - github.repos.get
    decision: allow

  - id: approve-stripe-refunds
    priority: 20
    match:
      tenants: [acme]
      agents: [billing-bot]
      actions:
        - stripe.refunds.create
    conditions:
      - field: input.amount
        operator: lte
        value: 10000
    decision: require_approval

  - id: block-raw-unclassified-posts
    priority: 30
    match:
      actions: ["*.raw_request"]
    decision: deny

  - id: github-rate-limit
    priority: 40
    match:
      tenants: [acme]
      services: [github]
    rate_limit:
      max_requests: 300
      window_sec: 3600
      scope: tenant
    decision: allow
```

### 15.5 Policy requirements

- evaluation must be deterministic
- explanation output must show matched rules
- dry-run must return the same decision path as execution, minus side effects
- policies must be exportable/importable as files
- console remains read-only for v1 policy visualization

---

## 16. Human-in-the-Loop (HITL) Architecture

### 16.1 HITL as a two-channel problem

HITL requires two distinct capabilities:

1. **notification channel** — tell a human that approval is needed
2. **decision channel** — receive and authenticate the human's decision

Kimbap must model these separately.

### 16.2 Approval flow

```text
Agent requests action
  -> policy says require_approval
  -> Kimbap creates approval record
  -> notification adapter sends message
  -> human approves/denies via console, CLI, or signed link
  -> Kimbap resumes or rejects execution
  -> audit records full lifecycle
```

### 16.3 Messaging adapters

| Adapter | Purpose | Phase |
|---|---|---|
| Generic webhook | Lowest-common-denominator integration | Phase 1 |
| Slack | Primary team workflow | Phase 2 |
| Telegram | Lightweight ops / startup / international workflow | Phase 2 |
| Email | Fallback channel | Phase 2 |
| WhatsApp | Regional/field-ops workflow | Phase 3 |

### 16.4 Approval requirements

- requests must expire
- signed links must be one-time or replay-safe
- decision actor must be authenticated and recorded
- agent should receive a deterministic final outcome
- duplicate approvals/denials must not double-execute the action

---

## 17. Service Integration Model

### 17.1 Core idea

Many services do not have an official CLI, but they do have an API.  
Kimbap must turn those APIs into reusable agent-safe actions.

### 17.2 Three integration tiers

Tiers are based on **runtime requirements**, not prestige or business importance.

#### Tier 1 — Declarative skill YAML
Use when the service can be expressed as request/response with static or derivable auth.

Examples:
- bearer key
- API key header
- query param auth
- basic auth
- custom header formatting
- AWS SigV4 patterns where runtime can sign from stored credentials

Typical services:
- Notion
- Linear
- Airtable
- Zendesk
- Intercom
- Figma
- Canva
- many internal REST APIs
- Stripe for standard key-based API usage
- GitHub REST for PAT-based/common cases

#### Tier 2 — Connector / plugin adapter
Use when a service needs a managed auth lifecycle or provider-specific runtime logic.

Examples:
- OAuth 2.0 with refresh lifecycle
- provider-specific token exchange
- app installation flows
- provider-specific pagination or throttling semantics that are too bespoke for simple YAML

Typical services:
- Gmail
- Google Calendar
- Google Docs
- Google Drive
- Google Sheets
- Slack
- Teams
- HubSpot
- GitHub Apps
- Stripe Connect edge cases

#### Tier 3 — Downstream stateful runtime
Use only when request/response is genuinely insufficient.

Examples:
- databases with connection pooling
- streaming protocols
- stateful sessions
- domains requiring long-lived protocol management

Typical services:
- PostgreSQL
- MySQL
- long-lived streaming backends

### 17.3 Tiering rules

A service belongs in Tier 3 **only if** it truly requires:
- stateful sessions
- streaming
- protocol-specific runtime behavior
- or a workflow not reducible to action-oriented request/response

This bar must stay high.

### 17.4 Existing CLI role in the tier model

An existing service CLI may appear in any tier only as an **execution strategy**, not as the top-level product abstraction.

Examples:
- Tier 1 or 2 action implemented internally by calling a service CLI
- plugin adapter that shells out to an official CLI safely
- never "just let the agent own the CLI login state"

---


### 17.5 Why Stripe and GitHub belong in Tier 1 for common cases

Kimbap should explicitly classify the common cases for Stripe and GitHub as Tier 1.

#### Stripe
Standard Stripe usage is API-key based request/response HTTP. The difficulty is not transport complexity; it is **risk**. Refunds, deletions, and financial mutations should therefore be handled by:

- policy
- approval
- idempotency
- audit

They do **not** automatically justify a custom server or a Tier 3 implementation.

#### GitHub
Most common GitHub REST usage with a PAT or similar bearer credential is also request/response HTTP and belongs in Tier 1.  
GitHub Apps installation-token flows and some enterprise-specific auth patterns can remain Tier 2 where lifecycle logic is required.

### 17.6 How services without an official CLI become Kimbap commands

A service that lacks an official CLI should become a **Kimbap action surface**, not a one-off bespoke binary.

Preferred model:

- canonical form: `kimbap call service.action`
- optional sugar: `kimbap service action ...` or repo-specific aliases later
- audit, policy, approval, and SDK surfaces always anchor to the canonical action name

Examples:

```bash
kimbap call notion.pages.create --title "Q2 planning"
kimbap call linear.issues.create --title "Fix token refresh"
```

This avoids command sprawl and keeps all integrations in one shared namespace.


## 18. Skill YAML Specification

### 18.1 Skill concepts

A Service Skill must define:

- metadata
- base URL or transport target
- auth model
- optional connector reference
- actions
- request templates
- argument schemas
- response extraction
- pagination
- error mapping
- risk / approval hints
- request classification matchers for proxy mode

### 18.2 Example skill

```yaml
name: linear
version: 1.0.0
base_url: https://api.linear.app

auth:
  type: bearer
  credential: LINEAR_KEY

error_mapping:
  401: credential_invalid_or_expired
  429: rate_limited

classifiers:
  - service: linear
    action: issues.list
    method: GET
    path: /graphql

actions:
  issues.list:
    method: GET
    path: /graphql
    description: List issues
    risk: read
    args:
      - name: team_id
        type: string
        required: false
    response:
      extract: ".data.issues.nodes.[].{id,title,state}"

  issues.create:
    method: POST
    path: /graphql
    description: Create an issue
    risk: write
    approval_hint: optional
    args:
      - name: title
        type: string
        required: true
      - name: team_id
        type: string
        required: true
    idempotency:
      required: true
      key_field: title
    response:
      extract: ".data.issueCreate.issue.{id,identifier,title,url}"
```

### 18.3 Auth types supported by the runtime

| Auth type | Description |
|---|---|
| `bearer` | `Authorization: Bearer ...` |
| `header` | arbitrary header injection |
| `query_param` | query-string auth |
| `basic` | username/password auth |
| `oauth2` | provider connector + token refresh |
| `aws_sig_v4` | signed AWS requests |
| `custom` | non-standard mappings |
| `mtls` | roadmap / advanced |
| `file` | roadmap / advanced |

### 18.4 Important clarification

"Runtime supports `oauth2`" does **not** mean every OAuth-backed service is "YAML only."

For many services, the skill YAML will:
- define actions
- declare connector reference
- define error mappings

But a provider connector/plugin still owns the lifecycle:
- login
- refresh
- revocation
- provider quirks

### 18.5 Skill validation

Kimbap must provide:
- linting
- schema validation
- dry-run request rendering
- classifier validation
- install-time diff of actions introduced or changed

---

## 19. Generation Pipeline

### 19.1 Generation sources

| Source | Support | Notes |
|---|---|---|
| OpenAPI | Phase 1 | Primary generation path |
| Postman Collection | Phase 3 | Useful for APIs without good OpenAPI |
| Manual authoring | Phase 1 | 20–100 lines for many Tier 1 services |
| Docs URL -> AI draft | Phase 4 | Human review required |

### 19.2 Generator requirements

Generated skills must be:
- editable
- deterministic
- reviewable
- lintable
- safe to diff in PRs

### 19.3 Human review requirement

AI-generated or spec-generated skills must still be reviewable before use in sensitive environments.

---

## 20. Skill Registry and Trust Model

### 20.1 Registry types

| Registry type | Owner | Use case |
|---|---|---|
| Official | Dunia Labs | trusted/common services |
| Community | GitHub-based | open ecosystem |
| Private | organization | internal APIs |

### 20.2 Supply-chain requirements

Kimbap must not treat "YAML is not executable" as sufficient supply-chain safety.

Required mechanisms:
- install lockfile
- digest pinning
- signed official packages
- publisher identity metadata
- capability diff on install/upgrade
- destructive action warnings
- explicit trust prompts for community packages

### 20.3 Lockfile behavior

The installed state must record:
- skill name
- version
- digest
- source
- signer / publisher identity where applicable

### 20.4 Plugin/binary safety

If plugin binaries are supported:
- they must be signed
- execution policy must be stricter than for YAML
- official/private trust channels only in early phases

---

## 21. Agent Operating Skills and Profiles

### 21.1 Problem

A service skill tells Kimbap how to talk to a service.  
It does **not** by itself tell the agent to use Kimbap.

Kimbap therefore needs a second artifact type:

- **Agent Operating Skill / Profile**

### 21.2 Purpose

This artifact teaches the agent:

- do not call third-party APIs directly if Kimbap can do it
- do not request raw credentials
- use `kimbap actions list` and `kimbap actions describe`
- execute through `kimbap call`
- use `kimbap run` / `kimbap proxy` for legacy environments
- ask for a new Kimbap integration instead of a raw API key

### 21.3 Example agent operating skill

```md
# Kimbap Agent Operating Skill

- Use Kimbap for all third-party actions whenever available.
- Discover capabilities with `kimbap actions list` and `kimbap actions describe`.
- Execute external operations with `kimbap call <service>.<action>`.
- For legacy codebases, prefer running through `kimbap run` or the configured Kimbap proxy.
- Never ask for, print, store, or pass raw API keys, passwords, refresh tokens, or cookie values.
- If the required action does not exist, request a new Kimbap integration instead of using direct credentials.
```

### 21.4 Installation requirements

Kimbap should ship commands such as:

```bash
kimbap profile install claude-code
kimbap profile install generic
kimbap profile print cursor
```

Expected outputs may include:
- `.agents/skills/kimbap-operating/SKILL.md`
- `.claude/skills/kimbap-operating/SKILL.md`
- `AGENTS.md` snippet
- shell initialization for proxy env
- docs snippet for CI or repo onboarding

### 21.5 Why prompts are not enough

Instruction alone is weak. Kimbap must pair it with runtime control:

- `run`
- `proxy`
- session tokens
- optional egress restrictions
- discoverable action inventory

### 21.6 Enforcement options

#### v1
- process wrapper (`run`)
- proxy environment variables
- Kimbap-only session tokens
- clear operating skill/profile

#### Later
- egress allowlisting
- PATH shims
- container-level networking policy
- organization-wide policy bundles

---

## 22. Existing CLIs: Product Policy

### 22.1 Direct answer to the product question

**No, existing CLIs are not "just the vault role."**  
They are, at most, one possible **execution adapter** inside Kimbap.

### 22.2 Preferred order of implementation

1. **Direct API action via skill YAML** — preferred
2. **Connector/plugin adapter** — for lifecycle-heavy providers
3. **Internal CLI executor** — only when it gives real leverage and Kimbap can preserve the trust boundary

### 22.3 Safe existing-CLI usage requirements

If Kimbap internally uses a service CLI:
- the agent must not own the CLI's login state
- auth material must not be visible to the agent
- temp auth/config dirs must be Kimbap-controlled and scrubbed
- policy and audit must wrap the call
- CLI stdout/stderr must be sanitized/redacted if necessary
- the CLI cannot become a side door around approval/policy

### 22.4 Unsafe usage patterns to reject

- "Tell the agent to run `gh auth login`."
- "Mount a token file and let the agent call the vendor CLI directly."
- "Store provider sessions in shared home directories."
- "Bypass Kimbap audit by letting agents use CLIs outside Kimbap."

---

## 23. REST API Surface (`kimbap serve`)

### 23.1 Design principle

Do not reuse `/admin` numeric action routing.

Use typed, resource-oriented endpoints.

### 23.2 Representative resources

- `POST /v1/actions/{service.action}:invoke`
- `GET /v1/actions`
- `GET /v1/actions/{service.action}`
- `POST /v1/tokens`
- `GET /v1/tokens`
- `POST /v1/connectors/{service}/login`
- `POST /v1/policy/evaluate`
- `GET /v1/audit`
- `POST /v1/approvals/{id}:approve`
- `POST /v1/approvals/{id}:deny`

### 23.3 API requirements

- versioned pathing
- OpenAPI spec
- correlation IDs in headers and body
- typed errors
- consistent auth model
- no implicit side effects in evaluation endpoints

---

## 24. Proxy Scope and Limitations

### 24.1 Supported in v1

- standard HTTP/1.1 and HTTP/2 clients that honor proxy env vars
- HTTPS MITM with trusted local CA
- request/response interception
- host/path-based classification
- header/query/body credential injection where configured
- per-request policy and audit

### 24.2 Not supported in v1

- certificate pinning
- WebSocket
- gRPC over HTTP/2 with full protocol awareness
- QUIC / HTTP/3
- arbitrary raw TCP
- SSH
- database wire protocols through the proxy
- tunneled protocols requiring opaque CONNECT pass-through only

### 24.3 Precise "zero code changes" statement

"Zero code changes" applies to agents that:

- use standard HTTP stacks
- honor `HTTP_PROXY` / `HTTPS_PROXY`
- do not pin certs
- do not depend on unsupported protocols

This statement must not be broadened beyond that.

### 24.4 CA trust UX

Kimbap must support at least one of:

- local CA installation
- per-process trust bundle via `SSL_CERT_FILE`
- OS-specific install helpers

A `doctor` command is required to reduce support burden.

---

## 25. Kimbap Console

### 25.1 Scope

Kimbap Console is the narrowed successor to `peta-console` for Kimbap use cases.

### 25.2 What stays

- approval queue UI
- audit viewer
- usage dashboards
- token status / last used
- active runtime status
- read-only policy visualization
- read-only skill inventory

### 25.3 What moves to CLI/config

- skill creation and editing
- policy authoring
- token issuance and rotation
- credential value management
- connector bootstrap flows where CLI/device flow is better

### 25.4 Why this matters

The console should help operators observe and decide—not become the primary authoring surface for everything.

---

## 26. Observability, Audit, and Reliability

### 26.1 Observability requirements

Kimbap must provide:
- structured logs
- audit records
- correlation IDs
- latency metrics
- error-class metrics
- rate-limit visibility
- approval queue metrics
- connector refresh health metrics

### 26.2 Reliability requirements

- downstream retries where safe
- connector refresh retry/backoff
- durable approval queue
- idempotent resume behavior
- graceful degradation when a connector is temporarily unavailable
- clear operator alerts for refresh or policy failures

### 26.3 Audit export

Supported exports:
- JSONL
- CSV
- filtered by tenant, agent, service, action, date range, decision

---

## 27. Performance and Operational Requirements

### 27.1 Performance targets

- policy eval < 50 ms p99
- Kimbap overhead before downstream network call < 25 ms p50 for cached classifications
- approval decision resume < 2 s after decision receipt (excluding downstream latency)
- token introspection/validation suitable for high-frequency proxy use

### 27.2 Operational requirements

- single-binary install for core runtime
- macOS and Linux support in v1
- deterministic config loading
- safe startup defaults
- clear `doctor` diagnostics
- embedded mode without mandatory external DB
- connected mode with durable production store

### 27.3 Storage model

#### Embedded mode
- local encrypted store
- local policy files
- local audit JSONL or SQLite-backed local store

#### Connected mode
- durable database for metadata, tokens, approvals, audit
- local or external KMS/key material provider for encryption roots
- filesystem/object storage for larger artifacts if needed later

---

## 28. Roadmap

### Phase 1 — Core Runtime (Weeks 1–5)

**Objective:** establish the Action Runtime and safe local workflows.

Deliver:
- fork/extract from `peta-core-go`
- remove MCP inbound surface
- remove numeric `/admin` style control plane
- implement canonical `internal/runtime`
- embedded mode
- `kimbap call`
- `kimbap actions list/describe/validate`
- `vault` commands
- service tokens
- session model skeleton
- policy DSL + `policy eval`
- Tier 1 skill parser and executor
- OpenAPI generator
- audit JSONL
- `doctor` basics

### Phase 2 — Proxy, Connectors, Connected Mode (Weeks 6–10)

**Objective:** deliver zero-code-change adoption and central/team operation.

Deliver:
- `kimbap proxy`
- `kimbap run`
- request classification layer
- HTTPS MITM + CA UX
- `kimbap serve`
- durable datastore
- multi-tenant namespaces
- connector login/refresh for Tier 2 services
- auth/session exchange
- REST API v1
- policy persistence
- connected audit store
- `kimbap auth login/status`

### Phase 3 — Approvals, Console, Ecosystem (Weeks 11–15)

**Objective:** add governance and ecosystem usability.

Deliver:
- approval queue
- webhook adapter
- Slack/Telegram/email adapters
- Kimbap Console
- skill install/list/remove
- install lockfile and digest pinning
- private registry
- Postman generator
- token rotation
- tenant KEK rotation flow
- agent operating profile installer

### Phase 4 — Enterprise and Compatibility (Weeks 16–20)

**Objective:** add enterprise features and optional compatibility layers.

Deliver:
- official/community registry UX
- AI-assisted docs-to-skill draft generation
- TypeScript/Python SDKs over REST
- optional MCP adapter for Kimbap actions
- workload identity
- SSO / SAML
- external vault provider adapters
- more advanced egress enforcement

---

## 29. Open Questions

1. **Proxy trust UX**  
   Local CA installation vs per-process trust bundle: which becomes the default?
2. **Embedded vault unlock UX**  
   Passphrase, OS keychain, or both?
3. **Policy language evolution**  
   When, if ever, should Rego be added beneath or alongside the native DSL?
4. **Classifier strictness**  
   Should unmapped proxy requests default to deny in all connected-mode environments?
5. **Official plugin policy**  
   How restrictive should binary plugin trust be in v1/v2?
6. **GitHub Apps / Stripe Connect exact boundary**  
   Which cases stay Tier 2 vs can be collapsed into declarative connectors?
7. **MCP compatibility**  
   When should Kimbap expose an MCP adapter, and how much semantic compatibility is required?
8. **External vault providers**  
   Which should be first: Bitwarden, 1Password, AWS Secrets Manager, or others?

---

## 30. Risks and Mitigations

| Risk | Why it matters | Mitigation |
|---|---|---|
| Product confusion with Peta Core | Market and internal roadmap confusion | Position Kimbap clearly as CLI/action runtime, not MCP gateway |
| Overpromising proxy support | User trust erosion | Narrowly define supported protocols and "zero code changes" |
| YAML skill oversimplification | Users think everything is 30 minutes | Tie KPI to Tier 1 only; define Tier 2 clearly |
| Existing CLI boundary leaks | Breaks core security promise | Treat existing CLIs as internal adapters only |
| Registry supply-chain risk | Unsafe community skills | Signing, lockfiles, diffing, trust prompts |
| OAuth connector sprawl | Maintenance burden | Shared connector framework and tier gating |
| Policy DSL too weak | Need for custom escape hatches | Internal AST; future bridge to Rego |
| Embedded vs connected drift | Inconsistent behavior | Runtime-first architecture and parity tests |

---

## 31. Acceptance Criteria

### 31.1 v1 core acceptance

- A user can install one binary and execute a Tier 1 action locally.
- No raw credential is present in the agent environment or CLI arguments.
- `actions list/describe/validate` work for installed skills.
- `policy eval` returns deterministic allow/deny/require_approval.
- Audit JSONL contains request IDs and decision metadata.
- `vault set` accepts stdin/file only.
- Service tokens can be issued, inspected, revoked.

### 31.2 proxy acceptance

- An existing HTTP-based agent can be run through `kimbap run` or `kimbap proxy` without source changes.
- Kimbap can classify at least a subset of requests into canonical actions.
- Unsupported proxy cases fail clearly.
- Per-request audit contains mode=proxy and request classification metadata.

### 31.3 connector acceptance

- A supported OAuth service can be logged into via device flow.
- Refresh occurs without the agent seeing refresh tokens.
- Access-token expiry is recovered automatically in supported happy paths.

### 31.4 approval acceptance

- A policy can require approval for a write action.
- Slack or webhook can deliver the approval request.
- Approval or denial resumes the held request deterministically.
- Audit captures requester, approver, decision, and timestamps.

---

## 32. Relationship to Peta Core and OneCLI

### 32.1 What Kimbap keeps from `peta-core-go`

- vault
- downstream OAuth brokerage
- policy engine
- approval concepts
- audit schema
- runtime supervision
- REST adapter / action templating
- skill packaging concepts

### 32.2 What Kimbap removes or de-centers

- inbound MCP as the main product surface
- reverse routing
- request-id mapping for upstream/downstream MCP sessions
- event resumption as a core requirement
- Webhook delivery and console polling as default notification dependency
- numeric `/admin` control plane
- per-client capability filtering as the main policy model
- desktop app as the main approval surface

### 32.3 What Kimbap learns from OneCLI

Adopt:
- proxy as adoption wedge
- agent token model
- host/path matching
- strong gateway hot path
- external vault integration mindset
- repo-local agent skill pattern

Do not inherit:
- credential swap as the only model
- shallow `block/rate_limit`-only policy
- dashboard-first product center
- host/path-only semantics without canonical actions

---

## 33. Appendix A — Example Command Taxonomy

```bash
# Core execution
kimbap call github.pull_requests.list --repo dunia-labs/kimbap
kimbap run -- python agent.py
kimbap proxy --port 10255
kimbap serve --port 8080

# Discovery
kimbap actions list --service github
kimbap actions describe stripe.refunds.create
kimbap actions validate stripe.refunds.create --charge ch_123 --amount 500

# Vault
kimbap vault set STRIPE_KEY --file stripe_key.txt
printf '%s' "$LINEAR_KEY" | kimbap vault set LINEAR_KEY --stdin
kimbap vault list
kimbap vault get STRIPE_KEY
kimbap vault rotate STRIPE_KEY
kimbap vault unlock

# Identity
kimbap token create --agent billing-bot --ttl 30d
kimbap token inspect tok_123
kimbap token revoke tok_123
kimbap token rotate tok_123
kimbap auth login --server https://kimbap.internal
kimbap auth status

# Connectors
kimbap connector login gmail
kimbap connector list
kimbap connector refresh gmail

# Policy
kimbap policy set --file policy.yaml
kimbap policy get
kimbap policy list
kimbap policy eval --agent billing-bot --action stripe.refunds.create --params params.json

# Approval
kimbap approve list
kimbap approve req_123
kimbap deny req_123 --reason "Amount exceeds approved threshold"

# Audit
kimbap audit tail --agent billing-bot
kimbap audit export --from 2026-03-01 --to 2026-03-31 --format jsonl

# Skills
kimbap skill generate --openapi https://api.example.com/openapi.json
kimbap skill lint ./skills/linear.yaml
kimbap skill install github:dunialabs/kimbap-skills/linear
kimbap skill list
kimbap skill remove linear

# Agent profiles
kimbap profile install claude-code
kimbap profile install generic
kimbap profile print generic

# Diagnostics
kimbap doctor
kimbap doctor proxy
kimbap doctor connectors
```

---

## 34. Appendix B — Example Agent Operating Profile Output

```md
# Kimbap Operating Rules for Agents

1. Use Kimbap for external service access whenever possible.
2. Discover available actions with `kimbap actions list`.
3. Inspect an action before using it with `kimbap actions describe <service.action>`.
4. Execute via `kimbap call <service.action>`.
5. For legacy apps or scripts, prefer `kimbap run` or a configured Kimbap proxy.
6. Never ask for, print, or store raw API keys, passwords, refresh tokens, cookies, or session files.
7. If the needed capability is missing, request a new Kimbap integration or skill instead of using direct credentials.
8. Treat Kimbap as the only approved pathway for third-party API access in this repository.
```

---

## 35. Appendix C — Final Product Statement

Kimbap is the **secure action runtime for AI agents**.

It is built from the strongest operational and security primitives of `peta-core-go`, informed by the best zero-code-change lessons from OneCLI, and focused on a simple outcome:

> Agents should be able to do real work across real services without ever being handed the real secrets.

---

## 36. Appendix D — Reference Baseline Used for This PRD

This PRD synthesizes three sources of truth:

1. **The attached Kimbap PRD v0.5 draft**  
   Used as the immediate product baseline and source for terminology, roadmap direction, and core framing.

2. **`peta-core-go` (private) / public `peta-core` analogue**  
   Used as the feature and architectural baseline for:
   - vault and encrypted credential handling
   - downstream OAuth brokerage
   - approval and audit concepts
   - runtime supervision
   - REST adapter / skill concepts
   - the parts to keep vs the MCP-specific wrapper to remove

3. **`onecli`**  
   Used as the design reference for:
   - proxy-first adoption
   - per-agent token model
   - host/path request matching
   - external vault integration mindset
   - repo-local agent skill / instruction-pack patterns
   - practical gateway UX lessons

The resulting product definition is intentionally not a copy of either source project.  
It is a new product specification centered on **Action Runtime first, adapters second**.


# Kimbap Engineering Implementation Specification

| Field | Value |
|---|---|
| Product | Kimbap |
| Document type | Engineering-ready implementation specification |
| Version | 0.1 |
| Based on | Kimbap PRD v0.6 |
| Date | March 23, 2026 |
| Owner | Dunia Labs Engineering + Product |
| Primary baseline | Private `peta-core-go` and `peta-console` |
| Public design analogues | `dunialabs/peta-core`, `onecli/onecli` |
| Delivery target | v1 GA in 20 weeks |
| Packaging target | Single Go binary for core runtime; optional embedded console assets |

---

## 1. Purpose of This Document

This document translates the Kimbap PRD into an implementation plan that engineering can execute directly. It is intended to be used for:

- repository and module setup
- milestone planning
- epic creation
- GitHub issue creation
- staffing and dependency management
- release gating
- test planning
- launch readiness

This is **not** a narrative product document. It is a build document.

---

## 2. Release Objective

### 2.1 v1 Goal

Ship a self-hostable **secure action runtime for AI agents** that supports:

- embedded mode and connected mode
- canonical `service.action` execution via `kimbap call`
- existing HTTP(S)-based agents via `kimbap run` and `kimbap proxy`
- encrypted credential storage and server-side injection
- policy evaluation with allow / deny / require-approval outcomes
- audit logging
- skill-based Tier 1 integrations
- at least two Tier 2 OAuth connector implementations
- human approval workflow with webhook plus one chat adapter
- minimal console for approvals, audit, policies, and integrations

### 2.2 v1 In Scope

1. Canonical Action Runtime
2. CLI-first UX
3. Embedded local install with no mandatory external DB
4. Connected multi-tenant mode on Postgres
5. HTTP/HTTPS proxy for supported traffic
6. OAuth token brokerage for supported connectors
7. Skill install / validate / execute
8. Private registry, lockfile, digest pinning
9. Agent operating profiles / skills
10. Console for governance workflows

### 2.3 v1 Out of Scope

1. Inbound MCP as the product center of gravity
2. Full parity with all possible SDK protocols
3. Generic arbitrary binary plugin execution from untrusted publishers
4. SSO / SAML
5. Workload identity federation
6. Community registry
7. Advanced non-HTTP protocol interception
8. Full enterprise analytics / billing
9. Mobile approval apps

### 2.4 Hard Release Principles

- Kimbap remains the trust boundary, not the downstream CLI.
- The runtime contract is canonical; CLI, proxy, REST, and future MCP adapters are all front doors into the same internal path.
- No raw credential may be intentionally surfaced to the agent process in supported happy paths.
- Embedded and connected mode behavior must be functionally equivalent for action resolution, policy, and audit.

---

## 3. Staffing Assumptions

This plan assumes a minimum staffing model of:

- 1 Staff/Lead engineer across runtime/security
- 2 backend engineers across runtime/API/storage
- 1 proxy/network engineer
- 1 frontend/full-stack engineer for console
- 1 PM/EM shared across product + delivery
- 0.5 design support for console and CLI UX

If the team is smaller, preserve milestone order and extend the calendar rather than reducing the security model.

---

## 4. Architecture Decisions (Implementation ADR Summary)

### ADR-001 — Runtime-first architecture
**Decision:** `internal/runtime` is the canonical core.  
**Why:** Every mode must converge to the same execution path for policy, credential brokerage, approval, and audit.  
**Implication:** Never implement business logic directly inside CLI handlers, HTTP handlers, or proxy code.

### ADR-002 — Single binary, multi-module internals
**Decision:** ship one Go binary with internal module boundaries.  
**Why:** matches product promise and simplifies embedded deployment.  
**Implication:** optional console assets are embedded into the binary at build time; no separate app server is required for v1.

### ADR-003 — SQLite for embedded, Postgres for connected
**Decision:** use SQLite in embedded mode and Postgres in connected mode.  
**Why:** local usability without infra, durable multi-tenant operation in team mode.  
**Implication:** repository layer must abstract dialect differences for only the supported subset; avoid ORM lock-in.

### ADR-004 — `sqlc` + `pgx`/`database/sql`, not a heavy ORM
**Decision:** use explicit SQL and generated query bindings.  
**Why:** clearer control over crypto-adjacent data, migrations, and performance.  
**Implication:** schema changes require deliberate migrations and query updates.

### ADR-005 — Cobra CLI + structured JSON output support
**Decision:** use Cobra for command hierarchy, shell completions, and help; every machine-oriented command also supports JSON output.  
**Why:** speed of implementation and predictable UX.  
**Implication:** command behavior must be backed by reusable services, not Cobra logic.

### ADR-006 — Native policy DSL in v1
**Decision:** implement a Kimbap-native DSL with internal AST; do not adopt Rego for v1.  
**Why:** tighter fit to the action model and lower implementation risk.  
**Implication:** parser, evaluator, explain output, and persistence schema are first-class tasks.

### ADR-007 — Service token and session token are distinct
**Decision:** long-lived service tokens authenticate workloads; short-lived session tokens authorize runtime requests.  
**Why:** enables revocation, display-once issuance, and safer runtime handling.  
**Implication:** auth middleware, proxy, and CLI all need token exchange support.

### ADR-008 — Existing CLIs are adapters, never trust anchors
**Decision:** a service CLI may be invoked as an adapter only after runtime policy and credential resolution.  
**Why:** preserves Kimbap as the actual security boundary.  
**Implication:** existing CLI support is opt-in and constrained by adapter rules.

### ADR-009 — Proxy classification is mandatory, not “best effort”
**Decision:** `kimbap proxy` must classify supported requests into canonical actions or fail according to mode policy.  
**Why:** audit and policy require a canonical action, not just a hostname.  
**Implication:** there is a dedicated classifier layer with persistent mappings and explain tooling.

### ADR-010 — Registry installs are supply-chain events
**Decision:** every installed skill must be resolved through a lockfile entry with digest pinning; signatures are verified when present.  
**Why:** skills are executable configuration and must be treated as code.  
**Implication:** install, upgrade, diff, and trust prompts are part of the v1 implementation.

### ADR-011 — Console is thin and secondary
**Decision:** the console is an operational surface, not the primary product interface.  
**Why:** Kimbap is CLI-first and automation-first.  
**Implication:** anything mission-critical must also be scriptable through CLI and REST.

### ADR-012 — Reuse where possible, rewrite where coupling is structural
**Decision:** reuse `peta-core-go` internals where they already fit the runtime abstraction; rewrite MCP-coupled surfaces instead of forcing compatibility.  
**Why:** faster delivery without inheriting the wrong product center of gravity.  
**Implication:** extraction work must explicitly identify “lift”, “adapt”, and “replace” categories.

---

## 5. Target Technical Stack

| Layer | Choice | Notes |
|---|---|---|
| Core language | Go | same language family as `peta-core-go` |
| CLI | Cobra | command tree, completions, help |
| Config | YAML + env + flags | deterministic precedence |
| Embedded DB | SQLite | local mode |
| Connected DB | PostgreSQL | team mode |
| Query layer | `sqlc` + `pgx` / `database/sql` | avoid heavy ORM |
| Migrations | `goose` or equivalent explicit migration tool | pick one and standardize in sprint 1 |
| HTTP server | stdlib `net/http` + lightweight router | keep dependency surface small |
| OAuth | `golang.org/x/oauth2` + provider adapters | extended where device flow is needed |
| Crypto | Go stdlib + `x/crypto` | PBKDF2/HKDF/AES-GCM as needed |
| Logging | structured JSON logger | request-scoped fields mandatory |
| Metrics | Prometheus-compatible endpoint | p95/p99 and queue metrics |
| Console frontend | React + Vite (embedded static assets) | operational UI only |
| Packaging | goreleaser + Docker | macOS/Linux binaries in v1 |

---

## 6. Repository and Module Layout

```text
kimbap/
  cmd/
    kimbap/
  internal/
    app/
    runtime/
    actions/
    adapters/
    classifier/
    skills/
    generator/
    vault/
    crypto/
    auth/
    sessions/
    policy/
    approvals/
    audit/
    store/
    api/
    proxy/
    runner/
    connectors/
    registry/
    profiles/
    doctor/
    config/
    observability/
    jobs/
  migrations/
    sqlite/
    postgres/
  web/
    console/
  test/
    e2e/
    fixtures/
    integration/
    load/
  docs/
    adr/
    api/
    skills/
  packaging/
    docker/
    goreleaser/
```

### 6.1 Ownership Model

| Area | Default owner |
|---|---|
| runtime, actions, adapters | Runtime team |
| vault, crypto, auth, sessions | Security/platform |
| proxy, runner, classifier | Proxy/network |
| api, store, jobs | Backend/platform |
| web/console | Frontend/full-stack |
| docs/adr, release notes | Shared |

### 6.2 “Lift / Adapt / Replace” Guidance for `peta-core-go`

**Lift directly where low coupling exists**
- vault primitives
- downstream OAuth token brokerage primitives
- audit schema concepts
- approval state machine concepts
- REST action templating helpers

**Adapt where semantics are useful but MCP assumptions leak**
- policy evaluation model
- runtime supervision
- server lifecycle management
- connector configs

**Replace where coupling is structural**
- inbound MCP gateway/session layer
- request-id mapping for MCP reverse routing
- Webhook-first notification model with console polling fallback
- MCP capability filtering surfaces
- event resumption protocol specifics

---

## 7. Core Runtime Contracts

### 7.1 Canonical Action Definition

Every executable thing in Kimbap resolves to an `ActionDefinition`.

```yaml
name: github.issues.create
version: 1
display_name: Create GitHub issue
namespace: github
verb: create
resource: issue
risk: write
idempotent: false
approval_hint: optional
auth:
  type: bearer
  source: connector_or_secret
input_schema:
  type: object
  required: [owner, repo, title]
output_schema:
  type: object
adapter:
  type: http
  method: POST
  url_template: https://api.github.com/repos/{owner}/{repo}/issues
  headers:
    Accept: application/vnd.github+json
```

### 7.2 Runtime Execution Pipeline

All modes must converge to:

1. identify principal/workload
2. resolve action or classify request
3. load action definition
4. evaluate policy
5. request approval if required
6. resolve credential or connector token
7. invoke adapter
8. normalize response
9. write audit event
10. return result

### 7.3 Adapter SPI

```go
type Adapter interface {
    Type() string
    Validate(def ActionDefinition) error
    Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error)
}

type ExecutionRequest struct {
    RequestID       string
    TenantID        string
    Principal       Principal
    Action          ActionDefinition
    Input           map[string]any
    Session         SessionContext
    Credentials     ResolvedCredentialSet
    Classification  *ClassificationInfo
    ApprovalContext *ApprovalContext
}

type ExecutionResult struct {
    Status          string
    Output          map[string]any
    HTTPStatus      int
    Retryable       bool
    IdempotencyKey  string
    DurationMS      int64
    RawMetadata     map[string]any
}
```

### 7.4 Error Taxonomy

Use stable machine-readable errors:

- `ERR_UNAUTHENTICATED`
- `ERR_UNAUTHORIZED`
- `ERR_APPROVAL_REQUIRED`
- `ERR_APPROVAL_TIMEOUT`
- `ERR_ACTION_NOT_FOUND`
- `ERR_CLASSIFICATION_FAILED`
- `ERR_SKILL_INVALID`
- `ERR_CONNECTOR_NOT_LOGGED_IN`
- `ERR_TOKEN_EXPIRED`
- `ERR_RATE_LIMITED`
- `ERR_DOWNSTREAM_UNAVAILABLE`
- `ERR_UNSAFE_EXISTING_CLI`
- `ERR_UNSUPPORTED_PROXY_PROTOCOL`

### 7.5 Core Domain Entities

| Entity | Embedded | Connected | Notes |
|---|---|---|---|
| tenant | optional local default | required | default tenant exists in embedded mode |
| principal | local user/agent | user/service/workload | auth identity |
| agent | optional | recommended | named non-human actor |
| service_token | yes | yes | display once, hashed at rest |
| session_token | yes | yes | short-lived |
| secret_record | yes | yes | logical secret |
| secret_version | yes | yes | encrypted versions |
| connector | yes | yes | provider config |
| oauth_account | optional | yes | login state |
| oauth_token | optional | yes | encrypted access/refresh data |
| skill_package | yes | yes | install unit |
| action_definition | yes | yes | canonical action |
| policy_set | yes | yes | compiled + source |
| approval_request | optional | yes | durable workflow object |
| audit_event | yes | yes | append-only |
| classification_rule | optional | yes | proxy mapping rules |
| registry_lock | yes | yes | digest pinning |

### 7.6 Minimum REST API Surface

| Method | Path | Purpose |
|---|---|---|
| POST | `/v1/sessions/exchange` | exchange service token for session token |
| GET | `/v1/actions` | list actions |
| GET | `/v1/actions/{name}` | describe one action |
| POST | `/v1/actions/{name}:invoke` | invoke action |
| POST | `/v1/policies/eval` | dry-run policy evaluation |
| GET/POST | `/v1/policies` | list/create policies |
| GET/POST | `/v1/skills` | list/install skills |
| POST | `/v1/skills:validate` | validate skill bundle |
| POST | `/v1/connectors/{id}/login/start` | begin OAuth flow |
| POST | `/v1/connectors/{id}/login/complete` | finish OAuth flow |
| GET | `/v1/audit` | list audit events |
| GET | `/v1/approvals` | list approval requests |
| POST | `/v1/approvals/{id}:approve` | approve |
| POST | `/v1/approvals/{id}:deny` | deny |
| GET/POST | `/v1/tokens` | issue/list tokens |
| POST | `/v1/tokens/{id}:revoke` | revoke token |

### 7.7 Minimum CLI Surface

```text
kimbap call <service.action> [--input file.json]
kimbap actions list
kimbap actions describe <service.action>
kimbap actions validate
kimbap vault set <name>
kimbap vault get-meta <name>
kimbap vault rotate <name>
kimbap auth login <connector>
kimbap auth status
kimbap token create
kimbap token revoke <id>
kimbap policy eval --action <service.action> --input file.json
kimbap run -- <command ...>
kimbap proxy
kimbap serve
kimbap skill install <ref>
kimbap skill list
kimbap skill remove <name>
kimbap profile install <target>
kimbap doctor
```

---

## 8. Security and Data Handling Requirements

### 8.1 Credential Handling

- Secrets are encrypted at rest.
- Raw secrets are accepted only through stdin, file descriptors, device/browser flow completion, or interactive secure prompts.
- Raw secrets are never accepted through normal CLI args.
- Raw secrets are never printed back to stdout.
- Logs must redact all configured secret-like fields.
- Existing CLIs may receive ephemeral credentials only through controlled adapter mechanisms and only when the adapter contract permits it.

### 8.2 Token Handling

- Service tokens are displayed once, hashed at rest, revocable, and last-used tracked.
- Session tokens are short-lived and audience-bound.
- Refresh tokens are encrypted, never exposed to agent processes, and only used inside Kimbap.
- Token exchange endpoints must enforce tenant and scope binding.

### 8.3 Tenant Isolation

- Tenant-scoped DEKs encrypt tenant secrets.
- Tenant KEKs are versioned and rotatable.
- Cross-tenant reads are impossible at repository API boundaries.
- Audit tables always carry `tenant_id`.

### 8.4 Proxy Trust Claims

Kimbap may honestly claim:

- raw real credentials are not intentionally exposed to supported agent flows
- outbound HTTP(S) credentials are injected server-side
- unsupported or unclassified proxy traffic can be blocked

Kimbap may not honestly claim:

- that every possible existing CLI or SDK can be mediated safely
- that non-HTTP traffic is comprehensively protected in v1
- that a compromised host OS cannot exfiltrate decrypted memory

---

## 9. Non-Functional Requirements

| Area | Requirement |
|---|---|
| Policy evaluation | p99 < 50 ms |
| Warm runtime overhead before downstream call | p50 < 25 ms |
| Approval resume | < 2 s after decision receipt, excluding downstream latency |
| Startup | deterministic; safe defaults on first boot |
| Platforms | macOS and Linux in v1 |
| Embedded mode | no mandatory external DB |
| Connected mode | durable storage and restart-safe workflows |
| Audit | append-only, request IDs mandatory |
| Reliability | idempotent retries where safe; clear failure modes where not |
| Observability | metrics, structured logs, health endpoints, trace correlation |
| Migrations | zero data loss on supported version upgrades |

---

## 10. Delivery Milestones

| Milestone | Weeks | Outcome | Exit criteria |
|---|---:|---|---|
| M0 Foundation | 1-2 | repo, build, storage, architecture skeleton | repo builds on macOS/Linux; migrations run; config precedence defined |
| M1 Local Runtime Alpha | 3-5 | embedded mode, `call`, vault, skills, policy, audit | Tier 1 action executes locally; audit + policy working |
| M2 Team/Proxy Beta | 6-10 | proxy, `run`, connected mode, sessions, classifier | supported HTTP agent can run through proxy with audit + policy |
| M3 Governance RC | 11-15 | approvals, connectors, console, registry trust | one approval flow and two OAuth connectors end-to-end |
| M4 GA Hardening | 16-20 | performance, packaging, docs, migration safety | release candidates pass security, performance, and upgrade gates |

### 10.1 Critical Path

The hard dependency chain is:

`Foundation -> Runtime -> Vault/Auth -> Skills/Policy/Audit -> Proxy/Connected -> Connectors -> Approvals -> Console -> GA hardening`

### 10.2 Parallel Workstreams

| Workstream | Primary epics |
|---|---|
| Runtime/Security | E1, E3, E4, E6, E7 |
| CLI/Integrations | E2, E5, E13 |
| Proxy/Connected | E8, E9, E10, E11 |
| Console/Operations | E12, E14 |

---

## 11. Epic Plan

### EPIC E0 — Foundations and Extraction

**Objective:** create the repo, build system, migration layer, and extraction plan from `peta-core-go`.

**Must ship**
- repository skeleton
- CI and release scaffolding
- migration tool choice and bootstrap
- explicit extraction matrix: lift/adapt/replace

**Exit criteria**
- one command bootstraps local development
- both SQLite and Postgres migration paths compile and run
- extraction plan is approved by engineering lead

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-001 | M0 | Bootstrap repo skeleton, module boundaries, Makefile/task runner | Platform | M | — | repo layout exists, local `build/test/lint` entrypoints work |
| KIM-002 | M0 | Set up CI for lint, unit tests, race tests, cross-platform build | Platform | M | KIM-001 | CI passes on Linux and macOS targets |
| KIM-003 | M0 | Choose migration tooling and add SQLite/Postgres bootstrap migrations | Backend | M | KIM-001 | fresh databases can be created and migrated in both modes |
| KIM-004 | M0 | Produce `peta-core-go` extraction matrix (lift/adapt/replace) and ADR pack | Lead/Runtime | S | KIM-001 | reviewed design doc committed under `docs/adr/` |

---

### EPIC E1 — Canonical Runtime Core

**Objective:** implement the execution kernel that all modes call.

**Must ship**
- canonical action schema
- execution pipeline
- adapter SPI
- request/result/error envelope
- timeout/cancellation behavior

**Exit criteria**
- `kimbap call` and REST invoke path use the same runtime function
- unit and integration tests cover at least one HTTP action end-to-end

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-010 | M1 | Define `ActionDefinition`, `ExecutionRequest`, `ExecutionResult`, and stable error codes | Runtime | M | KIM-004 | contracts are versioned, documented, and used by all runtime callers |
| KIM-011 | M1 | Implement runtime execution pipeline and middleware chain | Runtime | L | KIM-010 | identify -> resolve -> policy -> credential -> execute -> audit pipeline works |
| KIM-012 | M1 | Implement adapter SPI with HTTP adapter v1 | Runtime | M | KIM-010 | HTTP adapter validates definitions and executes templated calls |
| KIM-013 | M1 | Add timeout, cancellation, and retry semantics with idempotency guardrails | Runtime | M | KIM-011 | retries occur only when action/adapter marks request as safe |
| KIM-014 | M1 | Add runtime supervisor hooks and execution context propagation | Runtime | M | KIM-011 | every execution has request IDs, tenant IDs, and cancellation propagation |

---

### EPIC E2 — CLI Surface and UX

**Objective:** expose runtime capabilities through a coherent CLI.

**Must ship**
- stable command tree
- machine-readable output
- secure input handling
- diagnostics

**Exit criteria**
- a new user can install the binary, validate config, store a secret, install a skill, and invoke an action without the console

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-020 | M1 | Implement root command, global config flags, profile loading, and shell completions | CLI | M | KIM-001 | command tree is stable and completions generate correctly |
| KIM-021 | M1 | Implement `actions list/describe/validate` and `call` commands | CLI | M | KIM-010, KIM-011 | commands call runtime services and support `--output json` |
| KIM-022 | M1 | Implement `vault`, `token`, and `auth status` command families | CLI | M | KIM-030, KIM-040 | secure input paths are enforced and documented |
| KIM-023 | M1 | Implement `policy eval` and `doctor` commands | CLI | M | KIM-060, KIM-070 | policy explain and health diagnostics are visible from CLI |
| KIM-024 | M1 | Add human-readable formatting, exit code contract, and stderr/stdout policy | CLI | S | KIM-021 | command behavior is script-safe and documented |

---

### EPIC E3 — Vault, Crypto, and Secret Lifecycle

**Objective:** provide secure storage and secret lifecycle management for embedded and connected modes.

**Must ship**
- encryption envelope
- secret repositories
- secret versioning
- rotation
- redaction and safe input/output rules

**Exit criteria**
- secrets are encrypted at rest in both modes
- rotation creates new versions without data loss
- secret metadata is auditable without exposing values

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-030 | M1 | Implement crypto envelope service (salt/nonce generation, KDF, AES-GCM helpers) | Security | L | KIM-003 | encryption/decryption API is tested with corrupt-data rejection cases |
| KIM-031 | M1 | Implement embedded secret store on SQLite with local default tenant | Security | M | KIM-030 | local secrets can be create/read-metadata/rotate/delete |
| KIM-032 | M2 | Implement connected secret store on Postgres with tenant-scoped keys | Security | L | KIM-030, KIM-091 | secrets are isolated by tenant and migration-tested |
| KIM-033 | M1 | Add secret versioning, labels, metadata, and rotation workflow | Security | M | KIM-031 | latest and historical versions are tracked safely |
| KIM-034 | M1 | Enforce secure input/output constraints for secrets and redaction middleware | Security | M | KIM-030 | raw secrets cannot be passed by normal args or logged |

---

### EPIC E4 — Identity, Authentication, and Sessions

**Objective:** model principals and implement service-token and session-token flows.

**Must ship**
- principal model
- service token issuance/revocation
- session exchange
- auth middleware
- audit of token use

**Exit criteria**
- a workload authenticates with a service token and receives a short-lived session token
- revocation takes effect predictably

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-040 | M1 | Implement principal/agent/service token data model and issuance flow | Auth | L | KIM-003 | tokens are display-once, hashed at rest, and scoped |
| KIM-041 | M2 | Implement session token exchange service and auth middleware | Auth | L | KIM-040 | REST and proxy paths can validate exchanged session tokens |
| KIM-042 | M2 | Implement token revoke/list/last-used tracking and introspection | Auth | M | KIM-040 | revoked tokens fail immediately and audit records include token IDs |
| KIM-043 | M2 | Implement tenant- and audience-bound auth claims enforcement | Auth | M | KIM-041 | cross-tenant and wrong-audience use is rejected with stable errors |

---

### EPIC E5 — Skills, Action Packaging, and Generators

**Objective:** make Tier 1 integrations declarative and installable.

**Must ship**
- skill manifest schema
- validation
- install/remove/list
- local resolver
- OpenAPI/Postman generation path
- canonical action discovery

**Exit criteria**
- a supported REST API can become installable actions without handwritten Go code
- install is locked and reproducible

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-050 | M1 | Define skill manifest schema, validation rules, and canonical naming checks | Skills | M | KIM-010 | invalid skills fail with actionable diagnostics |
| KIM-051 | M1 | Implement local skill installer, resolver, and uninstall flow | Skills | M | KIM-050 | installed skills are discoverable and removable cleanly |
| KIM-052 | M1 | Implement Tier 1 HTTP executor templates and parameter mapping | Skills | L | KIM-012, KIM-050 | templated actions execute through runtime successfully |
| KIM-053 | M2 | Build OpenAPI/Postman generator to draft skill bundles | Generator | L | KIM-050 | generator outputs compilable draft skills with review warnings |
| KIM-054 | M1 | Implement action catalog, `describe`, examples, and collision resolution | Skills | M | KIM-051 | action discovery is stable across multiple installed skills |

---

### EPIC E6 — Policy Engine

**Objective:** enforce action-level authorization with explainable outcomes.

**Must ship**
- DSL parser
- evaluator
- built-in predicates
- explain / dry-run
- persisted policy sets

**Exit criteria**
- policy evaluation is deterministic
- evaluation explains which rule decided the outcome
- approval-required is treated as a first-class result

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-060 | M1 | Design DSL grammar, parser, AST, and validation errors | Policy | L | KIM-004 | policies parse into stable AST with source locations |
| KIM-061 | M1 | Implement evaluator with allow/deny/require-approval semantics | Policy | L | KIM-060, KIM-010 | runtime can ask policy engine for decisions synchronously |
| KIM-062 | M1 | Implement built-in predicates (tenant, principal, action, risk, idempotency, input fields) | Policy | M | KIM-061 | predicates cover all v1 policy objects |
| KIM-063 | M2 | Implement `policy eval --explain`, persisted policy packs, and test fixtures | Policy | M | KIM-061 | dry-run output is stable and policies are persistence-backed in connected mode |

---

### EPIC E7 — Audit, Logging, and Observability

**Objective:** make every action inspectable and operable.

**Must ship**
- audit schema
- append-only sinks
- structured logs
- metrics/health
- correlation IDs

**Exit criteria**
- every runtime request emits an audit event
- proxy, CLI, and REST all share one request ID model
- operators can diagnose failures without seeing secrets

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-070 | M1 | Define audit event schema and append-only writers (JSONL + DB sink) | Observability | M | KIM-010 | audit writer supports embedded and connected backends |
| KIM-071 | M1 | Add structured logs, request-scoped fields, and secret-redaction filters | Observability | M | KIM-030, KIM-070 | logs are machine-readable and redact protected material |
| KIM-072 | M2 | Add health endpoints, readiness checks, metrics, and pprof/debug hooks | Observability | M | KIM-070 | operators can inspect health and latency metrics |
| KIM-073 | M2 | Implement audit export, retention hooks, and replay-safe request IDs | Observability | M | KIM-070 | audit streams can be exported without breaking append-only semantics |

---

### EPIC E8 — Proxy, Runner, and Classification Layer

**Objective:** support zero-code-change adoption for supported HTTP(S) agents while preserving canonical action semantics.

**Must ship**
- HTTP proxy
- HTTPS CONNECT MITM
- CA UX
- classification layer
- `kimbap run`
- strict behavior for unmapped traffic

**Exit criteria**
- an existing HTTP-based agent can be run through Kimbap without source changes
- classified requests show canonical action names in audit
- unsupported traffic fails clearly

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-080 | M2 | Implement HTTP proxy core with outbound middleware hooks | Proxy | L | KIM-011, KIM-041 | proxy can intercept HTTP requests and invoke runtime services |
| KIM-081 | M2 | Implement HTTPS CONNECT MITM path, local CA generation, and trust UX | Proxy | XL | KIM-080 | supported HTTPS traffic can be intercepted after explicit trust setup |
| KIM-082 | M2 | Implement request classification store, matcher, and explain tooling | Classifier | L | KIM-010, KIM-050 | requests map to canonical actions with explain output |
| KIM-083 | M2 | Implement `kimbap run` wrapper with environment/proxy injection and subprocess supervision | Proxy | M | KIM-080 | existing commands can be launched with Kimbap mediation |
| KIM-084 | M2 | Implement unmapped request behavior, deny-by-policy mode, and egress allow-list controls | Proxy | M | KIM-082, KIM-061 | unsafe or unclassified traffic is handled predictably |
| KIM-085 | M2 | Add proxy-specific audit fields, retry policy, and latency/perf test suite | Proxy | M | KIM-080, KIM-070 | proxy traffic has complete audit records and meets performance budgets |

---

### EPIC E9 — Connected Mode API, Multi-Tenancy, and Persistence

**Objective:** turn Kimbap into a team-operable control plane without changing the core runtime contract.

**Must ship**
- REST server
- tenant/workspace model
- repositories and jobs
- connected persistence for all control-plane objects
- parity with embedded runtime semantics

**Exit criteria**
- connected mode can host multiple tenants safely
- the same action invoked in embedded and connected mode yields equivalent policy/audit semantics

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-090 | M2 | Implement `kimbap serve` server skeleton, middleware stack, and config model | API | M | KIM-001, KIM-041 | server boots with auth, logging, health, and config validation |
| KIM-091 | M2 | Implement tenant/workspace/project data model and repository boundaries | API/Store | L | KIM-003 | tenant scoping is enforced at repository interfaces |
| KIM-092 | M2 | Implement connected repositories for secrets, skills, policies, audit, approvals, and tokens | Store | L | KIM-091, KIM-032 | all runtime state can persist durably in Postgres |
| KIM-093 | M2 | Implement v1 REST resources for actions, skills, policies, audit, approvals, and tokens | API | L | KIM-090, KIM-092 | documented v1 endpoints pass contract tests |
| KIM-094 | M3 | Build embedded-to-connected import/export and parity test suite | API/Runtime | M | KIM-093 | migration path exists and parity tests run in CI |

---

### EPIC E10 — Downstream OAuth Connectors

**Objective:** enable Tier 2 services with secure OAuth login and refresh handled inside Kimbap.

**Must ship**
- connector framework
- browser/device login
- encrypted token storage
- refresh flow
- at least two launch providers

**Exit criteria**
- a supported provider can be logged into without exposing refresh tokens to the agent
- expired access tokens are refreshed automatically in the happy path

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-100 | M2 | Implement connector framework, provider contract, and config schema | Connectors | L | KIM-030, KIM-091 | providers plug into common login/token/execution hooks |
| KIM-101 | M2 | Implement browser/device flow manager and callback handling | Connectors | L | KIM-100, KIM-090 | login start/complete endpoints and CLI flows work |
| KIM-102 | M2 | Implement encrypted OAuth token store and automatic refresh worker | Connectors | L | KIM-100, KIM-032 | refresh tokens remain internal and rotation is supported |
| KIM-103 | M3 | Select launch providers and ship provider A connector end-to-end | Connectors | M | KIM-101, KIM-102 | provider A works in embedded and connected mode |
| KIM-104 | M3 | Ship provider B connector end-to-end and add connector integration tests | Connectors | M | KIM-101, KIM-102 | provider B passes login/refresh/invoke test suite |

---

### EPIC E11 — Approval Workflow and Messaging Adapters

**Objective:** gate risky actions with durable human approval.

**Must ship**
- approval state machine
- pending execution storage
- webhook adapter
- one interactive chat adapter
- resume/timeout/deny behavior

**Exit criteria**
- a policy can suspend a write action pending approval
- approve/deny resumes or terminates deterministically
- audit captures requester and approver identities

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-110 | M3 | Implement approval request state machine and persistence model | Approvals | L | KIM-061, KIM-092 | lifecycle states are durable and replay-safe |
| KIM-111 | M3 | Implement held-execution store and resume/deny/timeout mechanics | Approvals | L | KIM-110, KIM-011 | suspended actions can be resumed exactly once |
| KIM-112 | M3 | Implement webhook approval adapter | Approvals | M | KIM-110, KIM-090 | external systems can receive and answer approval requests |
| KIM-113 | M3 | Implement one chat adapter (Slack recommended) and audit integration | Approvals | M | KIM-112 | chat-based approve/deny path works end-to-end |

---

### EPIC E12 — Console

**Objective:** provide a minimal but useful operational UI.

**Must ship**
- login / auth
- action/skill/policy views
- approval inbox
- audit explorer
- integration status
- embedded asset delivery

**Exit criteria**
- operators can manage approvals, inspect audit, and review integrations without using the CLI
- all core console screens are backed by stable REST APIs

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-120 | M3 | Bootstrap console shell, routing, auth, and embedded asset build pipeline | Frontend | M | KIM-093 | console assets build and serve from the Go binary |
| KIM-121 | M3 | Implement policies, skills, and actions screens | Frontend | M | KIM-120, KIM-093 | operators can browse, inspect, and trigger core workflows |
| KIM-122 | M3 | Implement approval inbox/detail flow and live refresh polling | Frontend | M | KIM-120, KIM-110 | approvals can be reviewed and acted on in UI |
| KIM-123 | M3 | Implement audit explorer and connector status views | Frontend | M | KIM-120, KIM-102 | audit and integration health are inspectable in UI |

---

### EPIC E13 — Registry Trust Model and Agent Operating Profiles

**Objective:** make skills distributable and make agents default to Kimbap.

**Must ship**
- private registry
- lockfile and digest pinning
- install diff/trust prompts
- agent operating skill/profile installers

**Exit criteria**
- skills can be installed reproducibly from trusted sources
- an agent repo can be updated with Kimbap operating instructions automatically

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-130 | M3 | Implement private registry API/storage and local cache | Registry | M | KIM-051, KIM-093 | skills can be fetched from a registry and cached locally |
| KIM-131 | M3 | Implement lockfile format, digest pinning, signature hooks, and upgrade diff | Registry | L | KIM-130 | every install/upgrade writes reproducible lock entries |
| KIM-132 | M3 | Implement agent operating profile generator and installers for `.agents/` and `.claude/` | Profiles | M | KIM-051 | repo-local Kimbap usage instructions can be installed automatically |
| KIM-133 | M3 | Add trust prompts, policy hooks, and unsafe install blocking rules | Registry | M | KIM-131, KIM-061 | untrusted or changed packages surface explicit approval flows |

---

### EPIC E14 — GA Hardening, Packaging, and Launch Readiness

**Objective:** make the product releasable, upgrade-safe, and supportable.

**Must ship**
- performance test suite
- security review
- packaging and upgrade tooling
- docs and examples
- release checklists

**Exit criteria**
- RC build passes performance, security, migration, and end-to-end test gates
- release artifacts are signed and reproducible

### Issues

| ID | Milestone | Title | Area | Size | Depends on | Done when |
|---|---|---|---|---|---|---|
| KIM-140 | M4 | Build benchmark/load suite for runtime, proxy, policy, and approvals | QA/Perf | M | KIM-085, KIM-093 | performance budgets are enforced in CI or release pipeline |
| KIM-141 | M4 | Run threat-model review, dependency audit, and security hardening pass | Security | L | all M3 epics | critical findings are fixed or explicitly waived |
| KIM-142 | M4 | Implement goreleaser, Docker packaging, binary signing, and release pipeline | Platform | M | KIM-002 | signed artifacts are published for supported platforms |
| KIM-143 | M4 | Add upgrade/migration compatibility tests and rollback playbook | Platform | M | KIM-092 | supported version upgrades preserve data safely |
| KIM-144 | M4 | Write operator docs, quickstarts, examples, and launch checklist | Docs/Eng | M | KIM-120, KIM-142 | docs cover embedded, connected, proxy, approval, and connector flows |

---

## 12. Sprint-Level Cut (Suggested)

| Sprint | Focus | Primary issues |
|---|---|---|
| Sprint 1 | foundations | KIM-001 to KIM-004 |
| Sprint 2 | runtime contracts + crypto | KIM-010, KIM-011, KIM-030, KIM-031 |
| Sprint 3 | CLI + policy + audit | KIM-020, KIM-021, KIM-060, KIM-061, KIM-070 |
| Sprint 4 | skills + local alpha | KIM-050, KIM-051, KIM-052, KIM-033, KIM-034 |
| Sprint 5 | auth + connected skeleton | KIM-040, KIM-041, KIM-090, KIM-091 |
| Sprint 6 | proxy + classifier | KIM-080, KIM-081, KIM-082 |
| Sprint 7 | connected APIs + runner | KIM-083, KIM-092, KIM-093, KIM-071, KIM-072 |
| Sprint 8 | connectors + approvals | KIM-100, KIM-101, KIM-102, KIM-110, KIM-111 |
| Sprint 9 | console + registry | KIM-120, KIM-121, KIM-130, KIM-131, KIM-132 |
| Sprint 10 | hardening + launch | KIM-140 to KIM-144, remaining M3 spillover |

---

## 13. Definition of Ready / Definition of Done

### 13.1 Definition of Ready

A ticket is ready when:

- problem statement is clear
- owning area is assigned
- dependencies are identified
- acceptance criteria are written
- API/schema impacts are called out
- test plan is specified

### 13.2 Definition of Done

A ticket is done when:

- code is merged behind the correct feature flag if needed
- unit/integration tests pass
- docs and CLI help text are updated
- audit/logging behavior is implemented where relevant
- metrics/alerts are added where relevant
- security implications are reviewed if the ticket touches auth/crypto/proxy
- migration notes are included if schema changed

---

## 14. Test Strategy

### 14.1 Unit Tests
Required for:
- parser/evaluator
- runtime request normalization
- crypto primitives
- token validation
- classifier matching
- registry lockfile logic

### 14.2 Integration Tests
Required for:
- SQLite and Postgres repositories
- HTTP adapter execution
- service token -> session token exchange
- OAuth login/refresh flows with mock providers
- approval resume path
- proxy classification path

### 14.3 End-to-End Tests
Required scenarios:
1. embedded install -> vault secret -> install skill -> invoke action
2. connected install -> create tenant -> issue token -> invoke action over REST
3. proxy existing HTTP agent -> classified action -> audit record emitted
4. policy requires approval -> webhook/chat approval -> action resumes
5. connector login -> refresh token stored -> access token refresh on expiry
6. registry install -> lockfile created -> upgrade diff shown

### 14.4 Security Tests
Required for:
- secret redaction
- token replay/revocation
- cross-tenant isolation
- malformed ciphertext rejection
- proxy trust downgrade attempts
- unclassified traffic deny behavior

### 14.5 Performance Tests
Required for:
- policy p99
- proxy warm-path overhead
- audit append throughput
- approval resume latency
- session exchange throughput

---

## 15. Feature Flags

Use feature flags aggressively until RC:

| Flag | Default | Purpose |
|---|---|---|
| `feature.proxy` | off until M2 beta | proxy and runner surfaces |
| `feature.connected_mode` | off until M2 beta | team/server mode |
| `feature.approvals` | off until M3 | HITL flows |
| `feature.registry` | off until M3 | registry install and lockfile behaviors |
| `feature.console` | off until M3 | UI surface |
| `feature.oauth_connectors` | off until M3 | external provider login flows |
| `feature.existing_cli_adapters` | off by default | optional existing CLI executor paths |

---

## 16. Launch Gates

Kimbap v1 cannot launch unless all gates below are green.

### Gate A — Security
- threat model reviewed
- no known critical or high-severity auth/crypto issues
- redaction tests pass
- cross-tenant isolation tests pass

### Gate B — Functionality
- Tier 1 skills execute in embedded and connected mode
- proxy supports at least documented HTTP(S) happy paths
- approval flow works through at least two delivery channels
- two OAuth connectors pass login/refresh tests

### Gate C — Operability
- health endpoints, metrics, and logs exist
- docs cover installation, upgrade, and failure recovery
- migrations are tested on realistic fixture data

### Gate D — UX
- CLI help is coherent
- `doctor` catches the most common misconfigurations
- console is sufficient for approvals, audit, policy review, and connector status

---

## 17. GitHub Project Setup Recommendation

### 17.1 Labels

- `epic/*`
- `area/runtime`
- `area/security`
- `area/proxy`
- `area/api`
- `area/store`
- `area/frontend`
- `area/skills`
- `area/observability`
- `type/feature`
- `type/chore`
- `type/bug`
- `type/spike`
- `mode/embedded`
- `mode/connected`
- `priority/p0`
- `priority/p1`
- `priority/p2`

### 17.2 Milestones

- `M0-foundation`
- `M1-local-alpha`
- `M2-team-proxy-beta`
- `M3-governance-rc`
- `M4-ga`

### 17.3 Project Fields

- Issue ID
- Epic
- Milestone
- Area
- Owner
- Size
- Status
- Dependency
- Risk level
- Feature flag
- ADR touched?
- Docs required?

---

## 18. Immediate Next Steps

Within the first 5 working days after approval:

1. approve ADR-001 through ADR-012
2. create the repo skeleton and CI
3. commit the extraction matrix from `peta-core-go`
4. choose migration tool
5. assign owners for E0, E1, E3, E4
6. create GitHub epics and import all KIM issue shells
7. start Sprint 1 with M0-only scope; do not begin console work early

---

## 19. Appendix A — Recommended Launch Connector Selection

Because v1 only needs two launch connectors, choose providers that exercise the hardest required paths:

- one provider with browser/device login + refresh token lifecycle
- one provider with scope management and high-value business use

Selection criteria:
- refresh token support
- stable API surface
- clear scope semantics
- representative of real customer demand
- minimal provider-specific operational risk

Do not choose both launch connectors from trivial PAT-only providers; that would under-test the connector framework.

---

## 20. Appendix B — Existing CLI Policy

The implementation team should treat existing service CLIs as follows:

**Allowed**
- adapter-style invocation under Kimbap supervision
- temporary materialization of short-lived access tokens when unavoidable
- sandboxed subprocess execution
- explicit audit tagging of the adapter path

**Disallowed by default**
- using the service CLI’s own stored home-directory login state as the source of truth
- giving the agent direct access to service-native credential caches
- bypassing Kimbap policy and audit because “the CLI already authenticated”
- inheriting arbitrary environment variables into an existing CLI subprocess

If a downstream CLI cannot fit this model, it is not a v1 adapter candidate.

---

## 21. Appendix C — Sample GitHub Epic Descriptions

### Runtime Core
Build the canonical Action Runtime used by every Kimbap surface. This epic owns action schema, execution pipeline, adapter contracts, timeouts, and result/error normalization.

### Proxy and Runner
Build transparent mediation for supported HTTP(S) workloads while preserving action-level policy and audit semantics. This epic owns proxy interception, CONNECT MITM, subprocess wrapper, and request classification.

### Registry and Profiles
Build trusted skill distribution and agent onboarding. This epic owns registry storage, lockfiles, digest pinning, install/upgrade UX, and repo-local agent operating profile generation.

---

## 22. Final Delivery Statement

If engineering implements this document in milestone order, Kimbap v1 will launch as a coherent product rather than a collection of partially connected security features.

The non-negotiables are:

- runtime-first architecture
- strict token separation
- credential non-exposure
- canonical action model
- classifier-backed proxying
- durable approvals
- reproducible skill installs

Anything that weakens those points should be deferred rather than merged half-finished.

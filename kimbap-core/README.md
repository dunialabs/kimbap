# Kimbap

> **Secure action runtime for AI agents**
>
> Let agents use GitHub, Gmail, Stripe, Notion, internal APIs, and existing CLIs without handing them raw credentials.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-ELv2-blue.svg)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-blue.svg)

Kimbap is the layer between **AI agents** and **external systems**.

Instead of giving an agent API keys, OAuth tokens, or a human CLI session, you give it access to **Kimbap**. Kimbap handles identity, policy, credential injection, OAuth refresh, approvals, and audit. The agent only sees **actions** and **results**.

**Quick Start** → [https://kimbap.sh/quick-start](https://kimbap.sh/quick-start)

**Website** → [https://kimbap.sh](https://kimbap.sh)

**Documentation** → [https://docs.kimbap.sh](https://docs.kimbap.sh)

---

## Why Kimbap

AI agents are starting to do real work:

- read and write GitHub issues
- send email
- update CRMs
- create docs
- refund payments
- run infra workflows
- call internal APIs

Most teams solve this in one of four bad ways:

1. **Raw API keys in env vars**
   Fast, but unsafe. Secrets leak into prompts, logs, traces, or shell history.

2. **Human-oriented service CLIs**
   Useful for developers, but not safe trust boundaries for autonomous agents.

3. **Vault only**
   A vault stores secrets, but does not execute actions, refresh OAuth tokens, enforce policy, or manage approvals.

4. **Custom wrapper per service**
   Secure-ish, but expensive. Every integration becomes another codebase to build and maintain.

Kimbap replaces that mess with one runtime.

---

## What it looks like

```text
agent -> kimbap -> policy -> approval -> credentials -> execution -> audit
```

The agent does not need to know:

- how GitHub auth works
- how Gmail refreshes tokens
- whether Stripe refunds need approval
- where secrets live
- whether an action is backed by REST, OAuth, or an existing CLI

It only needs to know:

```bash
kimbap call <service>.<action>
```

Examples:

```bash
kimbap call github.list_pull_requests --repo owner/repo
kimbap call notion.create_page --database-id db_123 --title "Q2 Plan"
kimbap call stripe.refund_charge --charge ch_abc --amount 500 --idempotency-key refund-001
```

---

## Core idea

**Action Runtime is the product.**

Kimbap is not just:

- a vault
- a proxy
- a CLI wrapper
- an MCP server
- an integration zoo

Those are surfaces and adapters.

The actual product is a **canonical action runtime** that every surface goes through:

```text
identity -> action lookup -> policy -> approval -> credential injection -> execution -> audit
```

That gives you:

- one security model
- one policy layer
- one audit trail
- one integration model
- one path from local dev to multi-tenant deployment

---

## Product surfaces

Kimbap ships as a **single Go binary**.

### Explicit action mode

For new agent systems and direct integrations:

```bash
kimbap call github.list_pull_requests --repo owner/repo
```

### Proxy mode

For existing HTTP-based agents with minimal or zero code changes:

```bash
kimbap proxy --port 10255
```

### Subprocess mode

For local agents, scripts, or agent frameworks that already run as a process:

```bash
kimbap run -- python agent.py
```

### Connected server mode

For shared deployments, long-running connectors, centralized audit, and multi-tenant use:

```bash
kimbap serve --port 8080
```

### Management commands

```bash
kimbap token ...
kimbap vault ...
kimbap policy ...
kimbap connector ...
kimbap audit ...
kimbap skill ...
```

### Agent onboarding

Kimbap can sync skill files and operating rules directly into your AI agent's project directory, so the agent knows how to discover and call Kimbap actions without manual setup.

```bash
# Auto-detect installed agents and sync skills + rules
kimbap agents setup

# Re-sync after installing a new skill
kimbap skill install slack.yaml
kimbap agents sync

# Target specific agents
kimbap agents sync --agent claude-code,opencode

# Check current sync status
kimbap agents status

# Preview changes without writing
kimbap agents sync --dry-run
```

Supported agents: Claude Code, OpenCode, Codex, Cursor, and a generic fallback.

Each agent gets:
- **Per-service SKILL.md files** — generated from installed skills, placed in the agent's skill directory (e.g. `.claude/skills/github/SKILL.md`)
- **A meta-skill** — a thin "how to discover Kimbap actions" guide at `.claude/skills/kimbap/SKILL.md`
- **Operating rules** — credential handling and access policies at `.claude/KIMBAP_OPERATING_RULES.md`

Skills are auto-generated from installed skill manifests. Unchanged files are skipped unless `--force` is used.

#### Keeping agents up to date

When you add, update, or remove a skill, connected agents won't know about the change until you re-sync:

```bash
# Install a new skill → agents don't know yet
kimbap skill install notion.yaml

# Sync propagates the new skill to all detected agents
kimbap agents sync

# Remove a skill → its SKILL.md stays until you force-sync or delete manually
kimbap skill remove notion
kimbap agents sync --force
```

The source of truth is always the Kimbap runtime. Agents can also discover actions dynamically at any time via `kimbap actions list`, even without synced skill files. Synced SKILL.md files are a speed-up for agent onboarding, not a requirement — the meta-skill teaches agents to use runtime discovery as the authoritative fallback.

One binary. One install. One runtime model.

---

## How integrations work

Kimbap does **not** want a bespoke codebase for every service.

### Tier 1 — Declarative skills

Most modern APIs can be exposed through a human-readable YAML skill:

- auth type
- endpoint shape
- arguments
- error mapping
- pagination
- output extraction
- risk metadata

That turns a REST API into a first-class Kimbap action.

### Tier 2 — Connectors

Some services need OAuth device flow, token refresh, or provider-specific lifecycle handling.

Examples:

- Gmail
- Google Workspace
- Slack
- HubSpot
- GitHub Apps
- Stripe Connect

These use runtime-owned connectors.

### Tier 2b — Existing CLI adapters

Some mature CLIs are useful, but only if Kimbap wraps them safely. The CLI is **never** the trust boundary. Kimbap is the trust boundary.

### Tier 3 — Justified stateful services

Only truly stateful or streaming systems should require a separate downstream service.

Examples:

- pooled database access
- non-HTTP protocols
- long-lived stateful connections

---

## Security model

Kimbap's core promise is simple:

> **Raw credentials should not appear in agent-visible env vars, prompts, logs, CLI history, or persisted traces.**

### What Kimbap guarantees

- agents do not receive long-lived raw credentials
- OAuth refresh tokens stay under Kimbap control
- policy is enforced before execution
- risky actions can require human approval
- every action is auditable
- tenant boundaries are enforced by namespace, policy, and key hierarchy

### What Kimbap does not claim

- that the final outbound HTTP request is credential-free
- that proxy mode works for every protocol
- that arbitrary existing CLIs become safe without an adapter
- that host compromise is out of scope

The product wins by being precise and enforceable.

---

## Multi-tenant by design

Kimbap is built for teams running multiple agents, workflows, and customers.

Tenant isolation is enforced in three layers:

1. **Namespace isolation**
   Vault entries, policy, approvals, and audit are tenant-scoped.

2. **Policy isolation**
   Every decision is evaluated in tenant context.

3. **Cryptographic isolation**
   Each tenant has its own key hierarchy.

That means token rotation and key rotation are independent.

---

## Human approval built in

Some actions should not run automatically.

Kimbap makes approval a first-class runtime feature for actions like:

- refunds
- deletes
- external messages
- production changes
- high-risk internal operations

Typical flow:

1. Agent requests a risky action
2. Policy marks it `require_approval`
3. Kimbap creates an approval record
4. Operator approves in console, CLI, or messaging adapter
5. Kimbap executes or rejects
6. Audit records the full decision path

---

## Quick examples

### New code

```bash
kimbap actions list --service github
kimbap actions describe github.list_pull_requests
kimbap call github.list_pull_requests --repo owner/repo
```

### Existing subprocess agent

```bash
kimbap run -- python agent.py
```

### Existing HTTP agent

```bash
kimbap proxy --port 10255
export HTTP_PROXY=http://127.0.0.1:10255
export HTTPS_PROXY=http://127.0.0.1:10255
python agent.py
```

### OAuth-backed service

```bash
kimbap connector login gmail
kimbap call gmail.send_message --to user@example.com --subject "Hello"
```

---

## Why this is different

### Not just a vault

Kimbap stores credentials, but more importantly it **uses** them safely.

### Not just a proxy

Proxy mode is an adoption path, not the product boundary.

### Not just another integration framework

Most services should be declarative skills, not custom runtimes.

### Built for agents, not humans

Existing CLIs assume a human operator. Kimbap assumes:

- agents are untrusted by default
- credentials must stay outside agent memory
- policy and approval are first-class
- audit must be complete
- multi-tenant isolation matters

---

## Typical use cases

- secure GitHub access for coding agents
- Gmail / calendar access for workflow agents
- Stripe and billing actions behind approval
- internal admin tools exposed as governed actions
- multi-agent teams with shared runtime and isolated tenants
- gradual migration from env-var secrets to secure runtime mediation

---

## Status

Kimbap is an active, early-stage product. Today it operates as a **server runtime** with REST interface. The CLI surface (`kimbap call`, `kimbap run`, etc.) is under development as a thin client over the same runtime.

### Current availability

| Capability | Status | Notes |
|---|---|---|
| REST v1 API (`/api/v1`) | Available | Tokens, policies, approvals, audit, action execution |
| Action runtime | Available | Policy evaluation, credential injection, execution |
| Vault (encrypted credential storage) | Available | AES-256-GCM, per-record encryption |
| OAuth connectors | Available | Google, GitHub, Slack |
| Policy engine | Available | YAML DSL, dry-run simulation |
| Approval workflow | Available | Console + Socket.IO real-time |
| Audit trail | Available | Structured events, export |
| Tier 1 skills (YAML) | Available | Declarative REST API integration |
| Console (admin UI) | Available | Monitoring, approvals, audit viewer |
| `kimbap call` (CLI) | In progress | Explicit action execution |
| `kimbap run` (subprocess) | In progress | Agent process wrapper |
| `kimbap proxy` (transparent) | In progress | HTTP/HTTPS proxy mode |
| `kimbap serve` (connected) | In progress | Multi-tenant REST server |
| Embedded mode (local-only) | Planned | Single-user, no server required |
| SDKs | Planned | Python, TypeScript, Go |
| Skill registry | Planned | Install, publish, verify |
| Messaging adapters (Slack, Telegram) | Planned | Approval notification channels |

### Supported interfaces

For all new integrations, use the canonical interfaces:

| Interface | Path | Status | Use When |
|---|---|---|---|
| **REST v1 API** | `/api/v1/*` | **Canonical** | Programmatic access, automation, SDKs |
| Admin API | `/admin` | Legacy (frozen) | Console uses this today; migration to REST v1 planned |
| User API | `/user` | Legacy (frozen) | Console uses this today; migration to REST v1 planned |
| Socket.IO | `/socket.io` | Stable | Real-time events (approvals, notifications) |
| Health | `/health`, `/ready` | Stable | Liveness and readiness probes |

**Console Integration:** Kimbap Console currently communicates with Core via the legacy `/admin` and `/user` endpoints. New external integrations should use `/api/v1`.

---

## Roadmap

### Phase 1 (current)

- Action runtime with REST v1 API
- Vault and token lifecycle
- Tier 1 skill execution via YAML
- OAuth connectors (Google, GitHub, Slack)
- Policy engine and approval workflow
- Audit trail and observability
- Console admin UI

### Phase 2

- CLI surface: `kimbap call`, `kimbap run`, `kimbap proxy`, `kimbap serve`
- Multi-tenant connected mode
- Embedded (local-only) mode

### Phase 3

- Messaging adapters for approval notifications
- Skill registry (install, publish, verify)
- Provenance and verification

### Phase 4

- SDKs (Python, TypeScript, Go)
- Enterprise identity (SPIFFE/OIDC)
- Advanced registry and policy features

---

## Design principles

- **Action Runtime is canonical**
- **One binary, clean internals**
- **No raw credentials in agent space**
- **Existing CLIs are adapters, not trust boundaries**
- **Most integrations should be declarative**
- **Adoption should be incremental**
- **Governance should be built in, not bolted on**

---

## FAQ

### Is Kimbap just a vault?

No. A vault stores secrets. Kimbap uses them safely to execute governed actions.

### Is Kimbap just a proxy?

No. Proxy mode is one adapter. The core is the action runtime.

### Do services need official CLIs?

No. If a service has a usable REST API, Kimbap should usually expose it as actions through a skill.

### How do agents learn to use Kimbap?

Run `kimbap agents setup` in your project directory. This auto-detects installed AI agents (Claude Code, OpenCode, Codex, Cursor) and writes skill files and operating rules into their config directories. Agents then discover available actions via `kimbap actions list` at runtime. For enforcement, combine with `kimbap run`, proxying, and policy.

---

## Vision

AI agents will not become useful in production because they can generate better text alone.

They become useful when they can **take action safely** across real systems.

Kimbap is the runtime for that layer.

---

## Getting Started

### Prerequisites

- Go 1.24 or higher
- PostgreSQL 15+
- Docker (optional, for containerized deployment)

### Installation

1. Clone the repository:

```bash
git clone https://github.com/dunialabs/kimbap-core.git
cd kimbap-core
```

2. Install dependencies:

```bash
make deps
```

3. Set up environment variables:

```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Start PostgreSQL:

```bash
docker compose up -d
```

5. Run the development server:

```bash
make dev
```

The server will start on `http://localhost:3002` by default.

Connection precedence: **1. Database config → 2. `KIMBAP_CORE_URL` env var → 3. Error (no auto-detection)**.

### Common Commands

#### Development

```bash
make dev              # Start development server
make build            # Build the binary
make run              # Build and run
make test             # Run tests
make clean            # Clean build artifacts
```

#### Database

```bash
docker compose up -d              # Start PostgreSQL
docker compose down               # Stop PostgreSQL
docker compose logs postgres      # View PostgreSQL logs
```

#### Docker Deployment

```bash
docker build -t kimbap-core .
docker run -p 3002:3002 --env-file .env kimbap-core
```

---

## Project Structure

```
kimbap-core/
├── cmd/
│   ├── kimbap/           # CLI entry point
│   │   └── main.go
│   └── server/           # Server entry point
│       └── main.go
├── internal/
│   ├── admin/            # Admin API handlers
│   ├── config/           # Configuration management
│   ├── database/         # Database connection & migrations
│   ├── log/              # Log service & sync
│   ├── logger/           # Zerolog wrapper
│   ├── middleware/        # HTTP middleware (auth, rate limit, IP allowlist)
│   ├── oauth/            # OAuth 2.0 implementation
│   ├── repository/       # Data access layer (GORM)
│   ├── security/         # Authentication & authorization
│   ├── service/          # Application services
│   ├── socket/           # Socket.IO real-time communication
│   ├── types/            # Shared types
│   ├── user/             # User API handlers
│   └── utils/            # Utility functions
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── .env.example
```

---

## Companion Applications

Kimbap Console and Kimbap Desk are companion apps that work with Kimbap Core.

### Kimbap Console (Admin Interface)

<details>
<summary>
Kimbap Console is a web-based administration UI for operators and security teams. It communicates with Kimbap Core through the Admin API (for example, <code>POST /admin</code>).
</summary>

#### Key Features

- **User management**
  - Create, query, update, and delete users.
  - Enable or disable accounts.
  - Assign roles and permissions.
  - Configure per-user rate limits.

- **Credential security**
  - Store per-user tokens and credentials encrypted locally with a master password chosen by the user.
  - Optionally unlock the local vault with platform biometrics (Touch ID / Windows Hello) instead of retyping the password.
  - The master key never leaves the device and is never sent to Kimbap Core; only encrypted blobs are stored on disk.

- **Action and connector management**
  - Register and configure connectors and downstream services.
  - Control which actions are exposed per workspace or environment.
  - Enable or disable connectors per workspace.

- **Permission and policy management**
  - Define per-user and per-workspace permissions for actions.
  - Mark high-risk actions as approval-required.
  - Inspect effective permissions for a given user or agent.

- **Monitoring and audit**
  - Browse recent action calls and their outcomes.
  - Inspect audit logs for compliance and debugging.
  - View health indicators for connectors and downstream services.

#### Interaction Model

Kimbap Console talks to Kimbap Core using the Admin API:

- A single `/admin` endpoint with action codes for operations (user, connector, and policy management).
- Authenticated with a Kimbap access token (opaque bearer token) for an Owner/Admin user.
- Designed to be scriptable; you can call the same API from your own automation.
</details>

### Kimbap Desk (Desktop Client)

<details>
<summary>
Kimbap Desk is a desktop application that gives end users a real-time control surface on top of Kimbap Core. It connects to the runtime's Socket.IO endpoints.
</summary>

#### Key Features

- **Capability configuration**
  - Display the actions currently available to the user.
  - Let users further restrict their own capabilities on a per-agent basis.
  - Apply updates in real time when administrators change permissions.

- **Connector configuration**
  - Allow users to configure connectors that require their own credentials (for example, personal API keys or OAuth logins).
  - Unconfigure or revoke previously stored user configuration.
  - Automatically trigger connector startup once configuration is complete.

- **Approval workflow**
  - Receive approval requests when an agent triggers an action that requires human review.
  - Show the parameters the agent intends to send.
  - Let the user approve, reject, or modify the request.

#### Interaction Model

Kimbap Desk uses **Socket.IO** for capability updates, approval requests, and general notifications.

</details>

---

## More Documentation

- [**CLAUDE.md**](./CLAUDE.md)
  Architecture notes, core patterns, and development guidance.

- [**AGENTS.md**](./AGENTS.md)
  Codex-oriented development workflow and knowledge base index.

- [**CONTRIBUTING.md**](./CONTRIBUTING.md)
  Contribution workflow, standards, and verification checklist.

- [**Architecture & Internals**](./docs/architecture.md)
  System architecture, project structure, request/data flows, and core design patterns.

- [**Security & Permissions**](./docs/security.md)
  Vault encryption model (PBKDF2 + AES-GCM) and the three-layer permission model with human-in-the-loop controls.

- [**Deployment & Configuration**](./docs/deployment.md)
  Quick start, Docker and Go binary deployment, environment variables, and common commands.

- [**Reference**](./docs/reference.md)
  Usage examples, API surfaces, testing notes, troubleshooting, contributing, and license.

---

## Troubleshooting

- **Docker not running**
  Ensure Docker Desktop or your Docker daemon is running before using `docker compose up -d` or the Docker deployment script.

- **Port already in use**
  Change `BACKEND_PORT` in your `.env` file or update your Docker configuration if port `3002` is already taken.

- **Database connection failed**
  Check `DATABASE_URL` in your `.env` file, firewall rules, and confirm that the PostgreSQL container is healthy. `docker compose logs postgres` can help diagnose issues.

- **Authentication issues**
  Verify that `JWT_SECRET` and related auth configuration are set consistently across Kimbap Core and any companion applications.

- **Build errors**
  Run `make deps` to ensure all Go dependencies are downloaded. Check that you're using Go 1.24 or higher with `go version`.

For more detailed troubleshooting, see the `docs/` folder or open an issue with logs and reproduction steps.

---

## License

This project is licensed under the [Elastic License 2.0 (ELv2)](https://www.elastic.co/licensing/elastic-license).

**What We Encourage**
Subject to the terms of the Elastic License 2.0, you are encouraged to:

- Freely review, test, and verify the safety and reliability of this product
- Modify and adapt the code for your own use cases
- Apply and integrate this project in a wide variety of scenarios
- Contribute improvements, bug fixes, and other enhancements that help evolve the codebase

**Key Restrictions**:

- You may not provide the software to third parties as a hosted or managed service
- You may not remove or circumvent license key functionality
- You may not remove or obscure licensing notices

For detailed terms, see the official ELv2 text linked above.

Copyright © 2026 [Dunia Labs, Inc.](https://dunialabs.io)

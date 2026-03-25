# Kimbap

> **Secure action runtime for AI agents**
>
> Agents get GitHub, Gmail, Stripe, Notion, and internal APIs. You keep the credentials.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-ELv2-blue.svg)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-blue.svg)

**Quick Start** → [https://kimbap.sh/quick-start](https://kimbap.sh/quick-start) | **Docs** → [https://docs.kimbap.sh](https://docs.kimbap.sh) | **Website** → [https://kimbap.sh](https://kimbap.sh)

---

Kimbap sits between your AI agents and the external systems they need to touch. Instead of giving an agent raw credentials, you give it Kimbap. The agent calls `kimbap call <service>.<action>`. Kimbap handles identity, policy, credential injection, OAuth refresh, approval, and audit. The agent only sees actions and results.

```text
agent → kimbap → policy → approval → credentials → execution → audit
```

---

## The problem with every other approach

| Approach | Credential Safety | Context Cost | Policy | Audit | OAuth Support |
|---|---|---|---|---|---|
| Raw API keys in env | ❌ Leaks into prompts | ✅ Zero | ❌ None | ❌ None | ❌ Manual |
| MCP server | ⚠️ Depends on impl | ❌ 44K+ tokens overhead | ⚠️ Limited | ⚠️ Limited | ⚠️ Complex |
| Direct CLI (gh, aws) | ❌ Human credentials | ✅ Efficient | ❌ None | ❌ None | ⚠️ Human flow |
| **Kimbap CLI** | ✅ Never in agent space | ✅ Efficient | ✅ Built-in | ✅ Full trail | ✅ Runtime-owned |

**Raw API keys** leak into prompts, logs, traces, and shell history. There's no policy layer, no approval workflow, no audit trail, and OAuth refresh is a manual problem per service.

**MCP** looked promising, but the context cost is real and documented. A typical MCP agent injecting 43 tool definitions burns **44,026 tokens** before doing any work. A CLI-based agent completing the same task uses **1,365 tokens** — a 32x difference. Perplexity's CTO publicly moved away from MCP, citing 72% context window consumption before the agent starts. Cloudflare reached the same conclusion and built a code-generation alternative.

**Service CLIs** like `gh`, `aws`, and `stripe` assume a human operator with credentials loaded. They're not trust boundaries. An agent using `gh` is using your GitHub token directly, with no policy layer and no audit trail.

---

## Why CLI is the right interface for agents

**LLMs are trained on billions of terminal interactions.** They understand CLI patterns natively, without schema injection eating context.

`kimbap call github.list_pull_requests` is unambiguous and composable. The output is structured and predictable. `--help` is just-in-time documentation — no pre-loaded schemas burning context before the agent does anything. CLI tools chain naturally with Unix pipes. Terminal-native agents like Claude Code and OpenCode already think in CLI terms.

But Kimbap CLI is not a raw CLI.

Raw CLIs like `gh` or `aws` assume a credentialed human operator. Kimbap is the trust boundary. Credentials never reach agent space. Policy and approval are enforced before execution. Every action is logged. OAuth tokens stay under Kimbap control — agents never see refresh tokens. One runtime model covers all services.

---

## What it looks like

```bash
# OAuth-backed service — agent never sees the token
kimbap connector login gmail
kimbap call gmail.send_message --to user@example.com --subject "Hello"

# Explicit action calls
kimbap call github.list_pull_requests --repo owner/repo
kimbap call stripe.refund_charge --charge ch_abc --amount 500

# Auto-detect Claude Code, OpenCode, Codex, Cursor and write skill files
kimbap agents setup

# Wrap an existing agent process
kimbap run -- python agent.py

# Transparent proxy for HTTP agents
kimbap proxy --port 10255
```

---

## How it works

Every action goes through the same runtime pipeline:

```text
identity → action lookup → policy → approval → credential injection → execution → audit
```

One security model. One policy layer. One audit trail. One integration model — from local dev to multi-tenant deployment.

### Credentials stay out of agent space

Agents don't receive API keys or OAuth tokens. They call actions. Kimbap injects credentials at execution time, under the runtime's control. OAuth refresh tokens never leave Kimbap.

### Policy before execution

Policy is a YAML DSL with per-action rules and dry-run simulation. Mark any action `require_approval` and risky calls (refunds, deletes, production changes) get human sign-off before anything runs.

### Approval workflow

1. Agent requests a risky action
2. Policy marks it `require_approval`
3. Kimbap creates an approval record
4. Operator approves in console, CLI, or messaging adapter
5. Kimbap executes or rejects
6. Audit records the full decision path

### Integrations stay declarative

Most APIs don't need custom code. A YAML skill file specifies auth type, endpoint shape, arguments, error mapping, pagination, output extraction, and risk metadata. That turns any REST API into a governed Kimbap action. For OAuth lifecycle, Kimbap uses runtime-owned connectors (Google, GitHub, Slack available now).

---

## Four ways to use it

Kimbap ships as a **single Go binary**.

**Explicit action** — for new agent systems and direct integrations:
```bash
kimbap call github.list_pull_requests --repo owner/repo
```

**Subprocess wrapper** — wrap an existing agent process:
```bash
kimbap run -- python agent.py
```

**Transparent proxy** — for HTTP agents with minimal code changes:
```bash
kimbap proxy --port 10255
export HTTP_PROXY=http://127.0.0.1:10255
```

**Connected server** — for shared deployments and multi-tenant use:
```bash
kimbap serve --port 8080
```

---

## Agent onboarding

`kimbap agents setup` auto-detects Claude Code, OpenCode, Codex, and Cursor, then writes skill files and operating rules into their config directories.

```bash
kimbap agents setup                           # detect and configure all agents
kimbap skill install slack.yaml               # add a new skill
kimbap agents sync                            # propagate to connected agents
kimbap agents sync --agent claude-code        # target a specific agent
kimbap agents sync --dry-run                  # preview without writing
```

Each agent gets per-service `SKILL.md` files in its skill directory (e.g. `.claude/skills/github/SKILL.md`), a meta-skill for runtime discovery, and `KIMBAP_OPERATING_RULES.md` for credential handling policies. Agents can also discover actions at runtime via `kimbap actions list` without synced skill files — the meta-skill teaches that fallback.

---

## Multi-tenant by design

Tenant isolation runs three layers deep:

1. **Namespace isolation** — vault entries, policy, approvals, and audit are tenant-scoped
2. **Policy isolation** — every decision is evaluated in tenant context
3. **Cryptographic isolation** — each tenant has its own key hierarchy

Token rotation and key rotation are independent.

---

## Status

Kimbap is early-stage. The action runtime and REST v1 API are available now. The CLI surface (`kimbap call`, `kimbap run`, `kimbap proxy`) is in progress.

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

### API interfaces

For all new integrations, use `/api/v1`:

| Interface | Path | Status | Use when |
|---|---|---|---|
| **REST v1 API** | `/api/v1/*` | Canonical | Programmatic access, automation, SDKs |
| Admin API | `/admin` | Legacy (frozen) | Console uses this today |
| User API | `/user` | Legacy (frozen) | Console uses this today |
| Socket.IO | `/socket.io` | Stable | Real-time events (approvals, notifications) |
| Health | `/health`, `/ready` | Stable | Liveness and readiness probes |

---

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL 15+
- Docker (optional, for containerized deployment)

### Installation

```bash
git clone https://github.com/dunialabs/kimbap-core.git
cd kimbap-core
make deps
cp .env.example .env   # edit with your configuration
docker compose up -d   # start PostgreSQL
make dev               # server starts on http://localhost:3002
```

### Common commands

```bash
make dev              # start development server
make build            # build the binary
make run              # build and run
make test             # run tests
make clean            # clean build artifacts
```

```bash
docker compose up -d              # start PostgreSQL
docker compose down               # stop PostgreSQL
docker compose logs postgres      # view PostgreSQL logs
```

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

- [**CLAUDE.md**](./CLAUDE.md) — Architecture notes, core patterns, and development guidance.
- [**AGENTS.md**](./AGENTS.md) — Codex-oriented development workflow and knowledge base index.
- [**CONTRIBUTING.md**](./CONTRIBUTING.md) — Contribution workflow, standards, and verification checklist.
- [**Architecture & Internals**](./docs/architecture.md) — System architecture, project structure, request/data flows, and core design patterns.
- [**Security & Permissions**](./docs/security.md) — Vault encryption model (PBKDF2 + AES-GCM) and the three-layer permission model with human-in-the-loop controls.
- [**Deployment & Configuration**](./docs/deployment.md) — Quick start, Docker and Go binary deployment, environment variables, and common commands.
- [**Reference**](./docs/reference.md) — Usage examples, API surfaces, testing notes, troubleshooting, contributing, and license.

---

## Troubleshooting

- **Docker not running** — ensure Docker Desktop or your Docker daemon is running before `docker compose up -d`.
- **Port already in use** — change `BACKEND_PORT` in `.env` if port 3002 is taken.
- **Database connection failed** — check `DATABASE_URL` in `.env`, firewall rules, and confirm the PostgreSQL container is healthy. `docker compose logs postgres` helps.
- **Authentication issues** — verify that `JWT_SECRET` is set consistently across Kimbap Core and any companion applications.
- **Build errors** — run `make deps` to download Go dependencies. Check you're on Go 1.24+ with `go version`.

For more, see the `docs/` folder or open an issue with logs and reproduction steps.

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

# Kimbap

> **Secure action runtime for AI agents**
>
> Agents get GitHub, Gmail, Stripe, Notion, and internal APIs. You keep the credentials.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-blue.svg)

**Quick Start** → [https://kimbap.sh/quick-start](https://kimbap.sh/quick-start) | **Docs** → [https://docs.kimbap.sh](https://docs.kimbap.sh) | **Website** → [https://kimbap.sh](https://kimbap.sh)

---

Instead of giving agents raw credentials, you give them Kimbap. Agents call `kimbap call <service>.<action>`. Kimbap holds the credentials, enforces policy, handles OAuth refresh, routes approvals, and logs everything. The agent only sees actions and results.

```text
agent → kimbap → policy → approval → credentials → execution → audit
```

**Terminology**
- **service**: a declarative action manifest (REST API integration)
- **connector**: an OAuth authentication connection

---

## Why CLI, not MCP or raw keys

| Approach | Credential Safety | Context Cost | Policy | Audit | OAuth Support |
|---|---|---|---|---|---|
| Raw API keys in env | ❌ Leaks into prompts | ✅ Zero | ❌ None | ❌ None | ❌ Manual |
| MCP server | ⚠️ Depends on impl | ❌ 44K+ tokens overhead | ⚠️ Limited | ⚠️ Limited | ⚠️ Complex |
| Direct CLI (gh, aws) | ❌ Human credentials | ✅ Efficient | ❌ None | ❌ None | ⚠️ Human flow |
| **Kimbap CLI** | ✅ Never in agent space | ✅ Efficient | ✅ Built-in | ✅ Full trail | ✅ Runtime-owned |

**Raw API keys** leak into prompts, logs, traces, and shell history. There's no policy layer, no approval workflow, no audit trail, and OAuth refresh is a manual problem per service.

**MCP** tool schema injection can consume tens of thousands of tokens before the agent does any work. Perplexity and Cloudflare independently moved away from MCP citing context window overhead.

**Service CLIs** like `gh`, `aws`, and `stripe` assume a human operator with credentials loaded. An agent using `gh` is using your GitHub token directly, with no policy layer and no audit trail.

LLMs are trained on terminal interactions and understand CLI patterns natively. `kimbap call <service>.<action>` is unambiguous, composable, and predictable. `--help` is just-in-time documentation with no pre-loaded schemas burning context before the agent does anything. Agents like Claude Code and OpenCode already think in CLI terms. But Kimbap CLI is the trust boundary, not a raw CLI. Policy and approval are enforced before execution, OAuth tokens stay under Kimbap control, and every action is logged.

---

## What Kimbap does

### OAuth without credential exposure

Kimbap handles OAuth differently:

1. An operator completes the login once via CLI or browser flow
2. Kimbap stores and refreshes the token. It never leaves the runtime.
3. The agent calls `kimbap call gmail.send_message` and only receives the action result

```bash
# Operator logs in once (browser OAuth flow)
kimbap connector login gmail

# Agent calls the action (no token involved)
kimbap call gmail.send_message --to user@example.com --subject "Hello"
```

### Any REST API becomes a CLI action

Write a YAML service file (auth type, endpoint, arguments, error mapping) and `kimbap call yourservice.action` works immediately. No custom code per integration.

```yaml
# slack.yaml (simplified)
name: slack
auth: bearer
actions:
  post_message:
    method: POST
    path: /chat.postMessage
    args: [channel, text]
    risk: low
```

Most modern APIs can be described this way. For services that need OAuth lifecycle (token refresh, device flow), Kimbap uses runtime-owned connectors. Google, GitHub, and Slack are available now.

### Policy, approval, audit

Policy is a YAML DSL with per-action rules and dry-run simulation. Mark any action `require_approval` and Kimbap pauses execution, notifies the operator (console or webhook), and waits for sign-off. Every action is logged with the full decision path.

---

## Usage modes

Kimbap ships as a **single Go binary**.

**Explicit action**, for new agent systems and direct integrations:
```bash
kimbap call github.list_pull_requests --repo owner/repo
```

**Subprocess wrapper**: wrap an existing agent process:
```bash
kimbap run -- python agent.py
```

**Transparent proxy**, for HTTP agents with minimal code changes:
```bash
kimbap proxy --port 10255
export HTTP_PROXY=http://127.0.0.1:10255
```

**Connected server** (in progress), for shared deployments and multi-tenant use:
```bash
kimbap serve --port 8080
```

---

## Agent onboarding

`kimbap agents setup` auto-detects Claude Code, OpenCode, Codex, and Cursor, then writes service files and operating rules into their config directories.

```bash
kimbap agents setup                           # detect and configure all agents
kimbap service install slack.yaml             # add a new service
kimbap agents sync                            # propagate to connected agents
kimbap agents sync --agent claude-code        # target a specific agent
kimbap agents sync --dry-run                  # preview without writing
```

Each agent gets `SKILL.md` files in its config directory, a meta-service for runtime discovery, and `KIMBAP_OPERATING_RULES.md` for credential handling policies.

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
| Approval workflow | Available | Console + webhook notification |
| Audit trail | Available | Structured events, export |
| Tier 1 services (YAML) | Available | Declarative REST API integration |
| Console (admin UI) | Available | Monitoring, approvals, audit viewer |
| `kimbap call` (CLI) | Available | Explicit action execution |
| `kimbap run` (subprocess) | Available | Agent process wrapper |
| `kimbap proxy` (transparent) | Available | HTTP/HTTPS proxy mode |
| `kimbap serve` (connected) | Available | Multi-tenant REST server |
| Embedded mode (local-only) | Available | SQLite-backed, no external DB required |
| SDKs | Planned | Python, TypeScript, Go |
| Service registry | Planned | Install, publish, verify |
| Webhook notifications (Slack, Telegram, Email, Webhook) | Available | Approval notification channels; configure via YAML |

### API interfaces

For all new integrations, use `/api/v1`:

| Interface | Path | Status | Use when |
|---|---|---|---|
| **REST v1 API** | `/api/v1/*` | Canonical | Programmatic access, automation, SDKs |
| Admin API | `/admin` | Legacy (frozen) | Console uses this today |
| User API | `/user` | Legacy (frozen) | Console uses this today |
| Health | `/health`, `/ready` | Stable | Liveness and readiness probes |

### Notification configuration

Configure approval notification channels in `~/.kimbap/config.yaml` or via environment variables:

```yaml
notifications:
  slack:
    webhook_url: ""        # KIMBAP_NOTIFICATIONS_SLACK_WEBHOOK_URL
  telegram:
    bot_token: ""          # KIMBAP_NOTIFICATIONS_TELEGRAM_BOT_TOKEN
    chat_id: ""            # KIMBAP_NOTIFICATIONS_TELEGRAM_CHAT_ID
  email:
    smtp_host: ""          # KIMBAP_NOTIFICATIONS_EMAIL_SMTP_HOST
    smtp_port: 587         # KIMBAP_NOTIFICATIONS_EMAIL_SMTP_PORT
    from: ""               # KIMBAP_NOTIFICATIONS_EMAIL_FROM
    to: []                 # KIMBAP_NOTIFICATIONS_EMAIL_TO (comma-separated in env)
  webhook:
    url: ""                # KIMBAP_NOTIFICATIONS_WEBHOOK_URL
    sign_key: ""           # KIMBAP_NOTIFICATIONS_WEBHOOK_SIGN_KEY
```

---

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL 15+ (required for server/shared mode; not needed for local CLI-only use)
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

<details>
<summary>Project Structure</summary>

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

</details>

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
  - Kimbap Console supports an owner master-password flow: during initialization, the owner access token is encrypted with PBKDF2 + AES-256-GCM using the master password, and the encrypted blob is stored in Kimbap Core.
  - Kimbap Core stores runtime credentials server-side and injects them only at execution time. Vault secrets use per-tenant envelope encryption (AES-256-GCM, per-record DEK wrapped by a tenant-scoped KEK).

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

<details>
<summary>Shared deployments</summary>

Kimbap Core carries tenant context through the full runtime pipeline: policy, approvals, audit, vault, and encryption. For shared server deployments, vault keys are derived per tenant for cryptographic separation.

Connected server mode (`kimbap serve`) is in progress.

</details>

---

## More Documentation

- [**CLAUDE.md**](./CLAUDE.md): Architecture notes, core patterns, and development guidance.
- [**AGENTS.md**](./AGENTS.md): Codex-oriented development workflow and knowledge base index.
- [**CONTRIBUTING.md**](./CONTRIBUTING.md): Contribution workflow, standards, and verification checklist.
- [**Architecture & Internals**](./docs/architecture.md): System architecture, project structure, request/data flows, and core design patterns.
- [**Security & Permissions**](./docs/security.md): Vault encryption model (PBKDF2 + AES-GCM) and the three-layer permission model with human-in-the-loop controls.
- [**Deployment & Configuration**](./docs/deployment.md): Quick start, Docker and Go binary deployment, environment variables, and common commands.
- [**Reference**](./docs/reference.md): Usage examples, API surfaces, testing notes, troubleshooting, contributing, and license.

---

## Troubleshooting

- **Docker not running**: ensure Docker Desktop or your Docker daemon is running before `docker compose up -d`.
- **Port already in use**: change `BACKEND_PORT` in `.env` if port 3002 is taken.
- **Database connection failed**: check `DATABASE_URL` in `.env`, firewall rules, and confirm the PostgreSQL container is healthy. `docker compose logs postgres` helps.
- **Authentication issues**: verify that `JWT_SECRET` is set consistently across Kimbap Core and any companion applications.
- **Build errors**: run `make deps` to download Go dependencies. Check you're on Go 1.24+ with `go version`.

For more, see the `docs/` folder or open an issue with logs and reproduction steps.

---

## License

MIT License. Copyright © 2026 [Dunia Labs, Inc.](https://dunialabs.io)

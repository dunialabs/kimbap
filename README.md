# Kimbap

> **Turn any API into a secure CLI action for AI agents.**
> Define actions in YAML. Kimbap handles credentials, policy, approval, audit — and generates SKILL.md so agents discover and use them automatically.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

[Quick Start](https://kimbap.sh/quick-start) · [Docs](https://docs.kimbap.sh) · [Website](https://kimbap.sh)

---

```yaml
# stripe.yaml — define a service and its actions
name: stripe
adapter: http
base_url: https://api.stripe.com/v1
auth:
  type: bearer
  credential_ref: stripe.api_key
actions:
  list-charges:
    method: GET
    path: /charges
    risk: { level: low }
  create-refund:
    method: POST
    path: /refunds
    risk: { level: high }
```

```bash
kimbap service install stripe.yaml
kimbap call stripe.list-charges
```

The agent calls `kimbap call`. Kimbap injects credentials at execution time, enforces policy, and logs the result. The agent does not access stored credentials directly.

---

## Agents discover actions automatically

```bash
kimbap agents sync
```

Detects supported agents (Claude Code, Cursor, Codex, OpenCode) and writes SKILL.md files for installed services into their skills directories. Each file describes actions, arguments, and risk levels — so agents call `kimbap call` without handling credentials directly.

---

## Why Kimbap

| Problem | Without Kimbap | With Kimbap |
|---|---|---|
| **Credentials** | API keys in env vars, visible to agent | Encrypted vault, injected at execution time |
| **OAuth expiry** | Agent breaks when token expires | Automatic refresh, handled by Kimbap |
| **Access control** | Per-service, manual, fragmented | One policy engine across all services |
| **Risky operations** | No guardrails | Human approval for destructive actions |
| **Audit** | Scattered logs per service | Unified trail for every action and decision |
| **New integrations** | Custom wrapper per API | One YAML file |

---

## Add your own service

Three adapters, same pipeline:

- **HTTP** — REST APIs. Most SaaS services.
- **Command** — wraps any CLI tool.
- **AppleScript** — macOS native apps (Finder, Calendar, Notes, Safari, etc.) directly, no MCP server needed.

```bash
kimbap service validate my-service.yaml
kimbap service install my-service.yaml
kimbap call my-service.my-action
```

See the [Command Reference](./kimbap-core/docs/commands.md) for the full manifest schema.

---

## Built-in capabilities

| Capability | Description |
|---|---|
| **Vault** | AES-256-GCM encrypted credential storage |
| **Policy** | YAML rules — `allow`, `deny`, or `require_approval` per agent, service, or action |
| **Approval** | Human-in-the-loop via Slack, Telegram, webhook, or console |
| **Audit** | Structured log of every action, decision, and outcome |
| **OAuth** | Automatic token refresh for Google, Slack, HubSpot, GitHub Apps, and more |

---

## 50+ built-in services

Includes GitHub, Slack, Stripe, Notion, Linear, HubSpot, Telegram, WhatsApp, Gmail, Finder, Safari, Notes, Calendar, Blender, Ollama, Spotify, Wikipedia, and more. [Full list →](./kimbap-core/docs/commands.md#action-discovery)

---

## Install

```bash
brew install kimbap
```

Or [build from source](./kimbap-core/docs/deployment.md) (Go 1.24+) or run via [Docker](./kimbap-core/docs/deployment.md). macOS and Linux supported.

---

## Quick setup

```bash
printf '%s' "$GITHUB_TOKEN" | kimbap vault set github.token --stdin
kimbap link github
kimbap call github.list-repos --owner octocat
```

See [Quick Start](https://kimbap.sh/quick-start) for OAuth, server mode, and token setup.

---

## Documentation

| Doc | Description |
|---|---|
| **[Command Reference](./kimbap-core/docs/commands.md)** | Full CLI command list with usage and examples |
| [Architecture](./kimbap-core/docs/architecture.md) | Internals and module structure |
| [Security](./kimbap-core/docs/security.md) | Trust model, encryption, credential isolation |
| [Deployment](./kimbap-core/docs/deployment.md) | Configuration, Docker, production setup |
| [REST API](./kimbap-core/docs/api/API.md) | Endpoint reference for connected mode |
| [Contributing](./kimbap-core/CONTRIBUTING.md) | How to add services and contribute |

---

MIT License · [Dunia Labs](https://dunialabs.io) · [kimbap.sh](https://kimbap.sh)

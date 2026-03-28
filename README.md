# Kimbap

<p align="center">
  <img src="docs/logo.svg" width="120" alt="Kimbap">
</p>

*Do you like kimbap? It's delicious, healthy, and you can make your own.*

> **Turn anything into a CLI your agent can use. Securely.**
> REST APIs, CLI tools, macOS apps — one YAML, one command.
> Credentials never enter the agent process.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

[Quick Start](https://kimbap.sh/quick-start) · [Docs](https://docs.kimbap.sh) · [Website](https://kimbap.sh)

---

## How it works

```
agent → kimbap → policy → approval → credentials → execute → audit
```

Every action goes through the same pipeline. Credentials live in an encrypted vault and are injected at execution time. They never enter the agent process — not in env vars, not in prompts, not in logs.

| Without kimbap | With kimbap |
|---|---|
| API keys passed via env vars — leak into logs and prompts | Encrypted vault injects credentials at execution time |
| Every service has different auth | One manifest format for REST APIs, CLI tools, and macOS apps |
| Agents run dangerous actions unchecked | Policies and approvals enforced on every action |
| No record of what the agent did | Audit trail on every action, automatically |

---

## Try it now

**1. Install**

```bash
curl -fsSL https://raw.githubusercontent.com/dunialabs/kimbap/main/install.sh | bash
```

Or with Homebrew: `brew install kimbap`

**2. Initialize**

```bash
kimbap init --mode dev --services all
```

**3. Call**

```bash
# Send a Slack message
kimbap call slack.send-message --channel general --text "deployed v2.1"

# List Stripe charges
kimbap call stripe.list-charges --limit 5

# Search notes — no API key needed (macOS)
kimbap call apple-notes.search-notes --query "meeting"
```

Agent calls `kimbap call <service>.<action>`. That's it.

---

## Add a service

**1. Connect**

```bash
printf '%s' "$GITHUB_TOKEN" | kimbap link github --stdin
```

**2. Call**

```bash
kimbap call github.list-repos --sort updated
```

`kimbap link` works for both API keys (`--stdin`/`--file`) and OAuth services (Slack, Notion, Zoom, and more).

---

## Connect your AI agent

```bash
kimbap agents setup
```

Auto-detects and configures Claude Code, OpenCode, Codex, Cursor, and any agent that can run a CLI command.

---

## Build your own service

Add any REST API, CLI tool, or macOS app with a single YAML manifest:

```yaml
name: stripe
version: 1.0.0
base_url: https://api.stripe.com/v1
auth:
  type: bearer
  credential_ref: stripe.api_key
actions:
  list-charges:
    method: GET
    path: /charges
    risk:
      level: low
```

```bash
kimbap service install stripe.yaml
kimbap call stripe.list-charges
```

Three adapter types: **HTTP** (REST APIs), **Command** (CLI wrappers), **AppleScript** (macOS native apps).

Full schema and examples: **[Service Development Guide](./docs/service-development.md)**

---

## Built-in services

### SaaS & APIs

GitHub · Slack · Stripe · Notion · Linear · HubSpot · Airtable · Pinecone · Todoist · PostHog · Sentry · SendGrid · Resend · Exa · Brave Search

### Communication

Telegram · WhatsApp · WeChat · Zoom · Apple Mail · Messages

### Local apps

Blender · ComfyUI · Ollama · Mermaid · Spotify · NotebookLM

### macOS native

Finder · Safari · Contacts · Shortcuts · Apple Notes · Apple Calendar · Apple Reminders · Keynote · Pages · Numbers

### Office

Microsoft Word · Excel · PowerPoint

### Data

Wikipedia · Hacker News · CoinGecko · Open-Meteo (weather, air quality, historical, geocoding) · Financial Datasets · REST Countries · Exchange Rate · Public Holidays · Nominatim · ntfy

One command for all of them: `kimbap call <service>.<action>`

---

## Advanced

### Integration modes

| Mode | Command | Use case |
|---|---|---|
| **Call** (recommended) | `kimbap call <service>.<action>` | Direct use, scripts, agent integration |
| Run | `kimbap run -- <cmd>` | Wrap any agent subprocess |
| Proxy | `kimbap proxy` | Existing HTTP agents, zero code changes |
| Serve | `kimbap serve` | Connected-mode REST API server |

All modes share the same policy, credentials, and audit pipeline.

### Production setup

```bash
export KIMBAP_MASTER_KEY_HEX="$(openssl rand -hex 32)"
kimbap init --services all
```

### Documentation

- **[Installation Guide](./docs/installation.md)** — step-by-step setup, agent-readable
- **[CLI Reference](./docs/cli-reference.md)** — commands, flags, configuration
- **[Service Development Guide](./docs/service-development.md)** — manifest authoring, adapters
- **[Architecture & Internals](./docs/architecture.md)**
- **[Security & Permissions](./docs/security.md)**
- **[Deployment & Configuration](./docs/deployment.md)**
- **[HTTP API Reference](./docs/api/API.md)**

---

## Contributing

Write a YAML manifest, validate it, open a PR. See **[CONTRIBUTING.md](./CONTRIBUTING.md)**.

---

MIT License · [Dunia Labs](https://dunialabs.io)

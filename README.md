# Kimbap

*Do you like kimbap? It's delicious, healthy, and you can make your own.*

> **Secure action runtime for AI agents.**
> APIs, CLIs, OAuth services — governed agent actions with one CLI.
> Credentials never touch the agent.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Services](https://img.shields.io/badge/services-54-orange.svg)

[Quick Start](https://kimbap.sh/quick-start) · [Docs](https://docs.kimbap.sh) · [Website](https://kimbap.sh)

---

## How it works

```
agent → kimbap → policy → approval → credentials → execute → audit
```

Agent calls `kimbap call <service>.<action>`. That's it.

Credentials live in an encrypted vault and are injected at execution time. They never enter the agent process — not in env vars, not in prompts, not in logs. Policy, approval, and audit apply to every action automatically.

---

## Get started

### Tell your agent (recommended)

Paste this into Claude Code, Cursor, OpenCode, or any CLI-capable agent:

```
Read https://raw.githubusercontent.com/dunialabs/kimbap/main/docs/installation.md
and set up kimbap for this project.
```

### Or install yourself

```bash
curl -fsSL https://raw.githubusercontent.com/dunialabs/kimbap/main/install.sh | bash
```

In interactive shells, the installer prompts to run `kimbap init`. If skipped or non-interactive, run manually:

```bash
kimbap init --mode dev
```

Or with Homebrew:

```bash
brew install kimbap
kimbap init --mode dev
```

> **Production?** Set `KIMBAP_MASTER_KEY_HEX` instead of using `--mode dev`.

### Try it — no API keys needed (macOS)

```bash
kimbap call apple-notes.list-notes
```

Your notes are listed in the terminal. No credentials, no setup. macOS may prompt for Automation access on first use.

---

## Works with

Claude Code · OpenCode · Cursor · Codex · any agent that can run a CLI command.

```bash
kimbap profile install claude-code   # installs operating rules for your agent
kimbap agents setup                  # auto-detect and configure installed agents
kimbap agents sync                   # sync SKILL.md to agent discovery directories
```

---

## 54 built-in services

### SaaS & APIs

GitHub · Slack · Stripe · Notion · Linear · HubSpot · Airtable · Pinecone · Todoist · PostHog · Sentry · SendGrid · Resend · Exa · Brave Search

### Communication

Telegram · WhatsApp · WeChat · Zoom · Apple Mail · Messages

### Local applications

Blender · ComfyUI · Ollama · Mermaid · Spotify · NotebookLM

### macOS native

Finder · Safari · Contacts · Shortcuts · Notes · Calendar · Reminders · Keynote · Pages · Numbers

### Office & data

Microsoft Word · Excel · PowerPoint · Wikipedia · Hacker News · CoinGecko · Open-Meteo (weather, air quality, historical, geocoding) · Financial Datasets · REST Countries · Exchange Rate · Public Holidays · Nominatim · ntfy · Peta

---

## 4 modes

| Mode | Command | Use case |
|---|---|---|
| Call | `kimbap call <service>.<action>` | Direct use, scripts, agent integration |
| Run | `kimbap run -- <cmd>` | Wrap any agent subprocess |
| Proxy | `kimbap proxy` | Existing HTTP agents, zero code changes |
| Serve | `kimbap serve` | Persistent daemon with HTTP API |

All modes go through the same pipeline. Same credentials, same policy, same audit.

---

## Build your own

Add any REST API with a single YAML file:

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

Three adapter types: **HTTP** (REST APIs), **Command** (CLI wrappers), **AppleScript** (macOS native apps).

Validate before install:

```bash
kimbap service validate my-service.yaml
kimbap service install my-service.yaml
```

Full schema and examples: **[Service Development Guide](./docs/service-development.md)**

---

## Documentation

- **[Installation Guide](./docs/installation.md)** — step-by-step setup, agent-readable
- **[CLI Reference](./docs/cli-reference.md)** — commands, flags, configuration
- **[Service Development Guide](./docs/service-development.md)** — manifest authoring, adapters, contributing services
- **[Architecture & Internals](./docs/architecture.md)**
- **[Security & Permissions](./docs/security.md)**
- **[Deployment & Configuration](./docs/deployment.md)**
- **[HTTP API Reference](./docs/api/API.md)**

---

## Contributing

Write a YAML manifest, validate it, open a PR. See **[CONTRIBUTING.md](./CONTRIBUTING.md)**.

---

MIT License · [Dunia Labs](https://dunialabs.io)

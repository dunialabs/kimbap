# Kimbap

*Do you like kimbap? It's delicious, healthy, and you can make your own.*

> **Turn anything into a CLI your agent can use. Securely.**
> REST APIs, CLI tools, macOS apps — one YAML, one command.
> Credentials never enter the agent process.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

[Quick Start](https://kimbap.sh/quick-start) · [Docs](https://docs.kimbap.sh) · [Website](https://kimbap.sh)

---

## This is kimbap

```bash
# Send a Slack message
$ kimbap call slack.send-message --channel general --text "deployed v2.1"
✓ sent

# List Stripe charges
$ kimbap call stripe.list-charges --limit 5
✓ 5 charges (JSON)

# Search your notes — no API key needed (macOS)
$ kimbap call apple-notes.search-notes --query "meeting"
✓ 3 notes found
```

Agent calls `kimbap call <service>.<action>`. That's it.

Credentials live in an encrypted vault and are injected at execution time. They never enter the agent process — not in env vars, not in prompts, not in logs.

---

## Why kimbap?

| Without kimbap | With kimbap |
|---|---|
| Hand API keys to agents via env vars — they can leak into logs and prompts | Encrypted vault injects credentials at execution time. Never enters the agent process |
| Every service has different auth — painful to integrate | One manifest format works across REST APIs, CLI tools, and macOS apps |
| Agents run dangerous actions unchecked | Policies and approvals are enforced on every action |
| No idea what the agent actually did | Audit trail on every action, automatically |

```
agent → kimbap → policy → approval → credentials → execute → audit
```

Every action goes through the same pipeline.

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

In interactive shells, the installer prompts to run `kimbap init`. If skipped or non-interactive, run manually (the installer prints the full path if the binary isn't on your PATH):

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

## Turn any API into a CLI

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

```bash
kimbap vault set stripe.api_key      # store credential once
kimbap service install stripe.yaml
kimbap call stripe.list-charges       # done
```

Three adapter types: **HTTP** (REST APIs), **Command** (CLI wrappers), **AppleScript** (macOS native apps).

Wrap a REST API, CLI tool, or macOS app as an agent-usable action in minutes.

Full schema and examples: **[Service Development Guide](./docs/service-development.md)**

---

## Built-in services

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

One command for all of them. `kimbap call <service>.<action>`
New services added regularly. Or turn your own API into a CLI.

---

## Works with

Claude Code · OpenCode · Cursor · Codex · any agent that can run a CLI command.

```bash
kimbap profile install claude-code   # installs operating rules for your agent
kimbap agents setup                  # auto-detect and configure installed agents
kimbap agents sync                   # sync SKILL.md to agent discovery directories
```

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

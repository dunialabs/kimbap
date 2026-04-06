# Kimbap

<img src="docs/logo.svg" width="120" alt="Kimbap">

*Do you like kimbap? It's delicious, healthy, and you can make your own.*

> **Turn anything into a CLI your agent can use. Securely.**
> REST APIs, CLI tools, macOS apps — one YAML, one command.
> Credentials never enter the agent process.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

[Quick Start](https://kimbap.sh/docs/quick-start) · [Docs](https://kimbap.sh/docs) · [Website](https://kimbap.sh)

---

## Try it now

**1. Install**

```bash
curl -fsSL https://kimbap.sh/install.sh | bash
```

Or with Homebrew:

```bash
brew install dunialabs/kimbap/kimbap
```

**2. Initialize**

```bash
kimbap init --services select
```

`select` opens an interactive checklist with a preselected recommended set (and an `all` command). Shortcut aliases are enabled by default; use `--no-shortcuts` to opt out.

Pick what you want to install, then run immediately with shortcuts.

**3. Run immediately with shortcut commands**

```bash
geosearch --name "San Francisco"
```

Then fetch weather:

```bash
weather --latitude 37.7749 --longitude -122.4194
```

No API key, no localhost setup.

---

## Connect your AI agent

```bash
kimbap agents setup
```

Detects installed agents and installs global discovery hints.
Use `kimbap agents setup --sync --dir <project>` when you want project-local skill sync.

Works with Claude Code, OpenCode, Codex, OpenClaw, NanoClaw, and any agent that can run a CLI command.

---

## Turn your API into a CLI

Got a REST API? Turn it into a secure CLI tool in four steps:

1) Start from your API endpoint.

2) Define it in one YAML file:

```yaml
name: inventory-api
version: 1.0.0
aliases: [inventory]
base_url: https://api.internal.company.com/v1
auth:
  type: bearer
  credential_ref: inventory_api.token
actions:
  list-items:
    aliases: [items]
    method: GET
    path: /warehouses/{warehouse}/items
    params:
      warehouse:
        required: true
    risk:
      level: low
```

3) Install the service definition:

```bash
kimbap service install inventory-api.yaml
```

Register once.

4) Run it as a direct CLI command:

```bash
items --warehouse seoul
```

Then run it like a native CLI command.

Three adapter types: **HTTP** (REST APIs), **Command** (local executable CLIs), **AppleScript** (macOS native apps).

Full schema and examples: **[Service Development Guide](./docs/service-development.md)**

Already have an OpenAPI 3.x spec? Start there instead:

```bash
kimbap service generate --openapi ./openapi.yaml --output inventory-api.yaml
kimbap service generate --openapi http://127.0.0.1:8080/openapi.yaml --name inventory-api --install
```

Remote OpenAPI URLs must use `https://`. Plain `http://` is only allowed for localhost/loopback during local development.

Local file specs may use split-file relative `$ref` chains.

---

## Built-in services

### SaaS & APIs

GitHub · Slack · Stripe · Notion · Linear · HubSpot · Airtable · Pinecone · Todoist · PostHog · Sentry · SendGrid · Resend · Exa · Brave Search · Peta

### Communication

Telegram · WhatsApp · Zoom · Apple Mail · Messages

### Local apps

Blender · ComfyUI · Ollama · Mermaid · Kitty · Spotify

### macOS native

Finder · Safari · Contacts · Shortcuts · Apple Notes · Apple Calendar · Apple Reminders · Keynote · Pages · Numbers

### Office

Microsoft Word · Excel · PowerPoint

### Data

Wikipedia · Hacker News · CoinGecko · Open-Meteo (weather, air quality, historical, geocoding) · Financial Datasets · REST Countries · Exchange Rate · Public Holidays · Nominatim · ntfy

If you need a dedicated MCP control plane (gateway, policy, audit), see [dunialabs/peta-core](https://github.com/dunialabs/peta-core).

Direct call format: `kimbap call <service>.<action>`

Optional frictionless shortcut (per action):

`kimbap alias set <shortcut> <service>.<action>` → run `<shortcut> ...`

You can inspect configured shortcuts with `kimbap service list` (`SHORTCUTS` column).
To browse built-in services before installing, use `kimbap service list --available`, `kimbap service search <query>`, and `kimbap service describe <name>`.

---

## How it works

```
call → policy → execute
```

Every action goes through the same pipeline. Policy is checked before execution. Credentials are injected at execution time — they never enter the agent process.

| Without kimbap | With kimbap |
|---|---|
| Build an MCP server per service | One YAML manifest. No server needed |
| API keys passed via env vars — leak into logs and prompts | Encrypted vault injects credentials at execution time |
| Every service has different auth | One manifest format for REST APIs, CLI tools, and macOS apps |
| Agents run dangerous actions unchecked | Policies and approvals enforced on every action |
| No record of what the agent did | Audit trail on every action, automatically |
| API responses flood the agent's context | [Output filtering](./docs/output-filtering.md) trims responses to only what the agent needs (83–94% reduction) |

---

## Link a service

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

## Advanced

### Integration modes

| Mode | Command | Use case |
|---|---|---|
| **Call** (recommended) | `kimbap call <service>.<action>` | Direct use, scripts, agent integration |
| Run | `kimbap run -- <cmd>` | Wrap any agent subprocess |
| Proxy | `kimbap proxy` | Existing HTTP agents, zero code changes |
| Serve | `kimbap serve` | Connected-mode REST API server |

All modes share the same policy, credentials, and audit pipeline.

### Execution modes

`kimbap init --mode` controls the security profile of the local runtime:

- `dev` — relaxed security, auto-generated vault key. For local development only.
- `embedded` — policy-enforced, vault key from `KIMBAP_MASTER_KEY_HEX`. Default for production.
- `connected` — routes execution through a running `kimbap serve` REST server.

For most use cases: use `kimbap call` with `--mode embedded`.

### Production setup

```bash
export KIMBAP_MASTER_KEY_HEX="$(openssl rand -hex 32)"
kimbap init --mode embedded --services all
```

### kimbap-web (embedded web console)

```bash
kimbap serve --console --port 8080
```

Opens the operations console at `http://localhost:8080/console`. Shows audit logs, pending approvals, and service health. Disabled by default.
Action detail includes a shortcut setup snippet (`kimbap alias set <shortcut> <service>.<action>`) for frictionless command use.

### Documentation

- **[Installation Guide](./docs/installation.md)** — step-by-step setup, agent-readable
- **[CLI Reference](./docs/cli-reference.md)** — commands, flags, configuration
- **[Service Development Guide](./docs/service-development.md)** — manifest authoring, adapters
- **[Output Filtering](./docs/output-filtering.md)** — reduce LLM token usage by 83–94% with declarative response shaping
- **[Architecture & Internals](./docs/architecture.md)**
- **[Security & Permissions](./docs/security.md)**
- **[Deployment & Configuration](./docs/deployment.md)**
- **[HTTP API Reference](./docs/api/API.md)**


---

## Contributing

Write a YAML manifest, validate it, open a PR. See **[CONTRIBUTING.md](./CONTRIBUTING.md)**.

---

MIT License · [Dunia Labs](https://dunialabs.io)

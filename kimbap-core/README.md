# Kimbap CLI

*Do you like kimbap? It's delicious, healthy, and you can make your own.*

> **CLI for AI agents to use any service — fast to use, fast to build, safe by default.**

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Services](https://img.shields.io/badge/services-53-orange.svg)

[Quick Start](https://kimbap.sh/quick-start) · [Docs](https://docs.kimbap.sh) · [Website](https://kimbap.sh)

---

One CLI. 53 built-in services. Add new ones with a single YAML file.

```bash
brew install kimbap

# Store a credential
printf '%s' "$GITHUB_TOKEN" | kimbap vault set github.token --stdin

# See what's available
kimbap actions list --service github

# Use it
kimbap call github.create-issue --owner acme --repo api --title "fix auth bug"
```

---

## 53 services included

### SaaS & APIs

GitHub · Slack · Stripe · Notion · Linear · HubSpot · Airtable · Pinecone · Todoist · PostHog · Sentry · SendGrid · Resend · Exa · Brave Search

### Communication

Telegram · WhatsApp · WeChat · Zoom · Apple Mail · Messages

### Local applications

Blender · ComfyUI · Ollama · Mermaid · Spotify · NotebookLM

### macOS native

Finder · Safari · Contacts · Shortcuts · Notes · Calendar · Reminders · Keynote · Pages · Numbers

### Office suites

Microsoft Word · Excel · PowerPoint

### Data & reference

Wikipedia · Hacker News · CoinGecko · Open-Meteo (weather, air quality, historical, geocoding) · Financial Datasets · REST Countries · Exchange Rate · Public Holidays · Nominatim · ntfy

```bash
kimbap actions list                    # all services
kimbap actions list --service stripe   # one service
kimbap actions describe stripe.list-charges
```

---

## CLI modes

| Mode | Command | Use case |
|---|---|---|
| Call | `kimbap call <service>.<action>` | Direct use, scripts, agent integration |
| Run | `kimbap run -- <cmd>` | Wrap any existing agent process |
| Proxy | `kimbap proxy --port 10255` | Existing HTTP agents, zero code changes |
| Serve | `kimbap serve --port 8080` | Team deployments, multi-tenant, REST API |

All modes go through the same pipeline. Same credentials, same policy, same audit.

---

## Build your own in minutes

Three adapter types. Same CLI pipeline.

**REST API** — most SaaS integrations:

```yaml
name: stripe
version: 1.0.0
adapter: http
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

**Command** — wraps any CLI tool:

```yaml
name: blender
version: 1.0.0
adapter: command
auth:
  type: none
command_spec:
  executable: cli-anything-blender
  json_flag: "--json"
actions:
  render:
    command: "render execute"
    risk:
      level: high
```

**AppleScript** — macOS native apps:

```yaml
name: finder
version: 1.0.0
adapter: applescript
auth:
  type: none
target_app: Finder
actions:
  list-items:
    command: finder-list-items
    risk:
      level: low
```

```bash
kimbap service validate my-service.yaml
kimbap service install my-service.yaml
kimbap call my-service.my-action
```

---

## How it works

```
agent → kimbap CLI → policy → approval → credential injection → execution → audit
```

Credentials are stored in an encrypted vault and injected server-side at execution time. They never enter the agent process — not in env vars, not in prompts, not in logs.

Policy, approval, and audit apply to every action automatically, regardless of which CLI mode is used.

| Capability | What it does |
|---|---|
| **Vault** | Encrypted credential storage. `kimbap vault set`, `kimbap vault list`, `kimbap vault rotate`. |
| **Policy** | YAML rules evaluated on every action. `allow`, `deny`, or `require_approval`. |
| **Approval** | Human-in-the-loop for risky actions. `kimbap approve list`, `kimbap approve accept`. |
| **Audit** | Structured log of every action and decision. `kimbap audit tail`, `kimbap audit export`. |
| **Connectors** | OAuth lifecycle. `kimbap connector login`, `kimbap connector status`. |
| **Link** | Credential linking. `kimbap link <service>` to bind services to vault entries or OAuth connectors. |
| **Search** | Action discovery. `kimbap search <query>` to find installed actions by keyword or description. |
| **Generate** | Code generation. `kimbap generate ts` and `kimbap generate py` produce typed client interfaces. |
| **Agent sync** | Generates SKILL.md for agent discovery. `kimbap agents setup`, `kimbap agents sync`. |

---

## Getting started from source

```bash
git clone https://github.com/dunialabs/kimbap-core.git
cd kimbap-core
make deps && make build    # binary → bin/kimbap
```

```bash
cp .env.example .env
make dev     # starts dev server
make test
make lint
```

---

## Contributing

Add a service: write a YAML manifest, validate it, open a PR.

```bash
kimbap service validate my-service.yaml
kimbap service install my-service.yaml   # test locally
```

See [CONTRIBUTING.md](./CONTRIBUTING.md) for standards.

---

## Documentation

- [Architecture & Internals](./docs/architecture.md)
- [Security & Permissions](./docs/security.md)
- [Deployment & Configuration](./docs/deployment.md)
- [Reference](./docs/reference.md)

---

MIT License · [Dunia Labs](https://dunialabs.io)

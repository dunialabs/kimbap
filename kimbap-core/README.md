# Kimbap

> **Secure action runtime for AI agents**
>
> Agents execute actions. Kimbap owns credentials, policy, approvals, and audit.

![Go](https://img.shields.io/badge/go-%3E%3D1.24-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-blue.svg)

**Quick Start** → [https://kimbap.sh/quick-start](https://kimbap.sh/quick-start) | **Docs** → [https://docs.kimbap.sh](https://docs.kimbap.sh) | **Website** → [https://kimbap.sh](https://kimbap.sh)

---

Kimbap is the runtime boundary between agents and external systems.

```text
agent -> kimbap runtime -> policy -> approval -> credential resolution -> execution -> audit
```

- **service**: declarative action manifest (YAML)
- **connector**: OAuth connection managed by Kimbap

---

## Product strengths

1. **Credential isolation by default**  
   Vault secrets, connector tokens, and other server-side credential material stay on the Kimbap side. Agents receive action inputs/outputs, not raw credentials.

2. **Runtime guardrails, not prompt-only rules**  
   Policy evaluation, optional approval hold, input validation/sanitization, and credential resolution run in the execution pipeline.

3. **Full execution traceability**  
   Every action execution can be audited with decision metadata, status, and timing.

4. **One runtime, multiple integration surfaces**  
   Use the same runtime through explicit action calls, subprocess wrapping, proxy mode, or connected API server mode.

5. **Local-first default operation**  
   Embedded mode defaults to local SQLite, JSONL audit logs, and filesystem-managed services under `~/.kimbap`.

6. **Declarative integration model**  
   Most API integrations are service manifests (`service install/validate/verify/sign/generate`) instead of custom per-service code.

7. **Manifest integrity controls**  
   Installed services are lockfile-backed and support verification/signing policy checks.

8. **Agent onboarding automation**  
   Kimbap can sync generated `SKILL.md` files and operating rules into detected agent directories.

---

## `SKILL.md` writing vs Kimbap CLI

`SKILL.md` is a discovery format. Kimbap CLI/API is the runtime path that applies policy, approval, and credential controls.

| Concern | Writing `SKILL.md` manually | Using Kimbap via CLI |
|---|---|---|
| Source of truth | Static markdown in agent config | Installed service manifests + runtime action catalog |
| Invocation | Agent reads instructions and chooses a path | Explicit runtime call (`kimbap call <service>.<action>`) |
| Security controls | No hard enforcement by itself | Policy, approval, credential resolution enforced at runtime |
| Data safety | Easy to drift into direct credential handling | Kimbap-managed credentials stay outside agent context |
| Change management | Manual updates per agent/project | `kimbap agents sync` regenerates from installed services |
| Auditability | No built-in execution audit | Structured audit events and export |
| Operator workflow | Manual file upkeep per agent/project | CLI/API operations (`actions`, `call`, `service`, `vault`, `policy`, `approve`, `audit`) |

Practical model:
- Use `SKILL.md` as **agent-facing hints**.
- Use Kimbap CLI/API as **execution and control plane**.

---

## Core workflow

```bash
# 1) Validate and install service manifest
kimbap service validate slack.yaml
kimbap service install slack.yaml

# 2) Discover and inspect actions
kimbap actions list --format json
kimbap actions describe slack.post_message --format json

# 3) Dry-run before execution
kimbap call slack.post_message --dry-run --format json --channel C123 --text "hello"

# 4) Execute with runtime enforcement
kimbap call slack.post_message --channel C123 --text "hello"
```

---

## Usage surfaces

```bash
# Explicit action execution
kimbap call github.list_pull_requests --repo owner/repo

# Wrap an existing agent process
kimbap run -- python agent.py

# Transparent HTTP/HTTPS proxy mode
kimbap proxy --port 10255

# Connected REST server mode
kimbap serve --port 8080
```

---

## Current capabilities

| Capability | Status | Notes |
|---|---|---|
| Action runtime pipeline | Available | Auth, policy, approval, credentials, adapter execution, audit |
| Service manifests | Available | Install, validate, verify, sign, generate from OpenAPI |
| Action discovery & execution | Available | `actions list/describe`, `call`, dry-run, trace |
| Vault | Available | Encrypted secret storage, rotate/delete/list/get |
| Policy engine | Available | YAML policy load/get/eval |
| Approval workflow | Available | Request listing + accept/deny operations |
| Audit tooling | Available | Tail and export |
| OAuth connectors | Available | Login/list/status/refresh workflows |
| Agent sync | Available | Setup/sync/status, generated `SKILL.md` output |
| Proxy / subprocess / server modes | Available | `proxy`, `run`, `serve` |
| REST v1 API | Available | Actions, tokens, policies, approvals, audit |

---

## Agent onboarding

```bash
# Install global discovery hints for detected agents
kimbap agents setup

# Sync installed services to project agent directories
kimbap agents sync

# Check synchronization status
kimbap agents status

# Export one service as SKILL.md
kimbap service export-skillmd slack --dir ./.opencode/skills
```

---

## Getting started

### Prerequisites

- Go 1.24+
- PostgreSQL 15+ (for connected/server deployments)
- Docker (optional)

### Local development

```bash
git clone https://github.com/dunialabs/kimbap-core.git
cd kimbap-core
make deps
cp .env.example .env
docker compose up -d
make dev
```

### Common commands

```bash
make dev
make build
make run
make test
make lint
```

---

## API surfaces

| Runtime entrypoint | Path surface | Notes |
|---|---|---|
| `kimbap serve` | `/v1/*`, `/v1/health` | Connected-mode API server |
| `cmd/server` (`make dev`) | `/api/v1/*`, `/health`, `/ready` | Core server runtime |
| `cmd/server` legacy surfaces | `/admin`, `/user` | Legacy (frozen) endpoints |

---

## More documentation

- [**CLAUDE.md**](./CLAUDE.md): architecture and development guidance
- [**AGENTS.md**](./AGENTS.md): Codex workflow and repo knowledge index
- [**CONTRIBUTING.md**](./CONTRIBUTING.md): contribution standards and verification
- [**Architecture & Internals**](./docs/architecture.md)
- [**Security & Permissions**](./docs/security.md)
- [**Deployment & Configuration**](./docs/deployment.md)
- [**Reference**](./docs/reference.md)

---

## License

MIT License. Copyright © 2026 [Dunia Labs, Inc.](https://dunialabs.io)

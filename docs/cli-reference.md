# Kimbap CLI Reference

Complete reference for the `kimbap` CLI.

---

## Action execution

### kimbap call \<service\>.\<action\>

Execute a registered action.

Flags are derived from the action's input schema. Each flag corresponds to an input field defined in the service manifest. Required fields without defaults must be provided.

**Syntax:**

```
kimbap call <service>.<action> [--<arg> <value> ...]
```

**Examples:**

```bash
kimbap call github.list-repos --owner octocat
kimbap call stripe.list-charges --limit 10
kimbap call slack.post-message --channel C123 --text "hello"
kimbap call github.create-issue --owner acme --repo api --title "fix bug" --body "details"
```

---

### kimbap search \<query\>

Search installed actions by keyword or description. Matches against action names, descriptions, and tags.

**Syntax:**

```
kimbap search <query>
```

**Examples:**

```bash
kimbap search "send message"
kimbap search "list repos"
kimbap search refund
```

---

## Action discovery

### kimbap actions list

List all installed actions across all services.

**Flags:**

| Flag | Description |
|---|---|
| `--service <name>` | Filter results to a single service |

**Examples:**

```bash
kimbap actions list
kimbap actions list --service github
kimbap actions list --service stripe
```

---

### kimbap actions describe \<service\>.\<action\>

Show the full schema for an action, including input field types, required fields, auth requirements, risk level, and usage examples embedded in the manifest.

**Syntax:**

```
kimbap actions describe <service>.<action>
```

**Example:**

```bash
kimbap actions describe stripe.list-charges
kimbap actions describe github.create-issue
```

---

## Service management

### kimbap service install \<file|name\>

Install a service manifest. Accepts a path to a local YAML file or a built-in service name from the official registry.

**Syntax:**

```
kimbap service install <file|name>
```

**Examples:**

```bash
kimbap service install my-service.yaml
kimbap service install github
kimbap service install stripe
```

---

### kimbap service validate \<file\>

Parse and validate a service manifest without installing it. Reports all errors found, not just the first.

Catches:

- Schema errors and unknown fields
- Invalid or missing `base_url`
- Missing required top-level fields
- Duplicate argument names within an action
- Type mismatches in argument definitions
- Invalid risk level values

**Syntax:**

```
kimbap service validate <file>
```

**Example:**

```bash
kimbap service validate my-service.yaml
```

---

### kimbap service list

List all installed services, their versions, and their action counts.

**Example:**

```bash
kimbap service list
```

---

### kimbap service export-agent-skill \<name\>

Export a `SKILL.md` file for a given service to stdout. This file is formatted for AI agent consumption and describes available actions, input schemas, and usage examples.

**Syntax:**

```
kimbap service export-agent-skill <name>
```

**Example:**

```bash
kimbap service export-agent-skill github
kimbap service export-agent-skill stripe > ./skills/stripe.md
```

---

## Credential management

### kimbap vault set \<key\>

Store a secret in the encrypted vault. Secrets are never accepted as inline CLI arguments to avoid shell history exposure.

**Flags:**

| Flag | Description |
|---|---|
| `--stdin` | Read the secret value from stdin |
| `--file <path>` | Read the secret value from a file |

**Examples:**

```bash
printf '%s' "$TOKEN" | kimbap vault set github.token --stdin
kimbap vault set stripe.api_key --file ./key.txt
```

---

### kimbap vault list

List vault key metadata: names, last-used timestamps, and last-rotated timestamps. Secret values are never shown.

**Example:**

```bash
kimbap vault list
```

---

### kimbap vault rotate

Rotate a stored credential. Prompts for the new value via stdin or `--file`.

**Syntax:**

```
kimbap vault rotate <key>
```

**Example:**

```bash
printf '%s' "$NEW_TOKEN" | kimbap vault rotate github.token --stdin
```

---

### kimbap link \<service\>

Link a service to vault credentials or an OAuth connector. Kimbap will look up the correct vault keys or connector tokens when executing actions for that service.

**Syntax:**

```
kimbap link <service>
```

**Examples:**

```bash
kimbap link github
kimbap link stripe
```

---

## OAuth connectors

### kimbap connector login \<provider\>

Start an OAuth device flow or browser-based flow for the given provider. On completion, the token is stored in the vault and linked to the provider.

**Syntax:**

```
kimbap connector login <provider>
```

**Examples:**

```bash
kimbap connector login notion
kimbap connector login slack
```

---

### kimbap connector list

Show connector health and token state for all linked providers. Includes expiry status and whether the token is valid.

**Example:**

```bash
kimbap connector list
```

---

### kimbap connector status \<provider\>

Show connector health and token state for a single provider.

**Syntax:**

```
kimbap connector status <provider>
```

**Example:**

```bash
kimbap connector status notion
kimbap connector status slack
```

---

### kimbap auth providers list

List all bundled OAuth providers available in the built-in registry.

**Example:**

```bash
kimbap auth providers list
```

---

### kimbap auth connect \<provider\>

Authenticate with an OAuth provider. Starts the appropriate flow (device code or browser redirect) and persists the resulting token.

**Syntax:**

```
kimbap auth connect <provider>
```

**Example:**

```bash
kimbap auth connect github
kimbap auth connect google
```

---

### kimbap auth revoke \<provider\>

Revoke an active OAuth session and remove the stored token for that provider.

**Syntax:**

```
kimbap auth revoke <provider>
```

**Example:**

```bash
kimbap auth revoke github
kimbap auth revoke slack
```

---

## Policy

### kimbap policy set --file \<path\>

Load a policy document from a YAML file. Replaces the active policy entirely.

**Syntax:**

```
kimbap policy set --file <path>
```

**Example:**

```bash
kimbap policy set --file ./policy.yaml
```

**Example policy document:**

```yaml
version: "1.0.0"
rules:
  - id: allow-github-read
    priority: 10
    match:
      agents: [repo-bot]
      actions: [github.list-repos, github.list-pull-requests]
    decision: allow

  - id: approve-stripe-refunds
    priority: 20
    match:
      agents: [billing-bot]
      actions: [stripe.create-refund]
    decision: require_approval

  - id: deny-destructive
    priority: 5
    match:
      risk: critical
    decision: deny
```

**Policy decisions:**

| Decision | Behavior |
|---|---|
| `allow` | Action proceeds immediately |
| `deny` | Action is blocked and returns an error |
| `require_approval` | Action is held until a human approves it |

**Match fields:**

| Field | Description |
|---|---|
| `agents` | List of agent identifiers. Supports glob patterns. |
| `services` | List of service names. Supports glob patterns. |
| `actions` | List of `service.action` identifiers. Supports glob patterns. |
| `risk` | Risk level: `low`, `medium`, `high`, or `critical` |

Rules are evaluated in ascending `priority` order. The first matching rule wins.

---

### kimbap policy get

Print the active policy document to stdout.

**Example:**

```bash
kimbap policy get
```

---

## Approval

### kimbap approve list

List all pending approval requests with their IDs, action names, requesting agents, and creation times.

**Example:**

```bash
kimbap approve list
```

---

### kimbap approve accept \<id\>

Approve a held action, allowing it to proceed.

Expired approvals and already-resolved approvals return distinct error codes so callers can distinguish between the two cases.

**Syntax:**

```
kimbap approve accept <id>
```

**Example:**

```bash
kimbap approve accept req_01HX...
```

---

## Audit

### kimbap audit tail

Stream recent audit log entries to stdout. Error messages in audit records are capped at 256 characters.

**Example:**

```bash
kimbap audit tail
```

---

### kimbap audit export

Export all audit records as newline-delimited JSON.

**Example:**

```bash
kimbap audit export
kimbap audit export > audit-$(date +%Y%m%d).jsonl
```

---

## Runtime modes

### kimbap run -- \<cmd\>

Run an agent subprocess inside a Kimbap-controlled environment. The subprocess gets no direct access to vault secrets. Credentials are injected server-side by the proxy or daemon, not passed through environment variables or files.

**Syntax:**

```
kimbap run -- <command> [args...]
```

**Example:**

```bash
kimbap run -- python agent.py
kimbap run -- node ./agent/index.js
```

---

### kimbap proxy

Start an HTTP/HTTPS proxy that intercepts outbound requests, classifies them against installed action patterns, and injects credentials server-side before forwarding.

Route matching uses specificity-based priority: exact paths take precedence over parameterized paths, which take precedence over wildcard patterns.

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `--port` | `10255` | Port to listen on |

**Example:**

```bash
kimbap proxy --port 10255
```

Then in the agent's environment:

```bash
export HTTPS_PROXY=http://127.0.0.1:10255
python agent.py
```

---

### kimbap serve

Start a persistent HTTP server that exposes the action runtime over a local HTTP API.

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `--port` | `8080` | Port to listen on |
| `--console` | disabled | Enable the embedded operations console at `/console` |

The `/console` route is disabled by default. Enable it via `--console` flag or by setting `console.enabled: true` in `~/.kimbap/config.yaml`. Do not expose this endpoint on a network interface in production.

**Examples:**

```bash
kimbap serve
kimbap serve --port 9000 --console
```

---

### kimbap daemon

Start the background job runner. Handles token refresh, scheduled credential rotation, and other async tasks. Communicates via a Unix domain socket.

**Example:**

```bash
kimbap daemon
```

---

## Agent configuration

### kimbap agents setup

Install global Kimbap discovery hints for detected AI agent environments. Writes marker files that tell AI agents this workspace has Kimbap installed.

**Example:**

```bash
kimbap agents setup
```

---

### kimbap agents sync

Sync installed services to detected agent skill directories. Generates a `SKILL.md` file per service in each discovered agent directory.

**Example:**

```bash
kimbap agents sync
```

---

### kimbap agents status

Show the sync status for all known AI agent environments: which agents were detected, which skill files exist, and which are stale.

**Example:**

```bash
kimbap agents status
```

---

### kimbap profile install \<profile\>

Install an agent operating profile. Writes a `KIMBAP_OPERATING_RULES.md` file into the agent's config directory.

**Available profiles:**

| Profile | Install location |
|---|---|
| `claude-code` | `.claude/KIMBAP_OPERATING_RULES.md` |
| `opencode` | `.opencode/KIMBAP_OPERATING_RULES.md` |
| `cursor` | `.cursor/KIMBAP_OPERATING_RULES.md` |
| `codex` | `.codex/KIMBAP_OPERATING_RULES.md` |
| `generic` | `.agents/KIMBAP_OPERATING_RULES.md` |

**Syntax:**

```
kimbap profile install <profile>
```

**Example:**

```bash
kimbap profile install claude-code
kimbap profile install opencode
```

---

### kimbap profile list

List all available agent profiles.

**Example:**

```bash
kimbap profile list
```

---

### kimbap profile print \<profile\>

Print the contents of an agent profile to stdout without installing it.

**Syntax:**

```
kimbap profile print <profile>
```

**Example:**

```bash
kimbap profile print claude-code
kimbap profile print generic > ./rules.md
```

---

## Code generation

### kimbap generate ts

Generate TypeScript input interfaces for installed actions.

**Flags:**

| Flag | Description |
|---|---|
| `--service <name>` | Limit output to a single service |
| `-o <path>` | Write output to a file instead of stdout |

**Examples:**

```bash
kimbap generate ts
kimbap generate ts --service github -o ./types/github.ts
kimbap generate ts --service stripe -o ./types/stripe.ts
```

---

### kimbap generate py

Generate Python `TypedDict` input types for installed actions.

**Flags:**

| Flag | Description |
|---|---|
| `--service <name>` | Limit output to a single service |
| `-o <path>` | Write output to a file instead of stdout |

**Examples:**

```bash
kimbap generate py
kimbap generate py --service stripe -o ./types/stripe.py
kimbap generate py --service github -o ./types/github.py
```

---

## Setup and diagnostics

### kimbap init

Bootstrap a fresh Kimbap installation. Creates the data directory, generates a vault master key (if not already set), and optionally installs services.

**Flags:**

| Flag | Description |
|---|---|
| `--services <list>` | Comma-separated official service names to install, or `all` |
| `--no-services` | Skip service installation entirely |
| `--with-console` | Enable the `/console` route in the generated config |
| `--with-agents` | Run `agents setup` and `agents sync` after init |
| `--agents-project-dir <path>` | Project directory to target for agent sync |
| `--force` | Overwrite an existing config file |

**Examples:**

```bash
kimbap init
kimbap init --services all --with-agents --agents-project-dir .
kimbap init --services github,stripe --force
kimbap init --no-services
```

---

### kimbap doctor

Run environment diagnostics. Checks vault status, proxy CA certificate trust, network connectivity, and configuration validity. Reports pass/fail for each check with actionable error messages.

**Syntax:**

```
kimbap doctor [check]
```

**Examples:**

```bash
kimbap doctor
kimbap doctor proxy
```

---

## Configuration reference

Configuration is read from environment variables or `~/.kimbap/config.yaml`. Environment variables take precedence over the config file.

| Variable | Default | Description |
|---|---|---|
| `KIMBAP_DATA_DIR` | `~/.kimbap` | Data directory for the database, vault, and audit logs |
| `KIMBAP_MASTER_KEY_HEX` | auto-generated in dev mode | Hex-encoded 32-byte vault master key |
| `KIMBAP_DEV` | `false` | Dev mode: relaxed security checks and verbose logging |
| `KIMBAP_DATABASE_DRIVER` | `sqlite` | Database driver: `sqlite` or `postgres` |
| `KIMBAP_DATABASE_DSN` | `$KIMBAP_DATA_DIR/kimbap.db` | Database connection string |
| `KIMBAP_LOG_LEVEL` | `info` | Log verbosity: `trace`, `debug`, `info`, `warn`, `error` |

---

## Risk levels

Risk levels are declared per-action in service manifests. Policy rules can match on risk level to apply blanket decisions across action categories.

| Level | Meaning | Default behavior |
|---|---|---|
| `low` | Read-only, no side effects, fully reversible | Auto-approved |
| `medium` | Writes data, limited blast radius | Auto-approved |
| `high` | Destructive or financially significant | Policy-dependent |
| `critical` | Irreversible or high blast radius | Requires explicit approval |

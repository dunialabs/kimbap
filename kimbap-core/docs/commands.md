# Kimbap CLI Command Reference

Complete reference for all `kimbap` CLI commands.

---

## Action execution

### `kimbap call <service>.<action>`

Execute a registered action.

```bash
kimbap call github.list-repos --owner octocat
kimbap call stripe.list-charges --limit 10
kimbap call slack.post-message --channel C123 --text "hello"
```

Flags are derived from the action's input schema. Use `kimbap actions describe` to see available arguments.

### `kimbap search <query>`

Search installed actions by keyword or description.

```bash
kimbap search "send message"
kimbap search "list repos"
```

---

## Action discovery

### `kimbap actions list`

List all installed actions.

```bash
kimbap actions list                      # all services
kimbap actions list --service github     # one service
```

### `kimbap actions describe <service>.<action>`

Show schema, auth requirements, risk level, and examples for an action.

```bash
kimbap actions describe stripe.list-charges
kimbap actions describe github.create-issue
```

---

## Service management

### `kimbap service install <file|name>`

Install a service manifest.

```bash
kimbap service install my-service.yaml
kimbap service install github
```

### `kimbap service validate <file>`

Validate a service manifest against the strict parser. Use before installing or submitting a PR.

```bash
kimbap service validate my-service.yaml
```

### `kimbap service list`

List all installed services and their action counts.

### `kimbap service export-agent-skill`

Export a SKILL.md file for agent discovery.

---

## Credential management

### `kimbap vault set <key>`

Store a secret in the encrypted vault.

```bash
printf '%s' "$TOKEN" | kimbap vault set github.token --stdin
kimbap vault set stripe.api_key --file ./key.txt
```

Secrets are never accepted as inline CLI arguments.

### `kimbap vault list`

List vault key metadata (names, last used, last rotated). Values are never shown.

### `kimbap vault rotate`

Rotate a stored credential.

### `kimbap link <service>`

Link a service to vault credentials or an OAuth connector.

```bash
kimbap link github
kimbap link stripe
```

---

## OAuth connectors

### `kimbap connector login <provider>`

Start an OAuth device/browser flow for a downstream provider.

```bash
kimbap connector login gmail
kimbap connector login slack
```

### `kimbap connector status`

Show connector health and token state for all linked providers.

### `kimbap auth providers list`

List all bundled OAuth providers available in the registry.

### `kimbap auth connect <provider>`

Authenticate with an OAuth provider.

### `kimbap auth revoke <provider>`

Revoke an OAuth session.

---

## Policy

### `kimbap policy set --file <path>`

Load a policy document.

```bash
kimbap policy set --file policy.yaml
```

Example policy:

```yaml
version: 1
defaults:
  mode: deny
rules:
  - id: allow-github-read
    match:
      agent: repo-bot
      actions:
        - github.list-repos
        - github.list-pull-requests
    effect: allow
  - id: approve-stripe-refunds
    match:
      agent: billing-bot
      actions:
        - stripe.create-refund
    effect: require_approval
```

### `kimbap policy get`

Show the active policy document.

---

## Approval

### `kimbap approve list`

List pending approval requests.

### `kimbap approve accept <id>`

Approve a held action.

```bash
kimbap approve list
kimbap approve accept req_01HX...
```

---

## Audit

### `kimbap audit tail`

Stream recent audit entries.

### `kimbap audit export`

Export audit records (JSON).

---

## Token management

### `kimbap token create`

Issue a new access token for an agent.

```bash
kimbap token create --agent billing-bot --scopes actions:execute
```

### `kimbap token list`

List active tokens with metadata.

### `kimbap token revoke <id>`

Revoke a token.

---

## Server and runtime modes

### `kimbap serve [--port 8080]`

Start the connected-mode REST API server.

```bash
kimbap serve
kimbap serve --port 9090
```

### `kimbap run -- <cmd>`

Run an agent subprocess inside a Kimbap-controlled environment. Credentials are never exposed to the child process.

```bash
kimbap run -- python agent.py
kimbap run --token <service-token> -- node bot.js
```

### `kimbap proxy [--port 10255]`

Start an HTTP/HTTPS proxy that intercepts outbound requests, classifies them into actions, and injects credentials server-side.

```bash
kimbap proxy --port 10255
export HTTPS_PROXY=http://127.0.0.1:10255
python agent.py
```

### `kimbap daemon`

Start the background job runner for token refresh and scheduled tasks.

---

## Agent configuration

### `kimbap agents setup`

Install global kimbap discovery hints for detected AI agents.

### `kimbap agents sync`

Sync installed services to detected agent skill directories. Generates SKILL.md per service.

### `kimbap agents status`

Show sync status for known AI agents.

### `kimbap agent-profile install <profile>`

Install an agent operating profile for a specific agent framework.

```bash
kimbap agent-profile install claude-code
kimbap agent-profile install generic
```

### `kimbap agent-profile list`

List installed agent profiles.

### `kimbap agent-profile print <profile>`

Print an agent profile to stdout.

---

## Code generation

### `kimbap generate ts`

Generate TypeScript input interfaces for installed actions.

```bash
kimbap generate ts --service github -o ./types/github.ts
```

### `kimbap generate py`

Generate Python TypedDict inputs for installed actions.

```bash
kimbap generate py --service stripe -o ./types/stripe.py
```

---

## Setup and diagnostics

### `kimbap init`

Initialize a new Kimbap workspace in the current directory.

### `kimbap doctor`

Run environment diagnostics — checks vault status, proxy CA trust, connectivity, and configuration.

```bash
kimbap doctor
kimbap doctor proxy
```

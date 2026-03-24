# Kimbap

> **Secure action runtime for AI agents**
>
> Let agents use GitHub, Gmail, Stripe, Notion, internal APIs, and existing CLIs **without handing them raw credentials**.

Kimbap is the layer between **AI agents** and **external systems**.

Instead of giving an agent API keys, OAuth tokens, or a human CLI session, you give it access to **Kimbap**.  
Kimbap handles identity, policy, credential injection, OAuth refresh, approvals, and audit. The agent only sees **actions** and **results**.

---

## Why Kimbap

AI agents are starting to do real work:

- read and write GitHub issues
- send email
- update CRMs
- create docs
- refund payments
- run infra workflows
- call internal APIs

Most teams solve this in one of four bad ways:

1. **Raw API keys in env vars**  
   Fast, but unsafe. Secrets leak into prompts, logs, traces, or shell history.

2. **Human-oriented service CLIs**  
   Useful for developers, but not safe trust boundaries for autonomous agents.

3. **Vault only**  
   A vault stores secrets, but does not execute actions, refresh OAuth tokens, enforce policy, or manage approvals.

4. **Custom wrapper per service**  
   Secure-ish, but expensive. Every integration becomes another codebase to build and maintain.

Kimbap replaces that mess with one runtime.

---

## What it looks like

```text
agent -> kimbap -> policy -> approval -> credentials -> execution -> audit
```

The agent does not need to know:

- how GitHub auth works
- how Gmail refreshes tokens
- whether Stripe refunds need approval
- where secrets live
- whether an action is backed by REST, OAuth, or an existing CLI

It only needs to know:

```bash
kimbap call <service>.<action>
```

Examples:

```bash
kimbap call github.list_pull_requests --repo owner/repo
kimbap call notion.create_page --database-id db_123 --title "Q2 Plan"
kimbap call stripe.refund_charge --charge ch_abc --amount 500 --idempotency-key refund-001
```

---

## Core idea

**Action Runtime is the product.**

Kimbap is not just:

- a vault
- a proxy
- a CLI wrapper
- an MCP server
- an integration zoo

Those are surfaces and adapters.

The actual product is a **canonical action runtime** that every surface goes through:

```text
identity -> action lookup -> policy -> approval -> credential injection -> execution -> audit
```

That gives you:

- one security model
- one policy layer
- one audit trail
- one integration model
- one path from local dev to multi-tenant deployment

---

## Product surfaces

Kimbap ships as a **single Go binary**.

### Explicit action mode

For new agent systems and direct integrations:

```bash
kimbap call github.list_pull_requests --repo owner/repo
```

### Proxy mode

For existing HTTP-based agents with minimal or zero code changes:

```bash
kimbap proxy --port 10255
```

### Subprocess mode

For local agents, scripts, or agent frameworks that already run as a process:

```bash
kimbap run -- python agent.py
```

### Connected server mode

For shared deployments, long-running connectors, centralized audit, and multi-tenant use:

```bash
kimbap serve --port 8080
```

### Management commands

```bash
kimbap token ...
kimbap vault ...
kimbap policy ...
kimbap connector ...
kimbap audit ...
kimbap skill ...
```

One binary. One install. One runtime model.

---

## How integrations work

Kimbap does **not** want a bespoke codebase for every service.

### Tier 1 — Declarative skills

Most modern APIs can be exposed through a human-readable YAML skill:

- auth type
- endpoint shape
- arguments
- error mapping
- pagination
- output extraction
- risk metadata

That turns a REST API into a first-class Kimbap action.

### Tier 2 — Connectors

Some services need OAuth device flow, token refresh, or provider-specific lifecycle handling.

Examples:

- Gmail
- Google Workspace
- Slack
- HubSpot
- GitHub Apps
- Stripe Connect

These use runtime-owned connectors.

### Tier 2b — Existing CLI adapters

Some mature CLIs are useful, but only if Kimbap wraps them safely.  
The CLI is **never** the trust boundary. Kimbap is the trust boundary.

### Tier 3 — Justified stateful services

Only truly stateful or streaming systems should require a separate downstream service.

Examples:

- pooled database access
- non-HTTP protocols
- long-lived stateful connections

---

## Security model

Kimbap’s core promise is simple:

> **Raw credentials should not appear in agent-visible env vars, prompts, logs, CLI history, or persisted traces.**

### What Kimbap guarantees

- agents do not receive long-lived raw credentials
- OAuth refresh tokens stay under Kimbap control
- policy is enforced before execution
- risky actions can require human approval
- every action is auditable
- tenant boundaries are enforced by namespace, policy, and key hierarchy

### What Kimbap does not claim

- that the final outbound HTTP request is credential-free
- that proxy mode works for every protocol
- that arbitrary existing CLIs become safe without an adapter
- that host compromise is out of scope

The product wins by being precise and enforceable.

---

## Multi-tenant by design

Kimbap is built for teams running multiple agents, workflows, and customers.

Tenant isolation is enforced in three layers:

1. **Namespace isolation**  
   Vault entries, policy, approvals, and audit are tenant-scoped.

2. **Policy isolation**  
   Every decision is evaluated in tenant context.

3. **Cryptographic isolation**  
   Each tenant has its own key hierarchy.

That means token rotation and key rotation are independent.

---

## Human approval built in

Some actions should not run automatically.

Kimbap makes approval a first-class runtime feature for actions like:

- refunds
- deletes
- external messages
- production changes
- high-risk internal operations

Typical flow:

1. Agent requests a risky action
2. Policy marks it `require_approval`
3. Kimbap creates an approval record
4. Operator approves in console, CLI, or messaging adapter
5. Kimbap executes or rejects
6. Audit records the full decision path

---

## Quick examples

### New code

```bash
kimbap actions list --service github
kimbap actions describe github.list_pull_requests
kimbap call github.list_pull_requests --repo owner/repo
```

### Existing subprocess agent

```bash
kimbap run -- python agent.py
```

### Existing HTTP agent

```bash
kimbap proxy --port 10255
export HTTP_PROXY=http://127.0.0.1:10255
export HTTPS_PROXY=http://127.0.0.1:10255
python agent.py
```

### OAuth-backed service

```bash
kimbap connector login gmail
kimbap call gmail.send_message --to user@example.com --subject "Hello"
```

---

## Why this is different

### Not just a vault

Kimbap stores credentials, but more importantly it **uses** them safely.

### Not just a proxy

Proxy mode is an adoption path, not the product boundary.

### Not just another integration framework

Most services should be declarative skills, not custom runtimes.

### Built for agents, not humans

Existing CLIs assume a human operator. Kimbap assumes:

- agents are untrusted by default
- credentials must stay outside agent memory
- policy and approval are first-class
- audit must be complete
- multi-tenant isolation matters

---

## Typical use cases

- secure GitHub access for coding agents
- Gmail / calendar access for workflow agents
- Stripe and billing actions behind approval
- internal admin tools exposed as governed actions
- multi-agent teams with shared runtime and isolated tenants
- gradual migration from env-var secrets to secure runtime mediation

---

## Status

Kimbap is being designed as a **CLI-first secure runtime**, reoriented for subprocess agents, local agents, and connected deployments.

Initial focus:

- action runtime
- vault and token lifecycle
- Tier 1 skill execution
- proxy and subprocess onboarding
- OAuth connectors
- policy and approval
- audit and observability

---

## Roadmap

### Phase 1
- core action runtime
- vault
- tokens
- `kimbap call`
- Tier 1 skills
- embedded mode

### Phase 2
- `kimbap proxy`
- `kimbap run`
- `kimbap serve`
- OAuth connectors
- multi-tenant connected mode
- policy evaluation

### Phase 3
- approval queue
- console
- messaging adapters
- skill install and registry
- provenance and verification

### Phase 4
- SDKs
- optional MCP compatibility
- enterprise identity
- advanced registry and policy features

---

## Design principles

- **Action Runtime is canonical**
- **One binary, clean internals**
- **No raw credentials in agent space**
- **Existing CLIs are adapters, not trust boundaries**
- **Most integrations should be declarative**
- **Adoption should be incremental**
- **Governance should be built in, not bolted on**

---

## FAQ

### Is Kimbap just a vault?
No. A vault stores secrets. Kimbap uses them safely to execute governed actions.

### Is Kimbap just a proxy?
No. Proxy mode is one adapter. The core is the action runtime.

### Do services need official CLIs?
No. If a service has a usable REST API, Kimbap should usually expose it as actions through a skill.

### How do agents learn to use Kimbap?
Through an official Kimbap operating skill/profile plus runtime enforcement such as `kimbap run`, proxying, and policy.

---

## Vision

AI agents will not become useful in production because they can generate better text alone.

They become useful when they can **take action safely** across real systems.

Kimbap is the runtime for that layer.
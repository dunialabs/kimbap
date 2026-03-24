# Kimbap

**Secure action runtime for AI agents**

Kimbap lets AI agents use external services **without ever handling raw credentials directly**.

Instead of giving agents API keys, OAuth tokens, or human CLI sessions, Kimbap sits between agents and external systems. It handles identity, policy, credential injection, OAuth refresh, approvals, and audit in one runtime.

Think of it as the missing execution layer between **AI agents** and **the real world**.

---

## One-line pitch

**Kimbap is a secure action runtime that turns APIs, existing CLIs, and OAuth-backed services into policy-governed agent actions.**

---

## Why this matters

AI agents are starting to do real work:

- reading and writing GitHub issues,
- sending email,
- updating CRMs,
- creating docs,
- refunding payments,
- running infra workflows,
- touching internal tools and private APIs.

Today, most teams give those agents one of three bad options:

1. **Raw API keys in env vars**  
   Easy, but unsafe. Keys leak into prompts, logs, shell history, and agent memory.

2. **Human-oriented service CLIs**  
   Useful for developers, but not for untrusted autonomous processes. They assume shared login state, local config files, and weak isolation.

3. **Custom per-service wrappers or servers**  
   Secure-ish, but expensive. Every new integration becomes another codebase to build, host, and maintain.

That model does not scale.

What teams actually need is a **single runtime** that can safely mediate all agent access to external systems.

That is Kimbap.

---

## What Kimbap does

Kimbap turns external service usage into a single, consistent abstraction:

```text
agent -> kimbap -> policy -> credentials -> execution -> audit
```

An agent does not need to know how GitHub auth works, how Gmail refreshes tokens, whether Stripe refunds need approval, or where secrets live.

It only needs to know:

```bash
kimbap call <service>.<action>
```

Examples:

```bash
kimbap call github.list_pull_requests --repo owner/repo
kimbap call notion.create_page --database-id db_123 --title "Q2 Plan"
kimbap call stripe.refund_charge --charge ch_abc --amount 500
```

Kimbap handles the rest.

---

## The core insight

**Action Runtime is the product.**

CLI, proxy, subprocess wrappers, and REST are just delivery surfaces into the same runtime.

That matters because most current tools are fragments:

- vaults store secrets but do not execute actions,
- proxies inject credentials but do not manage lifecycle or approvals,
- MCP gateways solve a different interface problem,
- service CLIs are not safe trust boundaries for agents.

Kimbap unifies those pieces into one runtime with one security model.

---

## What makes Kimbap different

### 1. Not just a vault
Kimbap stores credentials, but more importantly it **uses** them safely.  
It is not a password manager for agents. It is an action executor.

### 2. Not just a proxy
Kimbap supports proxy mode for zero-code-change adoption, but proxying is only one adapter.  
The canonical model is always the same: resolve action -> evaluate policy -> inject credential -> execute -> audit.

### 3. Not just another integration zoo
Most services should not require custom servers.  
If an API is request/response and auth is straightforward, Kimbap should expose it through a declarative skill.

### 4. Built for agents, not humans
Existing CLIs assume a human operator. Kimbap assumes:
- agents are untrusted by default,
- credentials must stay outside agent memory,
- approvals and policy are first-class,
- audit must be complete,
- multi-tenant isolation matters.

---

## Product surfaces

Kimbap ships as a **single Go binary** with multiple modes.

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
For agent frameworks or tools that already run as local processes:

```bash
kimbap run -- python agent.py
```

### Connected server mode
For team deployments, multi-tenant usage, centralized audit, and long-running connectors:

```bash
kimbap serve --port 8080
```

### Management commands
Kimbap also includes:
- `kimbap token ...`
- `kimbap vault ...`
- `kimbap policy ...`
- `kimbap connector ...`
- `kimbap audit ...`
- `kimbap skill ...`

One binary, one install, one runtime model.

---

## How service integrations work

Kimbap does **not** want a bespoke codebase for every service.

Instead, it uses a tiered model.

### Tier 1 — Declarative skills
Most modern APIs can be described in YAML:

- endpoint shape,
- auth type,
- arguments,
- error mapping,
- pagination,
- output extraction,
- risk metadata.

That makes a REST API instantly available as a Kimbap action.

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
The CLI is never the trust boundary. Kimbap remains the trust boundary.

### Tier 3 — Justified stateful services
Only truly stateful or streaming systems should require a separate downstream service.

Examples:
- pooled database access,
- non-HTTP protocols,
- long-lived stateful connections.

This keeps the default path lightweight and scalable.

---

## Security model

Kimbap’s core promise is simple:

**Raw credentials should not appear in agent-visible env vars, prompts, logs, CLI history, or persisted traces.**

### What Kimbap guarantees
- agents do not receive long-lived raw credentials,
- OAuth refresh tokens remain under Kimbap control,
- policy is enforced before execution,
- high-risk actions can require human approval,
- every action is auditable.

### What Kimbap does not claim
- the final outbound HTTP request is credential-free,
- proxy mode works for every protocol,
- arbitrary existing CLIs become safe without an adapter,
- host compromise is magically out of scope.

This precise trust model is important. The product wins by being honest and enforceable.

---

## Multi-tenant by design

Kimbap is intended for teams running multiple agents, workflows, and customers.

Tenant isolation is enforced in three layers:

1. **Namespace isolation**  
   Vault entries, policy, approvals, and audit are tenant-scoped.

2. **Policy isolation**  
   Every evaluation runs in tenant context.

3. **Cryptographic isolation**  
   Each tenant has its own key hierarchy.

That means token rotation and key rotation are independent.  
A compromised token can be revoked without re-encrypting all tenant data.

---

## Policy and approval

The real product moat is not just credential storage.  
It is **controlled execution**.

Kimbap makes it possible to express rules like:

- Agent A can read GitHub but cannot write.
- Agent B can draft emails but cannot send them without approval.
- Stripe refunds require approval above a threshold.
- Gmail send is rate-limited per tenant.
- Internal CRM writes are allowed only during business hours.

This turns “agent access” into something operators can actually govern.

---

## Why developers will adopt it

Kimbap is designed for fast adoption in real codebases.

### For teams with existing agents
Use proxy mode or `kimbap run`.

### For teams building new systems
Use explicit `kimbap call`.

### For teams with internal APIs
Generate or write a skill.

### For teams with OAuth-heavy workflows
Use connectors.

### For teams with governance requirements
Use approval, policy, and audit from day one.

The product is designed so teams can start simple and grow into more control, without changing the core model.

---

## Why now

Three trends are converging:

### 1. Agents are moving from toy workflows to real operations
They are no longer just searching the web. They are mutating external systems.

### 2. The current auth model is broken for agents
The ecosystem still defaults to “put a token in the environment.” That is not acceptable at scale.

### 3. Integration complexity is exploding
Teams need access to dozens of services, but they cannot afford to build and maintain a separate secure server for each one.

Kimbap turns that fragmentation into a platform opportunity.

---

## Why Kimbap can win

### 1. Strong technical lineage
Kimbap is not starting from scratch. It is derived from core runtime concepts already proven in production: vaulting, OAuth lifecycle, policy, approval, and audit.

### 2. Better abstraction than proxies or per-service servers
The action runtime model is broader than a proxy and cheaper than custom integrations.

### 3. Compounding integration ecosystem
Every skill adds value to every deployment.  
That creates a registry/network-effect dynamic.

### 4. Security becomes a product wedge, not just a feature
As agent usage expands, secure mediation becomes mandatory. Kimbap is built around that requirement from the start.

### 5. Works with how teams already build
Teams do not need to rewrite everything. They can adopt Kimbap incrementally.

---

## Example quickstart

```bash
brew install kimbap

# store a credential safely
printf '%s' "$GITHUB_TOKEN" | kimbap vault set GITHUB_TOKEN

# issue an agent identity
kimbap token create --agent agent-dev

# discover actions
kimbap actions list --service github

# execute one action
kimbap call github.list_pull_requests --repo owner/repo
```

---

## Initial product roadmap

### Phase 1
- core action runtime
- vault and tenant key hierarchy
- service token issuance
- explicit `kimbap call`
- Tier 1 skill execution
- local audit and embedded mode

### Phase 2
- proxy mode and subprocess mode
- connected multi-tenant server
- OAuth connectors and refresh lifecycle
- policy eval and action discovery
- OpenAPI-to-skill generation

### Phase 3
- approval queue
- Slack / Telegram / Email approvals
- lightweight console for approvals and audit
- GitHub-based skill installs
- private registry support

### Phase 4
- public registry
- SDKs
- workload identity
- optional MCP compatibility layer
- enterprise auth controls

---

## The long-term vision

Kimbap becomes the default secure execution layer for AI agents.

Not another secret store.  
Not another proxy.  
Not another collection of one-off integrations.

A single runtime that makes agent access to external systems:

- safer,
- easier,
- more governable,
- and much cheaper to scale.

---

## Summary

Kimbap sits at the exact point where the agent stack is weakest today:

**between reasoning and action.**

That is where credentials leak.  
That is where policy is missing.  
That is where approval is bolted on later.  
That is where most companies will feel pain first.

Kimbap turns that weak point into infrastructure.

**Secure actions for AI agents.**
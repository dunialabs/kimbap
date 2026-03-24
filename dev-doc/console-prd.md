# Kimbap Console PRD

**Document Version:** v0.2  
**Status:** Draft  
**Language:** English  
**Scope:** Console only  
**Target Product:** Kimbap Console  
**Related Components:** `kimbap-core`, existing TypeScript `kimbap-console` (legacy baseline)

---

## 1. Executive Summary

Kimbap Console is the web-based **operations and observability console** for the Kimbap platform. It remains implemented in TypeScript, but it is not deployed as an independent production runtime. Instead, the console is built into static assets and embedded into the main Go binary at build time.

In production, the product must behave as a **single-binary system**:

- one Go process
- one deployment artifact
- one network origin
- one authentication authority
- one operational control plane

The console is **not** the trust boundary, OAuth engine, vault, policy decision point, or action execution runtime. It is the **operator-facing UI** for inspecting, reviewing, approving, troubleshooting, and observing those capabilities.

The primary purpose of the console is **not broad configuration management**. Its primary purpose is:

- audit exploration
- log exploration
- stats and operational insight
- approval handling
- connector and OAuth health visibility
- incident triage and operator workflows

Configuration remains necessary, but it is a **secondary** concern. Where possible, configuration should be owned by the core runtime, APIs, and declarative configuration layers, while the console provides safe, server-mediated views and workflows over that state.

---

## 2. Product Positioning

Kimbap Console should be positioned as an **operations console**, not a generic admin/settings portal.

The console should answer questions such as:

- What happened?
- Who did it?
- What was attempted?
- What was allowed or denied?
- Why did it fail?
- Which connector or OAuth connection is degraded?
- Which approvals are blocking progress?
- What usage, error, and health trends are changing?

The console should only secondarily answer questions such as:

- Where do I change a setting?
- How do I initiate an integration connect flow?
- How do I revoke or reconnect a broken external service?

This means Kimbap Console should be optimized for **daily operational use**, not just initial setup.

---

## 3. Problem Statement

The current console exists as a TypeScript application separate from the main Go runtime. While this enables fast front-end iteration, a split production stack introduces several problems:

1. **Operational ambiguity**: multiple runtimes, multiple processes, and potentially multiple ports create unnecessary deployment and support complexity.
2. **Security ambiguity**: when a Node-based web tier exists separately from the main runtime, operators may incorrectly assume that the console participates in privileged logic.
3. **Version skew**: the backend and console can drift out of sync across releases.
4. **Authentication complexity**: cross-origin routing, cookies, CSRF, CORS, and session ownership become harder to reason about.
5. **Product ambiguity**: the console risks becoming a catch-all settings surface instead of a focused operational tool.
6. **OAuth ambiguity**: if OAuth initiation, status, and error handling are not clearly split between core and console, teams may place provider-specific logic in the UI where it does not belong.

The desired model is to preserve the strengths of the existing TypeScript console while removing the operational and packaging downsides of a separate production stack and clarifying that the console is primarily an **observability and operations surface**.

---

## 4. Product Decision

Kimbap will adopt the following product decision:

- **Keep the existing console in TypeScript.**
- **Do not run a Node web server in production.**
- **Build the console as static assets.**
- **Embed those assets into the Go binary at build time.**
- **Serve the console and API from the same Go process and same origin.**
- **Make the console primarily an audit/log/stats/approvals/health product.**
- **Keep OAuth protocol handling and token lifecycle in core, not in the console.**

This means:

- **development remains split** for speed and ergonomics
- **production becomes unified** for simplicity, security, and deployability
- **operations and observability become the center of the console IA**
- **configuration moves down in priority and stays server-mediated**

---

## 5. Goals

### 5.1 Primary Goals

1. Deliver a high-quality operations console without rewriting the existing TS frontend in Go.
2. Ship the console as part of a single production binary.
3. Ensure the console is same-origin with the Kimbap API.
4. Centralize all privileged behavior in Go runtime APIs.
5. Make audit, logs, stats, approvals, and health the primary console experience.
6. Make OAuth connection status, failure triage, reconnect, and revoke easy for operators without moving OAuth protocol logic into the UI.
7. Eliminate the need for a production Node runtime.
8. Preserve a fast front-end developer experience using the existing TS toolchain.

### 5.2 Secondary Goals

1. Guarantee console/backend version compatibility within a release artifact.
2. Reduce surface area for auth and session bugs.
3. Make local embedded mode and connected mode use the same console product.
4. Provide safe, minimal configuration and integration setup surfaces where needed.
5. Give operators enough information to diagnose connector/OAuth problems without access to raw secrets.

---

## 6. Non-Goals

This PRD does **not** define:

1. the full Kimbap core runtime architecture
2. the vault cryptography design
3. the policy language specification
4. the connector execution protocol
5. proxy classification internals
6. CLI command taxonomy outside console-related entry points
7. a full website or marketing site strategy
8. public anonymous web access
9. browser-side OAuth protocol implementation
10. broad settings-first admin UX as the primary console identity

This document is intentionally limited to the **operator console product**.

---

## 7. Target Users

### 7.1 Platform Operator
Responsible for observing instance health, investigating failures, handling approvals, and operating connectors and integrations.

### 7.2 Security / Compliance Reviewer
Responsible for reviewing approvals, policy effects, audit records, token/session metadata, and sensitive operational events.

### 7.3 On-Call Engineer / Incident Responder
Responsible for triaging connector degradation, refresh failures, runtime errors, queue backlogs, and service incidents.

### 7.4 Developer / Integrator
Responsible for adding or configuring connectors, skills, action mappings, and integration settings, and for investigating operational failures.

### 7.5 Tenant Administrator
Responsible for managing tenant-scoped configuration, member access, service connection health, and operational visibility within tenant boundaries.

---

## 8. Core Product Principles

### 8.1 TypeScript Stays
The existing console stack remains TypeScript-based. Kimbap will not rewrite the console simply to achieve language uniformity.

### 8.2 Production Runtime Must Be Single-Binary
The console must not require a separately deployed Node web server in production.

### 8.3 The Console Is an Operations UI, Not a Trust Boundary
The console may initiate privileged requests, but it must never directly own or implement privileged logic. Authoritative behavior lives in Go APIs and runtime services.

### 8.4 Observability First
The default design center for the console is operational visibility: audit, logs, stats, approvals, health, and troubleshooting.

### 8.5 Settings Are Secondary
Settings and configuration should exist, but they should not dominate the IA or the product identity. The console should increasingly prefer visibility, validation, and guided actions over unrestricted mutable admin forms.

### 8.6 OAuth Belongs to Core
OAuth protocol handling, provider definitions, callback handling, token exchange, refresh, revocation, and secret storage belong to core. The console only provides operator-facing workflows on top of those server-side capabilities.

### 8.7 Same-Origin by Default
The console and API should be served under the same origin to reduce auth complexity and simplify browser security posture.

### 8.8 Development and Production May Differ
A split dev topology is acceptable and desirable. A split production topology is not.

### 8.9 Release Coherency Matters
The console build shipped in a binary must correspond to the backend contract shipped in the same binary.

---

## 9. Product Definition

Kimbap Console is the embedded administrative and operational UI for operating a Kimbap instance.

Its **primary product areas** are:

- overview and health
- approvals inbox and review workflows
- audit exploration
- log exploration
- stats and usage trends
- connector / integration health
- OAuth connection health and recovery
- policy visibility during investigation and review

Its **secondary product areas** are:

- instance and tenant configuration screens
- token and session metadata visibility
- integration setup initiation
- skill and package inspection surfaces

Kimbap Console does **not**:

- execute actions outside the normal API/runtime path
- read secrets directly from storage
- implement OAuth token exchange or refresh in the browser
- independently issue tokens without server-side policy
- bypass approvals or policy enforcement
- hold authoritative business logic in front-end code

---

## 10. Responsibility Split: Core vs Console

### 10.1 Core Responsibilities

The Go core owns the authoritative implementation of:

- OAuth provider definitions/templates
- OAuth client installation state
- redirect URI validation
- PKCE and state handling
- callback handling
- code exchange
- token encryption and storage
- refresh token lifecycle
- revoke / disconnect semantics
- connector execution state
- policy evaluation
- approval state transitions
- audit generation
- structured logs and metrics collection
- token and session lifecycle

### 10.2 Console Responsibilities

The console owns the operator-facing UI for:

- initiating connect / reconnect flows through server APIs
- displaying integration and OAuth status
- displaying scopes, expiry, last refresh, and failure summaries
- initiating revoke / reconnect / retry workflows through server APIs
- browsing audit records
- browsing operational logs
- browsing usage and health statistics
- reviewing and acting on approvals
- visualizing policy effects and related operational context
- presenting safe, minimal configuration workflows where needed

### 10.3 Explicit Boundary

The console must never become the place where OAuth becomes “easy” by hiding real protocol logic in frontend code. OAuth should become easy because **core exposes clean abstractions and APIs**.

The console may make OAuth easier to operate, but not easier by owning protocol behavior.

---

## 11. Information Architecture Priority

The default console navigation should reflect the product decision that the console is primarily an operations surface.

### 11.1 Primary Navigation

- Overview
- Approvals
- Audit
- Logs
- Stats
- Integrations

### 11.2 Secondary Navigation

- Policies
- Tokens / Sessions
- Skills / Packages
- Settings

### 11.3 IA Rules

1. Audit, logs, stats, and approvals must be top-level destinations.
2. Settings must not dominate the left nav or homepage.
3. Integration/OAuth health should be presented as an operational concern, not only as a setup wizard concern.
4. The overview page should primarily summarize operational status, not configuration completeness.

---

## 12. User Experience Requirements

### 12.1 Entry Point
Operators should be able to open the console directly from the Kimbap instance without installing an additional web service.

Examples:

- `http://localhost:<port>/console`
- `<base-url>/console`

### 12.2 First-Time Experience
The first-time experience should support:

- instance readiness indication
- authenticated entry
- basic setup guidance
- visibility into missing prerequisites
- clear state when the instance is not yet fully configured

The first-time experience should not redefine the console into a settings portal.

### 12.3 Readability and Operational Clarity
The console must prioritize:

- fast scanability
- explicit status labels
- clear risk or approval indicators
- searchable and filterable tables
- stable URLs for deep linking
- predictable operator workflows
- obvious degraded / warning / healthy states

### 12.4 Failure States
The console must present understandable states for:

- backend unavailable
- auth expired
- insufficient permission
- unsupported feature on current instance mode
- partial data load failures
- stale version mismatch or invalid embedded assets
- connector disconnected
- OAuth refresh failed
- invalid redirect / missing provider configuration
- degraded metrics or log pipeline availability

---

## 13. Functional Requirements

### 13.1 Embedded Console Delivery

Requirements:

1. The production console must be generated as static assets.
2. The assets must be embedded into the Go binary at build time.
3. The Go server must serve the console without external Node runtime dependencies.
4. The console must support SPA routing with server-side fallback to `index.html` for valid client routes.
5. Static asset paths must support cache-safe hashed filenames.

### 13.2 Same-Origin Serving

Requirements:

1. The console should be served from the same origin as the API.
2. Browser sessions should not require cross-origin coordination.
3. The console should use the same auth/session authority as the rest of the product.
4. Production deployments should not require separate CORS configuration for normal console usage.

### 13.3 Authenticated Operator Access

Requirements:

1. Unauthenticated access to operator pages must be blocked.
2. Session expiration must be handled gracefully in the UI.
3. Permission failures must be rendered clearly.
4. The console must not reveal privileged data until authorized server-side responses are received.
5. The frontend must assume all sensitive access is server-mediated.

### 13.4 Overview / Operations Dashboard

Requirements:

1. Show instance identity and mode.
2. Show high-level service health.
3. Show recent operational activity.
4. Show pending approval count.
5. Show connector and OAuth health summary.
6. Show key error and failure indicators.
7. Show usage and trend highlights.
8. Show clear warnings for degraded subsystems.

### 13.5 Approvals UI

Requirements:

1. Operators must be able to view approval requests.
2. Requests must show requester, action summary, risk indicators, time, and current status.
3. Authorized reviewers must be able to approve or deny through the standard API.
4. Approval actions must show success/failure and resulting state.
5. The UI must support viewing approval history.

### 13.6 Audit Explorer

Requirements:

1. Operators must be able to browse audit records.
2. Audit views must support filtering by time, actor, tenant, action, status, and resource where applicable.
3. Individual records must have a detailed view.
4. The UI must make it easy to distinguish requested action, evaluated policy, approval state, and resulting outcome.
5. Export is desirable but not required in v1.

### 13.7 Log Explorer

Requirements:

1. Operators must be able to browse structured operational logs.
2. Log views must support filtering by time, severity, subsystem, tenant, connector, request/session identifier, and correlation identifier where available.
3. The UI must support quick pivots between related audit, log, approval, and connector records when correlation data exists.
4. The UI must clearly distinguish logs from audit records.
5. Raw secret material must never be displayed in logs.

### 13.8 Stats and Operational Insights

Requirements:

1. Operators must be able to view usage, latency, error, approval, and connector health trends.
2. The stats surface should support time range selection and basic segmentation where practical.
3. The UI must make degraded trends or unusual spikes visually obvious.
4. Early versions may prioritize tables and summary cards over sophisticated charting, but the product must expose operational stats as a first-class area.
5. The console should expose enough information to support routine capacity and reliability review.

### 13.9 Integrations and OAuth Health

Requirements:

1. Show installed connectors/integrations.
2. Show current status and last successful activity.
3. Show auth/configuration state without revealing secret material.
4. Show whether a connection is configured, connected, expired, degraded, revoked, or disconnected.
5. Show scope summary, expiry information, last refresh time, and recent refresh failures where available.
6. Allow authorized operators to initiate supported connect, reconnect, retry, or revoke workflows through server APIs.
7. Show actionable errors when a connector is broken, misconfigured, or disconnected.
8. Distinguish clearly between provider template/configuration problems, installation problems, and connection/token problems.

### 13.10 Policy Surfaces

Requirements:

1. Operators must be able to inspect policy state.
2. Policy views must clearly indicate scope and effect.
3. Policy views should be strongly connected to audit, approvals, and operational outcomes.
4. Any editing workflow must go through server-side validation.
5. Dangerous policy changes must be visibly highlighted.
6. Read-only visibility is acceptable before broad editability.

### 13.11 Tokens / Sessions Visibility

Requirements:

1. Authorized users must be able to inspect token/session inventory metadata.
2. Secret values must not be revealed by default.
3. Revocation and lifecycle actions must be server-mediated.
4. The console must clearly distinguish between identities, service tokens, OAuth connections, and runtime sessions.

### 13.12 Skills / Packages Visibility

Requirements:

1. Operators must be able to see installed skills/packages relevant to the instance.
2. UI should expose version, source, status, and change information where available.
3. Install/upgrade workflows may be limited in early versions, but visibility is required.

### 13.13 Settings and Instance Configuration

Requirements:

1. Provide a place for instance-level settings relevant to operators.
2. Organize settings clearly by area.
3. Gate dangerous mutations behind confirmation and authorization.
4. Keep the console strictly as the UI layer for configuration changes.
5. Prefer focused setup and recovery workflows over large generic admin forms.
6. Settings should be a secondary destination, not the primary product identity.

---

## 14. OAuth Product Requirements for Console-Only Scope

The console can only succeed as a low-friction OAuth surface if the core platform provides a normalized model.

### 14.1 Required Core Abstractions

For the console to support OAuth well, core should expose APIs and state around at least the following concepts:

- **OAuth Definition**: provider template and supported protocol metadata
- **OAuth Installation**: instance- or tenant-level configured provider credentials and policy binding
- **OAuth Connection**: the actual connected external account or workspace state

### 14.2 Console Expectations on Top of Those Abstractions

The console should be able to:

1. List supported OAuth-backed integrations.
2. Show whether an installation exists and whether it is valid.
3. Show whether a connection exists and whether it is healthy.
4. Initiate connect and reconnect through a backend-generated flow.
5. Display callback or redirect problems as server-reported states.
6. Display refresh degradation, scope mismatch, or revocation states.
7. Trigger revoke or disconnect through server APIs.

### 14.3 Explicit Non-Goals for the Console

The console should not itself implement:

- provider-specific callback handlers
- token exchange logic
- refresh logic
- client secret storage
- PKCE code verifier handling outside server-mediated flows
- browser-owned OAuth protocol state as a source of truth

---

## 15. Security Requirements

### 15.1 No Front-End Secret Authority
The console must never become a secret authority. It must not:

- embed long-lived secrets in client code
- persist raw secret material in browser storage
- fetch or display secrets without explicit server authorization and auditability
- perform direct privileged operations outside authenticated API paths

### 15.2 Browser Storage Minimization
The console should minimize use of persistent browser storage for sensitive state.

### 15.3 Session Safety
The browser-facing auth model should be compatible with same-origin serving and standard secure session handling.

### 15.4 Permission-Scoped Rendering
The console should avoid presenting actions the current operator cannot perform when feasible, but the backend remains authoritative.

### 15.5 Auditability
Administrative actions initiated through the console must remain fully auditable through the backend.

### 15.6 Safe Operational Visibility
The console should expose enough detail for incident triage without turning audit, logs, or integration views into secret disclosure surfaces.

---

## 16. Architecture Constraints

### 16.1 Frontend Technology
The console will continue to use the existing TypeScript codebase and front-end tooling.

### 16.2 Production Delivery Model
Production delivery must follow this model:

1. front-end builds to static files
2. static files are embedded into Go
3. Go serves both `/api` and `/console`
4. no production Node web server is required

### 16.3 Development Model
Development may use a split topology:

- TS dev server for hot reload
- Go API server for backend functionality
- local proxying between them

This split is allowed only for development convenience.

### 16.4 Routing Model
Recommended route model:

- `/api/*` -> server APIs
- `/console/*` -> embedded console
- client-side routes under `/console/*` -> SPA fallback

### 16.5 Versioning Model
The shipped binary must contain a console build that is intended for the bundled API surface.

At minimum, the product should support:

- a backend version identifier
- a console build identifier
- a compatibility check or metadata visibility for debugging

---

## 17. Build and Release Requirements

### 17.1 Build Pipeline
The release pipeline must support:

1. installing front-end dependencies
2. building the console static assets
3. copying or placing final assets into the Go embed target location
4. building the final Go binary
5. validating that the embedded console exists and is loadable

### 17.2 Deterministic Release Behavior
As much as practical, the release build should be reproducible and version-pinned.

### 17.3 No Runtime Front-End Dependency
A production Kimbap instance should not require:

- Node runtime
- pnpm/npm/yarn at runtime
- an external frontend container
- a separate web server for the operator console

### 17.4 Asset Caching
The console build should support cache-friendly asset delivery with hashed asset filenames where appropriate.

### 17.5 Binary-Level Cohesion
The artifact should behave as a single product release, not as a loosely coupled backend/frontend bundle.

---

## 18. Development Workflow Requirements

### 18.1 Front-End Development Experience
The product must preserve a productive front-end workflow, including:

- fast rebuilds or HMR
- independent UI iteration during development
- clear local API integration patterns

### 18.2 Contract Discipline
The frontend should consume stable backend contracts. Prefer generated or strongly typed client contracts where practical.

### 18.3 Local Dev Ergonomics
A developer should be able to run:

- backend only
- frontend only against a backend
- both together

without unusual manual setup.

---

## 19. Source Topology Requirements

Kimbap may support either of these source topologies:

### Option A: Same Repository
- Go code and TS console live in one repo
- console source stays in a dedicated web directory
- built assets are copied into an embed target directory within the Go module

### Option B: Separate Repositories
- TS console remains in a separate repo
- CI or release tooling fetches/builds the console
- final static assets are imported into the Go repo or build context before binary compilation

Regardless of source topology, the **production artifact must still be a single Go binary**.

---

## 20. Migration Requirements for Existing `peta-console`

The current TypeScript console should be adapted rather than rewritten.

### 20.1 Required Migration Direction

1. Retain TypeScript front-end implementation.
2. Remove any production requirement for a dedicated Node web server.
3. Move any privileged or server-like behavior into Go APIs.
4. Ensure the console can build to static assets suitable for embedding.
5. Adjust routing and asset assumptions to work under `/console` and same-origin deployment.
6. Rebalance the information architecture toward audit, logs, stats, approvals, and integration health.
7. Reduce settings-first assumptions in the current UX where they conflict with the new product identity.
8. Move OAuth complexity out of frontend code and into core-managed APIs and flows.

### 20.2 Anti-Patterns to Remove

The console should not depend on:

- production SSR as a hard requirement
- server-only Node middleware in production
- direct secret handling in frontend runtime
- environment assumptions incompatible with embedded serving
- provider-specific OAuth logic living in the browser app
- settings-heavy UX as the dominant homepage and nav model

### 20.3 Acceptable Transitional State

During migration, development may still use the existing TS server/tooling, but production must converge on embedded static delivery.

---

## 21. Success Metrics

### 21.1 Product Metrics

1. Operators can access the full console from a single Kimbap binary.
2. Production deployment requires no separate frontend service.
3. The console is used primarily for audit, logs, stats, approvals, and integration health workflows.
4. Release mismatches between backend and console materially decrease.
5. OAuth connect/reconnect/revoke workflows become easier to operate without adding OAuth protocol logic to the UI.

### 21.2 Engineering Metrics

1. No production Node runtime dependency.
2. Same-origin console and API.
3. Repeatable build pipeline for embedded assets.
4. Local dev workflow remains fast enough for UI development.
5. Frontend owns no privileged OAuth or secret lifecycle logic.

### 21.3 Reliability Metrics

1. Console asset serving works after fresh install and upgrade.
2. SPA routes resolve correctly from deep links.
3. Authentication/session flows work without cross-origin hacks.
4. Audit/log/stats pages remain usable under normal operational load.
5. Integration/OAuth status and failure states are visible enough for routine triage.

---

## 22. Acceptance Criteria for v1

Kimbap Console v1 is accepted when all of the following are true:

1. The existing TypeScript console can be built into static assets.
2. The static assets are embedded into the main Go binary.
3. The binary serves the console under a stable route such as `/console`.
4. The binary also serves the API under the same origin.
5. No production Node process is required for console delivery.
6. Core operator screens are present: overview, approvals, audit, logs, stats, integrations, and settings.
7. Privileged operations are performed exclusively via backend APIs.
8. Session expiration and permission failures are handled clearly in the UI.
9. Deep links to console routes work after page refresh.
10. The release process produces one production artifact.
11. OAuth connect/reconnect/revoke flows are initiated and displayed through the console, but the actual OAuth lifecycle remains core-owned.
12. The homepage and IA clearly reflect an operations-first product identity.

---

## 23. Out of Scope for v1

The following are explicitly out of scope unless separately approved:

1. a public marketing website bundled into the same console
2. offline-first browser behavior
3. plugin-executed frontend code from untrusted third parties
4. public anonymous dashboards
5. console-side execution of privileged workflows without backend mediation
6. multi-runtime production deployments as a primary target model
7. browser-owned OAuth lifecycle management as a source of truth
8. turning the console into a generic full-surface configuration IDE

---

## 24. Open Questions

These items may be resolved during implementation but do not block the core product decision:

1. Should the console use `/console` or be served from `/` with API namespacing?
2. What minimum generated contract strategy should be enforced between Go and TS?
3. Should static assets be copied into the Go repo during CI, or generated inside a unified monorepo workspace?
4. What backend metadata should be exposed to help diagnose console/backend version mismatches?
5. Which settings and policy edits should be enabled in v1 versus read-only visibility first?
6. How much charting sophistication is needed in v1 for stats, versus summary cards and tables first?
7. Which correlation pivots between logs, audit, approvals, and integrations are guaranteed in v1?

---

## 25. Final Product Statement

Kimbap Console will remain a TypeScript-based operator UI, but it will no longer exist as an independently deployed production web stack. The product will ship as a **single-binary embedded console**, with the Go runtime serving both the API and the built static console from the same origin.

Its identity is not a broad settings portal. Its identity is an **operations and observability console** centered on audit, logs, stats, approvals, integration health, and safe operator workflows. OAuth remains a core capability; the console exists to initiate, inspect, recover, and manage OAuth-backed integrations without owning OAuth protocol logic itself.


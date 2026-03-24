# Kimbap Core Go + Kimbap Console External Deployment and Load Balancing Review

> **Scope**: Track A only — sticky-session load balancing for Kimbap Core Go + stateless scaling for Kimbap Console.
> Track B (full distributed-state refactor) is documented for awareness but explicitly out of scope.

## 1. Executive Answer (Direct)

### 1) Is implementation possible?

- **Yes, with conditions.**
- **Kimbap Console** can be load-balanced as a multi-instance stateless web/API layer after a small set of fixes.
- **Kimbap Core Go** can be load-balanced in near term **only with sticky routing** for MCP sessions and Socket.IO flows, **and with per-replica server-connection reconciliation for regular MCP servers**.
- **Kimbap Core Go is not currently ready for fully stateless active-active (non-sticky) load balancing** because core runtime session/socket/request state is instance-local.

### 2) Recommended implementation method

- **Track A (recommended now, lower risk):** sticky-session load balancing for Kimbap Core Go + stateless scaling for Kimbap Console.
- *Track B (long-term, out of scope):* full distributed-state refactor for Kimbap Core Go (Redis/pub-sub/distributed stores) to remove sticky dependency.

### 3) What code should be changed and how?

- A concrete file-by-file change list is provided in section **6. File-Level Change Plan**.
- The minimum viable set is:
  - Kimbap Console host binding + scheduler controls + log dedupe hardening.
  - Kimbap Core Go distributed-safe rate limit and background-job ownership control.
  - LB routing/sticky policy aligned to MCP + Socket.IO behavior.

---

## 2. Scope and Evidence Baseline

This review is based on the current code in:

- `/Users/rentamac/Desktop/projects/kimbap-core`
- `/Users/rentamac/Desktop/projects/kimbap-console`
- `/Users/rentamac/Desktop/projects/kimbap-desk`

Key evidence points (kimbap-core):

- Kimbap Core Go session state in memory: `internal/mcp/core/sessionstore.go:15` — `SessionStore` struct with `sessions`, `proxySessions`, `userSessions` maps (lines 18–20)
- Kimbap Core Go auth requires local session lookup by `mcp-session-id`: `internal/middleware/auth.go:86` — `core.SessionStoreInstance().GetSession(sessionID)`
- Kimbap Core Go MCP controller requires local `ProxySession`: `internal/mcp/core/proxysession.go`
- Kimbap Core Go `ProxySession` keeps local transport/mappers: `internal/mcp/core/proxysession.go`
- Kimbap Core Go Socket.IO connections/pending maps are in memory: `internal/socket/service.go:40` (`connections map[string][]UserConnection`), `internal/socket/service.go:43` (`pendingRequests map[string]pendingRequest`)
- Kimbap Core Go approval path requires local socket-online check: `internal/socket/notifier.go:182` — `svc.IsUserOnline(userID)`
- Kimbap Core Go auto-starts Cloudflared tunnel (if configured) at startup: `cmd/server/main.go:601` — `cloudflared.AutoStartIfConfigExists()`
- Kimbap Console hardcodes `hostname = 'localhost'`: `server.js:17`
- Kimbap Console starts scheduler in app process: `server.js:160`
- Kimbap Console Docker entrypoint also starts scheduler in background: `docker/docker-entrypoint.sh:84–90`
- Kimbap Console log sync dedupe is app-level, but DB uniqueness is not enforced for `(proxyKey, idInCore)`: `jobs/log-sync.js:359`, `prisma/schema.prisma:34`

---

## 3. Current Architecture Constraints

### 3.1 Kimbap Core Go constraints (critical)

- **Session affinity is currently mandatory**
  - `SessionStore` uses process memory maps for sessions and proxy sessions (`internal/mcp/core/sessionstore.go:18–20`).
  - `AuthMiddleware` and MCP controller read these maps directly (`internal/middleware/auth.go:86`).
  - If a request lands on a different instance, session lookup fails.

- **MCP stream lifecycle is instance-bound**
  - `ProxySession` holds upstream transport, request-id mapper, progress trackers, and notification subscriptions in memory (`internal/mcp/core/proxysession.go`).
  - This prevents arbitrary cross-instance continuation without distributed runtime state.

- **Socket approval flow is instance-bound**
  - Socket connections and pending request resolvers are local maps (`internal/socket/service.go:40,43`).
  - `SocketNotifier.SendRequest()` first checks `IsUserOnline()` in local instance (`internal/socket/notifier.go:182`).
  - Under non-sticky routing, approval prompts can fail even if user is connected to another node.

- **Rate limit is local-memory only**
  - `RateLimitService` uses in-memory counters (`internal/security/ratelimit.go:14–16`).
  - Behavior becomes inconsistent across nodes behind a load balancer.

- **Background timers run per instance (mix of per-node and cluster-global work)**
  - Per-node timers that manage in-memory state must remain per instance (session cleanup in `SessionStore.startCleanupTimer()`, rate-limit cleanup in `RateLimitService.cleanupLoop()`, log batching/flush via `LogService`, idle server checker in `serverManager.startIdleChecker()`).
  - Cluster-global timers that act on shared DB state or external side effects should be single-owner or protected by a distributed lock (log webhook sync via `LogSyncService`, DB cleanup jobs via `EventCleanupService`).
  - In production multi-replica deployments, explicitly classify each loop as `per-node` vs `singleton` to avoid duplicate work and data races.
  - **Additional per-session goroutines (Oracle review finding):**
    - `RequestIDMapper.runCleanupLoop()` — per-session cleanup goroutine for expired request ID mappings (`internal/mcp/core/requestidmapper.go:171`). Per-node, safe under sticky sessions.
    - `PersistentEventStore.persistWorker()` — per-session background goroutine that batches event persistence to DB (`internal/mcp/core/eventstore.go:123`). Per-node, safe under sticky sessions.
    - `serverContext.tokenRefreshTimer` — per-server-connection `time.AfterFunc` for OAuth token refresh (`internal/mcp/core/servercontext.go:569`). **CRITICAL in multi-replica**: if multiple replicas connect to the same server, they may race on refresh token rotation. For servers using rotating refresh tokens, only one replica should manage the connection, or use non-rotating tokens.
  - **Token refresh race condition (Track A concern):** When server reconciliation causes N replicas to connect to the same regular server, each replica schedules its own `tokenRefreshTimer`. If the downstream server uses rotating refresh tokens (where using a refresh token invalidates it), concurrent refreshes from multiple replicas will cause N-1 failures. Mitigation options: (a) use non-rotating refresh tokens, (b) designate one replica as the token refresher via leader election, or (c) accept that Track A sticky-session deployment typically has each server connected to one replica only (no reconciliation of the same server across replicas).
  - **Additional per-connection/per-request goroutines (Oracle review finding #2):**
    - Server connection monitor ping ticker: `internal/mcp/core/servermanager.go:1982` — per connected downstream server, sends periodic pings.
    - `SocketNotifier` request timeout timers: `internal/socket/notifier.go:207` — `time.AfterFunc` per pending approval/request.
    - `PersistentEventStore` queue backpressure timer: `internal/mcp/core/eventstore.go:102` — fires when persist queue is full.
    - IP whitelist DB refresh-on-stale: `internal/security/ipwhitelist.go:25,78-98` — not a ticker, but periodic DB reads triggered by requests (TTL-based).
  - **EventCleanupService constructor auto-start caveat:** `NewEventCleanupService()` calls `Start()` in its constructor (`internal/mcp/core/eventcleanup.go:26`). When gating cluster jobs, you must call `Stop()` immediately after construction to prevent the periodic cleanup from running on non-primary replicas. The instance is still needed for per-session `CleanupStream()` calls.
  - **Socket.IO `Authorization` header is required for Go auth, not just LB stickiness:** Go's `SocketService` authenticates using the `Authorization` header (`internal/socket/service.go:258-269`). The `extraHeaders.Authorization` in kimbap-desk is functionally required, not optional.

- **Cloudflared auto-start is instance-local**
  - On startup, core can auto-start a Cloudflared Docker container if a tunnel config exists (`cmd/server/main.go:601`, `internal/service/cloudflared.go:31`).
  - In an external LB deployment, Cloudflared should be treated as ingress infrastructure: disable auto-start on core replicas or run Cloudflared as a separate, explicitly scaled component.

- **Regular (non-template) server connections are instance-local and not auto-reconciled cluster-wide**
  - `serverManager` stores regular server contexts in local memory (`internal/mcp/core/servermanager.go:45` — `serverContexts map[string]*ServerContext`).
  - `PreloadServers()` exists and is called on startup (`cmd/server/main.go:405–419`), but only preloads metadata — it does not establish connections.
  - Server start/connect is triggered via admin flows (`AddServer()` in `internal/mcp/core/servermanager.go:168`).
  - In multi-core deployment, `/admin` start/connect actions can initialize a server on one core instance while other core instances remain unaware.

### 3.2 Kimbap Console constraints

- **External deployment hardening**
  - `server.js` hardcodes `hostname = 'localhost'` (line 17).
  - The server `listen()` calls do not pass a hostname (so it typically binds on all interfaces), but explicit host binding is still recommended to prevent confusion and environment-specific behavior.

- **Scheduler duplication risk**
  - Scheduler starts in `server.js` (line 160) and also in Docker entrypoint background job (lines 84–90).
  - In multi-replica setup this multiplies quickly.

- **Log dedupe race risk**
  - Log sync pre-checks existing `idInCore`, then `createMany(skipDuplicates: true)` (lines 359–414 in `jobs/log-sync.js`).
  - Without a unique DB constraint, concurrent workers can still insert duplicates.

---

## 4. Feasibility Matrix

| Area | Current Feasibility | Notes |
|---|---|---|
| Kimbap Console horizontal scaling | **High** | After host binding + scheduler ownership fixes |
| Kimbap Core Go with sticky LB | **Medium** | Practical near-term route only if regular-server contexts are reconciled on every core replica |
| Kimbap Core Go without sticky (true stateless active-active) | **Low (currently)** | Requires significant distributed-state refactor (Track B) |

---

## 5. Recommended Implementation Design (Track A)

### Topology

- External LB in front of both services.
- Kimbap Console: N replicas (shared console DB).
- Kimbap Core Go: N replicas (shared core DB), **sticky routing required**.

### Routing policy

- Route to Kimbap Core Go paths:
  - `/mcp`
  - `/socket.io`
  - `/admin`
  - `/health`
  - `/.well-known/*` (OAuth metadata)
  - `/register`, `/authorize`, `/token`, `/introspect`, `/revoke` (OAuth core)
  - `/oauth/*` (OAuth admin/client mgmt)
  - `/user` (User API)
- Route UI/API paths to Kimbap Console.

### Sticky policy for Kimbap Core Go

You must keep all traffic that depends on the same in-memory state on the same `kimbap-core` replica.

- **MCP session affinity (mandatory)**
  - `/mcp` requests and SSE streams must be routed to the replica that created the session (`SessionStore`/`ProxySession` are in-memory).
  - Best key: `mcp-session-id` header.

- **Socket.IO approval affinity (mandatory only if you use approval/request-response workflows)**
  - Socket connections are stored by `userId` in the local `SocketService` maps (`internal/socket/service.go:40`).
  - If an MCP request triggers an approval prompt, the MCP session and the user's Socket.IO connection must land on the same replica.
  - Important nuance: Socket.IO `auth.token` is not an HTTP header and may not be usable for LB routing decisions.

Practical affinity strategies (pick based on the flows you need):

1) **If you only need MCP traffic (no cross-channel approval dependency)**:
- Prefer hash key order:
  1. `mcp-session-id` header
  2. `Authorization` bearer token
  3. LB cookie fallback

2) **If you need MCP + Socket.IO approval flows to work reliably without Track B**:
- Use a **user-level** affinity key that is visible to the LB for both `/mcp` and `/socket.io`:
  - Require `Authorization: Bearer <token>` on every `/mcp` request (not only initialize).
  - Ensure the Socket.IO client sends an LB-visible token (example: `extraHeaders.Authorization` in Node/Electron; browsers generally cannot set upgrade headers).
- Configure LB stickiness on that user-level key.
- Important token-type nuance:
  - `/socket.io` authentication currently uses `TokenValidator` (Kimbap access tokens), not the OAuth JWT validator (`internal/socket/service.go:274`, `internal/security/token.go`).
  - For this Track A design, prefer using the **same Kimbap access token** as the bearer token for both `/mcp` and `/socket.io` (so both auth and LB affinity stay consistent).
  - If you must use OAuth JWTs for `/mcp`, you likely need one of:
    1) extend Socket.IO auth to accept OAuth JWTs too, or
    2) implement Track B (cross-node approvals), or
    3) use an ingress that can hash on a stable user identifier derived from the token.

Fallback:
- If Socket.IO clients cannot provide any LB-visible affinity key and cross-node approvals must work, Track B (Socket.IO adapter / pub-sub) is required.

### Operational policy

- Run DB migrations as a one-time job (CI/CD pre-step), not per replica startup.
- Run scheduled jobs as dedicated singleton workers (or leader-elected mode), not in every web replica:
  - console log ingestion/sync (core → console DB)
  - core webhook log sync (if enabled)
  - cluster-global DB cleanup jobs (if centralized)
- If you run regular/shared servers: add per-replica regular-server reconciliation (startup + periodic) so every core instance has compatible server contexts.

---

## 6. File-Level Change Plan

### 6.1 Kimbap Console (minimum required)

**1. `server.js`**
- Current:
  - `hostname = 'localhost'` (line 17)
  - Runs DB initialization/migrations on every boot (`initializeDatabase()`)
  - scheduler starts unconditionally (lines 160, 192, 210)
- Change:
  - Make hostname configurable via `CONSOLE_HOSTNAME` env var (default `'0.0.0.0'`).
  - In multi-replica production, disable/gate boot-time DB init and run migrations as a one-time pipeline job (or leader-only).
  - Gate scheduler startup with env: `RUN_LOG_SYNC=true`.
- Why:
  - Enables external/LB deployment and prevents scheduler fan-out.

**2. `docker/docker-entrypoint.sh`**
- Current:
  - Starts background job always (lines 84–90).
- Change:
  - Only start background log-sync if explicit env flag `RUN_LOG_SYNC=true` is enabled.
  - If using dedicated worker container, remove this startup from web entrypoint.
- Why:
  - Prevents duplicate sync loops and race inserts.

**3. `jobs/log-sync.js`**
- Current:
  - App-level dedupe (lines 359–377) + createMany (lines 411–414).
- Change:
  - Keep app-level dedupe, but treat DB uniqueness as source of truth.
  - Add explicit logging when duplicates are rejected.
- Why:
  - Safe behavior under retries and concurrent workers.

**4. `prisma/schema.prisma`**
- Current:
  - No unique key for `(proxyKey, idInCore)` in `Log` model.
- Change:
  - Add: `@@unique([proxyKey, idInCore], map: "uniq_log_proxy_key_id_in_core")`
  - Generate migration and deploy.
- Why:
  - Makes log ingestion idempotent across replicas.

**5. `lib/mcp-gateway-cache.ts`**
- Current:
  - In-memory cache (lines 23–30) per instance.
- Change (optional hardening):
  - Add finite TTL for "valid" entries or DB-version-based cache busting.
- Why:
  - Faster config convergence after core endpoint changes.

**6. `lib/proxy-api.ts`**
- Current:
  - `connectAllServers(userid, token)` ignores `token` when `userid` is present (uses `makeProxyRequestWithUserId(..., undefined)` path, line 473–475).
- Change:
  - Give explicit token precedence when provided, even if `userid` exists.
- Why:
  - Owner-token-driven reconnect flows are required for deterministic server warm-up/reconcile behavior.

### 6.2 Kimbap Core Go (minimum required for sticky-scale)

**1. `internal/security/ratelimit.go`**
- Current:
  - In-memory counters in `RateLimitService` struct (lines 14–20).
- Change:
  - Introduce storage abstraction (`RateLimitStore` interface) and provide an in-memory default + optional Redis-backed implementation selectable via env var `RATE_LIMIT_STORE=memory|redis`.
- Why:
  - Consistent rate limiting across replicas.

**2. `cmd/server/main.go`**
- Current:
  - Instantiates multiple timers/services on every process: session cleanup, rate-limit cleanup, log batching/flush, event cleanup (`eventCleanup.Start()` at line 421), log sync (`logSyncSvc.Initialize()` at line 424), cloudflared auto-start (line 601).
  - Server preload runs on startup (lines 404–419).
- Change:
  - Split loops into:
    - **per-node (keep always)**: session cleanup, rate-limit cleanup, log batching/flush, idle server checker
    - **singleton/leader**: log webhook sync (`LogSyncService`), DB cleanup jobs (`EventCleanupService`), cloudflared auto-start
  - Gate singleton/leader loops behind env: `RUN_CLUSTER_JOBS=true|false` (or leader election).
  - Add server reconciliation loop (see item 6 below).
- Why:
  - Avoid uncontrolled multi-run behavior in large clusters.

**3. `cmd/server/main.go` (CORS)**
- Current:
  - Broad CORS `AllowedOrigins: []string{"*"}` (line 441).
  - Socket origin validation via `isAllowedSocketOrigin()` in `internal/socket/service.go:371` (request host matching only).
- Change:
  - Add configurable CORS allowlist via `CORS_ALLOWED_ORIGINS` env var.
  - If env is set, parse comma-separated origins and use them. If empty/unset, keep `*` as default for dev.
- Why:
  - Required hardening for external internet-facing deployment.

**4. `internal/middleware/auth.go`, `internal/mcp/core/proxysession.go`**
- Current:
  - Require local session/proxy maps.
- Change for Track A:
  - No major code change required if sticky policy is enforced.
- Why:
  - This is the main blocker for non-sticky active-active (Track B scope).

**5. `internal/socket/service.go`, `internal/socket/notifier.go`**
- Current:
  - Connection/pending maps local (`service.go:40,43`; `notifier.go:182`).
- Change for Track A:
  - Keep as-is with strict sticky policy.
- Why:
  - Required for reliable user approval flow across nodes (Track B scope for non-sticky).

**6. `internal/mcp/core/servermanager.go` + `cmd/server/main.go`**
- Current:
  - Regular server contexts are local-memory only (`servermanager.go:45`).
  - `PreloadServers()` is called on startup but only stores metadata, doesn't connect.
  - `AddServer()` is triggered by admin flows, not by per-node reconciler.
- Change:
  - Add per-node reconciliation loop:
    - On startup and periodically: load enabled regular servers from DB and ensure local contexts are connected (call `AddServer()` for missing ones).
    - Remove local contexts for disabled/deleted servers.
  - Add feature flags: `AUTO_CONNECT_ENABLED_SERVERS=true`, `SERVER_RECONCILE_INTERVAL_SEC=30`.
  - Start reconcile loop in `main.go` behind the feature flag.
- Why:
  - Without this, multi-core LB can route users to nodes that never connected the enabled regular servers.

---

## 7. Suggested LB/Infra Configuration

### 7.1 Health checks

- Kimbap Core Go: `GET /health` (returns JSON with status, timestamp, uptime — see `cmd/server/main.go:521`)
- Kimbap Console: `GET /` (or dedicated health endpoint if added)

### 7.2 Timeouts and transport

- Increase idle/read timeout for MCP streaming and Socket.IO.
- Keep websocket upgrade enabled for `/socket.io`.
- Disable proxy buffering for SSE/streaming responses (otherwise clients can hang waiting for buffered output).
- Note: Kimbap Core Go sets `WriteTimeout: 0` (line 543) and `IdleTimeout: 120s` (line 544) for long-lived SSE streams.

### 7.3 Header forwarding

- Preserve:
  - `x-forwarded-proto`
  - `x-forwarded-host`
  - `x-forwarded-for`
  - `last-event-id` (SSE resume support)
- Ensure forwarded headers are **owned by the LB/ingress** (sanitize/override at the edge) and that `kimbap-core` is not directly reachable from untrusted networks (otherwise clients can spoof `x-forwarded-*`).
- Current code already uses forwarded proto/host for metadata URL generation:
  - `internal/middleware/auth.go:673` — `isTrustedForwardedAuthSource()` validates loopback-only trust
  - `internal/middleware/auth.go:660` — `authPublicURL()` reads `X-Forwarded-Host` and `X-Forwarded-Proto`
  - Environment var `KIMBAP_PUBLIC_BASE_URL` provides canonical URL override (`internal/middleware/auth.go:685–704`)

### 7.4 Sticky capability note (practical)

- Not all managed LBs/ingresses support **header-based** affinity (especially using `Authorization`).
- If your LB only supports **cookie-based** stickiness:
  - verify your target MCP clients actually store and replay the LB cookie across requests/reconnects.
  - if they do not, prefer an ingress that supports header-based hashing (or implement Track B distributed state).

---

## 8. Rollout Plan (Track A)

### Single implementation pass

- Implement Kimbap Console minimum fixes (host binding config, scheduler gating, log dedupe hardening + DB uniqueness).
- Implement Kimbap Core Go cross-replica-safe rate limit (or explicitly accept per-instance limits with a documented caveat).
- Implement Kimbap Core Go CORS restriction and env-based configuration.
- Add server reconciliation loop for regular servers in Kimbap Core Go.
- Gate singleton background jobs behind env flag in both Kimbap Core Go and Kimbap Console.
- Deploy LB with sticky policy for Kimbap Core Go (MCP + Socket.IO approval affinity requirements).
- Move DB migrations to a one-time deployment pipeline job (do not run per replica boot).
- Add baseline observability:
  - session-not-found / invalid-session rate
  - approval timeout / user-offline rate
  - duplicate-log insert rejection rate

---

## 9. Validation Checklist (Go-Live)

- Functional:
  - MCP initialize/post/get/delete works through LB.
  - Socket approval flow succeeds under load.
- Resilience:
  - Kill one Kimbap Core Go pod during active sessions; verify expected recovery behavior.
  - Kill one Kimbap Console pod during dashboard usage.
- Data integrity:
  - No duplicate `log` rows for same `(proxyKey, idInCore)`.
- Security:
  - CORS allowlist enforced in production.
  - TLS termination + secure forwarded headers verified.

---

## 10. MCP Server Lifecycle Impact

### 10.1 Current behavior in code

- **Regular MCP servers are not lazy-started on first request**
  - They are started via admin actions (`AddServer()`) and stored in instance-local `serverContexts` (`internal/mcp/core/servermanager.go:45`).
  - If a node has not executed the start/connect flow, it cannot route to that server.

- **User temporary MCP servers are lazy-started per session**
  - When a client session is initialized, `StartUserTemporaryServersForSession()` iterates `user.launchConfigs` and creates temporary servers on that instance (`internal/mcp/core/servermanager.go:1296`).
  - Configure/unconfigure socket handlers also start/stop temporary servers immediately (in `internal/socket/service.go` event handlers).
  - Cleanup closes temporary servers when user sessions are gone on that instance (`internal/mcp/core/sessionstore.go:241`).

- **Transport/runtime execution model**
  - `launchConfig.command` ⇒ stdio transport (local process execution on core host) (`internal/mcp/core/transportfactory.go:92–112`).
  - `launchConfig.url` ⇒ HTTP/SSE transport (remote connection) (`internal/mcp/core/transportfactory.go:69–91`).
  - Runtime dependencies are on each core node (not on console).

### 10.2 What must change for external LB deployment

- **Regular server per-node reconciliation (required)**
  - Add cluster-safe regular-server reconciliation so every core replica has consistent regular server contexts.
  - Reconcile on startup and periodically from DB.

- **Installation model (required hardening)**
  - Every core replica must have identical runtime dependencies (Docker, binaries, credentials, network egress) for stdio/docker servers.
  - If you reconcile regular/shared servers on every core replica, you are effectively running **N copies** of each regular server (one per replica). Ensure downstream servers are safe to run concurrently without port/resource collisions, or switch regular servers to remote endpoints.
  - Add preflight validation at install/enable time:
    - command exists / executable
    - docker socket availability (if `command=docker`)
    - required env variables/secrets present

### 10.3 Decrypt key for regular/shared server launchConfig

Regular/shared server connections require decrypting `server.launchConfig` at runtime:

- Core decrypts using a raw token key: `internal/mcp/core/servermanager.go:794` — `decryptLaunchConfig()`
- Console encrypts server launchConfig using the owner token.

Implication:
- A fresh Core replica cannot auto-connect regular/shared servers unless it has access to the decrypt key.

Minimal viable options:

1) **Provide a cluster-wide service token as a secret to all Core replicas**
   - Treat this as a "service account" token with Owner/Admin power.
   - Use it only for reconcile/connect (not exposed externally).
   - Pros: no code refactor; enables per-node reconcile.
   - Cons: secret management and rotation are now critical.

2) **Refactor encryption for regular/shared servers to use a cluster key**
   - Encrypt/decrypt `server.launchConfig` with an env-provided cluster secret, not a human owner token.
   - Pros: removes dependency on owner token distribution.
   - Cons: requires a schema/data migration + code changes in install/update flows.

---

## 11. Load-Balanced Runtime Architecture

### 11.1 Component Architecture

```text
[Clients]
  - MCP Clients (Claude, ChatGPT, Cursor, etc.)
  - Kimbap Console Web UI
  - Kimbap Desk (Electron)
            |
            v
[External Load Balancer]
  - Path route to kimbap-console: UI/API
  - Path route to kimbap-core: /mcp, /socket.io, /admin, /user, /health, /.well-known/*, /register, /authorize, /token, /introspect, /revoke, /oauth/*
  - Sticky policy for kimbap-core (see section 5):
    - MCP-only: stick /mcp by mcp-session-id (Authorization/cookie as fallback)
    - MCP + Socket approvals: use a shared LB-visible user affinity key for both /mcp and /socket.io
        |                                  |
        v                                  v
[kimbap-console pool]                    [kimbap-core pool]
  - console-1                            - core-1 (SessionStore local, serverManager local, SocketService local)
  - console-2                            - core-2 (SessionStore local, serverManager local, SocketService local)
  - console-N                            - core-N
        |                                  |
        v                                  v
[Console DB (PostgreSQL)]              [Core DB (PostgreSQL)]

Additional runtime links:
  - kimbap-console -> (via canonical LB URL) -> kimbap-core /admin API
  - kimbap-core -> downstream MCP servers (stdio/http/sse/docker)
```

### 11.2 Request and Session Operation Flow

```text
Step 1: Session initialize
  MCP Client -> LB -> kimbap-core A
  Request: POST /mcp (initialize, Bearer token)
  kimbap-core A validates user against Core DB, creates local session/proxy-session,
  and returns mcp-session-id to client.

Step 2: Session continuity
  MCP Client -> LB -> kimbap-core A
  Requests: GET/POST /mcp with mcp-session-id
  LB keeps routing to the same core node using sticky policy.

Step 3: Socket channel continuity (approval / Kimbap Desk)
  Socket.IO Client -> LB -> kimbap-core A
  Request: /socket.io connect (token-based auth)
  If an MCP request triggers an approval prompt, this socket must land on the same
  replica as the MCP session owner.

Step 4: Admin operations from console
  kimbap-console -> LB -> kimbap-core B
  Request: /admin action (start/stop/connect server)
  The admin call may hit any core node.

Step 5: Multi-core consistency requirement
  If you rely on regular/shared servers, each core node must run startup + periodic
  reconcile so all nodes can serve routed MCP traffic safely.
```

### 11.3 Operational Model Under Load Balancing

1. **Startup**
   - Every core replica boots independently.
   - Each replica runs regular-server reconcile (startup + periodic) so it can serve routed traffic.
   - One-time DB migrations in CI/CD job, not in each web replica.

2. **Session lifecycle**
   - Initialize request creates a local session and returns `mcp-session-id`.
   - All subsequent MCP/Socket traffic for that session must be sticky to the same core node.
   - If sticky is broken, session lookup fails (`invalid/missing session` behavior).

3. **Temporary server lazy loading**
   - User temporary servers are started on the node that owns the user session.
   - Configure/unconfigure actions update DB + local temporary runtime on that owning node.

4. **Regular server runtime**
   - Regular (non-template) servers are node-local runtime contexts.
   - Reconcile is mandatory in multi-core mode if you rely on regular/shared servers.

5. **Scale-out / replacement**
   - When a new core pod is added, it starts empty.
   - Reconcile loop warms it up with enabled regular servers from DB.
   - Until reconcile finishes, that node should be considered not-ready for MCP traffic.

6. **Failure handling**
   - If the pinned core node fails, the session is effectively lost (current architecture).
   - Client should re-initialize and obtain a new `mcp-session-id`.
   - This is expected in Track A; Track B removes this limitation via distributed session/state.

### 11.4 LB Routing and Health Policy

1. **Path routing**
   - `kimbap-core`: `/mcp`, `/socket.io`, `/admin`, `/user`, `/health`, `/.well-known/*`, `/register`, `/authorize`, `/token`, `/introspect`, `/revoke`, `/oauth/*`
   - `kimbap-console`: UI/API paths except core routes above

2. **Sticky policy**
   - MCP-only deployments:
     1. Primary hash: `mcp-session-id`
     2. Secondary hash: `Authorization` token
     3. Fallback: LB cookie affinity
   - MCP + Socket approvals (without Track B):
     - Prefer a shared LB-visible user affinity key for both `/mcp` and `/socket.io` (commonly `Authorization`).

3. **Readiness/health**
   - Core readiness should include: process healthy + DB reachable + first reconcile completed (if using regular/shared servers).
   - Console readiness should include: process healthy + DB reachable.
   - Note: `GET /health` currently returns `200` always when the process is up (see `cmd/server/main.go:521`). For strict readiness gating, add a dedicated `/ready` endpoint or enhance `/health` semantics.

4. **Timeouts**
   - Keep websocket upgrade enabled.
   - Use longer idle/read timeout for MCP streaming and approvals.

---

## 12. Client Connection Guidance (Kimbap Desk)

### 12.1 Principle

- Clients connect to **one canonical public gateway URL** only.
- Clients must **not** select core instances directly.
- Traffic branching/affinity is handled by LB/ingress, not by client logic.

### 12.2 Current kimbap-desk Socket.IO connection

The Socket.IO client in kimbap-desk (`frontend/contexts/socket-context.tsx:185–217`) currently uses:

```typescript
const socketOptions = {
  auth: { token: config.token },
  reconnection: true,
  reconnectionAttempts: 5,
  reconnectionDelay: 1000,
  reconnectionDelayMax: 5000,
  timeout: 10000,
  transports: ['websocket', 'polling'],
  forceNew: true,
  query: { clientId: config.id }
}
```

### 12.3 Required change for LB compatibility

Add `extraHeaders` with `Authorization` bearer token so the LB can use it as an affinity key:

```typescript
const socketOptions = {
  auth: { token: config.token },
  extraHeaders: {
    Authorization: `Bearer ${config.token}`,  // LB-visible affinity key
  },
  reconnection: true,
  reconnectionAttempts: 5,
  reconnectionDelay: 1000,
  reconnectionDelayMax: 5000,
  timeout: 10000,
  transports: ['websocket'],  // Prefer websocket-only to avoid polling variability behind proxies
  forceNew: true,
  query: { clientId: config.id }
}
```

### 12.4 Reconnect behavior

- On transient disconnect: reconnect to the same gateway URL (already handled by Socket.IO reconnection).
- If session is reported invalid/expired: re-run initialize to get new `mcp-session-id`.

---

## 13. Deployment Execution Strategy

### 13.1 Target execution model

1. **Control-plane vs data-plane split**
   - `kimbap-console`: control-plane (setup/admin/operations UI)
   - `kimbap-core`: data-plane (actual MCP runtime path)

2. **Runtime separation**
   - Run console and core as separate services/process groups.

3. **Database separation**
   - Console DB and Core DB should be isolated (logical DB separation at minimum, separate clusters preferred).

### 13.2 Practical topology options

**Minimum viable production (small team):**
- 1 external LB
- 1 console instance (or 2 for HA)
- 2+ core instances behind LB (sticky enabled)
- Managed PostgreSQL (separate databases for console/core)
- 1 dedicated worker/scheduler instance (log sync + singleton jobs)

**Recommended production (standard):**
- 1 regional LB/ingress
- Console autoscaling group (2+ replicas, low traffic priority)
- Core autoscaling group (N replicas, strict sticky policy)
- Managed PostgreSQL with HA (Multi-AZ), separate DB users/roles for console/core
- Separate migration job and separate cron/worker deployment

### 13.3 Startup and rollout sequence

1. **Pre-deploy**: Run DB migrations once (pipeline job). Validate secrets/config.
2. **Deploy core**: Start/roll core pool first. Wait until readiness passes (including initial reconcile completion).
3. **Deploy console**: Roll console after core is healthy. Verify admin/API actions resolve to canonical core endpoint via LB.
4. **Deploy worker/scheduler**: Start singleton worker last. Ensure only one active scheduler is running.

### 13.4 Failure and recovery strategy

1. **Core node failure**: LB removes unhealthy node. Existing sticky sessions on failed node require client re-initialize. Surviving nodes continue serving healthy sessions.
2. **DB degradation**: Core should degrade safely (reject new sessions if DB unavailable).
3. **Worker failure**: Log sync/scheduled jobs delayed but should not break active MCP serving. Restart worker independently.

---

## 14. Multi-Replica Config Propagation

### 14.1 How config changes should propagate

When Console updates Core config, the write lands on one Core node (whichever the LB routed to).
To make every Core replica apply the change:

**Periodic reconcile (pull) — recommended for Track A:**
- Every Core replica periodically:
  - queries enabled servers from DB,
  - compares against its local `serverContexts`,
  - connects missing servers, reconnects changed ones, and removes disabled ones.
- This is the simplest operationally and works with eventual consistency.
- Accept a bounded convergence window (e.g., 10–60 seconds).
- Gate readiness for newly started pods until the first reconcile completes.

### 14.2 Do downstream MCP servers get replicated with Core?

It depends on transport type:

1) **Remote servers (HTTP / SSE)**: Core makes a network connection from each replica. You deploy and scale the downstream server **independently**. Core replicas do not "replicate the server process", only the client connections.

2) **Local servers (stdio / docker-run)**: Core spawns the server **locally on that Core host**. If you connect regular/shared servers on every Core replica, you run **N copies** of that server.

### 14.3 Risks when local servers are replicated with Core

- **Resource amplification**: CPU/memory usage scales with Core replica count.
- **Port/resource collisions**: if a server binds fixed ports, replicas can conflict.
- **Side-effect duplication**: N copies can amplify mistakes for write-heavy tools.
- **External rate limits / quotas**: multiple replicas may hit third-party APIs concurrently.
- **Operational coupling**: every Core node must have identical runtime dependencies.

If you need load-balanced Core with predictable behavior, prefer remote HTTP/SSE servers for shared/global tools.

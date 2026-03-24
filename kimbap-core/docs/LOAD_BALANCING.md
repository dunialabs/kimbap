# Load Balancing Deployment Guide

This document covers how to deploy Kimbap Core, Kimbap Console, and Kimbap Desk behind a load balancer for horizontal scaling and high availability.

## Architecture Overview

Kimbap's load balancing strategy uses **sticky sessions** for Kimbap Core (stateful MCP gateway) and **stateless scaling** for Kimbap Console (admin UI).

```
                    ┌──────────────┐
                    │  Load        │
  Clients ────────► │  Balancer    │
  (Desk, MCP)       │  (Sticky)   │
                    └──┬───┬───┬──┘
                       │   │   │
              ┌────────┘   │   └────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │ Core #1  │ │ Core #2  │ │ Core #N  │
        │ (primary)│ │          │ │          │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │             │             │
             └──────┬──────┘─────────────┘
                    ▼
            ┌──────────────┐
            │  PostgreSQL  │
            │  (shared)    │
            └──────────────┘
```

### Key Principles

- **Kimbap Core** uses in-memory state for MCP sessions, socket connections, and server contexts. All requests from a given client **must** route to the same Core replica (sticky sessions).
- **Kimbap Console** is stateless — all state lives in PostgreSQL. Console replicas can be scaled freely behind a round-robin LB.
- **Kimbap Desk** clients connect via WebSocket to a specific Core replica and maintain that connection for the duration of their session.

---

## Environment Variables Reference

### Kimbap Core (`kimbap-core`)

#### LB-Specific Variables

| Variable | Default | Description |
|---|---|---|
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated list of allowed CORS origins. Set to specific origins in production (e.g., `https://console.example.com,https://desk.example.com`). |
| `RUN_CLUSTER_JOBS` | `true` | Set to `false` on non-primary replicas to disable singleton background jobs (event cleanup, log sync, cloudflared). |
| `AUTO_CONNECT_ENABLED_SERVERS` | `false` | Set to `true` to enable the server reconciliation loop. Each replica will periodically connect to all enabled servers from the DB. |
| `SERVER_RECONCILE_INTERVAL_SEC` | `30` | How often (in seconds) each replica polls the DB for server changes. Minimum: 5. Only used when `AUTO_CONNECT_ENABLED_SERVERS=true`. |

#### Must Match Across All Core Replicas

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string. All replicas must share the same database. |
| `JWT_SECRET` | Secret for signing/validating tokens. Must be identical or token validation fails across replicas. |
| `KIMBAP_PUBLIC_BASE_URL` | Canonical public URL (e.g., `https://core.example.com`). Used for OAuth metadata and redirect URLs. Must be set in production behind a LB so all replicas generate consistent URLs. |
| `BACKEND_PORT` | Port to listen on (default `3002`). Typically the same across replicas. |

#### Other Relevant Variables

See `.env.example` for the full list. Key variables like `ENCRYPTION_KEY`, `KIMBAP_AUTH_BASE_URL`, and TLS settings (`ENABLE_HTTPS`, `SSL_CERT_PATH`, `SSL_KEY_PATH`) should also be consistent across replicas.

### Kimbap Console (`kimbap-console`)

| Variable | Default | Description |
|---|---|---|
| `CONSOLE_HOSTNAME` | `0.0.0.0` | Hostname/IP to bind the server to. Use `0.0.0.0` for container deployments. |
| `RUN_LOG_SYNC` | `true` | Set to `false` on non-scheduler replicas to disable the log sync job. Only one console instance should run log sync. |

### Kimbap Desk (`kimbap-desk`)

No additional environment variables required. Kimbap Desk automatically sends `Authorization` headers for LB session affinity and uses WebSocket-only transport (HTTP long-polling is disabled because each poll creates a new HTTP request that may land on a different backend, breaking session affinity).

---

## Deployment Topologies

### Minimum Viable Production (Small Team)

```yaml
# 1 external LB
# 1 console instance (or 2 for HA)
# 2+ core instances behind LB (sticky enabled)
# Managed PostgreSQL
# 1 dedicated worker instance (log sync + singleton jobs)
```

**Core primary replica:**
```env
CORS_ALLOWED_ORIGINS=https://console.example.com,https://desk.example.com
RUN_CLUSTER_JOBS=true
AUTO_CONNECT_ENABLED_SERVERS=true
SERVER_RECONCILE_INTERVAL_SEC=30
```

**Core additional replicas:**
```env
CORS_ALLOWED_ORIGINS=https://console.example.com,https://desk.example.com
RUN_CLUSTER_JOBS=false
AUTO_CONNECT_ENABLED_SERVERS=true
SERVER_RECONCILE_INTERVAL_SEC=30
```

**Console primary replica:**
```env
CONSOLE_HOSTNAME=0.0.0.0
RUN_LOG_SYNC=true
```

**Console additional replicas (if any):**
```env
CONSOLE_HOSTNAME=0.0.0.0
RUN_LOG_SYNC=false
```

### Recommended Production (Standard)

```yaml
# 1 regional LB/ingress (e.g., AWS ALB, GCP LB, nginx)
# Console autoscaling group (2+ replicas, low traffic)
# Core autoscaling group (N replicas, strict sticky policy)
# Managed PostgreSQL with HA (Multi-AZ)
# Separate migration job and separate cron/worker deployment
```

---

## Load Balancer Configuration

### Sticky Session Policy

Kimbap Core **requires** sticky sessions (session affinity). Configure your LB to route all requests from a given client to the same backend.

**Recommended sticky methods (in order of preference):**

1. **Cookie-based** (LB-generated cookie): Most reliable. LB sets a cookie on first response, routes subsequent requests by cookie value. Set `Secure; HttpOnly` flags.
2. **Header-based** (`Mcp-Session-Id` or `Authorization`): Kimbap Desk and MCP clients send these on every request. LB can hash for affinity. Prefer `Mcp-Session-Id` over `Authorization` to avoid credential exposure in LB logs.
3. **Source IP**: Simplest but unreliable behind NAT/proxies.

### Example: nginx (Open-Source)

> **Note:** The `sticky` directive requires NGINX Plus or the third-party `nginx-sticky-module`. For open-source nginx, use `hash`-based upstream selection as shown below.

```nginx
# WebSocket connection upgrade map
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

upstream kimbap_core {
    # Hash-based affinity using Mcp-Session-Id header.
    # All requests with the same session ID route to the same backend.
    # Use 'consistent' to minimize disruption when replicas are added/removed.
    hash $http_mcp_session_id consistent;

    server core-1:3002;
    server core-2:3002;
    server core-3:3002;
}

upstream kimbap_console {
    # Round-robin (stateless)
    server console-1:3000;
    server console-2:3000;
}

server {
    listen 443 ssl;
    server_name core.example.com;

    # WebSocket support for Socket.IO
    location /socket.io/ {
        proxy_pass http://kimbap_core;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    # MCP SSE streaming
    location /mcp {
        proxy_pass http://kimbap_core;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;           # Required for SSE
        proxy_cache off;
        proxy_read_timeout 3600s;      # Long-lived SSE streams
    }

    # All other Core routes
    location / {
        proxy_pass http://kimbap_core;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 443 ssl;
    server_name console.example.com;

    location / {
        proxy_pass http://kimbap_console;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

> **NGINX Plus alternative:** If using NGINX Plus, you can replace the `hash` directive with `sticky cookie srv_id expires=1h httponly secure;` for LB-managed cookie-based affinity.

### Example: AWS ALB

Configure via AWS Console or CloudFormation. Key settings:

```yaml
# Target Group for Kimbap Core
CoreTargetGroup:
  Type: instance
  Protocol: HTTP
  Port: 3002
  HealthCheckPath: /ready
  HealthCheckIntervalSeconds: 15
  HealthyThresholdCount: 2
  UnhealthyThresholdCount: 3
  # Stickiness via ALB target group attributes
  TargetGroupAttributes:
    - Key: stickiness.enabled
      Value: "true"
    - Key: stickiness.type
      Value: lb_cookie
    - Key: stickiness.lb_cookie.duration_seconds
      Value: "86400"  # 24h

# Target Group for Kimbap Console
ConsoleTargetGroup:
  Type: instance
  Protocol: HTTP
  Port: 3000
  HealthCheckPath: /
  TargetGroupAttributes:
    - Key: stickiness.enabled
      Value: "false"
```

> **ALB idle timeout:** ALB's default idle timeout is 60s, configurable up to 4000s. For long-lived SSE/WebSocket connections, set the ALB idle timeout to the maximum (`4000`) and rely on application-level keepalives (Socket.IO ping/pong every 25s) and client-side reconnection for connections that exceed this.

### Health Checks

| Service | Liveness | Readiness |
|---|---|---|
| Kimbap Core | `GET /health` → 200 | `GET /ready` → 200 (waits for initial server reconciliation) |
| Kimbap Console | `GET /` → 200 | Same as liveness |

**Important:** Use `/ready` (not `/health`) for LB health checks on Kimbap Core. The `/ready` endpoint returns `503` until the first server reconciliation completes **successfully** (owner token available and servers connected), preventing traffic from reaching a replica that isn't ready to serve.

> **Note:** `/health` is process-liveness only — it does not check DB connectivity or downstream server state. `/ready` gates on the first successful reconciliation but is not a continuous "DB is healthy" signal after that.

### Timeouts

| Setting | Recommended Value | Reason |
|---|---|---|
| Idle timeout | `≥ 120s` (nginx) / `4000s` max (ALB) | MCP SSE streams and WebSocket connections are long-lived. Socket.IO ping/pong (25s interval) keeps connections alive within the idle timeout. |
| Read timeout | `3600s` | SSE streams can last hours. Clients auto-reconnect on timeout. Set to 1h as a practical maximum; the application handles reconnection. |
| WebSocket timeout | `3600s` | Socket.IO connections are persistent. Rely on client reconnection for longer sessions. |
| Proxy buffering | `off` | SSE streams must not be buffered. |

### Public URL Configuration

In production behind a load balancer, set `KIMBAP_PUBLIC_BASE_URL` to the public-facing URL (e.g., `https://core.example.com`). This is used for:

- OAuth authorization server metadata URLs
- Token endpoint URLs in OAuth flows
- Redirect URI validation

> **Note:** Kimbap Core only trusts `X-Forwarded-*` headers from loopback addresses by default. Setting `KIMBAP_PUBLIC_BASE_URL` is the reliable way to ensure correct URL generation behind a LB.

Ensure your LB forwards these headers for operational purposes:

- `X-Forwarded-For` — Used for rate limiting and IP whitelisting (trusted from loopback only)
- `Upgrade` / `Connection` — Required for WebSocket upgrade
- `Authorization` — Required for authentication (forwarded as-is)
---

## Server Reconciliation

When `AUTO_CONNECT_ENABLED_SERVERS=true`, each Core replica independently:

1. **On startup**: Runs initial reconciliation (queries DB, connects enabled servers)
2. **Periodically**: Every `SERVER_RECONCILE_INTERVAL_SEC` seconds, re-queries the DB
3. **Connects** servers that are enabled in DB but not connected locally
4. **Disconnects** servers that have been disabled/deleted in DB

### How It Works

```
Admin enables server S via Console
    ↓
Console writes to DB: server S enabled=true
    ↓
Core Replica 1 reconcile tick → finds S missing → calls AddServer(S)
Core Replica 2 reconcile tick → finds S missing → calls AddServer(S)
    ↓
Both replicas now have S connected (eventual consistency)
```

### Convergence Window

Changes propagate within `SERVER_RECONCILE_INTERVAL_SEC` seconds. During this window:
- A client routed to a replica that hasn't reconciled yet may get a "server not found" error
- The LB readiness probe prevents this during startup (initial reconcile must complete first)
- For ongoing changes, accept eventual consistency or reduce the interval

### Owner Token Requirement

Server connections require the **owner token** for decrypting `launchConfig`. The owner token is set when an Owner user first accesses the API. If a fresh replica starts before any Owner has authenticated, reconciliation will log a warning and skip server connections until the token becomes available.

### Local (stdio) vs Remote (HTTP/SSE) Servers

- **Remote servers**: Each replica creates a separate network connection. The downstream server is deployed independently. Safe and recommended for LB.
- **Local servers**: Each replica spawns a local process. N replicas = N copies. Use with caution — can cause resource amplification, port conflicts, and side-effect duplication.

**Recommendation:** For shared/global tools in a multi-replica setup, prefer remote HTTP/SSE servers.

---

## Singleton Job Management

Certain background jobs must run on exactly one instance:

| Job | Environment Variable | Description |
|---|---|---|
| Event DB cleanup | `RUN_CLUSTER_JOBS=false` (Core) | Periodic deletion of expired events |
| Log webhook sync | `RUN_CLUSTER_JOBS=false` (Core) | Syncs logs to external webhooks |
| Cloudflared tunnel | `RUN_CLUSTER_JOBS=false` (Core) | Auto-starts Cloudflare tunnel |
| Log sync scheduler | `RUN_LOG_SYNC=false` (Console) | Syncs logs from Core to Console DB |

### Deployment Pattern

**Option A: Designate one replica as "primary"**
```yaml
# Primary replica
env:
  RUN_CLUSTER_JOBS: "true"
  AUTO_CONNECT_ENABLED_SERVERS: "true"

# Additional replicas
env:
  RUN_CLUSTER_JOBS: "false"
  AUTO_CONNECT_ENABLED_SERVERS: "true"
```

**Option B: Separate worker deployment**
```yaml
# Core API replicas (all)
env:
  RUN_CLUSTER_JOBS: "false"
  AUTO_CONNECT_ENABLED_SERVERS: "true"

# Core worker (1 replica, no LB traffic)
env:
  RUN_CLUSTER_JOBS: "true"
  AUTO_CONNECT_ENABLED_SERVERS: "false"
```

---

## Startup and Rollout Sequence

1. **Pre-deploy**: Run DB migrations once (pipeline job or init container). Validate secrets/config.
2. **Deploy Core**: Start/roll Core pool first. Wait until readiness passes on all replicas (including initial reconcile).
3. **Deploy Console**: Roll Console after Core is healthy. Verify admin/API actions resolve to Core via LB.
4. **Deploy Worker/Scheduler**: Start singleton worker last. Ensure only one active scheduler is running.

### Rolling Updates

For zero-downtime rolling updates:

1. LB marks old replica as draining (stop sending new connections)
2. Existing connections on old replica continue until they disconnect or timeout
3. New replica starts, passes readiness probe, begins receiving traffic
4. Old replica shuts down gracefully (10s timeout for in-flight requests)

**Note:** Clients with active MCP sessions on a draining replica will need to re-initialize when the replica shuts down. This is expected behavior for sticky-session deployments.

---

## Failure and Recovery

| Scenario | Behavior | Recovery |
|---|---|---|
| Core replica failure | LB removes unhealthy node. Existing sticky sessions on failed node are lost. | Clients re-initialize on surviving replicas. |
| DB degradation | Core degrades safely (rejects new sessions if DB unavailable). | Fix DB. Core auto-recovers. |
| Worker failure | Log sync and cleanup jobs delayed, but MCP serving continues. | Restart worker independently. |
| Network partition | Replicas may have inconsistent server state during partition. | Reconciliation loop auto-heals when connectivity restores. |

---

## Monitoring and Observability

### Key Metrics to Monitor

- **Per-replica**: Active sessions, connected servers, memory usage, goroutine count
- **Cluster-wide**: Total active sessions (sum across replicas), reconciliation lag, DB connection pool
- **LB-level**: Request distribution, sticky session effectiveness, 5xx rates per replica

### Health Endpoints

```bash
# Liveness (is the process running?)
curl http://core:3002/health
# {"status":"healthy","timestamp":"2026-02-26T00:00:00.000Z","uptime":3600}

# Readiness (is the replica ready for traffic?)
curl http://core:3002/ready
# {"status":"ready","timestamp":"2026-02-26T00:00:00.000Z"}
# or 503: {"status":"not_ready","reason":"initial server reconciliation in progress"}
```

---

## Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kimbap-core
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kimbap-core
  template:
    metadata:
      labels:
        app: kimbap-core
    spec:
      containers:
      - name: kimbap-core
        image: your-registry/kimbap-core:latest
        ports:
        - containerPort: 3002
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: kimbap-secrets
              key: database-url
        - name: CORS_ALLOWED_ORIGINS
          value: "https://console.example.com"
        - name: AUTO_CONNECT_ENABLED_SERVERS
          value: "true"
        - name: SERVER_RECONCILE_INTERVAL_SEC
          value: "30"
        - name: RUN_CLUSTER_JOBS
          value: "false"   # Use a separate worker for cluster jobs
        livenessProbe:
          httpGet:
            path: /health
            port: 3002
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 3002
          initialDelaySeconds: 10
          periodSeconds: 5
          failureThreshold: 12  # Allow up to 60s for initial reconcile
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kimbap-core-worker
spec:
  replicas: 1   # Exactly one worker
  # Recreate strategy ensures no overlap: old pod terminates before new starts.
  # This prevents two workers running singleton jobs simultaneously during rollouts.
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: kimbap-core-worker
  template:
    metadata:
      labels:
        app: kimbap-core-worker
    spec:
      containers:
      - name: kimbap-core
        image: your-registry/kimbap-core:latest
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: kimbap-secrets
              key: database-url
        - name: RUN_CLUSTER_JOBS
          value: "true"
        - name: AUTO_CONNECT_ENABLED_SERVERS
          value: "false"
---
apiVersion: v1
kind: Service
metadata:
  name: kimbap-core
  annotations:
    # If using AWS LB Controller, configure stickiness at the Ingress level:
    # alb.ingress.kubernetes.io/target-group-attributes: stickiness.enabled=true,stickiness.type=lb_cookie,stickiness.lb_cookie.duration_seconds=86400
spec:
  selector:
    app: kimbap-core
  ports:
  - port: 3002
    targetPort: 3002
  # NOTE: Do NOT use sessionAffinity: ClientIP here — it does not work correctly
  # behind an Ingress controller. Configure stickiness at the Ingress/LB layer instead.
```

---

## Troubleshooting

### Client gets "session not found" errors
- **Cause**: LB lost sticky session affinity and routed to a different replica.
- **Fix**: Verify sticky session configuration. Check LB logs for session cookie/header presence.

### Server not available on some replicas
- **Cause**: Reconciliation hasn't run yet, or owner token is unavailable.
- **Fix**: Check `/ready` endpoint. Verify `AUTO_CONNECT_ENABLED_SERVERS=true`. Check Core logs for "owner token not available" warnings.

### Duplicate log entries in Console
- **Cause**: Multiple Console replicas running log sync.
- **Fix**: Ensure only one Console replica has `RUN_LOG_SYNC=true`. The unique constraint on `(proxy_key, id_in_core)` prevents duplicates at the DB level.

### High memory usage on Core replicas
- **Cause**: Each replica maintains in-memory state for all sessions and server connections.
- **Fix**: Scale horizontally (add replicas) rather than vertically. Monitor per-replica session counts.

### CORS errors from Console/Desk
- **Cause**: `CORS_ALLOWED_ORIGINS` not configured for the Console/Desk domains.
- **Fix**: Set `CORS_ALLOWED_ORIGINS` to include all client origins.

### `/ready` stuck at 503 (never becomes ready)
- **Cause**: Owner token not available (no Owner user has authenticated yet), or database is unreachable.
- **Fix**: Check Core logs for `reconcile: owner token not available` or `failed to query enabled servers`. Ensure at least one Owner user has logged in to populate the owner token. Verify `DATABASE_URL` is correct and the DB is accessible.

### `/ready` returns 200 but servers are not connected
- **Cause**: Reconciliation completed but some servers failed to connect (connection errors are logged but don't block readiness after the first successful pass).
- **Fix**: Check Core logs for `reconcile: failed to connect server` warnings. Verify downstream server health and network connectivity.

### ALB disconnects long-lived SSE/WebSocket connections
- **Cause**: ALB idle timeout (max 4000s) closes connections that exceed it, even with keepalives.
- **Fix**: Set ALB idle timeout to maximum (4000s). Socket.IO's ping/pong (25s) keeps WebSocket connections alive within this limit. For SSE, ensure clients handle reconnection via `Last-Event-ID`. If connections still drop, consider using NLB (Network Load Balancer) which supports indefinite TCP idle timeouts.

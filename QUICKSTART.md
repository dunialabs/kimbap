# Kimbap Quick Start — Core + Console

## Prerequisites

- Go 1.24+
- Node.js 18+
- Docker
- PostgreSQL 15+

## Step 1: Start Kimbap Core

```bash
cd kimbap-core
make deps
cp .env.example .env
docker compose up -d
make dev
```

Core runs on http://localhost:3002.

## Step 2: Start Kimbap Console

```bash
cd kimbap-console
npm install
cp .env.example .env.local
npm run dev
```

Console runs on http://localhost:3000.

## Step 3: First Use

1. Open http://localhost:3000
2. Set master password; creates owner token; done.

## Connection Details

To configure Core connection, set in `kimbap-console/.env.local`:

```env
KIMBAP_CORE_URL="http://localhost:3002"
```

> **Note:** Priority: 1. Database config → 2. `KIMBAP_CORE_URL` env var → 3. Error (no auto-detection)

## Supported Interfaces

Kimbap Core exposes multiple API surfaces. Use the canonical ones for all new work:

| Interface | Path | Status | Use When |
|---|---|---|---|
| **REST v1 API** | `/api/v1/*` | **Canonical** | Programmatic access, automation, new integrations |
| Admin API | `/admin` | Legacy (frozen) | Internal use only; do not build new features against this |
| User API | `/user` | Legacy (frozen) | Internal use only; do not build new features against this |
| Health | `/health`, `/ready` | Stable | Liveness and readiness probes |

**For new integrations, always use `/api/v1`.**

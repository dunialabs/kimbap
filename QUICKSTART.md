# Kimbap Quick Start — Core + Console

Prerequisites:
- Go 1.24+
- Node.js 18+
- Docker
- PostgreSQL 15+

Step 1: Start Kimbap Core
- Clone the core repo (or use existing workspace)
- `make deps`
- `cp .env.example .env`
- `docker compose up -d`
- `make dev` — Core runs on port 3002

Step 2: Start Kimbap Console
- `cd ../kimbap-console`
- `npm install`
- `cp .env.example .env.local`
- `npm run dev` — Console runs on port 3000 and auto-detects Core at localhost:3002

Step 3: First Use
- Open http://localhost:3000
- Set master password; creates owner token; done

Connection details
- Console auto-detects Core. Override with:
- MCP_GATEWAY_URL="http://localhost:3002" in console/.env.local
- Note: The actual env var used in code is MCP_GATEWAY_URL. The documented alias is KIMBAP_CORE_URL.
- When configuring in docs, use MCP_GATEWAY_URL to connect Console to Core.

Keep it concise — this guide is ~80 lines max when expanded.

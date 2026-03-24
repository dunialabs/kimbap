# Kimbap Console

Kimbap Console — Operations Console for the Kimbap platform. A web app for audit, logs, approvals, and policy management.

## 🚀 Features
- 🔐 **Secure auth** – email verification with JWT
- 👥 **Role-based users** – Owner, Admin, Member permissions
- 🔑 **Token management** – create and manage API access tokens
- 🛠️ **Tool configuration** – configure MCP tools and permissions
- 📊 **Live monitoring** – real-time server performance and usage
- 💾 **Backup and restore** – cloud and local data recovery

## 🛠️ Tech Stack
- **App runtime:** Next.js 15 App Router + TypeScript
- **API layer:** Next.js API Routes (`app/api/*`)
- **Database:** PostgreSQL + Prisma ORM
- **Auth:** JWT + bcrypt
- **UI:** Radix UI + shadcn/ui + Tailwind CSS
- **Tooling:** Docker + Docker Compose

## 📁 Project Structure
```
kimbap-console/
├── app/                    # Next.js App Router
│   ├── (auth)/             # Auth pages
│   ├── (dashboard)/        # Dashboard pages
│   └── api/                # API routes
│       ├── auth/           # Auth APIs
│       ├── external/       # External REST API (automation clients)
│       └── v1/             # Internal protocol API (cmdId-based)
├── components/             # React components
│   └── ui/                 # Base UI primitives
├── lib/                    # Utilities (api client, auth, prisma, types)
├── prisma/                 # Database schema
├── hooks/                  # React hooks
├── docker-compose.yml      # Docker configuration
└── package.json            # Project config
```

## 🚀 Getting Started

### Deployment Options

#### 🐳 Option 1: Docker (production recommended)
Use the prebuilt Docker image for quick deployment.

**Standard ports:**
```bash
curl -O https://raw.githubusercontent.com/dunialabs/kimbap-console/main/docker-compose.yml

docker compose up -d
# Frontend: http://localhost:3000
# Console API (same app): http://localhost:3000/api/*
```

**Auto port allocation:**
```bash
npm run docker:deploy:auto
# or find and apply available ports
npm run docker:find-ports
docker compose --env-file .env.ports up -d
```

See [DOCKER_USAGE.md](./DOCKER_USAGE.md) for details.

#### 💻 Option 2: Local development

**Requirements**
- Node.js 18+
- Docker & Docker Compose (for Postgres)
- Git

**Setup**
1) Clone and install
```bash
git clone https://github.com/dunialabs/kimbap-console.git
cd kimbap-console
npm install
```

2) Configure env vars
```bash
cp .env.example .env.local
```
Edit `.env.local` as needed:
```env
DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_console?schema=public"
JWT_SECRET="your-super-secret-jwt-key"
# Optional override when Kimbap Core is not auto-detected
KIMBAP_CORE_URL="http://localhost:3002"
```

3) Start dev environment
```bash
npm run dev
```
This will:
- Check/install Cloudflared (Docker)
- Start PostgreSQL (Docker) and wait until ready
- Push the Prisma schema
- Start the Next.js dev server

4) Access
- 🌐 Frontend: http://localhost:3000
- 🚀 Console API: http://localhost:3000/api/*
- 🗄️ Adminer: http://localhost:8080
- 🧠 Kimbap Core (separate service): http://localhost:3002

## 🛠️ Developer Commands

### Unified dev (recommended)
```bash
npm run dev              # DB + backend + frontend with smart port selection
npm run dev:frontend-only
npm run dev:backend-only
npm run dev:quick        # Skip DB checks
npm run dev:manual       # Manual port selection

# Port helpers
node scripts/port-manager.js get
node scripts/port-manager.js allocate
```

### Build & production
```bash
npm run build            # Build frontend + backend
npm run build:frontend
npm run build:backend
npm run rebuild:backend
npm run start            # Start production server
npm run start:backend
```

### Testing
```bash
npm run test
npm run test:backend
```

### Database management
```bash
npm run db:start
npm run db:stop
npm run db:restart
npm run db:logs
npm run db:push          # Push schema changes
npm run db:migrate       # Create/apply migrations
npm run db:studio        # Prisma Studio
npm run db:generate      # Generate Prisma client
npm run db:adminer       # Launch Adminer
npm run setup-db
npm run migrate:backend

# Shared schema helpers
npx prisma db pull
npx prisma db push --force-reset   # Development reset
```

### Code quality
```bash
npm run type-check
npm run lint
```

### System checks
```bash
npm run check-docker
npm run test-services
```

## 📊 Data Model
- **User**: account and auth info
- **Server**: MCP server instance
- **ServerUser**: user-to-server membership
- **AccessToken**: API tokens
- **Tool**: tool configuration
- **ServerMetric**: performance metrics
- **Activity**: user activity logs

### Roles
- **Owner**: full control
- **Admin**: manage users/config
- **Member**: limited access

## API Endpoints

### Canonical (use for new integrations)

| Endpoint | Method | Purpose |
|---|---|---|
| `/api/external/*` | REST | External REST API for automation and third-party clients |
| `/api/auth/*` | REST | Authentication (login, register, token refresh) |

### Legacy (frozen — no new features)

| Endpoint | Method | Purpose |
|---|---|---|
| `/api/v1` | POST | Internal `cmdId`-based protocol. Being migrated to REST. |

> **For contributors:** New features MUST use RESTful endpoints under `/api/external/*`. The `cmdId`-based `/api/v1` protocol is frozen. See [handlers README](./app/api/v1/handlers/README.md) for migration guidance.

Detailed API references:
- [External REST API](./app/api/external/API.md)
- [Internal v1 Protocol API](./app/api/v1/API.md) (legacy)

## 🌐 Cloudflare Tunnel
Expose services securely without opening firewall ports.
```bash
npm run cloudflared:setup
npm run cloudflared:start
npm run cloudflared:logs
```
See [DOCKER_USAGE.md](./DOCKER_USAGE.md) and [DEPLOYMENT.md](./DEPLOYMENT.md) for deployment notes.

## 🚨 Troubleshooting
1) **Unified dev issues**
```bash
npm run dev:backend-only
npm run dev:frontend-only
npm run dev:manual
```
- Smart ports avoid conflicts; check with `node scripts/port-manager.js get`
- Ensure Docker is running and the DB is reachable
- Run `npm install` if dependencies look broken

2) **Docker issues**
```bash
npm run check-docker
```
- Install Docker Desktop if missing
- Start Docker Desktop if not running

3) **Database connection failures**
- Verify Docker status and port 5432 availability
- Confirm `DATABASE_URL`
- If Docker isn’t available, use `npm run dev:frontend-only`

4) **Backend issues**
- Ports are auto-assigned; override with `BACKEND_PORT=4001 npm run dev`
- Check backend logs and Prisma connectivity

5) **Email delivery**
- Verify SMTP settings and app passwords

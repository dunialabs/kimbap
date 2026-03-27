# Kimbap Console

Kimbap Console — Operations Console for the Kimbap platform. A web app for audit, logs, approvals, and policy management.

## 🚀 Features
- 🔐 **Secure auth** – email verification with JWT
- 👥 **Role-based users** – Owner, Admin, Member permissions
- 🔑 **Token management** – create and manage API access tokens
- 🛠️ **Tool configuration** – configure tools and permissions
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
console/
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
curl -O https://raw.githubusercontent.com/dunialabs/kimbap/main/console/docker-compose.yml

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


#### 💻 Option 2: Local development

**Requirements**
- Node.js 18+
- Docker & Docker Compose (for Postgres)
- Git

**Setup**
1) Clone and install
```bash
git clone https://github.com/dunialabs/kimbap.git
cd kimbap/console
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
KIMBAP_CORE_URL="http://localhost:3002"
```

3) Start dev environment
```bash
npm run dev
```
This will:
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
npm run dev              # DB + frontend (full dev)
npm run dev:quick        # Skip DB checks
npm run dev:frontend-only  # UI only, no DB setup
```

### Build & production
```bash
npm run build            # Build for production
npm run start            # Start production server
npm run start:production # Start with explicit port
```

### Testing
```bash
npm run test:log-sync        # Test log sync pipeline
npm run test:openapi-parity  # Check OpenAPI converter parity
```

### Database management
```bash
npm run db:start         # Start PostgreSQL (Docker)
npm run db:stop          # Stop PostgreSQL
npm run db:restart       # Restart PostgreSQL
npm run db:logs          # View PostgreSQL logs
npm run db:studio        # Prisma Studio (browser UI)
npm run db:generate      # Generate Prisma client
npm run db:adminer       # Launch Adminer
npm run db:init          # Initialize database
npm run db:reset         # Reset database (destructive)
npm run setup-db         # Run DB setup script
npm run migrate:backend  # Run Prisma migrations
```

### Code quality
```bash
npm run type-check
npm run lint
```

### Docker
```bash
npm run docker:deploy:auto   # Auto-assign ports and deploy
npm run docker:find-ports    # Find available ports
npm run docker:up            # Start Docker stack
npm run docker:down          # Stop Docker stack
npm run docker:logs          # View Docker logs
npm run docker:build         # Build Docker image
```

### System checks
```bash
npm run check-docker
```

## 📊 Data Model
- **User**: account and auth info
- **AccessToken**: API tokens

See `prisma/schema.prisma` for the current schema.

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

## 🚨 Troubleshooting
1) **Unified dev issues**
```bash
npm run dev:frontend-only
```
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

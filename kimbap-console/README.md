# Kimbap Console

KIMBAP MCP (Model Context Protocol) Console is a web app for managing and monitoring MCP servers.

## 🚀 Features
- 🔐 **Secure auth** – email verification with JWT
- 👥 **Role-based users** – Owner, Admin, Member permissions
- 🔑 **Token management** – create and manage API access tokens
- 🛠️ **Tool configuration** – configure MCP tools and permissions
- 📊 **Live monitoring** – real-time server performance and usage
- 💾 **Backup and restore** – cloud and local data recovery

## 🛠️ Tech Stack
- **Frontend:** Next.js 15 + TypeScript + Tailwind CSS
- **Backend:** Express + TypeScript + MCP Protocol
- **Database:** PostgreSQL + Prisma ORM (shared across frontend/backend)
- **Auth:** JWT + bcrypt
- **UI:** Radix UI + Shadcn/ui
- **Tooling:** Docker + Docker Compose

## 📁 Project Structure
```
kimbap-console/
├── app/                    # Next.js App Router
│   ├── (auth)/             # Auth pages
│   ├── (dashboard)/        # Dashboard pages
│   └── api/                # API routes
│       ├── auth/           # Auth APIs
│       ├── servers/        # Server management
│       ├── tokens/         # Token management
│       └── dashboard/      # Dashboard data
├── backend-src/            # Backend source (TypeScript)
│   ├── src/                # Backend TypeScript
│   │   ├── config/         # Config
│   │   ├── core/           # Core MCP proxy logic
│   │   ├── entities/       # Database entities
│   │   ├── middleware/     # Middleware
│   │   └── security/       # Security helpers
│   ├── scripts/            # Migration scripts
│   └── tsconfig.json       # Backend TypeScript config
├── proxy-server/           # Compiled backend (JavaScript)
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
curl -O https://raw.githubusercontent.com/your-org/kimbap-console/main/docker-compose.yml

docker compose up -d
# Frontend: http://localhost:3000
# Backend:  http://localhost:3002
```

**Auto port allocation:**
```bash
npm run docker:deploy:auto
# or find and apply available ports
npm run docker:find-ports
docker compose --env-file .env.ports up -d
```

See the [Docker deployment guide](./docs/DOCKER_DEPLOYMENT.md) for details.

#### 💻 Option 2: Local development

**Requirements**
- Node.js 18+
- Docker & Docker Compose (for Postgres)
- Git

**Setup**
1) Clone and install
```bash
git clone <repository-url>
cd kimbap-console
npm install
```

2) Configure env vars
```bash
cp .env.example .env.local
```
Edit `.env.local` as needed:
```env
DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"
JWT_SECRET="your-super-secret-jwt-key"
# Optional mail settings
SMTP_HOST="smtp.gmail.com"
SMTP_PORT="587"
SMTP_USER="your-email@gmail.com"
SMTP_PASS="your-app-password"
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
- 🚀 Backend API: http://localhost:3002
- 🗄️ Adminer: http://localhost:8080

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

## 🔗 API Endpoints
- Auth: `POST /api/auth/login/email`, `POST /api/auth/login/verify`, `POST /api/auth/master-password/set`
- Servers: `GET /api/servers`, `POST /api/servers`, `DELETE /api/servers/:id`
- Tokens: `GET /api/tokens`, `POST /api/tokens`, `DELETE /api/tokens/:id`
- Dashboard: `GET /api/dashboard/overview`, `GET /api/dashboard/metrics`

## 🌐 Cloudflare Tunnel
Expose services securely without opening firewall ports.
```bash
npm run cloudflared:setup
npm run cloudflared:start
npm run cloudflared:logs
```
See [Quick Start](./docs/CLOUDFLARED_QUICK_START.md) and [Full setup](./docs/CLOUDFLARED_SETUP.md).

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

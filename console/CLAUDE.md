# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Development
- `npm run dev` - Start development environment (SQLite + Next.js with API routes)
- `npm run dev:frontend-only` - Start only Next.js without database (for external DB)
- `npm run dev:custom` - Start with custom server configuration
- `npm run build` - Build for production
- `npm run start` - Start production server
- `npm run start:custom` - Start production server with custom configuration

### Database Management
- `npm run db:push` - Sync Prisma schema to SQLite database
- `npm run db:studio` - Open Prisma Studio for database management
- `npm run db:generate` - Generate Prisma client
- `npm run db:reset` - Reset database (warning: data loss)
- `npm run db:reset:complete` - Full database reset (delete file + re-push)

### Code Quality
- `npm run type-check` - Run TypeScript type checking for frontend
- `npm run lint` - Run ESLint to check code quality

### Docker Deployment
- `npm run docker:build` - Build Docker image locally
- `npm run docker:up` - Start services using docker-compose
- `npm run docker:down` - Stop all Docker services
- `npm run docker:logs` - View container logs
- `npm run docker:push` - Push image to Docker Hub
- `npm run docker:deploy:auto` - Deploy with automatic port allocation

### Standalone Deployment
- `npm run build:complete` - Build standalone package for current platform
- `npm run build:complete:x64` - Build for x64 architecture
- `npm run build:linux:x64` - Build for Linux x64
- `npm run build:windows:x64` - Build for Windows x64
- `npm run build:all` - Build standalone packages for all platforms

## Architecture

This is the Kimbap Console — operations and observability console for the Kimbap platform, built with Next.js.

### Technology Stack
- **Frontend/Backend**: Next.js 15 with App Router, TypeScript, Tailwind CSS
- **API Routes**: Next.js API routes (`/app/api/*`) for all backend operations
- **Database**: SQLite (default, embedded) via Prisma ORM
- **Authentication**: JWT with bcrypt password hashing
- **Development**: SQLite by default (zero config)

### Project Structure
- **Next.js Application**:
  - `/app/(auth)` - Authentication pages (login, master password)
  - `/app/(dashboard)` - Dashboard and sub-pages (billing, logs, members)
  - `/app/(server-management)` - Server creation and connection
  - `/app/api/*` - API routes for all backend operations (protocols, database, etc.)
  - `/lib` - Shared libraries and utilities
  - `/components` - React components

### Key Patterns
- **Unified Application**: Single Next.js application with API routes
- **Database Management**: SQLite via `prisma db push` (schema sync)
- **Protocol Handlers**: API routes handle MCP protocols (10001-10020)
- **Authentication**: JWT-based authentication with role-based access control
- **Role-Based Access**: Owner, Admin, and Member roles with different permissions

### Database Models (Prisma)
- `User` - User accounts with roles and permissions
- `Log` - Activity logging with server_id reference
- `Config` - Kimbap Core host/port configuration
- `TokenMetadata` - Token namespace and tags
- `RateLimitBucket` - API rate limiting state
- `RuntimeLock` - Distributed lock for background jobs
- `License` - License management

### API Routes Structure (`/app/api/`)
- `/app/api/external/*` - **Canonical** RESTful API endpoints (use for all new features)
- `/app/api/auth/*` - Authentication endpoints (login, register, token refresh)
- `/app/api/v1` - **Legacy (frozen)** cmdId-based protocol handler; no new handlers should be added
- `/app/api/v1/handlers/` - Legacy protocol handlers (see README.md in that directory for migration guidance)

### Port Configuration
- Application default: 3000 (can be configured via PORT environment variable)

### Database
- SQLite is the default — `npm run dev` works with zero config (`file:./data/kimbap-console.db`)
- Schema sync via `prisma db push`; no migration history for SQLite
- All raw SQL in handlers is portable (standard SQL only)
- Configure via `DATABASE_URL` in `.env.local` or environment

### Important Notes
- All backend logic is implemented in Next.js API routes (`/app/api/*`)
- Schema changes: edit `prisma/schema.prisma`, then `npm run db:push`
- Use `npm run db:studio` to inspect and manage database via Prisma Studio
- Legacy protocol handlers are in `/app/api/v1/handlers/` (frozen — new features go to `/app/api/external/*`)
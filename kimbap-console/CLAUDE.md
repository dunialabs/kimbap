# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Development
- `npm run dev` - Start development environment (PostgreSQL + Next.js with API routes)
- `npm run dev:frontend-only` - Start only Next.js without database (for external DB)
- `npm run dev:custom` - Start with custom server configuration
- `npm run build` - Build for production
- `npm run start` - Start production server
- `npm run start:custom` - Start production server with custom configuration

### Database Management
- `npm run db:start` - Start PostgreSQL database using Docker
- `npm run db:stop` - Stop all database services
- `npm run db:push` - Push Prisma schema changes to PostgreSQL
- `npm run db:migrate:create` - Create new database migration
- `npm run db:migrate:deploy` - Deploy database migrations
- `npm run db:studio` - Open Prisma Studio for database management
- `npm run db:generate` - Generate Prisma client
- `npm run db:reset` - Reset database (warning: data loss)

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
- `npm run build:complete` - Build standalone package for current platform (with embedded PostgreSQL)
- `npm run build:complete:x64` - Build for x64 architecture
- `npm run build:linux:x64` - Build for Linux x64
- `npm run build:windows:x64` - Build for Windows x64
- `npm run build:all` - Build standalone packages for all platforms

## Architecture

This is the Kimbap Console — operations and observability console for the Kimbap platform, built with Next.js.

### Technology Stack
- **Frontend/Backend**: Next.js 15 with App Router, TypeScript, Tailwind CSS
- **API Routes**: Next.js API routes (`/app/api/*`) for all backend operations
- **Database**: PostgreSQL via Prisma ORM
- **Authentication**: JWT with bcrypt password hashing
- **Development**: Dockerized PostgreSQL for local setup

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
- **Database Management**: PostgreSQL via Docker and Prisma migrations
- **Protocol Handlers**: API routes handle MCP protocols (10001-10020)
- **Authentication**: JWT-based authentication with role-based access control
- **Role-Based Access**: Owner, Admin, and Member roles with different permissions

### Database Models (Prisma)
- `User` - User accounts with roles and permissions
- `Server` - MCP server instances with configurations
- `Proxy` - Proxy configurations
- `Event` - MCP event storage
- `Log` - Activity logging with server_id reference
- `License` - License management
- `IpWhitelist` - IP access control

### API Routes Structure (`/app/api/`)
- `/app/api/external/*` - **Canonical** RESTful API endpoints (use for all new features)
- `/app/api/auth/*` - Authentication endpoints (login, register, token refresh)
- `/app/api/v1` - **Legacy (frozen)** cmdId-based protocol handler; no new handlers should be added
- `/app/api/v1/handlers/` - Legacy protocol handlers (see README.md in that directory for migration guidance)

### Port Configuration
- Application default: 3000 (can be configured via PORT environment variable)
- PostgreSQL: 5432
- Adminer: 8080 (optional, for database management)

### Database
- PostgreSQL is required for the local runtime; start it with `docker compose up -d postgres`
- Configure via `DATABASE_URL` in `.env.local` or environment

### Important Notes
- All backend logic is implemented in Next.js API routes (`/app/api/*`)
- Database migrations managed through Prisma CLI (`npm run db:migrate:create`)
- Use `npm run db:studio` to inspect and manage database via Prisma Studio
- Legacy protocol handlers are in `/app/api/v1/handlers/` (frozen — new features go to `/app/api/external/*`)
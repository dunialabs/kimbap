# Repository Guidelines

## Project Structure & Module Organization
- `app/` houses the Next.js App Router (auth, dashboard) plus API routes under `app/api`.
- `components/`, `contexts/`, and `hooks/` share UI atoms, shared state, and reusable hooks; `components/ui/` contains Shadcn/Radix primitives.
- `lib/` holds utilities such as the API client, auth helpers, Prisma client, and shared types (import via the `@/` alias).
- `prisma/` stores the Prisma schema and migrations; `public/` keeps static assets.
- `scripts/` covers database setup, port management, and build helpers; `docker/` and the various `docker-compose*.yml` files define local/production stacks.
- `proxy-server/` contains the compiled backend bundle; `docs/` holds deployment and configuration guides.

## Architecture Overview & Runtime
- Single Next.js 15 App Router app with co-located API routes under `app/api`. New API features go under `app/api/external/*` as RESTful endpoints. The legacy `app/api/v1/handlers/*` cmdId-based protocol system is **frozen** — no new handlers should be added there (see `app/api/v1/handlers/README.md` for migration guidance).
- Custom entry `server.js` wraps Next for HTTP/HTTPS with graceful shutdown; override ports via `PORT`, `FRONTEND_PORT`, `FRONTEND_HTTPS_PORT`.
- Default ports: app `3000`, Postgres `5432`, optional Adminer `8080`; auto-port scripts help avoid conflicts.

## Build, Test, and Development Commands
- `npm run dev` boots Dockerized Postgres (if needed) and starts the unified dev server (backend proxy + Next.js).
- `npm run dev:frontend-only` runs only the UI against an external DB; `npm run dev:quick` skips DB prep when you already have a database.
- `npm run build` creates the production bundle; `npm run start:production` (or `npm run start`) initializes the DB if required and serves the built app.
- Quality gates: `npm run type-check` and `npm run lint` should be clean before PRs.
- Database helpers: `npm run db:start|db:stop|db:reset|db:studio|db:generate`; `npm run docker:deploy:auto` spins up the stack with auto-assigned ports.
- Tests: Jest/ts-jest deps are present; place `*.test.ts(x)` beside code and run with `npx jest` (add a script if desired). `npm run test:log-sync` checks the log sync pipeline.
- Packaging: `npm run build:complete` produces a standalone bundle with embedded Postgres; platform-specific variants (`build:complete:x64`, `build:linux:x64`, etc.) live under `scripts/`.

## Coding Style & Naming Conventions
- TypeScript-first with functional React components; use `*.tsx` for UI and `*.ts` for utilities/server code.
- 2-space indentation, camelCase for variables/functions, PascalCase for components/types; keep filenames kebab- or lowerCamel to match existing folders.
- Favor Tailwind utilities and reuse primitives from `components/ui`; use helpers like `cn` for consistent class composition.
- Run `next lint` before pushing; prefer named exports and keep shared logic in `lib/` instead of duplicating across pages/components.

## Testing Guidelines
- Use Jest with `ts-jest` for unit/integration coverage; co-locate tests as `*.test.ts(x)` next to the module.
- Mock network (axios) calls; for Prisma/DB tests, ensure Docker Postgres is running (`npm run db:start`) to keep runs deterministic.
- Prefer assertion-based tests over heavy snapshots; document any required env vars in the test header.
- Add regression tests alongside bug fixes to lock behavior.

## Commit & Pull Request Guidelines
- Follow the conventional style seen in history: `feat: …`, `fix: …`, `chore: …`, `refactor: …` (scope optional).
- Squash noisy WIP commits; messages should describe the behavior change.
- PRs should include a concise summary, tests performed, schema/migration notes, and any env-var changes; attach UI screenshots/GIFs for visual updates.
- Link related issues/tickets and ensure `lint`, `type-check`, and relevant tests pass before requesting review.

## Security & Configuration Tips
- Keep secrets in `.env.local` (`DATABASE_URL`, `JWT_SECRET`, SMTP creds, SSL paths); do not commit PEM/key material.
- After editing `prisma/schema.prisma`, run `npm run db:generate` and include migrations; verify compose port choices when using auto-port scripts.

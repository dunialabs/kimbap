# Migration History Note

This directory contains timestamped SQL migration files that were generated when kimbap-console used PostgreSQL.

**These SQL files are historical artifacts and are NOT executed.**

The project has migrated to SQLite and uses `prisma db push` exclusively:
- `npm run migrate:backend` → runs `prisma db push`
- `npm run db:push` → runs `prisma db push`

The `migration_lock.toml` reflects the current provider: `sqlite`.

Do not run `prisma migrate deploy` — it will fail because the SQL files contain PostgreSQL-only syntax.

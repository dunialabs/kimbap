# Database Migration Guide

## Overview

KIMBAP Console uses a unified database migration system based on Prisma Migrate. This system automatically handles both new installations and upgrades, ensuring database consistency across all environments.

## Key Features

- **Automatic Detection**: Distinguishes between new installations and existing databases
- **Incremental Updates**: Only applies necessary migrations
- **Cross-Platform**: Works in Docker, local development, and production
- **Zero Configuration**: Users don't need to manage migrations manually

## For Developers

### Creating a New Migration

When you need to modify the database schema:

1. **Edit the Prisma Schema**
   ```bash
   # Edit prisma/schema.prisma
   ```

2. **Create a Migration**
   ```bash
   npm run db:migrate:create -- --name your_change_description
   ```
   This generates a new migration file in `prisma/migrations/`

3. **Test the Migration**
   ```bash
   npm run db:init:verbose
   ```
   This runs the migration in verbose mode for debugging

4. **Commit the Migration**
   ```bash
   git add prisma/migrations/
   git commit -m "Add migration: your_change_description"
   ```

### Important Notes

- **Never edit migration files manually** after they're created
- **Always test migrations** on a fresh database before committing
- **Name migrations descriptively** (e.g., `add_user_email_field`, `remove_deprecated_table`)
- **Sequential migrations**: If you add then remove a field before committing, create a single migration that represents the final state

## For Users

### New Installation

When starting KIMBAP Console for the first time:

```bash
# Using Docker
docker-compose up

# Using npm
npm run dev
```

The database will be automatically initialized with all tables and indexes.

### Upgrading

When updating to a new version:

```bash
# Pull latest changes
git pull

# Start the application - migrations run automatically
npm run dev
# or
docker-compose up
```

The system will:
1. Detect existing database
2. Check for pending migrations
3. Apply only new migrations
4. Start the application

### Troubleshooting

If you encounter migration issues:

```bash
# Run migrations in verbose mode
npm run db:init:verbose

# Reset database (WARNING: Deletes all data!)
npm run db:reset

# View migration status
npx prisma migrate status
```

## How It Works

### Migration Flow

1. **Application Start**
   - `unified-db-init.js` runs automatically
   - Checks database connection
   - Detects if database is new or existing

2. **New Database**
   - Applies all migrations from the beginning
   - Creates `_prisma_migrations` table
   - Records migration history

3. **Existing Database**
   - Reads `_prisma_migrations` table
   - Compares with local migration files
   - Applies only pending migrations

### File Structure

```
prisma/
├── schema.prisma              # Database schema definition
└── migrations/                # Migration history
    ├── 20241201000000_initial_baseline/
    │   └── migration.sql
    ├── 20241201120000_add_feature_x/
    │   └── migration.sql
    └── migration_lock.toml    # Prevents concurrent migrations
```

### Environment Variables

- `DB_INIT_VERBOSE=true` - Enable verbose logging for debugging
- `DATABASE_URL` - Database connection string

## Migration Commands

| Command | Description |
|---------|------------|
| `npm run db:init` | Run migrations (silent mode) |
| `npm run db:init:verbose` | Run migrations with detailed output |
| `npm run db:migrate:create` | Create a new migration |
| `npm run db:migrate:deploy` | Apply migrations (production) |
| `npm run db:reset` | Reset database (deletes all data) |
| `npm run db:studio` | Open Prisma Studio GUI |

## Best Practices

1. **Always use Prisma Migrate** - Don't modify database directly
2. **Test migrations locally** - Before pushing to production
3. **Keep migrations small** - One logical change per migration
4. **Document breaking changes** - In commit messages and release notes
5. **Backup before major updates** - Especially in production

## Common Scenarios

### Adding a New Table

```prisma
// prisma/schema.prisma
model NewFeature {
  id        Int      @id @default(autoincrement())
  name      String
  createdAt DateTime @default(now())
}
```

```bash
npm run db:migrate:create -- --name add_new_feature_table
```

### Adding a Field

```prisma
model User {
  // existing fields...
  newField String? // Add nullable field first
}
```

```bash
npm run db:migrate:create -- --name add_new_field_to_user
```

### Removing a Field

```prisma
model User {
  // Remove the field from schema
  // oldField String <- deleted
}
```

```bash
npm run db:migrate:create -- --name remove_old_field_from_user
```

## Docker Integration

The Docker image includes automatic migration on startup:

```dockerfile
# docker/docker-entrypoint.sh
node /app/scripts/unified-db-init.js
```

This ensures migrations run before the application starts, preventing version mismatches.

## Security Considerations

- Migrations run with database admin privileges
- Always review migration SQL before applying
- Use read-only credentials for application runtime
- Keep migration files in version control
- Don't store sensitive data in migrations

## Support

If you encounter issues:
1. Check the [troubleshooting section](#troubleshooting)
2. Run migrations in verbose mode
3. Check Prisma documentation: https://www.prisma.io/docs/
4. Open an issue on GitHub with migration logs
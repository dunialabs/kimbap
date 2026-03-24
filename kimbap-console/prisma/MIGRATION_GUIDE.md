# Database Migration Guide

## Recommended Workflow

### ✅ Use Prisma Schema for Automatic Migrations

**Always define database changes in `schema.prisma` and let Prisma generate migrations automatically.**

Example - Adding indexes:
```prisma
model Log {
  id       Int    @id @default(autoincrement())
  proxyKey String @db.VarChar(64)
  addtime  BigInt @default(0)

  // Single column index
  @@index([proxyKey])

  // Composite index with custom name
  @@index([proxyKey, addtime(sort: Desc)], map: "idx_log_proxy_key_addtime")

  @@map("log")
}
```

Then generate migration:
```bash
npm run db:migrate:create
```

This ensures:
- ✅ Migrations are compatible with Prisma's transaction handling
- ✅ No manual SQL needed
- ✅ Type-safe database schema
- ✅ Automatic migration generation

## Important Rules

### ❌ DO NOT Use `CREATE INDEX CONCURRENTLY`

Prisma runs migrations inside transactions. PostgreSQL's `CREATE INDEX CONCURRENTLY` cannot run inside transaction blocks.

**Wrong:**
```sql
-- This will FAIL
CREATE INDEX CONCURRENTLY "idx_name" ON "table" ("column");
```

**Wrong:**
```sql
-- This will also FAIL
DO $$
BEGIN
  CREATE INDEX CONCURRENTLY "idx_name" ON "table" ("column");
END $$;
```

**Correct:**
```sql
-- Use this instead
CREATE INDEX IF NOT EXISTS "idx_name" ON "table" ("column");
```

### Error Message You'll See

```
Error: P3018
Database error code: 25001
Database error:
ERROR: CREATE INDEX CONCURRENTLY cannot run inside a transaction block
```

### When You Need Non-Blocking Index Creation

If you absolutely need `CONCURRENTLY` for large tables in production:

1. Create the migration without the index
2. Apply the migration
3. Manually run the index creation outside of Prisma:

```bash
psql -d your_database -c "CREATE INDEX CONCURRENTLY \"idx_name\" ON \"table\" (\"column\");"
```

## Best Practices

1. **Use `IF NOT EXISTS`** - Makes migrations idempotent
   ```sql
   CREATE INDEX IF NOT EXISTS "idx_name" ON "table" ("column");
   ```

2. **Keep migrations simple** - One logical change per migration

3. **Test migrations locally** - Always run `npm run db:reset` before pushing

4. **Never edit applied migrations** - Create a new migration to fix issues

## Recovery from Failed Migrations

If a migration fails:

1. Fix the migration SQL file
2. Mark it as resolved:
   ```bash
   npx prisma migrate resolve --applied <migration_name>
   ```
3. Test with:
   ```bash
   npm run db:reset
   ```

-- AddLogPerformanceIndexes
-- This migration adds performance indexes for the log table

-- 1. Create composite index on proxy_key and addtime (most important for queries)
CREATE INDEX IF NOT EXISTS "idx_log_proxy_key_addtime" ON "log" ("proxy_key", "addtime" DESC);

-- 2. Create composite index on proxy_key and id_in_core (for incremental sync)
CREATE INDEX IF NOT EXISTS "idx_log_proxy_key_id_in_core" ON "log" ("proxy_key", "id_in_core" DESC);

-- 3. Create index on id_in_core (for batch duplicate checking)
CREATE INDEX IF NOT EXISTS "idx_log_id_in_core" ON "log" ("id_in_core");

-- 4. Create composite index on proxy_key and status_code (for log level filtering)
CREATE INDEX IF NOT EXISTS "idx_log_proxy_key_status_code" ON "log" ("proxy_key", "status_code");

-- 5. Create composite index on proxy_key and userid (for user filtering)
CREATE INDEX IF NOT EXISTS "idx_log_proxy_key_userid" ON "log" ("proxy_key", "userid");

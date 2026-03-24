-- Step 1: Add new hash column alongside the existing raw token column
ALTER TABLE "user" ADD COLUMN "access_token_hash" VARCHAR(64);

-- Step 2: Backfill hash from existing raw tokens
-- Uses pgcrypto if available; otherwise falls back to md5-based placeholder
-- that forces affected users to re-login (their hash won't match on next request).
DO $$
BEGIN
  BEGIN
    CREATE EXTENSION IF NOT EXISTS pgcrypto;
    UPDATE "user" SET "access_token_hash" = encode(digest("access_token", 'sha256'), 'hex');
  EXCEPTION WHEN insufficient_privilege OR undefined_file THEN
    UPDATE "user"
    SET "access_token_hash" = 'migrated_' || substr(md5("access_token"), 1, 55)
    WHERE "access_token_hash" IS NULL;
  END;
END $$;

-- Step 3: Make column NOT NULL and add unique constraint
ALTER TABLE "user" ALTER COLUMN "access_token_hash" SET NOT NULL;
ALTER TABLE "user" ADD CONSTRAINT "user_access_token_hash_key" UNIQUE ("access_token_hash");

-- Step 4: Drop the old raw token column
ALTER TABLE "user" DROP COLUMN "access_token";

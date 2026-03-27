-- Step 1: Add new hash column alongside the existing raw token column
ALTER TABLE "user" ADD COLUMN "access_token_hash" VARCHAR(64);

-- Step 2: Backfill hash from existing raw tokens
DO $$
BEGIN
  BEGIN
    CREATE EXTENSION IF NOT EXISTS pgcrypto;
  EXCEPTION WHEN insufficient_privilege OR undefined_file THEN
    RAISE EXCEPTION 'Cannot migrate user.access_token to access_token_hash without pgcrypto extension support';
  END;

  UPDATE "user"
  SET "access_token_hash" = encode(digest("access_token", 'sha256'), 'hex')
  WHERE "access_token_hash" IS NULL;
END $$;

-- Step 3: Make column NOT NULL and add unique constraint
ALTER TABLE "user" ALTER COLUMN "access_token_hash" SET NOT NULL;
ALTER TABLE "user" ADD CONSTRAINT "user_access_token_hash_key" UNIQUE ("access_token_hash");

-- Step 4: Drop the old raw token column
ALTER TABLE "user" DROP COLUMN "access_token";

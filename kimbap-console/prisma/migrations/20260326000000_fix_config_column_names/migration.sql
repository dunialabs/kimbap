DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'config'
      AND column_name = 'peta_core_host'
  ) AND NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'config'
      AND column_name = 'kimbap_core_host'
  ) THEN
    EXECUTE 'ALTER TABLE "public"."config" RENAME COLUMN "peta_core_host" TO "kimbap_core_host"';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'config'
      AND column_name = 'peta_core_prot'
  ) AND NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'config'
      AND column_name = 'kimbap_core_port'
  ) THEN
    EXECUTE 'ALTER TABLE "public"."config" RENAME COLUMN "peta_core_prot" TO "kimbap_core_port"';
  END IF;
END $$;

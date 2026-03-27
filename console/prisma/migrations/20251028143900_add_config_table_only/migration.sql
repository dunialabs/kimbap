-- CreateTable
-- Config table for storing PETA Core host and port configuration
CREATE TABLE IF NOT EXISTS "public"."config" (
    "id" SERIAL NOT NULL,
    "peta_core_host" VARCHAR(256) NOT NULL DEFAULT '',
    "peta_core_prot" INTEGER NOT NULL DEFAULT 80,

    CONSTRAINT "config_pkey" PRIMARY KEY ("id")
);

-- No default configuration will be inserted
-- The config will be populated when user configures PETA Core host/port via protocol-10021 or protocol-10022
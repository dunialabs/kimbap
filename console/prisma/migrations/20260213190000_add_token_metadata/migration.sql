-- CreateTable
CREATE TABLE "token_metadata" (
    "proxy_id" INTEGER NOT NULL,
    "userid" VARCHAR(256) NOT NULL,
    "namespace" VARCHAR(64) NOT NULL DEFAULT 'default',
    "tags" TEXT NOT NULL DEFAULT '[]',
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "token_metadata_pkey" PRIMARY KEY ("proxy_id","userid")
);

-- CreateIndex
CREATE INDEX "token_metadata_proxy_id_idx" ON "token_metadata"("proxy_id");

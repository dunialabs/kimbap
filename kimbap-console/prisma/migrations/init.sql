-- CreateSchema
CREATE SCHEMA IF NOT EXISTS "public";

-- CreateTable
CREATE TABLE "public"."user" (
    "userid" VARCHAR(256) NOT NULL,
    "access_token" VARCHAR(256) NOT NULL,

    CONSTRAINT "user_pkey" PRIMARY KEY ("userid")
);

-- CreateTable
CREATE TABLE "public"."log" (
    "id" SERIAL NOT NULL,
    "addtime" BIGINT NOT NULL DEFAULT 0,
    "action" INTEGER NOT NULL DEFAULT 0,
    "userid" VARCHAR NOT NULL DEFAULT '',
    "server_id" VARCHAR(128),
    "session_id" VARCHAR NOT NULL DEFAULT '',
    "upstream_request_id" VARCHAR NOT NULL DEFAULT '',
    "uniform_request_id" VARCHAR,
    "parent_uniform_request_id" VARCHAR,
    "proxy_request_id" VARCHAR,
    "ip" VARCHAR NOT NULL DEFAULT '',
    "machine_code" VARCHAR NOT NULL DEFAULT '',
    "ua" VARCHAR NOT NULL DEFAULT '',
    "token_mask" VARCHAR NOT NULL DEFAULT '',
    "request_params" TEXT NOT NULL DEFAULT '',
    "response_result" TEXT NOT NULL DEFAULT '',
    "error" TEXT NOT NULL DEFAULT '',
    "duration" INTEGER,
    "status_code" INTEGER,

    CONSTRAINT "log_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "public"."dns_conf" (
    "subdomain" VARCHAR(128) NOT NULL DEFAULT '',
    "type" INTEGER NOT NULL DEFAULT 0,
    "public_ip" VARCHAR(128) NOT NULL DEFAULT '',
    "id" SERIAL NOT NULL,
    "addtime" INTEGER NOT NULL DEFAULT 0,
    "update_time" INTEGER NOT NULL DEFAULT 0,
    "tunnel_id" VARCHAR(256) NOT NULL DEFAULT '',

    CONSTRAINT "dns_conf_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "public"."license" (
    "id" SERIAL NOT NULL,
    "license_str" TEXT NOT NULL,
    "addtime" INTEGER NOT NULL,
    "status" INTEGER NOT NULL,

    CONSTRAINT "license_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "log_userid_idx" ON "public"."log"("userid");

-- CreateIndex
CREATE INDEX "log_session_id_idx" ON "public"."log"("session_id");

-- CreateIndex
CREATE INDEX "log_uniform_request_id_idx" ON "public"."log"("uniform_request_id");

-- CreateIndex
CREATE INDEX "log_server_id_idx" ON "public"."log"("server_id");

-- CreateIndex
CREATE INDEX "log_addtime_idx" ON "public"."log"("addtime");


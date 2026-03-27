/*
  Warnings:

  - You are about to drop the column `machine_code` on the `log` table. All the data in the column will be lost.
  - Added the required column `proxy_key` to the `license` table without a default value. This is not possible if the table is not empty.

*/
-- AlterTable
ALTER TABLE "public"."dns_conf" ADD COLUMN     "proxy_key" VARCHAR(64) NOT NULL DEFAULT '';

-- AlterTable
ALTER TABLE "public"."license" ADD COLUMN     "proxy_key" VARCHAR(64) NOT NULL DEFAULT '';

-- AlterTable
ALTER TABLE "public"."log" 
DROP COLUMN "machine_code",
ADD COLUMN     "id_in_core" BIGINT,
ADD COLUMN     "proxy_key" VARCHAR(64) NOT NULL DEFAULT '';
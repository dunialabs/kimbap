/*
  Warnings:

  - Added the required column `proxy_key` to the `user` table without a default value. This is not possible if the table is not empty.
  - Added the required column `role` to the `user` table without a default value. This is not possible if the table is not empty.

*/
-- AlterTable
ALTER TABLE "public"."license" ALTER COLUMN "proxy_key" DROP DEFAULT;

-- AlterTable
ALTER TABLE "public"."user" ADD COLUMN     "proxy_key" VARCHAR(64) NOT NULL,
ADD COLUMN     "role" INTEGER NOT NULL;

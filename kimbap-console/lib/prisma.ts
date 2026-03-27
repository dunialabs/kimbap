import type { PrismaClient as BasePrismaClient } from '@prisma/client'
import { isSQLite } from '@/lib/db/dialect'

const globalForPrisma = globalThis as unknown as {
  prisma: BasePrismaClient | undefined
}

type PrismaClientConstructor = new () => BasePrismaClient

function loadPrismaClient(): PrismaClientConstructor {
  if (isSQLite()) {
    return require('@prisma/client').PrismaClient as PrismaClientConstructor
  }

  return require('@kimbap/prisma-postgres-client').PrismaClient as PrismaClientConstructor
}

function createPrismaClient(): BasePrismaClient {
  const PrismaClient = loadPrismaClient()
  const client = new PrismaClient()

  if (isSQLite() && (process.env.DATABASE_URL ?? '').trim() !== '') {
    // WAL mode: better concurrent read performance for dashboard workloads
    client.$executeRawUnsafe('PRAGMA journal_mode = WAL')
      .catch((e: unknown) => console.warn('Failed to set WAL mode:', e))
    client.$executeRawUnsafe('PRAGMA busy_timeout = 5000')
      .catch((e: unknown) => console.warn('Failed to set busy_timeout:', e))
  }

  return client
}

export const prisma = globalForPrisma.prisma ?? createPrismaClient()

if (process.env.NODE_ENV !== 'production') globalForPrisma.prisma = prisma

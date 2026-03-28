import { NextRequest } from 'next/server';
import { prisma } from '@/lib/prisma';

const DEFAULT_RATE_WINDOW_MS = 60_000;
const STALE_SWEEP_INTERVAL_MS = 10 * 60_000;

let lastSweepMs = 0;

function isSqliteBusyError(error: unknown): boolean {
  const message = error instanceof Error ? error.message : String(error);
  return message.includes('database is locked') || message.includes('SQLITE_BUSY');
}

async function withSqliteRetry<T>(op: () => Promise<T>, maxAttempts = 3): Promise<T> {
  let attempt = 0;
  let lastError: unknown;

  while (attempt < maxAttempts) {
    try {
      return await op();
    } catch (error) {
      lastError = error;
      attempt += 1;

      if (!isSqliteBusyError(error) || attempt >= maxAttempts) {
        throw error;
      }

      const backoffMs = attempt * 25;
      await new Promise((resolve) => setTimeout(resolve, backoffMs));
    }
  }

  throw lastError instanceof Error ? lastError : new Error(String(lastError));
}

function maybeSweepStaleRows(nowMs: number): void {
  if (nowMs - lastSweepMs < STALE_SWEEP_INTERVAL_MS) return;
  lastSweepMs = nowMs;

  const staleBucketThreshold = BigInt(nowMs - DEFAULT_RATE_WINDOW_MS);
  const staleLockThreshold = BigInt(nowMs);

  prisma
    .$transaction([
      prisma.rateLimitBucket.deleteMany({ where: { resetAtMs: { lt: staleBucketThreshold } } }),
      prisma.runtimeLock.deleteMany({ where: { expiresAtMs: { lt: staleLockThreshold } } }),
    ])
    .catch((error: unknown) => {
      console.warn('[RequestGuard] Failed to sweep stale guard rows:', error);
    });
}

export function getBearerToken(request: NextRequest): string | null {
  const authHeader = request.headers.get('authorization') || request.headers.get('Authorization');
  if (!authHeader) return null;
  const [scheme, ...rest] = authHeader.trim().split(/\s+/);
  if (!scheme || scheme.toLowerCase() !== 'bearer') return null;
  const token = rest.join(' ').trim();
  return token || null;
}

export function getClientIdentity(request: NextRequest): string {
  const directIp = (request as NextRequest & { ip?: string }).ip?.trim();
  if (directIp) {
    return directIp;
  }

  const trustForwarded = process.env.TRUST_PROXY_HEADERS === 'true' || process.env.KIMBAP_TRUST_PROXY === 'true';
  if (!trustForwarded) {
    return 'unknown';
  }

  const forwarded = request.headers.get('x-forwarded-for')?.split(',')[0]?.trim()
    || request.headers.get('x-real-ip')?.trim();
  if (!forwarded) {
    return 'unknown';
  }

  const bracketedIpv6 = forwarded.match(/^\[([0-9a-fA-F:.]+)\](?::\d+)?$/);
  if (bracketedIpv6?.[1]) {
    return bracketedIpv6[1];
  }

  const ipv4WithPort = forwarded.match(/^((?:\d{1,3}\.){3}\d{1,3}):\d+$/);
  if (ipv4WithPort?.[1]) {
    return ipv4WithPort[1];
  }

  const hostnameWithPort = forwarded.match(/^([a-zA-Z0-9.-]+):\d+$/);
  if (hostnameWithPort?.[1]) {
    return hostnameWithPort[1];
  }

  const sanitizedForwarded = forwarded.trim();
  if (!/^[0-9a-zA-Z:.[\]%-]+$/.test(sanitizedForwarded)) {
    return 'unknown';
  }

  return sanitizedForwarded || 'unknown';
}

export async function checkRateLimitDb(
  key: string,
  limit: number,
  windowMs: number = DEFAULT_RATE_WINDOW_MS,
): Promise<boolean> {
  if (limit <= 0) {
    return true;
  }

  const nowMs = Date.now();
  maybeSweepStaleRows(nowMs);

  const resetAtMs = BigInt(nowMs + windowMs);

  return withSqliteRetry(async () => {
    return prisma.$transaction(async (tx) => {
      const existing = await tx.rateLimitBucket.findUnique({ where: { key } });

      if (!existing || Number(existing.resetAtMs) <= nowMs) {
        await tx.rateLimitBucket.upsert({
          where: { key },
          create: { key, count: 1, resetAtMs },
          update: { count: 1, resetAtMs },
        });
        return true;
      }

      const updated = await tx.rateLimitBucket.update({
        where: { key },
        data: { count: { increment: 1 } },
      });

      return updated.count <= limit;
    });
  });
}

export async function acquireRuntimeLock(lockKey: string, ttlMs: number): Promise<boolean> {
  const nowMs = Date.now();
  maybeSweepStaleRows(nowMs);

  const expiresAtMs = BigInt(nowMs + ttlMs);

  return withSqliteRetry(async () => {
    return prisma.$transaction(async (tx) => {
      const existing = await tx.runtimeLock.findUnique({ where: { key: lockKey } });
      if (existing && Number(existing.expiresAtMs) > nowMs) {
        return false;
      }

      await tx.runtimeLock.upsert({
        where: { key: lockKey },
        create: { key: lockKey, expiresAtMs },
        update: { expiresAtMs },
      });

      return true;
    });
  });
}

export async function releaseRuntimeLock(lockKey: string): Promise<void> {
  await withSqliteRetry(async () => {
    await prisma.runtimeLock.deleteMany({ where: { key: lockKey } });
  });
}

import { NextRequest } from 'next/server';
import { ApiResponse } from '@/lib/api-response';
import { getProtocolHandler } from './handlers';
import { prisma } from '@/lib/prisma';
import { hashToken } from '@/lib/auth';
import { getUserByAccessToken } from '@/lib/proxy-api';

export const dynamic = 'force-dynamic';
export const revalidate = 0;
export const runtime = 'nodejs';

const PUBLIC_CMD_IDS = new Set([10015]);

const LOGIN_RATE_LIMIT = 10;
const GENERAL_RATE_LIMIT = 120;
const RATE_WINDOW_MS = 60_000;

const rateBuckets = new Map<string, { count: number; resetAt: number }>();

function checkRateLimit(key: string, limit: number): boolean {
  const now = Date.now();
  const bucket = rateBuckets.get(key);
  if (!bucket || now >= bucket.resetAt) {
    rateBuckets.set(key, { count: 1, resetAt: now + RATE_WINDOW_MS });
    return true;
  }
  bucket.count++;
  return bucket.count <= limit;
}

if (typeof globalThis !== 'undefined') {
  setInterval(() => {
    const now = Date.now();
    rateBuckets.forEach((bucket, key) => {
      if (now >= bucket.resetAt) rateBuckets.delete(key);
    });
  }, RATE_WINDOW_MS * 2).unref?.();
}

function getBearerToken(request: NextRequest): string | null {
  const authHeader = request.headers.get('authorization') || request.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) return null;
  const token = authHeader.slice('Bearer '.length).trim();
  return token ? token : null;
}

function getClientIdentity(request: NextRequest): string {
  const directIp = (request as NextRequest & { ip?: string }).ip?.trim();
  if (directIp) {
    return directIp;
  }

  const trustForwarded = process.env.TRUST_PROXY_HEADERS === 'true' || process.env.KIMBAP_TRUST_PROXY === 'true';
  if (!trustForwarded) {
    return 'unknown';
  }

  const forwarded = request.headers.get('x-forwarded-for')?.split(',')[0]?.trim();
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

export async function POST(request: NextRequest) {
  let cmdId = 0;
  
  try {
    const body = await request.json();
    
    cmdId = body.common?.cmdId || 0;
    
    if (!cmdId) {
      return ApiResponse.missingCmdId();
    }

    const handler = getProtocolHandler(cmdId);
    
    if (!handler) {
      return ApiResponse.protocolNotImplemented(cmdId);
    }

    const isPublic = PUBLIC_CMD_IDS.has(cmdId);
    body.common = body.common || {};

    if (isPublic && 'rawToken' in body.common) {
      delete body.common.rawToken;
    }

    if (isPublic) {
      const clientIp = getClientIdentity(request);
      if (clientIp !== 'unknown' && !checkRateLimit(`login:${clientIp}`, LOGIN_RATE_LIMIT)) {
        return ApiResponse.error(cmdId, 429, 'Rate limit exceeded. Try again later.', 429);
      }
    }

    if (!isPublic) {
      const clientIp = getClientIdentity(request);
      if (clientIp !== 'unknown' && !checkRateLimit(`anon:${clientIp}`, GENERAL_RATE_LIMIT)) {
        return ApiResponse.error(cmdId, 429, 'Rate limit exceeded. Try again later.', 429);
      }

      const token = getBearerToken(request);
      if (!token) {
        return ApiResponse.unauthorized(cmdId);
      }

      const tokenPrefix = token.substring(0, 16);
      if (!checkRateLimit(`api:${tokenPrefix}`, GENERAL_RATE_LIMIT)) {
        return ApiResponse.error(cmdId, 429, 'Rate limit exceeded. Try again later.', 429);
      }

      const user = await prisma.user.findUnique({
        where: { accessTokenHash: hashToken(token) }
      });

      if (!user) {
        return ApiResponse.unauthorized(cmdId);
      }

      let resolvedRole = user.role;
      try {
        const upstreamUser = await getUserByAccessToken(user.userid, token);
        const currentTime = Math.floor(Date.now() / 1000);

        if (upstreamUser?.status !== 1) {
          return ApiResponse.unauthorized(cmdId);
        }

        if (typeof upstreamUser?.expiresAt === 'number' && upstreamUser.expiresAt > 0 && upstreamUser.expiresAt < currentTime) {
          return ApiResponse.unauthorized(cmdId);
        }

        if (typeof upstreamUser?.role === 'number') {
          resolvedRole = upstreamUser.role;
        }
      } catch {
        return ApiResponse.unauthorized(cmdId);
      }

      if (resolvedRole !== 1 && resolvedRole !== 2) {
        return ApiResponse.unauthorized(cmdId);
      }

      body.common.userid = user.userid;
      body.common.userRole = resolvedRole;
      body.common.rawToken = token;
    }

    const responseData = await handler(body);
    
    return ApiResponse.success(cmdId, responseData);
    
  } catch (error) {
    return ApiResponse.handleError(cmdId, error);
  }
}

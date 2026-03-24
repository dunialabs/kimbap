import { NextRequest } from 'next/server';
import { ApiResponse } from '@/lib/api-response';
import { getProtocolHandler } from './handlers';
import { prisma } from '@/lib/prisma';
import { hashToken } from '@/lib/auth';

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
  return request.headers.get('x-real-ip')
    || request.headers.get('x-forwarded-for')?.split(',')[0]?.trim()
    || 'unknown';
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

    if (isPublic) {
      const clientIp = getClientIdentity(request);
      if (!checkRateLimit(`login:${clientIp}`, LOGIN_RATE_LIMIT)) {
        return ApiResponse.error(cmdId, 429, 'Rate limit exceeded. Try again later.', 429);
      }
    }

    if (!isPublic) {
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

      body.common = body.common || {};
      body.common.userid = user.userid;
    }

    const responseData = await handler(body);
    
    return ApiResponse.success(cmdId, responseData);
    
  } catch (error) {
    return ApiResponse.handleError(cmdId, error);
  }
}

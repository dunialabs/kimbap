import { NextRequest } from 'next/server';
import { ApiResponse } from '@/lib/api-response';
import { getProtocolHandler } from './handlers';
import { prisma } from '@/lib/prisma';
import { hashToken } from '@/lib/auth';
import { getUserByAccessToken } from '@/lib/proxy-api';
import { checkRateLimitDb, getBearerToken, getClientIdentity } from '@/lib/request-guard';

export const dynamic = 'force-dynamic';
export const revalidate = 0;
export const runtime = 'nodejs';

const PUBLIC_CMD_IDS = new Set([10015]);

const LOGIN_RATE_LIMIT = 10;
const GENERAL_RATE_LIMIT = 120;

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
      const rateLimitKey = `login:${clientIp !== 'unknown' ? clientIp : `unknown:${cmdId}`}`;
      if (!await checkRateLimitDb(rateLimitKey, LOGIN_RATE_LIMIT)) {
        return ApiResponse.error(cmdId, 429, 'Rate limit exceeded. Try again later.', 429);
      }
    }

    if (!isPublic) {
      const clientIp = getClientIdentity(request);
      const rateLimitKey = `anon:${clientIp !== 'unknown' ? clientIp : `unknown:${cmdId}`}`;
      if (!await checkRateLimitDb(rateLimitKey, GENERAL_RATE_LIMIT)) {
        return ApiResponse.error(cmdId, 429, 'Rate limit exceeded. Try again later.', 429);
      }

      const token = getBearerToken(request);
      if (!token) {
        return ApiResponse.unauthorized(cmdId);
      }

      const tokenPrefix = token.substring(0, 16);
      if (!await checkRateLimitDb(`api:${tokenPrefix}`, GENERAL_RATE_LIMIT)) {
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
      } catch (err: unknown) {
        const status = (err as { status?: number })?.status;
        if (status === 401 || status === 403 || status === 404) {
          return ApiResponse.unauthorized(cmdId);
        }
        return ApiResponse.error(cmdId, 502, 'Upstream validation unavailable', 502);
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

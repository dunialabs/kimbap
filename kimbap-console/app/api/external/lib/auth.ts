import { NextRequest } from 'next/server';
import { prisma } from '@/lib/prisma';
import { ExternalApiError, E2001, E2002, E2003, E5005 } from './error-codes';
import { hashToken } from '@/lib/auth';
import { getUserByAccessToken } from '@/lib/proxy-api';

export interface AuthUser {
  userid: string;
  role: number; // 1=owner, 2=admin, 3=member
  accessToken: string;
  proxyKey: string;
}

/**
 * Extract and validate Bearer token from request
 * Returns the authenticated user info
 */
export async function authenticate(request: NextRequest): Promise<AuthUser> {
  const authHeader = request.headers.get('Authorization');

  if (!authHeader) {
    throw new ExternalApiError(E2001, 'Access token is required');
  }

  // Check Bearer token format
  if (!authHeader.startsWith('Bearer ')) {
    throw new ExternalApiError(E2002, 'Invalid access token');
  }

  const token = authHeader.slice(7); // Remove 'Bearer ' prefix

  if (!token) {
    throw new ExternalApiError(E2002, 'Invalid access token');
  }

  const user = await prisma.user.findUnique({
    where: { accessTokenHash: hashToken(token) },
  });

  if (!user) {
    throw new ExternalApiError(E2002, 'Invalid access token');
  }

  let resolvedRole = user.role;
  try {
    const upstreamUser = await getUserByAccessToken(user.userid, token);
    const currentTime = Math.floor(Date.now() / 1000);

    if (upstreamUser?.status !== 1) {
      throw new ExternalApiError(E2002, 'Invalid access token');
    }

    if (typeof upstreamUser?.expiresAt === 'number' && upstreamUser.expiresAt > 0 && upstreamUser.expiresAt < currentTime) {
      throw new ExternalApiError(E2002, 'Invalid access token');
    }

    if (typeof upstreamUser?.role === 'number') {
      resolvedRole = upstreamUser.role;
    }
  } catch (err: unknown) {
    if (err instanceof ExternalApiError) {
      throw err;
    }
    const status = (err as { status?: number })?.status;
    if (status === 401 || status === 403 || status === 404) {
      throw new ExternalApiError(E2002, 'Invalid access token');
    }
    // Fail closed: upstream validation errors must not silently fall back to stale local data.
    // A revoked/expired/disabled upstream user could still have a valid local token hash.
    throw new ExternalApiError(E5005, 'Upstream validation unavailable');
  }

  if (resolvedRole !== 1 && resolvedRole !== 2) {
    throw new ExternalApiError(E2003, 'Permission denied: only owner or admin can access external API');
  }

  return {
    userid: user.userid,
    role: resolvedRole,
    accessToken: token,
    proxyKey: user.proxyKey,
  };
}

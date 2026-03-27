import { NextRequest } from 'next/server';
import { getProxy, getUsers, getServers } from '@/lib/proxy-api';
import { getTokenMetadata } from '@/lib/token-metadata';
import { prisma } from '@/lib/prisma';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3003, E5005 } from '../../lib/error-codes';
import { getUserPermissions, PermissionsFetchError, TokenItem } from '../lib/permissions';

export const dynamic = 'force-dynamic';

interface GetTokenRequest {
  tokenId: string;
}

async function getTokenById(request: NextRequest, tokenIdRaw: string) {
  const user = await authenticate(request);
  const tokenId = tokenIdRaw.trim();
  if (!tokenId) {
    throw new ExternalApiError(E1001, 'Missing required field: tokenId');
  }

  const proxy = await getProxy();
  const { users } = await getUsers({ userId: tokenId, proxyId: proxy.id }, undefined, user.accessToken);

  const targetUser = users.find(u => u.userId === tokenId);
  if (!targetUser) {
    throw new ExternalApiError(E3003, `Token not found: ${tokenId}`);
  }

  const { servers } = await getServers({ proxyId: proxy.id }, undefined, user.accessToken);
  const serverNameMap: Record<string, string> = {};
  servers.forEach((server) => {
    serverNameMap[server.serverId] = server.serverName;
  });
  let permissions: Awaited<ReturnType<typeof getUserPermissions>>;
  let metadata: Awaited<ReturnType<typeof getTokenMetadata>>;
  let lastLog: { addtime: bigint } | null;
  try {
    [permissions, metadata, lastLog] = await Promise.all([
      getUserPermissions(targetUser.userId, user.accessToken, serverNameMap),
      getTokenMetadata(targetUser.userId),
      prisma.log.findFirst({
        where: { userid: targetUser.userId },
        orderBy: { addtime: 'desc' },
        select: { addtime: true },
      }),
    ]);
  } catch (error) {
    if (error instanceof PermissionsFetchError) {
      throw new ExternalApiError(E5005, 'Permission service unavailable');
    }
    throw error;
  }

  const token: TokenItem = {
    tokenId: targetUser.userId,
    name: targetUser.name,
    role: targetUser.role,
    notes: (targetUser as Record<string, unknown>).notes as string || '',
    lastUsed: lastLog ? Number(lastLog.addtime) : 0,
    createdAt: targetUser.createdAt || 0,
    expiresAt: targetUser.expiresAt || 0,
    rateLimit: targetUser.ratelimit,
    permissions,
    namespace: metadata.namespace,
    tags: metadata.tags,
  };

  return ApiResponse.success(token, 200, request);
}

/**
 * GET|POST /api/external/tokens/get
 *
 * Get a specific token by ID.
 * Requires authentication (owner or admin).
 *
 * GET params: ?tokenId=string
 * POST body:  { tokenId: string }
 */
export async function POST(request: NextRequest) {
  try {
    let body: GetTokenRequest;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }
    if (!body || typeof body.tokenId !== 'string') {
      throw new ExternalApiError(E1001, 'Missing required field: tokenId');
    }

    return await getTokenById(request, body.tokenId);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

export async function GET(request: NextRequest) {
  try {
    const tokenId = request.nextUrl.searchParams.get('tokenId');
    if (!tokenId) {
      throw new ExternalApiError(E1001, 'Missing required field: tokenId');
    }
    return await getTokenById(request, tokenId);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

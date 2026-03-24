import { NextRequest } from 'next/server';
import { getProxy, getUsers, getServers } from '@/lib/proxy-api';
import { getTokenMetadata } from '@/lib/token-metadata';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3003 } from '../../lib/error-codes';
import { getUserPermissions, TokenItem } from '../lib/permissions';

export const dynamic = 'force-dynamic';

interface GetTokenRequest {
  tokenId: string;
}

/**
 * POST /api/external/tokens/get
 *
 * Get a specific token by ID.
 * Requires authentication (owner only).
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: GetTokenRequest;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate required parameters
    const { tokenId } = body;

    if (!tokenId || typeof tokenId !== 'string' || !tokenId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: tokenId');
    }

    // Get proxy info
    const proxy = await getProxy();

    // Get the specific user by userId (tokenId)
    const { users } = await getUsers({ userId: tokenId.trim(), proxyId: proxy.id }, undefined, user.accessToken);

    // Check if user exists
    const targetUser = users.find(u => u.userId === tokenId.trim());
    if (!targetUser) {
      throw new ExternalApiError(E3003, `Token not found: ${tokenId}`);
    }

    // Get servers and permissions for the user
    const { servers } = await getServers({ proxyId: proxy.id }, undefined, user.accessToken);
    const serverNameMap: Record<string, string> = {};
    servers.forEach((server) => {
      serverNameMap[server.serverId] = server.serverName;
    });
    const permissions = await getUserPermissions(targetUser.userId, user.accessToken, serverNameMap);
    const metadata = await getTokenMetadata(proxy.id, targetUser.userId);

    const token: TokenItem = {
      tokenId: targetUser.userId,
      name: targetUser.name,
      role: targetUser.role,
      notes: (targetUser as Record<string, unknown>).notes as string || '',
      lastUsed: 0, // TODO: Implement lastUsed tracking
      createdAt: targetUser.createdAt || 0,
      expiresAt: targetUser.expiresAt || 0,
      rateLimit: targetUser.ratelimit,
      permissions,
      namespace: metadata.namespace,
      tags: metadata.tags,
    };

    return ApiResponse.success(token);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

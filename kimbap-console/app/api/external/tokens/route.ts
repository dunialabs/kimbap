import { NextRequest } from 'next/server';
import { getProxy, getUsers, getServers } from '@/lib/proxy-api';
import { getTokenMetadataMap, normalizeNamespace, normalizeTags } from '@/lib/token-metadata';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../lib/error-codes';
import { getUserPermissions, TokenItem } from './lib/permissions';

export const dynamic = 'force-dynamic';

interface ListTokensResponse {
  tokens: TokenItem[];
}

/**
 * POST /api/external/tokens
 *
 * Get access tokens with optional filtering.
 * Requires authentication.
 *
 * Optional body:
 *   namespace?: string  - filter by exact namespace match
 *   tags?: string[]     - filter by tags (AND match: token must have all specified tags)
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse optional filter parameters
    let filterNamespace: string | undefined;
    let filterTags: string[] | undefined;
    const text = await request.text();
    if (text.trim()) {
      let body: any;
      try {
        body = JSON.parse(text);
      } catch {
        throw new ExternalApiError(E1001, 'Invalid request body');
      }
      if (body && typeof body === 'object') {
        if (body.namespace !== undefined) {
          if (typeof body.namespace !== 'string') {
            throw new ExternalApiError(E1003, 'Invalid field value: namespace must be a string');
          }
          filterNamespace = normalizeNamespace(body.namespace);
        }
        if (body.tags !== undefined) {
          if (!Array.isArray(body.tags)) {
            throw new ExternalApiError(E1003, 'Invalid field value: tags must be an array');
          }
          if (body.tags.some((t: unknown) => typeof t !== 'string')) {
            throw new ExternalApiError(E1003, 'Invalid field value: tags must be an array of strings');
          }
          filterTags = normalizeTags(body.tags);
        }
      }
    }

    // Get proxy info
    const proxy = await getProxy();

    // Get all users for this proxy (using token directly from Authorization header)
    const { users } = await getUsers({ proxyId: proxy.id }, undefined, user.accessToken);
    const { servers } = await getServers({ proxyId: proxy.id }, undefined, user.accessToken);

    // Sort by createdAt ascending
    users.sort((a, b) => (a.createdAt || 0) - (b.createdAt || 0));

    const serverNameMap: Record<string, string> = {};
    servers.forEach((server) => {
      serverNameMap[server.serverId] = server.serverName;
    });

    const tokenIds = users.map((u) => u.userId);
    const metadataMap = await getTokenMetadataMap(proxy.id, tokenIds);

    // Map users to response format
    const tokens: TokenItem[] = await Promise.all(
      users.map(async (u) => {
        const permissions = await getUserPermissions(u.userId, user.accessToken, serverNameMap);
        const metadata = metadataMap.get(u.userId) || { namespace: 'default', tags: [] };

        return {
          tokenId: u.userId,
          name: u.name,
          role: u.role,
          notes: (u as Record<string, unknown>).notes as string || '',
          lastUsed: 0, // TODO: Implement lastUsed tracking
          createdAt: u.createdAt || 0,
          expiresAt: u.expiresAt || 0,
          rateLimit: u.ratelimit,
          permissions,
          namespace: metadata.namespace,
          tags: metadata.tags,
        };
      })
    );

    // Apply filters
    let filteredTokens = tokens;
    if (filterNamespace !== undefined) {
      filteredTokens = filteredTokens.filter(t => t.namespace === filterNamespace);
    }
    if (filterTags && filterTags.length > 0) {
      filteredTokens = filteredTokens.filter(t =>
        filterTags!.every(tag => t.tags.includes(tag))
      );
    }

    const responseData: ListTokensResponse = { tokens: filteredTokens };

    return ApiResponse.success(responseData);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

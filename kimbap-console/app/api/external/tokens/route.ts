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

interface TokenFilters {
  namespace?: string;
  tags?: string[];
}

function parseTokenFiltersFromBody(body: any): TokenFilters {
  let filterNamespace: string | undefined;
  let filterTags: string[] | undefined;

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

  return { namespace: filterNamespace, tags: filterTags };
}

function parseTokenFiltersFromQuery(request: NextRequest): TokenFilters {
  const namespaceRaw = request.nextUrl.searchParams.get('namespace');
  const tagsRaw = request.nextUrl.searchParams.get('tags');
  return {
    namespace: namespaceRaw !== null ? normalizeNamespace(namespaceRaw) : undefined,
    tags: tagsRaw ? normalizeTags(tagsRaw.split(',')) : undefined,
  };
}

async function listTokens(request: NextRequest, filters: TokenFilters) {
  const user = await authenticate(request);

  const proxy = await getProxy();
  const { users } = await getUsers({ proxyId: proxy.id }, undefined, user.accessToken);
  const { servers } = await getServers({ proxyId: proxy.id }, undefined, user.accessToken);

  users.sort((a, b) => (a.createdAt || 0) - (b.createdAt || 0));

  const serverNameMap: Record<string, string> = {};
  servers.forEach((server) => {
    serverNameMap[server.serverId] = server.serverName;
  });

  const tokenIds = users.map((u) => u.userId);
  const metadataMap = await getTokenMetadataMap(proxy.id, tokenIds);

  const tokens: TokenItem[] = await Promise.all(
    users.map(async (u) => {
      const permissions = await getUserPermissions(u.userId, user.accessToken, serverNameMap);
      const metadata = metadataMap.get(u.userId) || { namespace: 'default', tags: [] };

      return {
        tokenId: u.userId,
        name: u.name,
        role: u.role,
        notes: (u as Record<string, unknown>).notes as string || '',
        lastUsed: 0,
        createdAt: u.createdAt || 0,
        expiresAt: u.expiresAt || 0,
        rateLimit: u.ratelimit,
        permissions,
        namespace: metadata.namespace,
        tags: metadata.tags,
      };
    })
  );

  let filteredTokens = tokens;
  if (filters.namespace !== undefined) {
    filteredTokens = filteredTokens.filter(t => t.namespace === filters.namespace);
  }
  if (filters.tags && filters.tags.length > 0) {
    filteredTokens = filteredTokens.filter(t => filters.tags!.every(tag => t.tags.includes(tag)));
  }

  const responseData: ListTokensResponse = { tokens: filteredTokens };
  return ApiResponse.success(responseData);
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
    let body: any = {};
    const text = await request.text();
    if (text.trim()) {
      try {
        body = JSON.parse(text);
      } catch {
        throw new ExternalApiError(E1001, 'Invalid request body');
      }
    }
    return await listTokens(request, parseTokenFiltersFromBody(body));
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

export async function GET(request: NextRequest) {
  try {
    return await listTokens(request, parseTokenFiltersFromQuery(request));
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

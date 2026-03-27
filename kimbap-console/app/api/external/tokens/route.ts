import { NextRequest } from 'next/server';
import { getProxy, getUsers, getServers } from '@/lib/proxy-api';
import { getTokenMetadataMap, normalizeNamespace, normalizeTags } from '@/lib/token-metadata';
import { prisma } from '@/lib/prisma';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';
import { ExternalApiError, E1001, E1003, E5005 } from '../lib/error-codes';
import { getUserPermissions, PermissionsFetchError, TokenItem } from './lib/permissions';

export const dynamic = 'force-dynamic';

interface ListTokensResponse {
  tokens: TokenItem[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
    hasMore: boolean;
  };
}

interface TokenFilters {
  namespace?: string;
  tags?: string[];
  page: number;
  pageSize: number;
}

const DEFAULT_PAGE = 1;
const DEFAULT_PAGE_SIZE = 20;
const MAX_PAGE_SIZE = 100;
const PERMISSIONS_CONCURRENCY = 8;

async function mapWithConcurrency<T, R>(
  items: T[],
  concurrency: number,
  mapper: (item: T, index: number) => Promise<R>
): Promise<R[]> {
  if (items.length === 0) return [];

  const results = new Array<R>(items.length);
  let cursor = 0;

  const workers = Array.from({ length: Math.min(concurrency, items.length) }, async () => {
    while (true) {
      const index = cursor;
      cursor += 1;
      if (index >= items.length) break;
      results[index] = await mapper(items[index], index);
    }
  });

  await Promise.all(workers);
  return results;
}

async function getLastUsedMap(userIds: string[]): Promise<Map<string, number>> {
  if (userIds.length === 0) return new Map();

  const rows = await prisma.log.groupBy({
    by: ['userid'],
    where: { userid: { in: userIds } },
    _max: { addtime: true },
  });

  const map = new Map<string, number>();
  for (const row of rows) {
    if (row._max.addtime !== null) {
      map.set(row.userid, Number(row._max.addtime));
    }
  }
  return map;
}

function parsePositiveInt(value: unknown, field: string): number | undefined {
  if (value === undefined || value === null || value === '') {
    return undefined;
  }
  if (typeof value === 'number') {
    if (Number.isInteger(value) && value > 0) {
      return value;
    }
    throw new ExternalApiError(E1003, `Invalid field value: ${field} must be a positive integer`);
  }
  if (typeof value === 'string') {
    const n = Number(value.trim());
    if (Number.isInteger(n) && n > 0) {
      return n;
    }
    throw new ExternalApiError(E1003, `Invalid field value: ${field} must be a positive integer`);
  }
  throw new ExternalApiError(E1003, `Invalid field value: ${field} must be a positive integer`);
}

function resolvePagination(pageRaw: unknown, pageSizeRaw: unknown): { page: number; pageSize: number } {
  const page = parsePositiveInt(pageRaw, 'page') ?? DEFAULT_PAGE;
  const pageSize = parsePositiveInt(pageSizeRaw, 'pageSize') ?? DEFAULT_PAGE_SIZE;

  if (pageSize > MAX_PAGE_SIZE) {
    throw new ExternalApiError(E1003, `Invalid field value: pageSize must be <= ${MAX_PAGE_SIZE}`);
  }

  return { page, pageSize };
}

function parseTokenFiltersFromBody(body: Record<string, unknown>): TokenFilters {
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

  const safeBody = (body && typeof body === 'object') ? body : {};
  const { page, pageSize } = resolvePagination(safeBody.page, safeBody.pageSize);

  return { namespace: filterNamespace, tags: filterTags, page, pageSize };
}

function parseTokenFiltersFromQuery(request: NextRequest): TokenFilters {
  const namespaceRaw = request.nextUrl.searchParams.get('namespace');
  const tagsRaw = request.nextUrl.searchParams.get('tags');
  const { page, pageSize } = resolvePagination(
    request.nextUrl.searchParams.get('page'),
    request.nextUrl.searchParams.get('pageSize')
  );
  return {
    namespace: namespaceRaw !== null ? normalizeNamespace(namespaceRaw) : undefined,
    tags: tagsRaw ? normalizeTags(tagsRaw.split(',')) : undefined,
    page,
    pageSize,
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

  const hasMetadataFilters = filters.namespace !== undefined || (filters.tags && filters.tags.length > 0);
  const metadataMap = hasMetadataFilters
    ? await getTokenMetadataMap(users.map((u) => u.userId))
    : new Map<string, { namespace: string; tags: string[] }>();

  const matchedUsers = users.filter((u) => {
    if (!hasMetadataFilters) {
      return true;
    }
    const metadata = metadataMap.get(u.userId) || { namespace: 'default', tags: [] };
    if (filters.namespace !== undefined && metadata.namespace !== filters.namespace) {
      return false;
    }
    if (filters.tags && filters.tags.length > 0) {
      return filters.tags.every((tag) => metadata.tags.includes(tag));
    }
    return true;
  });

  const total = matchedUsers.length;
  const totalPages = Math.max(1, Math.ceil(total / filters.pageSize));
  const start = (filters.page - 1) * filters.pageSize;
  const pagedUsers = matchedUsers.slice(start, start + filters.pageSize);

  const pageMetadataMap = hasMetadataFilters
    ? metadataMap
    : await getTokenMetadataMap(pagedUsers.map((u) => u.userId));
  const lastUsedMap = await getLastUsedMap(pagedUsers.map((u) => u.userId));

  let tokens: TokenItem[];
  try {
    tokens = await mapWithConcurrency(pagedUsers, PERMISSIONS_CONCURRENCY, async (u) => {
      const permissions = await getUserPermissions(u.userId, user.accessToken, serverNameMap);
      const metadata = pageMetadataMap.get(u.userId) || { namespace: 'default', tags: [] };

      return {
        tokenId: u.userId,
        name: u.name,
        role: u.role,
        notes: (u as Record<string, unknown>).notes as string || '',
        lastUsed: lastUsedMap.get(u.userId) || 0,
        createdAt: u.createdAt || 0,
        expiresAt: u.expiresAt || 0,
        rateLimit: u.ratelimit,
        permissions,
        namespace: metadata.namespace,
        tags: metadata.tags,
      };
    });
  } catch (error) {
    if (error instanceof PermissionsFetchError) {
      throw new ExternalApiError(E5005, 'Permission service unavailable');
    }
    throw error;
  }

  const responseData: ListTokensResponse = {
    tokens,
    pagination: {
      page: filters.page,
      pageSize: filters.pageSize,
      total,
      totalPages,
      hasMore: filters.page < totalPages,
    },
  };
  return ApiResponse.success(responseData, 200, request);
}

/**
 * GET|POST /api/external/tokens
 *
 * List access tokens with optional filtering.
 * Requires authentication (owner or admin).
 *
 * GET params: ?namespace=string&tags=a,b (comma-separated)
 * POST body:  { namespace?: string, tags?: string[] }
 */
export async function POST(request: NextRequest) {
  try {
    let body: Record<string, unknown> = {};
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
    return ApiResponse.handleError(error, request);
  }
}

export async function GET(request: NextRequest) {
  try {
    return await listTokens(request, parseTokenFiltersFromQuery(request));
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

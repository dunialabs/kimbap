import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../lib/error-codes';
import { throwCoreAdminError } from '../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface ListApprovalsInput {
  userId?: string;
  serverId?: string;
  toolName?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}

function isPositiveInteger(n: unknown): n is number {
  return typeof n === 'number' && Number.isFinite(n) && Number.isInteger(n) && n > 0;
}

function normalizeListApprovalsInput(body: ListApprovalsInput): ListApprovalsInput {
  if (body.userId !== undefined && typeof body.userId !== 'string') {
    throw new ExternalApiError(E1003, 'Invalid field value: userId must be a string');
  }
  if (body.serverId !== undefined && typeof body.serverId !== 'string') {
    throw new ExternalApiError(E1003, 'Invalid field value: serverId must be a string');
  }
  if (body.toolName !== undefined && typeof body.toolName !== 'string') {
    throw new ExternalApiError(E1003, 'Invalid field value: toolName must be a string');
  }
  if (body.status !== undefined && typeof body.status !== 'string') {
    throw new ExternalApiError(E1003, 'Invalid field value: status must be a string');
  }
  if (body.page !== undefined && !isPositiveInteger(body.page)) {
    throw new ExternalApiError(E1003, 'Invalid field value: page must be a positive integer');
  }
  if (body.pageSize !== undefined && !isPositiveInteger(body.pageSize)) {
    throw new ExternalApiError(E1003, 'Invalid field value: pageSize must be a positive integer');
  }
  return body;
}

function parsePositiveIntParam(raw: string | null, name: string): number | undefined {
  if (raw === null || raw.trim() === '') return undefined;
  const n = Number(raw.trim());
  if (!Number.isFinite(n) || !Number.isInteger(n) || n < 1) {
    throw new ExternalApiError(E1003, `Invalid field value: ${name} must be a positive integer`);
  }
  return n;
}

function parseListApprovalsQuery(request: NextRequest): ListApprovalsInput {
  return {
    userId: request.nextUrl.searchParams.get('userId') ?? undefined,
    serverId: request.nextUrl.searchParams.get('serverId') ?? undefined,
    toolName: request.nextUrl.searchParams.get('toolName') ?? undefined,
    status: request.nextUrl.searchParams.get('status') ?? undefined,
    page: parsePositiveIntParam(request.nextUrl.searchParams.get('page'), 'page'),
    pageSize: parsePositiveIntParam(request.nextUrl.searchParams.get('pageSize'), 'pageSize'),
  };
}

async function listApprovals(request: NextRequest, input: ListApprovalsInput) {
  const user = await authenticate(request);

  const response = await makeProxyRequestWithUserId<{
    page: number;
    pageSize: number;
    hasMore: boolean;
    requests: any[];
  }>(
    AdminActionType.LIST_APPROVAL_REQUESTS,
    {
      userId: input.userId,
      serverId: input.serverId,
      toolName: input.toolName,
      status: input.status,
      page: input.page,
      pageSize: input.pageSize,
    },
    user.userid,
    user.accessToken
  );

  if (!response.success) {
    throwCoreAdminError(response.error?.message || 'Failed to list approval requests', undefined, response.error?.code);
  }

  return ApiResponse.success({
    page: response.data?.page || 1,
    pageSize: response.data?.pageSize || 20,
    hasMore: response.data?.hasMore || false,
    requests: response.data?.requests || [],
  }, 200, request);
}

export async function POST(request: NextRequest) {
  try {
    let body: ListApprovalsInput = {};
    const text = await request.text();
    if (text.trim()) {
      try {
        const parsed = JSON.parse(text);
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
          throw new ExternalApiError(E1001, 'Invalid request body');
        }
        body = parsed;
      } catch (error) {
        if (error instanceof ExternalApiError) throw error;
        throw new ExternalApiError(E1001, 'Invalid request body');
      }
    }
    return await listApprovals(request, normalizeListApprovalsInput(body));
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

export async function GET(request: NextRequest) {
  try {
    return await listApprovals(request, normalizeListApprovalsInput(parseListApprovalsQuery(request)));
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

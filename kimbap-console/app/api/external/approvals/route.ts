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

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: ListApprovalsInput = {};
    const text = await request.text();
    if (text.trim()) {
      try {
        body = JSON.parse(text);
      } catch {
        throw new ExternalApiError(E1001, 'Invalid request body');
      }
    }

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
    if (body.page !== undefined && typeof body.page !== 'number') {
      throw new ExternalApiError(E1003, 'Invalid field value: page must be a number');
    }
    if (body.pageSize !== undefined && typeof body.pageSize !== 'number') {
      throw new ExternalApiError(E1003, 'Invalid field value: pageSize must be a number');
    }

    const response = await makeProxyRequestWithUserId<{
      page: number;
      pageSize: number;
      hasMore: boolean;
      requests: any[];
    }>(
      AdminActionType.LIST_APPROVAL_REQUESTS,
      {
        userId: body.userId,
        serverId: body.serverId,
        toolName: body.toolName,
        status: body.status,
        page: body.page,
        pageSize: body.pageSize,
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
    });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

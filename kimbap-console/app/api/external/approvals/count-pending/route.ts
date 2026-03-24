import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../../lib/error-codes';
import { throwCoreAdminError } from '../../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface CountPendingInput {
  userId?: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: CountPendingInput = {};
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

    const targetUserId = body.userId && body.userId.trim() ? body.userId.trim() : undefined;

    const response = await makeProxyRequestWithUserId<{ count: number }>(
      AdminActionType.COUNT_PENDING_APPROVALS,
      { userId: targetUserId },
      user.userid,
      user.accessToken
    );

    if (!response.success) {
      throwCoreAdminError(response.error?.message || 'Failed to count pending approvals', undefined, response.error?.code);
    }

    return ApiResponse.success({ count: response.data?.count || 0 });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

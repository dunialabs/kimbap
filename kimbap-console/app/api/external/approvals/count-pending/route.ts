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

function normalizeCountPendingInput(body: CountPendingInput): CountPendingInput {
  if (body.userId !== undefined && typeof body.userId !== 'string') {
    throw new ExternalApiError(E1003, 'Invalid field value: userId must be a string');
  }
  return {
    userId: body.userId && body.userId.trim() ? body.userId.trim() : undefined,
  };
}

async function countPending(request: NextRequest, input: CountPendingInput) {
  const user = await authenticate(request);

  const response = await makeProxyRequestWithUserId<{ count: number }>(
    AdminActionType.COUNT_PENDING_APPROVALS,
    { userId: input.userId },
    user.userid,
    user.accessToken
  );

  if (!response.success) {
    throwCoreAdminError(response.error?.message || 'Failed to count pending approvals', undefined, response.error?.code);
  }

  return ApiResponse.success({ count: response.data?.count || 0 });
}

export async function POST(request: NextRequest) {
  try {
    let body: CountPendingInput = {};
    const text = await request.text();
    if (text.trim()) {
      try {
        body = JSON.parse(text);
      } catch {
        throw new ExternalApiError(E1001, 'Invalid request body');
      }
    }
    return await countPending(request, normalizeCountPendingInput(body));
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

export async function GET(request: NextRequest) {
  try {
    const userId = request.nextUrl.searchParams.get('userId') ?? undefined;
    return await countPending(request, normalizeCountPendingInput({ userId }));
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3002 } from '../../lib/error-codes';
import { throwCoreAdminError } from '../../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface GetApprovalInput {
  id: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: GetApprovalInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    if (!body.id || typeof body.id !== 'string' || !body.id.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: id');
    }

    const response = await makeProxyRequestWithUserId<any>(
      AdminActionType.GET_APPROVAL_REQUEST,
      { id: body.id.trim() },
      user.userid,
      user.accessToken
    );

    if (!response.success) {
      throwCoreAdminError(response.error?.message || 'Failed to get approval request', E3002, response.error?.code);
    }

    return ApiResponse.success(response.data || null);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

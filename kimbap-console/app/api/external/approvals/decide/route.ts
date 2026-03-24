import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../../lib/error-codes';
import { throwCoreAdminError } from '../../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface DecideApprovalInput {
  id: string;
  decision: 'APPROVED' | 'REJECTED';
  reason?: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: DecideApprovalInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    if (!body.id || typeof body.id !== 'string' || !body.id.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: id');
    }

    if (body.decision !== 'APPROVED' && body.decision !== 'REJECTED') {
      throw new ExternalApiError(E1003, 'Invalid field value: decision must be APPROVED or REJECTED');
    }

    if (body.reason !== undefined && typeof body.reason !== 'string') {
      throw new ExternalApiError(E1003, 'Invalid field value: reason must be a string');
    }

    const response = await makeProxyRequestWithUserId<any>(
      AdminActionType.DECIDE_APPROVAL_REQUEST,
      {
        id: body.id.trim(),
        decision: body.decision,
        reason: body.reason,
      },
      user.userid,
      user.accessToken
    );

    if (!response.success) {
      throwCoreAdminError(response.error?.message || 'Failed to decide approval request', undefined, response.error?.code);
    }

    return ApiResponse.success(response.data || null);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

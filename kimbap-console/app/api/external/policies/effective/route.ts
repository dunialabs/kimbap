import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../../lib/error-codes';
import { throwCoreAdminError } from '../../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface EffectivePolicyInput {
  serverId?: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: EffectivePolicyInput = {};
    const text = await request.text();
    if (text.trim()) {
      try {
        body = JSON.parse(text);
      } catch {
        throw new ExternalApiError(E1001, 'Invalid request body');
      }
    }

    if (body.serverId !== undefined && typeof body.serverId !== 'string') {
      throw new ExternalApiError(E1003, 'Invalid field value: serverId must be a string');
    }

    const response = await makeProxyRequestWithUserId<{ policySets: any[] }>(
      AdminActionType.GET_EFFECTIVE_POLICY,
      { serverId: body.serverId },
      user.userid,
      user.accessToken
    );

    if (!response.success) {
      throwCoreAdminError(response.error?.message || 'Failed to get effective policy', undefined, response.error?.code);
    }

    return ApiResponse.success({ policySets: response.data?.policySets || [] });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

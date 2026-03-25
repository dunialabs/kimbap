import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001 } from '../../lib/error-codes';
import { throwCoreAdminError } from '../../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface UpdatePolicyInput {
  id: string;
  dsl?: {
    schemaVersion?: 1;
    rules: any[];
  };
  status?: 'active' | 'archived';
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: UpdatePolicyInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    if (!body || typeof body !== 'object' || Array.isArray(body)) {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    if (!body.id || typeof body.id !== 'string' || !body.id.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: id');
    }

    if (body.dsl === undefined && body.status === undefined) {
      throw new ExternalApiError(E1001, 'Missing required field: dsl or status');
    }

    const response = await makeProxyRequestWithUserId<any>(
      AdminActionType.UPDATE_TOOL_POLICY,
      {
        id: body.id.trim(),
        dsl: body.dsl,
        status: body.status,
      },
      user.userid,
      user.accessToken
    );

    if (!response.success) {
      throwCoreAdminError(response.error?.message || 'Failed to update policy', undefined, response.error?.code);
    }

    return ApiResponse.success(response.data || null, 200, request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

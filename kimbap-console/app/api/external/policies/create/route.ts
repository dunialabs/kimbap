import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../../lib/error-codes';
import { throwCoreAdminError } from '../../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface CreatePolicyInput {
  serverId?: string;
  dsl: {
    schemaVersion: 1;
    rules: any[];
  };
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: CreatePolicyInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    if (!body || typeof body !== 'object' || Array.isArray(body)) {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    if (!body.dsl || typeof body.dsl !== 'object') {
      throw new ExternalApiError(E1001, 'Missing required field: dsl');
    }

    if (body.serverId !== undefined && typeof body.serverId !== 'string') {
      throw new ExternalApiError(E1003, 'Invalid field value: serverId must be a string');
    }

    const response = await makeProxyRequestWithUserId<any>(
      AdminActionType.CREATE_TOOL_POLICY,
      {
        serverId: body.serverId,
        dsl: body.dsl,
      },
      user.userid,
      user.accessToken
    );

    if (!response.success) {
      throwCoreAdminError(response.error?.message || 'Failed to create policy', undefined, response.error?.code);
    }

    return ApiResponse.success(response.data || null, 201, request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

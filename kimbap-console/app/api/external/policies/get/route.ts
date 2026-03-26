import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3002 } from '../../lib/error-codes';
import { throwCoreAdminError } from '../../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface GetPolicyInput {
  id: string;
}

async function getPolicyById(request: NextRequest, idRaw: string) {
  const user = await authenticate(request);
  const id = idRaw.trim();
  if (!id) {
    throw new ExternalApiError(E1001, 'Missing required field: id');
  }

  const response = await makeProxyRequestWithUserId<{ policySets: any[] }>(
    AdminActionType.GET_TOOL_POLICY,
    { id },
    user.userid,
    user.accessToken
  );

  if (!response.success) {
    throwCoreAdminError(response.error?.message || 'Failed to get policy', E3002, response.error?.code);
  }

  const policySets = response.data?.policySets || [];
  const target = policySets[0];
  if (!target) {
    throw new ExternalApiError(E3002, `Policy not found: ${id}`);
  }

  return ApiResponse.success(target, 200, request);
}

export async function POST(request: NextRequest) {
  try {
    let body: GetPolicyInput;
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
    return await getPolicyById(request, body.id);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

export async function GET(request: NextRequest) {
  try {
    const id = request.nextUrl.searchParams.get('id');
    if (!id) {
      throw new ExternalApiError(E1001, 'Missing required field: id');
    }
    return await getPolicyById(request, id);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

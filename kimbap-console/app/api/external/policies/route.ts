import { NextRequest } from 'next/server';
import { makeProxyRequestWithUserId, AdminActionType } from '@/lib/proxy-api';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../lib/error-codes';
import { throwCoreAdminError } from '../lib/core-admin-error';

export const dynamic = 'force-dynamic';

interface ListPoliciesInput {
  serverId?: string;
}

function normalizeListPoliciesInput(body: ListPoliciesInput): ListPoliciesInput {
  if (body.serverId !== undefined && typeof body.serverId !== 'string') {
    throw new ExternalApiError(E1003, 'Invalid field value: serverId must be a string');
  }
  return body;
}

async function listPolicies(request: NextRequest, input: ListPoliciesInput) {
  const user = await authenticate(request);
  const response = await makeProxyRequestWithUserId<{ policySets: any[] }>(
    AdminActionType.GET_TOOL_POLICY,
    { serverId: input.serverId },
    user.userid,
    user.accessToken
  );

  if (!response.success) {
    throwCoreAdminError(response.error?.message || 'Failed to list policies', undefined, response.error?.code);
  }

  return ApiResponse.success({ policySets: response.data?.policySets || [] });
}

export async function POST(request: NextRequest) {
  try {
    let body: ListPoliciesInput = {};
    const text = await request.text();
    if (text.trim()) {
      try {
        body = JSON.parse(text);
      } catch {
        throw new ExternalApiError(E1001, 'Invalid request body');
      }
    }
    return await listPolicies(request, normalizeListPoliciesInput(body));
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

export async function GET(request: NextRequest) {
  try {
    const serverId = request.nextUrl.searchParams.get('serverId') ?? undefined;
    return await listPolicies(request, normalizeListPoliciesInput({ serverId }));
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

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

function normalizeEffectivePolicyInput(body: EffectivePolicyInput): EffectivePolicyInput {
  if (body.serverId !== undefined && typeof body.serverId !== 'string') {
    throw new ExternalApiError(E1003, 'Invalid field value: serverId must be a string');
  }
  return body;
}

async function getEffectivePolicy(request: NextRequest, input: EffectivePolicyInput) {
  const user = await authenticate(request);
  const response = await makeProxyRequestWithUserId<{ policySets: any[] }>(
    AdminActionType.GET_EFFECTIVE_POLICY,
    { serverId: input.serverId },
    user.userid,
    user.accessToken
  );

  if (!response.success) {
    throwCoreAdminError(response.error?.message || 'Failed to get effective policy', undefined, response.error?.code);
  }

  return ApiResponse.success({ policySets: response.data?.policySets || [] }, 200, request);
}

export async function POST(request: NextRequest) {
  try {
    let body: EffectivePolicyInput = {};
    const text = await request.text();
    if (text.trim()) {
      try {
        const parsed = JSON.parse(text);
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
          throw new ExternalApiError(E1001, 'Invalid request body');
        }
        body = parsed;
      } catch (error) {
        if (error instanceof ExternalApiError) throw error;
        throw new ExternalApiError(E1001, 'Invalid request body');
      }
    }
    return await getEffectivePolicy(request, normalizeEffectivePolicyInput(body));
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

export async function GET(request: NextRequest) {
  try {
    const serverId = request.nextUrl.searchParams.get('serverId') ?? undefined;
    return await getEffectivePolicy(request, normalizeEffectivePolicyInput({ serverId }));
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

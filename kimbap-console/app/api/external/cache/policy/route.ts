import { NextRequest } from 'next/server';
import { getCachePolicy } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface GetPolicyInput {
  serverId: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: GetPolicyInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: serverId');
    }

    if (!body?.serverId || typeof body.serverId !== 'string' || !body.serverId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: serverId');
    }

    const policy = await getCachePolicy(body.serverId, undefined, user.accessToken);
    return ApiResponse.success(policy);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

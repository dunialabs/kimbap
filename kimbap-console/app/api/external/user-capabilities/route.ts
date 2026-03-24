import { NextRequest } from 'next/server';
import { getUserAvailableServersCapabilities } from '@/lib/proxy-api';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';
import { ExternalApiError, E1001 } from '../lib/error-codes';

export const dynamic = 'force-dynamic';

interface GetCapabilitiesInput {
  userId: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const ownerToken = user.accessToken;

    let body: GetCapabilitiesInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: userId');
    }

    if (!body?.userId || typeof body.userId !== 'string' || !body.userId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: userId');
    }

    const capabilities = await getUserAvailableServersCapabilities(body.userId, undefined, ownerToken);

    return ApiResponse.success({ capabilities });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

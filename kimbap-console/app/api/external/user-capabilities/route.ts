import { NextRequest } from 'next/server';
import { getUserAvailableServersCapabilities } from '@/lib/proxy-api';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';
import { ExternalApiError, E1001 } from '../lib/error-codes';

export const dynamic = 'force-dynamic';

interface GetCapabilitiesInput {
  userId: string;
}

async function getCapabilities(request: NextRequest, input: GetCapabilitiesInput) {
  const user = await authenticate(request);
  const ownerToken = user.accessToken;

  if (!input?.userId || typeof input.userId !== 'string' || !input.userId.trim()) {
    throw new ExternalApiError(E1001, 'Missing required field: userId');
  }

  const capabilities = await getUserAvailableServersCapabilities(input.userId, undefined, ownerToken);

  return ApiResponse.success({ capabilities });
}

export async function POST(request: NextRequest) {
  try {
    let body: GetCapabilitiesInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: userId');
    }

    return await getCapabilities(request, body);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

export async function GET(request: NextRequest) {
  try {
    const userId = request.nextUrl.searchParams.get('userId');
    if (!userId) {
      throw new ExternalApiError(E1001, 'Missing required field: userId');
    }
    return await getCapabilities(request, { userId });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

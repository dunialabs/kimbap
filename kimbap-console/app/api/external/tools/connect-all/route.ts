import { NextRequest } from 'next/server';
import { connectAllServers } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';

export const dynamic = 'force-dynamic';

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const ownerToken = user.accessToken;

    const result = await connectAllServers(undefined, ownerToken);

    return ApiResponse.success({
      successServers: result.successServers,
      failedServers: result.failedServers,
    });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

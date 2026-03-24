import { NextRequest } from 'next/server';
import { deleteIpWhitelist } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface DeleteIpWhitelistInput {
  ips: string[];
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: DeleteIpWhitelistInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: ips');
    }

    if (!body.ips) {
      throw new ExternalApiError(E1001, 'Missing required field: ips');
    }

    if (!Array.isArray(body.ips) || body.ips.length === 0 || body.ips.some((ip) => typeof ip !== 'string')) {
      throw new ExternalApiError(E1003, 'Invalid field value: ips');
    }

    const result = await deleteIpWhitelist(body.ips, user.userid);
    return ApiResponse.success(result);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

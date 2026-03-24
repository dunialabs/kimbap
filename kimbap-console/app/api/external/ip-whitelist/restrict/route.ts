import { NextRequest } from 'next/server';
import { specialIpWhitelistOperation } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';

export const dynamic = 'force-dynamic';

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    await specialIpWhitelistOperation('deny-all', user.userid);
    return ApiResponse.success({ success: true });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

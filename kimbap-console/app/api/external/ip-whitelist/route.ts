import { NextRequest } from 'next/server';
import { getIpWhitelist } from '@/lib/proxy-api';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';

export const dynamic = 'force-dynamic';

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const result = await getIpWhitelist(user.userid);
    const allowAll = result.whitelist.includes('0.0.0.0/0');
    const ips = result.whitelist.filter((ip: string) => ip !== '0.0.0.0/0');
    return ApiResponse.success({ allowAll, ips });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

import { NextRequest } from 'next/server';
import { prisma } from '@/lib/prisma';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E4014 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

export async function POST(request: NextRequest) {
  try {
    await authenticate(request);

    const config = await prisma.config.findFirst();

    if (!config || !config.kimbap_core_host) {
      throw new ExternalApiError(E4014, 'KIMBAP Core not configured');
    }

    return ApiResponse.success({
      host: config.kimbap_core_host,
      port: config.kimbap_core_prot || null,
      connected: true,
    });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

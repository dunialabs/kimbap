import { NextRequest } from 'next/server';
import { prisma } from '@/lib/prisma';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E4014 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

async function getKimbapCoreConnection(request: NextRequest) {
  await authenticate(request);

  const config = await prisma.config.findFirst();

  if (!config || !config.kimbap_core_host) {
    throw new ExternalApiError(E4014, 'Kimbap Core not configured');
  }

  const currentPort = Reflect.get(config, 'kimbap_core_port');
  const legacyPort = Reflect.get(config, 'kimbap_core_prot');
  const port =
    typeof currentPort === 'number'
      ? currentPort
      : typeof legacyPort === 'number'
        ? legacyPort
        : null;

  return ApiResponse.success({
    host: config.kimbap_core_host,
    port,
    connected: true,
  });
}

export async function POST(request: NextRequest) {
  try {
    return await getKimbapCoreConnection(request);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

export async function GET(request: NextRequest) {
  try {
    return await getKimbapCoreConnection(request);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

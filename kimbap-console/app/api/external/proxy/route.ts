import { getProxy } from '@/lib/proxy-api';
import { ApiResponse } from '../lib/response';
import { ExternalApiError, E3001 } from '../lib/error-codes';

export const dynamic = 'force-dynamic';

interface ProxyResponse {
  proxyId: number;
  proxyKey: string;
  proxyName: string;
  createdAt: number;
  fingerprint: string;
}

async function getProxyInfo() {
  const proxy = await getProxy();

  if (!proxy) {
    throw new ExternalApiError(E3001, 'Proxy not found');
  }

  const responseData: ProxyResponse = {
    proxyId: proxy.id,
    proxyKey: proxy.proxyKey || '',
    proxyName: proxy.name,
    createdAt: proxy.addtime,
    fingerprint: '',
  };

  return ApiResponse.success(responseData);
}

/**
 * GET|POST /api/external/proxy
 *
 * Get current proxy server information.
 * No authentication required.
 */
export async function POST() {
  try {
    return await getProxyInfo();
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

export async function GET() {
  try {
    return await getProxyInfo();
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

import { type NextRequest } from 'next/server';
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

async function getProxyInfo(request: NextRequest) {
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

  return ApiResponse.success(responseData, 200, request);
}

export async function POST(request: NextRequest) {
  try {
    return await getProxyInfo(request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

export async function GET(request: NextRequest) {
  try {
    return await getProxyInfo(request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

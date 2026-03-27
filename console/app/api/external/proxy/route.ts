import { type NextRequest } from 'next/server';
import { getProxy } from '@/lib/proxy-api';
import { ApiResponse } from '../lib/response';

import { authenticate } from '../lib/auth';

export const dynamic = 'force-dynamic';

interface ProxyResponse {
  proxyId: number;
  proxyName: string;
  createdAt: number;
  fingerprint: string;
}

async function getProxyInfo(request: NextRequest) {
  await authenticate(request);
  const proxy = await getProxy();

  const responseData: ProxyResponse = {
    proxyId: proxy.id,
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

import { NextRequest } from 'next/server';
import { getProxy, updateProxy } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface UpdateProxyRequest {
  proxyId: number;
  proxyName: string;
}

interface UpdateProxyResponse {
  proxyId: number;
  proxyName: string;
}

/**
 * POST /api/external/proxy/update
 *
 * Update proxy configuration.
 * Requires authentication.
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: UpdateProxyRequest;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: proxyId');
    }

    if (!body || typeof body !== 'object' || Array.isArray(body)) {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate required parameters
    const { proxyId, proxyName } = body;

    if (proxyId === undefined || proxyId === null) {
      throw new ExternalApiError(E1001, 'Missing required field: proxyId');
    }

    if (!proxyName || typeof proxyName !== 'string' || !proxyName.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: proxyName');
    }

    // Get current proxy
    const proxy = await getProxy();

    // Check if proxyId matches
    if (proxy.id !== proxyId) {
      throw new ExternalApiError(E1003, 'Invalid field value: proxyId');
    }

    // Update proxy name (using token directly from Authorization header)
    await updateProxy(proxy.id, { name: proxyName.trim() }, undefined, user.accessToken);

    const responseData: UpdateProxyResponse = {
      proxyId: proxy.id,
      proxyName: proxyName.trim(),
    };

    return ApiResponse.success(responseData, 200, request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

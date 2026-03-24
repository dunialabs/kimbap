import { getProxy } from '@/lib/proxy-api';
import { LicenseService } from '@/license-system';
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

/**
 * POST /api/external/proxy
 *
 * Get current proxy server information.
 * No authentication required.
 */
export async function POST() {
  try {
    // Get the proxy record (this implicitly validates KIMBAP Core is reachable)
    const proxy = await getProxy();

    if (!proxy) {
      throw new ExternalApiError(E3001, 'Proxy not found');
    }

    // Get hardware fingerprint
    const licenseService = LicenseService.getInstance();
    const fingerprint = licenseService.getHardwareFingerprint();

    const responseData: ProxyResponse = {
      proxyId: proxy.id,
      proxyKey: proxy.proxyKey || '',
      proxyName: proxy.name,
      createdAt: proxy.addtime,
      fingerprint: fingerprint,
    };

    return ApiResponse.success(responseData);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

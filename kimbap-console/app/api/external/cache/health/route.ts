import { NextRequest } from 'next/server';
import { getCacheHealth } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';

export const dynamic = 'force-dynamic';

/**
 * POST /api/external/cache/health
 *
 * Get result cache health status, backend info, and metrics.
 * Requires authentication (owner only).
 */
export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const health = await getCacheHealth(undefined, user.accessToken);

    return ApiResponse.success({
      enabled: health.enabled,
      backend: health.health?.backend ?? 'unknown',
      status: health.health?.ok ? 'healthy' : 'unhealthy',
      details: health.health?.details,
      metrics: health.metrics,
    });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

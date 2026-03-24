import { NextRequest } from 'next/server';
import { KimbapCloudApiService } from '@/lib/KimbapCloudApiService';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';

export const dynamic = 'force-dynamic';

/**
 * POST /api/external/templates
 *
 * Get all available tool templates.
 * Requires authentication (owner only).
 * Reference: protocol-10004
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    await authenticate(request);

    // Fetch tool templates
    const kimbapCloudApi = new KimbapCloudApiService();
    const templates = await kimbapCloudApi.fetchToolTemplatesWithFallback();

    // Map to external API format (simplified)
    const templateList = templates.map((t) => ({
      toolTmplId: t.toolTmplId,
      name: t.name,
      description: t.description,
      mcpJsonConf: t.mcpJsonConf,
    }));

    return ApiResponse.success({ templates: templateList });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

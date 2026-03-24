import { NextRequest } from 'next/server';
import { getServers, stopMCPServer, deleteServer } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3002 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface DeleteToolInput {
  toolId: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const ownerToken = user.accessToken;

    let body: DeleteToolInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: toolId');
    }

    if (!body?.toolId || typeof body.toolId !== 'string' || !body.toolId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: toolId');
    }

    const { toolId } = body;
    const { servers } = await getServers({ serverId: toolId }, undefined, ownerToken);
    const server = servers.length > 0 ? servers[0] : null;
    if (!server) {
      throw new ExternalApiError(E3002, `Tool not found: ${toolId}`);
    }

    try {
      await stopMCPServer(toolId, undefined, ownerToken);
    } catch (stopError) {
      console.warn(`[External API] Failed to stop server before delete: ${toolId}`, stopError);
    }

    await deleteServer(toolId, undefined, ownerToken);
    return ApiResponse.success({ message: 'Tool deleted successfully' });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

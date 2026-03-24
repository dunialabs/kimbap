import { NextRequest } from 'next/server';
import { getServers, getServersStatus, getServersCapabilities, ServerStatus } from '@/lib/proxy-api';
import { KimbapCloudApiService } from '@/lib/KimbapCloudApiService';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3002 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface GetToolInput {
  toolId: string;
}

interface ToolFunction {
  funcName: string;
  enabled: boolean;
  dangerLevel: number;
  description?: string;
}

interface ToolResource {
  uri: string;
  enabled: boolean;
}

/**
 * POST /api/external/tools/get
 *
 * Get a specific tool by ID with its capabilities.
 * Requires authentication (owner only).
 * Reference: protocol-10006 + protocol-10010
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: GetToolInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate toolId
    const { toolId } = body;
    if (!toolId || typeof toolId !== 'string' || !toolId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: toolId');
    }

    const ownerToken = user.accessToken;

    // Fetch server by ID
    const { servers } = await getServers({ serverId: toolId }, undefined, ownerToken);
    const server = servers.length > 0 ? servers[0] : null;

    if (!server) {
      throw new ExternalApiError(E3002, `Tool not found: ${toolId}`);
    }

    // Fetch tool templates for matching
    const kimbapCloudApi = new KimbapCloudApiService();
    const templates = await kimbapCloudApi.fetchToolTemplatesWithFallback();
    const matchedTemplate = server.toolTmplId
      ? templates.find((t) => t.toolTmplId === server.toolTmplId)
      : null;

    // Fetch server status
    let runningState = ServerStatus.Offline;
    try {
      const statusMap = await getServersStatus();
      runningState = statusMap[server.serverId] ?? ServerStatus.Offline;
    } catch (error) {
      console.error('Failed to fetch server status:', error);
    }

    // Fetch capabilities (functions and resources)
    let functions: ToolFunction[] = [];
    let resources: ToolResource[] = [];

    try {
      const capabilities = await getServersCapabilities(toolId, undefined, ownerToken);

      if (capabilities.tools) {
        functions = Object.entries(capabilities.tools).map(
          ([funcName, config]: [string, any]) => ({
            funcName,
            enabled: config.enabled || false,
            dangerLevel: config.dangerLevel !== undefined ? config.dangerLevel : 0,
            description: config.description || undefined,
          })
        );
      }

      if (capabilities.resources) {
        resources = Object.entries(capabilities.resources).map(
          ([uri, config]: [string, any]) => ({
            uri,
            enabled: config.enabled || false,
          })
        );
      }
    } catch (error) {
      console.error('Failed to fetch capabilities:', error);
      // Continue with empty arrays
    }

    const tool = {
      toolId: server.serverId,
      toolTmplId: matchedTemplate?.toolTmplId || '',
      name: server.serverName,
      description: matchedTemplate?.description || '',
      enabled: server.enabled,
      runningState,
      functions,
      resources,
      lazyStartEnabled: server.lazyStartEnabled,
      publicAccess: server.publicAccess,
      allowUserInput: Boolean(server.allowUserInput),
      createdAt: server.createdAt || 0,
    };

    return ApiResponse.success(tool);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

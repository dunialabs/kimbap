import { NextRequest } from 'next/server';
import { getServers, getServersStatus, ServerStatus } from '@/lib/proxy-api';
import { KimbapCloudApiService } from '@/lib/KimbapCloudApiService';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';

export const dynamic = 'force-dynamic';

interface ListToolsInput {
  enabled?: boolean;
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
 * POST /api/external/tools
 *
 * List all configured tools/servers.
 * Requires authentication (owner only).
 * Reference: protocol-10006 handleType=1/2
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: ListToolsInput = {};
    try {
      body = await request.json();
    } catch {
      // Empty body is acceptable
    }

    const ownerToken = user.accessToken;

    // Build filter
    const filters: { enabled?: boolean } = {};
    if (body.enabled !== undefined) {
      filters.enabled = body.enabled;
    }

    // Fetch servers from proxy API
    const { servers } = await getServers(filters, undefined, ownerToken);

    // Sort by createdAt desc
    servers.sort((a: any, b: any) => (b.createdAt || 0) - (a.createdAt || 0));

    // Fetch tool templates for matching
    const kimbapCloudApi = new KimbapCloudApiService();
    const templates = await kimbapCloudApi.fetchToolTemplatesWithFallback();

    // Fetch server status from proxy
    let serverStatusMap: { [serverId: string]: ServerStatus } = {};
    try {
      serverStatusMap = await getServersStatus();
    } catch (error) {
      console.error('Failed to fetch server status:', error);
    }

    // Map to response format
    const tools = servers.map((server: any) => {
      // Find matching template
      const matchedTemplate = server.toolTmplId
        ? templates.find((t) => t.toolTmplId === server.toolTmplId)
        : null;

      // Parse capabilities
      let functions: ToolFunction[] = [];
      let resources: ToolResource[] = [];

      try {
        const capabilities = JSON.parse(server.capabilities);

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
        console.error('Failed to parse capabilities:', error);
      }

      // Get server status
      const runningState = serverStatusMap[server.serverId] ?? ServerStatus.Offline;

      return {
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
        lastUsed: 0,
        createdAt: server.createdAt || 0,
      };
    });

    return ApiResponse.success({ tools });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

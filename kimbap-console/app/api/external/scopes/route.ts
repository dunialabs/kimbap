import { NextRequest } from 'next/server';
import { getAvailableServersCapabilities } from '@/lib/proxy-api';
import { ApiResponse } from '../lib/response';
import { authenticate } from '../lib/auth';

export const dynamic = 'force-dynamic';

interface ScopeFunc {
  funcName: string;
  enabled: boolean;
}

interface ScopeResource {
  uri: string;
  enabled: boolean;
}

interface ScopeItem {
  toolId: string;
  name: string;
  enabled: boolean;
  functions: ScopeFunc[];
  resources: ScopeResource[];
}

interface ScopesResponse {
  scopes: ScopeItem[];
}

/**
 * POST /api/external/scopes
 *
 * Get all available tools with their functions and resources for permission configuration.
 * Requires authentication (owner only).
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Get all available servers capabilities (using token directly)
    const capabilities = await getAvailableServersCapabilities(undefined, user.accessToken);

    // Map capabilities to scopes format
    const scopes: ScopeItem[] = [];

    for (const [serverId, serverCap] of Object.entries(capabilities)) {
      // Extract functions
      const functions: ScopeFunc[] = [];
      if (serverCap.tools && typeof serverCap.tools === 'object') {
        for (const [funcName, config] of Object.entries(serverCap.tools)) {
          functions.push({
            funcName,
            enabled: config.enabled || false,
          });
        }
      }

      // Extract resources
      const resources: ScopeResource[] = [];
      if (serverCap.resources && typeof serverCap.resources === 'object') {
        for (const [uri, config] of Object.entries(serverCap.resources)) {
          resources.push({
            uri,
            enabled: config.enabled || false,
          });
        }
      }

      scopes.push({
        toolId: serverId,
        name: serverCap.serverName,
        enabled: serverCap.enabled,
        functions,
        resources,
      });
    }

    const responseData: ScopesResponse = { scopes };

    return ApiResponse.success(responseData);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

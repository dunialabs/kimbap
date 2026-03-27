import { getUserAvailableServersCapabilities } from '@/lib/proxy-api';

export class PermissionsFetchError extends Error {
  constructor(message: string, public cause?: unknown) {
    super(message);
    this.name = 'PermissionsFetchError';
  }
}

export interface ToolFunc {
  funcName: string;
  enabled: boolean;
}

export interface ToolResource {
  uri: string;
  enabled: boolean;
}

export interface TokenPermission {
  toolId: string;
  toolName: string;
  functions: ToolFunc[];
  resources: ToolResource[];
}

export interface TokenItem {
  tokenId: string;
  name: string;
  role: number; // 1=owner, 2=admin, 3=member
  notes: string;
  lastUsed: number;
  createdAt: number;
  expiresAt: number;
  rateLimit: number;
  permissions: TokenPermission[];
  namespace: string;
  tags: string[];
}

/**
 * Get user permissions from proxy API
 */
export async function getUserPermissions(
  userId: string,
  accessToken: string,
  serverNameMap: Record<string, string>
): Promise<TokenPermission[]> {
  try {
    // Get user capabilities from proxy API (using token directly)
    const userCapabilities = await getUserAvailableServersCapabilities(userId, undefined, accessToken);

    const permissions: TokenPermission[] = [];

    // Process each server capability
    for (const [serverId, serverCapabilities] of Object.entries(userCapabilities)) {
      // Only include servers that belong to this proxyId
      if (!serverNameMap[serverId]) {
        continue;
      }

      // Extract functions
      const functions: ToolFunc[] = [];
      if (serverCapabilities.tools && typeof serverCapabilities.tools === 'object') {
        for (const [funcName, config] of Object.entries(serverCapabilities.tools)) {
          functions.push({
            funcName,
            enabled: (config as { enabled?: boolean }).enabled === true,
          });
        }
      }

      // Extract resources
      const resources: ToolResource[] = [];
      if (serverCapabilities.resources && typeof serverCapabilities.resources === 'object') {
        for (const [uri, config] of Object.entries(serverCapabilities.resources)) {
          resources.push({
            uri,
            enabled: (config as { enabled?: boolean }).enabled === true,
          });
        }
      }

      permissions.push({
        toolId: serverId,
        toolName: serverNameMap[serverId],
        functions,
        resources,
      });
    }

    return permissions;
  } catch (error) {
    console.error('Failed to get user permissions:', error);
    throw new PermissionsFetchError(`Failed to get user permissions for ${userId}`, error);
  }
}

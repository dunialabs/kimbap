import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getServersStatus, ServerStatus, getServers } from '@/lib/proxy-api';
import { KimbapCloudApiService } from '@/lib/KimbapCloudApiService';
import { UserService } from '@/lib/user-service';
import { CryptoUtils } from '@/lib/crypto';

interface Request10006 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    handleType: number;
    proxyId?: number;
  };
}

interface Credential {
  name: string;
  description: string;
  dataType: number;
  key: string;
  value: string;
  options: Array<{ key: string; value: string }>;
  selected: { key: string; value: string };
}

interface ToolFunction {
  funcName: string;
  enabled: boolean;
  dangerLevel: number; // 0: No validation, 1: hint only, 2: validation required
}

interface ToolResource {
  uri: string;
  enabled: boolean;
}

interface Tool {
  toolTmplId: string;
  toolType: number;
  name: string;
  description: string;
  tags: string[];
  authtags: string[];
  credentials: Credential[];
  toolFuncs: ToolFunction[];
  toolResources: ToolResource[];
  lastUsed: number;
  enabled: boolean;
  toolId: string;
  runningState: number; // 0-Online, 1-Offline, 2-Connecting, 3-Error
  authType: number; // Authentication type for the tool
  allowUserInput: number; // 1-True, else-False
  category: number; // 1-Template, 2-Custom Remote HTTP, 3-REST API, 4-Skills, 5-Custom Stdio
  restApiConfig?: any; // REST API configuration (only for category === 3)
  customRemoteConfig?: string; // Custom MCP URL configuration (only for category === 2)
  stdioConfig?: string; // Custom Stdio configuration (only for category === 5)
  lazyStartEnabled: boolean; // true or false
  publicAccess: boolean; // true or false
  anonymousAccess: boolean; // true or false
  anonymousRateLimit: number; // Rate limit for anonymous access (req/min per IP)
}

/** Typed shape of server objects returned by the proxy API (getServers). */
interface ProxyServer {
  serverId: string;
  serverName: string;
  enabled: boolean;
  allowUserInput: boolean;
  capabilities: string;
  launchConfig: string;
  lazyStartEnabled: boolean;
  publicAccess: boolean;
  anonymousAccess?: boolean;
  anonymousRateLimit?: number;
  createdAt?: number;
  category: number;
  configTemplate?: string;
  toolTmplId?: string;
  authType?: number;
}

interface Response10006Data {
  toolList: Tool[];
}

export async function handleProtocol10006(body: Request10006): Promise<Response10006Data> {
  const { handleType, proxyId } = body.params || {};

  // Validate handleType
  if (!handleType || handleType < 1 || handleType > 4) {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'handleType' });
  }

  try {
    let whereCondition: any = {};

    // Add proxyId filter if provided
    if (proxyId !== undefined) {
      whereCondition.proxyId = proxyId;
    }

    switch (handleType) {
      case 1: // Get all servers
        // No additional filtering needed
        break;

      case 2: // Get enabled servers only
        whereCondition.enabled = true;
        break;

      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'handleType' });
    }

    // Query servers from proxy API
    const { servers } = await getServers(
      {
        ...(whereCondition.proxyId && { proxyId: whereCondition.proxyId }),
        ...(whereCondition.enabled !== undefined && { enabled: whereCondition.enabled }),
      },
      body.common.userid,
    );

    // Sort by createdAt desc to match original ordering
    servers.sort((a, b) => (b.createdAt || 0) - (a.createdAt || 0));

    // Fetch tool templates for matching with fallback
    const kimbapCloudApi = new KimbapCloudApiService();
    const templates = await kimbapCloudApi.fetchToolTemplatesWithFallback();

    // Fetch server status from proxy
    let serverStatusMap: { [serverId: string]: ServerStatus } = {};
    try {
      serverStatusMap = await getServersStatus(body.common.userid);
    } catch (error) {
      console.error('Failed to fetch server status:', error);
      // Continue with empty status map if fetch fails
    }

    // Map to response format with template matching (async for REST API config decryption)
    const toolList: Tool[] = await Promise.all(
      (servers as ProxyServer[]).map(async (server) => {
        // Find matching template by toolTmplId
        const matchedTemplate = server.toolTmplId
          ? templates.find((t) => t.toolTmplId === server.toolTmplId)
          : null;

        // Parse capabilities for toolFuncs and toolResources
        let toolFuncs: ToolFunction[] = [];
        let toolResources: ToolResource[] = [];

        try {
          const capabilities = JSON.parse(server.capabilities);

          // Extract toolFuncs from capabilities.tools
          if (capabilities.tools) {
            toolFuncs = Object.entries(capabilities.tools).map(
              ([funcName, config]: [string, any]) => ({
                funcName,
                enabled: config.enabled || false,
                dangerLevel: config.dangerLevel !== undefined ? config.dangerLevel : 0, // Default to 0 if not specified
              }),
            );
          }

          // Extract toolResources from capabilities.resources
          if (capabilities.resources) {
            toolResources = Object.entries(capabilities.resources).map(
              ([uri, config]: [string, any]) => ({
                uri,
                enabled: config.enabled || false,
              }),
            );
          }
        } catch (error) {
          // If parsing fails, use empty arrays
          console.error('Failed to parse capabilities:', error);
        }

        // Build credentials with empty values as placeholders
        const credentials: Credential[] =
          matchedTemplate?.credentials?.map((cred) => ({
            name: cred.name,
            description: cred.description,
            dataType: cred.dataType,
            key: cred.key,
            value: '', // Empty value as placeholder
            options: [], // Empty options
            selected: { key: '', value: '' }, // Empty selected
          })) || [];

        // Get server status from the map (default to Offline if not found)
        const runningState = serverStatusMap[server.serverId] ?? ServerStatus.Offline;

        // Restore restApiConfig for category === 3 (REST API tools)
        let restApiConfig: any = undefined;
        let customRemoteConfig: any = undefined;
        let stdioConfig: any = undefined;
        if (server.category === 3) {
          const configTemplate = server.configTemplate;
          if (configTemplate) {
            try {
              const template = JSON.parse(configTemplate);
              restApiConfig = template.apis?.[0] || null;

              if (restApiConfig) {
                const launchConfigStr = server.launchConfig;
                if (launchConfigStr) {
                  if (server.allowUserInput) {
                    const auth = JSON.parse(launchConfigStr).auth;
                    if (auth) {
                      restApiConfig.auth = auth;
                    }
                  } else {
                    // Try to get accessToken by userid to decrypt launchConfig
                    const accessToken = await UserService.getAccessTokenByUserId(
                      body.common.userid,
                    );
                    if (accessToken) {
                      try {
                        const decryptedStr = await CryptoUtils.decryptDataFromString(
                          launchConfigStr,
                          accessToken,
                        );
                        const decryptedConfig = JSON.parse(decryptedStr);
                        // Restore auth field from decrypted launchConfig
                        restApiConfig.auth = decryptedConfig.auth;
                      } catch (decryptError) {
                        console.error(
                          'Failed to decrypt launchConfig for REST API tool:',
                          decryptError,
                        );
                        // Continue without auth - graceful degradation
                      }
                    }
                  }
                }
              }
            } catch (parseError) {
              console.error('Failed to parse configTemplate for REST API tool:', parseError);
            }
          }
        } else if (server.category === 2) {
          if (server.allowUserInput) {
            customRemoteConfig = server.configTemplate;
          } else {
            customRemoteConfig = server.launchConfig;
          }
        } else if (server.category === 5) {
          if (server.allowUserInput) {
            stdioConfig = server.configTemplate;
          } else {
            stdioConfig = server.launchConfig;
          }
        }

        return {
          // Most fields from server table:
          toolTmplId: matchedTemplate?.toolTmplId || '', // server.tool_tmpl_id (from template matching)
          toolType: matchedTemplate?.toolType || 0, // From matched template
          name: server.serverName, // server.server_name
          description: matchedTemplate?.description || '', // From matched template
          tags: matchedTemplate?.tags || [], // From matched template
          authtags: matchedTemplate?.authtags || [], // From matched template
          credentials, // From matched template with placeholder values
          toolFuncs, // TEMPORARILY DISABLED - returning empty array
          toolResources, // TEMPORARILY DISABLED - returning empty array
          lastUsed: 0, // TODO: Implement lastUsed tracking in the future (hardcoded to 0)
          enabled: server.enabled, // server.enabled
          toolId: server.serverId, // server.server_id
          runningState, // Server running state from proxy
          authType: matchedTemplate?.authType || 0, // Authentication type from matched template
          allowUserInput: server.allowUserInput ? 1 : 0, // Convert boolean to number: true->1, false->0
          category: server.category, // Category from server table
          customRemoteConfig, // Custom MCP URL configuration (only for category === 2)
          stdioConfig, // Custom Stdio configuration (only for category === 5)
          restApiConfig, // REST API configuration (only for category === 3)
          lazyStartEnabled: server.lazyStartEnabled, // true or false
          publicAccess: server.publicAccess, // true or false
          anonymousAccess: server.anonymousAccess ?? false,
          anonymousRateLimit: server.anonymousRateLimit ?? 10,
        };
      }),
    );

    return {
      toolList,
    };
  } catch (error) {
    console.error('Protocol 10006 error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}

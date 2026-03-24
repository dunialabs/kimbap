import { NextRequest } from 'next/server';
import { randomUUID } from 'crypto';
import { getProxy, countServers, createServer, startMCPServer } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { LicenseService } from '@/license-system';
import { KimbapCloudApiService } from '@/lib/KimbapCloudApiService';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3004, E4001 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface CreateToolInput {
  toolTmplId: string;
  mcpJsonConf: Record<string, any>;
  allowUserInput?: boolean; // Whether this is a personal tool template, default false
  lazyStartEnabled?: boolean; // Whether to enable lazy start, default true
  publicAccess?: boolean; // Whether the tool is available to all users by default, default false
}

/**
 * POST /api/external/tools/basic-mcp
 *
 * Create a basic MCP tool from a template.
 * Requires authentication (owner only).
 * Reference: protocol-10005 handleType=1 (category=1)
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: CreateToolInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate toolTmplId
    const { toolTmplId, mcpJsonConf, allowUserInput, lazyStartEnabled, publicAccess } = body;
    if (!toolTmplId || typeof toolTmplId !== 'string' || !toolTmplId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: toolTmplId');
    }

    // Validate mcpJsonConf
    if (!mcpJsonConf || typeof mcpJsonConf !== 'object') {
      throw new ExternalApiError(E1001, 'Missing required field: mcpJsonConf');
    }

    const ownerToken = user.accessToken;

    // Get proxy info
    const proxy = await getProxy();

    // Check tool creation limit
    const currentToolCountResult = await countServers({ proxyId: proxy.id }, undefined, ownerToken);
    const currentToolCount = currentToolCountResult.count;

    const licenseService = LicenseService.getInstance();
    const limitCheck = await licenseService.checkToolCreationLimit(currentToolCount);

    if (!limitCheck.allowed) {
      throw new ExternalApiError(E4001, 'Tool limit reached');
    }

    // Fetch tool templates and validate toolTmplId
    const kimbapCloudApi = new KimbapCloudApiService();
    const templates = await kimbapCloudApi.fetchToolTemplatesWithFallback();
    const template = templates.find((t) => t.toolTmplId === toolTmplId);

    if (!template) {
      throw new ExternalApiError(E3004, `Template not found: ${toolTmplId}`);
    }

    // Generate server ID
    const serverId = randomUUID().replace(/-/g, '');

    // Get server name from template, add " Personal" suffix when allowUserInput is true
    let serverName = template.name;
    if (allowUserInput) {
      serverName = template.name + ' Personal';
    }

    // Encrypt mcpJsonConf with owner token
    const encryptedConfig = await CryptoUtils.encryptData(JSON.stringify(mcpJsonConf), ownerToken);
    const launchConfig = JSON.stringify(encryptedConfig);

    // Create server
    const serverResult = await createServer(
      {
        serverId,
        serverName,
        enabled: false, // Initially disabled until MCP server starts
        launchConfig,
        capabilities: { tools: {}, resources: {} },
        allowUserInput: Boolean(allowUserInput),
        proxyId: proxy.id,
        toolTmplId,
        authType: template.authType || 0,
        configTemplate: JSON.stringify(template),
        category: 1, // Template-based tool
        toolDefaultConfig: template.toolDefaultConfig ? JSON.stringify(template.toolDefaultConfig) : undefined,
        oAuthConfig: template.oAuthConfig ? JSON.stringify(template.oAuthConfig) : undefined,
        lazyStartEnabled: lazyStartEnabled ?? true,
        publicAccess: publicAccess ?? false,
      },
      undefined,
      ownerToken
    );

    const server = serverResult.server;

    // Start the MCP server
    let isStarted = false;
    try {
      await startMCPServer(server.serverId, ownerToken);
      isStarted = true;
    } catch (error) {
      console.error('Failed to start MCP server:', error);
      // Server remains disabled in database since start failed
    }

    return ApiResponse.success(
      {
        toolId: server.serverId,
        isStarted,
      },
      201
    );
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

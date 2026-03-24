import { NextRequest } from 'next/server';
import { randomUUID } from 'crypto';
import { getProxy, countServers, createServer, startMCPServer } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { LicenseService } from '@/license-system';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003, E1004, E4001 } from '../../lib/error-codes';
import { validateHttpsBaseUrl } from '@/lib/rest-api-utils';

export const dynamic = 'force-dynamic';

interface RestApiTool {
  name: string;
  description?: string;
  method: string;
  path: string;
  body?: {
    type?: string;
    schema?: Record<string, any>;
  };
  params?: Record<string, any>;
  headers?: Record<string, string>;
}

interface RestApiAuth {
  type: 'none' | 'bearer' | 'basic' | 'api_key';
  token?: string; // for bearer
  username?: string; // for basic
  password?: string; // for basic
  key?: string; // for api_key
  value?: string; // for api_key
  in?: 'header' | 'query'; // for api_key
}

interface RestApiConfig {
  name?: string;
  baseUrl: string;
  auth?: RestApiAuth;
  tools: RestApiTool[];
}

interface CreateRestApiInput {
  restApiConfig: RestApiConfig;
  allowUserInput?: boolean; // Whether this is a personal tool template, default false
  lazyStartEnabled?: boolean;
  publicAccess?: boolean;
}

/**
 * POST /api/external/tools/rest-api
 *
 * Create a REST API tool.
 * Requires authentication (owner only).
 * Reference: protocol-10005 handleType=1 (category=3)
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: CreateRestApiInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate restApiConfig
    const { restApiConfig, allowUserInput, lazyStartEnabled, publicAccess } = body;
    if (!restApiConfig || typeof restApiConfig !== 'object') {
      throw new ExternalApiError(E1001, 'Missing required field: restApiConfig');
    }

    // Validate baseUrl
    const baseUrl = restApiConfig.baseUrl?.trim() ?? '';
    if (baseUrl === '') {
      throw new ExternalApiError(E1003, 'Missing required field: restApiConfig.baseUrl');
    }

    // Validate URL format and security
    const baseUrlIssue = validateHttpsBaseUrl(baseUrl);
    if (baseUrlIssue) {
      throw new ExternalApiError(E1003, baseUrlIssue);
    }

    // Validate tools
    const tools = restApiConfig.tools ?? [];
    if (!Array.isArray(tools) || tools.length === 0) {
      throw new ExternalApiError(E1003, 'restApiConfig.tools must have at least one tool');
    }

    // Validate each tool
    for (let i = 0; i < tools.length; i++) {
      const tool = tools[i];
      if (!tool.name || typeof tool.name !== 'string' || !tool.name.trim()) {
        throw new ExternalApiError(E1003, `restApiConfig.tools[${i}].name is required`);
      }
      if (!tool.method || typeof tool.method !== 'string') {
        throw new ExternalApiError(E1003, `restApiConfig.tools[${i}].method is required`);
      }
      if (!tool.path || typeof tool.path !== 'string') {
        throw new ExternalApiError(E1003, `restApiConfig.tools[${i}].path is required`);
      }
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

    // Generate server ID
    const serverId = randomUUID().replace(/-/g, '');

    // Generate server name, add " Personal" suffix when allowUserInput is true
    let serverName = restApiConfig.name?.trim() || `REST API Tool ${Date.now()}`;
    if (allowUserInput) {
      serverName = serverName + ' Personal';
    }

    // Extract auth from restApiConfig
    const auth = restApiConfig.auth ?? { type: 'none' };

    // Build launchConfig (following protocol-10005 pattern)
    const launchConfig = {
      command: 'docker',
      args: [
        'run',
        '--pull=always',
        '-i',
        '--rm',
        '-e',
        'accessToken',
        '-e',
        'GATEWAY_CONFIG',
        'ghcr.io/dunialabs/mcp-servers/rest-gateway',
      ],
      auth: auth,
    };

    // Build template (restApiConfig without auth)
    const restApiWithoutAuth = { ...restApiConfig };
    delete (restApiWithoutAuth as any).auth;
    const template = {
      apis: [restApiWithoutAuth],
    };

    // For REST API with allowUserInput, do NOT encrypt launchConfig (following protocol-10005 line 368-369)
    // This allows personal tools to have users fill in their own credentials
    let finalLaunchConfig: string;
    if (allowUserInput) {
      finalLaunchConfig = JSON.stringify(launchConfig);
    } else {
      const encryptedConfig = await CryptoUtils.encryptData(
        JSON.stringify(launchConfig),
        ownerToken,
      );
      finalLaunchConfig = JSON.stringify(encryptedConfig);
    }

    // Create server
    const serverResult = await createServer(
      {
        serverId,
        serverName,
        enabled: false,
        launchConfig: finalLaunchConfig,
        capabilities: { tools: {}, resources: {} },
        allowUserInput: Boolean(allowUserInput),
        proxyId: proxy.id,
        toolTmplId: '',
        authType: 1,
        configTemplate: JSON.stringify(template),
        category: 3, // REST API tool
        lazyStartEnabled: lazyStartEnabled ?? true,
        publicAccess: publicAccess ?? false,
      },
      undefined,
      ownerToken,
    );

    const server = serverResult.server;

    // Start the MCP server
    let isStarted = false;
    try {
      await startMCPServer(server.serverId, ownerToken);
      isStarted = true;
    } catch (error) {
      console.error('Failed to start MCP server:', error);
    }

    return ApiResponse.success(
      {
        toolId: server.serverId,
        isStarted,
      },
      201,
    );
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

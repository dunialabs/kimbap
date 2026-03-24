import { NextRequest } from 'next/server';
import { randomUUID } from 'crypto';
import { getProxy, countServers, createServer, startMCPServer } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { LicenseService } from '@/license-system';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1002, E4001 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface CreateCustomStdioInput {
  customStdioConfig: {
    command: string;
    args?: string[];
    env?: Record<string, string>;
    cwd?: string;
  };
  allowUserInput?: boolean; // Whether this is a personal tool template, default false
  lazyStartEnabled?: boolean; // Whether to enable lazy start, default true
  publicAccess?: boolean; // Whether the tool is available to all users by default, default false
  serverName?: string; // Custom server name, defaults to 'Custom Stdio MCP Server'
}

/**
 * POST /api/external/tools/custom-stdio
 *
 * Create a custom stdio MCP tool.
 * Requires authentication (owner only).
 * Reference: protocol-10005 handleType=1 (category=5)
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: CreateCustomStdioInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate customStdioConfig
    const {
      customStdioConfig,
      allowUserInput,
      lazyStartEnabled,
      publicAccess,
      serverName: inputServerName,
    } = body;
    if (!customStdioConfig || typeof customStdioConfig !== 'object') {
      throw new ExternalApiError(E1001, 'Missing required field: customStdioConfig');
    }

    // Validate command
    const { command, args, env, cwd } = customStdioConfig;
    if (!command || typeof command !== 'string' || !command.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: customStdioConfig.command');
    }

    if (args !== undefined && !Array.isArray(args)) {
      throw new ExternalApiError(E1002, 'customStdioConfig.args must be an array');
    }

    if (env !== undefined && (typeof env !== 'object' || Array.isArray(env) || env === null)) {
      throw new ExternalApiError(E1002, 'customStdioConfig.env must be an object');
    }

    if (cwd !== undefined && typeof cwd !== 'string') {
      throw new ExternalApiError(E1002, 'customStdioConfig.cwd must be a string');
    }
    const normalizedCwd = cwd?.trim();

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

    if (inputServerName !== undefined && typeof inputServerName !== 'string') {
      throw new ExternalApiError(E1002, 'serverName must be a string');
    }

    let serverName = inputServerName?.trim() || 'Custom Stdio MCP Server';
    if (allowUserInput) {
      serverName = serverName + ' Personal';
    }

    // Build launch config
    const launchConfig = {
      command: command.trim(),
      args: args ?? [],
      env: env ?? {},
      ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
    };

    // Build template (always set for stdio, used when allowUserInput is true)
    const template = {
      command: command.trim(),
      args: args ?? [],
      env: env ?? {},
      ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
    };

    // Encrypt launch config with owner token
    const encryptedConfig = await CryptoUtils.encryptData(JSON.stringify(launchConfig), ownerToken);
    const encryptedLaunchConfig = JSON.stringify(encryptedConfig);

    // Create server
    const serverResult = await createServer(
      {
        serverId,
        serverName,
        enabled: false, // Initially disabled until MCP server starts
        launchConfig: encryptedLaunchConfig,
        capabilities: { tools: {}, resources: {} },
        allowUserInput: Boolean(allowUserInput),
        proxyId: proxy.id,
        toolTmplId: '', // No template for custom stdio
        authType: 1, // Default authType for custom stdio
        configTemplate: JSON.stringify(template),
        category: 5, // Custom Stdio MCP tool
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
      // Server remains disabled in database since start failed
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

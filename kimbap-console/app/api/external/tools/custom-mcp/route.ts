import { NextRequest } from 'next/server';
import { randomUUID } from 'crypto';
import { getProxy, countServers, createServer, startMCPServer } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { LicenseService } from '@/license-system';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003, E1005, E4001 } from '../../lib/error-codes';
import { validateHttpsBaseUrl } from '@/lib/rest-api-utils';

export const dynamic = 'force-dynamic';

interface CreateCustomMcpInput {
  customRemoteConfig?: {
    url: string;
    headers?: Record<string, string>;
  };
  stdioConfig?: {
    command: string;
    args?: string[];
    env?: Record<string, string>;
    cwd?: string;
  };
  allowUserInput?: boolean; // Whether this is a personal tool template, default false
  lazyStartEnabled?: boolean; // Whether to enable lazy start, default true
  publicAccess?: boolean; // Whether the tool is available to all users by default, default false
}

/**
 * POST /api/external/tools/custom-mcp
 *
 * Create a custom MCP tool with remote URL or stdio command.
 * Requires authentication (owner only).
 * Reference: protocol-10005 handleType=1 (category=2)
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: CreateCustomMcpInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    const { customRemoteConfig, stdioConfig, allowUserInput, lazyStartEnabled, publicAccess } =
      body;
    const hasStdioConfig = Boolean(stdioConfig && typeof stdioConfig === 'object');
    const hasRemoteConfig = Boolean(customRemoteConfig && typeof customRemoteConfig === 'object');

    if (!hasRemoteConfig && !hasStdioConfig) {
      throw new ExternalApiError(
        E1001,
        'Missing required field: customRemoteConfig or stdioConfig',
      );
    }

    // Guard: reject dual config (both stdio and URL)
    if (hasRemoteConfig && hasStdioConfig) {
      throw new ExternalApiError(
        E1001,
        'Cannot provide both stdioConfig and customRemoteConfig. Choose one.',
      );
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

    let serverName = '';
    let launchConfig: any = {};
    let template: any = {};

    if (hasStdioConfig && stdioConfig?.command) {
      if (typeof stdioConfig.command !== 'string') {
        throw new ExternalApiError(E1001, 'stdioConfig.command must be a string');
      }
      const command = stdioConfig.command.trim();
      if (!command) {
        throw new ExternalApiError(E1001, 'Missing required field: stdioConfig.command');
      }

      // Validate args and env types
      if (stdioConfig.args !== undefined) {
        if (!Array.isArray(stdioConfig.args)) {
          throw new ExternalApiError(E1001, 'stdioConfig.args must be an array');
        }
        for (const arg of stdioConfig.args) {
          if (typeof arg !== 'string') {
            throw new ExternalApiError(E1001, 'stdioConfig.args must be an array of strings');
          }
        }
      }
      if (stdioConfig.env !== undefined) {
        if (
          typeof stdioConfig.env !== 'object' ||
          Array.isArray(stdioConfig.env) ||
          stdioConfig.env === null
        ) {
          throw new ExternalApiError(E1001, 'stdioConfig.env must be an object');
        }
        for (const [key, value] of Object.entries(stdioConfig.env)) {
          if (typeof value !== 'string') {
            throw new ExternalApiError(
              E1001,
              `stdioConfig.env value for key "${key}" must be a string`,
            );
          }
        }
      }

      if (stdioConfig.cwd !== undefined && typeof stdioConfig.cwd !== 'string') {
        throw new ExternalApiError(E1001, 'stdioConfig.cwd must be a string');
      }
      const normalizedCwd = stdioConfig.cwd?.trim();

      const commandName = command.split('/').pop()?.split('\\').pop() || command;
      serverName = `Custom MCP Server (${commandName})`;
      if (allowUserInput) {
        serverName = serverName + ' Personal';
      }

      launchConfig = {
        command,
        args: stdioConfig.args ?? [],
        env: stdioConfig.env ?? {},
        ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
      };

      if (allowUserInput) {
        template = {
          command,
          args: stdioConfig.args ?? [],
          env: stdioConfig.env ?? {},
          ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
        };
      }
    } else if (hasRemoteConfig && customRemoteConfig) {
      const { url, headers } = customRemoteConfig;
      if (!url || typeof url !== 'string' || !url.trim()) {
        throw new ExternalApiError(E1001, 'Missing required field: customRemoteConfig.url');
      }

      const urlIssue = validateHttpsBaseUrl(url);
      if (urlIssue) {
        throw new ExternalApiError(E1003, urlIssue);
      }
      let parsedUrl: URL;
      try {
        parsedUrl = new URL(url);
      } catch {
        throw new ExternalApiError(E1005, 'Invalid URL format');
      }

      serverName = `Custom MCP Server (${parsedUrl.hostname})`;
      if (allowUserInput) {
        serverName = serverName + ' Personal';
      }

      const configHeaders = headers || {};
      launchConfig = {
        url: url,
        headers: configHeaders,
      };

      if (allowUserInput) {
        template = {
          url: url,
          headers: configHeaders,
        };
      }
    } else {
      throw new ExternalApiError(
        E1001,
        'Missing required field: customRemoteConfig.url or stdioConfig.command',
      );
    }

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
        toolTmplId: '', // No template for custom MCP
        authType: 1, // Default authType for custom MCP
        configTemplate: JSON.stringify(template),
        category: hasStdioConfig ? 5 : 2, // 5=Custom Stdio, 2=Custom Remote HTTP
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

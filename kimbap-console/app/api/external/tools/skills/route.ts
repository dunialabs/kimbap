import { NextRequest } from 'next/server';
import { randomUUID } from 'crypto';
import { getProxy, countServers, createServer, startMCPServer } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { LicenseService } from '@/license-system';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E4001 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface CreateSkillsInput {
  serverName?: string;
  allowUserInput?: boolean; // Whether this is a personal tool template, default false
  lazyStartEnabled?: boolean;
  publicAccess?: boolean;
}

/**
 * POST /api/external/tools/skills
 *
 * Create a Skills-based tool.
 * Skills are stored on Core server at skills/{toolId}/ directory.
 * Requires authentication (owner only).
 * Reference: protocol-10005 handleType=1 (category=4)
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: CreateSkillsInput = {};
    try {
      body = await request.json();
    } catch {
      // Empty body is acceptable for skills
    }

    const { serverName, allowUserInput, lazyStartEnabled, publicAccess } = body;

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
    let finalServerName = serverName?.trim() || 'Skills MCP Server';
    if (allowUserInput) {
      finalServerName = finalServerName + ' Personal';
    }

    // Build launchConfig for Skills MCP Server (following protocol-10005 pattern)
    // The serverId will be used to locate skills directory on Core
    const launchConfig = {
      command: 'docker',
      args: [
        'run',
        '--pull=always',
        '-i',
        '--rm',
        '-v',
        `./skills/${serverId}:/app/skills:ro`,
        '-e',
        'skills_dir=/app/skills',
        'ghcr.io/dunialabs/mcp-servers/skills:latest',
      ],
    };

    // Build template for Skills
    const template = {
      type: 'skills',
      serverName: finalServerName,
    };

    // Encrypt launch config with owner token
    const encryptedConfig = await CryptoUtils.encryptData(JSON.stringify(launchConfig), ownerToken);
    const encryptedLaunchConfig = JSON.stringify(encryptedConfig);

    // Create server
    const serverResult = await createServer(
      {
        serverId,
        serverName: finalServerName,
        enabled: false,
        launchConfig: encryptedLaunchConfig,
        capabilities: { tools: {}, resources: {} },
        allowUserInput: Boolean(allowUserInput),
        proxyId: proxy.id,
        toolTmplId: '',
        authType: 1, // Core requires authType >= 1
        configTemplate: JSON.stringify(template),
        category: 4, // Skills tool
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

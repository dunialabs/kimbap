import { NextRequest } from 'next/server';
import { getServers, updateServer } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { ApiResponse } from '../../../lib/response';
import { authenticate } from '../../../lib/auth';
import { ExternalApiError, E1001, E1003, E3002 } from '../../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface ToolFunctionInput {
  funcName: string;
  enabled: boolean;
  dangerLevel?: number;
  description?: string;
}

interface ToolResourceInput {
  uri: string;
  enabled: boolean;
}

interface UpdateToolInput {
  toolId: string;
  customStdioConfig?: {
    command: string;
    args?: string[];
    env?: Record<string, string>;
    cwd?: string;
  };
  serverName?: string;
  functions?: ToolFunctionInput[];
  resources?: ToolResourceInput[];
  lazyStartEnabled?: boolean;
  publicAccess?: boolean;
  allowUserInput?: boolean | number;
}

interface CapabilitiesPayload {
  tools: Record<string, { enabled: boolean; dangerLevel: number; description: string }>;
  resources: Record<string, { enabled: boolean }>;
}

type UpdateServerRequest = Omit<
  Parameters<typeof updateServer>[1],
  'launchConfig' | 'capabilities'
> & {
  launchConfig?: string;
  capabilities?: string;
};

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const ownerToken = user.accessToken;

    let body: UpdateToolInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: toolId');
    }

    if (!body || typeof body !== 'object') {
      throw new ExternalApiError(E1003, 'Invalid request body');
    }

    if (!body.toolId || typeof body.toolId !== 'string' || !body.toolId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: toolId');
    }

    const { toolId } = body;
    const { servers } = await getServers({ serverId: toolId }, undefined, ownerToken);
    const server = servers.length > 0 ? servers[0] : null;
    if (!server) {
      throw new ExternalApiError(E3002, `Tool not found: ${toolId}`);
    }

    if (server.category !== 5) {
      throw new ExternalApiError(
        E1003,
        'This endpoint only supports Custom Stdio MCP tools (category=5)',
      );
    }

    const serverUpdateData: UpdateServerRequest = {};

    if (body.functions !== undefined || body.resources !== undefined) {
      const capabilities: CapabilitiesPayload = {
        tools: {},
        resources: {},
      };

      if (body.functions !== undefined) {
        if (!Array.isArray(body.functions)) {
          throw new ExternalApiError(E1003, 'functions must be an array');
        }

        for (const func of body.functions) {
          if (!func || typeof func.funcName !== 'string' || !func.funcName.trim()) {
            throw new ExternalApiError(E1003, 'Invalid function config: funcName is required');
          }

          capabilities.tools[func.funcName] = {
            enabled: Boolean(func.enabled),
            dangerLevel: func.dangerLevel ?? 0,
            description: func.description || '',
          };
        }
      }

      if (body.resources !== undefined) {
        if (!Array.isArray(body.resources)) {
          throw new ExternalApiError(E1003, 'resources must be an array');
        }

        for (const resource of body.resources) {
          if (!resource || typeof resource.uri !== 'string' || !resource.uri.trim()) {
            throw new ExternalApiError(E1003, 'Invalid resource config: uri is required');
          }

          capabilities.resources[resource.uri] = {
            enabled: Boolean(resource.enabled),
          };
        }
      }

      serverUpdateData.capabilities = JSON.stringify(capabilities);
    }

    if (body.lazyStartEnabled !== undefined) {
      if (typeof body.lazyStartEnabled !== 'boolean') {
        throw new ExternalApiError(E1003, 'lazyStartEnabled must be a boolean');
      }
      serverUpdateData.lazyStartEnabled = body.lazyStartEnabled;
    }

    if (body.publicAccess !== undefined) {
      if (typeof body.publicAccess !== 'boolean') {
        throw new ExternalApiError(E1003, 'publicAccess must be a boolean');
      }
      serverUpdateData.publicAccess = body.publicAccess;
    }

    if (body.allowUserInput !== undefined) {
      if (
        typeof body.allowUserInput !== 'boolean' &&
        body.allowUserInput !== 0 &&
        body.allowUserInput !== 1
      ) {
        throw new ExternalApiError(E1003, 'allowUserInput must be boolean, 0, or 1');
      }
      serverUpdateData.allowUserInput = Boolean(body.allowUserInput);
    }

    if (body.serverName !== undefined) {
      if (typeof body.serverName !== 'string' || !body.serverName.trim()) {
        throw new ExternalApiError(E1003, 'serverName must be a non-empty string');
      }
      serverUpdateData.serverName = body.serverName.trim();
    }

    if (body.customStdioConfig !== undefined) {
      if (
        typeof body.customStdioConfig !== 'object' ||
        body.customStdioConfig === null ||
        Array.isArray(body.customStdioConfig)
      ) {
        throw new ExternalApiError(E1003, 'customStdioConfig must be an object');
      }

      const { command, args, env, cwd } = body.customStdioConfig;

      // Validate command
      if (!command || typeof command !== 'string' || !command.trim()) {
        throw new ExternalApiError(E1001, 'customStdioConfig.command is required');
      }

      if (args !== undefined && !Array.isArray(args)) {
        throw new ExternalApiError(E1003, 'customStdioConfig.args must be an array');
      }

      if (env !== undefined && (typeof env !== 'object' || Array.isArray(env) || env === null)) {
        throw new ExternalApiError(E1003, 'customStdioConfig.env must be an object');
      }

      if (cwd !== undefined && typeof cwd !== 'string') {
        throw new ExternalApiError(E1003, 'customStdioConfig.cwd must be a string');
      }
      const normalizedCwd = cwd?.trim();

      const stdioLaunchConfig = {
        command: command.trim(),
        args: args ?? [],
        env: env ?? {},
        ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
      };

      const encryptedConfig = await CryptoUtils.encryptData(
        JSON.stringify(stdioLaunchConfig),
        ownerToken,
      );
      serverUpdateData.launchConfig = JSON.stringify(encryptedConfig);

      // Also update configTemplate so user-facing config stays in sync
      serverUpdateData.configTemplate = JSON.stringify(stdioLaunchConfig);
    }

    if (Object.keys(serverUpdateData).length > 0) {
      await updateServer(
        toolId,
        serverUpdateData as Parameters<typeof updateServer>[1],
        undefined,
        ownerToken,
      );
    }

    return ApiResponse.success({ toolId: toolId });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

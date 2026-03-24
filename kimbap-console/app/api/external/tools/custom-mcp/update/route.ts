import { NextRequest } from 'next/server';
import { getServers, updateServer } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { ApiResponse } from '../../../lib/response';
import { authenticate } from '../../../lib/auth';
import { ExternalApiError, E1001, E1003, E3002 } from '../../../lib/error-codes';
import { validateHttpsBaseUrl } from '@/lib/rest-api-utils';

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
  customRemoteConfig?: Record<string, unknown>;
  stdioConfig?: {
    command: string;
    args?: string[];
    env?: Record<string, string>;
    cwd?: string;
  };
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

function normalizeArgsForComparison(args?: string[]): string {
  return JSON.stringify([...(args ?? [])].sort());
}

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

    if (server.category !== 2 && server.category !== 5) {
      throw new ExternalApiError(
        E1003,
        'This endpoint only supports Custom MCP tools (category=2 or category=5)',
      );
    }

    // Guard: reject dual config (both stdio and URL)
    if (body.stdioConfig && body.customRemoteConfig) {
      throw new ExternalApiError(
        E1003,
        'Cannot provide both stdioConfig and customRemoteConfig. Choose one.',
      );
    }

    let currentTransport = server.transportType ?? null;
    if (!currentTransport && server.allowUserInput) {
      try {
        const tpl = JSON.parse(server.configTemplate ?? '{}');
        currentTransport = tpl.command ? 'stdio' : tpl.url ? 'http' : null;
      } catch {
        /* invalid template, skip guard */
      }
    }

    if (currentTransport) {
      const isCurrentlyStdio = currentTransport === 'stdio';

      if (isCurrentlyStdio && body.customRemoteConfig) {
        throw new ExternalApiError(
          E1003,
          'Cannot switch from stdio to URL transport. Delete and recreate the tool instead.',
        );
      }
      if (!isCurrentlyStdio && body.stdioConfig) {
        throw new ExternalApiError(
          E1003,
          'Cannot switch from URL to stdio transport. Delete and recreate the tool instead.',
        );
      }
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

    if (body.customRemoteConfig !== undefined) {
      if (body.customRemoteConfig === null || typeof body.customRemoteConfig !== 'object') {
        throw new ExternalApiError(E1003, 'customRemoteConfig must be an object');
      }
      const url =
        typeof body.customRemoteConfig.url === 'string' ? body.customRemoteConfig.url.trim() : '';
      if (!url) {
        throw new ExternalApiError(E1001, 'Missing required field: customRemoteConfig.url');
      }
      const urlIssue = validateHttpsBaseUrl(url);
      if (urlIssue) {
        throw new ExternalApiError(E1003, urlIssue);
      }

      if (server.allowUserInput) {
        const oldConfigTemplate = JSON.parse(server.configTemplate ?? '{}');
        const oldUrl = (oldConfigTemplate.url ?? '').split('?')[0];
        const newUrl = url.split('?')[0];
        if (oldUrl && newUrl !== oldUrl) {
          throw new ExternalApiError(
            E1003,
            'URL cannot be changed for personal URL tools. Delete and recreate the tool instead.',
          );
        }
      }

      const launchConfig = {
        url,
        headers: body.customRemoteConfig.headers ?? {},
      };

      const encryptedConfig = await CryptoUtils.encryptData(
        JSON.stringify(launchConfig),
        ownerToken,
      );
      serverUpdateData.launchConfig = JSON.stringify(encryptedConfig);

      // Update configTemplate if allowUserInput
      if (server.allowUserInput) {
        serverUpdateData.configTemplate = JSON.stringify(launchConfig);
      }
    }

    if (body.stdioConfig !== undefined) {
      if (body.stdioConfig === null || typeof body.stdioConfig !== 'object') {
        throw new ExternalApiError(E1003, 'stdioConfig must be an object');
      }
      if (typeof body.stdioConfig.command !== 'string' || !body.stdioConfig.command.trim()) {
        throw new ExternalApiError(E1001, 'Missing required field: stdioConfig.command');
      }

      if (body.stdioConfig.args !== undefined) {
        if (!Array.isArray(body.stdioConfig.args)) {
          throw new ExternalApiError(E1003, 'stdioConfig.args must be an array of strings');
        }
        if (body.stdioConfig.args.some((arg: unknown) => typeof arg !== 'string')) {
          throw new ExternalApiError(E1003, 'All stdioConfig.args must be strings');
        }
      }

      if (body.stdioConfig.env !== undefined) {
        if (
          typeof body.stdioConfig.env !== 'object' ||
          body.stdioConfig.env === null ||
          Array.isArray(body.stdioConfig.env)
        ) {
          throw new ExternalApiError(E1003, 'stdioConfig.env must be an object with string values');
        }
        for (const [envKey, envValue] of Object.entries(body.stdioConfig.env)) {
          if (typeof envValue !== 'string') {
            throw new ExternalApiError(E1003, `stdioConfig.env.${envKey} must be a string`);
          }
        }
      }

      if (body.stdioConfig.cwd !== undefined && typeof body.stdioConfig.cwd !== 'string') {
        throw new ExternalApiError(E1003, 'stdioConfig.cwd must be a string');
      }

      // For personal stdio tools, prevent changing the command
      if (server.allowUserInput) {
        const oldConfigTemplate = JSON.parse(server.configTemplate ?? '{}');
        if (
          oldConfigTemplate.command &&
          body.stdioConfig.command.trim() !== oldConfigTemplate.command
        ) {
          throw new ExternalApiError(E1003, 'Command cannot be changed for personal stdio tools');
        }

        // For personal stdio tools, args are also immutable
        if (body.stdioConfig.args !== undefined) {
          const oldArgs = normalizeArgsForComparison(oldConfigTemplate.args);
          const newArgs = normalizeArgsForComparison(body.stdioConfig.args);
          if (oldArgs !== newArgs) {
            throw new ExternalApiError(
              E1003,
              'Args cannot be changed for personal stdio tools. Only env parameters can be modified.',
            );
          }
        }
      }

      const normalizedCwd = body.stdioConfig.cwd?.trim();

      const stdioLaunchConfig = {
        command: body.stdioConfig.command.trim(),
        args: body.stdioConfig.args ?? [],
        env: body.stdioConfig.env ?? {},
        ...(normalizedCwd ? { cwd: normalizedCwd } : {}),
      };

      const encryptedConfig = await CryptoUtils.encryptData(
        JSON.stringify(stdioLaunchConfig),
        ownerToken,
      );
      serverUpdateData.launchConfig = JSON.stringify(encryptedConfig);

      // Update configTemplate if allowUserInput
      if (server.allowUserInput) {
        serverUpdateData.configTemplate = JSON.stringify(stdioLaunchConfig);
      }
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

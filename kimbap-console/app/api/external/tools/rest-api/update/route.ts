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
  restApiConfig?: Record<string, unknown>;
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

    if (server.category !== 3) {
      throw new ExternalApiError(E1003, 'This endpoint only supports REST API tools (category=3)');
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

    if (body.restApiConfig !== undefined) {
      const newBaseUrl = String(body.restApiConfig?.baseUrl ?? '').trim();
      if (newBaseUrl) {
        const baseUrlIssue = validateHttpsBaseUrl(newBaseUrl);
        if (baseUrlIssue) {
          throw new ExternalApiError(E1003, baseUrlIssue);
        }
      }
      serverUpdateData.configTemplate = JSON.stringify({ apis: [body.restApiConfig] });
      const encryptedConfig = await CryptoUtils.encryptData(
        JSON.stringify(body.restApiConfig),
        ownerToken,
      );
      serverUpdateData.launchConfig = JSON.stringify(encryptedConfig);
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

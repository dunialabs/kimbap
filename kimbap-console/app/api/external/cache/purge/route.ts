import { NextRequest } from 'next/server';
import {
  purgeCacheGlobal,
  purgeCacheServer,
  purgeCacheTool,
  purgeCachePrompt,
  purgeCacheResource,
  purgeCacheExact,
} from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1002, E1010 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

type PurgeScope = 'global' | 'server' | 'tool' | 'prompt' | 'resource' | 'exact';

interface PurgeInput {
  scope: PurgeScope;
  serverId?: string;
  toolName?: string;
  promptName?: string;
  uri?: string;
  reason?: string;
  // Exact purge fields
  operation?: string;
  entityId?: string;
  policy?: Record<string, unknown>;
  scopeContext?: Record<string, unknown>;
  requestParams?: unknown;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const ownerToken = user.accessToken;

    let body: PurgeInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: scope');
    }

    if (!body?.scope) {
      throw new ExternalApiError(E1001, 'Missing required field: scope');
    }

    const validScopes: PurgeScope[] = ['global', 'server', 'tool', 'prompt', 'resource', 'exact'];
    if (!validScopes.includes(body.scope)) {
      throw new ExternalApiError(
        E1010,
        'Invalid scope. Must be one of: global, server, tool, prompt, resource, exact',
      );
    }

    const reason = body.reason || 'External API purge';
    const getTrimmedString = (value: unknown, field: string): string | undefined => {
      if (value === undefined) {
        return undefined;
      }
      if (typeof value !== 'string') {
        throw new ExternalApiError(E1002, `Invalid field type: ${field} must be a string`);
      }
      return value.trim();
    };

    switch (body.scope) {
      case 'global':
        await purgeCacheGlobal(reason, undefined, ownerToken);
        break;

      case 'server': {
        const serverScopeId = getTrimmedString(body.serverId, 'serverId');
        if (!serverScopeId) {
          throw new ExternalApiError(E1001, 'Missing required field: serverId');
        }
        await purgeCacheServer(serverScopeId, reason, undefined, ownerToken);
        break;
      }

      case 'tool': {
        const toolScopeServerId = getTrimmedString(body.serverId, 'serverId');
        const toolScopeToolName = getTrimmedString(body.toolName, 'toolName');
        if (!toolScopeServerId) {
          throw new ExternalApiError(E1001, 'Missing required field: serverId');
        }
        if (!toolScopeToolName) {
          throw new ExternalApiError(E1001, 'Missing required field: toolName');
        }
        await purgeCacheTool(toolScopeServerId, toolScopeToolName, reason, undefined, ownerToken);
        break;
      }

      case 'prompt': {
        const promptScopeServerId = getTrimmedString(body.serverId, 'serverId');
        const promptScopeName = getTrimmedString(body.promptName, 'promptName');
        if (!promptScopeServerId) {
          throw new ExternalApiError(E1001, 'Missing required field: serverId');
        }
        if (!promptScopeName) {
          throw new ExternalApiError(E1001, 'Missing required field: promptName');
        }
        await purgeCachePrompt(promptScopeServerId, promptScopeName, reason, undefined, ownerToken);
        break;
      }

      case 'resource': {
        const resourceScopeServerId = getTrimmedString(body.serverId, 'serverId');
        const resourceScopeUri = getTrimmedString(body.uri, 'uri');
        if (!resourceScopeServerId) {
          throw new ExternalApiError(E1001, 'Missing required field: serverId');
        }
        if (!resourceScopeUri) {
          throw new ExternalApiError(E1001, 'Missing required field: uri');
        }
        await purgeCacheResource(
          resourceScopeServerId,
          resourceScopeUri,
          reason,
          undefined,
          ownerToken,
        );
        break;
      }

      case 'exact': {
        const normalizedOperation = getTrimmedString(body.operation, 'operation');
        const normalizedServerId = getTrimmedString(body.serverId, 'serverId');
        const normalizedEntityId = getTrimmedString(body.entityId, 'entityId');

        if (!normalizedOperation) {
          throw new ExternalApiError(E1001, 'Missing required field: operation');
        }
        if (!['tool', 'resource', 'prompt'].includes(normalizedOperation)) {
          throw new ExternalApiError(
            E1010,
            'Invalid operation. Must be one of: tool, resource, prompt',
          );
        }
        if (!normalizedServerId) {
          throw new ExternalApiError(E1001, 'Missing required field: serverId');
        }
        if (!normalizedEntityId) {
          throw new ExternalApiError(E1001, 'Missing required field: entityId');
        }
        if (!body.policy || typeof body.policy !== 'object' || Array.isArray(body.policy)) {
          throw new ExternalApiError(E1001, 'Missing or invalid required field: policy');
        }
        await purgeCacheExact(
          normalizedOperation,
          normalizedServerId,
          normalizedEntityId,
          body.policy,
          body.scopeContext ?? null,
          body.requestParams ?? null,
          reason,
          undefined,
          ownerToken,
        );
        break;
      }
    }

    return ApiResponse.success({ purged: true, scope: body.scope });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

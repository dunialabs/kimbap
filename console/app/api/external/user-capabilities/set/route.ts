import { NextRequest } from 'next/server';
import { updateUser } from '@/lib/proxy-api';
import { parseExternalTokenPermissions } from '@/lib/token-metadata';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface ToolPermission {
  toolId: string;
  functions: Array<{ funcName: string; enabled: boolean }>;
  resources: Array<{ uri: string; enabled: boolean }>;
}

interface SetCapabilitiesInput {
  userId: string;
  permissions: ToolPermission[];
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const ownerToken = user.accessToken;

    let body: SetCapabilitiesInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: userId');
    }

    if (!body || typeof body !== 'object' || Array.isArray(body)) {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    if (!body.userId || typeof body.userId !== 'string' || !body.userId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: userId');
    }

    if (!body.permissions || !Array.isArray(body.permissions)) {
      throw new ExternalApiError(E1001, 'Missing required field: permissions');
    }

    for (const perm of body.permissions) {
      if (!perm || typeof perm !== 'object') {
        throw new ExternalApiError(E1003, 'Invalid permission: each item must be an object');
      }
      if (!perm.toolId || typeof perm.toolId !== 'string') {
        throw new ExternalApiError(E1003, 'Invalid permission: toolId is required');
      }

      if (perm.functions !== undefined) {
        if (!Array.isArray(perm.functions)) {
          throw new ExternalApiError(E1003, 'Invalid permission: functions must be an array');
        }
        for (const func of perm.functions) {
          if (!func || typeof func !== 'object' || !func.funcName || typeof func.funcName !== 'string') {
            throw new ExternalApiError(E1003, 'Invalid function: funcName is required');
          }
        }
      }

      if (perm.resources !== undefined) {
        if (!Array.isArray(perm.resources)) {
          throw new ExternalApiError(E1003, 'Invalid permission: resources must be an array');
        }
        for (const res of perm.resources) {
          if (!res || typeof res !== 'object' || !res.uri || typeof res.uri !== 'string') {
            throw new ExternalApiError(E1003, 'Invalid resource: uri is required');
          }
        }
      }
    }

    let parsedPermissions;
    try {
      parsedPermissions = parseExternalTokenPermissions(body.permissions);
    } catch (e) {
      throw new ExternalApiError(E1003, (e as Error).message);
    }

    await updateUser(
      body.userId,
      {
        permissions: JSON.stringify(parsedPermissions),
      },
      undefined,
      ownerToken
    );

    return ApiResponse.success({ userId: body.userId }, 200, request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

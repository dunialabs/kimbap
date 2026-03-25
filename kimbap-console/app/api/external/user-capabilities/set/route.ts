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

    if (!body?.userId || typeof body.userId !== 'string' || !body.userId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: userId');
    }

    if (!body.permissions || !Array.isArray(body.permissions)) {
      throw new ExternalApiError(E1001, 'Missing required field: permissions');
    }

    for (const perm of body.permissions) {
      if (!perm.toolId || typeof perm.toolId !== 'string') {
        throw new ExternalApiError(E1003, 'Invalid permission: toolId is required');
      }

      if (perm.functions && Array.isArray(perm.functions)) {
        for (const func of perm.functions) {
          if (!func.funcName || typeof func.funcName !== 'string') {
            throw new ExternalApiError(E1003, 'Invalid function: funcName is required');
          }
        }
      }

      if (perm.resources && Array.isArray(perm.resources)) {
        for (const res of perm.resources) {
          if (!res.uri || typeof res.uri !== 'string') {
            throw new ExternalApiError(E1003, 'Invalid resource: uri is required');
          }
        }
      }
    }

    const parsedPermissions = parseExternalTokenPermissions(body.permissions);

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

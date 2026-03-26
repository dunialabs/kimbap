import { NextRequest } from 'next/server';
import { getUsers, getProxy, updateUser } from '@/lib/proxy-api';
import { upsertTokenMetadata, validateMetadataInput, normalizeNamespace, normalizeTags, parseExternalTokenPermissions } from '@/lib/token-metadata';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003, E2003, E3003 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface TokenPermission {
  toolId: string;
  functions?: Array<{
    funcName: string;
    enabled: boolean;
  }>;
  resources?: Array<{
    uri: string;
    enabled: boolean;
  }>;
}

interface UpdateTokenInput {
  tokenId: string;
  name?: string;
  notes?: string;
  permissions?: TokenPermission[];
  namespace?: string;
  tags?: string[];
}

/**
 * POST /api/external/tokens/update
 *
 * Update an existing access token.
 * Requires authentication (owner or admin).
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: UpdateTokenInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    if (!body || typeof body !== 'object' || Array.isArray(body)) {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate tokenId
    const { tokenId } = body;
    if (!tokenId || typeof tokenId !== 'string' || !tokenId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: tokenId');
    }

    const ownerToken = user.accessToken;

    // Get proxy info for proxyId filter
    const proxy = await getProxy();

    // Check if token exists
    const { users } = await getUsers({ userId: tokenId, proxyId: proxy.id }, undefined, ownerToken);
    const existingUser = users.length > 0 ? users[0] : null;

    if (!existingUser) {
      throw new ExternalApiError(E3003, `Token not found: ${tokenId}`);
    }

    // Cannot modify owner token
    if (existingUser.role === 1) {
      throw new ExternalApiError(E2003, 'Permission denied: cannot modify owner token');
    }

    if (body.namespace !== undefined && typeof body.namespace !== 'string') {
      throw new ExternalApiError(E1003, 'Invalid field value: namespace must be a string');
    }
    if (body.tags !== undefined) {
      if (!Array.isArray(body.tags)) {
        throw new ExternalApiError(E1003, 'Invalid field value: tags must be an array');
      }
      if (body.tags.some((tag: unknown) => typeof tag !== 'string')) {
        throw new ExternalApiError(E1003, 'Invalid field value: tags must be an array of strings');
      }
    }

    if (body.namespace !== undefined || body.tags !== undefined) {
      const metaValidation = validateMetadataInput({ namespace: body.namespace, tags: body.tags });
      if (metaValidation) {
        throw new ExternalApiError(E1003, metaValidation);
      }
    }

    if (body.permissions !== undefined && !Array.isArray(body.permissions)) {
      throw new ExternalApiError(E1003, 'Invalid field value: permissions must be an array');
    }
    if (body.permissions) {
      for (const perm of body.permissions) {
        if (!perm || typeof perm !== 'object' || typeof perm.toolId !== 'string' || !perm.toolId.trim()) {
          throw new ExternalApiError(E1003, 'Invalid field value: permissions[] items must have a non-empty toolId string');
        }
        if (perm.functions !== undefined) {
          if (!Array.isArray(perm.functions)) {
            throw new ExternalApiError(E1003, 'Invalid field value: permissions[].functions must be an array');
          }
          for (const func of perm.functions) {
            if (!func || typeof func !== 'object' || typeof func.funcName !== 'string' || !func.funcName.trim()) {
              throw new ExternalApiError(E1003, 'Invalid field value: permissions[].functions[] items must have a non-empty funcName string');
            }
          }
        }
        if (perm.resources !== undefined) {
          if (!Array.isArray(perm.resources)) {
            throw new ExternalApiError(E1003, 'Invalid field value: permissions[].resources must be an array');
          }
          for (const res of perm.resources) {
            if (!res || typeof res !== 'object' || typeof res.uri !== 'string' || !res.uri.trim()) {
              throw new ExternalApiError(E1003, 'Invalid field value: permissions[].resources[] items must have a non-empty uri string');
            }
          }
        }
      }
    }

    // Build update data - only include provided fields
    const updateData: any = {};

    if (body.name !== undefined) {
      updateData.name = body.name;
    }

    if (body.notes !== undefined) {
      updateData.notes = body.notes;
    }

    if (body.permissions !== undefined && body.permissions !== null) {
      const parsedPermissions = parseExternalTokenPermissions(body.permissions);
      updateData.permissions = JSON.stringify(parsedPermissions);
    }

    // Only call update if there's something to update
    if (Object.keys(updateData).length > 0) {
      await updateUser(tokenId, updateData, undefined, ownerToken);
    }

    if (body.namespace !== undefined || body.tags !== undefined) {
      const metadataInput = {
        ...(body.namespace !== undefined ? { namespace: normalizeNamespace(body.namespace) } : {}),
        ...(body.tags !== undefined ? { tags: normalizeTags(body.tags) } : {}),
      };

      await upsertTokenMetadata(proxy.id, tokenId, metadataInput);
    }

    return ApiResponse.success({
      tokenId,
      message: 'Token updated successfully',
    }, 200, request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

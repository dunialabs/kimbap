import { NextRequest } from 'next/server';
import { getProxy, getUsers, updateUser, getUserAvailableServersCapabilities } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003 } from '../../lib/error-codes';
import {
  upsertTokenMetadata,
  getTokenMetadataMap,
  validateMetadataInput,
  normalizeNamespace,
  normalizeTags,
  applyTagsOperation,
  parseExternalTokenPermissions,
  mergeParsedPermissions,
} from '@/lib/token-metadata';
import type { TagsMode } from '@/lib/token-metadata';

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

interface BatchUpdateRequest {
  tokenIds: string[];
  permissions?: TokenPermission[];
  permissionsMode?: 'replace' | 'merge';
  namespace?: string;
  tags?: string[];
  tagsMode?: 'replace' | 'add' | 'remove' | 'clear';
}

interface BatchUpdateFailure {
  tokenId: string;
  error: string;
}

interface BatchUpdateResponse {
  updatedCount: number;
  failedCount: number;
  failures: BatchUpdateFailure[];
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);

    let body: BatchUpdateRequest;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: tokenIds');
    }

    const { tokenIds, permissions, permissionsMode, namespace, tags, tagsMode } = body;

    if (!tokenIds || !Array.isArray(tokenIds) || tokenIds.length === 0) {
      throw new ExternalApiError(E1001, 'Missing required field: tokenIds');
    }

    if (tokenIds.some((tokenId) => typeof tokenId !== 'string' || !tokenId.trim())) {
      throw new ExternalApiError(E1003, 'Invalid field value: tokenIds must be non-empty strings');
    }

    if (permissionsMode && permissionsMode !== 'replace' && permissionsMode !== 'merge') {
      throw new ExternalApiError(E1003, 'Invalid field value: permissionsMode must be replace or merge');
    }

    if (tagsMode && tagsMode !== 'replace' && tagsMode !== 'add' && tagsMode !== 'remove' && tagsMode !== 'clear') {
      throw new ExternalApiError(E1003, 'Invalid field value: tagsMode must be replace, add, remove, or clear');
    }

    if (tags !== undefined && !Array.isArray(tags)) {
      throw new ExternalApiError(E1003, 'Invalid field value: tags must be an array');
    }

    if (tags !== undefined && tags.some((t: unknown) => typeof t !== 'string')) {
      throw new ExternalApiError(E1003, 'Invalid field value: tags must be an array of strings');
    }

    if (namespace !== undefined && typeof namespace !== 'string') {
      throw new ExternalApiError(E1003, 'Invalid field value: namespace must be a string');
    }

    if (permissions !== undefined && !Array.isArray(permissions)) {
      throw new ExternalApiError(E1003, 'Invalid field value: permissions must be an array');
    }

    if (tagsMode && tagsMode !== 'clear' && tags === undefined) {
      throw new ExternalApiError(E1001, 'Missing required field: tags');
    }

    const shouldUpdatePermissions = permissions !== undefined;
    const shouldUpdateMetadata = namespace !== undefined || tags !== undefined || tagsMode === 'clear';

    if (!shouldUpdatePermissions && !shouldUpdateMetadata) {
      throw new ExternalApiError(E1001, 'Missing required field: permissions, namespace, or tags');
    }

    const normalizedNamespace = namespace !== undefined ? normalizeNamespace(namespace) : undefined;
    const normalizedTags = tags !== undefined ? normalizeTags(tags) : undefined;
    const resolvedTagsMode: TagsMode = tagsMode || 'replace';
    const resolvedPermissionsMode = permissionsMode || 'replace';
    const normalizedTokenIds = tokenIds.map((tokenId) => tokenId.trim());
    const parsedPermissions = shouldUpdatePermissions
      ? parseExternalTokenPermissions(permissions)
      : undefined;

    if (namespace !== undefined || tags !== undefined) {
      const metadataValidationError = validateMetadataInput({
        ...(normalizedNamespace !== undefined ? { namespace: normalizedNamespace } : {}),
        ...(normalizedTags !== undefined ? { tags: normalizedTags } : {}),
      });
      if (metadataValidationError) {
        throw new ExternalApiError(E1003, `Invalid field value: ${metadataValidationError}`);
      }
    }

    const proxy = await getProxy();
    const ownerToken = user.accessToken;
    const { users: proxyUsers } = await getUsers({ proxyId: proxy.id }, undefined, ownerToken);
    const userMap = new Map(proxyUsers.map((u) => [u.userId, u]));
    const metadataMap =
      shouldUpdateMetadata && resolvedTagsMode !== 'replace' && resolvedTagsMode !== 'clear' && normalizedTags !== undefined
        ? await getTokenMetadataMap(proxy.id, normalizedTokenIds)
        : null;

    let updatedCount = 0;
    let failedCount = 0;
    const failures: BatchUpdateFailure[] = [];

    for (const tokenId of normalizedTokenIds) {

      try {
        const targetUser = userMap.get(tokenId);

        if (!targetUser) {
          failedCount++;
          failures.push({ tokenId, error: 'Token not found' });
          continue;
        }

        if (targetUser.role === 1) {
          failedCount++;
          failures.push({ tokenId, error: 'Cannot bulk update owner token' });
          continue;
        }

        if (shouldUpdatePermissions) {
          if (resolvedPermissionsMode === 'replace') {
            await updateUser(tokenId, { permissions: JSON.stringify(parsedPermissions || {}) }, undefined, ownerToken);
          } else {
            const existingCapabilities = await getUserAvailableServersCapabilities(tokenId, undefined, ownerToken);
            const mergedPermissions = mergeParsedPermissions(existingCapabilities, parsedPermissions || {});
            await updateUser(tokenId, { permissions: JSON.stringify(mergedPermissions) }, undefined, ownerToken);
          }
        }

        if (shouldUpdateMetadata) {
          const metadataUpdate: { namespace?: string; tags?: string[] } = {};

          if (normalizedNamespace !== undefined) {
            metadataUpdate.namespace = normalizedNamespace;
          }

          if (tagsMode === 'clear') {
            metadataUpdate.tags = [];
          } else if (normalizedTags !== undefined) {
            if (resolvedTagsMode === 'replace') {
              metadataUpdate.tags = normalizedTags;
            } else {
              const currentMeta = metadataMap?.get(tokenId) || { namespace: 'default', tags: [] };
              metadataUpdate.tags = applyTagsOperation(currentMeta.tags, normalizedTags, resolvedTagsMode);
            }
          }

          if (Object.keys(metadataUpdate).length > 0) {
            await upsertTokenMetadata(proxy.id, tokenId, metadataUpdate);
          }
        }

        updatedCount++;
      } catch (error) {
        failedCount++;
        failures.push({
          tokenId,
          error: error instanceof Error ? error.message : 'Unknown error',
        });
      }
    }

    const responseData: BatchUpdateResponse = {
      updatedCount,
      failedCount,
      failures,
    };

    return ApiResponse.success(responseData, 200, request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

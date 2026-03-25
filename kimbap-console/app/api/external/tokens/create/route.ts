import { NextRequest } from 'next/server';
import { getProxy, createUser, deleteUser } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { hashToken } from '@/lib/auth';

import { prisma } from '@/lib/prisma';
import {
  upsertTokenMetadata,
  validateMetadataInput,
  normalizeNamespace,
  normalizeTags,
  parseExternalTokenPermissions,
} from '@/lib/token-metadata';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E1003, E4007, E5001 } from '../../lib/error-codes';

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

interface CreateTokenInput {
  name: string;
  role: number; // 2=admin, 3=member
  notes?: string;
  expiresAt?: number;
  rateLimit?: number;
  permissions?: TokenPermission[];
  namespace?: string;
  tags?: string[];
}

interface CreateTokenResult {
  tokenId: string;
  accessToken: string;
  name: string;
  role: number;
  createdAt: number;
  namespace: string;
  tags: string[];
  warning?: string;
}

interface BatchCreateRequest {
  tokens: CreateTokenInput[];
}

interface BatchCreateResponse {
  tokens: CreateTokenResult[];
}

/**
 * POST /api/external/tokens/create
 *
 * Batch create access tokens.
 * Requires authentication (owner or admin).
 * The owner token from Authorization header is used directly (no masterPwd needed).
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: BatchCreateRequest;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    const MAX_BATCH_SIZE = 50;

    // Validate tokens array
    const { tokens } = body;
    if (!tokens || !Array.isArray(tokens) || tokens.length === 0) {
      throw new ExternalApiError(E1001, 'Missing required field: tokens');
    }
    if (tokens.length > MAX_BATCH_SIZE) {
      throw new ExternalApiError(E1003, `Invalid field value: tokens array exceeds maximum batch size of ${MAX_BATCH_SIZE}`);
    }

    // Validate each token input
    for (let i = 0; i < tokens.length; i++) {
      const t = tokens[i];
      if (!t.name || typeof t.name !== 'string' || !t.name.trim()) {
        throw new ExternalApiError(E1001, `Missing required field: tokens[${i}].name`);
      }
      if (t.role === undefined || t.role === null) {
        throw new ExternalApiError(E1001, `Missing required field: tokens[${i}].role`);
      }
      if (t.role === 1) {
        throw new ExternalApiError(E4007, `Cannot create owner role: tokens[${i}].role`);
      }
      if (t.role !== 2 && t.role !== 3) {
        throw new ExternalApiError(E1003, `Invalid field value: tokens[${i}].role must be 2(admin) or 3(member)`);
      }
      if (t.namespace !== undefined && typeof t.namespace !== 'string') {
        throw new ExternalApiError(E1003, `Invalid field value: tokens[${i}].namespace must be a string`);
      }
      if (t.tags !== undefined) {
        if (!Array.isArray(t.tags)) {
          throw new ExternalApiError(E1003, `Invalid field value: tokens[${i}].tags must be an array`);
        }
        if (t.tags.some((tag: unknown) => typeof tag !== 'string')) {
          throw new ExternalApiError(E1003, `Invalid field value: tokens[${i}].tags must be an array of strings`);
        }
      }
      if (t.namespace !== undefined || t.tags !== undefined) {
        // validateMetadataInput internally normalizes before validating
        const metadataValidationError = validateMetadataInput({
          namespace: t.namespace,
          tags: t.tags,
        });
        if (metadataValidationError) {
          throw new ExternalApiError(E1003, `Invalid field value: tokens[${i}] metadata - ${metadataValidationError}`);
        }
      }
      if (t.permissions !== undefined && !Array.isArray(t.permissions)) {
        throw new ExternalApiError(E1003, `Invalid field value: tokens[${i}].permissions must be an array`);
      }
    }

    const proxy = await getProxy();
    // The owner token from the Authorization header
    const ownerToken = user.accessToken;

    // Create tokens — per-token errors are caught so partial results are returned
    const results: CreateTokenResult[] = [];
    const errors: Array<{ index: number; name: string; error: string }> = [];

    for (let idx = 0; idx < tokens.length; idx++) {
      const tokenInput = tokens[idx];
      try {
        const accessToken = CryptoUtils.generateToken();
        const userId = await CryptoUtils.calculateUserId(accessToken);
        const encryptedToken = await CryptoUtils.encryptData(accessToken, ownerToken);
        const parsedPermissions = parseExternalTokenPermissions(tokenInput.permissions);
        const createdAt = Math.floor(Date.now() / 1000);

        await createUser({
          userId,
          status: 1,
          role: tokenInput.role,
          permissions: JSON.stringify(parsedPermissions),
          serverApiKeys: JSON.stringify({}),
          ratelimit: tokenInput.rateLimit || 10,
          name: tokenInput.name.trim(),
          encryptedToken: JSON.stringify(encryptedToken),
          proxyId: proxy.id,
          notes: tokenInput.notes || '',
          expiresAt: tokenInput.expiresAt || 0,
        }, ownerToken);

        try {
          await prisma.user.create({
            data: {
              userid: userId,
              accessTokenHash: hashToken(accessToken),
              proxyKey: proxy.proxyKey,
              role: tokenInput.role,
            },
          });
        } catch (error) {
          console.error('Failed to save user to local table:', error);
          try {
            await deleteUser(userId, undefined, ownerToken);
          } catch (cleanupErr) {
            console.error('Compensation delete also failed:', cleanupErr);
          }
          throw new Error(`Local database save failed for token ${tokenInput.name.trim()}`);
        }

        let namespace = 'default';
        let tags: string[] = [];
        let metadataPersisted = true;

        if (tokenInput.namespace !== undefined || tokenInput.tags !== undefined) {
          const metadataInput = {
            ...(tokenInput.namespace !== undefined ? { namespace: normalizeNamespace(tokenInput.namespace) } : {}),
            ...(tokenInput.tags !== undefined ? { tags: normalizeTags(tokenInput.tags) } : {}),
          };

          try {
            await upsertTokenMetadata(proxy.id, userId, metadataInput);
          } catch (metaErr) {
            console.error('Failed to save token metadata (token was created successfully):', metaErr);
            metadataPersisted = false;
          }
          namespace = metadataInput.namespace ?? 'default';
          tags = metadataInput.tags ?? [];
        }

        results.push({
          tokenId: userId,
          accessToken,
          name: tokenInput.name.trim(),
          role: tokenInput.role,
          createdAt,
          namespace,
          tags,
          ...(!metadataPersisted && { warning: 'Token created but metadata (namespace/tags) failed to persist' }),
        });
      } catch (tokenErr) {
        const msg = tokenErr instanceof Error ? tokenErr.message : 'Unknown error';
        console.error(`Failed to create token [${idx}] "${tokenInput.name}":`, msg);
        errors.push({ index: idx, name: tokenInput.name.trim(), error: msg });
      }
    }

    if (results.length === 0) {
      throw new ExternalApiError(E5001, `All ${tokens.length} token(s) failed to create. First error: ${errors[0]?.error}`);
    }

    const responseData: BatchCreateResponse & { errors?: typeof errors } = { tokens: results };
    if (errors.length > 0) {
      responseData.errors = errors;
    }

    return ApiResponse.success(responseData, results.length === tokens.length ? 201 : 207, request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  }
}

import { NextRequest } from 'next/server';
import { getProxy, createUser, countUsers } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { LicenseService } from '@/license-system';
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
import { ExternalApiError, E1001, E1003, E4002, E4007 } from '../../lib/error-codes';

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
 * Requires authentication (owner only).
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

    // Validate tokens array
    const { tokens } = body;
    if (!tokens || !Array.isArray(tokens) || tokens.length === 0) {
      throw new ExternalApiError(E1001, 'Missing required field: tokens');
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
        throw new ExternalApiError(
          E1003,
          `Invalid field value: tokens[${i}].role must be 2(admin) or 3(member)`,
        );
      }
      if (t.namespace !== undefined && typeof t.namespace !== 'string') {
        throw new ExternalApiError(
          E1003,
          `Invalid field value: tokens[${i}].namespace must be a string`,
        );
      }
      if (t.tags !== undefined) {
        if (!Array.isArray(t.tags)) {
          throw new ExternalApiError(
            E1003,
            `Invalid field value: tokens[${i}].tags must be an array`,
          );
        }
        if (t.tags.some((tag: unknown) => typeof tag !== 'string')) {
          throw new ExternalApiError(
            E1003,
            `Invalid field value: tokens[${i}].tags must be an array of strings`,
          );
        }
      }
      if (t.namespace !== undefined || t.tags !== undefined) {
        // validateMetadataInput internally normalizes before validating
        const metadataValidationError = validateMetadataInput({
          namespace: t.namespace,
          tags: t.tags,
        });
        if (metadataValidationError) {
          throw new ExternalApiError(
            E1003,
            `Invalid field value: tokens[${i}] metadata - ${metadataValidationError}`,
          );
        }
      }
      if (t.permissions !== undefined && !Array.isArray(t.permissions)) {
        throw new ExternalApiError(
          E1003,
          `Invalid field value: tokens[${i}].permissions must be an array`,
        );
      }
    }

    // Get proxy info
    const proxy = await getProxy();

    // Check license token limit (current + batch size must not exceed max)
    const currentTokenCount = await countUsers({ excludeRole: 1 });
    const licenseService = LicenseService.getInstance();
    const limitCheck = await licenseService.checkAccessTokenLimit(currentTokenCount.count);
    if (!limitCheck.allowed) {
      throw new ExternalApiError(E4002, 'Token limit reached');
    }
    if (currentTokenCount.count + tokens.length > limitCheck.maxAccessTokens) {
      throw new ExternalApiError(
        E4002,
        `Token limit exceeded: current ${currentTokenCount.count} + requested ${tokens.length} > max ${limitCheck.maxAccessTokens}`,
      );
    }

    // The owner token from the Authorization header
    const ownerToken = user.accessToken;

    // Create tokens
    const results: CreateTokenResult[] = [];

    for (const tokenInput of tokens) {
      // Generate access token
      const accessToken = CryptoUtils.generateToken();

      // Calculate userId from access token
      const userId = await CryptoUtils.calculateUserId(accessToken);

      // Encrypt access token with owner token
      const encryptedToken = await CryptoUtils.encryptData(accessToken, ownerToken);

      // Parse permissions
      const parsedPermissions = parseExternalTokenPermissions(tokenInput.permissions);

      const createdAt = Math.floor(Date.now() / 1000);

      // Create user record via proxy API using owner token directly
      await createUser(
        {
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
        },
        ownerToken,
      );

      // Save to local prisma user table
      try {
        await prisma.user.create({
          data: {
            userid: userId,
            accessToken: accessToken,
            proxyKey: proxy.proxyKey,
            role: tokenInput.role,
          },
        });
      } catch (error) {
        console.error('Failed to save user to local table:', error);
        throw new ExternalApiError(E5001, 'Failed to persist token user locally');
      }

      let namespace = 'default';
      let tags: string[] = [];

      if (tokenInput.namespace !== undefined || tokenInput.tags !== undefined) {
        const metadataInput = {
          ...(tokenInput.namespace !== undefined
            ? { namespace: normalizeNamespace(tokenInput.namespace) }
            : {}),
          ...(tokenInput.tags !== undefined ? { tags: normalizeTags(tokenInput.tags) } : {}),
        };

        await upsertTokenMetadata(proxy.id, userId, metadataInput);

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
      });
    }

    const responseData: BatchCreateResponse = { tokens: results };

    return ApiResponse.success(responseData, 201);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

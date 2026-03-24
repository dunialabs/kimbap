import { NextRequest } from 'next/server';
import { getUsers, getProxy, deleteUser, disableUser } from '@/lib/proxy-api';
import { deleteTokenMetadata } from '@/lib/token-metadata';
import { prisma } from '@/lib/prisma';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3003, E4006 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface DeleteTokenInput {
  tokenId: string;
}

/**
 * POST /api/external/tokens/delete
 *
 * Delete an existing access token.
 * Requires authentication (owner or admin).
 */
export async function POST(request: NextRequest) {
  try {
    // Authenticate request
    const user = await authenticate(request);

    // Parse request body
    let body: DeleteTokenInput;
    try {
      body = await request.json();
    } catch {
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

    // Cannot delete owner token
    if (existingUser.role === 1) {
      throw new ExternalApiError(E4006, 'Cannot delete owner token');
    }

    // Disable the user first to prevent new connections
    try {
      await disableUser(tokenId, undefined, ownerToken);
    } catch (error) {
      console.error('Failed to disable user in proxy:', error);
      // Continue with deletion even if disable fails
    }

    // Delete the user via proxy API (from Kimbap Core)
    await deleteUser(tokenId, undefined, ownerToken);

    // Clean up local data
    try {
      await deleteTokenMetadata(proxy.id, tokenId);
    } catch (error) {
      console.error('Failed to delete token metadata:', error);
    }

    try {
      await prisma.user.deleteMany({ where: { userid: tokenId } });
    } catch (error) {
      console.error('Failed to delete user from local table:', error);
    }

    return ApiResponse.success({
      tokenId,
      message: 'Token deleted successfully',
    });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

import { NextRequest } from 'next/server';
import { getUsers, updateUser } from '@/lib/proxy-api';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E3002 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface UnconfigureInput {
  userId: string;
  serverId: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const ownerToken = user.accessToken;

    let body: UnconfigureInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required fields: userId, serverId');
    }

    if (!body?.userId || typeof body.userId !== 'string' || !body.userId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: userId');
    }

    if (!body?.serverId || typeof body.serverId !== 'string' || !body.serverId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: serverId');
    }

    const { users } = await getUsers({ userId: body.userId }, undefined, ownerToken);
    const targetUser = users.length > 0 ? users[0] : null;

    if (!targetUser) {
      throw new ExternalApiError(E3002, `User not found: ${body.userId}`);
    }

    let serverApiKeys: string[] = [];
    if (targetUser.serverApiKeys) {
      if (typeof targetUser.serverApiKeys === 'string') {
        try {
          serverApiKeys = JSON.parse(targetUser.serverApiKeys);
        } catch {
          serverApiKeys = [];
        }
      } else if (Array.isArray(targetUser.serverApiKeys)) {
        serverApiKeys = targetUser.serverApiKeys;
      }
    }

    serverApiKeys = serverApiKeys.filter((id) => id !== body.serverId);

    await updateUser(
      body.userId,
      {
        serverApiKeys,
      },
      undefined,
      ownerToken
    );

    return ApiResponse.success({ userId: body.userId, serverId: body.serverId });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

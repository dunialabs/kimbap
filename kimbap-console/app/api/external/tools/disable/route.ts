import { NextRequest } from 'next/server';
import { getServers, stopMCPServer, getUsers } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';
import { ApiResponse } from '../../lib/response';
import { authenticate } from '../../lib/auth';
import { ExternalApiError, E1001, E2004, E3002, E3009 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface DisableToolInput {
  toolId: string;
  masterPwd: string;
}

interface OwnerUser {
  encryptedToken?: string;
}

export async function POST(request: NextRequest) {
  try {
    const user = await authenticate(request);
    const ownerToken = user.accessToken;

    let body: DisableToolInput;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Missing required field: toolId');
    }

    if (!body?.toolId || typeof body.toolId !== 'string' || !body.toolId.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: toolId');
    }

    if (!body.masterPwd || typeof body.masterPwd !== 'string') {
      throw new ExternalApiError(E1001, 'Missing required field: masterPwd');
    }

    const { toolId } = body;
    const { servers } = await getServers({ serverId: toolId }, undefined, ownerToken);
    const server = servers.length > 0 ? servers[0] : null;

    if (!server) {
      throw new ExternalApiError(E3002, `Tool not found: ${toolId}`);
    }

    if (server.enabled === false) {
      throw new ExternalApiError(E3009, 'Tool already disabled');
    }

    const { users } = await getUsers({ role: 1 }, undefined, ownerToken);
    const owner = (users[0] as OwnerUser | undefined) ?? undefined;

    let decryptedOwnerToken: string;
    try {
      if (!owner?.encryptedToken) {
        throw new Error('Missing encrypted token');
      }
      decryptedOwnerToken = await CryptoUtils.decryptDataFromString(owner.encryptedToken, body.masterPwd);
    } catch {
      throw new ExternalApiError(E2004, 'Invalid master password');
    }

    await stopMCPServer(server.serverId, undefined, decryptedOwnerToken);

    return ApiResponse.success({ toolId });
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

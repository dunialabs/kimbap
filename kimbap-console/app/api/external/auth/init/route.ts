import { NextRequest } from 'next/server';
import { randomUUID } from 'crypto';
import { CryptoUtils } from '@/lib/crypto';
import { getProxy, createProxy, createUser } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';
import { ApiResponse } from '../../lib/response';
import { ExternalApiError, E1001, E3007, E5001 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

interface InitRequest {
  masterPwd: string;
}

interface InitResponse {
  accessToken: string;
  proxyId: number;
  proxyName: string;
  proxyKey: string;
  role: number;
  userid: string;
}

/**
 * POST /api/external/auth/init
 *
 * Initialize the KIMBAP MCP proxy system and create an owner token.
 * This is the first step to set up a new proxy instance.
 */
export async function POST(request: NextRequest) {
  try {
    // Parse request body
    let body: InitRequest;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate required parameters
    const { masterPwd } = body;
    if (!masterPwd) {
      throw new ExternalApiError(E1001, 'Missing required field: masterPwd');
    }

    if (typeof masterPwd !== 'string') {
      throw new ExternalApiError(E1001, 'Missing required field: masterPwd');
    }

    // Check if a proxy server already exists
    try {
      const existingProxy = await getProxy();
      if (existingProxy) {
        throw new ExternalApiError(E3007, 'Proxy has already been initialized');
      }
    } catch (error) {
      // If it's our ExternalApiError about proxy already initialized, re-throw
      if (error instanceof ExternalApiError) {
        throw error;
      }
      // Otherwise, getProxy threw an error meaning no proxy exists, which is expected
      // Continue with proxy creation
    }

    // Generate access token
    const accessToken = CryptoUtils.generateToken();

    // Encrypt access token with master password
    const encryptedData = await CryptoUtils.encryptData(accessToken, masterPwd);

    // Calculate user ID from access token
    const userId = await CryptoUtils.calculateUserId(accessToken);

    // Generate 32-character lowercase UUID for proxy_key
    const proxyKey = randomUUID().replace(/-/g, '');

    // Create proxy record
    let proxy;
    try {
      proxy = await createProxy({
        name: 'My MCP Server',
        proxyKey: proxyKey,
        startPort: 3002,
      });
    } catch (error) {
      console.error('Failed to create proxy:', error);
      throw new ExternalApiError(E5001, 'Database error');
    }

    // Create owner user record in proxy service
    try {
      await createUser({
        userId: userId,
        status: 1, // 1-running
        role: 1, // 1:owner
        permissions: {},
        serverApiKeys: [],
        ratelimit: 10000,
        name: 'Owner',
        encryptedToken: JSON.stringify(encryptedData),
        proxyId: proxy.id,
        expiresAt: 0, // 0 means never expires
      });
    } catch (error) {
      console.error('Failed to create user in proxy service:', error);
      throw new ExternalApiError(E5001, 'Database error');
    }

    // Save token and userid to local user table
    try {
      await prisma.user.create({
        data: {
          userid: userId,
          accessToken: accessToken,
          proxyKey: proxyKey,
          role: 1, // 1-owner
        },
      });
    } catch (error) {
      console.error('Failed to save user to local table:', error);
      throw new ExternalApiError(E5001, 'Failed to persist user locally');
    }

    // Return success response
    const responseData: InitResponse = {
      accessToken: accessToken,
      proxyId: proxy.id,
      proxyName: proxy.name,
      proxyKey: proxy.proxyKey || '',
      role: 1,
      userid: userId,
    };

    return ApiResponse.success(responseData, 201);
  } catch (error) {
    return ApiResponse.handleError(error);
  }
}

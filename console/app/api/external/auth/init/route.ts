import { NextRequest } from 'next/server';
import { randomUUID } from 'crypto';
import { CryptoUtils } from '@/lib/crypto';
import { getProxy, createProxy, createUser, deleteProxy } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';
import { hashToken } from '@/lib/auth';
import { acquireRuntimeLock, checkRateLimitDb, getClientIdentity, releaseRuntimeLock } from '@/lib/request-guard';
import { ApiResponse } from '../../lib/response';
import { ExternalApiError, E1001, E2008, E3007, E5001, E5005 } from '../../lib/error-codes';

export const dynamic = 'force-dynamic';

const INIT_RATE_LIMIT = 5;
const INIT_LOCK_KEY = 'external:init';
const INIT_LOCK_TTL_MS = 30_000;

interface InitRequest {
  masterPwd: string;
}

interface InitResponse {
  accessToken: string;
  proxyId: number;
  proxyName: string;
  role: number;
  userid: string;
}

/**
 * POST /api/external/auth/init
 *
 * Initialize the Kimbap proxy system and create an owner token.
 * This is the first step to set up a new proxy instance.
 */
export async function POST(request: NextRequest) {
  let lockAcquired = false;
  try {
    const clientIp = getClientIdentity(request);
    const rateLimitKey = `external:init:${clientIp}`;

    if (!await checkRateLimitDb(rateLimitKey, INIT_RATE_LIMIT)) {
      throw new ExternalApiError(E2008, 'Rate limit exceeded. Try again later.');
    }

    lockAcquired = await acquireRuntimeLock(INIT_LOCK_KEY, INIT_LOCK_TTL_MS);
    if (!lockAcquired) {
      throw new ExternalApiError(E5005, 'Initialization already in progress. Please retry shortly.');
    }

    // Parse request body
    let body: InitRequest;
    try {
      body = await request.json();
    } catch {
      throw new ExternalApiError(E1001, 'Invalid request body');
    }

    // Validate required parameters
    const { masterPwd } = body;
    if (typeof masterPwd !== 'string' || !masterPwd.trim()) {
      throw new ExternalApiError(E1001, 'Missing required field: masterPwd');
    }

    try {
      const existingProxy = await getProxy();
      if (existingProxy) {
        throw new ExternalApiError(E3007, 'Proxy has already been initialized');
      }
    } catch (error) {
      if (error instanceof ExternalApiError) throw error;
      const msg = error instanceof Error ? error.message : String(error);
      const isNotFound = msg.includes('not found') || msg.includes('No proxy') || msg.includes('Record not found');
      if (!isNotFound) throw error;
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
    const proxy = await createProxy({
      name: 'My MCP Server',
      proxyKey: proxyKey,
      startPort: 3002,
    }).catch((error) => {
      console.error('Failed to create proxy:', error);
      throw new ExternalApiError(E5001, 'Database error');
    });

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
      let rolledBack = false;
      try {
        await deleteProxy(proxy.id, '', accessToken);
        rolledBack = true;
      } catch (cleanupErr) {
        console.error('Compensation proxy delete failed:', cleanupErr);
      }
      throw new ExternalApiError(E5001, rolledBack
        ? 'Failed to create owner user. Proxy has been rolled back.'
        : 'Failed to create owner user. Manual cleanup may be required.');
    }

    // Save token and userid to local user table
    try {
      await prisma.user.create({
        data: {
          userid: userId,
          accessTokenHash: hashToken(accessToken),
          proxyKey: 'single',
          role: 1, // 1-owner
        },
      });
    } catch (error) {
      console.error('Failed to save user to local table:', error);
      // Compensation: deleteProxy cascades to remove associated users in Core
      let rolledBack = false;
      try {
        await deleteProxy(proxy.id, userId, accessToken);
        rolledBack = true;
      } catch (cleanupErr) {
        console.error('Compensation proxy delete failed:', cleanupErr);
      }
      throw new ExternalApiError(E5001, rolledBack
        ? 'Failed to save owner record to local database. Core state has been rolled back.'
        : 'Failed to save owner record to local database. Core cleanup failed — manual intervention may be required.');
    }

    // Return success response
    const responseData: InitResponse = {
      accessToken: accessToken,
      proxyId: proxy.id,
      proxyName: proxy.name,
      role: 1,
      userid: userId,
    };

    return ApiResponse.success(responseData, 201, request);
  } catch (error) {
    return ApiResponse.handleError(error, request);
  } finally {
    if (lockAcquired) {
      try {
        await releaseRuntimeLock(INIT_LOCK_KEY);
      } catch (lockReleaseError) {
        console.warn('Failed to release init lock:', lockReleaseError);
      }
    }
  }
}

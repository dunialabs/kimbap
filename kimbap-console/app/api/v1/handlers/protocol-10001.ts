import { CryptoUtils } from '@/lib/crypto';
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { randomUUID } from 'crypto';
import { getProxy, createProxy, createUser } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';

interface Request10001 {
  common: {
    cmdId: number;
  };
  params: {
    masterPwd: string;
  };
}

interface Response10001Data {
  accessToken: string;
  proxyId: number;
  proxyName: string;
  proxyKey: string;
  role: number; // 1-owner
  userid: string;
}

export async function handleProtocol10001(body: Request10001): Promise<Response10001Data> {
  const masterPwd = body.params?.masterPwd;

  // Validate required parameters
  if (!masterPwd) {
    throw new ApiError(ErrorCode.MISSING_MASTER_PASSWORD);
  }

  // Generate master token
  const accessToken = CryptoUtils.generateToken();
  
  // Encrypt master token to create user_id
  const encryptedData = await CryptoUtils.encryptData(accessToken, masterPwd);
  // Use the encrypted data as user_id
  const userId = await CryptoUtils.calculateUserId(accessToken);

  // Check if a proxy server already exists
  try {
    const existingProxy = await getProxy();
    if (existingProxy) {
      throw new ApiError(ErrorCode.PROXY_ALREADY_INITIALIZED);
    }
  } catch (error) {
    // If getProxy throws an error, it means no proxy exists, which is what we want
    // Continue with proxy creation
  }

  // Generate 32-character lowercase UUID for proxy_key
  const proxyKey = randomUUID().replace(/-/g, '');
  
  // Create proxy record
  const proxy = await createProxy({
    name: 'My MCP Server', // serverName hardcoded as requested
    proxyKey: proxyKey, // 32-character lowercase UUID
    startPort: 3002 // Default start port
  });

  // Create owner user record
  console.log('userid:', userId)
  const user = await createUser({
    userId: userId,
    status: 1, // 1-running (hardcoded as requested)
    role: 1, // 1:owner (hardcoded as requested)
    permissions: {}, // Owner has all permissions
    serverApiKeys: [], // Empty initially
    ratelimit: 10000, // High rate limit for owner
    name: 'Owner',
    encryptedToken: JSON.stringify(encryptedData), // Save encryptedData as JSON string
    proxyId: proxy.id, // Save the proxy_id
    expiresAt: 0, // 0 means never expires
  });

  // Save token and userid to local user table
  try {
    await prisma.user.create({
      data: {
        userid: userId,
        accessToken: accessToken, // Save plain text token
        proxyKey: proxyKey, // Save the proxy key
        role: 1 // 1-owner
      }
    });
    console.log('Saved user to local table:', { userid: userId, proxyKey: proxyKey, role: 1 });
  } catch (error) {
    console.error('Failed to save user to local table:', error);
    // Continue execution even if local save fails
  }
    
  const result = { proxy, user };

  // Return data for response
  return {
    accessToken: accessToken,
    proxyId: result.proxy.id,
    proxyName: result.proxy.name,
    proxyKey: result.proxy.proxyKey || '', // Return the generated proxyKey
    role: 1, // 1-owner
    userid: userId
  };
}

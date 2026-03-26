import { ApiError, ErrorCode } from '@/lib/error-codes';
import { CryptoUtils } from '@/lib/crypto';
import { getUserByAccessToken, getProxy, getOwner, connectAllServers } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';
import { hashToken } from '@/lib/auth';

interface Request10015 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    accessToken?: string;
    masterPwd?: string;
  };
}

interface TokenInfo {
  userid: string;
  role: number; // 1-owner, 2-admin
  createAt: number;
}

interface Response10015Data {
  tokenInfo: TokenInfo;
  accessToken?: string;
}

/**
 * Protocol 10015 - Login with Access Token or Master Password
 * Two login methods:
 * 1. accessToken: Calculate userId and query user info (all users)
 * 2. masterPwd: Find owner and decrypt their token (owner only)
 */
export async function handleProtocol10015(body: Request10015): Promise<Response10015Data> {
  const { accessToken, masterPwd } = body.params || {};
  
  // Validate that at least one authentication method is provided
  if (!accessToken && !masterPwd) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { 
      field: 'accessToken or masterPwd',
      details: 'Either accessToken or masterPwd must be provided' 
    });
  }
  
  // Cannot provide both methods at the same time
  if (accessToken && masterPwd) {
    throw new ApiError(ErrorCode.INVALID_REQUEST, 400, { 
      details: 'Provide either accessToken or masterPwd, not both' 
    });
  }
  
  try {
    let user: any = null;
    let masterPwdAccessToken: string | undefined;
    
    if (accessToken) {
      // Method 1: Login with access token
      // Calculate userId from the plain text token
      const userId = await CryptoUtils.calculateUserId(accessToken);
      
      if (!userId) {
        throw new ApiError(ErrorCode.INVALID_TOKEN, 401);
      }
      
      // Query user information from proxy API using the service function
      try {
        user = await getUserByAccessToken(userId, accessToken);
      } catch (error) {
        console.error('Failed to get user with access token:', error);
        if (error instanceof ApiError) {
          throw error;
        }
        throw error;
      }
      
      console.log(`User ${userId} logging in with access token`);
      
    } else if (masterPwd) {
      // Method 2: Login with master password (owner only)
      // Use the new GET_OWNER endpoint that doesn't require authentication
      const ownerResponse = await getOwner();
      const owner = ownerResponse.owner;
      
      if (!owner) {
        throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, { 
          details: 'Owner account not found' 
        });
      }
      
      // Check if owner has encrypted token
      if (!owner.encryptedToken) {
        throw new ApiError(ErrorCode.INVALID_REQUEST, 400, { 
          details: 'Owner token not configured' 
        });
      }
      
      // Try to decrypt the owner's token with the provided master password
      let decryptedToken: string | null = null;
      try {
        decryptedToken = await CryptoUtils.decryptDataFromString(
          owner.encryptedToken,
          masterPwd
        );
      } catch (decryptError) {
        if (decryptError instanceof ApiError) throw decryptError;
        console.error('Failed to decrypt owner token:', decryptError);
      }

      if (!decryptedToken) {
        throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD, 401);
      }

      masterPwdAccessToken = decryptedToken;
      user = owner;
      console.log(`Owner ${owner.userId} logging in with master password`);
    }
    
    // Ensure user is defined before proceeding
    if (!user) {
      throw new ApiError(ErrorCode.USER_NOT_FOUND, 404);
    }
    
    // Common validation for both methods
    // Check user role - only owner (1) and admin (2) can login
    if (user.role !== 1 && user.role !== 2) {
      throw new ApiError(ErrorCode.PERMISSION_DENIED, 403, { 
        details: 'Only owner and admin users can login with tokens' 
      });
    }
    
    // Check if user is active
    if (user.status !== 1) { // 1 = running/active
      throw new ApiError(ErrorCode.USER_DISABLED, 403);
    }
    
    // Check if token has expired
    const currentTime = Math.floor(Date.now() / 1000);
    if (user.expiresAt > 0 && user.expiresAt < currentTime) {
      throw new ApiError(ErrorCode.TOKEN_EXPIRED, 401);
    }
    
    try {
      const proxyInfo = await getProxy();
      const tokenToHash = accessToken ?? masterPwdAccessToken!;
      await prisma.user.upsert({
        where: { userid: user.userId },
        update: { accessTokenHash: hashToken(tokenToHash), proxyKey: proxyInfo.proxyKey, role: user.role },
        create: { userid: user.userId, accessTokenHash: hashToken(tokenToHash), proxyKey: proxyInfo.proxyKey, role: user.role },
      });
    } catch (saveError) {
      console.error('Failed to save user to local database:', saveError);
    }

    const tokenInfo: TokenInfo = {
      userid: user.userId,
      role: user.role,
      createAt: user.createdAt
    };

    console.log(`User ${user.userId} logged in successfully with role ${user.role}`);
    
    // Reconnect all servers if user is owner (role 1)
    if (user.role === 1) {
      try {
        console.log(`[Protocol 10015] Owner ${user.userId} logged in, reconnecting all servers...`);
        
        const ownerToken: string | null = accessToken ?? masterPwdAccessToken ?? null;
        
        if (ownerToken) {
          const reconnectResult = await connectAllServers(undefined, ownerToken);
          console.log(`[Protocol 10015] Server reconnection completed - Success: ${reconnectResult.successServers?.length || 0}, Failed: ${reconnectResult.failedServers?.length || 0}`);
          
          // Log any failed servers for debugging
          if (reconnectResult.failedServers && reconnectResult.failedServers.length > 0) {
            console.warn(`[Protocol 10015] Some servers failed to reconnect:`, reconnectResult.failedServers);
          }
        } else {
          console.warn(`[Protocol 10015] Could not get owner token for server reconnection`);
        }
      } catch (reconnectError) {
        // Log the error but don't fail the login
        console.error(`[Protocol 10015] Failed to reconnect servers after owner login:`, reconnectError);
      }
    }
    
    return {
      tokenInfo,
      accessToken: masterPwd ? masterPwdAccessToken : undefined
    };
    
  } catch (error) {
    console.error('Protocol 10015 error:', error);
    
    if (error instanceof ApiError) {
      throw error;
    }
    
    // Generic error for security reasons
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500);
  }
}

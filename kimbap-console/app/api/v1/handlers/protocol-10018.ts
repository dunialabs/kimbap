import { ApiError, ErrorCode } from '@/lib/error-codes';
import { connectAllServers, getUsers, makeProxyRequestWithUserId } from '@/lib/proxy-api';
import { CryptoUtils } from '@/lib/crypto';

interface Request10018 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    masterPwd: string;
  };
}

interface Response10018Data {
  // Empty data object as per proto definition
}

export async function handleProtocol10018(body: Request10018): Promise<Response10018Data> {
  const { masterPwd } = body.params || {};
  
  console.log('[Protocol 10018] Request received to connect all MCP servers');
  
  // Validate required masterPwd
  if (!masterPwd) {
    throw new ApiError(ErrorCode.MISSING_MASTER_PASSWORD);
  }
  
  try {
    console.log('[Protocol 10018] Validating master password...');
    
    // Get the owner user from database
    const usersResponse = await getUsers({ role: 1 }, body.common.userid);
    const owner = usersResponse.users.find(user => user.role === 1);
    
    if (!owner) {
      console.log('[Protocol 10018] No owner found in database');
      throw new ApiError(ErrorCode.USER_NOT_FOUND);
    }
    
    if (!owner.encryptedToken) {
      console.log('[Protocol 10018] Owner has no encrypted token');
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
    }
    
    // Validate master password by attempting to decrypt the owner token
    let ownerToken: string;
    try {
      ownerToken = await CryptoUtils.decryptDataFromString(
        owner.encryptedToken,
        masterPwd
      );
      
      // Validate decrypted token is not empty
      if (!ownerToken) {
        throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD);
      }
    } catch (error) {
      // If decryption fails, it's likely due to invalid master password
      console.error('[Protocol 10018] Failed to decrypt owner token:', error);
      throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD);
    }
    
    console.log('[Protocol 10018] Master password verified successfully');
    
    // Use the decrypted owner token for the API call
    const token = ownerToken;
    
    console.log('[Protocol 10018] Calling proxy API to connect all servers...');
    
    // Call proxy API 2005 to connect all servers with token
    const result = await connectAllServers(body.common.userid, token);
    
    console.log('[Protocol 10018] Connect all servers result:', {
      successCount: result.successServers?.length || 0,
      failedCount: result.failedServers?.length || 0,
      successServers: result.successServers?.map((s: any) => s.serverId || s) || [],
      failedServers: result.failedServers?.map((s: any) => s.serverId || s) || []
    });
    
    // Log any failed servers for debugging
    if (result.failedServers && result.failedServers.length > 0) {
      console.warn('[Protocol 10018] Some servers failed to connect:', result.failedServers);
    }
    
    // Return empty data object as per proto definition
    return {};
    
  } catch (error) {
    console.error('[Protocol 10018] Error connecting servers:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
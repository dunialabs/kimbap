import { ApiError, ErrorCode } from '@/lib/error-codes';
import { CryptoUtils } from '@/lib/crypto';
import { getOwner, updateUser } from '@/lib/proxy-api';

interface Request10024 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    accessToken: string;      // 原始token (与proto定义一致)
    masterPwd: string;        // 新的master password
  };
}

interface Response10024Data {
  success: boolean;
  message: string;
}

/**
 * Protocol 10024 - Reset Master Password
 * 重置master password功能：
 * 1. 验证上传的token计算出的userid与owner的userid一致
 * 2. 用新的master password加密token
 * 3. 调用proxy-api的1012接口更新owner的encryptedToken
 */
export async function handleProtocol10024(body: Request10024): Promise<Response10024Data> {
  const { accessToken, masterPwd } = body.params || {};
  
  // 验证必需字段
  if (!accessToken) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { 
      field: 'accessToken',
      details: 'Access token is required' 
    });
  }
  
  if (!masterPwd) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { 
      field: 'masterPwd',
      details: 'Master password is required' 
    });
  }
  
  try {
    // 1. 计算token的userid
    const tokenUserId = await CryptoUtils.calculateUserId(accessToken);
    
    if (!tokenUserId) {
      throw new ApiError(ErrorCode.INVALID_TOKEN, 401, {
        details: 'Unable to calculate userid from access token'
      });
    }
    
    console.log(`[Protocol-10024] Calculated userid from access token: ${tokenUserId}`);
    
    // 2. 获取owner信息
    let owner: any;
    try {
      const ownerResponse = await getOwner();
      owner = ownerResponse.owner;
    } catch (error) {
      console.error('[Protocol-10024] Failed to get owner:', error);
      if (error instanceof ApiError && error.statusCode === 404) {
        throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, { 
          details: 'Owner account not found in system' 
        });
      }
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to retrieve owner information'
      });
    }
    
    console.log(`[Protocol-10024] Owner userid: ${owner.userId}`);
    
    // 3. 验证token userid与owner userid一致
    if (tokenUserId !== owner.userId) {
      console.warn(`[Protocol-10024] Token userid mismatch - Token: ${tokenUserId}, Owner: ${owner.userId}`);
      throw new ApiError(ErrorCode.INVALID_TOKEN, 401, {
        details: 'Token does not belong to the owner account'
      });
    }
    
    console.log(`[Protocol-10024] Token userid verified successfully`);
    
    // 4. 用新的master password加密token
    let encryptedTokenData: string;
    try {
      const encryptedToken = await CryptoUtils.encryptData(accessToken, masterPwd);
      encryptedTokenData = JSON.stringify(encryptedToken);
      console.log(`[Protocol-10024] Access token encrypted successfully with new master password`);
    } catch (encryptError) {
      console.error('[Protocol-10024] Failed to encrypt access token:', encryptError);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to encrypt access token with master password'
      });
    }
    
    // 5. 调用proxy-api的1012接口更新owner的encryptedToken
    try {
      await updateUser(owner.userId, {
        encryptedToken: encryptedTokenData
      });
      
      console.log(`[Protocol-10024] Owner encryptedToken updated successfully`);
      
      return {
        success: true,
        message: 'Master password reset successfully'
      };
      
    } catch (updateError) {
      console.error('[Protocol-10024] Failed to update owner encryptedToken:', updateError);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to update owner encrypted token'
      });
    }
    
  } catch (error) {
    console.error('[Protocol-10024] Error:', error);
    
    if (error instanceof ApiError) {
      throw error;
    }
    
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
      details: 'Failed to reset master password'
    });
  }
}
import { ApiError, ErrorCode } from '@/lib/error-codes';
import { CryptoUtils } from '@/lib/crypto';
import { getUsers, backupDatabase, makeProxyRequestWithUserId } from '@/lib/proxy-api';

interface Request10016 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    masterPwd: string;
  };
}

interface Response10016Data {
  encryptedData: string;
}

/**
 * Protocol 10016 - Backup Server Data
 * Backs up data from proxy, server, user, and ip_whitelist tables
 * Encrypts the backup with master password
 */
export async function handleProtocol10016(body: Request10016): Promise<Response10016Data> {
  const { masterPwd } = body.params || {};
  
  
  // Validate master password is provided
  if (!masterPwd) {
    throw new ApiError(ErrorCode.MISSING_MASTER_PASSWORD, 400);
  }
  
  try {
    // Verify master password by checking owner's encrypted token
    const usersResponse = await getUsers({ role: 1 }, body.common.userid);
    const owner = usersResponse.users.find(user => user.role === 1);
    
    if (!owner) {
      throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, { 
        details: 'Owner account not found' 
      });
    }
    
    // Verify master password by attempting to decrypt owner's token
    if (owner.encryptedToken) {
      try {
        await CryptoUtils.decryptDataFromString(
          owner.encryptedToken,
          masterPwd
        );
      } catch (decryptError: any) {
        console.error('Invalid master password for backup:', decryptError);
        // Pass the actual error message to the client
        throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD, 401, {
          details: decryptError.message || 'Failed to verify master password'
        });
      }
    }
    
    console.log('Starting data backup process...');
    
    // Use proxy API to backup database
    const backupData = await backupDatabase(body.common.userid);
    
    console.log(`Backed up ${backupData.proxy.length} proxy records`);
    console.log(`Backed up ${backupData.server.length} server records`);
    console.log(`Backed up ${backupData.user.length} user records`);
    console.log(`Backed up ${backupData.ipWhitelist.length} IP whitelist records`);
    
    // Convert to the expected format for encryption
    const formattedBackupData = {
      tables: {
        proxy: backupData.proxy,
        server: backupData.server,
        user: backupData.user,
        ipWhitelist: backupData.ipWhitelist
      }
    };
    
    // Convert to JSON string
    const jsonData = JSON.stringify(formattedBackupData, null, 2);
    console.log(`Total backup size: ${jsonData.length} characters`);
  
    // Encrypt the JSON data with master password
    const encryptedBackup = await CryptoUtils.encryptData(jsonData, masterPwd);
    
    // Convert encrypted data object to string for transmission
    const encryptedDataString = JSON.stringify(encryptedBackup);
    
    console.log('Data backup completed successfully');
    
    const result = {
      encryptedData: encryptedDataString
    };


    return result;
    
  } catch (error) {
    console.error('Protocol 10016 backup error:', error);
    
    
    if (error instanceof ApiError) {
      throw error;
    }
    
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
      details: 'Failed to backup server data'
    });
  }
}
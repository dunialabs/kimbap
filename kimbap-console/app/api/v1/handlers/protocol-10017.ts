import { ApiError, ErrorCode } from '@/lib/error-codes';
import { CryptoUtils } from '@/lib/crypto';
import { exec } from 'child_process';
import { promisify } from 'util';
import { getUsers, restoreDatabase, makeProxyRequestWithUserId } from '@/lib/proxy-api';

const execAsync = promisify(exec);

interface Request10017 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    masterPwd: string;
    encryptedData: string;
  };
}

interface Response10017Data {
  // Empty response as per protocol definition
}

/**
 * Protocol 10017 - Restore Server Data from Backup
 * Decrypts and restores data to proxy, server, user, and ipWhitelist tables
 * Restarts proxy server after restoration
 */
export async function handleProtocol10017(body: Request10017): Promise<Response10017Data> {
  const { masterPwd, encryptedData } = body.params || {};
  
  
  // Validate required fields
  if (!masterPwd) {
    throw new ApiError(ErrorCode.MISSING_MASTER_PASSWORD, 400);
  }
  
  if (!encryptedData) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, {
      field: 'encryptedData'
    });
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
        console.error('Invalid master password for restore:', decryptError);
        throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD, 401, {
          details: decryptError.message || 'Failed to verify master password'
        });
      }
    }
    
    console.log('Starting data restore process...');
    
    // Decrypt the backup data
    let decryptedJson: string;
    try {
      decryptedJson = await CryptoUtils.decryptDataFromString(encryptedData, masterPwd);
    } catch (decryptError: any) {
      console.error('Failed to decrypt backup data:', decryptError);
      throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
        details: 'Failed to decrypt backup data: ' + (decryptError.message || 'Invalid encrypted data or password')
      });
    }
    
    // Parse the JSON data
    let backupData: any;
    try {
      backupData = JSON.parse(decryptedJson);
    } catch (parseError: any) {
      console.error('Failed to parse backup data:', parseError);
      throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
        details: 'Invalid backup data format'
      });
    }
    
    // Validate backup data structure
    if (!backupData.tables || typeof backupData.tables !== 'object') {
      throw new ApiError(ErrorCode.INVALID_REQUEST, 400, {
        details: 'Invalid backup data structure: missing tables'
      });
    }
    
    // Use proxy API to restore database
    console.log('Starting database restore...');
    
    let restoreResult;
    try {
      restoreResult = await restoreDatabase({
        proxy: backupData.tables.proxy || [],
        user: backupData.tables.user || [],
        server: backupData.tables.server || [],
        ipWhitelist: backupData.tables.ipWhitelist || []
      }, body.common.userid);
    } catch (restoreError: any) {
      console.error('Database restore failed:', restoreError);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Database restore failed: ' + (restoreError.message || 'Unknown error from proxy server')
      });
    }
    
    console.log('Database restore completed:', restoreResult.restoredCounts);
    
    console.log('Database restore completed successfully');
    
    // Restart proxy server
    console.log('Restarting proxy server...');
    try {
      // First, try to stop the existing proxy server gracefully
      try {
        await execAsync('pkill -TERM -f "node.*proxy-server" || true');
        // Wait a moment for graceful shutdown
        await new Promise(resolve => setTimeout(resolve, 2000));
      } catch (stopError) {
        // It's okay if the server wasn't running
        console.log('No existing proxy server to stop');
      }
      
      // Start the proxy server in background
      const backendPort = process.env.BACKEND_PORT || '3002';
      const startCommand = `cd ${process.cwd()} && PORT=${backendPort} node proxy-server/index.js > /dev/null 2>&1 &`;
      
      await execAsync(startCommand);
      console.log('Proxy server restart initiated');
      
      // Wait a moment for the server to start
      await new Promise(resolve => setTimeout(resolve, 3000));
      
      // Verify the server is running by checking if port is listening
      try {
        await execAsync(`lsof -i :${backendPort} | grep LISTEN`);
        console.log(`Proxy server successfully restarted on port ${backendPort}`);
      } catch (verifyError) {
        console.warn('Could not verify proxy server status, it may take a moment to fully start');
      }
      
    } catch (restartError: any) {
      console.error('Failed to restart proxy server:', restartError);
      // Don't throw here, data restore was successful even if restart failed
      console.warn('Data restored successfully but proxy server restart failed. Please restart manually.');
    }
    
    console.log('Data restore and server restart completed');
    
    const result = {};


    return result;
    
  } catch (error) {
    console.error('Protocol 10017 restore error:', error);
    
    
    if (error instanceof ApiError) {
      throw error;
    }
    
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
      details: 'Failed to restore server data'
    });
  }
}
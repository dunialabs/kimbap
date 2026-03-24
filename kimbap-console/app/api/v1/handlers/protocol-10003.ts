import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy, updateProxy, deleteProxy, makeProxyRequestWithUserId, getCloudflaredConfigs, getUsers } from '@/lib/proxy-api';
import { prisma } from '@/lib/prisma';
import { KimbapCloudApiService } from '@/lib/KimbapCloudApiService';
import { CryptoUtils } from '@/lib/crypto';

interface Request10003 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    handleType: number; // 1-edit base info, 8-reset server
    proxyId: number;
    proxyName?: string;
    masterPwd?: string; // Master password for reset server validation
    proxyKey?: string; // For reset server validation (backward compatibility)
  };
}

interface Response10003Data {
  success: boolean;
  message?: string;
}

// Handle type constants for better readability
const HandleType = {
  EDIT_BASE_INFO: 1,
  RESET_SERVER: 8
} as const;

export async function handleProtocol10003(body: Request10003): Promise<Response10003Data> {
  const { handleType, proxyId, proxyName, masterPwd, proxyKey } = body.params || {};


  // Validate required parameters
  if (!handleType) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'handleType' });
  }

  if (!proxyId) {
    throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'proxyId' });
  }

  try {
    // Get the proxy record
    const proxy = await getProxy();

    if (!proxy || proxy.id !== proxyId) {
      throw new ApiError(ErrorCode.RECORD_NOT_FOUND, 404);
    }

    // Handle different operations based on handleType
    switch (handleType) {
      case HandleType.EDIT_BASE_INFO:
        // Update proxy name in database if proxyName is provided
        if (proxyName && proxyName.trim()) {
          await updateProxy(proxy.id, { name: proxyName.trim() }, body.common.userid);
        }
        // Return success regardless of whether update was made
        return {
          success: true
        };

      case HandleType.RESET_SERVER:
        try {
          // 1. Validate master password (required for reset server)
          if (!masterPwd) {
            throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { 
              field: 'masterPwd',
              details: 'Master password is required for reset server operation' 
            });
          }
          
          // Verify master password by attempting to decrypt owner's token
          try {
            const ownerResult = await getUsers({
              proxyId: proxyId,
              role: 1 // Owner role
            }, body.common.userid);
            const owner = ownerResult.users[0];
            
            if (!owner || !owner.encryptedToken) {
              throw new ApiError(ErrorCode.USER_NOT_FOUND, 404, { 
                details: 'Owner user not found or no encrypted token' 
              });
            }
            
            // Attempt to decrypt owner token to verify master password
            const ownerToken = await CryptoUtils.decryptDataFromString(
              owner.encryptedToken,
              masterPwd
            );
            
            if (!ownerToken) {
              throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD, 401, {
                details: 'Invalid master password'
              });
            }
            
            console.log('[Protocol-10003] Master password validated successfully for reset server');
          } catch (error) {
            if (error instanceof ApiError) {
              throw error;
            }
            console.error('[Protocol-10003] Failed to verify master password:', error);
            throw new ApiError(ErrorCode.INVALID_MASTER_PASSWORD, 401, {
              details: 'Failed to verify master password'
            });
          }
          
          // 2. Validate proxyKey if provided (backward compatibility)
          if (proxyKey && proxyKey !== proxy.proxyKey) {
            throw new ApiError(ErrorCode.INVALID_PARAMS, 400, { 
              message: 'ProxyKey mismatch' 
            });
          }
          
          const currentProxyKey = proxy.proxyKey || '';
          console.log(`Starting server reset for proxyKey: ${currentProxyKey}`);
          
          // 3. Get tunnel information via proxy-api 8002 first
          let tunnelInfos: any[] = [];
          try {
            const tunnelResult = await getCloudflaredConfigs(body.common.userid, {
              type: 1,
              proxyKey: currentProxyKey
            });
            tunnelInfos = tunnelResult.dnsConfs || [];
            console.log(`Found ${tunnelInfos.length} tunnel(s) to delete`);
          } catch (tunnelError) {
            console.error('Failed to get tunnel info from proxy-api:', tunnelError);
            // Continue - we'll also check local records as fallback
          }
          
          // 4. Delete tunnels via KimbapCloudApi if found
          const kimbapCloudApi = new KimbapCloudApiService();
          const tunnelIds = new Set<string>();
          
          // Add tunnels from proxy-api result
          for (const tunnelInfo of tunnelInfos) {
            if (tunnelInfo.tunnelId) {
              tunnelIds.add(tunnelInfo.tunnelId);
            }
          }
          
          // Also check local records as fallback
          try {
            const localTunnelRecords = await prisma.dnsConf.findMany({
              where: { 
                proxyKey: currentProxyKey,
                type: 1
              }
            });
            
            for (const record of localTunnelRecords) {
              if (record.tunnelId) {
                tunnelIds.add(record.tunnelId);
              }
            }
          } catch (localError) {
            console.error('Failed to query local tunnel records:', localError);
          }
          
          // Delete each unique tunnel
          for (const tunnelId of Array.from(tunnelIds)) {
            try {
              await kimbapCloudApi.deleteTunnel(tunnelId);
              console.log(`Deleted tunnel: ${tunnelId}`);
            } catch (tunnelError) {
              console.error(`Failed to delete tunnel ${tunnelId}:`, tunnelError);
              // Continue even if tunnel deletion fails
            }
          }
          
          // 5. Clear local dnsConf data first
          try {
            const deletedDnsConfs = await prisma.dnsConf.deleteMany({
              where: { proxyKey: currentProxyKey }
            });
            console.log(`Deleted ${deletedDnsConfs.count} DNS config records`);
          } catch (dnsError) {
            console.error('Failed to delete DNS config records:', dnsError);
          }
          
          // 6. Delete other local table data
          try {
            // Delete logs
            const deletedLogs = await prisma.log.deleteMany({
              where: { proxyKey: currentProxyKey }
            });
            console.log(`Deleted ${deletedLogs.count} log records`);
            
            // Delete licenses
            const deletedLicenses = await prisma.license.deleteMany({
              where: { proxyKey: currentProxyKey }
            });
            console.log(`Deleted ${deletedLicenses.count} license records`);
            
          } catch (cleanupError) {
            console.error('Error cleaning up other local data:', cleanupError);
            // Continue even if cleanup fails
          }
          
          // 7. Finally delete proxy via proxy-api
          try {
            await deleteProxy(proxyId, body.common.userid);
            console.log('Proxy deleted successfully');
          } catch (proxyError) {
            console.error('Failed to delete proxy:', proxyError);
            // Continue - local data is already cleaned
          }

          return {
            success: true,
            message: 'Server reset successfully - all associated data deleted'
          };
          
        } catch (error) {
          console.error('Failed to reset server:', error);
          
          if (error instanceof ApiError) {
            throw error;
          }
          throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
            details: error instanceof Error ? error.message : 'Unknown error'
          });
        }

      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'handleType' });
    }
  } catch (error) {
    // Log error if it's not already logged (only for reset server operations)
    if (handleType === HandleType.RESET_SERVER && error instanceof ApiError) {
      // Error was already logged in the inner catch block, just re-throw
      throw error;
    }
    
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500);
  }
}
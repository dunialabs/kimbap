import { ApiError, ErrorCode } from '@/lib/error-codes';
import { addIpWhitelist, specialIpWhitelistOperation, makeProxyRequestWithUserId } from '@/lib/proxy-api';

interface Request10013 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    handleType: number; // 1-add ip, 2-allow all (add 0.0.0.0/0), 3-remove 0.0.0.0/0, 4-delete by id
    ipList?: string[];
    idList?: number[];
  };
}

interface Response10013Data {
  // Empty response data according to protocol
}

/**
 * Protocol 10013 - Maintain IP Whitelist
 * handleType=1: Add IPs from ipList
 * handleType=2: Add 0.0.0.0/0 (allow all)
 * handleType=3: Remove 0.0.0.0/0
 * handleType=4: Delete by ID list
 */
export async function handleProtocol10013(body: Request10013): Promise<Response10013Data> {
  const { handleType, ipList, idList } = body.params || {};
  
  console.log('Protocol 10013 received params:', { handleType, ipList, idList });
  
  // Validate handleType
  if (!handleType || handleType < 1 || handleType > 4) {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'handleType' });
  }
  
  const now = Math.floor(Date.now() / 1000); // Current timestamp in seconds
  
  try {
    switch (handleType) {
      case 1: {
        // Add IPs to whitelist
        if (!ipList || ipList.length === 0) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'ipList' });
        }
        
        // Use proxy API to add IPs to whitelist
        const result = await addIpWhitelist(ipList, body.common.userid);
        
        console.log(`Added ${result.addedCount} IPs to whitelist, skipped ${result.skippedCount} duplicates`);
        break;
      }
      
      case 2: {
        // Add 0.0.0.0/0 (allow all IPs)
        await specialIpWhitelistOperation('allow-all', body.common.userid);
        console.log('Applied allow-all IP whitelist operation');
        break;
      }
      
      case 3: {
        // Remove 0.0.0.0/0 (disable allow all)
        await specialIpWhitelistOperation('deny-all', body.common.userid);
        console.log('Applied deny-all IP whitelist operation');
        break;
      }
      
      case 4: {
        // Delete by ID list
        if (!idList || idList.length === 0) {
          throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'idList' });
        }
        
        // Note: The current proxy API deleteIpWhitelist method expects IP addresses, not IDs
        // This is a limitation that should be addressed in the proxy API design
        // Since the current proxy API doesn't return IDs, we cannot properly map IDs to IPs
        // This indicates a design gap in the proxy API that needs to be addressed
        throw new ApiError(ErrorCode.PROTOCOL_NOT_IMPLEMENTED, 501, {
          details: 'Delete by ID list requires the proxy API to support ID-based operations. The current proxy API only supports IP-based deletion.'
        });
      }
      
      default:
        throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { 
          field: 'handleType',
          details: `Invalid handleType: ${handleType}` 
        });
    }
    
    return {};
    
  } catch (error) {
    console.error('Protocol 10013 error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
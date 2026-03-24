import { prisma } from '@/lib/prisma';
import { ApiError, ErrorCode } from '@/lib/error-codes';

import { KimbapCloudApiService } from '@/lib/KimbapCloudApiService';
import { updateCloudflaredConfig, getProxy, getCloudflaredConfigs, restartCloudflared, stopCloudflared } from '@/lib/proxy-api';


const kimbapCloudApi = new KimbapCloudApiService();

interface Request10011 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    handleType: number; // 1-start remote access, 2-stop remote access, 3-add manual dns, 4-delete manual dns
    domain?: string; // for handleType=3
    publicIP?: string; // for handleType=3
    recordId?: number; // for handleType=4
    proxyKey?: string; // proxy key for filtering DNS records
  };
}

interface Response10011Data {
  subdomain?: string; // Return subdomain for handleType=1 according to pb definition
}

/**
 * Protocol 10011 - Operate DNS Record
 * handleType=1: Start remote access (Kimbap.io tunnel)
 * handleType=2: Stop remote access (Stop cloudflared and deactivate tunnel)
 * handleType=3: Add manual DNS record
 * handleType=4: Delete manual DNS record by recordId
 */
export async function handleProtocol10011(body: Request10011): Promise<Response10011Data> {
  const { handleType, domain, publicIP, recordId, proxyKey } = body.params || {};
  
  
  // Validate handleType
  if (!handleType || handleType < 1 || handleType > 4) {
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'handleType' });
  }
  
  // Validate proxyKey if provided
  let currentProxyKey = '';
  try {
    const proxyInfo = await getProxy();
    currentProxyKey = proxyInfo.proxyKey || '';
    
    // If proxyKey is provided, validate it matches
    if (proxyKey && proxyKey !== currentProxyKey) {
      console.error('ProxyKey mismatch - provided:', proxyKey, 'expected:', currentProxyKey);
      throw new ApiError(ErrorCode.INVALID_PARAMS, 400, { message: 'Invalid proxyKey' });
    }
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    console.error('Failed to validate proxyKey:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { message: 'Failed to validate proxyKey' });
  }
  
  try {
    if (handleType === 1) {
      // Start remote access (kimbap.io tunnel)
      
      // Check if ANY type=1 record already exists with matching proxyKey via proxy-api 8002
      let existingTunnelRecord = null;
      try {
        const cloudflaredResult = await getCloudflaredConfigs(body.common.userid, {
          type: 1,
          proxyKey: currentProxyKey
        });
        
        if (cloudflaredResult.dnsConfs?.length > 0) {
          existingTunnelRecord = cloudflaredResult.dnsConfs[0];
        }
      } catch (error) {
        console.error('Failed to get cloudflared configs from proxy-api:', error);
        // Continue without existing record if proxy-api fails
      }
      
      let tunnelId = '';
      let tunnelSubdomain = '';
      
      if (existingTunnelRecord) {
        // Tunnel record exists, just need to start cloudflared
        tunnelId = existingTunnelRecord.tunnelId;
        tunnelSubdomain = existingTunnelRecord.subdomain;
        
        // Note: For existing tunnel records, cloudflared management is now handled by proxy-api
        // The proxy-api will automatically manage cloudflared container lifecycle
        
        // Restart cloudflared to ensure it's running with the existing configuration
        try {
          await restartCloudflared(body.common.userid);
        } catch (restartError) {
          console.error('Failed to restart cloudflared:', restartError);
          // Continue even if restart fails - tunnel exists and user can manually restart
        }
        
        // Return existing subdomain
        const response: Response10011Data = {};
        if (tunnelSubdomain) {
          response.subdomain = tunnelSubdomain;
        }
        
        
        return response;
      }
      
      // No existing tunnel record, need to create new tunnel
      
      try {
        // Generate a unique 10-character appId based on machine info and timestamp
        const crypto = require('crypto');
        const os = require('os');
        
        // Combine hostname, timestamp, and random value for uniqueness
        const hostname = os.hostname().replace(/[^a-zA-Z0-9]/g, '').substring(0, 4);
        const timestamp = Date.now().toString(36).substring(0, 4);
        const random = crypto.randomBytes(2).toString('hex');
        const appId = `${hostname}${timestamp}${random}`.substring(0, 10).toLowerCase();
        
        const tunnelResponse = await kimbapCloudApi.createTunnel({
          appId: appId
        });
        
        tunnelId = tunnelResponse.tunnelId;
        tunnelSubdomain = tunnelResponse.subdomain;
        
        // Call proxy-api 8001 interface to update cloudflared config and restart container
        // Use currentProxyKey which was already fetched at line 47
        try {
          await updateCloudflaredConfig({
            proxyKey: currentProxyKey,
            tunnelId: tunnelResponse.tunnelId,
            subdomain: tunnelResponse.subdomain,
            credentials: tunnelResponse.credentials,
            publicIp: '' // Optional, default to empty string
          }, body.common.userid);
          
        } catch (proxyApiError) {
          console.error('Failed to update cloudflared config via proxy-api:', proxyApiError);
          // Continue even if proxy-api call fails - the tunnel is created
        }
        
      } catch (error) {
        console.error('Failed to create tunnel:', error);
        // Re-throw if it's an ApiError (like missing proxy)
        if (error instanceof ApiError) {
          throw error;
        }
        // Continue without tunnel ID if tunnel creation fails for other reasons
      }
      
      // Note: type=1 DNS records are now managed by KIMBAP Core via proxy-api
      // No need to save locally anymore
      
      // Return subdomain for handleType=1
      const response: Response10011Data = {};
      if (tunnelSubdomain) {
        response.subdomain = tunnelSubdomain;
      }


      return response;
      
    } else if (handleType === 2) {
      // Stop remote access (Stop cloudflared via proxy-api)
      
      try {
        // Stop cloudflared service via proxy-api 8005
        await stopCloudflared(body.common.userid);
        
        // Note: type=1 DNS records are managed by KIMBAP Core
        // The proxy-api handles all cloudflared container operations
        
        return {};
        
      } catch (error) {
        console.error('Failed to stop remote access:', error);
        throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { 
          message: 'Failed to stop remote access service' 
        });
      }
      
    } else if (handleType === 3) {
      // Add manual DNS record (original type=2 logic)
      
      // Validate required parameters for manual DNS
      if (!domain) {
        throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'domain' });
      }
      if (!publicIP) {
        throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'publicIP' });
      }
      
      // Add DNS record to database
      const now = Math.floor(Date.now() / 1000);
      
      await prisma.dnsConf.create({
        data: {
          subdomain: domain,
          type: 2, // manual DNS type
          publicIp: publicIP,
          tunnelId: '',
          addtime: now,
          updateTime: now,
          proxyKey: currentProxyKey
        }
      });


      return {};
      
    } else if (handleType === 4) {
      // Delete DNS record by recordId (original handleType=2 logic)
      
      // Validate required recordId
      if (!recordId) {
        throw new ApiError(ErrorCode.MISSING_REQUIRED_FIELD, 400, { field: 'recordId' });
      }
      
      // Find and delete the record with matching proxyKey
      const existingRecord = await prisma.dnsConf.findFirst({
        where: { 
          id: recordId,
          proxyKey: currentProxyKey
        }
      });
      
      if (!existingRecord) {
        throw new ApiError(ErrorCode.RECORD_NOT_FOUND, 404);
      }
      
      // Delete the record from database
      await prisma.dnsConf.delete({
        where: { id: recordId }
      });

      
      // Return empty response for delete operation
      return {};
    }
    
    throw new ApiError(ErrorCode.INVALID_FIELD_FORMAT, 400, { field: 'handleType' });
    
  } catch (error) {
    console.error('Protocol 10011 error:', error);
    
    
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
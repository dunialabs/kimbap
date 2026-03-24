import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import axios from 'axios';
import { getCloudflaredConfigs } from '@/lib/proxy-api';



interface Request10014 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    // No parameters required
  };
}

interface ManualDnsRecord {
  domain: string;
  publicIP: string;
  recordId: number;
}

interface Response10014Data {
  kimbapSubdomain: string;
  manualDnsList: ManualDnsRecord[];
  localPublicIP: string;
  manualConnection: string;
}

/**
 * Get public IP address of the local machine
 */
async function getLocalPublicIP(): Promise<string> {
  try {
    // Try multiple services for redundancy
    const services = [
      'https://api.ipify.org?format=text',
      'https://icanhazip.com',
      'https://ipecho.net/plain',
      'https://ifconfig.me/ip'
    ];
    
    for (const service of services) {
      try {
        const response = await axios.get(service, {
          timeout: 5000,
          headers: {
            'User-Agent': 'curl/7.68.0' // Some services require a user agent
          }
        });
        const ip = response.data.toString().trim();
        // Basic IP validation
        if (/^(\d{1,3}\.){3}\d{1,3}$/.test(ip)) {
          return ip;
        }
      } catch (error) {
        // Try next service
        continue;
      }
    }
    
    // If all services fail, return empty string
    console.error('Failed to get public IP from all services');
    return '';
  } catch (error) {
    console.error('Error getting public IP:', error);
    return '';
  }
}

/**
 * Protocol 10014 - Get DNS Configuration
 * Returns DNS configuration
 * - type=1: kimbap.io subdomain (from proxy-api 8002)
 * - type=2: manual DNS records (from local database)
 * - localPublicIP: current public IP of the server
 * - manualConnection: configured KIMBAP Core host and port from config table
 */
export async function handleProtocol10014(body: Request10014): Promise<Response10014Data> {
  try {
    // Get manualConnection from config table
    let manualConnection = '';
    try {
      const configData = await prisma.config.findFirst();
      if (configData && configData.kimbap_core_host) {
        const host = configData.kimbap_core_host;
        const port = configData.kimbap_core_prot;
        
        // Build the connection string
        if (host.startsWith('http://') || host.startsWith('https://')) {
          // Host already contains protocol
          const url = new URL(host);
          const isHttps = url.protocol === 'https:';
          const defaultPort = isHttps ? 443 : 80;
          
          // Add port if it's not the default for the protocol
          if (port && port !== defaultPort) {
            manualConnection = `${host}:${port}`;
          } else {
            manualConnection = host;
          }
        } else if (host) {
          // Host doesn't contain protocol, add it based on type
          const isIP = /^(\d{1,3}\.){3}\d{1,3}$/.test(host) || host === 'localhost';
          const protocol = isIP ? 'http' : 'https';
          const defaultPort = isIP ? 80 : 443;
          
          if (port && port !== defaultPort) {
            manualConnection = `${protocol}://${host}:${port}`;
          } else {
            manualConnection = `${protocol}://${host}`;
          }
        }
      }
    } catch (error) {
      console.error('Error getting config for manualConnection:', error);
      // Keep manualConnection as empty string
    }

    // Fallback to MCP_GATEWAY_URL env var when database config is empty
    // This matches the 3-tier priority used in proxy-api.ts and protocol-10021
    if (!manualConnection) {
      const mcpGatewayUrl = process.env.MCP_GATEWAY_URL?.trim();
      if (mcpGatewayUrl) {
        try {
          const parsed = new URL(mcpGatewayUrl);
          // Use origin to strip trailing slashes, paths, or accidental suffixes
          manualConnection = parsed.origin;
        } catch {
          // Invalid URL format, keep manualConnection as empty
        }
      }
    }
    
    // Get type=1 records from proxy-api 8002 (cloudflared configs)
    let kimbapSubdomain = '';
    try {
      const cloudflaredResult = await getCloudflaredConfigs(body.common.userid, { type: 1 });
      if (cloudflaredResult.dnsConfs?.length > 0) {
        // Only return subdomain if container is running
        const firstConfig = cloudflaredResult.dnsConfs[0];
        if (firstConfig.status === 'running') {
          kimbapSubdomain = firstConfig.subdomain || '';
        }
      }
    } catch (error) {
      console.error('Error getting cloudflared configs from proxy-api:', error);
    }
    
    // Get type=2 records for manualDnsList (still from local database)
    const manualRecords = await prisma.dnsConf.findMany({
      where: { type: 2 },
      orderBy: { id: 'asc' }
    });
    
    // Get local public IP
    const localPublicIP = await getLocalPublicIP();
    
    // Build response
    const response: Response10014Data = {
      kimbapSubdomain: kimbapSubdomain,
      manualDnsList: manualRecords.map(record => ({
        domain: record.subdomain,
        publicIP: record.publicIp || '',
        recordId: record.id
      })),
      localPublicIP: localPublicIP,
      manualConnection: manualConnection
    };
    
    console.log('Protocol 10014 response:', {
      kimbapSubdomain: response.kimbapSubdomain,
      manualDnsCount: response.manualDnsList.length,
      localPublicIP: response.localPublicIP,
      manualConnection: response.manualConnection
    });
    
    return response;
    
  } catch (error) {
    console.error('Protocol 10014 error:', error);
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR);
  }
}
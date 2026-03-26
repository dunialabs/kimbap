import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request21004 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number; // Time range: 1-today, 7-last 7 days, 30-last 30 days, 90-last 90 days
    tokenId: string;   // Specific token ID, empty means all tokens
  };
}

interface GeoLocation {
  country: string;      // country code
  countryName: string;  // country name
  city: string;         // city ​​name
  requests: number;     // Number of requests
  percentage: number;   // Proportion (%)
  uniqueIPs: number;    // Number of independent IPs
}

interface TokenGeoUsage {
  tokenId: string;         // Token ID
  tokenName: string;       // Token name
  locations: GeoLocation[]; // Geographical distribution
}

interface Response21004Data {
  geoUsage: TokenGeoUsage[];
}

/**
 * Protocol 21004 - Get Token Geographic Usage
 * Get token geographical location usage distribution
 */
export async function handleProtocol21004(body: Request21004): Promise<Response21004Data> {
  try {
    const { tokenId } = body.params;
    const normalizedTimeRange = Number.isFinite(Math.floor(Number(body.params.timeRange))) && Math.floor(Number(body.params.timeRange)) >= 1
      ? Math.floor(Number(body.params.timeRange))
      : 1;

    const proxy = await getProxy();
    const proxyKey = proxy.proxyKey;
    
    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (normalizedTimeRange * 24 * 60 * 60);
    
    // Build where condition
    const whereCondition: any = {
      proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      tokenMask: {
        not: ''
      },
      ip: {
        not: ''
      }
    };
    
    if (tokenId && tokenId.trim()) {
      whereCondition.userid = tokenId.trim();
    }
    
    // Get all relevant log data
    const logs = await prisma.log.findMany({
      where: whereCondition,
      select: {
        tokenMask: true,
        ip: true
      }
    });
    
    // Group by token
    const tokenLogsMap = new Map<string, typeof logs>();
    logs.forEach(log => {
      if (!tokenLogsMap.has(log.tokenMask)) {
        tokenLogsMap.set(log.tokenMask, []);
      }
      tokenLogsMap.get(log.tokenMask)!.push(log);
    });
    
    // Analyze geographical location distribution for each token
    const geoUsage: TokenGeoUsage[] = Array.from(tokenLogsMap.entries()).map(([tokenMask, tokenLogs]) => {
      const locationGroups = new Map<string, { requests: number; ips: Set<string> }>();

      tokenLogs.forEach(log => {
        const rawIp = (log.ip || '').trim();
        if (!rawIp) return;
        const parts = rawIp.split('.');
        const subnet = parts.length === 4 ? `${parts[0]}.${parts[1]}.${parts[2]}.0/24` : rawIp;
        if (!locationGroups.has(subnet)) {
          locationGroups.set(subnet, { requests: 0, ips: new Set<string>() });
        }
        const group = locationGroups.get(subnet)!;
        group.requests += 1;
        group.ips.add(rawIp);
      });

      const locations: GeoLocation[] = [];

      const totalRequests = tokenLogs.length;

      locationGroups.forEach((group, subnet) => {
        locations.push({
          country: 'UN',
          countryName: 'Unknown',
          city: subnet,
          requests: group.requests,
          percentage: totalRequests > 0 ? Math.round((group.requests / totalRequests) * 1000) / 10 : 0,
          uniqueIPs: group.ips.size
        });
      });
      
      // Sort by number of requests in descending order
      locations.sort((a, b) => b.requests - a.requests);
      
      return {
        tokenId: tokenMask.substring(0, 8) + '...',
        tokenName: `Token ${tokenMask.substring(0, 8)}...`,
        locations
      };
    });
    
    const response: Response21004Data = {
      geoUsage
    };
    
    console.log('Protocol 21004 response:', {
      tokensCount: geoUsage.length,
      timeRange: normalizedTimeRange,
      totalLocations: geoUsage.reduce((sum, token) => sum + token.locations.length, 0)
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21004 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token geographic usage distribution' });
  }
}

import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request21008 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number; // : 1-, 7-7, 30-30, 90-90
    tokenId: string;   // ID，
    page: number;      // ，1
    pageSize: number;  // 
  };
}

interface ClientInfo {
  clientIP: string;         // IP（）
  userAgent: string;        // （）
  requests: number;         // 
  successRequests: number;  // 
  failedRequests: number;   // 
  firstSeen: number;        // 
  lastSeen: number;         // 
  country: string;          // -
  city: string;             // -
  riskLevel: number;        //  1-, 2-, 3-
}

interface TokenClientAnalysis {
  tokenId: string;          // ID
  tokenName: string;        // 
  totalClients: number;     // 
  clients: ClientInfo[];    // 
}

interface Response21008Data {
  clientAnalysis: TokenClientAnalysis[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

/**
 * Protocol 21008 - Get Token Client Analysis
 * 
 */
export async function handleProtocol21008(body: Request21008): Promise<Response21008Data> {
  try {
    const { timeRange, tokenId, page = 1, pageSize = 20 } = body.params;
    const safePage = Math.max(1, Math.floor(Number(page) || 1));
    const safePageSize = Math.min(100, Math.max(1, Math.floor(Number(pageSize) || 20)));

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-21008] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    // 
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // where
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
    
    const groupedTokens = await prisma.log.groupBy({
      by: ['tokenMask'],
      where: whereCondition,
      _count: {
        id: true
      },
      orderBy: {
        _count: {
          id: 'desc'
        }
      }
    });

    const total = groupedTokens.length;
    const totalPages = Math.ceil(total / safePageSize);
    const startIndex = (safePage - 1) * safePageSize;
    const endIndex = startIndex + safePageSize;
    const pagedTokenGroups = groupedTokens.slice(startIndex, endIndex);
    const pagedTokenMasks = pagedTokenGroups
      .map((group) => group.tokenMask)
      .filter((mask): mask is string => typeof mask === 'string' && mask.length > 0);

    const logs = pagedTokenMasks.length === 0
      ? []
      : await prisma.log.findMany({
          where: {
            ...whereCondition,
            tokenMask: {
              in: pagedTokenMasks
            }
          },
          select: {
            tokenMask: true,
            ip: true,
            ua: true,
            statusCode: true,
            addtime: true
          },
          orderBy: {
            addtime: 'desc'
          }
        });

    const tokenLogsMap = new Map<string, typeof logs>();
    logs.forEach(log => {
      if (!tokenLogsMap.has(log.tokenMask)) {
        tokenLogsMap.set(log.tokenMask, []);
      }
      tokenLogsMap.get(log.tokenMask)!.push(log);
    });
    
    // ：IP
    const maskIPAddress = (ip: string): string => {
      const parts = ip.split('.');
      if (parts.length === 4) {
        return `${parts[0]}.${parts[1]}.xxx.xxx`;
      }
      return 'xxx.xxx.xxx.xxx';
    };
    
    // ：
    const maskUserAgent = (ua: string): string => {
      if (!ua) return 'Unknown';
      return ua.substring(0, 50) + (ua.length > 50 ? '...' : '');
    };
    
    // ：
    const calculateRiskLevel = (clientStats: {
      requests: number;
      failedRequests: number;
      timeSpan: number;
    }): number => {
      const { requests, failedRequests, timeSpan } = clientStats;
      const failureRate = requests > 0 ? failedRequests / requests : 0;
      const requestRate = timeSpan > 0 ? requests / (timeSpan / 3600) : 0; // 
      
      // 
      if (failureRate > 0.5 || requestRate > 1000) {
        return 3; // 
      } else if (failureRate > 0.2 || requestRate > 500) {
        return 2; // 
      } else {
        return 1; // 
      }
    };
    
    // token
    const pagedClientAnalysis: TokenClientAnalysis[] = pagedTokenGroups.map((tokenGroup) => {
      const tokenMask = tokenGroup.tokenMask;
      if (!tokenMask) {
        return {
          tokenId: 'unknown',
          tokenName: 'Token unknown',
          totalClients: 0,
          clients: []
        };
      }

      const tokenLogs = tokenLogsMap.get(tokenMask) || [];
      // IPUA
      const clientGroups = new Map<string, typeof tokenLogs>();
      tokenLogs.forEach(log => {
        const clientKey = `${log.ip}_${log.ua || 'unknown'}`;
        if (!clientGroups.has(clientKey)) {
          clientGroups.set(clientKey, []);
        }
        clientGroups.get(clientKey)!.push(log);
      });
      
      // 
      const clients: ClientInfo[] = Array.from(clientGroups.entries()).map(([clientKey, clientLogs]) => {
        const [ip, ua] = clientKey.split('_');
        const requests = clientLogs.length;
        const successRequests = clientLogs.filter(log => log.statusCode != null && log.statusCode >= 200 && log.statusCode < 300).length;
        const failedRequests = requests - successRequests;
        
        const timestamps = clientLogs.map(log => Number(log.addtime)).sort((a, b) => a - b);
        const firstSeen = timestamps[0];
        const lastSeen = timestamps[timestamps.length - 1];
        const timeSpan = lastSeen - firstSeen;
        
        const riskLevel = calculateRiskLevel({ requests, failedRequests, timeSpan });
        
        return {
          clientIP: maskIPAddress(ip),
          userAgent: maskUserAgent(ua === 'unknown' ? '' : ua),
          requests,
          successRequests,
          failedRequests,
          firstSeen,
          lastSeen,
          country: 'unknown',
          city: 'unknown',
          riskLevel
        };
      });
      
      // 
      clients.sort((a, b) => b.requests - a.requests);
      
      return {
        tokenId: tokenMask.substring(0, 8) + '...',
        tokenName: `Token ${tokenMask.substring(0, 8)}...`,
        totalClients: clients.length,
        clients
      };
    });
    
    const response: Response21008Data = {
      clientAnalysis: pagedClientAnalysis,
      pagination: {
        page: safePage,
        pageSize: safePageSize,
        total,
        totalPages
      }
    };
    
    console.log('Protocol 21008 response:', {
      totalTokens: total,
      paginatedTokens: pagedClientAnalysis.length,
      totalClients: pagedClientAnalysis.reduce((sum, token) => sum + token.totalClients, 0),
      timeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21008 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token client analysis' });
  }
}

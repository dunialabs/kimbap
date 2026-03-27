import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request21007 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number; // : 1-, 7-7, 30-30, 90-90
    tokenId: string;   // ID，
  };
}

interface RateLimitEvent {
  timestamp: number;        // 
  requests: number;         // 
  blockedRequests: number;  // 
  clientIP: string;         // IP
}

interface TokenRateLimit {
  tokenId: string;           // ID
  tokenName: string;         // 
  configuredLimit: number;   // 
  totalHits: number;         // 
  peakHitsPerMinute: number; // 
  recentEvents: RateLimitEvent[]; // 
}

interface Response21007Data {
  rateLimitAnalysis: TokenRateLimit[];
}

/**
 * Protocol 21007 - Get Token Rate Limit Analysis
 * 
 */
export async function handleProtocol21007(body: Request21007): Promise<Response21007Data> {
  try {
    const { tokenId } = body.params;
    const normalizedTimeRange = Number.isFinite(Math.floor(Number(body.params.timeRange))) && Math.floor(Number(body.params.timeRange)) >= 1
      ? Math.floor(Number(body.params.timeRange))
      : 1;

    let proxyKey = '';
    try {
      const proxy = await getProxy();
      proxyKey = proxy.proxyKey;
    } catch (error) {
      console.error('[Protocol-21007] Failed to get proxy info:', error);
      throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, {
        details: 'Failed to get proxy information'
      });
    }
    
    const now = Math.floor(Date.now() / 1000);
    const startTime = now - (normalizedTimeRange * 24 * 60 * 60);
    
    // where
    const whereCondition: any = {
      proxyKey,
      addtime: {
        gte: BigInt(startTime)
      },
      tokenMask: {
        not: ''
      },
      statusCode: 429 // HTTP 429 Too Many Requests
    };
    
    if (tokenId && tokenId.trim()) {
      whereCondition.userid = tokenId.trim();
    }
    
    // 
    const rateLimitLogs = await prisma.log.findMany({
      where: whereCondition,
      select: {
        tokenMask: true,
        addtime: true,
        ip: true
      },
      orderBy: {
        addtime: 'desc'
      }
    });
    
    // token
    const tokenRateLimitMap = new Map<string, typeof rateLimitLogs>();
    rateLimitLogs.forEach(log => {
      if (!tokenRateLimitMap.has(log.tokenMask)) {
        tokenRateLimitMap.set(log.tokenMask, []);
      }
      tokenRateLimitMap.get(log.tokenMask)!.push(log);
    });
    
    // token
    const rateLimitAnalysis: TokenRateLimit[] = await Promise.all(
      Array.from(tokenRateLimitMap.entries()).map(async ([tokenMask, logs]) => {
        // 
        const minuteHits = new Map<number, number>();
        logs.forEach(log => {
          const minute = Math.floor(Number(log.addtime) / 60) * 60;
          minuteHits.set(minute, (minuteHits.get(minute) || 0) + 1);
        });
        
        // 
        const peakHitsPerMinute = minuteHits.size > 0 ? Math.max(...Array.from(minuteHits.values())) : 0;
        
        // （10）
        const recentEvents: RateLimitEvent[] = [];
        const recentLogs = logs.slice(0, 10);
        
        // 
        const eventGroups = new Map<number, { logs: typeof logs; ips: Set<string> }>();
        recentLogs.forEach(log => {
          const minute = Math.floor(Number(log.addtime) / 60) * 60;
          if (!eventGroups.has(minute)) {
            eventGroups.set(minute, { logs: [], ips: new Set() });
          }
          eventGroups.get(minute)!.logs.push(log);
          if (log.ip) {
            eventGroups.get(minute)!.ips.add(log.ip);
          }
        });
        
        // 
        Array.from(eventGroups.entries())
          .sort(([a], [b]) => b - a) // 
          .slice(0, 5) // 5
          .forEach(([timestamp, group]) => {
            const requests = group.logs.length;
            const blockedRequests = requests;
            
            // IP
            const primaryIP = group.ips.size > 0 ? Array.from(group.ips)[0] : 'unknown';
            
            recentEvents.push({
              timestamp,
              requests,
              blockedRequests,
              clientIP: primaryIP.length > 10 ? primaryIP.substring(0, 10) + '...' : primaryIP
            });
          });
        
        return {
          tokenId: tokenMask.substring(0, 8) + '...',
          tokenName: `Token ${tokenMask.substring(0, 8)}...`,
          configuredLimit: 0,
          totalHits: logs.length,
          peakHitsPerMinute,
          recentEvents
        };
      })
    );
    
    // 
    rateLimitAnalysis.sort((a, b) => b.totalHits - a.totalHits);
    
    const response: Response21007Data = {
      rateLimitAnalysis
    };
    
    console.log('Protocol 21007 response:', {
      tokensAnalyzed: rateLimitAnalysis.length,
      totalEvents: rateLimitAnalysis.reduce((sum, token) => sum + token.totalHits, 0),
      timeRange: normalizedTimeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21007 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token rate limit analysis' });
  }
}

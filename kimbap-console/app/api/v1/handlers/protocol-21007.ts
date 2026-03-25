import { prisma } from '@/lib/prisma';

import { ApiError, ErrorCode } from '@/lib/error-codes';
import { getProxy } from '@/lib/proxy-api';



interface Request21007 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {
    timeRange: number; // 时间范围: 1-今天, 7-最近7天, 30-最近30天, 90-最近90天
    tokenId: string;   // 特定令牌ID，空表示所有令牌
  };
}

interface RateLimitEvent {
  timestamp: number;        // 事件时间戳
  requests: number;         // 时间窗口内请求数
  blockedRequests: number;  // 被阻止的请求数
  clientIP: string;         // 客户端IP
}

interface TokenRateLimit {
  tokenId: string;           // 令牌ID
  tokenName: string;         // 令牌名称
  configuredLimit: number;   // 配置的速率限制
  totalHits: number;         // 总触发次数
  peakHitsPerMinute: number; // 每分钟峰值触发次数
  recentEvents: RateLimitEvent[]; // 最近的限制事件
}

interface Response21007Data {
  rateLimitAnalysis: TokenRateLimit[];
}

/**
 * Protocol 21007 - Get Token Rate Limit Analysis
 * 获取令牌速率限制分析
 */
export async function handleProtocol21007(body: Request21007): Promise<Response21007Data> {
  try {
    const { timeRange, tokenId } = body.params;

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
    
    // 计算时间范围
    const now = Math.floor(Date.now() / 1000);
    const timeRangeSeconds = timeRange * 24 * 60 * 60;
    const startTime = now - timeRangeSeconds;
    
    // 构建where条件
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
    
    // 获取所有速率限制相关的日志
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
    
    // 按token分组
    const tokenRateLimitMap = new Map<string, typeof rateLimitLogs>();
    rateLimitLogs.forEach(log => {
      if (!tokenRateLimitMap.has(log.tokenMask)) {
        tokenRateLimitMap.set(log.tokenMask, []);
      }
      tokenRateLimitMap.get(log.tokenMask)!.push(log);
    });
    
    // 为每个token分析速率限制情况
    const rateLimitAnalysis: TokenRateLimit[] = await Promise.all(
      Array.from(tokenRateLimitMap.entries()).map(async ([tokenMask, logs]) => {
        // 分析每分钟的速率限制触发次数
        const minuteHits = new Map<number, number>();
        logs.forEach(log => {
          const minute = Math.floor(Number(log.addtime) / 60) * 60;
          minuteHits.set(minute, (minuteHits.get(minute) || 0) + 1);
        });
        
        // 计算峰值每分钟触发次数
        const peakHitsPerMinute = minuteHits.size > 0 ? Math.max(...Array.from(minuteHits.values())) : 0;
        
        // 生成最近的限制事件（最多10个）
        const recentEvents: RateLimitEvent[] = [];
        const recentLogs = logs.slice(0, 10);
        
        // 按分钟聚合事件
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
        
        // 转换为事件格式
        Array.from(eventGroups.entries())
          .sort(([a], [b]) => b - a) // 按时间倒序
          .slice(0, 5) // 最多5个最近事件
          .forEach(([timestamp, group]) => {
            const requests = group.logs.length;
            const blockedRequests = requests;
            
            // 选择主要的客户端IP
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
    
    // 按总触发次数降序排列
    rateLimitAnalysis.sort((a, b) => b.totalHits - a.totalHits);
    
    const response: Response21007Data = {
      rateLimitAnalysis
    };
    
    console.log('Protocol 21007 response:', {
      tokensAnalyzed: rateLimitAnalysis.length,
      totalEvents: rateLimitAnalysis.reduce((sum, token) => sum + token.totalHits, 0),
      timeRange
    });
    
    return response;
    
  } catch (error) {
    if (error instanceof ApiError) throw error;
    console.error('Protocol 21007 error:', error);
    throw new ApiError(ErrorCode.INTERNAL_SERVER_ERROR, 500, { details: 'Failed to get token rate limit analysis' });
  }
}
